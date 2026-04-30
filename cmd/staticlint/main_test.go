package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/analysis"
	lintanalyzers "honnef.co/go/tools/analysis/lint"
)

func TestUnwrap(t *testing.T) {
	a1 := &analysis.Analyzer{Name: "a1"}
	a2 := &analysis.Analyzer{Name: "a2"}
	in := []*lintanalyzers.Analyzer{
		{Analyzer: a1},
		{Analyzer: a2},
	}
	got := unwrap(in)
	assert.Equal(t, []*analysis.Analyzer{a1, a2}, got)
}

func TestUnwrap_Empty(t *testing.T) {
	assert.Equal(t, []*analysis.Analyzer{}, unwrap(nil))
}
