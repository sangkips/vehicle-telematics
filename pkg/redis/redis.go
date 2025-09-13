package redis

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"fleet-backend/internal/config"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	client        *redis.Client
	config        config.RedisConfig
	mu            sync.RWMutex
	isConnected   bool
	reconnectChan chan struct{}
	ctx           context.Context
	cancel        context.CancelFunc
}

type HealthStatus struct {
	IsConnected    bool          `json:"isConnected"`
	LastPing       time.Time     `json:"lastPing"`
	ResponseTime   time.Duration `json:"responseTime"`
	ConnectionInfo string        `json:"connectionInfo"`
	Error          string        `json:"error,omitempty"`
}

// NewClient creates a new Redis client with connection pooling
func NewClient(cfg config.RedisConfig) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	
	client := &Client{
		config:        cfg,
		reconnectChan: make(chan struct{}, 1),
		ctx:           ctx,
		cancel:        cancel,
	}

	client.connect()
	go client.healthCheckLoop()
	go client.reconnectLoop()

	return client
}

// connect establishes the Redis connection with configured options
func (c *Client) connect() {
	if c.config.URL != "" {
		opt, err := redis.ParseURL(c.config.URL)
		if err != nil {
			log.Printf("Failed to parse Redis URL: %v, falling back to host:port", err)
			c.connectWithHostPort()
			return
		}

		// Apply additional configuration
		opt.PoolSize = c.config.PoolSize
		opt.MinIdleConns = c.config.MinIdleConns
		opt.MaxRetries = c.config.MaxRetries
		opt.MinRetryBackoff = c.config.RetryDelay
		opt.DialTimeout = c.config.DialTimeout
		opt.ReadTimeout = c.config.ReadTimeout
		opt.WriteTimeout = c.config.WriteTimeout
		opt.PoolTimeout = c.config.PoolTimeout
		// opt.IdleTimeout = c.config.IdleTimeout
		// opt.IdleCheckFrequency = c.config.IdleCheckFrequency

		c.mu.Lock()
		c.client = redis.NewClient(opt)
		c.mu.Unlock()
	} else {
		c.connectWithHostPort()
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client != nil {
		err := client.Ping(ctx).Err()
		c.mu.Lock()
		c.isConnected = (err == nil)
		c.mu.Unlock()

		if err != nil {
			log.Printf("Redis connection test failed: %v", err)
		} else {
			log.Printf("Redis connected successfully")
		}
	}
}

func (c *Client) connectWithHostPort() {
	opt := &redis.Options{
		Addr:               fmt.Sprintf("%s:%s", c.config.Host, c.config.Port),
		Password:           c.config.Password,
		DB:                 c.config.DB,
		PoolSize:           c.config.PoolSize,
		MinIdleConns:       c.config.MinIdleConns,
		MaxRetries:         c.config.MaxRetries,
		MinRetryBackoff:    c.config.RetryDelay,
		DialTimeout:        c.config.DialTimeout,
		ReadTimeout:        c.config.ReadTimeout,
		WriteTimeout:       c.config.WriteTimeout,
		PoolTimeout:        c.config.PoolTimeout,
		// IdleTimeout:        c.config.IdleTimeout,
		// IdleCheckFrequency: c.config.IdleCheckFrequency,
	}

	c.mu.Lock()
	c.client = redis.NewClient(opt)
	c.mu.Unlock()
}

// GetClient returns the Redis client instance (thread-safe)
func (c *Client) GetClient() *redis.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client
}

// IsConnected returns the current connection status
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// HealthCheck performs a health check and returns detailed status
func (c *Client) HealthCheck() HealthStatus {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	status := HealthStatus{
		IsConnected:    c.isConnected,
		ConnectionInfo: fmt.Sprintf("%s:%s", c.config.Host, c.config.Port),
	}

	if client == nil {
		status.Error = "Redis client not initialized"
		return status
	}

	// Perform ping with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	start := time.Now()
	err := client.Ping(ctx).Err()
	status.ResponseTime = time.Since(start)
	status.LastPing = time.Now()

	if err != nil {
		status.IsConnected = false
		status.Error = err.Error()
		c.mu.Lock()
		c.isConnected = false
		c.mu.Unlock()
		c.triggerReconnect()
	} else {
		c.mu.Lock()
		c.isConnected = true
		c.mu.Unlock()
		status.IsConnected = true
	}

	return status
}

// triggerReconnect signals the reconnection goroutine
func (c *Client) triggerReconnect() {
	select {
	case c.reconnectChan <- struct{}{}:
	default:
		// Channel is full, reconnection already triggered
	}
}

// healthCheckLoop runs periodic health checks
func (c *Client) healthCheckLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			status := c.HealthCheck()
			if !status.IsConnected {
				log.Printf("Redis health check failed: %s", status.Error)
			}
		}
	}
}

// reconnectLoop handles automatic reconnection with exponential backoff
func (c *Client) reconnectLoop() {
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.reconnectChan:
			if c.IsConnected() {
				continue
			}

			log.Printf("Attempting to reconnect to Redis...")
			
			// Close existing client if it exists
			c.mu.Lock()
			if c.client != nil {
				c.client.Close()
			}
			c.mu.Unlock()

			// Attempt reconnection
			c.connect()

			if !c.IsConnected() {
				log.Printf("Reconnection failed, retrying in %v", backoff)
				time.Sleep(backoff)
				
				// Exponential backoff
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				
				c.triggerReconnect()
			} else {
				log.Println("Successfully reconnected to Redis")
				backoff = 1 * time.Second // Reset backoff on successful connection
			}
		}
	}
}

// Close gracefully shuts down the Redis client
func (c *Client) Close() error {
	c.cancel()
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// GetConnectionStats returns connection pool statistics
func (c *Client) GetConnectionStats() map[string]interface{} {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil {
		return map[string]interface{}{
			"error": "Redis client not initialized",
		}
	}

	stats := client.PoolStats()
	return map[string]interface{}{
		"hits":         stats.Hits,
		"misses":       stats.Misses,
		"timeouts":     stats.Timeouts,
		"totalConns":   stats.TotalConns,
		"idleConns":    stats.IdleConns,
		"staleConns":   stats.StaleConns,
		"isConnected":  c.isConnected,
	}
}