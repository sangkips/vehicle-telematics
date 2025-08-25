package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fleet-backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebSocketHandler(t *testing.T) {
	manager := websocket.NewManager()
	handler := NewWebSocketHandler(manager)
	
	assert.NotNil(t, handler)
	assert.Equal(t, manager, handler.manager)
}

func TestGetConnectedClients(t *testing.T) {
	manager := websocket.NewManager()
	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()
	
	handler := NewWebSocketHandler(manager)
	
	// Create a test router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws/clients", handler.GetConnectedClients)
	
	// Make request
	req, _ := http.NewRequest("GET", "/ws/clients", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Contains(t, response, "connectedClients")
	assert.Contains(t, response, "stats")
	assert.Equal(t, float64(0), response["connectedClients"]) // No clients connected initially
}

func TestBroadcastUpdate(t *testing.T) {
	manager := websocket.NewManager()
	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()
	
	handler := NewWebSocketHandler(manager)
	
	// Create a test router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/ws/broadcast", handler.BroadcastUpdate)
	
	// Create test update
	update := websocket.VehicleUpdate{
		VehicleID:  "vehicle1",
		UpdateType: "location",
		Data: map[string]interface{}{
			"lat": 40.7128,
			"lng": -74.0060,
		},
		Timestamp: time.Now(),
		Priority:  websocket.PriorityMedium,
	}
	
	updateJSON, _ := json.Marshal(update)
	
	// Make request
	req, _ := http.NewRequest("POST", "/ws/broadcast", bytes.NewBuffer(updateJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "Update broadcasted successfully", response["message"])
}

func TestBroadcastUpdateInvalidJSON(t *testing.T) {
	manager := websocket.NewManager()
	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()
	
	handler := NewWebSocketHandler(manager)
	
	// Create a test router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/ws/broadcast", handler.BroadcastUpdate)
	
	// Make request with invalid JSON
	req, _ := http.NewRequest("POST", "/ws/broadcast", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "Invalid update format", response["error"])
}

func TestDisconnectClient(t *testing.T) {
	manager := websocket.NewManager()
	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()
	
	handler := NewWebSocketHandler(manager)
	
	// Create a test router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.DELETE("/ws/clients/:clientId", handler.DisconnectClient)
	
	// Make request
	req, _ := http.NewRequest("DELETE", "/ws/clients/test-client-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "Client disconnected successfully", response["message"])
}

func TestDisconnectClientMissingID(t *testing.T) {
	manager := websocket.NewManager()
	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()
	
	handler := NewWebSocketHandler(manager)
	
	// Create a test router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.DELETE("/ws/clients/:clientId", handler.DisconnectClient)
	
	// Make request without client ID
	req, _ := http.NewRequest("DELETE", "/ws/clients/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// This should return 404 since the route doesn't match
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleWebSocketQueryFilters(t *testing.T) {
	manager := websocket.NewManager()
	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()
	
	// Create a test context with query parameters
	gin.SetMode(gin.TestMode)
	
	// Create a mock context to test query parameter parsing
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	
	// Set up the request with query parameters
	req, _ := http.NewRequest("GET", "/ws/vehicles?vehicleIds=v1&vehicleIds=v2&statuses=active&drivers=john&alertTypes=fuel_low", nil)
	c.Request = req
	
	// Test that query parameters are parsed correctly
	vehicleIDs := c.QueryArray("vehicleIds")
	statuses := c.QueryArray("statuses")
	drivers := c.QueryArray("drivers")
	alertTypes := c.QueryArray("alertTypes")
	
	assert.Equal(t, []string{"v1", "v2"}, vehicleIDs)
	assert.Equal(t, []string{"active"}, statuses)
	assert.Equal(t, []string{"john"}, drivers)
	assert.Equal(t, []string{"fuel_low"}, alertTypes)
	
	// Note: We can't test the actual WebSocket upgrade in unit tests
	// because httptest.ResponseRecorder doesn't support hijacking
	// This test verifies that query parameter parsing works correctly
}