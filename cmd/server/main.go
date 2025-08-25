package main

import (
	"fleet-backend/internal/api/routes"
	"fleet-backend/internal/config"
	"fleet-backend/pkg/database"
	"fleet-backend/pkg/redis"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)


func main() {
	// Load configuration
	cfg := config.Load()
	
	// Connect to MongoDB
	db, err := database.Connect(cfg.MongoURI)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Disconnect(db.Client())
	
	// Initialize Redis client
	redisClient := redis.NewClient(cfg.Redis)
	defer redisClient.Close()
	
	// Perform initial health check
	healthStatus := redisClient.HealthCheck()
	if healthStatus.IsConnected {
		log.Printf("Redis connected successfully at %s", healthStatus.ConnectionInfo)
	} else {
		log.Printf("Redis connection failed: %s (will retry automatically)", healthStatus.Error)
	}
	
	// Setup Gin router
	router := gin.Default()
	
	// CORS middleware
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Upgrade", "Connection", "Sec-WebSocket-Key", "Sec-WebSocket-Version", "Sec-WebSocket-Protocol"},
		ExposeHeaders:    []string{"Content-Length"},
	}
	
	// Handle wildcard origin for development
	if len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*" {
		corsConfig.AllowAllOrigins = true
		corsConfig.AllowCredentials = false // Cannot use credentials with AllowAllOrigins
	} else {
		corsConfig.AllowOrigins = cfg.AllowedOrigins
		corsConfig.AllowCredentials = true
	}
	
	router.Use(cors.New(corsConfig))
	
	// Setup routes
	routes.SetupRoutes(router, db, redisClient, cfg)
	
	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	log.Fatal(router.Run(":" + cfg.Port))
}