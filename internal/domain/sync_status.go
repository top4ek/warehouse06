package domain

import "time"

type StorageCommit struct {
	Hash        string    `json:"hash"`
	CommittedAt time.Time `json:"committed_at"`
	Subject     string    `json:"subject,omitempty"`
}

type SyncStatus struct {
	Syncing       bool           `json:"syncing"`
	LastSyncedAt  *time.Time     `json:"last_synced_at,omitempty"`
	StorageCommit *StorageCommit `json:"storage_commit,omitempty"`
}
