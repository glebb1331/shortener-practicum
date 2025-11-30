package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress string
	BaseURL       string
}

func NewConfig() *Config {
	cfg := &Config{}

	defaultServerAddress := ":8080"
	defaultBaseURL := "http://localhost:8080"

	// Определяем флаги
	flag.StringVar(&cfg.ServerAddress, "a", defaultServerAddress, "server address")
	flag.StringVar(&cfg.BaseURL, "b", defaultBaseURL, "base url")

	flag.Parse()

	if envServerAddress := os.Getenv("SERVER_ADDRESS"); envServerAddress != "" {
		cfg.ServerAddress = envServerAddress
	}

	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}

	return cfg
}
