package web

import (
	"html/template"
	"strings"

	"nvidia_driver_monitor/internal/lrm"
)

// TemplateFunctions returns a map of custom template functions
func TemplateFunctions() template.FuncMap {
	return template.FuncMap{
		"eq": func(a, b string) bool {
			return a == b
		},
		"contains": func(s, substr string) bool {
			return strings.Contains(s, substr)
		},
		"simplifyDriver": func(driver string) string {
			return lrm.SimplifyNvidiaDriverName(driver)
		},
		"simplifyDriverName": func(driverName string) string {
			// Extract the driver branch (e.g., "535", "470-server") from the full name
			prefix := "nvidia-graphics-drivers-"
			if strings.HasPrefix(driverName, prefix) {
				return driverName[len(prefix):]
			}
			return driverName
		},
	}
}
