package shortenerpb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestProtoMessages_RoundTrip(t *testing.T) {
	for _, msg := range []proto.Message{
		&URLShortenRequest{Url: "https://example.com"},
		&URLShortenResponse{Result: "https://short.example/abc"},
		&URLExpandRequest{Id: "abc"},
		&URLExpandResponse{Result: "https://example.com"},
		&URLData{ShortUrl: "s", OriginalUrl: "o"},
		&UserURLsResponse{Url: []*URLData{{ShortUrl: "s", OriginalUrl: "o"}}},
	} {
		data, err := proto.Marshal(msg)
		assert.NoError(t, err)

		clone := proto.Clone(msg)
		proto.Reset(clone)
		assert.NoError(t, proto.Unmarshal(data, clone))
		assert.True(t, proto.Equal(msg, clone))
	}
}

func TestProtoMessages_Getters(t *testing.T) {
	a := &URLShortenRequest{Url: "u"}
	assert.Equal(t, "u", a.GetUrl())
	assert.NotEmpty(t, a.String())

	b := &URLShortenResponse{Result: "r"}
	assert.Equal(t, "r", b.GetResult())
	assert.NotEmpty(t, b.String())

	c := &URLExpandRequest{Id: "i"}
	assert.Equal(t, "i", c.GetId())
	assert.NotEmpty(t, c.String())

	d := &URLExpandResponse{Result: "r"}
	assert.Equal(t, "r", d.GetResult())
	assert.NotEmpty(t, d.String())

	e := &URLData{ShortUrl: "s", OriginalUrl: "o"}
	assert.Equal(t, "s", e.GetShortUrl())
	assert.Equal(t, "o", e.GetOriginalUrl())
	assert.NotEmpty(t, e.String())

	f := &UserURLsResponse{Url: []*URLData{e}}
	assert.Len(t, f.GetUrl(), 1)
	assert.NotEmpty(t, f.String())
}

func TestProtoMessages_NilGetters(t *testing.T) {
	var (
		a *URLShortenRequest
		b *URLShortenResponse
		c *URLExpandRequest
		d *URLExpandResponse
		e *URLData
		f *UserURLsResponse
	)
	assert.Equal(t, "", a.GetUrl())
	assert.Equal(t, "", b.GetResult())
	assert.Equal(t, "", c.GetId())
	assert.Equal(t, "", d.GetResult())
	assert.Equal(t, "", e.GetShortUrl())
	assert.Equal(t, "", e.GetOriginalUrl())
	assert.Nil(t, f.GetUrl())
}

func TestProtoMessages_Reset(t *testing.T) {
	a := &URLShortenRequest{Url: "u"}
	a.Reset()
	assert.Equal(t, "", a.GetUrl())

	b := &URLShortenResponse{Result: "r"}
	b.Reset()
	assert.Equal(t, "", b.GetResult())

	c := &URLExpandRequest{Id: "i"}
	c.Reset()
	assert.Equal(t, "", c.GetId())

	d := &URLExpandResponse{Result: "r"}
	d.Reset()
	assert.Equal(t, "", d.GetResult())

	e := &URLData{ShortUrl: "s"}
	e.Reset()
	assert.Equal(t, "", e.GetShortUrl())

	f := &UserURLsResponse{Url: []*URLData{{}}}
	f.Reset()
	assert.Nil(t, f.GetUrl())
}
