package api

import (
	"testing"
)

func TestIsSnapshotMetadataFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{
			name:     "valid snapshot metadata file",
			filename: "snapshot-123456789-AutUwEtGwA2wXfH4VqvpoY87d6vQzLkG1V6EugKx8t96.json",
			want:     true,
		},
		{
			name:     "valid snapshot metadata file with longer slot",
			filename: "snapshot-9876543210-AutUwEtGwA2wXfH4VqvpoY87d6vQzLkG1V6EugKx8t96.json",
			want:     true,
		},
		{
			name:     "invalid - not a json file",
			filename: "snapshot-123456789-AutUwEtGwA2wXfH4VqvpoY87d6vQzLkG1V6EugKx8t96.tar.gz",
			want:     false,
		},
		{
			name:     "invalid - wrong format",
			filename: "snapshot_123456789_AutUwEtGwA2wXfH4VqvpoY87d6vQzLkG1V6EugKx8t96.json",
			want:     false,
		},
		{
			name:     "invalid - no slot",
			filename: "snapshot-AutUwEtGwA2wXfH4VqvpoY87d6vQzLkG1V6EugKx8t96.json",
			want:     false,
		},
		{
			name:     "invalid - no node",
			filename: "snapshot-123456789.json",
			want:     false,
		},
		{
			name:     "invalid - random json file",
			filename: "metadata.json",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSnapshotMetadataFile(tt.filename); got != tt.want {
				t.Errorf("isSnapshotMetadataFile() = %v, want %v for filename %s", got, tt.want, tt.filename)
			}
		})
	}
}

func TestExtractSlotAndNode(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantSlot int64
		wantNode string
	}{
		{
			name:     "valid snapshot metadata file",
			filename: "snapshot-123456789-AutUwEtGwA2wXfH4VqvpoY87d6vQzLkG1V6EugKx8t96.json",
			wantSlot: 123456789,
			wantNode: "AutUwEtGwA2wXfH4VqvpoY87d6vQzLkG1V6EugKx8t96",
		},
		{
			name:     "valid snapshot metadata file with longer slot",
			filename: "snapshot-9876543210-AutUwEtGwA2wXfH4VqvpoY87d6vQzLkG1V6EugKx8t96.json",
			wantSlot: 9876543210,
			wantNode: "AutUwEtGwA2wXfH4VqvpoY87d6vQzLkG1V6EugKx8t96",
		},
		{
			name:     "invalid - not a json file",
			filename: "snapshot-123456789-AutUwEtGwA2wXfH4VqvpoY87d6vQzLkG1V6EugKx8t96.tar.gz",
			wantSlot: 0,
			wantNode: "",
		},
		{
			name:     "invalid - wrong format",
			filename: "snapshot_123456789_AutUwEtGwA2wXfH4VqvpoY87d6vQzLkG1V6EugKx8t96.json",
			wantSlot: 0,
			wantNode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSlot, gotNode := extractSlotAndNode(tt.filename)
			if gotSlot != tt.wantSlot {
				t.Errorf("extractSlotAndNode() slot = %v, want %v for filename %s", gotSlot, tt.wantSlot, tt.filename)
			}
			if gotNode != tt.wantNode {
				t.Errorf("extractSlotAndNode() node = %v, want %v for filename %s", gotNode, tt.wantNode, tt.filename)
			}
		})
	}
}
