package web

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"nvidia_driver_monitor/internal/lrm"
)

// LRMHandler handles the L-R-M verifier page
type LRMHandler struct {
	templatePath       string
	supportedReleases  interface{} // TODO: Define proper type
}

// NewLRMHandler creates a new LRM handler
func NewLRMHandler(templatePath string) *LRMHandler {
	return &LRMHandler{
		templatePath: templatePath,
	}
}

// ServeHTTP handles requests for L-R-M verifier information
func (h *LRMHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	var lrmData *lrm.LRMVerifierData
	if realData, fetchErr := lrm.FetchKernelLRMData("ubuntu/4"); fetchErr != nil {
		// Fallback to generating data from supported releases if available
		lrmData = &lrm.LRMVerifierData{
			KernelResults: []lrm.KernelLRMResult{},
			TotalKernels:  0,
			SupportedLRM:  0,
			IsInitialized: false,
		}
	} else {
		lrmData = realData
	}

	// Load and parse template
	templateFile := filepath.Join(h.templatePath, "lrm_verifier.html")
	tmpl := template.New("lrm_verifier.html").Funcs(TemplateFunctions())
	
	var err error
	tmpl, err = tmpl.ParseFiles(templateFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Template parsing error: %v", err), http.StatusInternalServerError)
		return
	}

	// Prepare template data
	templateData := struct {
		Data *lrm.LRMVerifierData
	}{
		Data: lrmData,
	}

	// Execute template
	if err := tmpl.Execute(w, templateData); err != nil {
		http.Error(w, fmt.Sprintf("Template execution error: %v", err), http.StatusInternalServerError)
		return
	}
}
