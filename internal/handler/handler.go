package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/glebb1331/shortener-practicum/internal/logger"
	"github.com/glebb1331/shortener-practicum/internal/storage"
	"go.uber.org/zap"
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

type UserURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

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

	result, err := h.service.Shorten(r.Context(), string(body), userID)
	if err != nil {
		if errors.Is(err, ErrEmptyURL) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if errors.Is(err, storage.ErrURLExists) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(result))
			return
		}
		logger.Log.Error("Internal server error",
			zap.Error(err),
			zap.String("userID", userID),
			zap.String("method", r.Method),
		)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(result))
}

func (h *Handler) RedirectHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/")

	originalURL, err := h.service.Resolve(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		if errors.Is(err, storage.ErrURLDeleted) {
			http.Error(w, http.StatusText(http.StatusGone), http.StatusGone)
			return
		}

		logger.Log.Error("Internal server error",
			zap.Error(err),
			zap.String("id", id),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
		)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	defer r.Body.Close()

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	result, err := h.service.Shorten(r.Context(), req.URL, userID)
	if err != nil {
		if errors.Is(err, ErrEmptyURL) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if errors.Is(err, storage.ErrURLExists) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ShortenResponse{Result: result})
			return
		}

		logger.Log.Error("Internal server error",
			zap.Error(err),
			zap.String("userID", userID),
			zap.String("method", r.Method),
		)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp := ShortenResponse{Result: result}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) PingHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Ping(r.Context()); err != nil {

		logger.Log.Error("Storage ping failed",
			zap.Error(err),
			zap.String("method", r.Method),
			zap.String("endpoint", "/ping"),
		)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) APIShortenBatchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json") // Устанавливаем заранее

	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "content type must be application/json"})
		return
	}

	defer r.Body.Close()

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing userID"})
		return
	}

	var batchRequests []BatchRequestItem
	if err := json.NewDecoder(r.Body).Decode(&batchRequests); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	if len(batchRequests) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "empty batch"})
		return
	}

	result, err := h.service.ShortenBatch(r.Context(), batchRequests, userID)
	if err != nil {
		if errors.Is(err, ErrEmptyURL) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": http.StatusText(http.StatusBadRequest)})
			return
		}

		logger.Log.Error("Internal server error",
			zap.Error(err),
			zap.String("userID", userID),
			zap.String("batchSize", strconv.Itoa(len(batchRequests))),
			zap.String("method", r.Method),
		)

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": http.StatusText(http.StatusInternalServerError)})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	records, err := h.service.GetByUserID(r.Context(), userID)
	if err != nil {

		logger.Log.Error("Internal server error",
			zap.Error(err),
			zap.String("userID", userID),
			zap.String("method", r.Method),
		)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if len(records) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp := make([]UserURLResponse, 0, len(records))
	for _, rec := range records {
		shortURL, err := url.JoinPath(h.service.BaseURL(), rec.ID)
		if err != nil {
			logger.Log.Error("Internal server error", zap.Error(err))
			continue
		}

		resp = append(resp, UserURLResponse{
			ShortURL:    shortURL,
			OriginalURL: rec.OriginalURL,
		})
	}

	if len(resp) == 0 && len(records) > 0 {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
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
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
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
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		logger.Log.Error("Internal server error",
			zap.Error(err),
			zap.String("userID", userID),
			zap.Strings("ids", ids),
			zap.String("method", r.Method),
		)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
