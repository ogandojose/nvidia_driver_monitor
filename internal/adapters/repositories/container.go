package repositories

import (
	httpclient "nvidia_driver_monitor/internal/adapters/http"
	"nvidia_driver_monitor/internal/config"
)

// RepositoryContainer holds all repository implementations
type RepositoryContainer struct {
	KernelSeries *KernelSeriesRepository
	Package      *PackageRepository
	LRMCache     *LRMCacheRepository
}

// NewRepositoryContainer creates a new container with all repository implementations
func NewRepositoryContainer(cacheFilePath string, cfg *config.Config) *RepositoryContainer {
	// Create HTTP client with retry logic
	retryClient := httpclient.NewClient()

	// Create repository instances
	kernelRepo := NewKernelSeriesRepository(retryClient, cfg)
	packageRepo := NewPackageRepository(retryClient, cfg)
	cacheRepo := NewLRMCacheRepository(cacheFilePath)

	return &RepositoryContainer{
		KernelSeries: kernelRepo,
		Package:      packageRepo,
		LRMCache:     cacheRepo,
	}
}
