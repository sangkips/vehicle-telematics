package routes

import (
	"fleet-backend/internal/api/handlers"
	"fleet-backend/internal/api/middleware"
	"fleet-backend/internal/config"
	"fleet-backend/internal/repository"
	"fleet-backend/internal/services"
	"fleet-backend/internal/websocket"
	"fleet-backend/pkg/batch"
	"fleet-backend/pkg/ratelimit"
	"fleet-backend/pkg/redis"
	"fleet-backend/pkg/telemetry"
	"log"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)


func SetupRoutes(router *gin.Engine, db *mongo.Database, redisClient *redis.Client, cfg *config.Config) {
	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	vehicleRepo := repository.NewVehicleRepository(db)
	alertRepo := repository.NewAlertRepository(db)
	maintenanceRepo := repository.NewMaintenanceRepository(db)
	
	// Initialize services
	authService := services.NewAuthService(userRepo)
	userService := services.NewUserService(userRepo)
	vehicleService := services.NewVehicleService(vehicleRepo)
	alertService := services.NewAlertService(alertRepo)
	maintenanceService := services.NewMaintenanceService(maintenanceRepo, vehicleRepo)
	
	// Initialize WebSocket manager
	wsManager := websocket.NewManager()
	wsManager.Start()
	
	// Initialize batch processor
	batchConfig := batch.LoadBatchConfigFromEnv()
	batchRepo := batch.NewVehicleRepositoryAdapter(vehicleRepo, db)
	batchProcessor := batch.NewBatchProcessorWithWebSocket(batchConfig, batchRepo, wsManager)
	
	// Initialize optimized telemetry service
	telemetryService := telemetry.NewOptimizedTelemetryService(vehicleService, batchProcessor)
	
	// Start telemetry service
	if err := telemetryService.Start(); err != nil {
		log.Printf("Warning: Failed to start telemetry service: %v", err)
	} else {
		log.Println("Optimized telemetry service started successfully")
	}
	
	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	vehicleHandler := handlers.NewVehicleHandler(vehicleService)
	alertHandler := handlers.NewAlertHandler(alertService)
	maintenanceHandler := handlers.NewMaintenanceHandler(maintenanceService)
	healthHandler := handlers.NewHealthHandler(db, redisClient)
	wsHandler := handlers.NewWebSocketHandler(wsManager)
	
	// Initialize vehicle WebSocket handler (for testing)
	// vehicleWSHandler := handlers.NewVehicleWebSocketHandler(wsManager, nil)
	
	// Initialize rate limiter
	rateLimitConfig := &ratelimit.Config{
		DefaultLimits:   ratelimit.DefaultConfig().DefaultLimits,
		RedisKeyPrefix:  cfg.RateLimit.RedisKeyPrefix,
		CleanupInterval: cfg.RateLimit.CleanupInterval,
		Enabled:         cfg.RateLimit.Enabled,
	}

	var rateLimiter ratelimit.RateLimiter
	if cfg.RedisEnabled && redisClient != nil {
		redisLimiter := ratelimit.NewRedisRateLimiter(redisClient.GetClient(), rateLimitConfig)
		// Load existing custom limits
		redisLimiter.LoadCustomLimits()
		rateLimiter = redisLimiter
	} else {
		rateLimiter = ratelimit.NewMemoryRateLimiter(rateLimitConfig)
		log.Println("Using in-memory rate limiter (Redis is disabled)")
	}
	
	// API routes with rate limiting
	api := router.Group("/api/v1")
	api.Use(middleware.RateLimitMiddleware(rateLimiter))
	
	// Health check endpoint (public)
	api.GET("/health", healthHandler.HealthCheck)
	
	// Public routes
	auth := api.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/logout", authHandler.Logout)
		auth.POST("/refresh", authHandler.RefreshTokenPublic)
		auth.POST("/register", userHandler.CreateUser)
	}
	

	
	// Protected auth routes
	authProtected := api.Group("/auth")
	authProtected.Use(middleware.AuthMiddleware())
	{
		authProtected.GET("/profile", authHandler.GetProfile)
		authProtected.POST("/change-password", authHandler.ChangePassword)
		authProtected.POST("/refresh-secure", authHandler.RefreshToken)
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
			alerts.POST("", alertHandler.CreateAlert)
			alerts.GET("/:id", alertHandler.GetAlert)
			alerts.PATCH("/:id", alertHandler.UpdateAlert)
			alerts.PATCH("/:id/resolve", alertHandler.ResolveAlert)
			alerts.DELETE("/:id/dismiss", alertHandler.DismissAlert)
			alerts.GET("/vehicle/:vehicleId", alertHandler.GetAlertsByVehicle)
			alerts.GET("/type", alertHandler.GetAlertsByType)
			alerts.GET("/severity", alertHandler.GetAlertsBySeverity)
			alerts.GET("/unresolved", alertHandler.GetUnresolvedAlerts)
			alerts.GET("/statistics", alertHandler.GetAlertStatistics)
			alerts.PATCH("/vehicle/:vehicleId/resolve", alertHandler.ResolveAlertsByVehicle)
			alerts.PATCH("/type/resolve", alertHandler.ResolveAlertsByType)
		}
		
		// Maintenance
		maintenance := protected.Group("/maintenance")
		{
			// Maintenance Records
			maintenance.POST("/records", maintenanceHandler.CreateMaintenanceRecord)
			maintenance.GET("/records", maintenanceHandler.GetMaintenanceRecords)
			maintenance.GET("/records/:id", maintenanceHandler.GetMaintenanceRecord)
			maintenance.PATCH("/records/:id", maintenanceHandler.UpdateMaintenanceRecord)
			maintenance.DELETE("/records/:id", maintenanceHandler.DeleteMaintenanceRecord)
			
			// Maintenance Schedules
			maintenance.POST("/schedules", maintenanceHandler.CreateSchedule)
			maintenance.GET("/schedules", maintenanceHandler.GetAllSchedules)
			maintenance.GET("/schedules/upcoming", maintenanceHandler.GetUpcomingSchedules)
			maintenance.GET("/schedules/vehicle/:vehicleId", maintenanceHandler.GetSchedulesByVehicle)
			maintenance.GET("/schedules/:id", maintenanceHandler.GetSchedule)
			maintenance.PATCH("/schedules/:id", maintenanceHandler.UpdateSchedule)
			maintenance.DELETE("/schedules/:id", maintenanceHandler.DeleteSchedule)
			
			// Service Reminders
			maintenance.GET("/reminders/vehicle/:vehicleId", maintenanceHandler.GetServiceReminders)
			maintenance.GET("/reminders/overdue", maintenanceHandler.GetOverdueReminders)
			maintenance.GET("/reminders/due", maintenanceHandler.GetNextServiceDue)
		}
		
		// WebSocket routes (protected)
		ws := protected.Group("/ws")
		{
			ws.GET("/secure", wsHandler.HandleWebSocket)
			ws.GET("/secure/clients", wsHandler.GetConnectedClients)
			ws.POST("/secure/broadcast", wsHandler.BroadcastUpdate)
			ws.DELETE("/secure/clients/:clientId", wsHandler.DisconnectClient)
		}
	}
}
