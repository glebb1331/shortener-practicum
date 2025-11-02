package main

import (
	"log"
	"net/http"

	"github.com/glebb1331/shortener-practicum/internal/handler"
	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Post("/", handler.ShortenHandler)

	r.Get("/{id}", handler.RedirectHandler)

	log.Println("Сервер запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
