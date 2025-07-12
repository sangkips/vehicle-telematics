package routes

import (
	"fleet-backend/internal/api/handlers"
	"fleet-backend/internal/api/middleware"
	"fleet-backend/internal/repository"
	"fleet-backend/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)


func SetupRoutes(router *gin.Engine, db *mongo.Database) {
	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	vehicleRepo := repository.NewVehicleRepository(db)
	alertRepo := repository.NewAlertRepository(db)
	
	// Initialize services
	authService := services.NewAuthService(userRepo)
	userService := services.NewUserService(userRepo)
	vehicleService := services.NewVehicleService(vehicleRepo)
	alertService := services.NewAlertService(alertRepo)
	
	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	vehicleHandler := handlers.NewVehicleHandler(vehicleService)
	alertHandler := handlers.NewAlertHandler(alertService)
	
	// API routes
	api := router.Group("/api/v1")
	
	// Public routes
	auth := api.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/logout", authHandler.Logout)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.POST("/register", userHandler.CreateUser)
	}
	
	// Protected routes
	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		// Vehicles
		vehicles := protected.Group("/vehicles")
		{
			vehicles.GET("", vehicleHandler.GetVehicles)
			vehicles.POST("", vehicleHandler.CreateVehicle)
			vehicles.GET("/:id", vehicleHandler.GetVehicle)
			vehicles.PATCH("/:id", vehicleHandler.UpdateVehicle)
			vehicles.DELETE("/:id", vehicleHandler.DeleteVehicle)
			vehicles.GET("/updates", vehicleHandler.GetVehicleUpdates)
		}
		
		// Users
		users := protected.Group("/users")
		{
			users.GET("", userHandler.GetUsers)
			// users.POST("", userHandler.CreateUser)
			users.GET("/:id", userHandler.GetUser)
			users.PATCH("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", userHandler.DeleteUser)
		}
		
		// Alerts
		alerts := protected.Group("/alerts")
		{
			alerts.GET("", alertHandler.GetAlerts)
			alerts.PATCH("/:id/resolve", alertHandler.ResolveAlert)
			alerts.DELETE("/:id/dismiss", alertHandler.DismissAlert)
		}
	}
}