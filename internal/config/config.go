package config

import (
	"flag"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	ServerAddress   string `env:"SERVER_ADDRESS" env-default:"localhost:8080"`
	BaseURL         string `env:"BASE_URL" env-default:"http://localhost:8080"`
	FileStoragePath string `env:"FILE_STORAGE_PATH" env-default:"storage.json"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}

	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "server address")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "base url")
	flag.StringVar(&cfg.FileStoragePath, "f", "storage.json", "file storage path")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database DSN")
	flag.Parse()

	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
