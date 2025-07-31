class StatisticsDashboard {
    constructor() {
        this.charts = {};
        this.isUpdating = false;
        this.autoRefreshEnabled = true;
        this.refreshInterval = 30000; // 30 seconds
        this.refreshTimer = null;
        
        this.initializeDashboard();
    }

    initializeDashboard() {
        // Wait for DOM to be fully loaded
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', () => this.init());
        } else {
            this.init();
        }
    }

    init() {
        this.setupEventListeners();
        this.initializeCharts();
        this.loadInitialData();
        this.startAutoRefresh();
    }

    setupEventListeners() {
        const refreshBtn = document.getElementById('refresh-btn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => this.refreshData());
        }
    }

    initializeCharts() {
        this.initializeHistoricalChart();
        this.initializeResponseTimeChart();
        this.initializeRequestVolumeChart();
        this.initializeSuccessRateChart();
        this.initializeRetryChart();
    }

    initializeHistoricalChart() {
        const ctx = document.getElementById('historicalChart');
        if (!ctx) return;

        // Fixed configuration to prevent jumping
        const config = {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'Average Response Time (ms)',
                    data: [],
                    borderColor: '#E95420',
                    backgroundColor: 'rgba(233, 84, 32, 0.1)',
                    borderWidth: 2,
                    fill: false,
                    tension: 0.1,
                    pointRadius: 4,
                    pointHoverRadius: 6
                }, {
                    label: 'Total Requests',
                    data: [],
                    borderColor: '#3182CE',
                    backgroundColor: 'rgba(49, 130, 206, 0.1)',
                    borderWidth: 2,
                    fill: false,
                    tension: 0.1,
                    pointRadius: 4,
                    pointHoverRadius: 6,
                    yAxisID: 'y1'
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                animation: false, // Disable all animations
                interaction: {
                    intersect: false,
                    mode: 'index'
                },
                plugins: {
                    legend: {
                        position: 'top',
                        display: true
                    },
                    tooltip: {
                        backgroundColor: 'rgba(0, 0, 0, 0.8)',
                        titleColor: '#ffffff',
                        bodyColor: '#ffffff',
                        borderColor: '#E95420',
                        borderWidth: 1
                    }
                },
                scales: {
                    x: {
                        display: true,
                        title: {
                            display: true,
                            text: 'Time Window'
                        },
                        grid: {
                            color: 'rgba(0, 0, 0, 0.1)'
                        }
                    },
                    y: {
                        type: 'linear',
                        display: true,
                        position: 'left',
                        title: {
                            display: true,
                            text: 'Response Time (ms)'
                        },
                        min: 0,
                        max: 5000, // Fixed max for response time in ms
                        grid: {
                            color: 'rgba(0, 0, 0, 0.1)'
                        }
                    },
                    y1: {
                        type: 'linear',
                        display: true,
                        position: 'right',
                        title: {
                            display: true,
                            text: 'Total Requests'
                        },
                        min: 0,
                        max: 2000, // Fixed max for requests
                        grid: {
                            drawOnChartArea: false
                        }
                    }
                }
            }
        };

        this.charts.historical = new Chart(ctx, config);
    }

    initializeResponseTimeChart() {
        const ctx = document.getElementById('responseTimeChart');
        if (!ctx) return;

        this.charts.responseTime = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: [],
                datasets: [{
                    label: 'Average Response Time (ms)',
                    data: [],
                    backgroundColor: 'rgba(233, 84, 32, 0.8)',
                    borderColor: '#E95420',
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                animation: false,
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

    initializeRequestVolumeChart() {
        const ctx = document.getElementById('requestVolumeChart');
        if (!ctx) return;

        this.charts.requestVolume = new Chart(ctx, {
            type: 'doughnut',
            data: {
                labels: [],
                datasets: [{
                    data: [],
                    backgroundColor: [
                        '#E95420', '#38A169', '#3182CE', '#805AD5',
                        '#D69E2E', '#E53E3E', '#38B2AC', '#DD6B20'
                    ],
                    borderWidth: 2,
                    borderColor: '#ffffff'
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                animation: false,
                plugins: {
                    legend: {
                        position: 'right'
                    }
                }
            }
        });
    }

    initializeSuccessRateChart() {
        const ctx = document.getElementById('successRateChart');
        if (!ctx) return;

        this.charts.successRate = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: [],
                datasets: [{
                    label: 'Success Rate (%)',
                    data: [],
                    backgroundColor: 'rgba(56, 161, 105, 0.8)',
                    borderColor: '#38A169',
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                animation: false,
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

    initializeRetryChart() {
        const ctx = document.getElementById('retryChart');
        if (!ctx) return;

        this.charts.retry = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'Total Retries',
                    data: [],
                    borderColor: '#D69E2E',
                    backgroundColor: 'rgba(214, 158, 46, 0.1)',
                    borderWidth: 2,
                    fill: true,
                    tension: 0.1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                animation: false,
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
                            text: 'Number of Retries'
                        }
                    }
                }
            }
        });
    }

    async loadInitialData() {
        try {
            await this.fetchAndUpdateData();
        } catch (error) {
            console.error('Failed to load initial data:', error);
            this.showErrorMessage('Failed to load dashboard data');
        }
    }

    async fetchAndUpdateData() {
        if (this.isUpdating) return;
        
        this.isUpdating = true;
        try {
            const response = await fetch('/api/statistics');
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            const data = await response.json();
            this.updateDashboard(data);
            this.updateLastUpdatedTime();
            
        } catch (error) {
            console.error('Error fetching statistics:', error);
            this.showErrorMessage('Failed to fetch statistics');
        } finally {
            this.isUpdating = false;
        }
    }

    updateDashboard(data) {
        this.updateSummaryCards(this.calculateSummaryFromCurrentWindow(data.current_window));
        this.updateCharts(data);
        this.updateDomainTable(this.extractDomainsFromCurrentWindow(data.current_window));
        this.updateHistoricalWindowsTable(data.historical_windows || []);
    }

    calculateSummaryFromCurrentWindow(currentWindow) {
        if (!currentWindow || !currentWindow.stats) {
            return {
                totalRequests: 0,
                successRate: 0,
                avgResponseTime: 0,
                totalRetries: 0
            };
        }

        let totalRequests = 0;
        let totalSuccessful = 0;
        let totalRetries = 0;
        let totalResponseTime = 0;
        let domainCount = 0;

        Object.values(currentWindow.stats).forEach(domainStats => {
            totalRequests += domainStats.total_requests || 0;
            totalSuccessful += domainStats.successful_reqs || 0;
            totalRetries += domainStats.total_retries || 0;
            totalResponseTime += domainStats.avg_response_ms || 0;
            domainCount++;
        });

        return {
            totalRequests,
            successRate: totalRequests > 0 ? (totalSuccessful / totalRequests * 100) : 0,
            avgResponseTime: domainCount > 0 ? (totalResponseTime / domainCount) : 0,
            totalRetries
        };
    }

    updateSummaryCards(summary) {
        const elements = {
            'total-requests': summary.totalRequests || 0,
            'success-rate': summary.successRate ? `${summary.successRate.toFixed(1)}%` : '0%',
            'avg-response-time': summary.avgResponseTime ? `${summary.avgResponseTime.toFixed(0)} ms` : '0 ms',
            'total-retries': summary.totalRetries || 0
        };

        Object.entries(elements).forEach(([id, value]) => {
            const element = document.getElementById(id);
            if (element) {
                element.textContent = value;
            }
        });
    }

    updateCharts(data) {
        this.updateHistoricalChart(data.historical_windows || []);
        this.updateResponseTimeChart(this.extractDomainsFromCurrentWindow(data.current_window));
        this.updateRequestVolumeChart(this.extractDomainsFromCurrentWindow(data.current_window));
        this.updateSuccessRateChart(this.extractDomainsFromCurrentWindow(data.current_window));
        this.updateRetryChart(data.historical_windows || []);
    }

    extractDomainsFromCurrentWindow(currentWindow) {
        if (!currentWindow || !currentWindow.stats) return [];
        
        return Object.values(currentWindow.stats).map(domainStats => ({
            domain: domainStats.domain,
            totalRequests: domainStats.total_requests || 0,
            successRate: domainStats.total_requests > 0 ? 
                (domainStats.successful_reqs / domainStats.total_requests * 100) : 0,
            failedRequests: domainStats.failed_reqs || 0,
            avgResponseTime: domainStats.avg_response_ms || 0,
            totalRetries: domainStats.total_retries || 0
        }));
    }

    updateHistoricalChart(historicalData) {
        if (!this.charts.historical || !historicalData.length) {
            this.showNoHistoricalData(true);
            return;
        }

        this.showNoHistoricalData(false);

        // Process historical windows data
        const processedData = historicalData.map(window => {
            // Aggregate data from all domains in this window
            let totalRequests = 0;
            let totalSuccessful = 0;
            let totalRetries = 0;
            let totalResponseTime = 0;
            let domainCount = 0;
            
            if (window.stats) {
                Object.values(window.stats).forEach(domainStats => {
                    totalRequests += domainStats.total_requests || 0;
                    totalSuccessful += domainStats.successful_reqs || 0;
                    totalRetries += domainStats.total_retries || 0;
                    totalResponseTime += domainStats.avg_response_ms || 0;
                    domainCount++;
                });
            }
            
            const successRate = totalRequests > 0 ? (totalSuccessful / totalRequests * 100) : 0;
            const avgResponseTime = domainCount > 0 ? (totalResponseTime / domainCount) : 0;
            
            return {
                timestamp: window.start_time || window.end_time,
                totalRequests,
                successRate,
                totalRetries,
                avgResponseTime
            };
        });

        // Update data without recreating the chart
        const chart = this.charts.historical;
        const labels = processedData.map(item => {
            const date = new Date(item.timestamp);
            return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
        });
        const responseTimeData = processedData.map(item => item.avgResponseTime || 0);
        const requestData = processedData.map(item => item.totalRequests || 0);

        // Update chart data - response time is now the primary dataset (index 0)
        chart.data.labels = labels;
        chart.data.datasets[0].data = responseTimeData;
        chart.data.datasets[1].data = requestData;

        // Adjust Y-axis max based on actual response time data, but keep it stable
        const maxResponseTime = Math.max(...responseTimeData);
        if (maxResponseTime > 0) {
            const newMax = Math.ceil(maxResponseTime * 1.2 / 1000) * 1000; // Round up to nearest 1000ms
            chart.options.scales.y.max = Math.max(newMax, 1000);
        }

        // Adjust secondary Y-axis for requests
        const maxRequests = Math.max(...requestData);
        if (maxRequests > 0) {
            const newMaxRequests = Math.ceil(maxRequests * 1.2 / 100) * 100; // Round up to nearest 100
            chart.options.scales.y1.max = Math.max(newMaxRequests, 100);
        }

        chart.update('none'); // Update without animation
    }

    updateResponseTimeChart(domains) {
        if (!this.charts.responseTime) return;

        const chart = this.charts.responseTime;
        chart.data.labels = domains.map(d => d.domain);
        chart.data.datasets[0].data = domains.map(d => d.avgResponseTime || 0);
        chart.update('none');
    }

    updateRequestVolumeChart(domains) {
        if (!this.charts.requestVolume) return;

        const chart = this.charts.requestVolume;
        chart.data.labels = domains.map(d => d.domain);
        chart.data.datasets[0].data = domains.map(d => d.totalRequests || 0);
        chart.update('none');
    }

    updateSuccessRateChart(domains) {
        if (!this.charts.successRate) return;

        const chart = this.charts.successRate;
        chart.data.labels = domains.map(d => d.domain);
        chart.data.datasets[0].data = domains.map(d => d.successRate || 0);
        chart.update('none');
    }

    updateRetryChart(historicalData) {
        if (!this.charts.retry) return;

        // Process historical data to extract retry information
        const processedData = historicalData.map(window => {
            let totalRetries = 0;
            
            if (window.stats) {
                Object.values(window.stats).forEach(domainStats => {
                    totalRetries += domainStats.total_retries || 0;
                });
            }
            
            return {
                timestamp: window.start_time || window.end_time,
                totalRetries
            };
        });

        const chart = this.charts.retry;
        const labels = processedData.map(item => {
            const date = new Date(item.timestamp);
            return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
        });
        const retryData = processedData.map(item => item.totalRetries || 0);

        chart.data.labels = labels;
        chart.data.datasets[0].data = retryData;
        chart.update('none');
    }

    updateDomainTable(domains) {
        const table = document.getElementById('domain-stats-table');
        if (!table) return;

        const tbody = table.querySelector('tbody') || table.createTBody();
        tbody.innerHTML = '';

        domains.forEach(domain => {
            const row = tbody.insertRow();
            // Match the HTML table column order: Domain, Total Requests, Success Rate, Failed Requests, Total Retries, Avg Response Time, Status
            const status = domain.failedRequests === 0 ? 
                '<span style="color: #38A169;">✓ Healthy</span>' : 
                '<span style="color: #E53E3E;">⚠ Issues</span>';
            
            row.innerHTML = `
                <td>${domain.domain}</td>
                <td>${domain.totalRequests || 0}</td>
                <td>${domain.successRate ? domain.successRate.toFixed(1) + '%' : '0%'}</td>
                <td>${domain.failedRequests || 0}</td>
                <td>${domain.totalRetries || 0}</td>
                <td>${domain.avgResponseTime ? domain.avgResponseTime.toFixed(0) + ' ms' : '0 ms'}</td>
                <td>${status}</td>
            `;
        });
    }

    updateHistoricalWindowsTable(historicalWindows) {
        const table = document.getElementById('historical-windows-table');
        const noDataEl = document.getElementById('no-historical-windows');
        
        if (!table) return;

        const tbody = table.querySelector('tbody') || table.createTBody();
        tbody.innerHTML = '';

        if (!historicalWindows.length) {
            if (noDataEl) {
                noDataEl.style.display = 'block';
                table.style.display = 'none';
            }
            return;
        }

        if (noDataEl) {
            noDataEl.style.display = 'none';
            table.style.display = 'table';
        }

        // Show last 100 windows (most recent first)
        const recentWindows = historicalWindows.slice(-100).reverse();

        recentWindows.forEach(window => {
            // Aggregate data from all domains in this window
            let totalRequests = 0;
            let totalSuccessful = 0;
            let totalFailed = 0;
            let totalRetries = 0;
            let totalResponseTime = 0;
            let domainCount = 0;

            if (window.stats) {
                Object.values(window.stats).forEach(domainStats => {
                    totalRequests += domainStats.total_requests || 0;
                    totalSuccessful += domainStats.successful_reqs || 0;
                    totalFailed += domainStats.failed_reqs || 0;
                    totalRetries += domainStats.total_retries || 0;
                    totalResponseTime += domainStats.avg_response_ms || 0;
                    domainCount++;
                });
            }

            const successRate = totalRequests > 0 ? (totalSuccessful / totalRequests * 100) : 0;
            const avgResponseTime = domainCount > 0 ? (totalResponseTime / domainCount) : 0;

            // Format the window period
            const startTime = new Date(window.start_time);
            const endTime = new Date(window.end_time);
            const windowPeriod = `${startTime.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })} - ${endTime.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
            
            // Calculate duration
            const durationMs = endTime.getTime() - startTime.getTime();
            const durationMinutes = Math.round(durationMs / (1000 * 60));

            const row = tbody.insertRow();
            // Columns: Window Period, Total Requests, Success Rate, Failed Requests, Total Retries, Avg Response Time, Domains Active, Duration
            row.innerHTML = `
                <td>${windowPeriod}</td>
                <td>${totalRequests}</td>
                <td>${successRate.toFixed(1)}%</td>
                <td>${totalFailed}</td>
                <td>${totalRetries}</td>
                <td>${avgResponseTime.toFixed(0)} ms</td>
                <td>${domainCount}</td>
                <td>${durationMinutes} min</td>
            `;
        });
    }

    showNoHistoricalData(show) {
        const noDataEl = document.getElementById('no-historical-data');
        const chartEl = document.getElementById('historicalChart');
        
        if (noDataEl && chartEl) {
            if (show) {
                noDataEl.style.display = 'block';
                chartEl.style.display = 'none';
            } else {
                noDataEl.style.display = 'none';
                chartEl.style.display = 'block';
            }
        }
    }

    showErrorMessage(message) {
        console.error(message);
        // Could add a toast notification here
    }

    updateLastUpdatedTime() {
        const element = document.getElementById('last-updated');
        if (element) {
            element.textContent = `Last Updated: ${new Date().toLocaleTimeString()}`;
        }
    }

    refreshData() {
        this.fetchAndUpdateData();
    }

    startAutoRefresh() {
        if (this.autoRefreshEnabled) {
            this.refreshTimer = setInterval(() => {
                this.fetchAndUpdateData();
            }, this.refreshInterval);
        }
    }

    stopAutoRefresh() {
        if (this.refreshTimer) {
            clearInterval(this.refreshTimer);
            this.refreshTimer = null;
        }
    }
}

// Initialize dashboard when the page loads
const dashboard = new StatisticsDashboard();
