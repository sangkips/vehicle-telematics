package handlers

import (
	"context"
	"fleet-backend/pkg/redis"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

type HealthHandler struct {
	db          *mongo.Database
	redisClient *redis.Client
}

type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Services  map[string]interface{} `json:"services"`
}

func NewHealthHandler(db *mongo.Database, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{
		db:          db,
		redisClient: redisClient,
	}
}

func (h *HealthHandler) HealthCheck(c *gin.Context) {
	response := HealthResponse{
		Timestamp: time.Now(),
		Services:  make(map[string]interface{}),
	}

	overallHealthy := true

	// Check MongoDB
	mongoStatus := h.checkMongoDB()
	response.Services["mongodb"] = mongoStatus
	if !mongoStatus["healthy"].(bool) {
		overallHealthy = false
	}

	// Check Redis
	redisStatus := h.checkRedis()
	response.Services["redis"] = redisStatus
	if !redisStatus["healthy"].(bool) {
		overallHealthy = false
	}

	if overallHealthy {
		response.Status = "healthy"
		c.JSON(http.StatusOK, response)
	} else {
		response.Status = "unhealthy"
		c.JSON(http.StatusServiceUnavailable, response)
	}
}

func (h *HealthHandler) checkMongoDB() map[string]interface{} {
	status := map[string]interface{}{
		"service": "mongodb",
		"healthy": false,
	}

	if h.db != nil {
		// Simple ping to check MongoDB connectivity
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		err := h.db.Client().Ping(ctx, nil)
		if err == nil {
			status["healthy"] = true
			status["message"] = "Connected"
		} else {
			status["error"] = err.Error()
		}
	} else {
		status["error"] = "Database client not initialized"
	}

	return status
}

func (h *HealthHandler) checkRedis() map[string]interface{} {
	status := map[string]interface{}{
		"service": "redis",
		"healthy": false,
	}

	if h.redisClient != nil {
		healthStatus := h.redisClient.HealthCheck()
		status["healthy"] = healthStatus.IsConnected
		status["connectionInfo"] = healthStatus.ConnectionInfo
		status["responseTime"] = healthStatus.ResponseTime.String()
		status["lastPing"] = healthStatus.LastPing

		if healthStatus.Error != "" {
			status["error"] = healthStatus.Error
		}

		// Add connection stats
		stats := h.redisClient.GetConnectionStats()
		status["connectionStats"] = stats
	} else {
		status["error"] = "Redis client not initialized"
	}

	return status
}