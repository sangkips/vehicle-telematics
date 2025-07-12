package main

import (
	"fleet-backend/internal/api/routes"
	"fleet-backend/internal/config"
	"fleet-backend/pkg/database"
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
	
	// Setup Gin router
	router := gin.Default()
	
	// CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.AllowedOrigins},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	
	// Setup routes
	routes.SetupRoutes(router, db)
	
	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	log.Fatal(router.Run(":" + cfg.Port))
}