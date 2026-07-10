package repository

import (
	"os"
	"time"
)

// ResolveStartupDSN chooses which on-disk DB file to open on process start.
// For blue-green rebuilds we may have both primary and peer on disk; opening the
// freshest existing one provides a warm start while sync rebuilds the peer.
func ResolveStartupDSN(primaryDSN string) string {
	if IsInMemoryDSN(primaryDSN) {
		return primaryDSN
	}

	peer := primaryDSN + ".next"

	pInfo, pErr := os.Stat(primaryDSN)
	nInfo, nErr := os.Stat(peer)

	if pErr != nil && nErr == nil {
		return peer
	}
	if nErr != nil && pErr == nil {
		return primaryDSN
	}
	if pErr != nil || nErr != nil {
		return primaryDSN
	}

	// Prefer the newer file; if equal modtime, prefer the larger one.
	pMod, nMod := pInfo.ModTime(), nInfo.ModTime()
	if nMod.After(pMod.Add(250 * time.Millisecond)) {
		return peer
	}
	if pMod.After(nMod.Add(250 * time.Millisecond)) {
		return primaryDSN
	}
	if nInfo.Size() > pInfo.Size() {
		return peer
	}
	return primaryDSN
}
