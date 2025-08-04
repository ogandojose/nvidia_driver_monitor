package services

import (
	"nvidia_driver_monitor/internal/stats"
)

// StatisticsService provides access to application statistics
type StatisticsService struct {
	collector *stats.StatsCollector
}

// NewStatisticsService creates a new statistics service
func NewStatisticsService() *StatisticsService {
	return &StatisticsService{
		collector: stats.GetStatsCollector(),
	}
}

// GetAllWindowsStats returns all statistics window data
func (s *StatisticsService) GetAllWindowsStats() interface{} {
	return s.collector.GetAllWindowsStats()
}
