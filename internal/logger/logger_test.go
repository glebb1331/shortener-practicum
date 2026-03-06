package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitialize_ValidLevel(t *testing.T) {
	err := Initialize("info")
	assert.NoError(t, err)
	assert.NotNil(t, Log)
}

func TestInitialize_DebugLevel(t *testing.T) {
	err := Initialize("debug")
	assert.NoError(t, err)
}

func TestInitialize_InvalidLevel(t *testing.T) {
	err := Initialize("invalid-level")
	assert.Error(t, err)
}
