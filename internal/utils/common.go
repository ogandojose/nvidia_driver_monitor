package utils

import (
	"fmt"
	"strings"
)

// ExtractSeriesFromLink extracts series name from a Launchpad distro series link
func ExtractSeriesFromLink(link string) string {
	parts := strings.Split(strings.TrimRight(link, "/"), "/")
	if len(parts) < 1 {
		return ""
	}
	return parts[len(parts)-1]
}

// ExtractSeriesAndArchFromLink extracts series and architecture from a Launchpad distro arch series link
func ExtractSeriesAndArchFromLink(link string) (string, string) {
	parts := strings.Split(strings.TrimRight(link, "/"), "/")
	if len(parts) < 2 {
		return "", ""
	}
	return parts[len(parts)-2], parts[len(parts)-1]
}

// FormatSupportedMap formats a map of supported releases as a string
func FormatSupportedMap(supported map[string]bool) string {
	var parts []string
	for k, v := range supported {
		parts = append(parts, fmt.Sprintf("%s:%t", k, v))
	}
	return strings.Join(parts, " ")
}

// IsValidVersion checks if a version string is valid
func IsValidVersion(version string) bool {
	return version != "" && len(version) > 0
}
