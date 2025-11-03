package main

import (
	"log"
	"net/http"

	"github.com/glebb1331/shortener-practicum/internal/config"
	"github.com/glebb1331/shortener-practicum/internal/handler"
	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := config.NewConfig()

	handler.BaseURL = cfg.BaseURL

	r := chi.NewRouter()

	r.Post("/", handler.ShortenHandler)

	r.Get("/{id}", handler.RedirectHandler)

	log.Println("Сервер запущен")
	log.Fatal(http.ListenAndServe(cfg.ServerAddress, r))
}
