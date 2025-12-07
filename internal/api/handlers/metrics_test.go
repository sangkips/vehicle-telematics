package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMetricsHandler(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test router
	router := gin.New()

	// Initialize metrics handler
	metricsHandler := NewMetricsHandler()

	// Register the metrics endpoint
	router.GET("/metrics", metricsHandler.Handler())

	// Make a request to the metrics endpoint
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify response contains Prometheus metrics
	body := w.Body.String()
	assert.Contains(t, body, "# HELP")
	assert.Contains(t, body, "# TYPE")

	// Verify some expected metrics are present
	assert.Contains(t, body, "go_goroutines")
}
