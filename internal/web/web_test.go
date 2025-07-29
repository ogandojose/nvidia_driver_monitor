package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiter(t *testing.T) {
	rateLimiter := NewRateLimiter(2, true) // 2 requests per minute

	handler := rateLimiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// First request should succeed
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "127.0.0.1:12345"
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w1.Code)
	}

	// Second request should succeed
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "127.0.0.1:12345"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w2.Code)
	}

	// Third request should be rate limited
	req3 := httptest.NewRequest("GET", "/", nil)
	req3.RemoteAddr = "127.0.0.1:12345"
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)

	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w3.Code)
	}
}

func TestRateLimiterDisabled(t *testing.T) {
	rateLimiter := NewRateLimiter(1, false) // Disabled

	handler := rateLimiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// Multiple requests should all succeed when rate limiting is disabled
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
		}
	}
}

func TestAPIHandler(t *testing.T) {
	apiHandler := NewAPIHandler()

	// Test health endpoint
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	apiHandler.HealthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name           string
		headers        map[string]string
		remoteAddr     string
		expectedIP     string
	}{
		{
			name:       "X-Forwarded-For header",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.1"},
			remoteAddr: "127.0.0.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "X-Real-IP header",
			headers:    map[string]string{"X-Real-IP": "10.0.0.1"},
			remoteAddr: "127.0.0.1:12345",
			expectedIP: "10.0.0.1",
		},
		{
			name:       "Remote address fallback",
			headers:    map[string]string{},
			remoteAddr: "127.0.0.1:12345",
			expectedIP: "127.0.0.1:12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			ip := getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("getClientIP() = %s, expected %s", ip, tt.expectedIP)
			}
		})
	}
}

func TestTemplateFunctions(t *testing.T) {
	funcs := TemplateFunctions()

	// Test eq function
	if eqFunc, ok := funcs["eq"]; ok {
		if eq, ok := eqFunc.(func(string, string) bool); ok {
			if !eq("test", "test") {
				t.Error("eq function should return true for equal strings")
			}
			if eq("test", "other") {
				t.Error("eq function should return false for different strings")
			}
		} else {
			t.Error("eq function has wrong type")
		}
	} else {
		t.Error("eq function not found")
	}

	// Test contains function
	if containsFunc, ok := funcs["contains"]; ok {
		if contains, ok := containsFunc.(func(string, string) bool); ok {
			if !contains("hello world", "world") {
				t.Error("contains function should return true when substring exists")
			}
			if contains("hello", "world") {
				t.Error("contains function should return false when substring doesn't exist")
			}
		} else {
			t.Error("contains function has wrong type")
		}
	} else {
		t.Error("contains function not found")
	}
}
