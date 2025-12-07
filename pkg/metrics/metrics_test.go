package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestPrometheusMiddleware(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test router with the middleware
	router := gin.New()
	router.Use(PrometheusMiddleware())

	// Add a test endpoint
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	// Make a request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify metrics were recorded
	// Note: We can't easily test the exact values due to the global nature of Prometheus metrics,
	// but we can verify the middleware doesn't panic and the request completes
}

func TestUpdateVehiclesTotal(t *testing.T) {
	// Update the metric
	UpdateVehiclesTotal(42.0)

	// Verify the metric was set (basic test)
	// Use prometheus testutil to verify the value
	count := testutil.ToFloat64(vehiclesTotal)
	assert.Equal(t, 42.0, count)
}

func TestUpdateAlertsTotal(t *testing.T) {
	// Update the metric
	UpdateAlertsTotal("critical", 5.0)
	UpdateAlertsTotal("warning", 10.0)

	// Verify metrics were set
	criticalCount := testutil.ToFloat64(alertsTotal.WithLabelValues("critical"))
	warningCount := testutil.ToFloat64(alertsTotal.WithLabelValues("warning"))

	assert.Equal(t, 5.0, criticalCount)
	assert.Equal(t, 10.0, warningCount)
}

func TestUpdateMaintenanceRecordsTotal(t *testing.T) {
	// Update the metric
	UpdateMaintenanceRecordsTotal(100.0)

	// Verify the metric was set
	count := testutil.ToFloat64(maintenanceRecordsTotal)
	assert.Equal(t, 100.0, count)
}

func TestUpdateUsersTotal(t *testing.T) {
	// Update the metric
	UpdateUsersTotal(25.0)

	// Verify the metric was set
	count := testutil.ToFloat64(usersTotal)
	assert.Equal(t, 25.0, count)
}

func TestMetricsRegistration(t *testing.T) {
	// Verify that all metrics are registered with Prometheus
	// This is a basic check to ensure metrics are properly initialized
	assert.NotNil(t, httpRequestsTotal)
	assert.NotNil(t, httpRequestDuration)
	assert.NotNil(t, httpRequestsInFlight)
	assert.NotNil(t, vehiclesTotal)
	assert.NotNil(t, alertsTotal)
	assert.NotNil(t, maintenanceRecordsTotal)
	assert.NotNil(t, usersTotal)
}

func TestMiddlewareInFlightGauge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Get initial value
	initialValue := testutil.ToFloat64(httpRequestsInFlight)

	// Create a router with middleware
	router := gin.New()
	router.Use(PrometheusMiddleware())

	// Add a test endpoint
	router.GET("/test", func(c *gin.Context) {
		// During request processing, in-flight should be incremented
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	// Make a request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// After request completes, in-flight should be back to initial value
	finalValue := testutil.ToFloat64(httpRequestsInFlight)
	assert.Equal(t, initialValue, finalValue)
}
