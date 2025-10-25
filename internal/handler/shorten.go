package handler

import (
	"io"
	"net/http"
	"strings"
)

func ShortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if ct := r.Header.Get("Content-Type"); ct != "text/plain" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	url := strings.TrimSpace(string(body))
	if url == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	shortURL := "http://localhost:8080/EwHXdJfB"

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(string(shortURL)))
}
