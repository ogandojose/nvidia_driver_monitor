package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"nvidia_driver_monitor/internal/services"
	"nvidia_driver_monitor/internal/web"
)

// PackageHandler handles package-related web requests
type PackageHandler struct {
	webService   *services.WebService
	templatePath string
}

// SeriesData represents the data for a single series row
type SeriesData struct {
	Series          string
	UpdatesSecurity string
	Proposed        string
	UpstreamVersion string
	ReleaseDate     string
	SRUCycle        string
	UpdatesColor    string
	ProposedColor   string
}

// PackageData represents the data for a complete package table
type PackageData struct {
	PackageName string
	Series      []SeriesData
}

// CachedData holds all cached package information
type CachedData struct {
	AllPackages   []*PackageData
	LastUpdated   time.Time
	IsInitialized bool
}

// NewPackageHandler creates a new package handler
func NewPackageHandler(webService *services.WebService, templatePath string) *PackageHandler {
	return &PackageHandler{
		webService:   webService,
		templatePath: templatePath,
	}
}

// IndexHandler handles the main page request
func (h *PackageHandler) IndexHandler(w http.ResponseWriter, r *http.Request, cache *CachedData) {
	if !cache.IsInitialized {
		http.Error(w, "Service is still initializing, please try again in a moment", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Read the index template
	templatePath := filepath.Join(h.templatePath, "index.html")
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading index template: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse the template
	tmpl, err := template.New("index").Parse(string(templateContent))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing index template: %v", err), http.StatusInternalServerError)
		return
	}

	// Create template data
	templateData := struct {
		Packages    []*PackageData `json:"packages"`
		LastUpdated time.Time      `json:"last_updated"`
	}{
		Packages:    cache.AllPackages,
		LastUpdated: cache.LastUpdated,
	}

	// Execute the template
	if err := tmpl.Execute(w, templateData); err != nil {
		http.Error(w, fmt.Sprintf("Error executing template: %v", err), http.StatusInternalServerError)
		return
	}
}

// PackageHandler handles package detail requests
func (h *PackageHandler) PackageHandler(w http.ResponseWriter, r *http.Request, cache *CachedData) {
	packageName := r.URL.Query().Get("name")
	if packageName == "" {
		http.Error(w, "Package name is required", http.StatusBadRequest)
		return
	}

	if !cache.IsInitialized {
		http.Error(w, "Service is still initializing, please try again in a moment", http.StatusServiceUnavailable)
		return
	}

	// Find package data
	var packageData *PackageData
	for _, pkg := range cache.AllPackages {
		if pkg.PackageName == packageName {
			packageData = pkg
			break
		}
	}

	if packageData == nil {
		http.Error(w, "Package not found", http.StatusNotFound)
		return
	}

	packageTemplate := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.PackageName}} - NVIDIA Driver Monitor</title>
    <link href="{{.CDN.BootstrapCSS}}" rel="stylesheet">
</head>
<body>
    <div class="container mt-4">
        <h1>{{.PackageName}}</h1>
        
        <div class="table-responsive">
            <table class="table table-striped table-hover">
                <thead class="table-dark">
                    <tr>
                        <th>Series</th>
                        <th>Updates/Security</th>
                        <th>Proposed</th>
                        <th>Upstream Version</th>
                        <th>Release Date</th>
                        <th>SRU Cycle</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Series}}
                    <tr>
                        <td><strong>{{.Series}}</strong></td>
                        <td>
                            <span class="badge" style="background-color: {{.UpdatesColor}}">{{.UpdatesSecurity}}</span>
                        </td>
                        <td>
                            <span class="badge" style="background-color: {{.ProposedColor}}">{{.Proposed}}</span>
                        </td>
                        <td>{{.UpstreamVersion}}</td>
                        <td>{{.ReleaseDate}}</td>
                        <td>{{.SRUCycle}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        
        <div class="mt-4">
            <a href="/" class="btn btn-secondary">← Back to Overview</a>
            <a href="/api?package={{.PackageName}}" class="btn btn-outline-primary">View JSON Data</a>
        </div>
    </div>

    <script src="{{.CDN.BootstrapJS}}"></script>
</body>
</html>`

	tmpl, err := template.New("package").Parse(packageTemplate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing template: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	// Create template data with CDN resources
	templateData := struct {
		*PackageData
		CDN map[string]string
	}{
		PackageData: packageData,
		CDN:         web.GetCDNResources(h.webService.GetConfig()),
	}

	if err := tmpl.Execute(w, templateData); err != nil {
		http.Error(w, fmt.Sprintf("Error executing template: %v", err), http.StatusInternalServerError)
		return
	}
}

// APIHandler handles JSON API requests for packages
func (h *PackageHandler) APIHandler(w http.ResponseWriter, r *http.Request, cache *CachedData) {
	packageName := r.URL.Query().Get("package")

	if !cache.IsInitialized {
		http.Error(w, "Service is still initializing, please try again in a moment", http.StatusServiceUnavailable)
		return
	}

	if packageName != "" {
		// Return data for specific package
		for _, pkg := range cache.AllPackages {
			if pkg.PackageName == packageName {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(pkg)
				return
			}
		}
		http.Error(w, "Package not found", http.StatusNotFound)
		return
	}

	// Return data for all packages
	allData := struct {
		Packages    map[string]*PackageData `json:"packages"`
		LastUpdated time.Time               `json:"last_updated"`
	}{
		Packages:    make(map[string]*PackageData),
		LastUpdated: cache.LastUpdated,
	}

	for _, pkg := range cache.AllPackages {
		allData.Packages[pkg.PackageName] = pkg
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allData)
}

// StatisticsPageHandler serves the statistics dashboard HTML page
func (h *PackageHandler) StatisticsPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Read the statistics template
	templatePath := filepath.Join(h.templatePath, "statistics.html")
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading statistics template: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse and execute the template
	tmpl, err := template.New("statistics").Parse(string(templateContent))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing statistics template: %v", err), http.StatusInternalServerError)
		return
	}

	// Execute the template with CDN resources
	templateData := struct {
		CDN map[string]string
	}{
		CDN: web.GetCDNResources(h.webService.GetConfig()),
	}
	if err := tmpl.Execute(w, templateData); err != nil {
		http.Error(w, fmt.Sprintf("Error executing statistics template: %v", err), http.StatusInternalServerError)
		return
	}
}
