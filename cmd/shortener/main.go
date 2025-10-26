package main

import (
	"log"
	"net/http"

	"github.com/glebb1331/shortener-practicum/internal/handler"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/" {
			handler.ShortenHandler(w, r)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path != "/" {
			handler.RedirectHandler(w, r)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	})

	log.Println("Сервер запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
