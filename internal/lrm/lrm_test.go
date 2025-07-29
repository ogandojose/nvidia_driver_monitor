package lrm

import (
	"testing"
	"time"
)

func TestLRMVerifierDataInitialization(t *testing.T) {
	data := &LRMVerifierData{
		KernelResults: []KernelLRMResult{},
		TotalKernels:  0,
		SupportedLRM:  0,
		LastUpdated:   time.Now(),
		IsInitialized: true,
	}

	if !data.IsInitialized {
		t.Error("Expected IsInitialized to be true")
	}

	if data.TotalKernels != 0 {
		t.Errorf("Expected TotalKernels to be 0, got %d", data.TotalKernels)
	}
}

func TestKernelLRMResult(t *testing.T) {
	result := KernelLRMResult{
		Series:    "22.04",
		Codename:  "jammy",
		Source:    "linux",
		Routing:   "ubuntu/4",
		Supported: true,
		LTS:       true,
		HasLRM:    true,
	}

	if result.Series != "22.04" {
		t.Errorf("Expected Series to be '22.04', got '%s'", result.Series)
	}

	if !result.Supported {
		t.Error("Expected Supported to be true")
	}

	if !result.LTS {
		t.Error("Expected LTS to be true")
	}
}

func TestNvidiaDriverStatus(t *testing.T) {
	status := NvidiaDriverStatus{
		DriverName:  "nvidia-graphics-drivers-535",
		DSCVersion:  "535.171.04-0ubuntu0.22.04.1",
		DKMSVersion: "535.171.04",
		Status:      "✅ Up to date",
		FullString:  "nvidia-graphics-drivers-535=535.171.04-0ubuntu0.22.04.1",
	}

	if status.DriverName != "nvidia-graphics-drivers-535" {
		t.Errorf("Expected DriverName to be 'nvidia-graphics-drivers-535', got '%s'", status.DriverName)
	}

	if status.Status != "✅ Up to date" {
		t.Errorf("Expected Status to be '✅ Up to date', got '%s'", status.Status)
	}
}

func TestSimplifyNvidiaDriverName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"nvidia-graphics-drivers-535=535.171.04-0ubuntu0.22.04.1", "535=535.171.04-0ubuntu0.22.04.1"},
		{"nvidia-graphics-drivers-470-server=470.256.02-0ubuntu0.22.04.1", "470-server=470.256.02-0ubuntu0.22.04.1"},
		{"nvidia-graphics-drivers-390=390.157-0ubuntu0.22.04.2", "390=390.157-0ubuntu0.22.04.2"},
		{"other-driver=1.0.0", "other-driver=1.0.0"},
		{"nvidia-graphics-drivers-535", "nvidia-graphics-drivers-535"}, // No equals sign
	}

	for _, test := range tests {
		result := SimplifyNvidiaDriverName(test.input)
		if result != test.expected {
			t.Errorf("SimplifyNvidiaDriverName(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}
