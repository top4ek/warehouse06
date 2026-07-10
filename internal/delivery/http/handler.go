package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"warehouse06/internal/domain"
	"warehouse06/internal/repository"
	"warehouse06/internal/sync"
)

var authorDirPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

type Handler struct {
	log        *zap.Logger
	holder     *repository.Holder
	syncStatus *sync.Status
}

func NewHandler(holder *repository.Holder, syncStatus *sync.Status, log *zap.Logger) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return &Handler{
		log:        log,
		holder:     holder,
		syncStatus: syncStatus,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api", func(r chi.Router) {
		r.Get("/status", h.handleGetStatus)
		r.Get("/entries", h.handleGetEntries)
		r.Get("/entries/search", h.handleSearchEntries)
		r.Get("/entries/{path:.*}", h.handleGetEntry)
		r.Get("/authors", h.handleListAuthors)
		r.Get("/authors/{dir}", h.handleGetAuthor)
		r.Get("/tags", h.handleListTags)
		r.Get("/platforms", h.handleListPlatforms)
	})
}

func (h *Handler) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	resp := domain.SyncStatus{
		Syncing: h.syncStatus.Syncing(),
	}

	if t := h.syncStatus.LastSyncedAt(); !t.IsZero() {
		resp.LastSyncedAt = &t
	}

	if commit, ok := h.syncStatus.StorageCommit(); ok {
		resp.StorageCommit = &domain.StorageCommit{
			Hash:        commit.Hash,
			CommittedAt: commit.CommittedAt,
			Subject:     commit.Subject,
		}
	}

	h.respondJSON(w, resp)
}

func (h *Handler) handleGetEntries(w http.ResponseWriter, r *http.Request) {
	limit := parseListLimit(r.URL.Query().Get("limit"))
	offset := parseListOffset(r.URL.Query().Get("offset"))

	opts := repository.EntryListOptions{
		Limit:    limit,
		Offset:   offset,
		Sort:     r.URL.Query().Get("sort"),
		Order:    r.URL.Query().Get("order"),
		Tag:      r.URL.Query().Get("tag"),
		Author:   r.URL.Query().Get("author"),
		Platform: r.URL.Query().Get("platform"),
	}

	repo := h.holder.Get()
	entries, err := repo.GetEntries(r.Context(), opts)
	if err != nil {
		if isRequestCanceled(err) {
			return
		}
		h.log.Error("failed to get entries", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	total, err := repo.CountEntries(r.Context(), opts)
	if err != nil {
		if isRequestCanceled(err) {
			return
		}
		h.log.Error("failed to count entries", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	h.respondJSON(w, domain.EntryListResult{Items: entries, Total: total})
}

func (h *Handler) handleSearchEntries(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing query parameter 'q'")
		return
	}

	limit := parseListLimit(r.URL.Query().Get("limit"))
	offset := parseListOffset(r.URL.Query().Get("offset"))

	ftsQuery := repository.FormatFTSQuery(query)
	if ftsQuery == "" {
		writeJSONError(w, http.StatusBadRequest, "Invalid query parameter 'q'")
		return
	}

	repo := h.holder.Get()
	entries, err := repo.SearchEntries(r.Context(), ftsQuery, limit, offset)
	if err != nil {
		if isRequestCanceled(err) {
			return
		}
		h.log.Error("failed to search entries", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	total, err := repo.CountSearchEntries(r.Context(), ftsQuery)
	if err != nil {
		if isRequestCanceled(err) {
			return
		}
		h.log.Error("failed to count search results", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	h.respondJSON(w, domain.EntryListResult{Items: entries, Total: total})
}

func (h *Handler) handleGetEntry(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "path")
	if decodedPath, err := url.PathUnescape(path); err == nil {
		path = decodedPath
	}
	// The value is used only as a SQL "path = ?" key, never as a filesystem
	// path, so no traversal check is needed (it would over-reject names
	// containing "..").
	if path == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing path")
		return
	}

	entry, err := h.holder.Get().GetEntryByPath(r.Context(), path)
	if err != nil {
		if isRequestCanceled(err) {
			return
		}
		if repository.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, "Not Found")
			return
		}
		h.log.Error("failed to get entry", zap.String("path", path), zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	h.respondJSON(w, entry)
}

func (h *Handler) handleGetAuthor(w http.ResponseWriter, r *http.Request) {
	dir, err := url.PathUnescape(chi.URLParam(r, "dir"))
	if err != nil || dir == "" || !authorDirPattern.MatchString(dir) {
		writeJSONError(w, http.StatusBadRequest, "Missing dir")
		return
	}

	author, err := h.holder.Get().GetAuthorByDir(r.Context(), dir)
	if err != nil {
		if isRequestCanceled(err) {
			return
		}
		if repository.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, "Not Found")
			return
		}
		h.log.Error("failed to get author", zap.String("dir", dir), zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	h.respondJSON(w, author)
}

func (h *Handler) handleListAuthors(w http.ResponseWriter, r *http.Request) {
	authors, err := h.holder.Get().ListAuthors(r.Context())
	if err != nil {
		if isRequestCanceled(err) {
			return
		}
		h.log.Error("failed to list authors", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	h.respondJSON(w, authors)
}

func (h *Handler) handleListTags(w http.ResponseWriter, r *http.Request) {
	tags, err := h.holder.Get().ListTags(r.Context())
	if err != nil {
		if isRequestCanceled(err) {
			return
		}
		h.log.Error("failed to list tags", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	h.respondJSON(w, tags)
}

func (h *Handler) handleListPlatforms(w http.ResponseWriter, r *http.Request) {
	platforms, err := h.holder.Get().ListPlatforms(r.Context())
	if err != nil {
		if isRequestCanceled(err) {
			return
		}
		h.log.Error("failed to list platforms", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	h.respondJSON(w, platforms)
}

func (h *Handler) respondJSON(w http.ResponseWriter, data any) {
	body, err := json.Marshal(data)
	if err != nil {
		h.log.Error("failed to encode json response", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(body); err != nil {
		h.log.Error("failed to write json response", zap.Error(err))
	}
}

func isRequestCanceled(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
