package shortenerpb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestProtoMessages_RoundTrip(t *testing.T) {
	for _, msg := range []proto.Message{
		URLShortenRequest_builder{Url: "https://example.com"}.Build(),
		URLShortenResponse_builder{Result: "https://short.example/abc"}.Build(),
		URLExpandRequest_builder{Id: "abc"}.Build(),
		URLExpandResponse_builder{Result: "https://example.com"}.Build(),
		URLData_builder{ShortUrl: "s", OriginalUrl: "o"}.Build(),
		UserURLsResponse_builder{Url: []*URLData{URLData_builder{ShortUrl: "s", OriginalUrl: "o"}.Build()}}.Build(),
		ListUserURLsRequest_builder{}.Build(),
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
	a := URLShortenRequest_builder{Url: "u"}.Build()
	assert.Equal(t, "u", a.GetUrl())
	assert.NotEmpty(t, a.String())

	b := URLShortenResponse_builder{Result: "r"}.Build()
	assert.Equal(t, "r", b.GetResult())
	assert.NotEmpty(t, b.String())

	c := URLExpandRequest_builder{Id: "i"}.Build()
	assert.Equal(t, "i", c.GetId())
	assert.NotEmpty(t, c.String())

	d := URLExpandResponse_builder{Result: "r"}.Build()
	assert.Equal(t, "r", d.GetResult())
	assert.NotEmpty(t, d.String())

	e := URLData_builder{ShortUrl: "s", OriginalUrl: "o"}.Build()
	assert.Equal(t, "s", e.GetShortUrl())
	assert.Equal(t, "o", e.GetOriginalUrl())
	assert.NotEmpty(t, e.String())

	f := UserURLsResponse_builder{Url: []*URLData{e}}.Build()
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

func TestProtoMessages_Setters(t *testing.T) {
	a := &URLShortenRequest{}
	a.SetUrl("u")
	assert.Equal(t, "u", a.GetUrl())

	b := &URLShortenResponse{}
	b.SetResult("r")
	assert.Equal(t, "r", b.GetResult())

	c := &URLExpandRequest{}
	c.SetId("i")
	assert.Equal(t, "i", c.GetId())

	d := &URLExpandResponse{}
	d.SetResult("r")
	assert.Equal(t, "r", d.GetResult())

	e := &URLData{}
	e.SetShortUrl("s")
	e.SetOriginalUrl("o")
	assert.Equal(t, "s", e.GetShortUrl())
	assert.Equal(t, "o", e.GetOriginalUrl())
}

func TestProtoMessages_Reset(t *testing.T) {
	a := URLShortenRequest_builder{Url: "u"}.Build()
	a.Reset()
	assert.Equal(t, "", a.GetUrl())

	b := URLShortenResponse_builder{Result: "r"}.Build()
	b.Reset()
	assert.Equal(t, "", b.GetResult())

	c := URLExpandRequest_builder{Id: "i"}.Build()
	c.Reset()
	assert.Equal(t, "", c.GetId())

	d := URLExpandResponse_builder{Result: "r"}.Build()
	d.Reset()
	assert.Equal(t, "", d.GetResult())

	e := URLData_builder{ShortUrl: "s"}.Build()
	e.Reset()
	assert.Equal(t, "", e.GetShortUrl())

	f := UserURLsResponse_builder{Url: []*URLData{URLData_builder{}.Build()}}.Build()
	f.Reset()
	assert.Nil(t, f.GetUrl())
}
