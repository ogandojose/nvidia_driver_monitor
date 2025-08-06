package web

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	// Create a simple handler for testing
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap with security middleware
	secureHandler := SecurityHeadersMiddleware(testHandler)

	// Test HTTP request (no HSTS)
	t.Run("HTTP Request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/", nil)
		w := httptest.NewRecorder()

		secureHandler.ServeHTTP(w, req)

		// Check basic security headers
		assertHeader(t, w, "X-Content-Type-Options", "nosniff")
		assertHeader(t, w, "X-Frame-Options", "DENY")
		assertHeader(t, w, "X-XSS-Protection", "1; mode=block")
		assertHeader(t, w, "Referrer-Policy", "strict-origin-when-cross-origin")
		assertHeader(t, w, "Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=()")

		// CSP should be present
		csp := w.Header().Get("Content-Security-Policy")
		if csp == "" {
			t.Error("Content-Security-Policy header is missing")
		}

		// HSTS should NOT be present for HTTP
		if hsts := w.Header().Get("Strict-Transport-Security"); hsts != "" {
			t.Errorf("Strict-Transport-Security should not be set for HTTP requests, got: %s", hsts)
		}
	})

	// Test HTTPS request (with HSTS)
	t.Run("HTTPS Request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "https://example.com/", nil)
		req.TLS = &tls.ConnectionState{} // Simulate TLS connection
		w := httptest.NewRecorder()

		secureHandler.ServeHTTP(w, req)

		// Check all security headers including HSTS
		assertHeader(t, w, "X-Content-Type-Options", "nosniff")
		assertHeader(t, w, "X-Frame-Options", "DENY")
		assertHeader(t, w, "X-XSS-Protection", "1; mode=block")
		assertHeader(t, w, "Referrer-Policy", "strict-origin-when-cross-origin")
		assertHeader(t, w, "Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// CSP should be present
		csp := w.Header().Get("Content-Security-Policy")
		if csp == "" {
			t.Error("Content-Security-Policy header is missing")
		}
	})
}

func assertHeader(t *testing.T, w *httptest.ResponseRecorder, header, expected string) {
	t.Helper()
	actual := w.Header().Get(header)
	if actual != expected {
		t.Errorf("Header %s: expected %q, got %q", header, expected, actual)
	}
}
