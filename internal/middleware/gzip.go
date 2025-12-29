package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w *gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func WithGzip(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = gz
		}

		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			h.ServeHTTP(w, r)
			return
		}

		crw := &compressResponseWriter{
			ResponseWriter: w,
		}
		defer crw.Close()

		h.ServeHTTP(crw, r)
	})
}

type compressResponseWriter struct {
	http.ResponseWriter
	gzWriter      *gzip.Writer
	headerWritten bool
}

func (c *compressResponseWriter) WriteHeader(statusCode int) {
	if c.headerWritten {
		return
	}
	c.headerWritten = true

	contentType := c.Header().Get("Content-Type")
	shouldCompress := strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "text/html")

	if shouldCompress {
		c.gzWriter = gzip.NewWriter(c.ResponseWriter)
		c.Header().Set("Content-Encoding", "gzip")
		c.Header().Del("Content-Length")
	}

	c.ResponseWriter.WriteHeader(statusCode)
}

func (c *compressResponseWriter) Write(b []byte) (int, error) {
	if !c.headerWritten {
		c.WriteHeader(http.StatusOK)
	}

	if c.gzWriter != nil {
		return c.gzWriter.Write(b)
	}
	return c.ResponseWriter.Write(b)
}

func (c *compressResponseWriter) Close() error {
	if c.gzWriter != nil {
		return c.gzWriter.Close()
	}
	return nil
}
