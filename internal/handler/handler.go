package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/glebb1331/shortener-practicum/internal/audit"
	"github.com/glebb1331/shortener-practicum/internal/logger"
	"github.com/glebb1331/shortener-practicum/internal/storage"
	"github.com/glebb1331/shortener-practicum/internal/usecase"
	"go.uber.org/zap"
)

// Handler содержит HTTP-обработчики сервиса сокращения ссылок.
type Handler struct {
	service       *usecase.URLService
	auditSvc      *audit.AuditService
	trustedSubnet *net.IPNet
}

// ShortenRequest — тело запроса для POST /api/shorten.
type ShortenRequest struct {
	URL string `json:"url"`
}

// ShortenResponse — тело ответа для POST /api/shorten.
type ShortenResponse struct {
	Result string `json:"result"`
}

// UserURLResponse — пара короткий/оригинальный URL для ответа пользователю.
type UserURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// NewHandler создаёт Handler с заданным сервисом и сервисом аудита.
func NewHandler(svc *usecase.URLService, auditSvc *audit.AuditService) *Handler {
	return &Handler{
		service:  svc,
		auditSvc: auditSvc,
	}
}

// SetTrustedSubnet устанавливает доверенную подсеть для эндпоинта /api/internal/stats.
// Пустая строка отключает доступ к эндпоинту.
func (h *Handler) SetTrustedSubnet(cidr string) error {
	if cidr == "" {
		h.trustedSubnet = nil
		return nil
	}
	_, subnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	h.trustedSubnet = subnet
	return nil
}

// ShortenHandler обрабатывает POST / — принимает plain-text URL, возвращает короткую ссылку.
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

	originalURL := string(body)

	result, err := h.service.Shorten(r.Context(), originalURL, userID)
	if err != nil {
		if errors.Is(err, usecase.ErrEmptyURL) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if errors.Is(err, storage.ErrURLExists) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(result))
			h.auditSvc.Notify(audit.AuditEvent{Action: "shorten", UserID: userID, URL: originalURL})
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
	h.auditSvc.Notify(audit.AuditEvent{Action: "shorten", UserID: userID, URL: originalURL})
}

// RedirectHandler обрабатывает GET /{id} — перенаправляет на оригинальный URL.
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

	userID := r.Header.Get("X-User-ID")
	h.auditSvc.Notify(audit.AuditEvent{Action: "follow", UserID: userID, URL: originalURL})
	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}

// APIShortenHandler обрабатывает POST /api/shorten — принимает JSON с URL, возвращает JSON с короткой ссылкой.
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
		if errors.Is(err, usecase.ErrEmptyURL) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if errors.Is(err, storage.ErrURLExists) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ShortenResponse{Result: result})
			h.auditSvc.Notify(audit.AuditEvent{Action: "shorten", UserID: userID, URL: req.URL})
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
	h.auditSvc.Notify(audit.AuditEvent{Action: "shorten", UserID: userID, URL: req.URL})
}

// PingHandler обрабатывает GET /ping — проверяет доступность хранилища.
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

// APIShortenBatchHandler обрабатывает POST /api/shorten/batch — пакетное сокращение ссылок.
func (h *Handler) APIShortenBatchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

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

	var batchRequests []usecase.BatchRequestItem
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
		if errors.Is(err, usecase.ErrEmptyURL) {
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

// GetUserURLs обрабатывает GET /api/user/urls — возвращает все ссылки текущего пользователя.
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

// DeleteUserURLs обрабатывает DELETE /api/user/urls — помечает ссылки пользователя как удалённые.
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

// StatsResponse — тело ответа для GET /api/internal/stats.
type StatsResponse struct {
	URLs  int `json:"urls"`
	Users int `json:"users"`
}

// InternalStatsHandler обрабатывает GET /api/internal/stats — возвращает статистику сервиса.
// Доступ разрешён только клиентам, чей IP-адрес из заголовка X-Real-IP входит в доверенную подсеть.
func (h *Handler) InternalStatsHandler(w http.ResponseWriter, r *http.Request) {
	if h.trustedSubnet == nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	clientIP := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP")))
	if clientIP == nil || !h.trustedSubnet.Contains(clientIP) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	urls, users, err := h.service.Stats(r.Context())
	if err != nil {
		logger.Log.Error("Failed to get stats",
			zap.Error(err),
			zap.String("method", r.Method),
		)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(StatsResponse{URLs: urls, Users: users})
}
