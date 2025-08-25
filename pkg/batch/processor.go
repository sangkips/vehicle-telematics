package batch

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"fleet-backend/internal/websocket"
)

// DefaultBatchProcessor implements the BatchProcessor interface
type DefaultBatchProcessor struct {
	config     BatchConfig
	repository VehicleRepository
	wsManager  websocket.WebSocketManager
	
	// Internal state
	updates    map[string]VehicleUpdateData
	updatesMux sync.RWMutex
	
	// Worker control
	ctx        context.Context
	cancel     context.CancelFunc
	workerWg   sync.WaitGroup
	
	// Statistics
	stats      BatchStats
	statsMux   sync.RWMutex
	
	// Channels for communication
	updateChan chan updateRequest
	stopChan   chan struct{}
}

type updateRequest struct {
	vehicleID string
	update    VehicleUpdateData
}

// NewBatchProcessor creates a new batch processor with the given configuration
func NewBatchProcessor(config BatchConfig, repository VehicleRepository) *DefaultBatchProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &DefaultBatchProcessor{
		config:     config,
		repository: repository,
		updates:    make(map[string]VehicleUpdateData),
		ctx:        ctx,
		cancel:     cancel,
		updateChan: make(chan updateRequest, config.MaxBatchSize*2), // Buffer for updates
		stopChan:   make(chan struct{}),
		stats: BatchStats{
			LastProcessedAt: time.Now(),
		},
	}
}

// NewBatchProcessorWithWebSocket creates a new batch processor with WebSocket broadcasting support
func NewBatchProcessorWithWebSocket(config BatchConfig, repository VehicleRepository, wsManager websocket.WebSocketManager) *DefaultBatchProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &DefaultBatchProcessor{
		config:     config,
		repository: repository,
		wsManager:  wsManager,
		updates:    make(map[string]VehicleUpdateData),
		ctx:        ctx,
		cancel:     cancel,
		updateChan: make(chan updateRequest, config.MaxBatchSize*2), // Buffer for updates
		stopChan:   make(chan struct{}),
		stats: BatchStats{
			LastProcessedAt: time.Now(),
		},
	}
}

// SetWebSocketManager sets the WebSocket manager for broadcasting updates
func (bp *DefaultBatchProcessor) SetWebSocketManager(wsManager websocket.WebSocketManager) {
	bp.wsManager = wsManager
}

// AddUpdate adds a vehicle update to the batch queue
func (bp *DefaultBatchProcessor) AddUpdate(vehicleID string, update VehicleUpdateData) error {
	select {
	case bp.updateChan <- updateRequest{vehicleID: vehicleID, update: update}:
		return nil
	case <-bp.ctx.Done():
		return fmt.Errorf("batch processor is stopped")
	default:
		return fmt.Errorf("update queue is full, dropping update for vehicle %s", vehicleID)
	}
}

// ProcessBatch processes the current batch of updates
func (bp *DefaultBatchProcessor) ProcessBatch() error {
	bp.updatesMux.Lock()
	currentUpdates := make(map[string]VehicleUpdateData)
	for k, v := range bp.updates {
		currentUpdates[k] = v
	}
	bp.updates = make(map[string]VehicleUpdateData) // Clear the updates map
	bp.updatesMux.Unlock()
	
	if len(currentUpdates) == 0 {
		return nil
	}
	
	startTime := time.Now()
	
	// Split into smaller batches if necessary
	batches := bp.splitIntoBatches(currentUpdates)
	
	var totalErrors int
	for _, batch := range batches {
		if err := bp.processSingleBatch(batch); err != nil {
			log.Printf("Error processing batch: %v", err)
			totalErrors++
		}
	}
	
	// Update statistics
	bp.updateStats(len(batches), len(currentUpdates), totalErrors, time.Since(startTime))
	
	if totalErrors > 0 {
		return fmt.Errorf("failed to process %d out of %d batches", totalErrors, len(batches))
	}
	
	return nil
}

// processSingleBatch processes a single batch with retry logic
func (bp *DefaultBatchProcessor) processSingleBatch(batch map[string]VehicleUpdateData) error {
	for attempt := 0; attempt <= bp.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoffDuration := time.Duration(math.Pow(2, float64(attempt-1))) * bp.config.RetryBackoff
			log.Printf("Retrying batch processing after %v (attempt %d/%d)", backoffDuration, attempt, bp.config.RetryAttempts)
			
			select {
			case <-time.After(backoffDuration):
			case <-bp.ctx.Done():
				return fmt.Errorf("batch processor stopped during retry")
			}
		}
		
		err := bp.repository.UpdateVehiclesBatch(batch)
		if err == nil {
			// Broadcast updates via WebSocket after successful database update
			bp.broadcastBatchUpdates(batch)
			return nil // Success
		}
		
		log.Printf("Batch processing attempt %d failed: %v", attempt+1, err)
		
		// If this is the last attempt, we'll fall back to individual updates
		if attempt == bp.config.RetryAttempts {
			log.Printf("All batch retries failed, falling back to individual updates")
			return bp.fallbackToIndividualUpdates(batch)
		}
	}
	
	// This should never be reached due to the return in the loop
	return fmt.Errorf("unexpected error in batch processing")
}

// fallbackToIndividualUpdates processes updates individually when batch processing fails
func (bp *DefaultBatchProcessor) fallbackToIndividualUpdates(batch map[string]VehicleUpdateData) error {
	var errors []string
	
	for vehicleID, update := range batch {
		if err := bp.repository.UpdateVehicle(vehicleID, update); err != nil {
			errors = append(errors, fmt.Sprintf("vehicle %s: %v", vehicleID, err))
			bp.incrementFailedUpdates()
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("individual update failures: %v", errors)
	}
	
	return nil
}

// splitIntoBatches splits a large update map into smaller batches
func (bp *DefaultBatchProcessor) splitIntoBatches(updates map[string]VehicleUpdateData) []map[string]VehicleUpdateData {
	if len(updates) <= bp.config.MaxBatchSize {
		return []map[string]VehicleUpdateData{updates}
	}
	
	var batches []map[string]VehicleUpdateData
	currentBatch := make(map[string]VehicleUpdateData)
	
	for vehicleID, update := range updates {
		currentBatch[vehicleID] = update
		
		if len(currentBatch) >= bp.config.MaxBatchSize {
			batches = append(batches, currentBatch)
			currentBatch = make(map[string]VehicleUpdateData)
		}
	}
	
	// Add remaining updates
	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}
	
	return batches
}

// Start starts the batch processing worker
func (bp *DefaultBatchProcessor) Start() error {
	bp.workerWg.Add(1)
	go bp.worker()
	log.Println("Batch processor started")
	return nil
}

// Stop stops the batch processing worker
func (bp *DefaultBatchProcessor) Stop() error {
	bp.cancel()
	close(bp.stopChan)
	bp.workerWg.Wait()
	log.Println("Batch processor stopped")
	return nil
}

// worker is the main worker goroutine that processes updates
func (bp *DefaultBatchProcessor) worker() {
	defer bp.workerWg.Done()
	
	ticker := time.NewTicker(bp.config.BatchInterval)
	defer ticker.Stop()
	
	maxWaitTimer := time.NewTimer(bp.config.MaxWaitTime)
	defer maxWaitTimer.Stop()
	
	for {
		select {
		case update := <-bp.updateChan:
			bp.addToCurrentBatch(update.vehicleID, update.update)
			
			// Reset max wait timer when we receive an update
			if !maxWaitTimer.Stop() {
				<-maxWaitTimer.C
			}
			maxWaitTimer.Reset(bp.config.MaxWaitTime)
			
			// Check if batch is full
			if bp.getCurrentBatchSize() >= bp.config.MaxBatchSize {
				if err := bp.ProcessBatch(); err != nil {
					log.Printf("Error processing full batch: %v", err)
				}
			}
			
		case <-ticker.C:
			// Process batch on interval
			if err := bp.ProcessBatch(); err != nil {
				log.Printf("Error processing interval batch: %v", err)
			}
			
		case <-maxWaitTimer.C:
			// Process batch when max wait time is reached
			if err := bp.ProcessBatch(); err != nil {
				log.Printf("Error processing max wait batch: %v", err)
			}
			maxWaitTimer.Reset(bp.config.MaxWaitTime)
			
		case <-bp.ctx.Done():
			// Process remaining updates before stopping
			if err := bp.ProcessBatch(); err != nil {
				log.Printf("Error processing final batch: %v", err)
			}
			return
		}
	}
}

// addToCurrentBatch adds an update to the current batch
func (bp *DefaultBatchProcessor) addToCurrentBatch(vehicleID string, update VehicleUpdateData) {
	bp.updatesMux.Lock()
	defer bp.updatesMux.Unlock()
	bp.updates[vehicleID] = update
}

// getCurrentBatchSize returns the current batch size
func (bp *DefaultBatchProcessor) getCurrentBatchSize() int {
	bp.updatesMux.RLock()
	defer bp.updatesMux.RUnlock()
	return len(bp.updates)
}

// SetBatchSize updates the batch size configuration
func (bp *DefaultBatchProcessor) SetBatchSize(size int) {
	bp.config.MaxBatchSize = size
}

// SetBatchInterval updates the batch interval configuration
func (bp *DefaultBatchProcessor) SetBatchInterval(interval time.Duration) {
	bp.config.BatchInterval = interval
}

// GetBatchStats returns current batch processing statistics
func (bp *DefaultBatchProcessor) GetBatchStats() BatchStats {
	bp.statsMux.RLock()
	defer bp.statsMux.RUnlock()
	return bp.stats
}

// updateStats updates the batch processing statistics
func (bp *DefaultBatchProcessor) updateStats(batchCount, updateCount, errorCount int, processingTime time.Duration) {
	bp.statsMux.Lock()
	defer bp.statsMux.Unlock()
	
	bp.stats.BatchesProcessed += batchCount
	bp.stats.TotalUpdates += int64(updateCount)
	bp.stats.LastProcessedAt = time.Now()
	bp.stats.ProcessingTime = processingTime
	
	// Calculate average batch size
	if bp.stats.BatchesProcessed > 0 {
		bp.stats.AverageSize = float64(bp.stats.TotalUpdates) / float64(bp.stats.BatchesProcessed)
	}
	
	// Calculate error rate
	if bp.stats.TotalUpdates > 0 {
		bp.stats.ErrorRate = float64(bp.stats.FailedUpdates) / float64(bp.stats.TotalUpdates)
	}
}

// incrementFailedUpdates increments the failed updates counter
func (bp *DefaultBatchProcessor) incrementFailedUpdates() {
	bp.statsMux.Lock()
	defer bp.statsMux.Unlock()
	bp.stats.FailedUpdates++
}

// broadcastBatchUpdates broadcasts vehicle updates via WebSocket after successful batch processing
func (bp *DefaultBatchProcessor) broadcastBatchUpdates(batch map[string]VehicleUpdateData) {
	if bp.wsManager == nil {
		return // No WebSocket manager configured
	}

	var updates []websocket.VehicleUpdate
	
	for vehicleID, updateData := range batch {
		// Convert VehicleUpdateData to VehicleUpdate format
		wsUpdate := bp.convertToWebSocketUpdate(vehicleID, updateData)
		updates = append(updates, wsUpdate)
	}
	
	// Broadcast all updates in the batch
	if err := bp.wsManager.BroadcastBatchUpdates(updates); err != nil {
		log.Printf("Failed to broadcast batch updates via WebSocket: %v", err)
	}
}

// convertToWebSocketUpdate converts VehicleUpdateData to WebSocket VehicleUpdate format
func (bp *DefaultBatchProcessor) convertToWebSocketUpdate(vehicleID string, updateData VehicleUpdateData) websocket.VehicleUpdate {
	data := make(map[string]interface{})
	updateType := "batch_update"
	priority := websocket.PriorityMedium // Default priority for batch updates
	
	// Add fields that were updated
	if updateData.FuelLevel != nil {
		data["fuelLevel"] = *updateData.FuelLevel
		updateType = "fuel"
		
		// Check for critical fuel levels
		if *updateData.FuelLevel < 10 {
			priority = websocket.PriorityCritical
		} else if *updateData.FuelLevel < 20 {
			priority = websocket.PriorityHigh
		}
	}
	
	if updateData.Location != nil {
		data["location"] = *updateData.Location
		if updateType == "batch_update" {
			updateType = "location"
		}
	}
	
	if updateData.Speed != nil {
		data["speed"] = *updateData.Speed
		if updateType == "batch_update" {
			updateType = "speed"
		}
		
		// Check for speeding
		if *updateData.Speed > 80 {
			priority = websocket.PriorityHigh
			updateType = "alert"
			data["alertType"] = "speeding"
		}
	}
	
	if updateData.Status != nil {
		data["status"] = *updateData.Status
		if updateType == "batch_update" {
			updateType = "status"
		}
		
		// Critical status changes get high priority
		if *updateData.Status == "maintenance" || *updateData.Status == "offline" {
			priority = websocket.PriorityHigh
		}
	}
	
	if updateData.Odometer != nil {
		data["odometer"] = *updateData.Odometer
		if updateType == "batch_update" {
			updateType = "odometer"
		}
	}
	
	return websocket.VehicleUpdate{
		VehicleID:  vehicleID,
		UpdateType: updateType,
		Data:       data,
		Timestamp:  updateData.Timestamp,
		Priority:   priority,
	}
}