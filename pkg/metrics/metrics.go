package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP request metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being processed",
		},
	)

	// Custom business metrics
	vehiclesTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "fleet_vehicles_total",
			Help: "Total number of vehicles in the fleet",
		},
	)

	alertsTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "fleet_alerts_total",
			Help: "Total number of alerts by severity",
		},
		[]string{"severity"},
	)

	maintenanceRecordsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "fleet_maintenance_records_total",
			Help: "Total number of maintenance records",
		},
	)

	usersTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "fleet_users_total",
			Help: "Total number of registered users",
		},
	)
)

// PrometheusMiddleware returns a Gin middleware that instruments HTTP requests
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Increment in-flight requests
		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		// Record start time
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Get path template (to avoid high cardinality from path parameters)
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// Record metrics
		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method

		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		httpRequestDuration.WithLabelValues(method, path, status).Observe(duration)
	}
}

// UpdateVehiclesTotal updates the total number of vehicles metric
func UpdateVehiclesTotal(count float64) {
	vehiclesTotal.Set(count)
}

// UpdateAlertsTotal updates the total number of alerts by severity
func UpdateAlertsTotal(severity string, count float64) {
	alertsTotal.WithLabelValues(severity).Set(count)
}

// UpdateMaintenanceRecordsTotal updates the total number of maintenance records
func UpdateMaintenanceRecordsTotal(count float64) {
	maintenanceRecordsTotal.Set(count)
}

// UpdateUsersTotal updates the total number of users
func UpdateUsersTotal(count float64) {
	usersTotal.Set(count)
}
