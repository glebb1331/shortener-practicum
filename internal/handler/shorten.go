package handler

import (
	"io"
	"net/http"
	"strings"
)

var BaseURL string

var urlStore = make(map[string]string)

func ShortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

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
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	id := "id1"
	urlStore[id] = url

	shortURL := BaseURL + "/" + id

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(string(shortURL)))
}
