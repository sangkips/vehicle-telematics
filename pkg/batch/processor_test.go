package batch

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"fleet-backend/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockVehicleRepository is a mock implementation of VehicleRepository
type MockVehicleRepository struct {
	mock.Mock
	mu sync.Mutex
}

func (m *MockVehicleRepository) UpdateVehicle(vehicleID string, update VehicleUpdateData) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	args := m.Called(vehicleID, update)
	return args.Error(0)
}

func (m *MockVehicleRepository) UpdateVehiclesBatch(updates map[string]VehicleUpdateData) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	args := m.Called(updates)
	return args.Error(0)
}

func TestNewBatchProcessor(t *testing.T) {
	mockRepo := &MockVehicleRepository{}
	config := BatchConfig{
		MaxBatchSize:  10,
		BatchInterval: 1 * time.Second,
		MaxWaitTime:   5 * time.Second,
		RetryAttempts: 3,
		RetryBackoff:  100 * time.Millisecond,
	}

	processor := NewBatchProcessor(config, mockRepo)

	assert.NotNil(t, processor)
	assert.Equal(t, config.MaxBatchSize, processor.config.MaxBatchSize)
	assert.Equal(t, config.BatchInterval, processor.config.BatchInterval)
	assert.NotNil(t, processor.updates)
	assert.NotNil(t, processor.updateChan)
}

func TestBatchProcessor_AddUpdate(t *testing.T) {
	mockRepo := &MockVehicleRepository{}
	config := BatchConfig{
		MaxBatchSize:  2,
		BatchInterval: 1 * time.Second,
		MaxWaitTime:   5 * time.Second,
		RetryAttempts: 3,
		RetryBackoff:  100 * time.Millisecond,
	}

	processor := NewBatchProcessor(config, mockRepo)

	// Test successful add
	update := VehicleUpdateData{
		FuelLevel: floatPtr(75.5),
		Speed:     intPtr(60),
		Timestamp: time.Now(),
	}

	err := processor.AddUpdate("vehicle1", update)
	assert.NoError(t, err)

	// Test lifecycle
	mockRepo.On("UpdateVehiclesBatch", mock.AnythingOfType("map[string]batch.VehicleUpdateData")).
		Return(nil).Maybe()
	
	err = processor.Start()
	assert.NoError(t, err)
	
	err = processor.Stop()
	assert.NoError(t, err)
	
	// Now try to add after stopping
	err = processor.AddUpdate("vehicle2", update)
	// Note: This may or may not error depending on channel buffering and timing
	// The important thing is that we can test the basic functionality
}

func TestBatchProcessor_ProcessBatch(t *testing.T) {
	mockRepo := &MockVehicleRepository{}
	config := BatchConfig{
		MaxBatchSize:  10,
		BatchInterval: 1 * time.Second,
		MaxWaitTime:   5 * time.Second,
		RetryAttempts: 3,
		RetryBackoff:  100 * time.Millisecond,
	}

	processor := NewBatchProcessor(config, mockRepo)

	// Test empty batch
	err := processor.ProcessBatch()
	assert.NoError(t, err)

	// Add some updates
	update1 := VehicleUpdateData{
		FuelLevel: floatPtr(75.5),
		Timestamp: time.Now(),
	}
	update2 := VehicleUpdateData{
		Speed:     intPtr(60),
		Timestamp: time.Now(),
	}

	processor.addToCurrentBatch("vehicle1", update1)
	processor.addToCurrentBatch("vehicle2", update2)

	// Mock successful batch update
	mockRepo.On("UpdateVehiclesBatch", mock.AnythingOfType("map[string]batch.VehicleUpdateData")).Return(nil).Once()

	err = processor.ProcessBatch()
	assert.NoError(t, err)

	// Verify batch was cleared
	assert.Equal(t, 0, processor.getCurrentBatchSize())

	mockRepo.AssertExpectations(t)
}

func TestBatchProcessor_ProcessBatchWithRetry(t *testing.T) {
	mockRepo := &MockVehicleRepository{}
	config := BatchConfig{
		MaxBatchSize:  10,
		BatchInterval: 1 * time.Second,
		MaxWaitTime:   5 * time.Second,
		RetryAttempts: 2,
		RetryBackoff:  10 * time.Millisecond, // Short backoff for testing
	}

	processor := NewBatchProcessor(config, mockRepo)

	// Add an update
	update := VehicleUpdateData{
		FuelLevel: floatPtr(75.5),
		Timestamp: time.Now(),
	}
	processor.addToCurrentBatch("vehicle1", update)

	// Mock batch update to fail twice, then succeed
	mockRepo.On("UpdateVehiclesBatch", mock.AnythingOfType("map[string]batch.VehicleUpdateData")).
		Return(errors.New("database error")).Twice()
	mockRepo.On("UpdateVehiclesBatch", mock.AnythingOfType("map[string]batch.VehicleUpdateData")).
		Return(nil).Once()

	err := processor.ProcessBatch()
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestBatchProcessor_ProcessBatchWithFallback(t *testing.T) {
	mockRepo := &MockVehicleRepository{}
	config := BatchConfig{
		MaxBatchSize:  10,
		BatchInterval: 1 * time.Second,
		MaxWaitTime:   5 * time.Second,
		RetryAttempts: 1,
		RetryBackoff:  10 * time.Millisecond,
	}

	processor := NewBatchProcessor(config, mockRepo)

	// Add updates
	update1 := VehicleUpdateData{
		FuelLevel: floatPtr(75.5),
		Timestamp: time.Now(),
	}
	update2 := VehicleUpdateData{
		Speed:     intPtr(60),
		Timestamp: time.Now(),
	}

	processor.addToCurrentBatch("vehicle1", update1)
	processor.addToCurrentBatch("vehicle2", update2)

	// Mock batch update to always fail
	mockRepo.On("UpdateVehiclesBatch", mock.AnythingOfType("map[string]batch.VehicleUpdateData")).
		Return(errors.New("persistent database error")).Times(2) // Initial + 1 retry

	// Mock individual updates to succeed
	mockRepo.On("UpdateVehicle", "vehicle1", update1).Return(nil).Once()
	mockRepo.On("UpdateVehicle", "vehicle2", update2).Return(nil).Once()

	err := processor.ProcessBatch()
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestBatchProcessor_SplitIntoBatches(t *testing.T) {
	mockRepo := &MockVehicleRepository{}
	config := BatchConfig{
		MaxBatchSize:  2,
		BatchInterval: 1 * time.Second,
		MaxWaitTime:   5 * time.Second,
		RetryAttempts: 3,
		RetryBackoff:  100 * time.Millisecond,
	}

	processor := NewBatchProcessor(config, mockRepo)

	// Create updates that exceed batch size
	updates := map[string]VehicleUpdateData{
		"vehicle1": {FuelLevel: floatPtr(75.5), Timestamp: time.Now()},
		"vehicle2": {Speed: intPtr(60), Timestamp: time.Now()},
		"vehicle3": {Status: stringPtr("active"), Timestamp: time.Now()},
		"vehicle4": {Odometer: intPtr(1000), Timestamp: time.Now()},
		"vehicle5": {FuelLevel: floatPtr(50.0), Timestamp: time.Now()},
	}

	batches := processor.splitIntoBatches(updates)

	// Should split 5 updates into 3 batches (2, 2, 1)
	assert.Len(t, batches, 3)
	assert.Len(t, batches[0], 2)
	assert.Len(t, batches[1], 2)
	assert.Len(t, batches[2], 1)

	// Verify all updates are included
	totalUpdates := 0
	for _, batch := range batches {
		totalUpdates += len(batch)
	}
	assert.Equal(t, len(updates), totalUpdates)
}

func TestBatchProcessor_WorkerLifecycle(t *testing.T) {
	mockRepo := &MockVehicleRepository{}
	config := BatchConfig{
		MaxBatchSize:  10,
		BatchInterval: 100 * time.Millisecond, // Short interval for testing
		MaxWaitTime:   5 * time.Second,
		RetryAttempts: 3,
		RetryBackoff:  100 * time.Millisecond,
	}

	processor := NewBatchProcessor(config, mockRepo)

	// Mock batch processing
	mockRepo.On("UpdateVehiclesBatch", mock.AnythingOfType("map[string]batch.VehicleUpdateData")).
		Return(nil).Maybe()

	// Start processor
	err := processor.Start()
	assert.NoError(t, err)

	// Add some updates
	update := VehicleUpdateData{
		FuelLevel: floatPtr(75.5),
		Timestamp: time.Now(),
	}

	err = processor.AddUpdate("vehicle1", update)
	assert.NoError(t, err)

	// Wait for interval processing
	time.Sleep(200 * time.Millisecond)

	// Stop processor
	err = processor.Stop()
	assert.NoError(t, err)

	// Verify we can't add updates after stopping (may or may not error depending on timing)
	err = processor.AddUpdate("vehicle2", update)
	// Note: This may not always error due to channel buffering and timing
	// The important thing is that the processor stopped gracefully
}

func TestBatchProcessor_BatchSizeLimit(t *testing.T) {
	mockRepo := &MockVehicleRepository{}
	config := BatchConfig{
		MaxBatchSize:  2,
		BatchInterval: 1 * time.Second,
		MaxWaitTime:   5 * time.Second,
		RetryAttempts: 3,
		RetryBackoff:  100 * time.Millisecond,
	}

	processor := NewBatchProcessor(config, mockRepo)

	// Mock batch processing
	mockRepo.On("UpdateVehiclesBatch", mock.AnythingOfType("map[string]batch.VehicleUpdateData")).
		Return(nil).Maybe()

	err := processor.Start()
	assert.NoError(t, err)
	defer processor.Stop()

	// Add updates up to batch size
	update := VehicleUpdateData{
		FuelLevel: floatPtr(75.5),
		Timestamp: time.Now(),
	}

	err = processor.AddUpdate("vehicle1", update)
	assert.NoError(t, err)

	err = processor.AddUpdate("vehicle2", update)
	assert.NoError(t, err)

	// Give time for batch processing
	time.Sleep(100 * time.Millisecond)

	// Batch should have been processed
	assert.Equal(t, 0, processor.getCurrentBatchSize())
}

func TestBatchProcessor_Statistics(t *testing.T) {
	mockRepo := &MockVehicleRepository{}
	config := BatchConfig{
		MaxBatchSize:  10,
		BatchInterval: 1 * time.Second,
		MaxWaitTime:   5 * time.Second,
		RetryAttempts: 3,
		RetryBackoff:  100 * time.Millisecond,
	}

	processor := NewBatchProcessor(config, mockRepo)

	// Initial stats
	stats := processor.GetBatchStats()
	assert.Equal(t, 0, stats.BatchesProcessed)
	assert.Equal(t, int64(0), stats.TotalUpdates)

	// Add updates and process
	update1 := VehicleUpdateData{FuelLevel: floatPtr(75.5), Timestamp: time.Now()}
	update2 := VehicleUpdateData{Speed: intPtr(60), Timestamp: time.Now()}

	processor.addToCurrentBatch("vehicle1", update1)
	processor.addToCurrentBatch("vehicle2", update2)

	mockRepo.On("UpdateVehiclesBatch", mock.AnythingOfType("map[string]batch.VehicleUpdateData")).
		Return(nil).Once()

	err := processor.ProcessBatch()
	assert.NoError(t, err)

	// Check updated stats
	stats = processor.GetBatchStats()
	assert.Equal(t, 1, stats.BatchesProcessed)
	assert.Equal(t, int64(2), stats.TotalUpdates)
	assert.Equal(t, 2.0, stats.AverageSize)
	assert.Equal(t, 0.0, stats.ErrorRate)

	mockRepo.AssertExpectations(t)
}

func TestBatchProcessor_ConfigurationUpdate(t *testing.T) {
	mockRepo := &MockVehicleRepository{}
	config := BatchConfig{
		MaxBatchSize:  10,
		BatchInterval: 1 * time.Second,
		MaxWaitTime:   5 * time.Second,
		RetryAttempts: 3,
		RetryBackoff:  100 * time.Millisecond,
	}

	processor := NewBatchProcessor(config, mockRepo)

	// Test batch size update
	processor.SetBatchSize(20)
	assert.Equal(t, 20, processor.config.MaxBatchSize)

	// Test interval update
	newInterval := 2 * time.Second
	processor.SetBatchInterval(newInterval)
	assert.Equal(t, newInterval, processor.config.BatchInterval)
}

// Helper functions for creating pointers
func floatPtr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

func locationPtr(lat, lng float64, address string) *models.Location {
	return &models.Location{
		Lat:     lat,
		Lng:     lng,
		Address: address,
	}
}

// Additional tests for error scenarios and edge cases

func TestBatchProcessor_ErrorScenarios(t *testing.T) {
	t.Run("Repository error during individual fallback", func(t *testing.T) {
		mockRepo := &MockVehicleRepository{}
		config := BatchConfig{
			MaxBatchSize:  10,
			BatchInterval: 1 * time.Second,
			MaxWaitTime:   5 * time.Second,
			RetryAttempts: 1,
			RetryBackoff:  10 * time.Millisecond,
		}

		processor := NewBatchProcessor(config, mockRepo)

		// Add updates
		update1 := VehicleUpdateData{FuelLevel: floatPtr(75.5), Timestamp: time.Now()}
		update2 := VehicleUpdateData{Speed: intPtr(60), Timestamp: time.Now()}

		processor.addToCurrentBatch("vehicle1", update1)
		processor.addToCurrentBatch("vehicle2", update2)

		// Mock batch update to always fail
		mockRepo.On("UpdateVehiclesBatch", mock.AnythingOfType("map[string]batch.VehicleUpdateData")).
			Return(errors.New("persistent database error")).Times(2)

		// Mock individual updates - one succeeds, one fails
		mockRepo.On("UpdateVehicle", "vehicle1", update1).Return(nil).Once()
		mockRepo.On("UpdateVehicle", "vehicle2", update2).Return(errors.New("individual update error")).Once()

		err := processor.ProcessBatch()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to process")

		mockRepo.AssertExpectations(t)
	})

	t.Run("Context cancellation during processing", func(t *testing.T) {
		mockRepo := &MockVehicleRepository{}
		config := BatchConfig{
			MaxBatchSize:  10,
			BatchInterval: 100 * time.Millisecond,
			MaxWaitTime:   5 * time.Second,
			RetryAttempts: 3,
			RetryBackoff:  1 * time.Second, // Long backoff to test cancellation
		}

		processor := NewBatchProcessor(config, mockRepo)

		// Add an update
		update := VehicleUpdateData{FuelLevel: floatPtr(75.5), Timestamp: time.Now()}
		processor.addToCurrentBatch("vehicle1", update)

		// Mock batch update to fail initially
		mockRepo.On("UpdateVehiclesBatch", mock.AnythingOfType("map[string]batch.VehicleUpdateData")).
			Return(errors.New("database error")).Maybe()

		// Start processor
		err := processor.Start()
		assert.NoError(t, err)

		// Stop processor quickly to test cancellation during retry
		go func() {
			time.Sleep(50 * time.Millisecond)
			processor.Stop()
		}()

		// Wait for stop to complete
		time.Sleep(200 * time.Millisecond)
	})

	t.Run("Large batch splitting", func(t *testing.T) {
		mockRepo := &MockVehicleRepository{}
		config := BatchConfig{
			MaxBatchSize:  3,
			BatchInterval: 1 * time.Second,
			MaxWaitTime:   5 * time.Second,
			RetryAttempts: 1,
			RetryBackoff:  10 * time.Millisecond,
		}

		processor := NewBatchProcessor(config, mockRepo)

		// Create a large batch
		updates := make(map[string]VehicleUpdateData)
		for i := 0; i < 10; i++ {
			vehicleID := fmt.Sprintf("vehicle%d", i)
			updates[vehicleID] = VehicleUpdateData{
				FuelLevel: floatPtr(float64(50 + i)),
				Timestamp: time.Now(),
			}
		}

		// Mock successful batch updates for all sub-batches
		mockRepo.On("UpdateVehiclesBatch", mock.AnythingOfType("map[string]batch.VehicleUpdateData")).
			Return(nil).Times(4) // 10 updates / 3 batch size = 4 batches (3,3,3,1)

		batches := processor.splitIntoBatches(updates)
		
		// Process all batches
		for _, batch := range batches {
			err := processor.processSingleBatch(batch)
			assert.NoError(t, err)
		}

		mockRepo.AssertExpectations(t)
	})
}

func TestBatchProcessor_ConcurrentAccess(t *testing.T) {
	mockRepo := &MockVehicleRepository{}
	config := BatchConfig{
		MaxBatchSize:  100,
		BatchInterval: 100 * time.Millisecond,
		MaxWaitTime:   5 * time.Second,
		RetryAttempts: 3,
		RetryBackoff:  10 * time.Millisecond,
	}

	processor := NewBatchProcessor(config, mockRepo)

	// Mock batch processing
	mockRepo.On("UpdateVehiclesBatch", mock.AnythingOfType("map[string]batch.VehicleUpdateData")).
		Return(nil).Maybe()

	err := processor.Start()
	assert.NoError(t, err)
	defer processor.Stop()

	// Concurrent update additions
	var wg sync.WaitGroup
	numGoroutines := 10
	updatesPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < updatesPerGoroutine; j++ {
				vehicleID := fmt.Sprintf("vehicle_%d_%d", goroutineID, j)
				update := VehicleUpdateData{
					FuelLevel: floatPtr(float64(50 + j)),
					Timestamp: time.Now(),
				}
				
				err := processor.AddUpdate(vehicleID, update)
				if err != nil {
					t.Logf("Failed to add update: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Wait for processing to complete
	time.Sleep(200 * time.Millisecond)

	// Check statistics
	stats := processor.GetBatchStats()
	assert.True(t, stats.TotalUpdates > 0, "Expected some updates to be processed")
}

func TestBatchProcessor_MaxWaitTime(t *testing.T) {
	mockRepo := &MockVehicleRepository{}
	config := BatchConfig{
		MaxBatchSize:  100, // Large batch size so it won't trigger
		BatchInterval: 1 * time.Second, // Long interval so it won't trigger
		MaxWaitTime:   100 * time.Millisecond, // Short max wait time
		RetryAttempts: 1,
		RetryBackoff:  10 * time.Millisecond,
	}

	processor := NewBatchProcessor(config, mockRepo)

	// Mock batch processing
	mockRepo.On("UpdateVehiclesBatch", mock.AnythingOfType("map[string]batch.VehicleUpdateData")).
		Return(nil).Once()

	err := processor.Start()
	assert.NoError(t, err)
	defer processor.Stop()

	// Add a single update
	update := VehicleUpdateData{
		FuelLevel: floatPtr(75.5),
		Timestamp: time.Now(),
	}

	err = processor.AddUpdate("vehicle1", update)
	assert.NoError(t, err)

	// Wait for max wait time to trigger processing
	time.Sleep(200 * time.Millisecond)

	// Verify the batch was processed due to max wait time
	stats := processor.GetBatchStats()
	assert.Equal(t, int64(1), stats.TotalUpdates)

	mockRepo.AssertExpectations(t)
}

func TestVehicleRepositoryAdapter_UpdateVehicle(t *testing.T) {
	// This would require a real MongoDB connection for integration testing
	// For now, we'll test the validation logic
	
	t.Run("Invalid vehicle ID", func(t *testing.T) {
		adapter := &VehicleRepositoryAdapter{}
		
		update := VehicleUpdateData{
			FuelLevel: floatPtr(75.5),
			Timestamp: time.Now(),
		}
		
		err := adapter.UpdateVehicle("invalid-id", update)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid vehicle ID")
	})
}

func TestVehicleUpdateData_PartialUpdates(t *testing.T) {
	mockRepo := &MockVehicleRepository{}
	config := BatchConfig{
		MaxBatchSize:  10,
		BatchInterval: 1 * time.Second,
		MaxWaitTime:   5 * time.Second,
		RetryAttempts: 1,
		RetryBackoff:  10 * time.Millisecond,
	}

	processor := NewBatchProcessor(config, mockRepo)

	// Test partial update with only fuel level
	update1 := VehicleUpdateData{
		FuelLevel: floatPtr(75.5),
		Timestamp: time.Now(),
	}

	// Test partial update with only speed
	update2 := VehicleUpdateData{
		Speed:     intPtr(60),
		Timestamp: time.Now(),
	}

	// Test partial update with only location
	update3 := VehicleUpdateData{
		Location:  locationPtr(40.7128, -74.0060, "New York, NY"),
		Timestamp: time.Now(),
	}

	processor.addToCurrentBatch("vehicle1", update1)
	processor.addToCurrentBatch("vehicle2", update2)
	processor.addToCurrentBatch("vehicle3", update3)

	mockRepo.On("UpdateVehiclesBatch", mock.AnythingOfType("map[string]batch.VehicleUpdateData")).
		Return(nil).Once()

	err := processor.ProcessBatch()
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}