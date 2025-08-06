package web

import (
	"context"
	"net/http"
	"time"
)

// RequestLimitsMiddleware enforces request body size limits and timeouts
func RequestLimitsMiddleware(maxBodySize int64, requestTimeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Limit request body size to prevent large request DoS attacks
			if maxBodySize > 0 {
				r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
			}

			// Set request timeout to prevent slow request attacks
			if requestTimeout > 0 {
				ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
				defer cancel()
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}
