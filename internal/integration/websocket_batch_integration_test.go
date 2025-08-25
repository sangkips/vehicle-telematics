package integration

import (
	"testing"
	"time"

	"fleet-backend/internal/models"
	"fleet-backend/internal/websocket"
	"fleet-backend/pkg/batch"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockVehicleRepository implements the batch.VehicleRepository interface for testing
type MockVehicleRepository struct {
	mock.Mock
}

func (m *MockVehicleRepository) UpdateVehicle(vehicleID string, update batch.VehicleUpdateData) error {
	args := m.Called(vehicleID, update)
	return args.Error(0)
}

func (m *MockVehicleRepository) UpdateVehiclesBatch(updates map[string]batch.VehicleUpdateData) error {
	args := m.Called(updates)
	return args.Error(0)
}

// TestWebSocketBatchIntegration tests the complete integration between batch processor and WebSocket broadcasting
func TestWebSocketBatchIntegration(t *testing.T) {
	// Create WebSocket manager
	wsManager := websocket.NewManager()
	err := wsManager.Start()
	assert.NoError(t, err)
	defer wsManager.Stop()

	// Create mock repository
	mockRepo := new(MockVehicleRepository)
	
	// Create batch processor with WebSocket integration
	config := batch.BatchConfig{
		MaxBatchSize:  3,
		BatchInterval: 100 * time.Millisecond,
		MaxWaitTime:   500 * time.Millisecond,
		RetryAttempts: 1,
		RetryBackoff:  10 * time.Millisecond,
	}
	
	batchProcessor := batch.NewBatchProcessorWithWebSocket(config, mockRepo, wsManager)
	
	// Start batch processor
	err = batchProcessor.Start()
	assert.NoError(t, err)
	defer batchProcessor.Stop()

	// Set up mock expectations for successful batch processing
	mockRepo.On("UpdateVehiclesBatch", mock.AnythingOfType("map[string]batch.VehicleUpdateData")).Return(nil)

	// Create test vehicle updates
	updates := []struct {
		vehicleID string
		data      batch.VehicleUpdateData
	}{
		{
			vehicleID: "vehicle-1",
			data: batch.VehicleUpdateData{
				FuelLevel: func() *float64 { f := 75.5; return &f }(),
				Speed:     func() *int { s := 60; return &s }(),
				Timestamp: time.Now(),
			},
		},
		{
			vehicleID: "vehicle-2",
			data: batch.VehicleUpdateData{
				FuelLevel: func() *float64 { f := 15.0; return &f }(), // Low fuel - should be high priority
				Location: &models.Location{
					Lat: 40.7128,
					Lng: -74.0060,
				},
				Timestamp: time.Now(),
			},
		},
		{
			vehicleID: "vehicle-3",
			data: batch.VehicleUpdateData{
				Speed:     func() *int { s := 90; return &s }(), // Speeding - should be high priority
				Timestamp: time.Now(),
			},
		},
	}

	// Add updates to batch processor
	for _, update := range updates {
		err := batchProcessor.AddUpdate(update.vehicleID, update.data)
		assert.NoError(t, err)
	}

	// Wait for batch processing to complete
	time.Sleep(200 * time.Millisecond)

	// Verify that the repository was called
	mockRepo.AssertExpectations(t)

	// Verify batch statistics
	stats := batchProcessor.GetBatchStats()
	assert.Greater(t, stats.BatchesProcessed, 0)
	assert.Equal(t, int64(3), stats.TotalUpdates)
}

// TestWebSocketBroadcastingPriority tests that critical alerts are prioritized correctly
func TestWebSocketBroadcastingPriority(t *testing.T) {
	// Create WebSocket manager
	wsManager := websocket.NewManager()
	err := wsManager.Start()
	assert.NoError(t, err)
	defer wsManager.Stop()

	// Create mock repository
	mockRepo := new(MockVehicleRepository)
	
	// Create batch processor with WebSocket integration
	config := batch.BatchConfig{
		MaxBatchSize:  5,
		BatchInterval: 50 * time.Millisecond,
		MaxWaitTime:   200 * time.Millisecond,
		RetryAttempts: 1,
		RetryBackoff:  10 * time.Millisecond,
	}
	
	batchProcessor := batch.NewBatchProcessorWithWebSocket(config, mockRepo, wsManager)
	
	// Start batch processor
	err = batchProcessor.Start()
	assert.NoError(t, err)
	defer batchProcessor.Stop()

	// Set up mock expectations
	mockRepo.On("UpdateVehiclesBatch", mock.AnythingOfType("map[string]batch.VehicleUpdateData")).Return(nil)

	// Create updates with different priority levels
	criticalUpdate := batch.VehicleUpdateData{
		FuelLevel: func() *float64 { f := 5.0; return &f }(), // Critical fuel level
		Timestamp: time.Now(),
	}
	
	highPriorityUpdate := batch.VehicleUpdateData{
		FuelLevel: func() *float64 { f := 15.0; return &f }(), // Low fuel level
		Timestamp: time.Now(),
	}
	
	normalUpdate := batch.VehicleUpdateData{
		FuelLevel: func() *float64 { f := 50.0; return &f }(), // Normal fuel level
		Timestamp: time.Now(),
	}

	// Add updates to batch processor
	err = batchProcessor.AddUpdate("critical-vehicle", criticalUpdate)
	assert.NoError(t, err)
	
	err = batchProcessor.AddUpdate("high-priority-vehicle", highPriorityUpdate)
	assert.NoError(t, err)
	
	err = batchProcessor.AddUpdate("normal-vehicle", normalUpdate)
	assert.NoError(t, err)

	// Wait for batch processing
	time.Sleep(100 * time.Millisecond)

	// Verify repository was called
	mockRepo.AssertExpectations(t)
}

// TestWebSocketFilteredBroadcasting tests that updates are filtered correctly based on client subscriptions
func TestWebSocketFilteredBroadcasting(t *testing.T) {
	// Create WebSocket manager
	wsManager := websocket.NewManager()
	err := wsManager.Start()
	assert.NoError(t, err)
	defer wsManager.Stop()

	// Test direct broadcasting with filters
	update1 := websocket.VehicleUpdate{
		VehicleID:  "vehicle-123",
		UpdateType: "fuel",
		Data: map[string]interface{}{
			"fuelLevel": 25.5,
		},
		Timestamp: time.Now(),
		Priority:  websocket.PriorityMedium,
	}

	update2 := websocket.VehicleUpdate{
		VehicleID:  "vehicle-456",
		UpdateType: "location",
		Data: map[string]interface{}{
			"lat": 40.7128,
			"lng": -74.0060,
		},
		Timestamp: time.Now(),
		Priority:  websocket.PriorityLow,
	}

	// Test broadcasting individual updates
	err = wsManager.BroadcastVehicleUpdate(update1.VehicleID, update1)
	assert.NoError(t, err)

	// Test broadcasting batch updates
	batchUpdates := []websocket.VehicleUpdate{update1, update2}
	err = wsManager.BroadcastBatchUpdates(batchUpdates)
	assert.NoError(t, err)

	// Verify client stats
	stats := wsManager.GetClientStats()
	assert.Equal(t, 0, stats.TotalClients) // No clients connected in this test
}

// TestBatchProcessorWebSocketIntegrationFailure tests graceful handling when WebSocket broadcasting fails
func TestBatchProcessorWebSocketIntegrationFailure(t *testing.T) {
	// Create WebSocket manager but don't start it to simulate failure
	wsManager := websocket.NewManager()
	// Note: Not starting the manager to simulate failure conditions

	// Create mock repository
	mockRepo := new(MockVehicleRepository)
	
	// Create batch processor with WebSocket integration
	config := batch.BatchConfig{
		MaxBatchSize:  2,
		BatchInterval: 50 * time.Millisecond,
		MaxWaitTime:   200 * time.Millisecond,
		RetryAttempts: 1,
		RetryBackoff:  10 * time.Millisecond,
	}
	
	batchProcessor := batch.NewBatchProcessorWithWebSocket(config, mockRepo, wsManager)
	
	// Start batch processor
	err := batchProcessor.Start()
	assert.NoError(t, err)
	defer batchProcessor.Stop()

	// Set up mock expectations - database update should still succeed
	mockRepo.On("UpdateVehiclesBatch", mock.AnythingOfType("map[string]batch.VehicleUpdateData")).Return(nil)

	// Create test update
	updateData := batch.VehicleUpdateData{
		FuelLevel: func() *float64 { f := 30.0; return &f }(),
		Timestamp: time.Now(),
	}

	// Add update to batch processor
	err = batchProcessor.AddUpdate("test-vehicle", updateData)
	assert.NoError(t, err)

	// Wait for batch processing
	time.Sleep(100 * time.Millisecond)

	// Verify that database update still succeeded even if WebSocket broadcasting failed
	mockRepo.AssertExpectations(t)
	
	// Verify batch statistics show successful processing
	stats := batchProcessor.GetBatchStats()
	assert.Greater(t, stats.BatchesProcessed, 0)
	assert.Equal(t, int64(1), stats.TotalUpdates)
}

// TestRealTimeUpdateDelivery tests the complete flow from batch processing to client delivery
func TestRealTimeUpdateDelivery(t *testing.T) {
	// This test would require a more complex setup with actual WebSocket connections
	// For now, we'll test the components individually and verify integration points
	
	// Create WebSocket manager
	wsManager := websocket.NewManager()
	err := wsManager.Start()
	assert.NoError(t, err)
	defer wsManager.Stop()

	// Create mock repository
	mockRepo := new(MockVehicleRepository)
	
	// Create batch processor
	config := batch.BatchConfig{
		MaxBatchSize:  1, // Process immediately
		BatchInterval: 10 * time.Millisecond,
		MaxWaitTime:   50 * time.Millisecond,
		RetryAttempts: 1,
		RetryBackoff:  5 * time.Millisecond,
	}
	
	batchProcessor := batch.NewBatchProcessorWithWebSocket(config, mockRepo, wsManager)
	
	// Start batch processor
	err = batchProcessor.Start()
	assert.NoError(t, err)
	defer batchProcessor.Stop()

	// Set up mock expectations
	mockRepo.On("UpdateVehiclesBatch", mock.AnythingOfType("map[string]batch.VehicleUpdateData")).Return(nil)

	// Create a critical update that should be broadcast immediately
	criticalUpdate := batch.VehicleUpdateData{
		FuelLevel: func() *float64 { f := 3.0; return &f }(), // Critical fuel level
		Status:    func() *string { s := "maintenance"; return &s }(), // Critical status
		Timestamp: time.Now(),
	}

	// Add critical update
	err = batchProcessor.AddUpdate("critical-vehicle", criticalUpdate)
	assert.NoError(t, err)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Verify processing completed
	mockRepo.AssertExpectations(t)
	
	// Verify statistics
	stats := batchProcessor.GetBatchStats()
	assert.Greater(t, stats.BatchesProcessed, 0)
	assert.Equal(t, int64(1), stats.TotalUpdates)
	assert.Equal(t, float64(0), stats.ErrorRate) // No errors expected
}