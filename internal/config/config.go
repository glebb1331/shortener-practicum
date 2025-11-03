package config

import "flag"

type Config struct {
	ServerAddress string
	BaseURL       string
}

func NewConfig() *Config {
	config := &Config{}

	flag.StringVar(&config.ServerAddress, "a", "localhost:8080", "server address")
	flag.StringVar(&config.BaseURL, "b", "http://localhost:8080", "base url")

	flag.Parse()

	return config
}
