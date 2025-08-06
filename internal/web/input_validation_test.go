package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestInputValidator_ValidateQueryParams(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name        string
		queryParams map[string]string
		expected    map[string]string
	}{
		{
			name: "Valid series parameter",
			queryParams: map[string]string{
				"series": "focal",
			},
			expected: map[string]string{
				"series": "focal",
			},
		},
		{
			name: "Invalid series parameter",
			queryParams: map[string]string{
				"series": "invalid-series-123",
			},
			expected: map[string]string{},
		},
		{
			name: "Valid status parameter",
			queryParams: map[string]string{
				"status": "supported",
			},
			expected: map[string]string{
				"status": "supported",
			},
		},
		{
			name: "Valid routing parameter",
			queryParams: map[string]string{
				"routing": "ubuntu/4",
			},
			expected: map[string]string{
				"routing": "ubuntu/4",
			},
		},
		{
			name: "Valid limit parameter",
			queryParams: map[string]string{
				"limit": "50",
			},
			expected: map[string]string{
				"limit": "50",
			},
		},
		{
			name: "Limit parameter too large",
			queryParams: map[string]string{
				"limit": "5000",
			},
			expected: map[string]string{
				"limit": "1000", // Clamped to maximum
			},
		},
		{
			name: "Valid offset parameter",
			queryParams: map[string]string{
				"offset": "100",
			},
			expected: map[string]string{
				"offset": "100",
			},
		},
		{
			name: "Invalid limit parameter",
			queryParams: map[string]string{
				"limit": "not-a-number",
			},
			expected: map[string]string{},
		},
		{
			name: "Valid package name",
			queryParams: map[string]string{
				"package": "nvidia-graphics-drivers-535",
			},
			expected: map[string]string{
				"package": "nvidia-graphics-drivers-535",
			},
		},
		{
			name: "Invalid package name with special chars",
			queryParams: map[string]string{
				"package": "nvidia-drivers@#$%",
			},
			expected: map[string]string{},
		},
		{
			name: "Multiple valid parameters",
			queryParams: map[string]string{
				"series":  "jammy",
				"status":  "supported",
				"routing": "ubuntu/4",
				"limit":   "25",
				"offset":  "0",
			},
			expected: map[string]string{
				"series":  "jammy",
				"status":  "supported",
				"routing": "ubuntu/4",
				"limit":   "25",
				"offset":  "0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with query parameters
			req := httptest.NewRequest("GET", "/test", nil)
			q := req.URL.Query()
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()

			result, err := validator.ValidateQueryParams(req)
			if err != nil {
				t.Errorf("ValidateQueryParams() error = %v", err)
				return
			}

			// Check if result matches expected
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d parameters, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, ok := result[key]; !ok {
					t.Errorf("Expected parameter %s not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("Parameter %s: expected %s, got %s", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestInputValidator_validateSeries(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		input    string
		expected string
	}{
		{"focal", "focal"},
		{"FOCAL", "focal"},   // Case normalization
		{" jammy ", "jammy"}, // Whitespace trimming
		{"noble", "noble"},
		{"invalid-series", ""},
		{"", ""},
		{"toolong series name", ""},
		{"xyz", ""},                // Too short but valid pattern
		{"validname", "validname"}, // Future series pattern
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := validator.validateSeries(tt.input)
			if result != tt.expected {
				t.Errorf("validateSeries(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestInputValidator_validatePackageName(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		input    string
		expected string
	}{
		{"nvidia-graphics-drivers-535", "nvidia-graphics-drivers-535"},
		{"linux-image-generic", "linux-image-generic"},
		{"package.name", "package.name"},
		{"package+name", "package+name"},
		{"1package", "1package"}, // Starting with digit is valid
		{"Package", ""},          // Uppercase not allowed
		{"package@name", ""},     // @ not allowed
		{"package name", ""},     // Space not allowed
		{"", ""},
		{"a", ""},                       // Too short
		{string(make([]byte, 300)), ""}, // Too long
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := validator.validatePackageName(tt.input)
			if result != tt.expected {
				t.Errorf("validatePackageName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestInputValidator_validatePositiveInt(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		input    string
		min      int
		max      int
		expected int
	}{
		{"50", 1, 100, 50},
		{"0", 0, 100, 0},
		{"150", 1, 100, 100}, // Clamped to max
		{"-5", 1, 100, 1},    // Clamped to min
		{"abc", 1, 100, -1},  // Invalid input
		{"", 1, 100, -1},     // Empty input
		{" 25 ", 1, 100, 25}, // Whitespace trimming
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := validator.validatePositiveInt(tt.input, tt.min, tt.max)
			if result != tt.expected {
				t.Errorf("validatePositiveInt(%q, %d, %d) = %d, expected %d",
					tt.input, tt.min, tt.max, result, tt.expected)
			}
		})
	}
}

func TestInputValidator_SanitizeHTML(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"<script>alert('xss')</script>", "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"},
		{"text with & ampersand", "text with &amp; ampersand"},
		{`"quoted text"`, "&quot;quoted text&quot;"},
		{"text with 'single quotes'", "text with &#39;single quotes&#39;"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := validator.SanitizeHTML(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeHTML(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestInputSanitizationMiddleware(t *testing.T) {
	// Create test handler that checks for validated parameters
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if validated parameters are in context
		if series := r.Context().Value("validated_series"); series != nil {
			w.Header().Set("X-Validated-Series", series.(string))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := InputSanitizationMiddleware()
	handler := middleware(testHandler)

	tests := []struct {
		name           string
		queryParams    map[string]string
		expectedStatus int
		expectedHeader string
	}{
		{
			name: "Valid parameters",
			queryParams: map[string]string{
				"series": "focal",
			},
			expectedStatus: http.StatusOK,
			expectedHeader: "focal",
		},
		{
			name:           "No parameters",
			queryParams:    map[string]string{},
			expectedStatus: http.StatusOK,
			expectedHeader: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with query parameters
			u := &url.URL{Path: "/test"}
			q := u.Query()
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}
			u.RawQuery = q.Encode()

			req := httptest.NewRequest("GET", u.String(), nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if actualHeader := w.Header().Get("X-Validated-Series"); actualHeader != tt.expectedHeader {
				t.Errorf("Expected header %q, got %q", tt.expectedHeader, actualHeader)
			}
		})
	}
}
