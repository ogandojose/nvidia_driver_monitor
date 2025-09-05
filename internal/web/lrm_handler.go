package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"nvidia_driver_monitor/internal/config"
	"nvidia_driver_monitor/internal/lrm"
)

// LRMHandler handles the L-R-M verifier page
type LRMHandler struct {
	templatePath      string
	config            *config.Config
	supportedReleases interface{} // TODO: Define proper type
}

// NewLRMHandler creates a new LRM handler
func NewLRMHandler(templatePath string, cfg *config.Config) *LRMHandler {
	return &LRMHandler{
		templatePath: templatePath,
		config:       cfg,
	}
}

// ServeHTTP handles requests for L-R-M verifier information
func (h *LRMHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	reqID := start.UnixNano()
	log.Printf("[LRM ServeHTTP] start req=%d method=%s path=%s at=%s", reqID, r.Method, r.URL.Path, start.Format(time.RFC3339Nano))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	var lrmData *lrm.LRMVerifierData
	cacheStart := time.Now()

	// If cache not initialized yet, render shell with progress bar and avoid blocking fetch
	cacheStatus := lrm.GetCacheStatus()
	if initVal, ok := cacheStatus["initialized"].(bool); ok && !initVal {
		log.Printf("[LRM ServeHTTP] req=%d cache not initialized; rendering progress shell", reqID)
		lrmData = &lrm.LRMVerifierData{
			KernelResults: []lrm.KernelLRMResult{},
			TotalKernels:  0,
			SupportedLRM:  0,
			IsInitialized: false,
		}
	} else if realData, fetchErr := lrm.GetCachedLRMData(); fetchErr != nil {
		log.Printf("[LRM ServeHTTP] req=%d GetCachedLRMData error after=%s err=%v", reqID, time.Since(cacheStart), fetchErr)
		// Fallback to generating data from supported releases if available
		lrmData = &lrm.LRMVerifierData{
			KernelResults: []lrm.KernelLRMResult{},
			TotalKernels:  0,
			SupportedLRM:  0,
			IsInitialized: false,
		}
	} else {
		log.Printf("[LRM ServeHTTP] req=%d GetCachedLRMData ok after=%s results=%d initialized=%v", reqID, time.Since(cacheStart), len(realData.KernelResults), realData.IsInitialized)
		lrmData = realData
	}

	// Load and parse template
	templateFile := filepath.Join(h.templatePath, "lrm_verifier.html")
	tmpl := template.New("lrm_verifier.html").Funcs(TemplateFunctions())

	var err error
	parseStart := time.Now()
	tmpl, err = tmpl.ParseFiles(templateFile)
	if err != nil {
		log.Printf("[LRM ServeHTTP] req=%d template parse error after=%s file=%s err=%v", reqID, time.Since(parseStart), templateFile, err)
		http.Error(w, fmt.Sprintf("Template parsing error: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("[LRM ServeHTTP] req=%d template parsed after=%s file=%s", reqID, time.Since(parseStart), templateFile)

	// Prepare template data
	templateData := struct {
		Data *lrm.LRMVerifierData
		CDN  map[string]string
	}{
		Data: lrmData,
		CDN:  GetCDNResources(h.config),
	}

	// Execute template
	execStart := time.Now()
	if err := tmpl.Execute(w, templateData); err != nil {
		log.Printf("[LRM ServeHTTP] req=%d template exec error after=%s err=%v", reqID, time.Since(execStart), err)
		http.Error(w, fmt.Sprintf("Template execution error: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("[LRM ServeHTTP] done req=%d total=%s (cache=%s, parse=%s, exec=%s)", reqID, time.Since(start), time.Since(cacheStart), time.Since(parseStart), time.Since(execStart))
}
