package websocket

import (
	"time"

	"github.com/gorilla/websocket"
)

// VehicleFilters defines filtering criteria for vehicle updates
type VehicleFilters struct {
	VehicleIDs []string `json:"vehicleIds,omitempty"`
	Statuses   []string `json:"statuses,omitempty"`
	Drivers    []string `json:"drivers,omitempty"`
	AlertTypes []string `json:"alertTypes,omitempty"`
}

// VehicleUpdate represents a vehicle update message
type VehicleUpdate struct {
	VehicleID  string                 `json:"vehicleId"`
	UpdateType string                 `json:"updateType"` // "location", "fuel", "status", "alert"
	Data       map[string]interface{} `json:"data"`
	Timestamp  time.Time              `json:"timestamp"`
	Priority   string                 `json:"priority"` // "low", "medium", "high", "critical"
}

// Client represents a WebSocket client connection
type Client struct {
	ID         string
	Conn       *websocket.Conn
	Filters    VehicleFilters
	Send       chan VehicleUpdate
	LastPing   time.Time
	IsActive   bool
}

// WebSocketManager interface defines the contract for WebSocket management
type WebSocketManager interface {
	RegisterClient(clientID string, conn *websocket.Conn, filters VehicleFilters) error
	UnregisterClient(clientID string) error
	BroadcastVehicleUpdate(vehicleID string, update VehicleUpdate) error
	BroadcastBatchUpdates(updates []VehicleUpdate) error
	GetConnectedClients() int
	Start() error
	Stop() error
	GetClientStats() ClientStats
}

// ClientStats provides statistics about connected clients
type ClientStats struct {
	TotalClients    int `json:"totalClients"`
	ActiveClients   int `json:"activeClients"`
	InactiveClients int `json:"inactiveClients"`
}

// Message types for WebSocket communication
const (
	MessageTypeVehicleUpdate = "vehicle_update"
	MessageTypeBatchUpdate   = "batch_update"
	MessageTypePing          = "ping"
	MessageTypePong          = "pong"
	MessageTypeError         = "error"
)

// Priority levels for message handling
const (
	PriorityLow      = "low"
	PriorityMedium   = "medium"
	PriorityHigh     = "high"
	PriorityCritical = "critical"
)