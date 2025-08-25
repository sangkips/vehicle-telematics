package telemetry

import (
	"context"
	"sync"
	"time"
)

type VehicleState string

const (
	StateActive      VehicleState = "active"      // Moving, frequent updates
	StateIdle        VehicleState = "idle"        // Stopped but engine on
	StateParked      VehicleState = "parked"      // Engine off, parked
	StateMaintenance VehicleState = "maintenance" // In maintenance
	StateOffline     VehicleState = "offline"     // No connection
)

type UpdateFrequency struct {
	Interval    time.Duration
	MaxInterval time.Duration
	MinInterval time.Duration
}

// AdaptiveScheduler manages update frequencies based on vehicle state
type AdaptiveScheduler struct {
	frequencies map[VehicleState]UpdateFrequency
	vehicles    map[string]*VehicleSchedule
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

type VehicleSchedule struct {
	VehicleID       string
	CurrentState    VehicleState
	LastUpdate      time.Time
	NextUpdate      time.Time
	UpdateInterval  time.Duration
	ConsecutiveIdle int
	ticker          *time.Ticker
	stopChan        chan struct{}
}

func NewAdaptiveScheduler() *AdaptiveScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &AdaptiveScheduler{
		frequencies: map[VehicleState]UpdateFrequency{
			StateActive:      {Interval: 30 * time.Second, MinInterval: 15 * time.Second, MaxInterval: 2 * time.Minute},
			StateIdle:        {Interval: 2 * time.Minute, MinInterval: 1 * time.Minute, MaxInterval: 5 * time.Minute},
			StateParked:      {Interval: 10 * time.Minute, MinInterval: 5 * time.Minute, MaxInterval: 30 * time.Minute},
			StateMaintenance: {Interval: 30 * time.Minute, MinInterval: 15 * time.Minute, MaxInterval: 2 * time.Hour},
			StateOffline:     {Interval: 1 * time.Hour, MinInterval: 30 * time.Minute, MaxInterval: 6 * time.Hour},
		},
		vehicles: make(map[string]*VehicleSchedule),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// UpdateVehicleState updates the state and adjusts update frequency
func (as *AdaptiveScheduler) UpdateVehicleState(vehicleID string, newState VehicleState, callback func(string)) {
	as.mu.Lock()
	defer as.mu.Unlock()
	
	schedule, exists := as.vehicles[vehicleID]
	if !exists {
		schedule = &VehicleSchedule{
			VehicleID:    vehicleID,
			CurrentState: newState,
			LastUpdate:   time.Now(),
			stopChan:     make(chan struct{}),
		}
		as.vehicles[vehicleID] = schedule
	}
	
	// Update state and adjust frequency if changed
	if schedule.CurrentState != newState {
		schedule.CurrentState = newState
		schedule.ConsecutiveIdle = 0
		as.adjustUpdateFrequency(schedule)
		as.startScheduler(schedule, callback)
	}
}

// adjustUpdateFrequency calculates optimal update interval based on state
func (as *AdaptiveScheduler) adjustUpdateFrequency(schedule *VehicleSchedule) {
	freq := as.frequencies[schedule.CurrentState]
	
	// Increase interval for consecutive idle states
	if schedule.CurrentState == StateIdle || schedule.CurrentState == StateParked {
		schedule.ConsecutiveIdle++
		multiplier := 1.0 + float64(schedule.ConsecutiveIdle)*0.2 // 20% increase per idle cycle
		newInterval := time.Duration(float64(freq.Interval) * multiplier)
		
		if newInterval > freq.MaxInterval {
			newInterval = freq.MaxInterval
		}
		schedule.UpdateInterval = newInterval
	} else {
		schedule.UpdateInterval = freq.Interval
	}
	
	schedule.NextUpdate = time.Now().Add(schedule.UpdateInterval)
}

// startScheduler starts the update scheduler for a vehicle
func (as *AdaptiveScheduler) startScheduler(schedule *VehicleSchedule, callback func(string)) {
	// Stop existing ticker if running
	if schedule.ticker != nil {
		schedule.ticker.Stop()
		close(schedule.stopChan)
		schedule.stopChan = make(chan struct{})
	}
	
	schedule.ticker = time.NewTicker(schedule.UpdateInterval)
	
	go func() {
		for {
			select {
			case <-schedule.ticker.C:
				callback(schedule.VehicleID)
				schedule.LastUpdate = time.Now()
				
				// Readjust frequency based on current state
				as.mu.Lock()
				as.adjustUpdateFrequency(schedule)
				schedule.ticker.Reset(schedule.UpdateInterval)
				as.mu.Unlock()
				
			case <-schedule.stopChan:
				return
			case <-as.ctx.Done():
				return
			}
		}
	}()
}

// GetVehicleSchedule returns the current schedule for a vehicle
func (as *AdaptiveScheduler) GetVehicleSchedule(vehicleID string) (*VehicleSchedule, bool) {
	as.mu.RLock()
	defer as.mu.RUnlock()
	schedule, exists := as.vehicles[vehicleID]
	return schedule, exists
}

// Stop stops all schedulers
func (as *AdaptiveScheduler) Stop() {
	as.cancel()
	as.mu.Lock()
	defer as.mu.Unlock()
	
	for _, schedule := range as.vehicles {
		if schedule.ticker != nil {
			schedule.ticker.Stop()
			close(schedule.stopChan)
		}
	}
}