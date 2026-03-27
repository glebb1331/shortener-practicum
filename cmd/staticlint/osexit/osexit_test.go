package osexit_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/glebb1331/shortener-practicum/cmd/staticlint/osexit"
)

// TestAnalyzer проверяет работу анализатора на тестовых данных из директории testdata.
func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), osexit.Analyzer, "a")
}
