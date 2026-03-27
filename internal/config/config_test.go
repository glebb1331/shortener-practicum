package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

func TestNewConfig_Defaults(t *testing.T) {
	resetFlags()

	cfg, err := NewConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "localhost:8080", cfg.ServerAddress)
	assert.Equal(t, "http://localhost:8080", cfg.BaseURL)
	assert.Equal(t, "storage.json", cfg.FileStoragePath)
	assert.Empty(t, cfg.DatabaseDSN)
}

func TestNewConfig_FromEnv(t *testing.T) {
	resetFlags()

	t.Setenv("SERVER_ADDRESS", "0.0.0.0:9090")
	t.Setenv("BASE_URL", "http://myserver:9090")

	cfg, err := NewConfig()
	require.NoError(t, err)

	assert.Equal(t, "0.0.0.0:9090", cfg.ServerAddress)
	assert.Equal(t, "http://myserver:9090", cfg.BaseURL)
}
