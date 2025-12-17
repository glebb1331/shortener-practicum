package config

import (
	"flag"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	ServerAddress   string `env:"SERVER_ADDRESS" envDefault:"localhost:8080"`
	BaseURL         string `env:"BASE_URL" envDefault:"http://localhost:8080"`
	FileStoragePath string `env:"FILE_STORAGE_PATH" envDefault:"C:\\tmp\\short-url-db.json"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}

	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, err
	}

	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "server address")
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "base url")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "file storage path")

	flag.Parse()

	return cfg, nil
}
