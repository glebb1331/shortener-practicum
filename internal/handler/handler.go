package handler

import (
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
	ShortURL      string `json:"original_url"`
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	result, err := h.service.Shorten(string(body))
	if err != nil {
		if errors.Is(err, ErrEmptyURL) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, storage.ErrURLExists) {
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

	originalURL, ok := h.service.Resolve(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
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

	defer r.Body.Close()

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	result, err := h.service.Shorten(req.URL)
	if err != nil {
		if err == ErrEmptyURL {
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

	var batchRequests []BatchRequestItem
	if err := json.NewDecoder(r.Body).Decode(&batchRequests); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(batchRequests) == 0 {
		http.Error(w, "empty batch", http.StatusBadRequest)
		return
	}

	result, err := h.service.ShortenBatch(batchRequests)
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
