package services

import (
	"context"
	"fmt"
	"log"

	"nvidia_driver_monitor/internal/config"
	"nvidia_driver_monitor/internal/domain/entities"
)

// WebService provides the main business logic for the web application
type WebService struct {
	lrmService        *LRMService
	statisticsService *StatisticsService
	config            *config.Config
	stopChan          chan bool
}

// WebServiceConfig holds configuration for the web service
type WebServiceConfig struct {
	Config       *config.Config
	TemplatePath string
	CacheDir     string
}

// NewWebService creates a new web service with all dependencies
func NewWebService(cfg *WebServiceConfig) (*WebService, error) {
	// Create LRM service with clean architecture
	lrmConfig := &LRMServiceConfig{
		CacheDir: cfg.CacheDir,
	}
	lrmService := NewLRMService(lrmConfig)

	// Create statistics service
	statisticsService := NewStatisticsService()

	ws := &WebService{
		lrmService:        lrmService,
		statisticsService: statisticsService,
		config:            cfg.Config,
		stopChan:          make(chan bool),
	}

	log.Printf("WebService: Initialized with clean architecture")
	return ws, nil
}

// GetConfig returns the configuration
func (ws *WebService) GetConfig() *config.Config {
	return ws.config
}

// GetLRMData returns LRM data using the new clean architecture
func (ws *WebService) GetLRMData(ctx context.Context, routing string) (*entities.LRMData, error) {
	return ws.lrmService.GetLRMData(ctx, routing)
}

// GetCachedLRMData returns cached LRM data
func (ws *WebService) GetCachedLRMData(ctx context.Context) (*entities.LRMData, error) {
	return ws.lrmService.GetCachedLRMData(ctx)
}

// RefreshLRMData forces refresh of LRM data
func (ws *WebService) RefreshLRMData(ctx context.Context, routing string) (*entities.LRMData, error) {
	return ws.lrmService.RefreshLRMData(ctx, routing)
}

// GetLRMStats returns LRM statistics
func (ws *WebService) GetLRMStats(ctx context.Context) (*LRMStats, error) {
	return ws.lrmService.GetLRMStats(ctx)
}

// FilterLRMData applies filters to LRM data
func (ws *WebService) FilterLRMData(ctx context.Context, criteria *entities.FilterCriteria) ([]entities.Kernel, error) {
	return ws.lrmService.FilterLRMData(ctx, criteria)
}

// ClearLRMCache clears the LRM cache
func (ws *WebService) ClearLRMCache(ctx context.Context) error {
	return ws.lrmService.ClearCache(ctx)
}

// Stop gracefully stops the web service
func (ws *WebService) Stop() {
	log.Printf("WebService: Stopping...")
	close(ws.stopChan)
}

// Initialize initializes the web service by setting up dependencies and initial data
func (ws *WebService) Initialize(ctx context.Context) error {
	log.Printf("WebService: Starting initialization...")

	// Initialize LRM service data
	if err := ws.lrmService.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize LRM service: %w", err)
	}

	log.Printf("WebService: Initialization completed successfully")
	return nil
}

// GetStatisticsData returns all statistics windows data
func (ws *WebService) GetStatisticsData() interface{} {
	return ws.statisticsService.GetAllWindowsStats()
}

// GetStatisticsService returns the statistics service
func (ws *WebService) GetStatisticsService() *StatisticsService {
	return ws.statisticsService
}
