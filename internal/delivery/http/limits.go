package http

import "strconv"

const (
	defaultListLimit = 50
	maxListLimit     = 100
	// maxListOffset bounds OFFSET so a crafted value cannot force SQLite
	// into an arbitrarily deep scan.
	maxListOffset = 100000
)

func parseListLimit(limitStr string) int {
	limit := defaultListLimit
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	return limit
}

func parseListOffset(offsetStr string) int {
	offset := 0
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}
	if offset > maxListOffset {
		offset = maxListOffset
	}
	return offset
}
