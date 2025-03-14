<script setup>
import { ref, onMounted, watch, onUnmounted } from 'vue'
import FileList from './components/FileList.vue'
import MetadataViewer from './components/MetadataViewer.vue'
import SearchFilter from './components/SearchFilter.vue'

const files = ref([])
const metadata = ref(null)
const loading = ref(true)
const error = ref(null)
const searchFilters = ref({
  solanaVersion: '',
  solanaFeatureSet: '',
  status: '',
  uploadedBy: '',
  startTime: '',
  endTime: '',
  minSlot: '',
  maxSlot: '',
  searchTerm: '',
  page: 1,
  pageSize: 20
})

// Debounce timer for filtering
let debounceTimer = null

// WebSocket connection
let ws = null

// Connect to WebSocket
const connectWebSocket = () => {
  // In development, the WebSocket URL is proxied through Vite
  // In production, it's at the same origin
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/api/ws`
  
  console.log(`Connecting to WebSocket at ${wsUrl}`)
  
  ws = new WebSocket(wsUrl)
  
  ws.onopen = () => {
    console.log('WebSocket connected')
  }
  
  ws.onmessage = (event) => {
    try {
      const newFiles = JSON.parse(event.data)
      files.value = newFiles
    } catch (err) {
      console.error('Failed to parse WebSocket message:', err)
    }
  }
  
  ws.onclose = () => {
    console.log('WebSocket disconnected')
    // Try to reconnect after a delay
    setTimeout(connectWebSocket, 5000)
  }
  
  ws.onerror = (err) => {
    console.error('WebSocket error:', err)
    ws.close()
  }
}

// Fetch files from the API
const fetchFiles = async () => {
  loading.value = true
  error.value = null
  
  try {
    const response = await fetch('/api/files')
    if (!response.ok) {
      throw new Error(`Failed to fetch files: ${response.statusText}`)
    }
    
    files.value = await response.json()
  } catch (err) {
    error.value = err.message
    console.error(err)
  } finally {
    loading.value = false
  }
}

// Fetch metadata for a file
const fetchMetadata = async (key) => {
  loading.value = true
  error.value = null
  metadata.value = null
  
  try {
    console.log(`Fetching metadata for ${key}...`)
    const response = await fetch(`/api/metadata/${encodeURIComponent(key)}`)
    if (!response.ok) {
      throw new Error(`Failed to fetch metadata: ${response.statusText}`)
    }
    
    const data = await response.json()
    console.log('Received metadata:', data)
    
    // Ensure we have a valid metadata object
    if (!data) {
      throw new Error('Received empty metadata')
    }
    
    // Add file name if not present
    if (!data.file_name && !data.fileName) {
      data.file_name = key
    }
    
    metadata.value = data
  } catch (err) {
    error.value = err.message
    console.error('Error fetching metadata:', err)
  } finally {
    loading.value = false
  }
}

// Handle file selection
const handleFileSelect = (file) => {
  if (file.IsTarGz) {
    fetchMetadata(file.Key)
  } else if (file.IsMetadata) {
    fetchMetadata(file.Key)
  }
}

// Handle search filter changes with debounce
const handleFilterChange = (filters) => {
  // Clear previous timer
  if (debounceTimer) {
    clearTimeout(debounceTimer)
  }
  
  // Update filters immediately
  searchFilters.value = { ...filters }
  
  // Debounce the API call
  debounceTimer = setTimeout(() => {
    fetchFilteredMetadata()
  }, 300) // 300ms debounce
}

// Fetch filtered metadata
const fetchFilteredMetadata = async () => {
  loading.value = true
  error.value = null
  
  try {
    // Build query string from filters
    const params = new URLSearchParams()
    
    console.log('Fetching filtered metadata with filters:', searchFilters.value)
    
    for (const [key, value] of Object.entries(searchFilters.value)) {
      if (value) {
        params.append(key, value)
      }
    }
    
    console.log(`Fetching filtered metadata with params: ${params.toString()}`)
    const response = await fetch(`/api/metadata?${params.toString()}`)
    if (!response.ok) {
      const errorText = await response.text()
      console.error(`Failed to fetch metadata: ${response.statusText}`, errorText)
      throw new Error(`Failed to fetch metadata: ${response.statusText}`)
    }
    
    const data = await response.json()
    console.log('Filtered metadata response:', data)
    
    // Check if data is valid and has items property
    if (!data || !data.items) {
      console.error('Invalid response data:', data)
      files.value = []
      return
    }
    
    // Map items to files
    files.value = data.items.map(item => ({
      Key: item.file_name || item.fileName || 'Unknown',
      Size: item.file_size || item.fileSize || 0,
      LastModified: item.timestamp ? new Date(item.timestamp) : new Date(),
      IsMetadata: true,
      IsTarGz: false,
      Metadata: item
    }))
  } catch (err) {
    error.value = err.message
    console.error('Error fetching filtered metadata:', err)
    files.value = [] // Reset files on error
  } finally {
    loading.value = false
  }
}

// Initialize the component
onMounted(() => {
  fetchFiles()
  connectWebSocket()
})

// Clean up WebSocket on component unmount
onUnmounted(() => {
  if (ws) {
    ws.close()
  }
  
  if (debounceTimer) {
    clearTimeout(debounceTimer)
  }
})
</script>

<template>
  <div class="min-h-screen bg-gray-100">
    <header class="bg-white shadow">
      <div class="container mx-auto py-4 px-4 flex items-center">
        <h1 class="text-2xl font-bold text-gray-900">S3 Bucket Browser</h1>
        <div class="ml-auto text-sm text-gray-500">
          <span v-if="files.length" class="font-medium">{{ files.length }}</span> files found
        </div>
      </div>
    </header>
    
    <main class="container mx-auto py-6 px-4">
      <div class="py-6">
        <div v-if="error" class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
          {{ error }}
        </div>
        
        <div class="flex flex-col lg:flex-row gap-6">
          <!-- Search and filters -->
          <div class="w-full lg:w-1/4">
            <SearchFilter @filter-change="handleFilterChange" />
          </div>
          
          <!-- File list and metadata viewer -->
          <div class="w-full lg:w-3/4">
            <div class="bg-white shadow rounded-lg">
              <div v-if="loading" class="p-8 text-center">
                <div class="animate-spin rounded-full h-10 w-10 border-b-2 border-blue-500 mx-auto"></div>
                <p class="mt-4 text-gray-600">Loading...</p>
              </div>
              
              <div v-else class="flex flex-col gap-6 p-6">
                <!-- Metadata viewer -->
                <div class="border rounded-lg overflow-hidden">
                  <div class="bg-gray-50 px-4 py-3 border-b">
                    <h2 class="text-lg font-semibold text-gray-900">Metadata</h2>
                  </div>
                  <div class="p-4">
                    <MetadataViewer :metadata="metadata" />
                  </div>
                </div>
                
                <!-- File list -->
                <div class="border rounded-lg overflow-hidden">
                  <div class="bg-gray-50 px-4 py-3 border-b">
                    <h2 class="text-lg font-semibold text-gray-900">Files</h2>
                  </div>
                  <div class="p-0">
                    <FileList :files="files" @file-select="handleFileSelect" />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>
    
    <footer class="bg-white border-t border-gray-200 mt-auto">
      <div class="container mx-auto py-4 px-4">
        <p class="text-sm text-gray-500 text-center">
          S3 Bucket Browser - Efficiently explore S3 bucket contents
        </p>
      </div>
    </footer>
  </div>
</template>

<style>
body {
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  margin: 0;
  padding: 0;
  width: 100%;
}

#app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  width: 100%;
}

.container {
  width: 100%;
  padding-right: 1rem;
  padding-left: 1rem;
  margin-right: auto;
  margin-left: auto;
}

@media (min-width: 640px) {
  .container {
    max-width: 640px;
  }
}

@media (min-width: 768px) {
  .container {
    max-width: 768px;
  }
}

@media (min-width: 1024px) {
  .container {
    max-width: 1024px;
  }
}

@media (min-width: 1280px) {
  .container {
    max-width: 1280px;
  }
}

@media (min-width: 1536px) {
  .container {
    max-width: 1536px;
  }
}

footer {
  margin-top: auto;
}
</style>
