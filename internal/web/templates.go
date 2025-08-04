package web

import (
	"html/template"
	"strings"

	"nvidia_driver_monitor/internal/config"
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

// GetCDNResources returns a map of CDN resources for templates
func GetCDNResources(cfg *config.Config) map[string]string {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return map[string]string{
		"BootstrapCSS": cfg.URLs.CDN.BootstrapCSS,
		"BootstrapJS":  cfg.URLs.CDN.BootstrapJS,
		"ChartJS":      cfg.URLs.CDN.ChartJS,
		"VanillaCSS":   cfg.URLs.CDN.VanillaCSS,
		"UbuntuAssets": cfg.URLs.Ubuntu.AssetsBaseURL,
	}
}

// TemplateData holds data passed to templates including configuration
type TemplateData struct {
	Config interface{} `json:"config,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

// NewTemplateData creates a new template data structure with config and data
func NewTemplateData(cfg *config.Config, data interface{}) *TemplateData {
	return &TemplateData{
		Config: cfg,
		Data:   data,
	}
}
