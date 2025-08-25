package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.clients)
	assert.NotNil(t, manager.register)
	assert.NotNil(t, manager.unregister)
	assert.NotNil(t, manager.broadcast)
}

func TestManagerStartStop(t *testing.T) {
	manager := NewManager()
	
	err := manager.Start()
	assert.NoError(t, err)
	
	// Give the manager a moment to start
	time.Sleep(10 * time.Millisecond)
	
	err = manager.Stop()
	assert.NoError(t, err)
}

func TestRegisterClient(t *testing.T) {
	manager := NewManager()
	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()
	
	// Create a mock WebSocket connection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := manager.upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()
		
		filters := VehicleFilters{
			VehicleIDs: []string{"vehicle1", "vehicle2"},
		}
		
		err = manager.RegisterClient("test-client", conn, filters)
		assert.NoError(t, err)
		
		// Give time for registration
		time.Sleep(50 * time.Millisecond)
		
		assert.Equal(t, 1, manager.GetConnectedClients())
	}))
	defer server.Close()
	
	// Connect to the test server
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()
	
	// Wait for the handler to complete
	time.Sleep(100 * time.Millisecond)
}

func TestUnregisterClient(t *testing.T) {
	manager := NewManager()
	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := manager.upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()
		
		filters := VehicleFilters{}
		err = manager.RegisterClient("test-client", conn, filters)
		require.NoError(t, err)
		
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, 1, manager.GetConnectedClients())
		
		err = manager.UnregisterClient("test-client")
		assert.NoError(t, err)
		
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, 0, manager.GetConnectedClients())
	}))
	defer server.Close()
	
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()
	
	time.Sleep(150 * time.Millisecond)
}

func TestBroadcastVehicleUpdate(t *testing.T) {
	manager := NewManager()
	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()
	
	// Test broadcasting without actual WebSocket connection
	update := VehicleUpdate{
		VehicleID:  "vehicle1",
		UpdateType: "location",
		Data: map[string]interface{}{
			"lat": 40.7128,
			"lng": -74.0060,
		},
		Timestamp: time.Now(),
		Priority:  PriorityMedium,
	}
	
	err = manager.BroadcastVehicleUpdate("vehicle1", update)
	assert.NoError(t, err)
	
	// Test that broadcast channel receives the update
	select {
	case received := <-manager.broadcast:
		assert.Equal(t, "vehicle1", received.VehicleID)
		assert.Equal(t, "location", received.UpdateType)
		assert.Equal(t, PriorityMedium, received.Priority)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Did not receive vehicle update in broadcast channel")
	}
}

func TestBroadcastBatchUpdates(t *testing.T) {
	manager := NewManager()
	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()
	
	updates := []VehicleUpdate{
		{
			VehicleID:  "vehicle1",
			UpdateType: "fuel",
			Data:       map[string]interface{}{"level": 75},
			Timestamp:  time.Now(),
			Priority:   PriorityCritical,
		},
		{
			VehicleID:  "vehicle2",
			UpdateType: "status",
			Data:       map[string]interface{}{"status": "active"},
			Timestamp:  time.Now(),
			Priority:   PriorityLow,
		},
	}
	
	err = manager.BroadcastBatchUpdates(updates)
	assert.NoError(t, err)
}

func TestShouldSendToClient(t *testing.T) {
	manager := NewManager()
	
	tests := []struct {
		name     string
		filters  VehicleFilters
		update   VehicleUpdate
		expected bool
	}{
		{
			name:    "no filters - should send all",
			filters: VehicleFilters{},
			update: VehicleUpdate{
				VehicleID: "vehicle1",
				Data:      map[string]interface{}{},
			},
			expected: true,
		},
		{
			name: "vehicle ID filter - matching",
			filters: VehicleFilters{
				VehicleIDs: []string{"vehicle1", "vehicle2"},
			},
			update: VehicleUpdate{
				VehicleID: "vehicle1",
			},
			expected: true,
		},
		{
			name: "vehicle ID filter - not matching",
			filters: VehicleFilters{
				VehicleIDs: []string{"vehicle2", "vehicle3"},
			},
			update: VehicleUpdate{
				VehicleID: "vehicle1",
			},
			expected: false,
		},
		{
			name: "status filter - matching",
			filters: VehicleFilters{
				Statuses: []string{"active", "maintenance"},
			},
			update: VehicleUpdate{
				VehicleID: "vehicle1",
				Data: map[string]interface{}{
					"status": "active",
				},
			},
			expected: true,
		},
		{
			name: "alert type filter - matching",
			filters: VehicleFilters{
				AlertTypes: []string{"fuel_low", "maintenance_due"},
			},
			update: VehicleUpdate{
				VehicleID:  "vehicle1",
				UpdateType: "alert",
				Data: map[string]interface{}{
					"alertType": "fuel_low",
				},
			},
			expected: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				ID:      "test-client",
				Filters: tt.filters,
			}
			
			result := manager.shouldSendToClient(client, tt.update)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetClientStats(t *testing.T) {
	manager := NewManager()
	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()
	
	// Initially no clients
	stats := manager.GetClientStats()
	assert.Equal(t, 0, stats.TotalClients)
	assert.Equal(t, 0, stats.ActiveClients)
	assert.Equal(t, 0, stats.InactiveClients)
	
	// Add a mock client directly for testing
	client := &Client{
		ID:       "test-client",
		Filters:  VehicleFilters{},
		Send:     make(chan VehicleUpdate, 256),
		LastPing: time.Now(),
		IsActive: true,
	}
	
	manager.mutex.Lock()
	manager.clients["test-client"] = client
	manager.mutex.Unlock()
	
	stats = manager.GetClientStats()
	assert.Equal(t, 1, stats.TotalClients)
	assert.Equal(t, 1, stats.ActiveClients)
	assert.Equal(t, 0, stats.InactiveClients)
	
	// Mark client as inactive
	client.IsActive = false
	
	stats = manager.GetClientStats()
	assert.Equal(t, 1, stats.TotalClients)
	assert.Equal(t, 0, stats.ActiveClients)
	assert.Equal(t, 1, stats.InactiveClients)
}

func TestHealthCheck(t *testing.T) {
	manager := NewManager()
	
	// Add a mock client with old ping time
	oldClient := &Client{
		ID:       "old-client",
		Filters:  VehicleFilters{},
		Send:     make(chan VehicleUpdate, 256),
		LastPing: time.Now().Add(-2 * time.Minute), // 2 minutes ago
		IsActive: true,
	}
	
	// Add a fresh client
	freshClient := &Client{
		ID:       "fresh-client",
		Filters:  VehicleFilters{},
		Send:     make(chan VehicleUpdate, 256),
		LastPing: time.Now(),
		IsActive: true,
	}
	
	manager.mutex.Lock()
	manager.clients["old-client"] = oldClient
	manager.clients["fresh-client"] = freshClient
	manager.mutex.Unlock()
	
	assert.Equal(t, 2, len(manager.clients))
	
	// Run health check
	manager.healthCheck()
	
	// Old client should be removed, fresh client should remain
	assert.Equal(t, 1, len(manager.clients))
	_, exists := manager.clients["fresh-client"]
	assert.True(t, exists)
	_, exists = manager.clients["old-client"]
	assert.False(t, exists)
}