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
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"nvidia_driver_monitor/internal/drivers"
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

// WebService handles the web server functionality
type WebService struct {
	supportedReleases []releases.SupportedRelease
	udaEntries        []drivers.DriverEntry
	allBranches       drivers.AllBranches
	sruCycles         *sru.SRUCycles

	// HTTPS Configuration
	EnableHTTPS bool
	CertFile    string
	KeyFile     string
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

	// Fetch SRU cycles
	sruCycles, err := sru.FetchSRUCycles()
	if err != nil {
		return nil, err
	}
	sruCycles.AddPredictedCycles()
	ws.sruCycles = sruCycles

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
		return fmt.Errorf("failed to create cert file: %v", err)
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

	privKeyBytes := x509.MarshalPKCS1PrivateKey(priv)
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privKeyBytes}); err != nil {
		return fmt.Errorf("failed to write private key: %v", err)
	}

	return nil
}

// indexHandler handles the main page request
func (ws *WebService) indexHandler(w http.ResponseWriter, r *http.Request) {
	var allPackages []PackageData

	// Generate package names from supported releases like main.go does
	for _, release := range ws.supportedReleases {
		packageName := "nvidia-graphics-drivers-" + release.BranchName
		packageData, err := ws.generatePackageData(packageName)
		if err != nil {
			// Log error but continue with other packages
			fmt.Printf("Error generating data for %s: %v\n", packageName, err)
			continue
		}
		allPackages = append(allPackages, *packageData)
	}

	indexTemplate := `
<!DOCTYPE html>
<html>
<head>
    <title>NVIDIA Driver Package Status</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/css/bootstrap.min.css" rel="stylesheet">
    <style>
        .container-fluid { max-width: 1400px; }
        .package-section { margin-bottom: 40px; }
        .package-title {
            font-size: 18px;
            font-weight: bold;
            color: #444;
            margin-bottom: 15px;
            padding: 15px;
            background-color: #f8f9fa;
            border-left: 4px solid #007bff;
            border-radius: 4px;
        }
        .table th { 
            background-color: #e9ecef;
            font-weight: bold;
            color: #495057;
        }
        .table-success {
            background-color: #d4edda;
            color: #155724;
        }
        .table-danger {
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
        .badge { font-size: 0.8em; }
    </style>
</head>
<body>
    <div class="container-fluid mt-4">
        <h1 class="mb-4">NVIDIA Driver Package Status Monitor</h1>
        
        <div class="alert alert-info">
            <strong>Status Legend:</strong>
            <span class="badge bg-success ms-2">Green</span> = Up to date
            <span class="badge bg-danger ms-2">Red</span> = Outdated (shows next SRU cycle date)
        </div>

        {{range .}}
        <div class="package-section">
            <div class="package-title">{{.PackageName}}</div>
            <div class="table-responsive">
                <table class="table table-striped table-bordered">
                    <thead>
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
                            <td class="series-cell"><strong>{{.Series}}</strong></td>
                            <td class="{{if eq .UpdatesColor "success"}}table-success{{else if eq .UpdatesColor "danger"}}table-danger{{end}}">
                                {{.UpdatesSecurity}}
                            </td>
                            <td class="{{if eq .ProposedColor "success"}}table-success{{else if eq .ProposedColor "danger"}}table-danger{{end}}">
                                {{.Proposed}}
                            </td>
                            <td class="upstream-cell">{{.UpstreamVersion}}</td>
                            <td class="upstream-cell">{{.ReleaseDate}}</td>
                            <td class="upstream-cell">
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
        </div>
        {{end}}

        <div class="card mt-4">
            <div class="card-header">
                <h5 class="card-title mb-0">API Endpoints</h5>
            </div>
            <div class="card-body">
                <p><a href="/api" class="btn btn-outline-primary">View JSON API Data</a></p>
                <small class="text-muted">Provides structured JSON data for all packages</small>
            </div>
        </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html>`

	tmpl, err := template.New("index").Parse(indexTemplate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing template: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, allPackages); err != nil {
		http.Error(w, fmt.Sprintf("Error executing template: %v", err), http.StatusInternalServerError)
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

	packageData, err := ws.generatePackageData(packageName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error generating package data: %v", err), http.StatusInternalServerError)
		return
	}

	packageTemplate := `
<!DOCTYPE html>
<html>
<head>
    <title>{{.PackageName}} - Package Status</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/css/bootstrap.min.css" rel="stylesheet">
    <style>
        .container-fluid { max-width: 1200px; }
        .table-success { background-color: #d1e7dd !important; }
        .table-danger { background-color: #f8d7da !important; }
        .badge { font-size: 0.9em; }
    </style>
</head>
<body>
    <div class="container-fluid mt-4">
        <nav aria-label="breadcrumb">
            <ol class="breadcrumb">
                <li class="breadcrumb-item"><a href="/">Home</a></li>
                <li class="breadcrumb-item active">{{.PackageName}}</li>
            </ol>
        </nav>
        
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

    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html>`

	tmpl, err := template.New("package").Parse(packageTemplate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing template: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, packageData); err != nil {
		http.Error(w, fmt.Sprintf("Error executing template: %v", err), http.StatusInternalServerError)
		return
	}
}

// apiHandler handles JSON API requests
func (ws *WebService) apiHandler(w http.ResponseWriter, r *http.Request) {
	packageName := r.URL.Query().Get("package")
	if packageName != "" {
		// Return data for specific package
		packageData, err := ws.generatePackageData(packageName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error generating package data: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(packageData)
		return
	}

	// Return data for all packages generated from supported releases
	allData := make(map[string]*PackageData)
	for _, release := range ws.supportedReleases {
		packageName := "nvidia-graphics-drivers-" + release.BranchName
		packageData, err := ws.generatePackageData(packageName)
		if err != nil {
			// Log error but continue with other packages
			fmt.Printf("Error generating data for %s: %v\n", packageName, err)
			continue
		}
		allData[packageName] = packageData
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allData)
}

// Start starts the web server with optional HTTPS support
func (ws *WebService) Start(addr string) error {
	http.HandleFunc("/", ws.indexHandler)
	http.HandleFunc("/package", ws.packageHandler)
	http.HandleFunc("/api", ws.apiHandler)

	if ws.EnableHTTPS {
		// Check if certificates exist, generate if they don't
		if ws.CertFile == "" || ws.KeyFile == "" {
			// Default certificate locations
			ws.CertFile = "server.crt"
			ws.KeyFile = "server.key"
		}

		// Check if certificate files exist
		if _, err := os.Stat(ws.CertFile); os.IsNotExist(err) {
			fmt.Printf("Certificate file not found, generating self-signed certificate...\n")
			if err := generateSelfSignedCert(ws.CertFile, ws.KeyFile); err != nil {
				return fmt.Errorf("failed to generate certificate: %v", err)
			}
			fmt.Printf("Generated certificate: %s\n", ws.CertFile)
			fmt.Printf("Generated private key: %s\n", ws.KeyFile)
		}

		// Configure TLS with security best practices
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			CurvePreferences: []tls.CurveID{
				tls.CurveP256,
				tls.X25519,
			},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		}

		// Create HTTPS server
		server := &http.Server{
			Addr:         addr,
			TLSConfig:    tlsConfig,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		fmt.Printf("Starting HTTPS server on %s\n", addr)
		fmt.Printf("Certificate: %s\n", ws.CertFile)
		fmt.Printf("Private Key: %s\n", ws.KeyFile)
		fmt.Printf("Access the service at: https://localhost%s\n", addr)

		return server.ListenAndServeTLS(ws.CertFile, ws.KeyFile)
	}

	// Default HTTP server
	fmt.Printf("Starting HTTP server on %s\n", addr)
	fmt.Printf("Access the service at: http://localhost%s\n", addr)
	return http.ListenAndServe(addr, nil)
}
