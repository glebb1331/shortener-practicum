package pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type buf struct {
	data []byte
}

func (b *buf) Reset() {
	b.data = b.data[:0]
}

func TestPool_GetReturnsNewObject(t *testing.T) {
	p := New(func() *buf { return &buf{data: make([]byte, 0, 8)} })

	b := p.Get()
	assert.NotNil(t, b)
	assert.Equal(t, 0, len(b.data))
}

func TestPool_PutResetsObject(t *testing.T) {
	p := New(func() *buf { return &buf{data: make([]byte, 0, 8)} })

	b := p.Get()
	b.data = append(b.data, 1, 2, 3)
	p.Put(b)

	assert.Equal(t, 0, len(b.data), "Put must call Reset")
}
