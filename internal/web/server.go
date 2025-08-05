package web

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"html/template"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"nvidia_driver_monitor/internal/config"
	"nvidia_driver_monitor/internal/drivers"
	"nvidia_driver_monitor/internal/lrm"
	"nvidia_driver_monitor/internal/packages"
	"nvidia_driver_monitor/internal/releases"
	"nvidia_driver_monitor/internal/sru"
)

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

// CachedData holds all the cached package data
type CachedData struct {
	AllPackages   []*PackageData
	LastUpdated   time.Time
	IsInitialized bool
}

// WebService handles the web server functionality
type WebService struct {
	supportedReleases []releases.SupportedRelease
	udaEntries        []drivers.DriverEntry
	allBranches       drivers.AllBranches
	sruCycles         *sru.SRUCycles

	// Cache and synchronization
	cache    *CachedData
	cacheMux sync.RWMutex
	stopChan chan bool

	// HTTPS Configuration
	EnableHTTPS bool
	CertFile    string
	KeyFile     string

	// Additional configuration
	config                 *config.Config
	templatePath           string
	supportedReleasesPath  string
}

// NewWebService creates a new web service instance
func NewWebService() (*WebService, error) {
	// Initialize the service with empty cache
	ws := &WebService{
		cache: &CachedData{
			AllPackages:   make([]*PackageData, 0),
			IsInitialized: false,
		},
		stopChan:              make(chan bool),
		supportedReleasesPath: "data/supportedReleases.json", // Default path for development
	}

	// Perform initial data load
	if err := ws.refreshData(); err != nil {
		return nil, fmt.Errorf("failed to perform initial data load: %v", err)
	}

	// Initialize LRM cache
	if err := lrm.InitializeLRMCache(); err != nil {
		log.Printf("Warning: Failed to initialize LRM cache: %v", err)
		// Don't fail startup, just log the warning
	} else {
		// Start background LRM cache refresh
		lrm.StartBackgroundRefresh()
	}

	// Start background data refresh goroutine
	go ws.dataRefreshLoop()

	return ws, nil
}

// NewWebServiceWithConfig creates a new web service instance with configuration
func NewWebServiceWithConfig(cfg *config.Config, templatePath string, supportedReleasesPath string) (*WebService, error) {
	// Set global configuration for packages
	packages.SetPackagesConfig(cfg)

	// Initialize the service with empty cache
	ws := &WebService{
		cache: &CachedData{
			AllPackages:   make([]*PackageData, 0),
			IsInitialized: false,
		},
		stopChan:              make(chan bool),
		config:                cfg,
		templatePath:          templatePath,
		supportedReleasesPath: supportedReleasesPath,
	}

	// Start initial data load in background
	log.Printf("Starting background data refresh...")
	go func() {
		if err := ws.refreshData(); err != nil {
			log.Printf("Background data refresh failed: %v", err)
		} else {
			log.Printf("Background data refresh completed successfully")
		}
	}()

	// Initialize LRM cache in background
	go func() {
		if err := lrm.InitializeLRMCache(); err != nil {
			log.Printf("Warning: Failed to initialize LRM cache: %v", err)
			// Don't fail startup, just log the warning
		} else {
			log.Printf("LRM cache initialized successfully")
			// Start background LRM cache refresh
			lrm.StartBackgroundRefresh()
		}
	}()

	// Start background data refresh goroutine with configured interval
	go ws.dataRefreshLoop()

	return ws, nil
}

// refreshData fetches all data and updates the cache
func (ws *WebService) refreshData() error {
	log.Printf("Refreshing data...")

	// Get the latest UDA releases from nvidia.com
	udaEntries, err := drivers.GetNvidiaDriverEntries(ws.config)
	if err != nil {
		return fmt.Errorf("failed to get UDA entries: %v", err)
	}

	// Get server driver versions
	_, allBranches, err := drivers.GetLatestServerDriverVersions(ws.config)
	if err != nil {
		return fmt.Errorf("failed to get server driver versions: %v", err)
	}

	// Read supported releases configuration
	supportedReleases, err := releases.ReadSupportedReleases(ws.supportedReleasesPath)
	if err != nil {
		return fmt.Errorf("failed to read supported releases: %v", err)
	}

	// Update supported releases with latest versions
	releases.UpdateSupportedUDAReleases(udaEntries, supportedReleases)
	releases.UpdateSupportedReleasesWithLatestERD(allBranches, supportedReleases)

	// Fetch SRU cycles with fallback
	sruCycles, err := sru.FetchSRUCycles()
	if err != nil {
		log.Printf("Warning: Failed to fetch SRU cycles: %v", err)
		log.Printf("Using fallback SRU cycles with estimated dates")
		sruCycles = sru.CreateFallbackSRUCycles()
	} else {
		sruCycles.AddPredictedCycles()
	}

	// Update service state
	ws.udaEntries = udaEntries
	ws.allBranches = allBranches
	ws.supportedReleases = supportedReleases
	ws.sruCycles = sruCycles

	// Generate all package data
	var allPackages []*PackageData
	for _, release := range ws.supportedReleases {
		packageName := "nvidia-graphics-drivers-" + release.BranchName
		packageData, err := ws.generatePackageData(packageName)
		if err != nil {
			log.Printf("Error generating data for %s: %v", packageName, err)
			continue
		}
		allPackages = append(allPackages, packageData)
	}

	// Update cache with write lock
	ws.cacheMux.Lock()
	ws.cache.AllPackages = allPackages
	ws.cache.LastUpdated = time.Now()
	ws.cache.IsInitialized = true
	ws.cacheMux.Unlock()

	log.Printf("Data refresh completed. Generated %d packages.", len(allPackages))
	return nil
}

// dataRefreshLoop runs in the background and refreshes data every 5 minutes
func (ws *WebService) dataRefreshLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := ws.refreshData(); err != nil {
				log.Printf("Background data refresh failed: %v", err)
			}
		case <-ws.stopChan:
			log.Printf("Stopping data refresh loop...")
			return
		}
	}
}

// Stop gracefully stops the background data refresh
func (ws *WebService) Stop() {
	log.Printf("Stopping web service...")

	// Stop the main data refresh loop
	close(ws.stopChan)

	// Stop the LRM background refresh
	lrm.StopBackgroundRefresh()

	log.Printf("Web service stopped")
}

// getCachedPackages returns a copy of the cached package data
func (ws *WebService) getCachedPackages() ([]*PackageData, time.Time, bool) {
	ws.cacheMux.RLock()
	defer ws.cacheMux.RUnlock()

	// Create a deep copy to avoid race conditions
	packages := make([]*PackageData, len(ws.cache.AllPackages))
	copy(packages, ws.cache.AllPackages)

	return packages, ws.cache.LastUpdated, ws.cache.IsInitialized
}

// generatePackageData generates the table data for a specific package
func (ws *WebService) generatePackageData(packageName string) (*PackageData, error) {
	// Get source package versions
	sourceVersions, err := packages.GetMaxSourceVersionsArchive(ws.config, packageName)
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

	// Check if we have any source versions at all
	hasSourceVersions := len(sourceVersions.VersionMap) > 0

	if hasSourceVersions {
		// Normal case: package exists in Launchpad archive
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
			sruCycleDate := "-"

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
						// If version is red (upstream is greater), find SRU cycle
						if ws.sruCycles != nil && supported.DatePublished != "" {
							if sruCycle := ws.sruCycles.GetMinimumCutoffAfterDate(supported.DatePublished); sruCycle != nil {
								sruCycleDate = sruCycle.ReleaseDate
							}
						}
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
						// If version is red (upstream is greater), find SRU cycle
						if ws.sruCycles != nil && supported.DatePublished != "" && sruCycleDate == "-" {
							if sruCycle := ws.sruCycles.GetMinimumCutoffAfterDate(supported.DatePublished); sruCycle != nil {
								sruCycleDate = sruCycle.ReleaseDate
							}
						}
					}
				}
			}

			seriesData = append(seriesData, SeriesData{
				Series:          series,
				UpdatesSecurity: updates,
				Proposed:        proposed,
				UpstreamVersion: upstreamVersion,
				ReleaseDate:     releaseDate,
				SRUCycle:        sruCycleDate,
				UpdatesColor:    updatesColor,
				ProposedColor:   proposedColor,
			})
		}
	} else if found && supported.CurrentUpstreamVersion != "" {
		// Special case: upstream version exists but no Launchpad packages yet
		// Show supported series with N/A for packages but upstream info
		upstreamVersion := supported.CurrentUpstreamVersion
		releaseDate := supported.DatePublished
		sruCycleDate := "-"
		
		// Calculate SRU cycle for when this might be available
		if ws.sruCycles != nil && supported.DatePublished != "" {
			if sruCycle := ws.sruCycles.GetMinimumCutoffAfterDate(supported.DatePublished); sruCycle != nil {
				sruCycleDate = sruCycle.ReleaseDate
			}
		}

		// Show entry for supported series where this driver should be available
		for _, series := range orderedSeries {
			// Check if this series is supported for this branch
			if supported.IsSupported != nil {
				seriesSupported := false
				switch series {
				case "bionic":
					seriesSupported = supported.IsSupported["bionic"]
				case "focal":
					seriesSupported = supported.IsSupported["focal"]
				case "jammy":
					seriesSupported = supported.IsSupported["jammy"]
				case "noble":
					seriesSupported = supported.IsSupported["noble"]
				case "plucky":
					seriesSupported = supported.IsSupported["plucky"]
				case "questing":
					seriesSupported = supported.IsSupported["devel"] // devel maps to development series
				}
				
				if seriesSupported {
					seriesData = append(seriesData, SeriesData{
						Series:          series,
						UpdatesSecurity: "N/A",
						Proposed:        "N/A",
						UpstreamVersion: upstreamVersion,
						ReleaseDate:     releaseDate,
						SRUCycle:        sruCycleDate,
						UpdatesColor:    "",
						ProposedColor:   "",
					})
				}
			}
		}
	}

	return &PackageData{
		PackageName: packageName,
		Series:      seriesData,
	}, nil
}

// generateSelfSignedCert generates a self-signed certificate for HTTPS
func generateSelfSignedCert(certFile, keyFile string) error {
	// Generate private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %v", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"NVIDIA Driver Monitor"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Local"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:    []string{"localhost"},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %v", err)
	}

	// Save certificate to file
	certOut, err := os.Create(certFile)
	if err != nil {
		return fmt.Errorf("failed to create certificate file: %v", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("failed to write certificate: %v", err)
	}

	// Save private key to file
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return fmt.Errorf("failed to create key file: %v", err)
	}
	defer keyOut.Close()

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %v", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privDER}); err != nil {
		return fmt.Errorf("failed to write private key: %v", err)
	}

	return nil
}

// indexHandler handles the main page request
func (ws *WebService) indexHandler(w http.ResponseWriter, r *http.Request) {
	// Get cached data
	allPackages, lastUpdated, isInitialized := ws.getCachedPackages()

	if !isInitialized {
		http.Error(w, "Service is still initializing, please try again in a moment", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Read the index template
	templatePath := filepath.Join(ws.templatePath, "index.html")
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
		AllPackages []*PackageData
		LastUpdated time.Time
		CDN         map[string]string
	}{
		AllPackages: allPackages,
		LastUpdated: lastUpdated,
		CDN:         GetCDNResources(ws.config),
	}

	// Execute the template
	if err := tmpl.Execute(w, templateData); err != nil {
		http.Error(w, fmt.Sprintf("Error executing index template: %v", err), http.StatusInternalServerError)
		return
	}
}

// packageHandler handles requests for specific package information
func (ws *WebService) packageHandler(w http.ResponseWriter, r *http.Request) {
	packageName := r.URL.Query().Get("name")
	if packageName == "" {
		http.Error(w, "Package name is required", http.StatusBadRequest)
		return
	}

	// Check cache first for the specific package
	allPackages, _, isInitialized := ws.getCachedPackages()
	if !isInitialized {
		http.Error(w, "Service is still initializing, please try again in a moment", http.StatusServiceUnavailable)
		return
	}

	// Find the package in cache
	var packageData *PackageData
	for _, pkg := range allPackages {
		if pkg.PackageName == packageName {
			packageData = pkg
			break
		}
	}

	if packageData == nil {
		http.Error(w, "Package not found", http.StatusNotFound)
		return
	}

	packageTemplate := `
<!DOCTYPE html>
<html>
<head>
    <title>{{.PackageName}} - NVIDIA Driver Package Status</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link href="{{.CDN.BootstrapCSS}}" rel="stylesheet">
    <style>
        .container-fluid { max-width: 1200px; }
        .table-success { background-color: #d1e7dd !important; }
        .table-danger { background-color: #f8d7da !important; }
        .badge { font-size: 0.9em; }
    </style>
</head>
<body>
    <div class="container-fluid mt-4">
        <h1 class="mb-4">{{.PackageName}}</h1>
        
        <div class="alert alert-info">
            <strong>Status Legend:</strong>
            <span class="badge bg-success ms-2">Green</span> = Up to date with upstream
            <span class="badge bg-danger ms-2">Red</span> = Outdated (shows next SRU cycle date)
        </div>

        <div class="table-responsive">
            <table class="table table-striped table-bordered">
                <thead class="table-dark">
                    <tr>
                        <th>Series</th>
                        <th>Updates/Security</th>
                        <th>Proposed</th>
                        <th>Upstream Version</th>
                        <th>Release Date</th>
                        <th>Next SRU Cycle</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Series}}
                    <tr>
                        <td><strong>{{.Series}}</strong></td>
                        <td class="{{if eq .UpdatesColor "success"}}table-success{{else if eq .UpdatesColor "danger"}}table-danger{{end}}">
                            {{.UpdatesSecurity}}
                        </td>
                        <td class="{{if eq .ProposedColor "success"}}table-success{{else if eq .ProposedColor "danger"}}table-danger{{end}}">
                            {{.Proposed}}
                        </td>
                        <td>{{.UpstreamVersion}}</td>
                        <td>{{.ReleaseDate}}</td>
                        <td>
                            {{if ne .SRUCycle "-"}}
                                <span class="badge bg-warning text-dark">{{.SRUCycle}}</span>
                            {{else}}
                                -
                            {{end}}
                        </td>
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
		CDN:         GetCDNResources(ws.config),
	}

	if err := tmpl.Execute(w, templateData); err != nil {
		http.Error(w, fmt.Sprintf("Error executing template: %v", err), http.StatusInternalServerError)
		return
	}
}

// apiHandler handles JSON API requests
func (ws *WebService) apiHandler(w http.ResponseWriter, r *http.Request) {
	packageName := r.URL.Query().Get("package")

	// Get cached data
	allPackages, lastUpdated, isInitialized := ws.getCachedPackages()
	if !isInitialized {
		http.Error(w, "Service is still initializing, please try again in a moment", http.StatusServiceUnavailable)
		return
	}

	if packageName != "" {
		// Return data for specific package
		for _, pkg := range allPackages {
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
		LastUpdated: lastUpdated,
	}

	for _, pkg := range allPackages {
		allData.Packages[pkg.PackageName] = pkg
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allData)
}

// Start starts the web server with optional HTTPS support
func (ws *WebService) Start(addr string) error {
	// Create rate limiter if configured
	var rateLimiter *RateLimiter
	if ws.config != nil && ws.config.RateLimit.Enabled {
		rateLimiter = NewRateLimiter(ws.config.RateLimit.RequestsPerMinute, true)
		log.Printf("Rate limiting enabled: %d requests per minute", ws.config.RateLimit.RequestsPerMinute)
	}

	// Create handlers
	lrmHandler := NewLRMHandler(ws.templatePath, ws.config)
	apiHandler := NewAPIHandler()

	// Setup routes with security headers and optional rate limiting
	if rateLimiter != nil {
		http.Handle("/", SecurityHeadersMiddleware(rateLimiter.Middleware(http.HandlerFunc(ws.indexHandler))))
		http.Handle("/package", SecurityHeadersMiddleware(rateLimiter.Middleware(http.HandlerFunc(ws.packageHandler))))
		http.Handle("/api", SecurityHeadersMiddleware(rateLimiter.Middleware(http.HandlerFunc(ws.apiHandler))))
		http.Handle("/l-r-m-verifier", SecurityHeadersMiddleware(rateLimiter.Middleware(lrmHandler)))
		http.Handle("/statistics", SecurityHeadersMiddleware(rateLimiter.Middleware(http.HandlerFunc(ws.statisticsPageHandler))))

		// Static files for statistics dashboard
		http.Handle("/static/", SecurityHeadersMiddleware(rateLimiter.Middleware(http.StripPrefix("/static", http.FileServer(http.Dir("static"))))))

		// New API endpoints
		http.Handle("/api/lrm", SecurityHeadersMiddleware(rateLimiter.Middleware(http.HandlerFunc(apiHandler.LRMDataHandler))))
		http.Handle("/api/health", SecurityHeadersMiddleware(rateLimiter.Middleware(http.HandlerFunc(apiHandler.HealthHandler))))
		http.Handle("/api/routings", SecurityHeadersMiddleware(rateLimiter.Middleware(http.HandlerFunc(apiHandler.RoutingsHandler))))
		http.Handle("/api/cache-status", SecurityHeadersMiddleware(rateLimiter.Middleware(http.HandlerFunc(apiHandler.CacheStatusHandler))))
		http.Handle("/api/statistics", SecurityHeadersMiddleware(rateLimiter.Middleware(http.HandlerFunc(apiHandler.StatisticsHandler))))
	} else {
		http.Handle("/", SecurityHeadersMiddleware(http.HandlerFunc(ws.indexHandler)))
		http.Handle("/package", SecurityHeadersMiddleware(http.HandlerFunc(ws.packageHandler)))
		http.Handle("/api", SecurityHeadersMiddleware(http.HandlerFunc(ws.apiHandler)))
		http.Handle("/l-r-m-verifier", SecurityHeadersMiddleware(lrmHandler))
		http.Handle("/statistics", SecurityHeadersMiddleware(http.HandlerFunc(ws.statisticsPageHandler)))

		// Static files for statistics dashboard
		http.Handle("/static/", SecurityHeadersMiddleware(http.StripPrefix("/static", http.FileServer(http.Dir("static")))))

		// New API endpoints
		http.Handle("/api/lrm", SecurityHeadersMiddleware(http.HandlerFunc(apiHandler.LRMDataHandler)))
		http.Handle("/api/health", SecurityHeadersMiddleware(http.HandlerFunc(apiHandler.HealthHandler)))
		http.Handle("/api/routings", SecurityHeadersMiddleware(http.HandlerFunc(apiHandler.RoutingsHandler)))
		http.Handle("/api/cache-status", SecurityHeadersMiddleware(http.HandlerFunc(apiHandler.CacheStatusHandler)))
		http.Handle("/api/statistics", SecurityHeadersMiddleware(http.HandlerFunc(apiHandler.StatisticsHandler)))
	}

	if ws.EnableHTTPS {
		// Check if certificates exist, generate if they don't
		log.Printf("Checking for certificates: cert=%s, key=%s", ws.CertFile, ws.KeyFile)
		if _, err := os.Stat(ws.CertFile); os.IsNotExist(err) {
			log.Printf("Certificate file not found at %s, generating self-signed certificate...", ws.CertFile)
			if err := generateSelfSignedCert(ws.CertFile, ws.KeyFile); err != nil {
				return fmt.Errorf("failed to generate certificate: %v", err)
			}
			log.Printf("Self-signed certificate generated: %s", ws.CertFile)
		} else {
			log.Printf("Using existing certificate: %s", ws.CertFile)
		}

		// Create TLS config
		cert, err := tls.LoadX509KeyPair(ws.CertFile, ws.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to load certificate: %v", err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		server := &http.Server{
			Addr:      addr,
			TLSConfig: tlsConfig,
		}

		log.Printf("Starting HTTPS server on %s", addr)
		log.Printf("Access the service at: https://localhost%s", addr)
		return server.ListenAndServeTLS("", "")
	} else {
		log.Printf("Starting HTTP server on %s", addr)
		log.Printf("Access the service at: http://localhost%s", addr)
		return http.ListenAndServe(addr, nil)
	}
}

// lrmVerifierHandler handles requests for L-R-M verifier information
func (ws *WebService) lrmVerifierHandler(w http.ResponseWriter, r *http.Request) {
	// Set content type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Create L-R-M data using cached implementation to avoid refetching if less than 5 minutes old
	log.Printf("Fetching L-R-M data from cache")
	var lrmData *lrm.LRMVerifierData
	if realData, fetchErr := lrm.GetCachedLRMData(); fetchErr != nil {
		log.Printf("Failed to fetch cached L-R-M data, falling back to supported releases: %v", fetchErr)
		lrmData = generateLRMDataFromSupportedReleases(ws.supportedReleases)
	} else {
		log.Printf("Successfully fetched cached L-R-M data with %d kernels", len(realData.KernelResults))
		lrmData = realData
	}

	// Note: The FetchKernelLRMData function already calculates the update status
	// using the same DKMS version source as the main dashboard (packages.GetMaxSourceVersionsArchive).
	// No need to override it here.

	// Create template
	lrmTemplate := `
<!DOCTYPE html>
<html>
<head>
    <title>Linux Restricted Modules (L-R-M) Verifier</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link href="{{.CDN.BootstrapCSS}}" rel="stylesheet">
    <style>
        .container-fluid { max-width: 1600px; }
        .table-success { background-color: #d1e7dd !important; }
        .table-warning { background-color: #fff3cd !important; }
        .table-danger { background-color: #f8d7da !important; }
        .badge { font-size: 0.9em; }
        .kernel-table th { background-color: #f8f9fa; font-weight: 600; }
        .last-updated { font-size: 0.9em; color: #6c757d; }
    </style>
</head>
<body>
    <div class="container-fluid mt-4">
        <div class="d-flex justify-content-between align-items-center mb-4">
            <h1>Linux Restricted Modules (L-R-M) Verifier</h1>
            <a href="/" class="btn btn-secondary">← Back to Main</a>
        </div>
        
        <div class="alert alert-info">
            <strong>What this does:</strong> This tool displays kernel L-R-M information for supported NVIDIA driver releases, 
            showing versioning of the kernels and their corresponding linux-restricted-modules packages, 
            and verifies that source files are using the latest DKMS version.
        </div>

        <div class="row mb-4">
            <div class="col-md-3">
                <div class="card text-center">
                    <div class="card-body">
                        <h5 class="card-title">{{.Data.TotalKernels}}</h5>
                        <p class="card-text">Total Kernels</p>
                    </div>
                </div>
            </div>
            <div class="col-md-3">
                <div class="card text-center">
                    <div class="card-body">
                        <h5 class="card-title">{{.Data.SupportedLRM}}</h5>
                        <p class="card-text">Supported with L-R-M</p>
                    </div>
                </div>
            </div>
            <div class="col-md-3">
                <div class="card text-center">
                    <div class="card-body">
                        <h5 class="card-title">{{len .Data.KernelResults}}</h5>
                        <p class="card-text">Displayed Results</p>
                    </div>
                </div>
            </div>
            <div class="col-md-3">
                <div class="card text-center">
                    <div class="card-body">
                        <h5 class="card-title text-muted">{{.Data.LastUpdated.Format "15:04:05"}}</h5>
                        <p class="card-text">Last Updated</p>
                    </div>
                </div>
            </div>
        </div>

        {{if .Data.KernelResults}}
        <div class="table-responsive">
            <table class="table table-striped table-hover kernel-table">
                <thead>
                    <tr>
                        <th>Series</th>
                        <th>Codename</th>
                        <th>Source & Version</th>
                        <th>Routing</th>
                        <th>Status</th>
                        <th>L-R-M Package & Version</th>
                        <th>NVIDIA Driver & Status</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Data.KernelResults}}
                    <tr>
                        <td><strong>{{.Series}}</strong></td>
                        <td>{{.Codename}}</td>
                        <td>
                            <div><code>{{.Source}}</code></div>
                            {{if and (ne .SourceVersion "N/A") (ne .SourceVersion "ERROR")}}
                            <div class="small text-muted">{{.SourceVersion}}</div>
                            {{else}}
                            <div class="small text-muted">{{.SourceVersion}}</div>
                            {{end}}
                        </td>
                        <td><span class="badge bg-secondary">{{.Routing}}</span></td>
                        <td>
                            {{if .Supported}}<span class="badge bg-success">SUPPORTED</span>{{else}}<span class="badge bg-warning">NOT SUPPORTED</span>{{end}}
                            {{if .Development}}<span class="badge bg-info">DEV</span>{{end}}
                            {{if .LTS}}<span class="badge bg-primary">LTS</span>{{end}}
                            {{if .ESM}}<span class="badge bg-secondary">ESM</span>{{end}}
                        </td>
                        <td>
                            {{range .LRMPackages}}
                            <div><code>{{.}}</code></div>
                            {{end}}
                            {{if and (ne .LatestLRMVersion "N/A") (ne .LatestLRMVersion "ERROR")}}
                            <div class="small text-muted">{{.LatestLRMVersion}}</div>
                            {{else}}
                            <div class="small text-muted">{{.LatestLRMVersion}}</div>
                            {{end}}
                        </td>
                        <td>
                            {{range .NvidiaDriverStatuses}}
                            <div class="mb-1 d-flex align-items-center justify-content-between">
                                <div>
                                    <div><strong>{{simplifyDriverName .DriverName}}</strong></div>
                                    <div class="small text-muted">DSC: {{.DSCVersion}}</div>
                                    {{if .DKMSVersion}}
                                    <div class="small text-muted">DKMS: {{.DKMSVersion}}</div>
                                    {{end}}
                                </div>
                                <div class="ms-2">
                                    {{if contains .Status "✅ Up to date"}}
                                    <span class="badge bg-success">{{.Status}}</span>
                                    {{else if contains .Status "🔄 Update available"}}
                                    <span class="badge bg-warning">{{.Status}}</span>
                                    {{else if contains .Status "⚠️ Unknown"}}
                                    <span class="badge bg-secondary">{{.Status}}</span>
                                    {{else}}
                                    <span class="badge bg-secondary">{{.Status}}</span>
                                    {{end}}
                                </div>
                            </div>
                            {{end}}
                            {{if not .NvidiaDriverStatuses}}
                            <span class="text-muted">N/A</span>
                            {{end}}
                        </td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        {{else}}
        <div class="alert alert-warning">
            <h4>No kernel sources found matching the criteria.</h4>
            <p>Try changing the routing filter or check if the kernel-series.yaml data is available.</p>
        </div>
        {{end}}

        <div class="mt-4">
            <div class="last-updated">
                Data generated from supported releases at {{.Data.LastUpdated.Format "2006-01-02 15:04:05 MST"}}
            </div>
        </div>
    </div>

    <script src="{{.CDN.BootstrapJS}}"></script>
</body>
</html>
`

	// Create template with custom functions
	tmpl := template.New("lrm").Funcs(template.FuncMap{
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
	})

	var err error
	tmpl, err = tmpl.Parse(lrmTemplate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Template parsing error: %v", err), http.StatusInternalServerError)
		return
	}

	// Prepare template data
	templateData := struct {
		Data *lrm.LRMVerifierData
		CDN  map[string]string
	}{
		Data: lrmData,
		CDN:  GetCDNResources(ws.config),
	}

	// Execute template
	if err := tmpl.Execute(w, templateData); err != nil {
		http.Error(w, fmt.Sprintf("Template execution error: %v", err), http.StatusInternalServerError)
		return
	}
}

// statisticsPageHandler serves the statistics dashboard HTML page
func (ws *WebService) statisticsPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Read the statistics template
	templatePath := filepath.Join(ws.templatePath, "statistics.html")
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
		CDN: GetCDNResources(ws.config),
	}
	if err := tmpl.Execute(w, templateData); err != nil {
		http.Error(w, fmt.Sprintf("Error executing statistics template: %v", err), http.StatusInternalServerError)
		return
	}
}

// Helper functions for L-R-M verifier

// generateLRMDataFromSupportedReleases creates L-R-M data from the supported releases
func generateLRMDataFromSupportedReleases(supportedReleases []releases.SupportedRelease) *lrm.LRMVerifierData {
	var kernelResults []lrm.KernelLRMResult
	totalKernels := 0
	supportedLRM := 0

	// Map Ubuntu codenames to series for display
	codenameToSeries := map[string]string{
		"noble":    "24.04",
		"jammy":    "22.04",
		"focal":    "20.04",
		"bionic":   "18.04",
		"plucky":   "25.04",
		"oracular": "24.10",
	}

	// Common kernel sources that have L-R-M packages
	kernelSources := []string{"linux", "linux-aws", "linux-azure", "linux-gcp", "linux-oracle"}

	// Group supported releases by codename to collect all available driver branches
	releasesByCodename := make(map[string][]releases.SupportedRelease)
	for _, release := range supportedReleases {
		// Skip server versions for now
		if release.IsServer {
			continue
		}

		for codename, isSupported := range release.IsSupported {
			if isSupported {
				releasesByCodename[codename] = append(releasesByCodename[codename], release)
			}
		}
	}

	// Generate L-R-M data for each codename and kernel source combination
	for codename, releases := range releasesByCodename {
		series, exists := codenameToSeries[codename]
		if !exists {
			series = codename
		}

		for _, kernelSource := range kernelSources {
			totalKernels++

			// Generate L-R-M package name for this kernel source
			// For "linux" source, package is "linux-restricted-modules"
			// For "linux-aws", package is "linux-restricted-modules-aws" (remove "linux-" prefix)
			var lrmPackage string
			if kernelSource == "linux" {
				lrmPackage = "linux-restricted-modules"
			} else {
				// Remove "linux-" prefix from kernel source
				suffix := strings.TrimPrefix(kernelSource, "linux-")
				lrmPackage = fmt.Sprintf("linux-restricted-modules-%s", suffix)
			}

			// Collect all NVIDIA driver branches available for this codename
			var nvidiaDrivers []string
			for _, release := range releases {
				// Extract branch number (remove "-server" suffix if present)
				branch := strings.TrimSuffix(release.BranchName, "-server")
				nvidiaDrivers = append(nvidiaDrivers, branch)
			}

			if len(nvidiaDrivers) > 0 {
				supportedLRM++

				kernelResult := lrm.KernelLRMResult{
					Series:               series,
					Codename:             codename,
					Source:               kernelSource,         // Actual kernel source
					Routing:              "ubuntu/4",           // Default routing
					LRMPackages:          []string{lrmPackage}, // Actual L-R-M package
					HasLRM:               true,
					Supported:            true,
					Development:          codename == "devel" || codename == "plucky",
					LTS:                  series == "20.04" || series == "22.04" || series == "24.04",
					ESM:                  series == "18.04",
					LatestLRMVersion:     "1.0.0",       // Will be updated later
					SourceVersion:        "1.0.0",       // Will be updated later
					NvidiaDriverVersions: nvidiaDrivers, // Available driver branches
					UpdateStatus:         "N/A",
				}

				kernelResults = append(kernelResults, kernelResult)
			}
		}
	}

	return &lrm.LRMVerifierData{
		TotalKernels:  totalKernels,
		SupportedLRM:  supportedLRM,
		KernelResults: kernelResults,
		LastUpdated:   time.Now(),
		IsInitialized: true,
	}
}
