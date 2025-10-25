package handler

import (
	"net/http"
	"strings"
)

func RedirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	originalURL := "https://practicum.yandex.ru/"

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
