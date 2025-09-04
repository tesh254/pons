package core

import (
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	bodySize   int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(data)
	rw.bodySize += n
	return n, err
}

func loggingHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		log.Printf("[INFO] %s | RequestID: %s | Incoming Request: %s %s | From: %s | User-Agent: %s | Content-Length: %s",
			start.Format(time.RFC3339),
			requestID,
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
			r.Header.Get("User-Agent"),
			r.Header.Get("Content-Length"),
		)

		handler.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		log.Printf("[INFO] %s | RequestID: %s | Response Sent: %s %s | Status: %d | Duration: %v | Response Size: %d bytes",
			time.Now().Format(time.RFC3339),
			requestID,
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration,
			wrapped.bodySize,
		)
	})
}
