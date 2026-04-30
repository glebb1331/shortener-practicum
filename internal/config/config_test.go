package config

import (
	"flag"
	"os"
	"path/filepath"
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

func TestApplyJSONConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	jsonBody := `{
		"server_address": "127.0.0.1:7777",
		"base_url": "http://shortener.local",
		"file_storage_path": "/tmp/storage.json",
		"database_dsn": "postgres://x",
		"enable_https": true,
		"trusted_subnet": "10.0.0.0/8",
		"grpc_address": ":3201"
	}`
	require.NoError(t, os.WriteFile(path, []byte(jsonBody), 0o600))

	resetFlags()
	cfg := &Config{}
	require.NoError(t, applyJSONConfig(cfg, path))

	assert.Equal(t, "127.0.0.1:7777", cfg.ServerAddress)
	assert.Equal(t, "http://shortener.local", cfg.BaseURL)
	assert.Equal(t, "/tmp/storage.json", cfg.FileStoragePath)
	assert.Equal(t, "postgres://x", cfg.DatabaseDSN)
	assert.True(t, cfg.EnableHTTPS)
	assert.Equal(t, "10.0.0.0/8", cfg.TrustedSubnet)
	assert.Equal(t, ":3201", cfg.GRPCAddress)
}

func TestApplyJSONConfig_FlagWins(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"server_address":"json:1111"}`), 0o600))

	// Имитируем флаг "-a" — applyJSONConfig не должен переписать поле.
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	var addr string
	flag.StringVar(&addr, "a", "", "addr")
	require.NoError(t, flag.CommandLine.Parse([]string{"-a", "flag:2222"}))

	cfg := &Config{ServerAddress: "flag:2222"}
	require.NoError(t, applyJSONConfig(cfg, path))
	assert.Equal(t, "flag:2222", cfg.ServerAddress)
}

func TestApplyJSONConfig_FileMissing(t *testing.T) {
	cfg := &Config{}
	err := applyJSONConfig(cfg, filepath.Join(t.TempDir(), "missing.json"))
	require.Error(t, err)
}

func TestApplyJSONConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("not-json"), 0o600))

	cfg := &Config{}
	require.Error(t, applyJSONConfig(cfg, path))
}
