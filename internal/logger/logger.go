package logger

import (
	"go.uber.org/zap"
)

// Log — глобальный логгер. По умолчанию nop (не пишет ничего).
// Инициализируется вызовом Initialize.
var Log *zap.Logger = zap.NewNop()

// Initialize настраивает глобальный логгер с заданным уровнем логирования.
func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl

	zl, err := cfg.Build()
	if err != nil {
		return err
	}

	Log = zl
	return nil
}
