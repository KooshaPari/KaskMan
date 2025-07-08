package monitoring

import (
	"runtime"
	"sync"
	"time"

	"github.com/kooshapari/kaskmanager-rd-platform/internal/config"
	"github.com/sirupsen/logrus"
)

// Monitor provides system monitoring and metrics collection
type Monitor struct {
	config  *config.MonitorConfig
	logger  *logrus.Logger
	metrics *Metrics
	mutex   sync.RWMutex
	running bool
}

// Metrics holds system metrics
type Metrics struct {
	System      *SystemMetrics      `json:"system"`
	Performance *PerformanceMetrics `json:"performance"`
	Application *ApplicationMetrics `json:"application"`
	Timestamp   time.Time           `json:"timestamp"`
}

// SystemMetrics holds system-level metrics
type SystemMetrics struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	LoadAvg     float64 `json:"load_avg"`
	Uptime      float64 `json:"uptime"`
	GoRoutines  int     `json:"goroutines"`
}

// PerformanceMetrics holds performance metrics
type PerformanceMetrics struct {
	ResponseTime   float64 `json:"response_time_ms"`
	Throughput     float64 `json:"throughput_rps"`
	ErrorRate      float64 `json:"error_rate"`
	ActiveRequests int     `json:"active_requests"`
	TotalRequests  int64   `json:"total_requests"`
	FailedRequests int64   `json:"failed_requests"`
}

// ApplicationMetrics holds application-specific metrics
type ApplicationMetrics struct {
	ActiveProjects int     `json:"active_projects"`
	PendingTasks   int     `json:"pending_tasks"`
	ActiveAgents   int     `json:"active_agents"`
	SuccessRate    float64 `json:"success_rate"`
	DatabaseConns  int     `json:"database_connections"`
	CacheHitRate   float64 `json:"cache_hit_rate"`
}

// NewMonitor creates a new monitoring instance
func NewMonitor(cfg config.MonitorConfig, logger *logrus.Logger) *Monitor {
	return &Monitor{
		config: &cfg,
		logger: logger,
		metrics: &Metrics{
			System:      &SystemMetrics{},
			Performance: &PerformanceMetrics{},
			Application: &ApplicationMetrics{},
			Timestamp:   time.Now(),
		},
	}
}

// Start starts the monitoring system
func (m *Monitor) Start() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.running {
		return
	}

	m.logger.Info("Starting monitoring system")

	// Start metrics collection
	go m.collectMetrics()

	m.running = true
	m.logger.Info("Monitoring system started")
}

// Stop stops the monitoring system
func (m *Monitor) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.running {
		return
	}

	m.logger.Info("Stopping monitoring system")
	m.running = false
	m.logger.Info("Monitoring system stopped")
}

// IsRunning returns whether the monitor is running
func (m *Monitor) IsRunning() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.running
}

// GetMetrics returns current system metrics
func (m *Monitor) GetMetrics() *Metrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy to avoid race conditions
	return &Metrics{
		System: &SystemMetrics{
			CPUUsage:    m.metrics.System.CPUUsage,
			MemoryUsage: m.metrics.System.MemoryUsage,
			DiskUsage:   m.metrics.System.DiskUsage,
			LoadAvg:     m.metrics.System.LoadAvg,
			Uptime:      m.metrics.System.Uptime,
			GoRoutines:  m.metrics.System.GoRoutines,
		},
		Performance: &PerformanceMetrics{
			ResponseTime:   m.metrics.Performance.ResponseTime,
			Throughput:     m.metrics.Performance.Throughput,
			ErrorRate:      m.metrics.Performance.ErrorRate,
			ActiveRequests: m.metrics.Performance.ActiveRequests,
			TotalRequests:  m.metrics.Performance.TotalRequests,
			FailedRequests: m.metrics.Performance.FailedRequests,
		},
		Application: &ApplicationMetrics{
			ActiveProjects: m.metrics.Application.ActiveProjects,
			PendingTasks:   m.metrics.Application.PendingTasks,
			ActiveAgents:   m.metrics.Application.ActiveAgents,
			SuccessRate:    m.metrics.Application.SuccessRate,
			DatabaseConns:  m.metrics.Application.DatabaseConns,
			CacheHitRate:   m.metrics.Application.CacheHitRate,
		},
		Timestamp: m.metrics.Timestamp,
	}
}

// GetSystemStatus returns system health status
func (m *Monitor) GetSystemStatus() map[string]interface{} {
	metrics := m.GetMetrics()

	status := "healthy"
	if metrics.System.CPUUsage > 80 || metrics.System.MemoryUsage > 80 {
		status = "warning"
	}
	if metrics.System.CPUUsage > 95 || metrics.System.MemoryUsage > 95 {
		status = "critical"
	}

	return map[string]interface{}{
		"status":        status,
		"timestamp":     metrics.Timestamp,
		"uptime":        metrics.System.Uptime,
		"cpu_usage":     metrics.System.CPUUsage,
		"memory_usage":  metrics.System.MemoryUsage,
		"goroutines":    metrics.System.GoRoutines,
		"response_time": metrics.Performance.ResponseTime,
		"throughput":    metrics.Performance.Throughput,
		"error_rate":    metrics.Performance.ErrorRate,
	}
}

// GetHealth returns health check information
func (m *Monitor) GetHealth() map[string]interface{} {
	return map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now(),
		"version":   "1.0.0",
		"uptime":    m.metrics.System.Uptime,
	}
}

// RecordRequest records an HTTP request for metrics
func (m *Monitor) RecordRequest(duration time.Duration, success bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.metrics.Performance.TotalRequests++
	if !success {
		m.metrics.Performance.FailedRequests++
	}

	// Update response time (simple moving average)
	responseTimeMs := float64(duration.Nanoseconds()) / 1e6
	m.metrics.Performance.ResponseTime = (m.metrics.Performance.ResponseTime + responseTimeMs) / 2

	// Update error rate
	if m.metrics.Performance.TotalRequests > 0 {
		m.metrics.Performance.ErrorRate = float64(m.metrics.Performance.FailedRequests) / float64(m.metrics.Performance.TotalRequests) * 100
	}
}

// IncrementActiveRequests increments the active request counter
func (m *Monitor) IncrementActiveRequests() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.metrics.Performance.ActiveRequests++
}

// DecrementActiveRequests decrements the active request counter
func (m *Monitor) DecrementActiveRequests() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.metrics.Performance.ActiveRequests--
	if m.metrics.Performance.ActiveRequests < 0 {
		m.metrics.Performance.ActiveRequests = 0
	}
}

// UpdateApplicationMetrics updates application-specific metrics
func (m *Monitor) UpdateApplicationMetrics(activeProjects, pendingTasks, activeAgents int, successRate float64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.metrics.Application.ActiveProjects = activeProjects
	m.metrics.Application.PendingTasks = pendingTasks
	m.metrics.Application.ActiveAgents = activeAgents
	m.metrics.Application.SuccessRate = successRate
}

// collectMetrics periodically collects system metrics
func (m *Monitor) collectMetrics() {
	ticker := time.NewTicker(m.config.CollectionInterval)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-ticker.C:
			if !m.running {
				return
			}

			m.updateSystemMetrics(startTime)
		}
	}
}

// updateSystemMetrics updates system-level metrics
func (m *Monitor) updateSystemMetrics(startTime time.Time) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Update system metrics
	m.metrics.System.GoRoutines = runtime.NumGoroutine()
	m.metrics.System.MemoryUsage = float64(memStats.Sys) / (1024 * 1024) // MB
	m.metrics.System.Uptime = time.Since(startTime).Seconds()

	// Simulate other metrics (in a real implementation, these would come from the OS)
	m.metrics.System.CPUUsage = float64(runtime.NumGoroutine()) / 100 * 10 // Rough approximation
	if m.metrics.System.CPUUsage > 100 {
		m.metrics.System.CPUUsage = 100
	}

	m.metrics.System.DiskUsage = 45.5 // Placeholder
	m.metrics.System.LoadAvg = 1.2    // Placeholder

	// Update timestamp
	m.metrics.Timestamp = time.Now()

	// Calculate throughput (requests per second)
	uptime := m.metrics.System.Uptime
	if uptime > 0 {
		m.metrics.Performance.Throughput = float64(m.metrics.Performance.TotalRequests) / uptime
	}
}
