// Statistics Dashboard JavaScript
class StatisticsDashboard {
    constructor() {
        this.charts = {};
        this.lastUpdateTime = null;
        this.refreshInterval = null;
        this.init();
    }

    async init() {
        await this.loadStatistics();
        this.setupEventListeners();
        this.startAutoRefresh();
    }

    setupEventListeners() {
        document.getElementById('refresh-btn').addEventListener('click', () => {
            this.loadStatistics();
        });
    }

    startAutoRefresh() {
        // Refresh every 30 seconds
        this.refreshInterval = setInterval(() => {
            this.loadStatistics();
        }, 30000);
    }

    async loadStatistics() {
        try {
            const response = await fetch('/api/statistics');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            const data = await response.json();
            this.updateDashboard(data);
            this.updateLastUpdatedTime();
        } catch (error) {
            console.error('Error loading statistics:', error);
            this.showError('Failed to load statistics data');
        }
    }

    updateDashboard(data) {
        this.updateSummaryCards(data);
        this.updateCharts(data);
        this.updateDomainTable(data);
        this.updateHistoricalWindowsTable(data);
        this.updateMetadata(data);
    }

    updateSummaryCards(data) {
        const stats = data.current_window.stats;
        const domains = Object.keys(stats);
        
        let totalRequests = 0;
        let totalSuccessful = 0;
        let totalRetries = 0;
        let totalResponseTime = 0;
        let requestCount = 0;

        domains.forEach(domain => {
            const domainStats = stats[domain];
            totalRequests += domainStats.total_requests;
            totalSuccessful += domainStats.successful_reqs;
            totalRetries += domainStats.total_retries;
            totalResponseTime += domainStats.avg_response_ms * domainStats.total_requests;
            requestCount += domainStats.total_requests;
        });

        const successRate = totalRequests > 0 ? ((totalSuccessful / totalRequests) * 100).toFixed(1) : 0;
        const avgResponseTime = requestCount > 0 ? (totalResponseTime / requestCount).toFixed(1) : 0;

        document.getElementById('total-requests').textContent = totalRequests.toLocaleString();
        document.getElementById('success-rate').textContent = successRate + '%';
        document.getElementById('avg-response-time').textContent = avgResponseTime + ' ms';
        document.getElementById('total-retries').textContent = totalRetries.toLocaleString();
    }

    updateCharts(data) {
        const stats = data.current_window.stats;
        const domains = Object.keys(stats);

        // Response Time Chart
        this.updateResponseTimeChart(domains, stats);
        
        // Request Volume Chart
        this.updateRequestVolumeChart(domains, stats);
        
        // Success Rate Chart
        this.updateSuccessRateChart(domains, stats);
        
        // Retry Chart
        this.updateRetryChart(domains, stats);
        
        // Historical Chart
        this.updateHistoricalChart(data);
    }

    updateResponseTimeChart(domains, stats) {
        const ctx = document.getElementById('responseTimeChart').getContext('2d');
        
        const data = {
            labels: domains.map(d => this.formatDomainName(d)),
            datasets: [{
                label: 'Average Response Time (ms)',
                data: domains.map(d => stats[d].avg_response_ms.toFixed(1)),
                backgroundColor: [
                    'rgba(255, 99, 132, 0.8)',
                    'rgba(54, 162, 235, 0.8)',
                    'rgba(255, 205, 86, 0.8)',
                    'rgba(75, 192, 192, 0.8)',
                    'rgba(153, 102, 255, 0.8)',
                    'rgba(255, 159, 64, 0.8)'
                ],
                borderColor: [
                    'rgba(255, 99, 132, 1)',
                    'rgba(54, 162, 235, 1)',
                    'rgba(255, 205, 86, 1)',
                    'rgba(75, 192, 192, 1)',
                    'rgba(153, 102, 255, 1)',
                    'rgba(255, 159, 64, 1)'
                ],
                borderWidth: 2
            }]
        };

        if (this.charts.responseTime) {
            this.charts.responseTime.destroy();
        }

        this.charts.responseTime = new Chart(ctx, {
            type: 'bar',
            data: data,
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        title: {
                            display: true,
                            text: 'Response Time (ms)'
                        }
                    }
                }
            }
        });
    }

    updateRequestVolumeChart(domains, stats) {
        const ctx = document.getElementById('requestVolumeChart').getContext('2d');
        
        const data = {
            labels: domains.map(d => this.formatDomainName(d)),
            datasets: [{
                label: 'Total Requests',
                data: domains.map(d => stats[d].total_requests),
                backgroundColor: 'rgba(54, 162, 235, 0.8)',
                borderColor: 'rgba(54, 162, 235, 1)',
                borderWidth: 2
            }]
        };

        if (this.charts.requestVolume) {
            this.charts.requestVolume.destroy();
        }

        this.charts.requestVolume = new Chart(ctx, {
            type: 'doughnut',
            data: data,
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'bottom'
                    }
                }
            }
        });
    }

    updateSuccessRateChart(domains, stats) {
        const ctx = document.getElementById('successRateChart').getContext('2d');
        
        const successRates = domains.map(d => {
            const domainStats = stats[d];
            return domainStats.total_requests > 0 ? 
                ((domainStats.successful_reqs / domainStats.total_requests) * 100).toFixed(1) : 0;
        });

        const data = {
            labels: domains.map(d => this.formatDomainName(d)),
            datasets: [{
                label: 'Success Rate (%)',
                data: successRates,
                backgroundColor: successRates.map(rate => 
                    rate >= 95 ? 'rgba(75, 192, 192, 0.8)' : 
                    rate >= 80 ? 'rgba(255, 205, 86, 0.8)' : 
                    'rgba(255, 99, 132, 0.8)'
                ),
                borderColor: successRates.map(rate => 
                    rate >= 95 ? 'rgba(75, 192, 192, 1)' : 
                    rate >= 80 ? 'rgba(255, 205, 86, 1)' : 
                    'rgba(255, 99, 132, 1)'
                ),
                borderWidth: 2
            }]
        };

        if (this.charts.successRate) {
            this.charts.successRate.destroy();
        }

        this.charts.successRate = new Chart(ctx, {
            type: 'bar',
            data: data,
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        max: 100,
                        title: {
                            display: true,
                            text: 'Success Rate (%)'
                        }
                    }
                }
            }
        });
    }

    updateRetryChart(domains, stats) {
        const ctx = document.getElementById('retryChart').getContext('2d');
        
        const data = {
            labels: domains.map(d => this.formatDomainName(d)),
            datasets: [{
                label: 'Total Retries',
                data: domains.map(d => stats[d].total_retries),
                backgroundColor: domains.map(d => 
                    stats[d].total_retries > 10 ? 'rgba(255, 99, 132, 0.8)' :
                    stats[d].total_retries > 0 ? 'rgba(255, 205, 86, 0.8)' :
                    'rgba(75, 192, 192, 0.8)'
                ),
                borderColor: domains.map(d => 
                    stats[d].total_retries > 10 ? 'rgba(255, 99, 132, 1)' :
                    stats[d].total_retries > 0 ? 'rgba(255, 205, 86, 1)' :
                    'rgba(75, 192, 192, 1)'
                ),
                borderWidth: 2
            }]
        };

        if (this.charts.retry) {
            this.charts.retry.destroy();
        }

        this.charts.retry = new Chart(ctx, {
            type: 'bar',
            data: data,
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        title: {
                            display: true,
                            text: 'Retry Count'
                        }
                    }
                }
            }
        });
    }

    updateHistoricalChart(data) {
        const historicalWindows = data.historical_windows || [];
        
        if (historicalWindows.length === 0) {
            document.getElementById('historicalChart').style.display = 'none';
            document.getElementById('no-historical-data').style.display = 'block';
            return;
        }

        document.getElementById('historicalChart').style.display = 'block';
        document.getElementById('no-historical-data').style.display = 'none';

        const ctx = document.getElementById('historicalChart').getContext('2d');
        
        // Prepare time series data
        const allWindows = [data.current_window, ...historicalWindows].reverse();
        const timeLabels = allWindows.map(window => 
            new Date(window.start_time).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})
        );
        
        // Get all unique domains
        const allDomains = new Set();
        allWindows.forEach(window => {
            Object.keys(window.stats || {}).forEach(domain => allDomains.add(domain));
        });
        
        const domainColors = {
            'nvidia': 'rgba(76, 175, 80, 0.8)',
            'launchpad': 'rgba(33, 150, 243, 0.8)',
            'ubuntu-kernel': 'rgba(255, 152, 0, 0.8)',
            'default': 'rgba(156, 39, 176, 0.8)'
        };
        
        const datasets = Array.from(allDomains).map(domain => ({
            label: this.formatDomainName(domain),
            data: allWindows.map(window => {
                const domainStats = window.stats && window.stats[domain];
                return domainStats ? domainStats.avg_response_ms.toFixed(1) : 0;
            }),
            borderColor: domainColors[domain] || domainColors.default,
            backgroundColor: (domainColors[domain] || domainColors.default).replace('0.8', '0.2'),
            tension: 0.4,
            fill: false
        }));

        if (this.charts.historical) {
            this.charts.historical.destroy();
        }

        this.charts.historical = new Chart(ctx, {
            type: 'line',
            data: {
                labels: timeLabels,
                datasets: datasets
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'top'
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        title: {
                            display: true,
                            text: 'Average Response Time (ms)'
                        }
                    },
                    x: {
                        title: {
                            display: true,
                            text: 'Time Windows'
                        }
                    }
                }
            }
        });
    }

    updateDomainTable(data) {
        const stats = data.current_window.stats;
        const tbody = document.querySelector('#domain-stats-table tbody');
        tbody.innerHTML = '';

        Object.keys(stats).forEach(domain => {
            const domainStats = stats[domain];
            const successRate = domainStats.total_requests > 0 ? 
                ((domainStats.successful_reqs / domainStats.total_requests) * 100).toFixed(1) : 0;
            
            const status = this.getStatusBadge(successRate, domainStats.total_retries);
            
            const row = tbody.insertRow();
            row.innerHTML = `
                <td><strong>${this.formatDomainName(domain)}</strong></td>
                <td>${domainStats.total_requests.toLocaleString()}</td>
                <td>${successRate}%</td>
                <td>${domainStats.failed_reqs.toLocaleString()}</td>
                <td>${domainStats.total_retries.toLocaleString()}</td>
                <td>${domainStats.avg_response_ms.toFixed(1)} ms</td>
                <td>${status}</td>
            `;
        });
    }

    updateHistoricalWindowsTable(data) {
        const tbody = document.querySelector('#historical-windows-table tbody');
        const noDataMessage = document.getElementById('no-historical-windows');
        
        // Combine current window with historical windows for display
        const allWindows = [...data.historical_windows];
        
        // Clear existing content
        tbody.innerHTML = '';
        
        if (allWindows.length === 0) {
            noDataMessage.style.display = 'block';
            return;
        }
        
        noDataMessage.style.display = 'none';
        
        // Sort windows by start time (newest first)
        allWindows.sort((a, b) => new Date(b.start_time) - new Date(a.start_time));
        
        // Take only the last 10 windows
        const recentWindows = allWindows.slice(0, 10);
        
        recentWindows.forEach(window => {
            const stats = window.stats;
            const domains = Object.keys(stats);
            
            // Calculate aggregated stats for this window
            let totalRequests = 0;
            let totalSuccessful = 0;
            let totalFailed = 0;
            let totalRetries = 0;
            let totalResponseTime = 0;
            let requestCount = 0;
            
            domains.forEach(domain => {
                const domainStats = stats[domain];
                totalRequests += domainStats.total_requests;
                totalSuccessful += domainStats.successful_reqs;
                totalFailed += domainStats.failed_reqs;
                totalRetries += domainStats.total_retries;
                totalResponseTime += domainStats.avg_response_ms * domainStats.total_requests;
                requestCount += domainStats.total_requests;
            });
            
            const successRate = totalRequests > 0 ? ((totalSuccessful / totalRequests) * 100).toFixed(1) : 0;
            const avgResponseTime = requestCount > 0 ? (totalResponseTime / requestCount).toFixed(1) : 0;
            
            // Format time period
            const startTime = new Date(window.start_time);
            const endTime = new Date(window.end_time);
            const timePeriod = this.formatTimePeriod(startTime, endTime);
            const duration = this.formatDuration(startTime, endTime);
            
            const row = tbody.insertRow();
            row.innerHTML = `
                <td><strong>${timePeriod}</strong></td>
                <td>${totalRequests.toLocaleString()}</td>
                <td>${successRate}%</td>
                <td>${totalFailed.toLocaleString()}</td>
                <td>${totalRetries.toLocaleString()}</td>
                <td>${avgResponseTime} ms</td>
                <td>${domains.length}</td>
                <td>${duration}</td>
            `;
        });
    }

    formatTimePeriod(startTime, endTime) {
        const start = startTime.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
        const end = endTime.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
        const date = startTime.toLocaleDateString([], { month: 'short', day: 'numeric' });
        return `${date} ${start}-${end}`;
    }

    formatDuration(startTime, endTime) {
        const diffMs = endTime - startTime;
        const diffMins = Math.round(diffMs / (1000 * 60));
        return `${diffMins} min`;
    }

    updateMetadata(data) {
        document.getElementById('window-duration').textContent = data.window_duration_minutes + ' minutes';
        document.getElementById('max-windows').textContent = data.max_stored_windows;
    }

    updateLastUpdatedTime() {
        const now = new Date();
        document.getElementById('last-updated').textContent = 
            `Last Updated: ${now.toLocaleTimeString()}`;
    }

    formatDomainName(domain) {
        const domainMap = {
            'nvidia': 'NVIDIA',
            'launchpad': 'Launchpad',
            'ubuntu-kernel': 'Ubuntu Kernel',
            'developer.nvidia.com': 'NVIDIA Developer',
            'launchpad.net': 'Launchpad.net'
        };
        return domainMap[domain] || domain.charAt(0).toUpperCase() + domain.slice(1);
    }

    getStatusBadge(successRate, retries) {
        if (successRate >= 95 && retries <= 5) {
            return '<span class="status-badge status-healthy">Healthy</span>';
        } else if (successRate >= 80 || retries <= 20) {
            return '<span class="status-badge status-warning">Warning</span>';
        } else {
            return '<span class="status-badge status-error">Error</span>';
        }
    }

    showError(message) {
        console.error(message);
        // Update status to show error
        document.getElementById('server-status').textContent = 'Connection Error';
        document.querySelector('.indicator.active').style.background = '#f44336';
    }
}

// Initialize dashboard when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new StatisticsDashboard();
});
