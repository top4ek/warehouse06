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
	// StorageURL is the public web URL of the content repository, omitted
	// when none is configured or it is not an http(s) URL.
	StorageURL string `json:"storage_url,omitempty"`
}
