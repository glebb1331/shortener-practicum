package config

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config содержит конфигурацию сервиса, читается из флагов и переменных окружения.
type Config struct {
	ServerAddress   string `env:"SERVER_ADDRESS" env-default:"localhost:8080" json:"server_address"`
	BaseURL         string `env:"BASE_URL" env-default:"http://localhost:8080" json:"base_url"`
	FileStoragePath string `env:"FILE_STORAGE_PATH" env-default:"storage.json" json:"file_storage_path"`
	DatabaseDSN     string `env:"DATABASE_DSN" json:"database_dsn"`
	AuditFile       string `env:"AUDIT_FILE"`
	AuditURL        string `env:"AUDIT_URL"`
	EnableHTTPS     bool   `env:"ENABLE_HTTPS" json:"enable_https"`
}

// NewConfig читает конфигурацию из флагов командной строки и переменных окружения.
func NewConfig() (*Config, error) {
	cfg := &Config{}

	var configFile string
	flag.StringVar(&configFile, "c", "", "path to JSON config file")
	flag.StringVar(&configFile, "config", "", "path to JSON config file")
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

	// Определяем путь к файлу конфигурации: флаг имеет приоритет над переменной окружения.
	if configFile == "" {
		configFile = os.Getenv("CONFIG")
	}

	// Загружаем JSON-конфигурацию, если указан файл.
	// Значения из файла применяются только если они не были заданы через флаги или переменные окружения.
	if configFile != "" {
		if err := applyJSONConfig(cfg, configFile); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// applyJSONConfig читает JSON-файл и применяет значения только для полей,
// которые не были установлены через флаги или переменные окружения.
func applyJSONConfig(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var fileCfg Config
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return err
	}

	// Собираем множество флагов, которые были явно переданы в командной строке.
	setFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	// Применяем значения из JSON только если поле не задано флагом и не задано переменной окружения.
	if !setFlags["a"] && os.Getenv("SERVER_ADDRESS") == "" && fileCfg.ServerAddress != "" {
		cfg.ServerAddress = fileCfg.ServerAddress
	}
	if !setFlags["b"] && os.Getenv("BASE_URL") == "" && fileCfg.BaseURL != "" {
		cfg.BaseURL = fileCfg.BaseURL
	}
	if !setFlags["f"] && os.Getenv("FILE_STORAGE_PATH") == "" && fileCfg.FileStoragePath != "" {
		cfg.FileStoragePath = fileCfg.FileStoragePath
	}
	if !setFlags["d"] && os.Getenv("DATABASE_DSN") == "" && fileCfg.DatabaseDSN != "" {
		cfg.DatabaseDSN = fileCfg.DatabaseDSN
	}
	if !setFlags["s"] && os.Getenv("ENABLE_HTTPS") == "" && fileCfg.EnableHTTPS {
		cfg.EnableHTTPS = fileCfg.EnableHTTPS
	}

	return nil
}
