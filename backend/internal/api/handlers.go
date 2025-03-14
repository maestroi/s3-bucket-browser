package api

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/blockdaemon/s3-bucket-browser/internal/cache"
	"github.com/blockdaemon/s3-bucket-browser/internal/models"
	"github.com/blockdaemon/s3-bucket-browser/internal/s3"
	"github.com/gorilla/mux"
)

const (
	defaultPageSize    = 20
	maxPageSize        = 100
	cacheExpiration    = 5 * time.Minute
	metadataOptionsKey = "metadata:options"
)

// Regular expression to match snapshot JSON files
var snapshotRegex = regexp.MustCompile(`snapshot-(\d+)-([A-Za-z0-9]+)\.json$`)

// FilterOptions represents the available filter options
type FilterOptions struct {
	SolanaVersions []string `json:"solanaVersions"`
	Statuses       []string `json:"statuses"`
	UploadedBy     []string `json:"uploadedBy"`
	Nodes          []string `json:"nodes"`
	SlotRanges     []string `json:"slotRanges"`
}

// Handler represents the API handler
type Handler struct {
	s3Service     *s3.Service
	cacheService  *cache.RedisCache
	hub           *Hub
	filterOptions *FilterOptions
	optionsLock   sync.RWMutex
}

// NewHandler creates a new API handler
func NewHandler(s3Service *s3.Service, cacheService *cache.RedisCache) *Handler {
	hub := NewHub(s3Service)

	handler := &Handler{
		s3Service:    s3Service,
		cacheService: cacheService,
		hub:          hub,
		filterOptions: &FilterOptions{
			SolanaVersions: []string{},
			Statuses:       []string{},
			UploadedBy:     []string{},
			Nodes:          []string{},
			SlotRanges:     []string{},
		},
		optionsLock: sync.RWMutex{},
	}

	// Start the WebSocket hub
	go hub.Run(context.Background())

	// Start initial metadata indexing
	go handler.indexMetadata(context.Background())

	return handler
}

// RegisterRoutes registers the API routes
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/files", h.ListFiles).Methods("GET")
	r.HandleFunc("/api/files/{key}", h.GetFile).Methods("GET")
	r.HandleFunc("/api/metadata/options", h.GetMetadataOptions).Methods("GET")
	r.HandleFunc("/api/metadata", h.ListMetadata).Methods("GET")
	r.HandleFunc("/api/metadata/{key}", h.GetMetadata).Methods("GET")
	r.HandleFunc("/api/ws", h.WebSocketHandler).Methods("GET")
	r.HandleFunc("/api/debug/reindex", h.DebugReindex).Methods("GET")
	r.HandleFunc("/api/debug/examine-file", h.DebugExamineFile).Methods("GET")
}

// isSnapshotMetadataFile checks if a file is a snapshot metadata file
func isSnapshotMetadataFile(key string) bool {
	return snapshotRegex.MatchString(key)
}

// extractSlotAndNode extracts the slot and node from a snapshot metadata file name
func extractSlotAndNode(key string) (int64, string) {
	matches := snapshotRegex.FindStringSubmatch(key)
	if len(matches) < 3 {
		return 0, ""
	}

	slot, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, ""
	}

	return slot, matches[2]
}

// getSlotRange returns a human-readable slot range
func getSlotRange(slot int64) string {
	// Create ranges like 0-1M, 1M-2M, etc.
	rangeSize := int64(1000000) // 1 million
	rangeStart := (slot / rangeSize) * rangeSize
	rangeEnd := rangeStart + rangeSize

	if rangeStart == 0 {
		return "< 1M"
	}

	return strconv.FormatInt(rangeStart/1000000, 10) + "M-" + strconv.FormatInt(rangeEnd/1000000, 10) + "M"
}

// Define a simplified metadata struct for parsing that doesn't use time.Time
type SimpleMetadata struct {
	SolanaVersion string `json:"solana_version"`
	Status        string `json:"status"`
	UploadedBy    string `json:"uploaded_by"`
}

// indexMetadata indexes all metadata files to build filter options
func (h *Handler) indexMetadata(ctx context.Context) {
	log.Println("Starting initial metadata indexing...")

	// Try to get from cache first
	if h.cacheService != nil {
		var options FilterOptions
		err := h.cacheService.Get(ctx, metadataOptionsKey, &options)
		if err == nil {
			// Cache hit, use cached options
			h.optionsLock.Lock()
			h.filterOptions = &options
			h.optionsLock.Unlock()
			log.Println("Loaded filter options from cache")
			return
		} else {
			log.Printf("Cache miss for metadata options: %v", err)
		}
	} else {
		log.Println("Cache service not available, skipping cache operations")
	}

	// List all objects
	objects, err := h.s3Service.ListObjects(ctx, "")
	if err != nil {
		log.Printf("Failed to list objects for indexing: %v", err)
		return
	}

	log.Printf("Found %d total objects in S3 bucket", len(objects))

	// Filter metadata files
	var metadataFiles []s3.Object
	for _, obj := range objects {
		if obj.IsMetadata && isSnapshotMetadataFile(obj.Key) {
			metadataFiles = append(metadataFiles, obj)
		}
	}

	log.Printf("Found %d snapshot metadata files to index", len(metadataFiles))

	if len(metadataFiles) == 0 {
		log.Println("No metadata files found to index. Check S3 bucket and file naming patterns.")
		// Set empty options to avoid repeated indexing attempts
		h.optionsLock.Lock()
		h.filterOptions = &FilterOptions{
			SolanaVersions: []string{},
			Statuses:       []string{},
			UploadedBy:     []string{},
			Nodes:          []string{},
			SlotRanges:     []string{},
		}
		h.optionsLock.Unlock()
		return
	}

	// Log some sample file names for debugging
	if len(metadataFiles) > 0 {
		sampleSize := 5
		if len(metadataFiles) < sampleSize {
			sampleSize = len(metadataFiles)
		}
		log.Printf("Sample metadata files: %v", metadataFiles[:sampleSize])

		// Examine the first file in detail
		if len(metadataFiles) > 0 {
			firstFile := metadataFiles[0]
			result, err := h.s3Service.GetObject(ctx, firstFile.Key)
			if err != nil {
				log.Printf("Failed to get first metadata file %s: %v", firstFile.Key, err)
			} else {
				body, err := io.ReadAll(result.Body)
				result.Body.Close()
				if err != nil {
					log.Printf("Failed to read first metadata file %s: %v", firstFile.Key, err)
				} else {
					log.Printf("Content of first metadata file %s: %s", firstFile.Key, string(body))

					// Try to parse as JSON
					var rawData map[string]interface{}
					if err := json.Unmarshal(body, &rawData); err != nil {
						log.Printf("Failed to parse first metadata file as JSON: %v", err)
					} else {
						log.Printf("First metadata file parsed as JSON: %v", rawData)
					}
				}
			}
		}
	}

	// Process metadata files
	versions := make(map[string]bool)
	statuses := make(map[string]bool)
	uploaders := make(map[string]bool)
	nodes := make(map[string]bool)
	slotRanges := make(map[string]bool)

	// Use a worker pool to process files in parallel
	workerCount := 10
	filesChan := make(chan s3.Object, len(metadataFiles))
	var wg sync.WaitGroup

	// Mutex for concurrent map access
	var mapMutex sync.Mutex

	// Start workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for obj := range filesChan {
				// Extract slot and node from filename
				slot, node := extractSlotAndNode(obj.Key)
				if slot > 0 && node != "" {
					mapMutex.Lock()
					nodes[node] = true
					slotRange := getSlotRange(slot)
					slotRanges[slotRange] = true
					mapMutex.Unlock()
				}

				// Get metadata from S3
				result, err := h.s3Service.GetObject(ctx, obj.Key)
				if err != nil {
					log.Printf("Failed to get metadata file %s: %v", obj.Key, err)
					continue
				}

				// Parse metadata
				body, err := io.ReadAll(result.Body)
				result.Body.Close()

				if err != nil {
					log.Printf("Failed to read metadata file %s: %v", obj.Key, err)
					continue
				}

				// Try to parse with the simplified struct first
				var simpleMetadata SimpleMetadata
				err = json.Unmarshal(body, &simpleMetadata)
				if err != nil {
					log.Printf("Failed to parse metadata file %s: %v", obj.Key, err)

					// Try to parse as a generic map as a fallback
					var rawData map[string]interface{}
					if jsonErr := json.Unmarshal(body, &rawData); jsonErr == nil {
						// Extract fields from the raw data
						mapMutex.Lock()
						if version, ok := rawData["solana_version"].(string); ok && version != "" && version != "unknown" {
							versions[version] = true
						}
						if status, ok := rawData["status"].(string); ok && status != "" && status != "unknown" {
							statuses[status] = true
						}
						if uploader, ok := rawData["uploaded_by"].(string); ok && uploader != "" && uploader != "unknown" {
							uploaders[uploader] = true
						}
						mapMutex.Unlock()
					}
					continue
				}

				// Add to unique sets
				mapMutex.Lock()
				if simpleMetadata.SolanaVersion != "" && simpleMetadata.SolanaVersion != "unknown" {
					versions[simpleMetadata.SolanaVersion] = true
				}

				if simpleMetadata.Status != "" && simpleMetadata.Status != "unknown" {
					statuses[simpleMetadata.Status] = true
				}

				if simpleMetadata.UploadedBy != "" && simpleMetadata.UploadedBy != "unknown" {
					uploaders[simpleMetadata.UploadedBy] = true
				}
				mapMutex.Unlock()
			}
		}()
	}

	// Send files to workers
	for _, file := range metadataFiles {
		filesChan <- file
	}
	close(filesChan)

	// Wait for all workers to finish
	wg.Wait()

	// Log the raw data collected
	log.Printf("Raw versions collected: %v", versions)
	log.Printf("Raw statuses collected: %v", statuses)
	log.Printf("Raw uploaders collected: %v", uploaders)
	log.Printf("Raw nodes collected: %v", nodes)
	log.Printf("Raw slot ranges collected: %v", slotRanges)

	// Convert maps to slices
	versionsList := make([]string, 0, len(versions))
	for v := range versions {
		versionsList = append(versionsList, v)
	}

	// Sort versions semantically
	sort.Slice(versionsList, func(i, j int) bool {
		// Extract major, minor, patch versions
		vI := strings.Split(versionsList[i], ".")
		vJ := strings.Split(versionsList[j], ".")

		// Compare major version
		if len(vI) > 0 && len(vJ) > 0 {
			majorI, errI := strconv.Atoi(vI[0])
			majorJ, errJ := strconv.Atoi(vJ[0])
			if errI == nil && errJ == nil && majorI != majorJ {
				return majorI < majorJ
			}
		}

		// Compare minor version
		if len(vI) > 1 && len(vJ) > 1 {
			minorI, errI := strconv.Atoi(vI[1])
			minorJ, errJ := strconv.Atoi(vJ[1])
			if errI == nil && errJ == nil && minorI != minorJ {
				return minorI < minorJ
			}
		}

		// Compare patch version
		if len(vI) > 2 && len(vJ) > 2 {
			patchI, errI := strconv.Atoi(vI[2])
			patchJ, errJ := strconv.Atoi(vJ[2])
			if errI == nil && errJ == nil {
				return patchI < patchJ
			}
		}

		// Fallback to string comparison
		return versionsList[i] < versionsList[j]
	})

	statusesList := make([]string, 0, len(statuses))
	for s := range statuses {
		statusesList = append(statusesList, s)
	}
	sort.Strings(statusesList)

	uploadersList := make([]string, 0, len(uploaders))
	for u := range uploaders {
		uploadersList = append(uploadersList, u)
	}
	sort.Strings(uploadersList)

	nodesList := make([]string, 0, len(nodes))
	for n := range nodes {
		nodesList = append(nodesList, n)
	}
	sort.Strings(nodesList)

	slotRangesList := make([]string, 0, len(slotRanges))
	for sr := range slotRanges {
		slotRangesList = append(slotRangesList, sr)
	}

	// Sort slot ranges numerically
	sort.Slice(slotRangesList, func(i, j int) bool {
		// Extract the first number from each range
		aMatch := regexp.MustCompile(`^(\d+)M`).FindStringSubmatch(slotRangesList[i])
		bMatch := regexp.MustCompile(`^(\d+)M`).FindStringSubmatch(slotRangesList[j])

		aNum := 0
		bNum := 0

		if len(aMatch) > 1 {
			aNum, _ = strconv.Atoi(aMatch[1])
		}

		if len(bMatch) > 1 {
			bNum, _ = strconv.Atoi(bMatch[1])
		}

		// Special case for "< 1M"
		if slotRangesList[i] == "< 1M" {
			return true
		}
		if slotRangesList[j] == "< 1M" {
			return false
		}

		return aNum < bNum
	})

	// Update filter options
	h.optionsLock.Lock()
	h.filterOptions.SolanaVersions = versionsList
	h.filterOptions.Statuses = statusesList
	h.filterOptions.UploadedBy = uploadersList
	h.filterOptions.Nodes = nodesList
	h.filterOptions.SlotRanges = slotRangesList
	h.optionsLock.Unlock()

	log.Printf("Metadata indexing complete. Found %d versions, %d statuses, %d uploaders, %d nodes, %d slot ranges",
		len(versionsList), len(statusesList), len(uploadersList), len(nodesList), len(slotRangesList))
}

// GetMetadataOptions returns the available filter options
func (h *Handler) GetMetadataOptions(w http.ResponseWriter, r *http.Request) {
	log.Println("GetMetadataOptions: Request received")

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Check if we have options in memory
	h.optionsLock.RLock()
	options := h.filterOptions
	h.optionsLock.RUnlock()

	// If we don't have options in memory, try to get from cache
	if options == nil || len(options.SolanaVersions) == 0 && len(options.Statuses) == 0 {
		log.Println("GetMetadataOptions: No options in memory, checking cache")

		// Try to get from cache if available
		if h.cacheService != nil {
			var cachedOptions FilterOptions
			err := h.cacheService.Get(r.Context(), metadataOptionsKey, &cachedOptions)
			if err == nil {
				log.Println("GetMetadataOptions: Using options from cache")
				options = &cachedOptions

				// Update in-memory options
				h.optionsLock.Lock()
				h.filterOptions = &cachedOptions
				h.optionsLock.Unlock()
			} else {
				log.Printf("GetMetadataOptions: Cache miss: %v", err)
			}
		} else {
			log.Println("GetMetadataOptions: Cache service not available")
		}
	}

	// If we still don't have options, trigger indexing
	if options == nil || len(options.SolanaVersions) == 0 && len(options.Statuses) == 0 {
		log.Println("GetMetadataOptions: No options available, triggering indexing")

		// Create empty options to avoid nil pointer
		options = &FilterOptions{
			SolanaVersions: []string{},
			Statuses:       []string{},
			UploadedBy:     []string{},
			Nodes:          []string{},
			SlotRanges:     []string{},
		}

		// Trigger indexing in a goroutine
		go func() {
			h.indexMetadata(context.Background())
		}()
	}

	log.Printf("GetMetadataOptions: Returning options with %d versions, %d statuses, %d uploaders, %d nodes, %d slot ranges",
		len(options.SolanaVersions), len(options.Statuses), len(options.UploadedBy), len(options.Nodes), len(options.SlotRanges))

	// Return options
	respondWithJSON(w, http.StatusOK, options)
}

// ListFiles lists files in the S3 bucket
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	log.Printf("ListFiles: Request received")

	// List objects directly from S3 (skip cache for now since it's causing issues)
	objects, err := h.s3Service.ListObjects(r.Context(), "")
	if err != nil {
		log.Printf("ListFiles: Failed to list objects: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to list objects")
		return
	}

	log.Printf("ListFiles: Found %d objects", len(objects))

	// Return the files
	respondWithJSON(w, http.StatusOK, objects)
}

// GetFile gets a file from the S3 bucket
func (h *Handler) GetFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	key := vars["key"]

	// Check if it's a .tar.gz file
	if s3.IsTarGzFile(key) {
		respondWithError(w, http.StatusForbidden, "Downloading .tar.gz files is not allowed")
		return
	}

	// Get the file from S3
	result, err := h.s3Service.GetObject(ctx, key)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get object: "+err.Error())
		return
	}
	defer result.Body.Close()

	// Set the content type
	w.Header().Set("Content-Type", *result.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(*result.ContentLength, 10))

	// Copy the file to the response
	_, err = io.Copy(w, result.Body)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to stream file: "+err.Error())
		return
	}
}

// ListMetadata lists metadata for .tar.gz files
func (h *Handler) ListMetadata(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	log.Printf("ListMetadata: Request received with query: %s", r.URL.RawQuery)

	// Parse filter and pagination parameters
	filter := parseMetadataFilter(r)
	page, pageSize := getPaginationParams(r)

	// List objects from S3 directly (skip cache for now since it's causing issues)
	objects, err := h.s3Service.ListObjects(r.Context(), "")
	if err != nil {
		log.Printf("ListMetadata: Error listing objects: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to list objects")
		return
	}

	// Filter for metadata files
	var metadataFiles []s3.Object
	for _, obj := range objects {
		if isSnapshotMetadataFile(obj.Key) {
			metadataFiles = append(metadataFiles, obj)
		}
	}

	log.Printf("ListMetadata: Found %d metadata files", len(metadataFiles))

	// If no metadata files found, return empty list
	if len(metadataFiles) == 0 {
		log.Println("ListMetadata: No metadata files found in S3 bucket")
		result := struct {
			Items []models.Metadata `json:"items"`
			Total int               `json:"total"`
		}{
			Items: []models.Metadata{},
			Total: 0,
		}
		respondWithJSON(w, http.StatusOK, result)
		return
	}

	// Process each metadata file
	var metadataList []models.Metadata
	for _, obj := range metadataFiles {
		// Get metadata content
		result, err := h.s3Service.GetObject(r.Context(), obj.Key)
		if err != nil {
			log.Printf("ListMetadata: Error getting object %s: %v", obj.Key, err)
			continue
		}

		// Read the content
		body, err := io.ReadAll(result.Body)
		result.Body.Close()
		if err != nil {
			log.Printf("ListMetadata: Error reading object %s: %v", obj.Key, err)
			continue
		}

		// Try to parse with the simplified struct first
		var simpleMetadata SimpleMetadata
		err = json.Unmarshal(body, &simpleMetadata)
		if err != nil {
			log.Printf("ListMetadata: Failed to parse simple metadata %s: %v", obj.Key, err)

			// Try to parse as a generic map as a fallback
			var rawData map[string]interface{}
			if jsonErr := json.Unmarshal(body, &rawData); jsonErr == nil {
				// Create a metadata object from the raw data
				metadata := models.Metadata{
					FileName: obj.Key,
					FileSize: *result.ContentLength,
				}

				// Extract fields from the raw data
				if version, ok := rawData["solana_version"].(string); ok {
					metadata.SolanaVersion = version
				}
				if status, ok := rawData["status"].(string); ok {
					metadata.Status = status
				}
				if uploader, ok := rawData["uploaded_by"].(string); ok {
					metadata.UploadedBy = uploader
				}
				if slot, ok := rawData["slot"].(float64); ok {
					metadata.Slot = int64(slot)
				}
				if hash, ok := rawData["hash"].(string); ok {
					metadata.Hash = hash
				}
				if timestamp, ok := rawData["timestamp"].(float64); ok {
					metadata.Timestamp = time.Unix(int64(timestamp), 0)
				}

				// Extract slot and node from filename if it's a snapshot file
				if isSnapshotMetadataFile(obj.Key) {
					slot, node := extractSlotAndNode(obj.Key)
					if metadata.Slot == 0 && slot > 0 {
						metadata.Slot = slot
					}
					if metadata.Node == "" && node != "" {
						metadata.Node = node
					}
					metadata.SlotRange = getSlotRange(metadata.Slot)
				}

				// Apply filter
				if matchesFilter(metadata, filter) {
					metadataList = append(metadataList, metadata)
				}
			}
			continue
		}

		// Create a metadata object from the simple metadata
		metadata := models.Metadata{
			FileName:      obj.Key,
			FileSize:      *result.ContentLength,
			SolanaVersion: simpleMetadata.SolanaVersion,
			Status:        simpleMetadata.Status,
			UploadedBy:    simpleMetadata.UploadedBy,
		}

		// Extract slot and node from filename if it's a snapshot file
		if isSnapshotMetadataFile(obj.Key) {
			slot, node := extractSlotAndNode(obj.Key)
			metadata.Slot = slot
			metadata.Node = node
			metadata.SlotRange = getSlotRange(slot)
		}

		// Apply filter
		if matchesFilter(metadata, filter) {
			metadataList = append(metadataList, metadata)
		}
	}

	log.Printf("ListMetadata: Processed %d metadata files, %d match filters", len(metadataFiles), len(metadataList))

	// Sort by timestamp (newest first) if we have timestamps
	sort.Slice(metadataList, func(i, j int) bool {
		// If timestamps are zero, sort by slot
		if metadataList[i].Timestamp.IsZero() || metadataList[j].Timestamp.IsZero() {
			return metadataList[i].Slot > metadataList[j].Slot
		}
		return metadataList[i].Timestamp.After(metadataList[j].Timestamp)
	})

	// Get total count before pagination
	totalCount := len(metadataList)

	// Apply pagination
	start, end := calculatePaginationBounds(page, pageSize, totalCount)
	if start < totalCount {
		if end > totalCount {
			end = totalCount
		}
		metadataList = metadataList[start:end]
	} else {
		metadataList = []models.Metadata{}
	}

	// Prepare response
	result := struct {
		Items []models.Metadata `json:"items"`
		Total int               `json:"total"`
	}{
		Items: metadataList,
		Total: totalCount,
	}

	log.Printf("ListMetadata: Returning %d items (page %d/%d)", len(metadataList), page, (totalCount+pageSize-1)/pageSize)

	// Return response
	respondWithJSON(w, http.StatusOK, result)
}

// GetMetadata gets metadata for a .tar.gz file
func (h *Handler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Get key from URL
	vars := mux.Vars(r)
	key := vars["key"]
	if key == "" {
		respondWithError(w, http.StatusBadRequest, "Missing key parameter")
		return
	}

	log.Printf("GetMetadata: Fetching metadata for key: %s", key)

	// Get object directly from S3 (skip cache for now since it's causing issues)
	result, err := h.s3Service.GetObject(r.Context(), key)
	if err != nil {
		log.Printf("GetMetadata: Failed to get object %s: %v", key, err)
		respondWithError(w, http.StatusNotFound, "Metadata not found")
		return
	}

	// Read the content
	body, err := io.ReadAll(result.Body)
	result.Body.Close()
	if err != nil {
		log.Printf("GetMetadata: Failed to read object %s: %v", key, err)
		respondWithError(w, http.StatusInternalServerError, "Failed to read metadata")
		return
	}

	// If it's a metadata file, parse it
	if strings.HasSuffix(key, ".json") {
		log.Printf("GetMetadata: Parsing JSON metadata for %s", key)

		// Try to parse with the simplified struct first
		var simpleMetadata SimpleMetadata
		err = json.Unmarshal(body, &simpleMetadata)
		if err != nil {
			log.Printf("GetMetadata: Failed to parse simple metadata %s: %v", key, err)

			// Try to parse as a generic map as a fallback
			var rawData map[string]interface{}
			if jsonErr := json.Unmarshal(body, &rawData); jsonErr == nil {
				log.Printf("GetMetadata: Parsed as raw JSON map: %v", rawData)

				// Create a metadata object from the raw data
				metadata := models.Metadata{
					FileName: key,
					FileSize: *result.ContentLength,
				}

				// Extract fields from the raw data
				if version, ok := rawData["solana_version"].(string); ok {
					metadata.SolanaVersion = version
				}
				if status, ok := rawData["status"].(string); ok {
					metadata.Status = status
				}
				if uploader, ok := rawData["uploaded_by"].(string); ok {
					metadata.UploadedBy = uploader
				}
				if slot, ok := rawData["slot"].(float64); ok {
					metadata.Slot = int64(slot)
				}
				if hash, ok := rawData["hash"].(string); ok {
					metadata.Hash = hash
				}
				if timestamp, ok := rawData["timestamp"].(float64); ok {
					metadata.Timestamp = time.Unix(int64(timestamp), 0)
				}

				// Extract slot and node from filename if it's a snapshot file
				if isSnapshotMetadataFile(key) {
					slot, node := extractSlotAndNode(key)
					if metadata.Slot == 0 && slot > 0 {
						metadata.Slot = slot
					}
					if metadata.Node == "" && node != "" {
						metadata.Node = node
					}
					metadata.SlotRange = getSlotRange(metadata.Slot)
				}

				log.Printf("GetMetadata: Returning metadata from raw JSON: %+v", metadata)
				respondWithJSON(w, http.StatusOK, metadata)
				return
			} else {
				// If we can't parse it as JSON at all, return the raw content
				log.Printf("GetMetadata: Could not parse as JSON at all: %v", jsonErr)
				w.Header().Set("Content-Type", http.DetectContentType(body))
				w.Header().Set("Content-Length", strconv.Itoa(len(body)))
				w.WriteHeader(http.StatusOK)
				w.Write(body)
				return
			}
		}

		// Create a metadata object from the simple metadata
		log.Printf("GetMetadata: Successfully parsed simple metadata: %+v", simpleMetadata)
		metadata := models.Metadata{
			FileName:      key,
			FileSize:      *result.ContentLength,
			SolanaVersion: simpleMetadata.SolanaVersion,
			Status:        simpleMetadata.Status,
			UploadedBy:    simpleMetadata.UploadedBy,
		}

		// Extract slot and node from filename if it's a snapshot file
		if isSnapshotMetadataFile(key) {
			slot, node := extractSlotAndNode(key)
			metadata.Slot = slot
			metadata.Node = node
			metadata.SlotRange = getSlotRange(slot)
		}

		log.Printf("GetMetadata: Returning metadata: %+v", metadata)
		respondWithJSON(w, http.StatusOK, metadata)
		return
	} else {
		// If it's not a metadata file, return the raw content
		log.Printf("GetMetadata: Not a JSON file, returning raw content for %s", key)
		w.Header().Set("Content-Type", http.DetectContentType(body))
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(http.StatusOK)
		w.Write(body)
		return
	}
}

// WebSocketHandler handles WebSocket connections
func (h *Handler) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	h.hub.ServeWs(w, r)
}

// DebugReindex is a debug endpoint to manually trigger the indexing process
func (h *Handler) DebugReindex(w http.ResponseWriter, r *http.Request) {
	log.Println("Manual reindex triggered")

	// Clear the cache for metadata options if available
	if h.cacheService != nil {
		err := h.cacheService.Delete(context.Background(), metadataOptionsKey)
		if err != nil {
			log.Printf("Failed to delete cache: %v", err)
		}
	} else {
		log.Println("Cache service not available, skipping cache deletion")
	}

	// Start indexing in a goroutine
	go func() {
		ctx := context.Background()
		h.indexMetadata(ctx)
	}()

	// Respond with success
	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Reindexing started",
	})
}

// DebugExamineFile examines a specific file and logs its content
func (h *Handler) DebugExamineFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get the first metadata file
	objects, err := h.s3Service.ListObjects(ctx, "")
	if err != nil {
		log.Printf("Failed to list objects: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to list objects")
		return
	}

	// Find the first metadata file
	var metadataFile string
	for _, obj := range objects {
		if obj.IsMetadata && isSnapshotMetadataFile(obj.Key) {
			metadataFile = obj.Key
			break
		}
	}

	if metadataFile == "" {
		respondWithError(w, http.StatusNotFound, "No metadata files found")
		return
	}

	// Get the file content
	result, err := h.s3Service.GetObject(ctx, metadataFile)
	if err != nil {
		log.Printf("Failed to get metadata file %s: %v", metadataFile, err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get metadata file")
		return
	}

	// Read the content
	body, err := io.ReadAll(result.Body)
	result.Body.Close()
	if err != nil {
		log.Printf("Failed to read metadata file %s: %v", metadataFile, err)
		respondWithError(w, http.StatusInternalServerError, "Failed to read metadata file")
		return
	}

	// Log the content
	log.Printf("Content of metadata file %s: %s", metadataFile, string(body))

	// Try to parse as a map
	var rawData map[string]interface{}
	if err := json.Unmarshal(body, &rawData); err != nil {
		log.Printf("Failed to parse metadata file as map: %v", err)
	} else {
		log.Printf("Parsed metadata file as map: %v", rawData)
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "File examined, check logs",
		"file":    metadataFile,
	})
}

// Helper functions

// respondWithError responds with an error
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// respondWithJSON responds with JSON
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to marshal JSON response"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// getPaginationParams gets pagination parameters from the request
func getPaginationParams(r *http.Request) (int, int) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")

	page := 1
	if pageStr != "" {
		pageInt, err := strconv.Atoi(pageStr)
		if err == nil && pageInt > 0 {
			page = pageInt
		}
	}

	pageSize := defaultPageSize
	if pageSizeStr != "" {
		pageSizeInt, err := strconv.Atoi(pageSizeStr)
		if err == nil && pageSizeInt > 0 {
			pageSize = pageSizeInt
			if pageSize > maxPageSize {
				pageSize = maxPageSize
			}
		}
	}

	return page, pageSize
}

// calculatePaginationBounds calculates pagination bounds
func calculatePaginationBounds(page, pageSize, totalItems int) (int, int) {
	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= totalItems {
		start = 0
		end = 0
	} else if end > totalItems {
		end = totalItems
	}

	return start, end
}

// parseMetadataFilter parses metadata filter from the request
func parseMetadataFilter(r *http.Request) models.MetadataFilter {
	// Get query parameters
	query := r.URL.Query()

	// Log the query parameters
	log.Printf("Parsing filter from query parameters: %v", query)

	// Create filter
	filter := models.MetadataFilter{
		SolanaVersion: query.Get("solanaVersion"),
		Status:        query.Get("status"),
		UploadedBy:    query.Get("uploadedBy"),
		Node:          query.Get("node"),
		SlotRange:     query.Get("slotRange"),
		SearchTerm:    query.Get("searchTerm"),
	}

	// Parse min slot
	if minSlot := query.Get("minSlot"); minSlot != "" {
		if val, err := strconv.ParseInt(minSlot, 10, 64); err == nil {
			filter.MinSlot = val
		} else {
			log.Printf("Failed to parse minSlot: %v", err)
		}
	}

	// Parse max slot
	if maxSlot := query.Get("maxSlot"); maxSlot != "" {
		if val, err := strconv.ParseInt(maxSlot, 10, 64); err == nil {
			filter.MaxSlot = val
		} else {
			log.Printf("Failed to parse maxSlot: %v", err)
		}
	}

	// Parse start time
	if startTime := query.Get("startTime"); startTime != "" {
		if t, err := time.Parse("2006-01-02", startTime); err == nil {
			filter.StartTime = t
		} else {
			log.Printf("Failed to parse startTime: %v", err)
		}
	}

	// Parse end time
	if endTime := query.Get("endTime"); endTime != "" {
		if t, err := time.Parse("2006-01-02", endTime); err == nil {
			// Set to end of day
			filter.EndTime = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
		} else {
			log.Printf("Failed to parse endTime: %v", err)
		}
	}

	// Log the parsed filter
	log.Printf("Parsed filter: %+v", filter)

	return filter
}

// matchesFilter checks if metadata matches the filter
func matchesFilter(metadata models.Metadata, filter models.MetadataFilter) bool {
	// Log the filter being applied
	log.Printf("Applying filter: %+v to metadata: %+v", filter, metadata)

	// Check Solana version
	if filter.SolanaVersion != "" && metadata.SolanaVersion != filter.SolanaVersion {
		return false
	}

	// Check status
	if filter.Status != "" && metadata.Status != filter.Status {
		return false
	}

	// Check uploaded by
	if filter.UploadedBy != "" && metadata.UploadedBy != filter.UploadedBy {
		return false
	}

	// Check node
	if filter.Node != "" && metadata.Node != filter.Node {
		return false
	}

	// Check slot range
	if filter.SlotRange != "" && metadata.SlotRange != filter.SlotRange {
		return false
	}

	// Check min slot
	if filter.MinSlot > 0 && metadata.Slot < filter.MinSlot {
		return false
	}

	// Check max slot
	if filter.MaxSlot > 0 && metadata.Slot > filter.MaxSlot {
		return false
	}

	// Check start time
	if !filter.StartTime.IsZero() && metadata.Timestamp.Before(filter.StartTime) {
		return false
	}

	// Check end time
	if !filter.EndTime.IsZero() && metadata.Timestamp.After(filter.EndTime) {
		return false
	}

	// Check search term (case insensitive)
	if filter.SearchTerm != "" {
		searchTerm := strings.ToLower(filter.SearchTerm)

		// Check if search term is in any of the string fields
		if !strings.Contains(strings.ToLower(metadata.SolanaVersion), searchTerm) &&
			!strings.Contains(strings.ToLower(metadata.Status), searchTerm) &&
			!strings.Contains(strings.ToLower(metadata.UploadedBy), searchTerm) &&
			!strings.Contains(strings.ToLower(metadata.Node), searchTerm) &&
			!strings.Contains(strings.ToLower(metadata.Hash), searchTerm) &&
			!strings.Contains(strings.ToLower(metadata.FileName), searchTerm) {
			return false
		}
	}

	// If all checks pass, the metadata matches the filter
	return true
}
