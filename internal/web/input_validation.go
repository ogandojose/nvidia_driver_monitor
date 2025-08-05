package web

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// InputValidator provides validation for user inputs
type InputValidator struct {
	// Allowed values for specific parameters
	allowedSeries   map[string]bool
	allowedStatuses map[string]bool
	allowedRoutings map[string]bool
}

// NewInputValidator creates a new input validator with allowed values
func NewInputValidator() *InputValidator {
	return &InputValidator{
		allowedSeries: map[string]bool{
			"focal":   true,
			"jammy":   true,
			"noble":   true,
			"kinetic": true,
			"lunar":   true,
			"mantic":  true,
		},
		allowedStatuses: map[string]bool{
			"supported":   true,
			"unsupported": true,
			"lts":         true,
			"esm":         true,
			"development": true,
		},
		allowedRoutings: map[string]bool{
			"ubuntu/4":     true,
			"ubuntu/2":     true,
			"signed/4":     true,
			"signed/2":     true,
			"pro/3":        true,
			"pro/2":        true,
			"fips-pro/3":   true,
			"fips-pro/2":   true,
			"realtime-pro/3": true,
		},
	}
}

// ValidateQueryParams validates and sanitizes query parameters
func (v *InputValidator) ValidateQueryParams(r *http.Request) (map[string]string, error) {
	params := make(map[string]string)
	
	// Validate series parameter
	if series := r.URL.Query().Get("series"); series != "" {
		if sanitized := v.validateSeries(series); sanitized != "" {
			params["series"] = sanitized
		}
	}
	
	// Validate status parameter
	if status := r.URL.Query().Get("status"); status != "" {
		if sanitized := v.validateStatus(status); sanitized != "" {
			params["status"] = sanitized
		}
	}
	
	// Validate routing parameter
	if routing := r.URL.Query().Get("routing"); routing != "" {
		if sanitized := v.validateRouting(routing); sanitized != "" {
			params["routing"] = sanitized
		}
	}
	
	// Validate numeric parameters
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if sanitized := v.validatePositiveInt(limit, 1, 1000); sanitized > 0 {
			params["limit"] = strconv.Itoa(sanitized)
		}
	}
	
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if sanitized := v.validatePositiveInt(offset, 0, 10000); sanitized >= 0 {
			params["offset"] = strconv.Itoa(sanitized)
		}
	}
	
	// Validate package name parameter
	if pkg := r.URL.Query().Get("package"); pkg != "" {
		if sanitized := v.validatePackageName(pkg); sanitized != "" {
			params["package"] = sanitized
		}
	}
	
	if name := r.URL.Query().Get("name"); name != "" {
		if sanitized := v.validatePackageName(name); sanitized != "" {
			params["name"] = sanitized
		}
	}
	
	return params, nil
}

// validateSeries validates Ubuntu series names
func (v *InputValidator) validateSeries(series string) string {
	// Normalize input
	series = strings.ToLower(strings.TrimSpace(series))
	
	// Check against allowed list
	if v.allowedSeries[series] {
		return series
	}
	
	// Also validate with regex for future series
	matched, _ := regexp.MatchString(`^[a-z]{4,10}$`, series)
	if matched {
		return series
	}
	
	return ""
}

// validateStatus validates status filter values
func (v *InputValidator) validateStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	if v.allowedStatuses[status] {
		return status
	}
	return ""
}

// validateRouting validates routing parameter values
func (v *InputValidator) validateRouting(routing string) string {
	routing = strings.TrimSpace(routing)
	if v.allowedRoutings[routing] {
		return routing
	}
	
	// Validate routing pattern: word/number
	matched, _ := regexp.MatchString(`^[a-z-]+/[0-9]+$`, routing)
	if matched {
		return routing
	}
	
	return ""
}

// validatePositiveInt validates and bounds integer parameters
func (v *InputValidator) validatePositiveInt(value string, min, max int) int {
	value = strings.TrimSpace(value)
	
	num, err := strconv.Atoi(value)
	if err != nil {
		return -1
	}
	
	if num < min {
		return min
	}
	if num > max {
		return max
	}
	
	return num
}

// validatePackageName validates package names for Ubuntu packages
func (v *InputValidator) validatePackageName(name string) string {
	name = strings.TrimSpace(name)
	
	// Ubuntu package names: lowercase letters, digits, hyphens, dots, plus signs
	// Must start with alphanumeric, length 2-214 chars
	matched, _ := regexp.MatchString(`^[a-z0-9][a-z0-9+.-]{1,213}$`, name)
	if matched {
		return name
	}
	
	return ""
}

// ValidateURLPath validates URL path components
func (v *InputValidator) ValidateURLPath(path string) string {
	// Remove leading/trailing slashes and normalize
	path = strings.Trim(path, "/")
	
	// Basic path validation - alphanumeric, hyphens, underscores, dots
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9._-]+$`, path)
	if matched && len(path) <= 255 {
		return path
	}
	
	return ""
}

// SanitizeHTML removes or escapes HTML content from user input
func (v *InputValidator) SanitizeHTML(input string) string {
	// Basic HTML escaping
	input = strings.ReplaceAll(input, "&", "&amp;")
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#39;")
	
	return input
}

// InputSanitizationMiddleware provides input validation middleware
func InputSanitizationMiddleware() func(http.Handler) http.Handler {
	validator := NewInputValidator()
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Validate query parameters
			validParams, err := validator.ValidateQueryParams(r)
			if err != nil {
				http.Error(w, "Invalid query parameters", http.StatusBadRequest)
				return
			}
			
			// Store validated parameters in request context for handlers to use
			// This prevents handlers from using raw, unvalidated input
			ctx := r.Context()
			for key, value := range validParams {
				ctx = context.WithValue(ctx, "validated_"+key, value)
			}
			r = r.WithContext(ctx)
			
			next.ServeHTTP(w, r)
		})
	}
}

// Helper functions for handlers to retrieve validated parameters from context

// GetValidatedString retrieves a validated string parameter from request context
func GetValidatedString(r *http.Request, param string) string {
	if value := r.Context().Value("validated_" + param); value != nil {
		return value.(string)
	}
	return ""
}

// GetValidatedInt retrieves a validated integer parameter from request context
func GetValidatedInt(r *http.Request, param string) int {
	if value := r.Context().Value("validated_" + param); value != nil {
		if str, ok := value.(string); ok {
			if num, err := strconv.Atoi(str); err == nil {
				return num
			}
		}
	}
	return 0
}

// LogSuspiciousInput logs potentially malicious input attempts
func LogSuspiciousInput(r *http.Request, param, value, reason string) {
	clientIP := getClientIP(r) // Use existing function from ratelimit.go
	log.Printf("SECURITY WARNING: Suspicious input from %s - param:%s value:%q reason:%s", 
		clientIP, param, value, reason)
}
