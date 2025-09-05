package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nvidia_driver_monitor/internal/lrm"
)

// restore the seam after each test
func withTryGetLRMData(fn func() (*lrm.LRMVerifierData, error), test func()) {
	old := tryGetLRMData
	tryGetLRMData = fn
	test()
	tryGetLRMData = old
}

func TestLRMDataHandler_PlaceholderWhenUnready(t *testing.T) {
	withTryGetLRMData(func() (*lrm.LRMVerifierData, error) {
		return nil, lrm.ErrCacheNotReady
	}, func() {
		h := NewAPIHandler()
		r := httptest.NewRequest(http.MethodGet, "/api/lrm", nil)
		w := httptest.NewRecorder()

		h.LRMDataHandler(w, r)

		res := w.Result()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}
		var payload APIResponse
		if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if payload.Data.IsInitialized {
			t.Fatalf("expected IsInitialized=false in placeholder")
		}
		if payload.Data.KernelResults == nil || len(payload.Data.KernelResults) != 0 {
			t.Fatalf("expected empty slice for KernelResults, got %v", payload.Data.KernelResults)
		}
		if payload.Meta.Total != 0 || payload.Meta.Filtered != 0 {
			t.Fatalf("expected meta counts 0, got %+v", payload.Meta)
		}
	})
}

func TestLRMDataHandler_HappyPath(t *testing.T) {
	fake := &lrm.LRMVerifierData{
		KernelResults: []lrm.KernelLRMResult{{Series: "jammy", Supported: true}},
		TotalKernels:  1,
		SupportedLRM:  1,
		LastUpdated:   time.Now(),
		IsInitialized: true,
	}
	withTryGetLRMData(func() (*lrm.LRMVerifierData, error) {
		return fake, nil
	}, func() {
		h := NewAPIHandler()
		r := httptest.NewRequest(http.MethodGet, "/api/lrm", nil)
		w := httptest.NewRecorder()

		h.LRMDataHandler(w, r)

		res := w.Result()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}
		var payload APIResponse
		if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !payload.Data.IsInitialized {
			t.Fatalf("expected IsInitialized=true")
		}
		if payload.Data.TotalKernels != 1 || payload.Data.SupportedLRM != 1 {
			t.Fatalf("unexpected totals: %+v", payload.Data)
		}
		if payload.Meta.Total != 1 || payload.Meta.Filtered != 1 {
			t.Fatalf("unexpected meta: %+v", payload.Meta)
		}
	})
}
