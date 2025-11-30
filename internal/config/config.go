package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress   string
	BaseURL         string
	FileStoragePath string
}

func NewConfig() *Config {
	cfg := &Config{}

	defaultServerAddress := ":8080"
	defaultBaseURL := "http://localhost:8080"
	defaultFileStoragePath := "C:\\tmp\\short-url-db.json"

	flag.StringVar(&cfg.ServerAddress, "a", defaultServerAddress, "server address")
	flag.StringVar(&cfg.BaseURL, "b", defaultBaseURL, "base url")
	flag.StringVar(&cfg.FileStoragePath, "f", defaultFileStoragePath, "file storage path")

	flag.Parse()

	if envServerAddress := os.Getenv("SERVER_ADDRESS"); envServerAddress != "" {
		cfg.ServerAddress = envServerAddress
	}

	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}

	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		cfg.FileStoragePath = envFileStoragePath
	}

	return cfg
}
