package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"fleet-backend/internal/websocket"
	"fleet-backend/pkg/batch"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWebSocketManager is a mock implementation of WebSocketManager
type MockWebSocketManager struct {
	mock.Mock
}

func (m *MockWebSocketManager) RegisterClient(clientID string, conn *gorillaws.Conn, filters websocket.VehicleFilters) error {
	args := m.Called(clientID, conn, filters)
	return args.Error(0)
}

func (m *MockWebSocketManager) UnregisterClient(clientID string) error {
	args := m.Called(clientID)
	return args.Error(0)
}

func (m *MockWebSocketManager) BroadcastVehicleUpdate(vehicleID string, update websocket.VehicleUpdate) error {
	args := m.Called(vehicleID, update)
	return args.Error(0)
}

func (m *MockWebSocketManager) BroadcastBatchUpdates(updates []websocket.VehicleUpdate) error {
	args := m.Called(updates)
	return args.Error(0)
}

func (m *MockWebSocketManager) GetConnectedClients() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockWebSocketManager) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockWebSocketManager) Stop() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockWebSocketManager) GetClientStats() websocket.ClientStats {
	args := m.Called()
	return args.Get(0).(websocket.ClientStats)
}

// MockBatchProcessor is a mock implementation of BatchProcessor
type MockBatchProcessor struct {
	mock.Mock
}

func (m *MockBatchProcessor) AddUpdate(vehicleID string, update batch.VehicleUpdateData) error {
	args := m.Called(vehicleID, update)
	return args.Error(0)
}

func (m *MockBatchProcessor) ProcessBatch() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockBatchProcessor) SetBatchSize(size int) {
	m.Called(size)
}

func (m *MockBatchProcessor) SetBatchInterval(interval time.Duration) {
	m.Called(interval)
}

func (m *MockBatchProcessor) GetBatchStats() batch.BatchStats {
	args := m.Called()
	return args.Get(0).(batch.BatchStats)
}

func (m *MockBatchProcessor) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockBatchProcessor) Stop() error {
	args := m.Called()
	return args.Error(0)
}

func TestVehicleWebSocketHandler_BroadcastVehicleUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockWebSocketManager, *MockBatchProcessor)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful broadcast",
			requestBody: websocket.VehicleUpdate{
				VehicleID:  "vehicle-123",
				UpdateType: "location",
				Data: map[string]interface{}{
					"lat": 40.7128,
					"lng": -74.0060,
				},
				Priority: websocket.PriorityMedium,
			},
			setupMocks: func(wsManager *MockWebSocketManager, batchProcessor *MockBatchProcessor) {
				wsManager.On("BroadcastVehicleUpdate", "vehicle-123", mock.AnythingOfType("websocket.VehicleUpdate")).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid JSON",
			requestBody: "invalid json",
			setupMocks: func(wsManager *MockWebSocketManager, batchProcessor *MockBatchProcessor) {
				// No mocks needed for invalid JSON
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid update format",
		},
		{
			name: "missing vehicle ID",
			requestBody: websocket.VehicleUpdate{
				UpdateType: "location",
				Data: map[string]interface{}{
					"lat": 40.7128,
					"lng": -74.0060,
				},
			},
			setupMocks: func(wsManager *MockWebSocketManager, batchProcessor *MockBatchProcessor) {
				// No mocks needed for validation error
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VehicleID is required",
		},
		{
			name: "broadcast failure",
			requestBody: websocket.VehicleUpdate{
				VehicleID:  "vehicle-123",
				UpdateType: "location",
				Data: map[string]interface{}{
					"lat": 40.7128,
					"lng": -74.0060,
				},
			},
			setupMocks: func(wsManager *MockWebSocketManager, batchProcessor *MockBatchProcessor) {
				wsManager.On("BroadcastVehicleUpdate", "vehicle-123", mock.AnythingOfType("websocket.VehicleUpdate")).Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Failed to broadcast update",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockWSManager := new(MockWebSocketManager)
			mockBatchProcessor := new(MockBatchProcessor)
			tt.setupMocks(mockWSManager, mockBatchProcessor)
			
			// Create handler
			handler := NewVehicleWebSocketHandler(mockWSManager, mockBatchProcessor)
			
			// Setup router
			router := gin.New()
			router.POST("/broadcast", handler.BroadcastVehicleUpdate)
			
			// Create request
			var requestBody []byte
			if str, ok := tt.requestBody.(string); ok {
				requestBody = []byte(str)
			} else {
				requestBody, _ = json.Marshal(tt.requestBody)
			}
			
			req, _ := http.NewRequest("POST", "/broadcast", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")
			
			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"], tt.expectedError)
			}
			
			// Verify mocks
			mockWSManager.AssertExpectations(t)
			mockBatchProcessor.AssertExpectations(t)
		})
	}
}

func TestVehicleWebSocketHandler_BroadcastBatchUpdates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockWebSocketManager, *MockBatchProcessor)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful batch broadcast",
			requestBody: []websocket.VehicleUpdate{
				{
					VehicleID:  "vehicle-123",
					UpdateType: "location",
					Data: map[string]interface{}{
						"lat": 40.7128,
						"lng": -74.0060,
					},
					Priority: websocket.PriorityMedium,
				},
				{
					VehicleID:  "vehicle-456",
					UpdateType: "fuel",
					Data: map[string]interface{}{
						"fuelLevel": 75.5,
					},
					Priority: websocket.PriorityLow,
				},
			},
			setupMocks: func(wsManager *MockWebSocketManager, batchProcessor *MockBatchProcessor) {
				wsManager.On("BroadcastBatchUpdates", mock.AnythingOfType("[]websocket.VehicleUpdate")).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "empty batch",
			requestBody: []websocket.VehicleUpdate{},
			setupMocks: func(wsManager *MockWebSocketManager, batchProcessor *MockBatchProcessor) {
				// No mocks needed for empty batch
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "No updates provided",
		},
		{
			name: "missing vehicle ID in batch",
			requestBody: []websocket.VehicleUpdate{
				{
					UpdateType: "location",
					Data: map[string]interface{}{
						"lat": 40.7128,
						"lng": -74.0060,
					},
				},
			},
			setupMocks: func(wsManager *MockWebSocketManager, batchProcessor *MockBatchProcessor) {
				// No mocks needed for validation error
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VehicleID is required for all updates",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockWSManager := new(MockWebSocketManager)
			mockBatchProcessor := new(MockBatchProcessor)
			tt.setupMocks(mockWSManager, mockBatchProcessor)
			
			// Create handler
			handler := NewVehicleWebSocketHandler(mockWSManager, mockBatchProcessor)
			
			// Setup router
			router := gin.New()
			router.POST("/broadcast-batch", handler.BroadcastBatchUpdates)
			
			// Create request
			requestBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/broadcast-batch", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")
			
			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"], tt.expectedError)
			}
			
			// Verify mocks
			mockWSManager.AssertExpectations(t)
			mockBatchProcessor.AssertExpectations(t)
		})
	}
}

func TestVehicleWebSocketHandler_GetSubscriptionStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Setup mocks
	mockWSManager := new(MockWebSocketManager)
	mockBatchProcessor := new(MockBatchProcessor)
	
	expectedWSStats := websocket.ClientStats{
		TotalClients:    10,
		ActiveClients:   8,
		InactiveClients: 2,
	}
	
	expectedBatchStats := batch.BatchStats{
		BatchesProcessed: 5,
		AverageSize:      25.5,
		ProcessingTime:   time.Millisecond * 150,
		ErrorRate:        0.02,
		TotalUpdates:     127,
		FailedUpdates:    3,
		LastProcessedAt:  time.Now(),
	}
	
	mockWSManager.On("GetClientStats").Return(expectedWSStats)
	mockBatchProcessor.On("GetBatchStats").Return(expectedBatchStats)
	
	// Create handler
	handler := NewVehicleWebSocketHandler(mockWSManager, mockBatchProcessor)
	
	// Setup router
	router := gin.New()
	router.GET("/stats", handler.GetSubscriptionStats)
	
	// Create request
	req, _ := http.NewRequest("GET", "/stats", nil)
	
	// Execute request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	// Verify WebSocket stats
	wsStats := response["websocket"].(map[string]interface{})
	assert.Equal(t, float64(expectedWSStats.TotalClients), wsStats["totalClients"])
	assert.Equal(t, float64(expectedWSStats.ActiveClients), wsStats["activeClients"])
	assert.Equal(t, float64(expectedWSStats.InactiveClients), wsStats["inactiveClients"])
	
	// Verify batch stats
	batchStats := response["batch"].(map[string]interface{})
	assert.Equal(t, float64(expectedBatchStats.BatchesProcessed), batchStats["batchesProcessed"])
	assert.Equal(t, expectedBatchStats.AverageSize, batchStats["averageSize"])
	assert.Equal(t, float64(expectedBatchStats.TotalUpdates), batchStats["totalUpdates"])
	assert.Equal(t, float64(expectedBatchStats.FailedUpdates), batchStats["failedUpdates"])
	
	// Verify mocks
	mockWSManager.AssertExpectations(t)
	mockBatchProcessor.AssertExpectations(t)
}

func TestVehicleWebSocketHandler_DisconnectClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		clientID       string
		setupMocks     func(*MockWebSocketManager)
		expectedStatus int
		expectedError  string
	}{
		{
			name:     "successful disconnect",
			clientID: "client-123",
			setupMocks: func(wsManager *MockWebSocketManager) {
				wsManager.On("UnregisterClient", "client-123").Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "missing client ID",
			clientID: " ", // Use space instead of empty string to avoid 404
			setupMocks: func(wsManager *MockWebSocketManager) {
				// No mocks needed for validation error
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Client ID is required",
		},
		{
			name:     "disconnect failure",
			clientID: "client-123",
			setupMocks: func(wsManager *MockWebSocketManager) {
				wsManager.On("UnregisterClient", "client-123").Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Failed to disconnect client",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockWSManager := new(MockWebSocketManager)
			mockBatchProcessor := new(MockBatchProcessor)
			tt.setupMocks(mockWSManager)
			
			// Create handler
			handler := NewVehicleWebSocketHandler(mockWSManager, mockBatchProcessor)
			
			// Setup router
			router := gin.New()
			router.DELETE("/disconnect/:clientId", handler.DisconnectClient)
			
			// Create request
			path := "/disconnect/" + tt.clientID
			req, _ := http.NewRequest("DELETE", path, nil)
			
			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"], tt.expectedError)
			}
			
			// Verify mocks
			mockWSManager.AssertExpectations(t)
		})
	}
}

func TestVehicleWebSocketHandler_TestConnection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name             string
		connectedClients int
		setupMocks       func(*MockWebSocketManager)
		expectedStatus   int
	}{
		{
			name:             "test with connected clients",
			connectedClients: 5,
			setupMocks: func(wsManager *MockWebSocketManager) {
				wsManager.On("GetConnectedClients").Return(5)
				wsManager.On("BroadcastVehicleUpdate", mock.AnythingOfType("string"), mock.AnythingOfType("websocket.VehicleUpdate")).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:             "test with no connected clients",
			connectedClients: 0,
			setupMocks: func(wsManager *MockWebSocketManager) {
				wsManager.On("GetConnectedClients").Return(0)
				// No broadcast call expected when no clients are connected
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:             "test with broadcast failure",
			connectedClients: 3,
			setupMocks: func(wsManager *MockWebSocketManager) {
				wsManager.On("GetConnectedClients").Return(3)
				wsManager.On("BroadcastVehicleUpdate", mock.AnythingOfType("string"), mock.AnythingOfType("websocket.VehicleUpdate")).Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockWSManager := new(MockWebSocketManager)
			mockBatchProcessor := new(MockBatchProcessor)
			tt.setupMocks(mockWSManager)
			
			// Create handler
			handler := NewVehicleWebSocketHandler(mockWSManager, mockBatchProcessor)
			
			// Setup router
			router := gin.New()
			router.GET("/test", handler.TestConnection)
			
			// Create request
			req, _ := http.NewRequest("GET", "/test", nil)
			
			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			
			assert.Equal(t, float64(tt.connectedClients), response["connectedClients"])
			
			// Verify mocks
			mockWSManager.AssertExpectations(t)
		})
	}
}

// Integration test for WebSocket connection handling
func TestVehicleWebSocketHandler_HandleVehicleUpdates_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create a real WebSocket manager for integration testing
	wsManager := websocket.NewManager()
	err := wsManager.Start()
	assert.NoError(t, err)
	defer wsManager.Stop()
	
	mockBatchProcessor := new(MockBatchProcessor)
	
	// Create handler
	handler := NewVehicleWebSocketHandler(wsManager, mockBatchProcessor)
	
	// Setup router
	router := gin.New()
	router.GET("/ws", handler.HandleVehicleUpdates)
	
	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()
	
	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	
	// Add query parameters for filters
	u, err := url.Parse(wsURL)
	assert.NoError(t, err)
	
	q := u.Query()
	q.Add("vehicleIds", "vehicle-123")
	q.Add("vehicleIds", "vehicle-456")
	q.Add("statuses", "active")
	q.Add("alertTypes", "fuel_theft")
	u.RawQuery = q.Encode()
	
	// Connect to WebSocket
	conn, _, err := gorillaws.DefaultDialer.Dial(u.String(), nil)
	assert.NoError(t, err)
	defer conn.Close()
	
	// Read connection confirmation message
	var confirmationMsg map[string]interface{}
	err = conn.ReadJSON(&confirmationMsg)
	assert.NoError(t, err)
	assert.Equal(t, "connection_confirmed", confirmationMsg["type"])
	assert.NotEmpty(t, confirmationMsg["clientId"])
	
	// Verify filters were applied
	filters := confirmationMsg["filters"].(map[string]interface{})
	vehicleIds := filters["vehicleIds"].([]interface{})
	assert.Contains(t, vehicleIds, "vehicle-123")
	assert.Contains(t, vehicleIds, "vehicle-456")
	
	statuses := filters["statuses"].([]interface{})
	assert.Contains(t, statuses, "active")
	
	alertTypes := filters["alertTypes"].([]interface{})
	assert.Contains(t, alertTypes, "fuel_theft")
	
	// Test broadcasting an update
	testUpdate := websocket.VehicleUpdate{
		VehicleID:  "vehicle-123",
		UpdateType: "location",
		Data: map[string]interface{}{
			"lat": 40.7128,
			"lng": -74.0060,
		},
		Timestamp: time.Now(),
		Priority:  websocket.PriorityMedium,
	}
	
	// Broadcast the update
	err = wsManager.BroadcastVehicleUpdate(testUpdate.VehicleID, testUpdate)
	assert.NoError(t, err)
	
	// Read the broadcasted message
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var receivedMsg map[string]interface{}
	err = conn.ReadJSON(&receivedMsg)
	assert.NoError(t, err)
	
	assert.Equal(t, "vehicle_update", receivedMsg["type"])
	
	data := receivedMsg["data"].(map[string]interface{})
	assert.Equal(t, "vehicle-123", data["vehicleId"])
	assert.Equal(t, "location", data["updateType"])
	
	updateData := data["data"].(map[string]interface{})
	assert.Equal(t, 40.7128, updateData["lat"])
	assert.Equal(t, -74.0060, updateData["lng"])
}