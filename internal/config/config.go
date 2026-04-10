package config

import (
	"flag"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config содержит конфигурацию сервиса, читается из флагов и переменных окружения.
type Config struct {
	ServerAddress   string `env:"SERVER_ADDRESS" env-default:"localhost:8080"`
	BaseURL         string `env:"BASE_URL" env-default:"http://localhost:8080"`
	FileStoragePath string `env:"FILE_STORAGE_PATH" env-default:"storage.json"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
	AuditFile       string `env:"AUDIT_FILE"`
	AuditURL        string `env:"AUDIT_URL"`
	EnableHTTPS     bool   `env:"ENABLE_HTTPS"`
}

// NewConfig читает конфигурацию из флагов командной строки и переменных окружения.
func NewConfig() (*Config, error) {
	cfg := &Config{}

	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "server address")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "base url")
	flag.StringVar(&cfg.FileStoragePath, "f", "storage.json", "file storage path")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database DSN")
	flag.StringVar(&cfg.AuditFile, "audit-file", "", "audit file path")
	flag.StringVar(&cfg.AuditURL, "audit-url", "", "audit url path")
	flag.BoolVar(&cfg.EnableHTTPS, "s", false, "enable HTTPS")
	flag.Parse()

	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
