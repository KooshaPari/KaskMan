package api

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a custom response writer to capture status code
		lw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		// Add request ID to context
		requestID := generateRequestID()
		ctx := context.WithValue(r.Context(), "request_id", requestID)
		r = r.WithContext(ctx)
		
		// Add request ID to response headers
		w.Header().Set("X-Request-ID", requestID)
		
		// Call the next handler
		next.ServeHTTP(lw, r)
		
		// Log the request
		duration := time.Since(start)
		if s.config.Server.Development {
			log.Printf("[%s] %s %s %d %v (Request ID: %s)",
				r.Method, r.RequestURI, r.RemoteAddr, lw.statusCode, duration, requestID)
		}
	})
}

// rateLimitMiddleware implements rate limiting
func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.limiter != nil && !s.limiter.Allow() {
			s.errorResponse(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED",
				"Rate limit exceeded", "Too many requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// recoveryMiddleware recovers from panics
func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v\n%s", err, debug.Stack())
				s.errorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR",
					"Internal server error", "An unexpected error occurred")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// contentTypeMiddleware sets content type for API responses
func (s *Server) contentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set common headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		
		// Set API version header
		w.Header().Set("X-API-Version", "v1")
		
		next.ServeHTTP(w, r)
	})
}

// loggingResponseWriter wraps http.ResponseWriter to capture status code
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lw *loggingResponseWriter) WriteHeader(code int) {
	lw.statusCode = code
	lw.ResponseWriter.WriteHeader(code)
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	// Simple request ID generation - in production you'd want something more robust
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}