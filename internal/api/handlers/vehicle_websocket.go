package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"fleet-backend/internal/websocket"
	"fleet-backend/pkg/batch"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// VehicleWebSocketHandler handles WebSocket connections specifically for vehicle updates
type VehicleWebSocketHandler struct {
	wsManager     websocket.WebSocketManager
	batchProcessor batch.BatchProcessor
}

// NewVehicleWebSocketHandler creates a new vehicle WebSocket handler
func NewVehicleWebSocketHandler(wsManager websocket.WebSocketManager, batchProcessor batch.BatchProcessor) *VehicleWebSocketHandler {
	return &VehicleWebSocketHandler{
		wsManager:     wsManager,
		batchProcessor: batchProcessor,
	}
}

// SubscriptionRequest represents a client subscription request
type SubscriptionRequest struct {
	Type    string                 `json:"type"`
	Filters websocket.VehicleFilters `json:"filters"`
}

// HandleVehicleUpdates upgrades HTTP connections to WebSocket for vehicle update subscriptions
func (h *VehicleWebSocketHandler) HandleVehicleUpdates(c *gin.Context) {
	// Generate unique client ID
	clientID := uuid.New().String()
	
	// Parse query parameters for initial filters
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
	manager := h.wsManager.(*websocket.Manager)
	
	// Upgrade the HTTP connection to WebSocket
	conn, err := manager.GetUpgrader().Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection to WebSocket: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to upgrade to WebSocket"})
		return
	}
	
	// Register the client with the WebSocket manager
	err = h.wsManager.RegisterClient(clientID, conn, filters)
	if err != nil {
		log.Printf("Failed to register WebSocket client: %v", err)
		conn.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register client"})
		return
	}
	
	log.Printf("Vehicle WebSocket client %s connected with filters: %+v", clientID, filters)
	
	// Send initial connection confirmation
	confirmationMsg := map[string]interface{}{
		"type":      "connection_confirmed",
		"clientId":  clientID,
		"timestamp": time.Now(),
		"filters":   filters,
	}
	
	if err := conn.WriteJSON(confirmationMsg); err != nil {
		log.Printf("Failed to send connection confirmation to client %s: %v", clientID, err)
	}
}

// BroadcastVehicleUpdate allows manual broadcasting of single vehicle updates
func (h *VehicleWebSocketHandler) BroadcastVehicleUpdate(c *gin.Context) {
	var update websocket.VehicleUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid update format", "details": err.Error()})
		return
	}
	
	// Validate required fields
	if update.VehicleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "VehicleID is required"})
		return
	}
	
	if update.UpdateType == "" {
		update.UpdateType = "manual"
	}
	
	if update.Priority == "" {
		update.Priority = websocket.PriorityMedium
	}
	
	if update.Timestamp.IsZero() {
		update.Timestamp = time.Now()
	}
	
	// Broadcast the update
	err := h.wsManager.BroadcastVehicleUpdate(update.VehicleID, update)
	if err != nil {
		log.Printf("Failed to broadcast vehicle update: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to broadcast update", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":   "Vehicle update broadcasted successfully",
		"vehicleId": update.VehicleID,
		"updateType": update.UpdateType,
		"timestamp": update.Timestamp,
	})
}

// BroadcastBatchUpdates allows manual broadcasting of batch vehicle updates
func (h *VehicleWebSocketHandler) BroadcastBatchUpdates(c *gin.Context) {
	var updates []websocket.VehicleUpdate
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid batch updates format", "details": err.Error()})
		return
	}
	
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No updates provided"})
		return
	}
	
	// Validate and set defaults for each update
	now := time.Now()
	for i := range updates {
		if updates[i].VehicleID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "VehicleID is required for all updates"})
			return
		}
		
		if updates[i].UpdateType == "" {
			updates[i].UpdateType = "batch"
		}
		
		if updates[i].Priority == "" {
			updates[i].Priority = websocket.PriorityMedium
		}
		
		if updates[i].Timestamp.IsZero() {
			updates[i].Timestamp = now
		}
	}
	
	// Broadcast the batch updates
	err := h.wsManager.BroadcastBatchUpdates(updates)
	if err != nil {
		log.Printf("Failed to broadcast batch updates: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to broadcast batch updates", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":     "Batch updates broadcasted successfully",
		"updateCount": len(updates),
		"timestamp":   now,
	})
}

// GetSubscriptionStats returns statistics about WebSocket subscriptions
func (h *VehicleWebSocketHandler) GetSubscriptionStats(c *gin.Context) {
	stats := h.wsManager.GetClientStats()
	
	response := gin.H{
		"websocket": stats,
		"timestamp": time.Now(),
	}
	
	// Only include batch stats if batch processor is available
	if h.batchProcessor != nil {
		batchStats := h.batchProcessor.GetBatchStats()
		response["batch"] = batchStats
	} else {
		response["batch"] = gin.H{
			"message": "Batch processor not configured",
		}
	}
	
	c.JSON(http.StatusOK, response)
}

// DisconnectClient allows manual disconnection of a specific client
func (h *VehicleWebSocketHandler) DisconnectClient(c *gin.Context) {
	clientID := strings.TrimSpace(c.Param("clientId"))
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Client ID is required"})
		return
	}
	
	err := h.wsManager.UnregisterClient(clientID)
	if err != nil {
		log.Printf("Failed to disconnect client %s: %v", clientID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect client", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":   "Client disconnected successfully",
		"clientId":  clientID,
		"timestamp": time.Now(),
	})
}

// TestConnection allows testing WebSocket connectivity
func (h *VehicleWebSocketHandler) TestConnection(c *gin.Context) {
	connectedClients := h.wsManager.GetConnectedClients()
	
	// Create a test update
	testUpdate := websocket.VehicleUpdate{
		VehicleID:  "test-vehicle-" + uuid.New().String()[:8],
		UpdateType: "test",
		Data: map[string]interface{}{
			"message": "This is a test update",
			"source":  "api_test",
		},
		Timestamp: time.Now(),
		Priority:  websocket.PriorityLow,
	}
	
	// Broadcast test update if there are connected clients
	if connectedClients > 0 {
		err := h.wsManager.BroadcastVehicleUpdate(testUpdate.VehicleID, testUpdate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":            "Failed to send test update",
				"connectedClients": connectedClients,
				"details":          err.Error(),
			})
			return
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":          "Test connection successful",
		"connectedClients": connectedClients,
		"testUpdate":       testUpdate,
		"timestamp":        time.Now(),
	})
}

// GetActiveFilters returns the active filters for all connected clients
func (h *VehicleWebSocketHandler) GetActiveFilters(c *gin.Context) {
	// This would require extending the WebSocket manager to expose client filters
	// For now, return basic stats
	stats := h.wsManager.GetClientStats()
	
	c.JSON(http.StatusOK, gin.H{
		"message":        "Active filters information",
		"clientStats":    stats,
		"timestamp":      time.Now(),
		"note":          "Detailed filter information requires WebSocket manager extension",
	})
}