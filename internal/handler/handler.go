package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/glebb1331/shortener-practicum/internal/storage"
)

type Handler struct {
	service *URLService
}

type BatchRequestItem struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type BatchResponseItem struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type BatchResponse []BatchResponseItem

func NewHandler(baseURL string, store storage.Storage) (*Handler, error) {
	service := NewURLService(store, baseURL)

	return &Handler{
		service: service,
	}, nil
}

func (h *Handler) ShortenHandler(w http.ResponseWriter, r *http.Request) {

	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	result, err := h.service.Shorten(context.Background(), string(body), userID)
	if err != nil {
		if errors.Is(err, ErrEmptyURL) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, storage.ErrURLExists) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(result))
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(result))
}

func (h *Handler) RedirectHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/")

	originalURL, err := h.service.Resolve(id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, storage.ErrURLDeleted) {
			http.Error(w, "gone", http.StatusGone)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if !strings.HasPrefix(originalURL, "http://") &&
		!strings.HasPrefix(originalURL, "https://") {
		originalURL = "http://" + originalURL
	}

	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
}

func (h *Handler) APIShortenHandler(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	defer r.Body.Close()

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	result, err := h.service.Shorten(context.Background(), req.URL, userID)
	if err != nil {
		if errors.Is(err, ErrEmptyURL) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, storage.ErrURLExists) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ShortenResponse{Result: result})
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := ShortenResponse{Result: result}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) PingHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Ping(r.Context()); err != nil {
		http.Error(w, "storage unavailable", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) APIShortenBatchHandler(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "missing userID", http.StatusUnauthorized)
		return
	}

	var batchRequests []BatchRequestItem
	if err := json.NewDecoder(r.Body).Decode(&batchRequests); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(batchRequests) == 0 {
		http.Error(w, "empty batch", http.StatusBadRequest)
		return
	}

	result, err := h.service.ShortenBatch(context.Background(), batchRequests, userID)
	if err != nil {
		if errors.Is(err, ErrEmptyURL) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	records, err := h.service.GetByUserID(context.Background(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if len(records) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp := make([]map[string]string, 0, len(records))
	for _, rec := range records {
		resp = append(resp, map[string]string{
			"short_url":    h.service.BaseURL() + "/" + rec.ID,
			"original_url": rec.OriginalURL,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) DeleteUserURLs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	defer r.Body.Close()

	var ids []string
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(ids) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteURLs(r.Context(), userID, ids); err != nil {
		if errors.Is(err, storage.ErrNotOwner) {
			http.Error(w, "user is not owner of some urls", http.StatusForbidden)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
