package handler

import (
	"io"
	"net/http"
	"strings"
)

type Handler struct {
	BaseURL string
	store   *URLStore
}

func NewHandler(baseURL string) *Handler {
	return &Handler{
		BaseURL: baseURL,
		store:   NewURLStore(),
	}
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

	url := strings.TrimSpace(string(body))
	if url == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id := h.store.Save(url)
	shortURL := h.BaseURL + "/" + id

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func (h *Handler) RedirectHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/")

	originalURL, ok := h.store.Get(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}
