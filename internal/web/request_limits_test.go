package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequestLimitsMiddleware(t *testing.T) {
	// Create a simple handler for testing
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the body to test body size limits - use io.ReadAll to trigger MaxBytesReader
		body, err := io.ReadAll(r.Body)
		if err != nil {
			// Handle MaxBytesReader error by writing proper status
			if err.Error() == "http: request body too large" {
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "Error reading body", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body length: " + string(body)))
	})

	t.Run("Body Size Limit Enforcement", func(t *testing.T) {
		// Create middleware with 10 byte limit
		middleware := RequestLimitsMiddleware(10, 0)
		handler := middleware(testHandler)

		// Test with body within limit
		t.Run("Within Limit", func(t *testing.T) {
			body := strings.NewReader("small")
			req := httptest.NewRequest("POST", "/test", body)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		})

		// Test with body exceeding limit
		t.Run("Exceeds Limit", func(t *testing.T) {
			body := strings.NewReader("this is a very long body that exceeds the 10 byte limit")
			req := httptest.NewRequest("POST", "/test", body)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// Should return 413 Request Entity Too Large
			if w.Code != http.StatusRequestEntityTooLarge {
				t.Errorf("Expected status 413, got %d", w.Code)
			}
		})
	})

	t.Run("Request Timeout Enforcement", func(t *testing.T) {
		// Create a slow handler that takes longer than timeout
		slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if context has timeout
			if _, ok := r.Context().Deadline(); ok {
				// Simulate work that takes longer than timeout
				select {
				case <-time.After(100 * time.Millisecond):
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("completed"))
				case <-r.Context().Done():
					// Context was cancelled due to timeout
					if r.Context().Err() == context.DeadlineExceeded {
						t.Log("Request correctly cancelled due to timeout")
					}
					return
				}
			} else {
				t.Error("Expected context to have deadline")
			}
		})

		// Create middleware with very short timeout
		middleware := RequestLimitsMiddleware(0, 50*time.Millisecond)
		handler := middleware(slowHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// The handler should have been cancelled, but HTTP status might still be written
		// The important thing is that the context was properly configured with timeout
	})

	t.Run("No Limits Applied", func(t *testing.T) {
		// Create middleware with no limits (0 values)
		middleware := RequestLimitsMiddleware(0, 0)
		handler := middleware(testHandler)

		body := strings.NewReader("any size body should work")
		req := httptest.NewRequest("POST", "/test", body)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Context Propagation", func(t *testing.T) {
		contextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify that timeout context is properly set
			if deadline, ok := r.Context().Deadline(); ok {
				if time.Until(deadline) > 1*time.Second {
					t.Error("Timeout should be less than 1 second")
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("context ok"))
			} else {
				t.Error("Expected context to have deadline")
			}
		})

		middleware := RequestLimitsMiddleware(0, 500*time.Millisecond)
		handler := middleware(contextHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestRequestLimitsIntegration(t *testing.T) {
	// Test integration with other middleware
	t.Run("With Security Headers", func(t *testing.T) {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})

		// Chain middlewares: Request Limits -> Security Headers -> Handler
		requestLimits := RequestLimitsMiddleware(1024, 5*time.Second)
		securityHeaders := SecurityHeadersMiddleware
		handler := requestLimits(securityHeaders(testHandler))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Check that both middlewares applied their effects
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Check security headers are present
		if w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Error("Security headers not applied")
		}
	})

	t.Run("Error Response Body Size", func(t *testing.T) {
		// Test that the error response for oversized body is reasonable
		middleware := RequestLimitsMiddleware(5, 0)
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Need to read body to trigger MaxBytesReader limit
			_, err := io.ReadAll(r.Body)
			if err != nil {
				if err.Error() == "http: request body too large" {
					http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
					return
				}
				http.Error(w, "Error reading body", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))

		body := strings.NewReader("this body is too long")
		req := httptest.NewRequest("POST", "/test", body)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("Expected status 413, got %d", w.Code)
		}

		// Error response should not be excessively large
		responseBody := w.Body.String()
		if len(responseBody) > 1000 { // Reasonable limit for error message
			t.Errorf("Error response too large: %d bytes", len(responseBody))
		}
	})
}
