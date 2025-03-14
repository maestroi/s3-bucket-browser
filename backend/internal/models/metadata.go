package models

import "time"

// Metadata represents the metadata for a .tar.gz file
type Metadata struct {
	SolanaVersion    string    `json:"solana_version"`
	SolanaFeatureSet int       `json:"solana_feature_set"`
	Slot             int64     `json:"slot"`
	Timestamp        time.Time `json:"timestamp"`
	Hash             string    `json:"hash"`
	Status           string    `json:"status"`
	UploadedBy       string    `json:"uploaded_by"`
	UploadedAt       time.Time `json:"uploaded_at"`
	FileSize         int64     `json:"file_size"`
	FileName         string    `json:"file_name"`
	Node             string    `json:"node,omitempty"`
	SlotRange        string    `json:"slot_range,omitempty"`
	// Additional fields can be added as needed
}

// MetadataList represents a list of metadata with pagination
type MetadataList struct {
	Items      []Metadata `json:"items"`
	TotalCount int        `json:"total_count"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
}

// MetadataFilter represents a filter for metadata
type MetadataFilter struct {
	SolanaVersion    string    `json:"solana_version"`
	SolanaFeatureSet int       `json:"solana_feature_set"`
	Status           string    `json:"status"`
	UploadedBy       string    `json:"uploaded_by"`
	Node             string    `json:"node"`
	SlotRange        string    `json:"slot_range"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
	MinSlot          int64     `json:"min_slot"`
	MaxSlot          int64     `json:"max_slot"`
	SearchTerm       string    `json:"search_term"`
	Page             int       `json:"page"`
	PageSize         int       `json:"page_size"`
}
