// Dashboard JavaScript
class LogPushDashboard {
    constructor() {
        this.charts = {};
        this.currentTimeRange = 24; // Default to 24 hours
        this.customDateRange = null;
        this.init();
    }

    async init() {
        console.log('🚀 Initializing LogPush Dashboard');
        this.setupEventListeners();
        this.initializeDatePickers();
        await this.loadDashboardData();
        this.startAutoRefresh();
    }

    initializeDatePickers() {
        // Initialize native HTML5 datetime-local inputs with default values
        const now = new Date();
        const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000);
        
        // Format dates for datetime-local input (YYYY-MM-DDTHH:mm)
        const formatForDatetimeLocal = (date) => {
            const year = date.getFullYear();
            const month = String(date.getMonth() + 1).padStart(2, '0');
            const day = String(date.getDate()).padStart(2, '0');
            const hours = String(date.getHours()).padStart(2, '0');
            const minutes = String(date.getMinutes()).padStart(2, '0');
            return `${year}-${month}-${day}T${hours}:${minutes}`;
        };

        document.getElementById('start-date').value = formatForDatetimeLocal(yesterday);
        document.getElementById('end-date').value = formatForDatetimeLocal(now);
    }

    setupEventListeners() {
        // Keep the old refresh button working for compatibility
        const oldRefreshBtn = document.getElementById('refresh-btn');
        if (oldRefreshBtn) {
            oldRefreshBtn.addEventListener('click', () => {
                this.loadDashboardData();
            });
        }

        // New navbar controls
        document.getElementById('nav-time-range').addEventListener('change', (e) => {
            this.handleTimeRangeChange(e.target.value);
        });

        document.getElementById('apply-custom').addEventListener('click', () => {
            this.applyCustomDateRange();
        });

        // Handle Enter key in date inputs
        document.getElementById('start-date').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.applyCustomDateRange();
        });

        document.getElementById('end-date').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.applyCustomDateRange();
        });

        // Keep old time-range selector working if it exists
        const oldTimeRange = document.getElementById('time-range');
        if (oldTimeRange) {
            oldTimeRange.addEventListener('change', (e) => {
                this.loadTimeSeriesData(parseInt(e.target.value));
            });
        }
    }

    formatDateForInput(date) {
        return date.toISOString().slice(0, 16); // Format: YYYY-MM-DDTHH:MM
    }

    handleTimeRangeChange(value) {
        if (value === 'custom') {
            document.getElementById('custom-range-controls').style.display = 'flex';
            this.updateChartTitle('Custom Range');
        } else {
            document.getElementById('custom-range-controls').style.display = 'none';
            this.currentTimeRange = parseInt(value);
            this.customDateRange = null;
            this.updateChartTitle(this.getTimeRangeLabel(parseInt(value)));
            // Load all data with the new time range
            this.loadDashboardData();
        }
    }

    getTimeRangeLabel(hours) {
        if (hours === 1) return 'Last Hour';
        if (hours === 6) return 'Last 6 Hours';
        if (hours === 24) return 'Last 24 Hours';
        if (hours === 168) return 'Last 7 Days';
        if (hours === 720) return 'Last 30 Days';
        return `Last ${hours} Hours`;
    }

    updateChartTitle(timeRangeLabel) {
        const chartTitle = document.getElementById('chart-title');
        if (chartTitle) {
            // For custom ranges that already include formatting, don't add extra parenthesis
            if (timeRangeLabel.includes(' - ')) {
                chartTitle.textContent = `📈 Ingestion Over Time [${timeRangeLabel}]`;
            } else {
                chartTitle.textContent = `📈 Ingestion Over Time (${timeRangeLabel})`;
            }
        }
    }

    applyCustomDateRange() {
        const startDateValue = document.getElementById('start-date').value;
        const endDateValue = document.getElementById('end-date').value;
        
        if (!startDateValue || !endDateValue) {
            this.showMessage('Please select both start and end dates', 'error');
            return;
        }
        
        const startDate = new Date(startDateValue);
        const endDate = new Date(endDateValue);
        
        if (startDate >= endDate) {
            this.showMessage('Start date must be before end date', 'error');
            return;
        }
        
        const now = new Date();
        if (endDate > now) {
            this.showMessage('End date cannot be in the future', 'error');
            return;
        }
        
        this.customDateRange = { start: startDate, end: endDate };
        this.currentTimeRange = null;
        
        // Format dates for display
        const formatDate = (date) => {
            return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});
        };
        
        const customRangeTitle = `${formatDate(startDate)} - ${formatDate(endDate)}`;
        this.updateChartTitle(customRangeTitle);
        
        // Load all data with the custom range
        this.loadDashboardData();
        this.showMessage('Custom date range applied', 'success');
    }

    async loadDashboardData() {
        console.log('📊 Loading dashboard data...');
        this.setLoadingState(true);
        
        try {
            // Set the correct chart title based on current state
            if (this.customDateRange) {
                const formatDate = (date) => {
                    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});
                };
                const customRangeTitle = `${formatDate(this.customDateRange.start)} - ${formatDate(this.customDateRange.end)}`;
                this.updateChartTitle(customRangeTitle);
            } else {
                this.updateChartTitle(this.getTimeRangeLabel(this.currentTimeRange));
            }
            
            await Promise.all([
                this.loadStats(),
                this.loadTimeSeriesData(),
                this.loadRecentLogs(),
                this.loadSizeBreakdown()
            ]);
            
            this.updateLastRefresh();
            this.showMessage('Dashboard updated successfully', 'success');
        } catch (error) {
            console.error('Error loading dashboard:', error);
            this.showMessage('Failed to load dashboard data', 'error');
        } finally {
            this.setLoadingState(false);
        }
    }

    async loadStats() {
        let url = '/api/stats/summary';
        
        // Add time range parameters if applicable
        if (this.customDateRange) {
            const params = new URLSearchParams({
                start: this.customDateRange.start.toISOString(),
                end: this.customDateRange.end.toISOString()
            });
            url += '?' + params.toString();
        } else if (this.currentTimeRange && this.currentTimeRange !== 24) {
            // Only add hours parameter if it's not the default 24 hours
            // When hours=0 or missing, API returns all data which is the default behavior
            url += `?hours=${this.currentTimeRange}`;
        }
        
        const response = await fetch(url);
        const result = await response.json();
        
        if (result.success) {
            this.updateStatsCards(result.data);
        } else {
            throw new Error(result.error);
        }
    }

    async loadTimeSeriesData(hours = null) {
        let url = '/api/charts/timeseries';
        
        // Use current state to determine parameters
        if (this.customDateRange) {
            // For custom date ranges, calculate hours and let the API filter
            const diffHours = Math.ceil((this.customDateRange.end - this.customDateRange.start) / (1000 * 60 * 60));
            const maxHours = Math.min(diffHours * 2, 8760); // Get a bit more data to ensure coverage
            url += `?hours=${maxHours}`;
        } else {
            const timeRange = hours || this.currentTimeRange || 24;
            url += `?hours=${timeRange}`;
        }
        
        const response = await fetch(url);
        const result = await response.json();
        
        if (result.success) {
            let data = result.data;
            
            // If using custom date range, filter the data client-side
            if (this.customDateRange) {
                data = result.data.filter(point => {
                    const pointDate = new Date(point.timestamp);
                    return pointDate >= this.customDateRange.start && pointDate <= this.customDateRange.end;
                });
            }
            
            this.updateTimeSeriesChart(data);
        } else {
            throw new Error(result.error);
        }
    }

    async loadRecentLogs() {
        let url = '/api/logs/recent';
        
        // Add time range parameters if applicable
        if (this.customDateRange) {
            const params = new URLSearchParams({
                start: this.customDateRange.start.toISOString(),
                end: this.customDateRange.end.toISOString()
            });
            url += '?' + params.toString();
        } else if (this.currentTimeRange) {
            url += `?hours=${this.currentTimeRange}`;
        }
        
        const response = await fetch(url);
        const result = await response.json();
        
        if (result.success) {
            this.updateLogsTable(result.data);
        } else {
            throw new Error(result.error);
        }
    }

    async loadSizeBreakdown() {
        let url = '/api/charts/breakdown';
        
        // Add time range parameters if applicable
        if (this.customDateRange) {
            const params = new URLSearchParams({
                start: this.customDateRange.start.toISOString(),
                end: this.customDateRange.end.toISOString()
            });
            url += '?' + params.toString();
        } else if (this.currentTimeRange && this.currentTimeRange !== 24) {
            // Only add hours parameter if it's not the default
            // When hours=0 or missing, API returns all data which is the default behavior
            url += `?hours=${this.currentTimeRange}`;
        }
        
        const response = await fetch(url);
        const result = await response.json();
        
        if (result.success) {
            this.updateSizeDistributionChart(result.data);
            this.updateBreakdownTable(result.data);
        } else {
            throw new Error(result.error);
        }
    }

    updateStatsCards(stats) {
        document.getElementById('total-records').textContent = stats.total_records?.toLocaleString() || '0';
        document.getElementById('total-size').textContent = this.formatBytes(stats.total_size || 0);
        document.getElementById('average-size').textContent = this.formatBytes(Math.round(stats.average_size || 0));
        document.getElementById('last-updated').textContent = stats.last_updated ? 
            new Date(stats.last_updated).toLocaleString() : 'Never';
    }

    updateTimeSeriesChart(data) {
        const ctx = document.getElementById('timeSeriesChart').getContext('2d');
        
        if (this.charts.timeSeries) {
            this.charts.timeSeries.destroy();
        }

        // Sort data by timestamp
        data.sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));

        this.charts.timeSeries = new Chart(ctx, {
            type: 'line',
            data: {
                labels: data.map(point => new Date(point.timestamp).toLocaleTimeString()),
                datasets: [{
                    label: 'Log Count',
                    data: data.map(point => point.count),
                    borderColor: '#667eea',
                    backgroundColor: 'rgba(102, 126, 234, 0.1)',
                    tension: 0.4,
                    fill: true
                }, {
                    label: 'Total Size (MB)',
                    data: data.map(point => point.total_size / (1024 * 1024)),
                    borderColor: '#764ba2',
                    backgroundColor: 'rgba(118, 75, 162, 0.1)',
                    tension: 0.4,
                    yAxisID: 'y1'
                }]
            },
            options: {
                responsive: true,
                plugins: {
                    legend: {
                        position: 'top',
                    }
                },
                scales: {
                    y: {
                        type: 'linear',
                        display: true,
                        position: 'left',
                        title: {
                            display: true,
                            text: 'Log Count'
                        }
                    },
                    y1: {
                        type: 'linear',
                        display: true,
                        position: 'right',
                        title: {
                            display: true,
                            text: 'Size (MB)'
                        },
                        grid: {
                            drawOnChartArea: false,
                        },
                    }
                }
            }
        });
    }

    updateSizeDistributionChart(data) {
        const ctx = document.getElementById('sizeDistributionChart').getContext('2d');
        
        if (this.charts.sizeDistribution) {
            this.charts.sizeDistribution.destroy();
        }

        this.charts.sizeDistribution = new Chart(ctx, {
            type: 'doughnut',
            data: {
                labels: data.map(item => item.range),
                datasets: [{
                    data: data.map(item => item.count),
                    backgroundColor: [
                        '#667eea',
                        '#764ba2',
                        '#f093fb',
                        '#f5576c',
                        '#4facfe',
                        '#43e97b'
                    ],
                    borderWidth: 2,
                    borderColor: '#fff'
                }]
            },
            options: {
                responsive: true,
                plugins: {
                    legend: {
                        position: 'bottom',
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                const percentage = data[context.dataIndex].percentage.toFixed(1);
                                return `${context.label}: ${context.parsed} (${percentage}%)`;
                            }
                        }
                    }
                }
            }
        });
    }

    updateLogsTable(logs) {
        const tbody = document.getElementById('logs-tbody');
        tbody.innerHTML = '';

        // Sort by timestamp descending (newest first)
        logs.sort((a, b) => new Date(b.Timestamp) - new Date(a.Timestamp));

        logs.slice(0, 50).forEach(log => { // Show only latest 50
            const row = tbody.insertRow();
            row.innerHTML = `
                <td>${log.ID}</td>
                <td>${new Date(log.Timestamp).toLocaleString()}</td>
                <td>${log.Filesize.toLocaleString()}</td>
                <td>${this.formatBytes(log.Filesize)}</td>
            `;
        });

        if (logs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="4" style="text-align: center; color: #666;">No logs found</td></tr>';
        }
    }

    updateBreakdownTable(data) {
        const tbody = document.getElementById('breakdown-tbody');
        tbody.innerHTML = '';

        data.forEach(item => {
            const row = tbody.insertRow();
            row.innerHTML = `
                <td><strong>${item.range}</strong></td>
                <td>${item.count.toLocaleString()}</td>
                <td>${item.percentage.toFixed(1)}%</td>
                <td>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: ${item.percentage}%"></div>
                    </div>
                </td>
            `;
        });
    }

    formatBytes(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    updateLastRefresh() {
        const now = new Date().toLocaleTimeString();
        // Update both old and new refresh indicators
        const oldRefresh = document.getElementById('last-refresh');
        const navRefresh = document.getElementById('nav-last-refresh');
        
        if (oldRefresh) oldRefresh.textContent = now;
        if (navRefresh) navRefresh.textContent = now;
    }

    setLoadingState(loading) {
        // Handle both old and new refresh buttons
        const refreshBtn = document.getElementById('refresh-btn');
        const navRefreshBtn = document.querySelector('#refresh-btn, .nav-btn');
        
        if (refreshBtn) {
            if (loading) {
                refreshBtn.classList.add('loading');
                refreshBtn.disabled = true;
            } else {
                refreshBtn.classList.remove('loading');
                refreshBtn.disabled = false;
            }
        }
        
        if (navRefreshBtn && navRefreshBtn !== refreshBtn) {
            if (loading) {
                navRefreshBtn.classList.add('loading');
                navRefreshBtn.disabled = true;
            } else {
                navRefreshBtn.classList.remove('loading');
                navRefreshBtn.disabled = false;
            }
        }
    }

    showMessage(message, type = 'info') {
        // Remove existing messages
        const existingMessages = document.querySelectorAll('.message');
        existingMessages.forEach(msg => msg.remove());

        // Create new message
        const messageDiv = document.createElement('div');
        messageDiv.className = `message ${type}`;
        messageDiv.textContent = message;
        
        // Insert after header
        const header = document.querySelector('header');
        header.insertAdjacentElement('afterend', messageDiv);

        // Auto-remove after 3 seconds
        setTimeout(() => {
            messageDiv.remove();
        }, 3000);
    }

    startAutoRefresh() {
        // Refresh every 30 seconds
        setInterval(() => {
            this.loadDashboardData();
        }, 30000);
        
        console.log('⏰ Auto-refresh enabled (30s interval)');
    }
}

// Initialize dashboard when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new LogPushDashboard();
});