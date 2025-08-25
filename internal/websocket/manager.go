package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Manager implements the WebSocketManager interface
type Manager struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan VehicleUpdate
	mutex      sync.RWMutex
	upgrader   websocket.Upgrader
	done       chan struct{}
}

// NewManager creates a new WebSocket manager
func NewManager() *Manager {
	return &Manager{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan VehicleUpdate, 1000), // Buffer for high-frequency updates
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement proper origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		done: make(chan struct{}),
	}
}

// Start begins the WebSocket manager's main loop
func (m *Manager) Start() error {
	go m.run()
	log.Println("WebSocket manager started")
	return nil
}

// Stop gracefully shuts down the WebSocket manager
func (m *Manager) Stop() error {
	close(m.done)
	
	// Close all client connections
	m.mutex.Lock()
	for _, client := range m.clients {
		close(client.Send)
		if client.Conn != nil {
			client.Conn.Close()
		}
	}
	m.mutex.Unlock()
	
	log.Println("WebSocket manager stopped")
	return nil
}

// run is the main event loop for the WebSocket manager
func (m *Manager) run() {
	ticker := time.NewTicker(30 * time.Second) // Health check interval
	defer ticker.Stop()

	for {
		select {
		case client := <-m.register:
			m.mutex.Lock()
			m.clients[client.ID] = client
			m.mutex.Unlock()
			log.Printf("Client %s registered", client.ID)
			go m.handleClient(client)

		case client := <-m.unregister:
			m.mutex.Lock()
			if _, ok := m.clients[client.ID]; ok {
				delete(m.clients, client.ID)
				close(client.Send)
				if client.Conn != nil {
					client.Conn.Close()
				}
			}
			m.mutex.Unlock()
			log.Printf("Client %s unregistered", client.ID)

		case update := <-m.broadcast:
			m.broadcastToClients(update)

		case <-ticker.C:
			m.healthCheck()

		case <-m.done:
			return
		}
	}
}

// RegisterClient registers a new WebSocket client
func (m *Manager) RegisterClient(clientID string, conn *websocket.Conn, filters VehicleFilters) error {
	client := &Client{
		ID:       clientID,
		Conn:     conn,
		Filters:  filters,
		Send:     make(chan VehicleUpdate, 256),
		LastPing: time.Now(),
		IsActive: true,
	}

	m.register <- client
	return nil
}

// UnregisterClient removes a WebSocket client
func (m *Manager) UnregisterClient(clientID string) error {
	m.mutex.RLock()
	client, exists := m.clients[clientID]
	m.mutex.RUnlock()

	if exists {
		m.unregister <- client
	}
	return nil
}

// BroadcastVehicleUpdate sends a single vehicle update to relevant clients
func (m *Manager) BroadcastVehicleUpdate(vehicleID string, update VehicleUpdate) error {
	select {
	case m.broadcast <- update:
		return nil
	default:
		return fmt.Errorf("broadcast channel full, dropping update for vehicle %s", vehicleID)
	}
}

// BroadcastBatchUpdates sends multiple vehicle updates efficiently
func (m *Manager) BroadcastBatchUpdates(updates []VehicleUpdate) error {
	// Sort updates by priority for efficient processing
	priorityOrder := map[string]int{
		PriorityCritical: 0,
		PriorityHigh:     1,
		PriorityMedium:   2,
		PriorityLow:      3,
	}

	// Process critical and high priority updates first
	for _, update := range updates {
		priority := priorityOrder[update.Priority]
		if priority <= 1 { // Critical and High priority
			select {
			case m.broadcast <- update:
			default:
				log.Printf("Dropping high priority update for vehicle %s due to full channel", update.VehicleID)
			}
		}
	}

	// Process medium and low priority updates
	for _, update := range updates {
		priority := priorityOrder[update.Priority]
		if priority > 1 { // Medium and Low priority
			select {
			case m.broadcast <- update:
			default:
				// Drop low priority updates if channel is full
				continue
			}
		}
	}

	return nil
}

// GetConnectedClients returns the number of connected clients
func (m *Manager) GetConnectedClients() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.clients)
}

// GetClientStats returns detailed client statistics
func (m *Manager) GetClientStats() ClientStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := ClientStats{
		TotalClients: len(m.clients),
	}

	for _, client := range m.clients {
		if client.IsActive {
			stats.ActiveClients++
		} else {
			stats.InactiveClients++
		}
	}

	return stats
}

// GetUpgrader returns the WebSocket upgrader for external use
func (m *Manager) GetUpgrader() *websocket.Upgrader {
	return &m.upgrader
}

// broadcastToClients sends an update to all relevant clients based on their filters
func (m *Manager) broadcastToClients(update VehicleUpdate) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, client := range m.clients {
		if m.shouldSendToClient(client, update) {
			select {
			case client.Send <- update:
			default:
				// Client's send channel is full, mark as inactive
				client.IsActive = false
				log.Printf("Client %s send channel full, marking as inactive", client.ID)
			}
		}
	}
}

// shouldSendToClient determines if an update should be sent to a specific client
func (m *Manager) shouldSendToClient(client *Client, update VehicleUpdate) bool {
	filters := client.Filters

	// If no filters are set, send all updates
	if len(filters.VehicleIDs) == 0 && len(filters.Statuses) == 0 && 
	   len(filters.Drivers) == 0 && len(filters.AlertTypes) == 0 {
		return true
	}

	// Check vehicle ID filter
	if len(filters.VehicleIDs) > 0 {
		found := false
		for _, id := range filters.VehicleIDs {
			if id == update.VehicleID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check status filter
	if len(filters.Statuses) > 0 {
		if status, ok := update.Data["status"].(string); ok {
			found := false
			for _, s := range filters.Statuses {
				if s == status {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Check alert type filter
	if len(filters.AlertTypes) > 0 && update.UpdateType == "alert" {
		if alertType, ok := update.Data["alertType"].(string); ok {
			found := false
			for _, at := range filters.AlertTypes {
				if at == alertType {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	return true
}

// handleClient manages individual client connections
func (m *Manager) handleClient(client *Client) {
	defer func() {
		m.unregister <- client
	}()

	// Set up ping/pong handlers for connection health
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.LastPing = time.Now()
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start goroutine to handle outgoing messages
	go m.writeMessages(client)

	// Handle incoming messages (mainly pings and filter updates)
	for {
		var message map[string]interface{}
		err := client.Conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for client %s: %v", client.ID, err)
			}
			break
		}

		// Handle filter updates
		if msgType, ok := message["type"].(string); ok && msgType == "update_filters" {
			if filtersData, ok := message["filters"]; ok {
				filtersJSON, _ := json.Marshal(filtersData)
				var newFilters VehicleFilters
				if err := json.Unmarshal(filtersJSON, &newFilters); err == nil {
					client.Filters = newFilters
					log.Printf("Updated filters for client %s", client.ID)
				}
			}
		}
	}
}

// writeMessages handles outgoing messages to a client
func (m *Manager) writeMessages(client *Client) {
	ticker := time.NewTicker(54 * time.Second) // Send ping every 54 seconds
	defer ticker.Stop()

	for {
		select {
		case update, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send the vehicle update
			if err := client.Conn.WriteJSON(map[string]interface{}{
				"type": MessageTypeVehicleUpdate,
				"data": update,
			}); err != nil {
				log.Printf("Error writing message to client %s: %v", client.ID, err)
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error sending ping to client %s: %v", client.ID, err)
				return
			}
		}
	}
}

// healthCheck monitors client connections and removes inactive ones
func (m *Manager) healthCheck() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	for clientID, client := range m.clients {
		// Remove clients that haven't responded to ping in 90 seconds
		if now.Sub(client.LastPing) > 90*time.Second {
			log.Printf("Client %s timed out, removing", clientID)
			delete(m.clients, clientID)
			close(client.Send)
			if client.Conn != nil {
				client.Conn.Close()
			}
		}
	}
}