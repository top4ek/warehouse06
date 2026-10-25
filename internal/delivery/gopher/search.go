package gopher

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
)

const searchCacheSize = 512

var errSearchExpired = errors.New("search expired; enter query again")

// Long searches use bounded, process-local keys so menu selectors stay within
// the client line limit. Existing keys survive until 512 distinct later queries.
func (s *Server) searchURL(query string) *url.URL {
	u := &url.URL{Path: "/search", RawQuery: url.Values{"q": {query}}.Encode()}
	if len(s.line('1', "Предыдущая страница", withOffset(u, int(^uint(0)>>1)-pageSize))) <= 255 {
		return u
	}
	key := fmt.Sprintf("%x", sha256.Sum256([]byte(query)))
	s.mu.Lock()
	if _, ok := s.queries[key]; !ok {
		if len(s.queryKeys) == searchCacheSize {
			delete(s.queries, s.queryKeys[0])
			s.queryKeys = s.queryKeys[1:]
		}
		s.queries[key] = query
		s.queryKeys = append(s.queryKeys, key)
	}
	s.mu.Unlock()
	u.RawQuery = url.Values{"key": {key}}.Encode()
	return u
}

func (s *Server) searchQuery(key string) (string, error) {
	s.mu.Lock()
	query, ok := s.queries[key]
	s.mu.Unlock()
	if !ok {
		return "", errSearchExpired
	}
	return query, nil
}
