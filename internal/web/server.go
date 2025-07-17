package web

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"

	"nvidia_example_550/internal/drivers"
	"nvidia_example_550/internal/packages"
	"nvidia_example_550/internal/releases"
)

// SeriesData represents the data for a single series row
type SeriesData struct {
	Series          string
	UpdatesSecurity string
	Proposed        string
	UpstreamVersion string
	ReleaseDate     string
	UpdatesColor    string
	ProposedColor   string
}

// PackageData represents the data for a complete package table
type PackageData struct {
	PackageName string
	Series      []SeriesData
}

// WebService handles the web server functionality
type WebService struct {
	supportedReleases []releases.SupportedRelease
	udaEntries        []drivers.DriverEntry
	allBranches       drivers.AllBranches
}

// NewWebService creates a new web service instance
func NewWebService() (*WebService, error) {
	// Initialize the service with data
	ws := &WebService{}

	// Get the latest UDA releases from nvidia.com
	udaEntries, err := drivers.GetNvidiaDriverEntries()
	if err != nil {
		return nil, err
	}
	ws.udaEntries = udaEntries

	// Get server driver versions
	_, allBranches, err := drivers.GetLatestServerDriverVersions()
	if err != nil {
		return nil, err
	}
	ws.allBranches = allBranches

	// Read supported releases configuration
	supportedReleases, err := releases.ReadSupportedReleases("supportedReleases.json")
	if err != nil {
		return nil, err
	}

	// Update supported releases with latest versions
	releases.UpdateSupportedUDAReleases(udaEntries, supportedReleases)
	releases.UpdateSupportedReleasesWithLatestERD(allBranches, supportedReleases)

	ws.supportedReleases = supportedReleases

	return ws, nil
}

// generatePackageData generates the table data for a specific package
func (ws *WebService) generatePackageData(packageName string) (*PackageData, error) {
	// Get source package versions
	sourceVersions, err := packages.GetMaxSourceVersionsArchive(packageName)
	if err != nil {
		return nil, err
	}

	// Build a lookup: branch name -> SupportedRelease
	supportedMap := make(map[string]releases.SupportedRelease)
	for _, rel := range ws.supportedReleases {
		supportedMap[rel.BranchName] = rel
	}

	// Extract branch name from package name
	branchName := ""
	parts := strings.Split(packageName, "-")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "server" && i > 0 {
			branchName = parts[i-1] + "-server"
			break
		}
		if _, ok := supportedMap[parts[i]]; ok {
			branchName = parts[i]
			break
		}
	}
	// Fallback: try just last digits
	if branchName == "" {
		for i := len(parts) - 1; i >= 0; i-- {
			if _, ok := supportedMap[parts[i]]; ok {
				branchName = parts[i]
				break
			}
		}
	}

	supported, found := supportedMap[branchName]

	orderedSeries := []string{"questing", "plucky", "noble", "jammy", "focal", "bionic"}
	var seriesData []SeriesData

	for _, series := range orderedSeries {
		pocket, exists := sourceVersions.VersionMap[series]
		if !exists {
			continue // Skip series that don't exist in the version map
		}

		updates := "-"
		proposed := "-"
		updatesColor := ""
		proposedColor := ""
		upstreamVersion := "-"
		releaseDate := "-"

		if found && supported.CurrentUpstreamVersion != "" {
			upstreamVersion = supported.CurrentUpstreamVersion
			if supported.DatePublished != "" {
				releaseDate = supported.DatePublished
			}
		}

		if pocket != nil && pocket.UpdatesSecurity.String() != "" {
			updates = pocket.UpdatesSecurity.String()
			if found && supported.CurrentUpstreamVersion != "" {
				// Check if the upstream version is contained in the package version
				if strings.Contains(updates, supported.CurrentUpstreamVersion) {
					updatesColor = "success"
				} else {
					updatesColor = "danger"
				}
			}
		}

		if pocket != nil && pocket.Proposed.String() != "" {
			proposed = pocket.Proposed.String()
			if found && supported.CurrentUpstreamVersion != "" {
				// Check if the upstream version is contained in the package version
				if strings.Contains(proposed, supported.CurrentUpstreamVersion) {
					proposedColor = "success"
				} else {
					proposedColor = "danger"
				}
			}
		}

		seriesData = append(seriesData, SeriesData{
			Series:          series,
			UpdatesSecurity: updates,
			Proposed:        proposed,
			UpstreamVersion: upstreamVersion,
			ReleaseDate:     releaseDate,
			UpdatesColor:    updatesColor,
			ProposedColor:   proposedColor,
		})
	}

	return &PackageData{
		PackageName: packageName,
		Series:      seriesData,
	}, nil
}

// indexHandler handles the main page
func (ws *WebService) indexHandler(w http.ResponseWriter, r *http.Request) {
	var allPackages []PackageData

	// Process each supported release
	for _, release := range ws.supportedReleases {
		currentPackageName := "nvidia-graphics-drivers-" + release.BranchName

		packageData, err := ws.generatePackageData(currentPackageName)
		if err != nil {
			http.Error(w, "Error generating package data: "+err.Error(), http.StatusInternalServerError)
			return
		}

		allPackages = append(allPackages, *packageData)
	}

	tmpl := `
<!DOCTYPE html>
<html>
<head>
	<title>NVIDIA Driver Package Status</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			margin: 20px;
			background-color: #f5f5f5;
		}
		.container {
			max-width: 1200px;
			margin: 0 auto;
			background-color: white;
			padding: 20px;
			border-radius: 8px;
			box-shadow: 0 2px 4px rgba(0,0,0,0.1);
		}
		h1 {
			color: #333;
			text-align: center;
			margin-bottom: 30px;
		}
		.package-section {
			margin-bottom: 40px;
		}
		.package-title {
			font-size: 18px;
			font-weight: bold;
			color: #444;
			margin-bottom: 10px;
			padding: 10px;
			background-color: #f8f9fa;
			border-left: 4px solid #007bff;
		}
		table {
			width: 100%;
			border-collapse: collapse;
			margin-bottom: 20px;
		}
		th, td {
			border: 1px solid #dee2e6;
			padding: 12px;
			text-align: left;
		}
		th {
			background-color: #e9ecef;
			font-weight: bold;
			color: #495057;
		}
		tr:nth-child(even) {
			background-color: #f8f9fa;
		}
		.success {
			background-color: #d4edda;
			color: #155724;
		}
		.danger {
			background-color: #f8d7da;
			color: #721c24;
		}
		.series-cell {
			font-weight: bold;
			color: #495057;
		}
		.upstream-cell {
			font-weight: bold;
			color: #007bff;
		}
		.footer {
			text-align: center;
			color: #6c757d;
			margin-top: 40px;
			font-size: 14px;
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>NVIDIA Driver Package Status</h1>
		
		{{range .}}
		<div class="package-section">
			<div class="package-title">{{.PackageName}}</div>
			<table>
				<thead>
					<tr>
						<th>Series</th>
						<th>Updates/Security</th>
						<th>Proposed</th>
						<th>Upstream Version</th>
						<th>Release Date</th>
					</tr>
				</thead>
				<tbody>
					{{range .Series}}
					<tr>
						<td class="series-cell">{{.Series}}</td>
						<td class="{{.UpdatesColor}}">{{.UpdatesSecurity}}</td>
						<td class="{{.ProposedColor}}">{{.Proposed}}</td>
						<td class="upstream-cell">{{.UpstreamVersion}}</td>
						<td class="upstream-cell">{{.ReleaseDate}}</td>
					</tr>
					{{end}}
				</tbody>
			</table>
		</div>
		{{end}}
		
		<div class="footer">
			<p>Green background indicates package version contains upstream version</p>
			<p>Red background indicates package version does not contain upstream version</p>
		</div>
	</div>
</body>
</html>
	`

	t, err := template.New("index").Parse(tmpl)
	if err != nil {
		http.Error(w, "Error parsing template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := t.Execute(w, allPackages); err != nil {
		http.Error(w, "Error executing template: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// packageHandler handles individual package requests
func (ws *WebService) packageHandler(w http.ResponseWriter, r *http.Request) {
	packageName := r.URL.Query().Get("package")
	if packageName == "" {
		http.Error(w, "Package name is required", http.StatusBadRequest)
		return
	}

	packageData, err := ws.generatePackageData(packageName)
	if err != nil {
		http.Error(w, "Error generating package data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl := `
<!DOCTYPE html>
<html>
<head>
	<title>{{.PackageName}} - NVIDIA Driver Package Status</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			margin: 20px;
			background-color: #f5f5f5;
		}
		.container {
			max-width: 1000px;
			margin: 0 auto;
			background-color: white;
			padding: 20px;
			border-radius: 8px;
			box-shadow: 0 2px 4px rgba(0,0,0,0.1);
		}
		h1 {
			color: #333;
			text-align: center;
			margin-bottom: 30px;
		}
		.package-title {
			font-size: 18px;
			font-weight: bold;
			color: #444;
			margin-bottom: 10px;
			padding: 10px;
			background-color: #f8f9fa;
			border-left: 4px solid #007bff;
		}
		table {
			width: 100%;
			border-collapse: collapse;
			margin-bottom: 20px;
		}
		th, td {
			border: 1px solid #dee2e6;
			padding: 12px;
			text-align: left;
		}
		th {
			background-color: #e9ecef;
			font-weight: bold;
			color: #495057;
		}
		tr:nth-child(even) {
			background-color: #f8f9fa;
		}
		.success {
			background-color: #d4edda;
			color: #155724;
		}
		.danger {
			background-color: #f8d7da;
			color: #721c24;
		}
		.series-cell {
			font-weight: bold;
			color: #495057;
		}
		.upstream-cell {
			font-weight: bold;
			color: #007bff;
		}
		.back-link {
			display: inline-block;
			margin-bottom: 20px;
			color: #007bff;
			text-decoration: none;
		}
		.back-link:hover {
			text-decoration: underline;
		}
	</style>
</head>
<body>
	<div class="container">
		<a href="/" class="back-link">← Back to All Packages</a>
		<h1>{{.PackageName}}</h1>
		
		<div class="package-title">Package Information</div>
		<table>
			<thead>
				<tr>
					<th>Series</th>
					<th>Updates/Security</th>
					<th>Proposed</th>
					<th>Upstream Version</th>
					<th>Release Date</th>
				</tr>
			</thead>
			<tbody>
				{{range .Series}}
				<tr>
					<td class="series-cell">{{.Series}}</td>
					<td class="{{.UpdatesColor}}">{{.UpdatesSecurity}}</td>
					<td class="{{.ProposedColor}}">{{.Proposed}}</td>
					<td class="upstream-cell">{{.UpstreamVersion}}</td>
					<td class="upstream-cell">{{.ReleaseDate}}</td>
				</tr>
				{{end}}
			</tbody>
		</table>
	</div>
</body>
</html>
	`

	t, err := template.New("package").Parse(tmpl)
	if err != nil {
		http.Error(w, "Error parsing template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := t.Execute(w, packageData); err != nil {
		http.Error(w, "Error executing template: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// apiHandler handles JSON API requests
func (ws *WebService) apiHandler(w http.ResponseWriter, r *http.Request) {
	packageName := r.URL.Query().Get("package")
	if packageName == "" {
		// Return all packages
		var allPackages []PackageData

		for _, release := range ws.supportedReleases {
			currentPackageName := "nvidia-graphics-drivers-" + release.BranchName

			packageData, err := ws.generatePackageData(currentPackageName)
			if err != nil {
				http.Error(w, "Error generating package data: "+err.Error(), http.StatusInternalServerError)
				return
			}

			allPackages = append(allPackages, *packageData)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(allPackages)
		return
	}

	// Return specific package
	packageData, err := ws.generatePackageData(packageName)
	if err != nil {
		http.Error(w, "Error generating package data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(packageData)
}

// Start starts the web server
func (ws *WebService) Start(addr string) error {
	http.HandleFunc("/", ws.indexHandler)
	http.HandleFunc("/package", ws.packageHandler)
	http.HandleFunc("/api", ws.apiHandler)

	return http.ListenAndServe(addr, nil)
}
