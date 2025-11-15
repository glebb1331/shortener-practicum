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

	r := chi.NewRouter()

	h := handler.NewHandler(cfg.BaseURL)

	r.Post("/", h.ShortenHandler)

	r.Get("/{id}", h.RedirectHandler)

	log.Println("Сервер запущен", cfg.ServerAddress)
	log.Fatal(http.ListenAndServe(cfg.ServerAddress, r))
}
