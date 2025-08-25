package handlers

import (
	"log"
	"net/http"
	"strings"

	"fleet-backend/internal/websocket"
	"fleet-backend/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WebSocketHandler handles WebSocket connections for real-time updates
type WebSocketHandler struct {
	manager websocket.WebSocketManager
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(manager websocket.WebSocketManager) *WebSocketHandler {
	return &WebSocketHandler{
		manager: manager,
	}
}

// HandleWebSocket upgrades HTTP connections to WebSocket for real-time vehicle updates
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// Validate JWT token from query parameter or Authorization header
	token := c.Query("token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}
	
	if token == "" {
		log.Printf("WebSocket connection rejected: no token provided")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication token required"})
		return
	}
	
	// Validate the JWT token
	jwtUtil := jwt.NewJWTUtil()
	_, err := jwtUtil.ValidateToken(token)
	if err != nil {
		log.Printf("WebSocket connection rejected: invalid token - %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token"})
		return
	}
	
	// Generate unique client ID
	clientID := uuid.New().String()
	
	// Parse query parameters for filters
	filters := websocket.VehicleFilters{}
	
	// Parse vehicle IDs filter
	if vehicleIDs := c.QueryArray("vehicleIds"); len(vehicleIDs) > 0 {
		filters.VehicleIDs = vehicleIDs
	}
	
	// Parse statuses filter
	if statuses := c.QueryArray("statuses"); len(statuses) > 0 {
		filters.Statuses = statuses
	}
	
	// Parse drivers filter
	if drivers := c.QueryArray("drivers"); len(drivers) > 0 {
		filters.Drivers = drivers
	}
	
	// Parse alert types filter
	if alertTypes := c.QueryArray("alertTypes"); len(alertTypes) > 0 {
		filters.AlertTypes = alertTypes
	}
	
	// Get the WebSocket manager from the handler
	manager := h.manager.(*websocket.Manager)
	
	// Upgrade the HTTP connection to WebSocket
	conn, err := manager.GetUpgrader().Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection to WebSocket: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to upgrade to WebSocket"})
		return
	}
	
	// Register the client with the WebSocket manager
	err = h.manager.RegisterClient(clientID, conn, filters)
	if err != nil {
		log.Printf("Failed to register WebSocket client: %v", err)
		conn.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register client"})
		return
	}
	
	log.Printf("WebSocket client %s connected with filters: %+v", clientID, filters)
}

// GetConnectedClients returns the number of connected WebSocket clients
func (h *WebSocketHandler) GetConnectedClients(c *gin.Context) {
	count := h.manager.GetConnectedClients()
	stats := h.manager.GetClientStats()
	
	c.JSON(http.StatusOK, gin.H{
		"connectedClients": count,
		"stats":           stats,
	})
}

// BroadcastUpdate allows manual broadcasting of vehicle updates (for testing/admin purposes)
func (h *WebSocketHandler) BroadcastUpdate(c *gin.Context) {
	var update websocket.VehicleUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid update format"})
		return
	}
	
	err := h.manager.BroadcastVehicleUpdate(update.VehicleID, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to broadcast update"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Update broadcasted successfully"})
}

// DisconnectClient allows manual disconnection of a client (for admin purposes)
func (h *WebSocketHandler) DisconnectClient(c *gin.Context) {
	clientID := c.Param("clientId")
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Client ID is required"})
		return
	}
	
	err := h.manager.UnregisterClient(clientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect client"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Client disconnected successfully"})
}