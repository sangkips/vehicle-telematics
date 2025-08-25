package redis

import (
	"fleet-backend/internal/config"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	cfg := config.RedisConfig{
		Host:               "localhost",
		Port:               "6379",
		Password:           "",
		DB:                 0,
		PoolSize:           10,
		MinIdleConns:       5,
		MaxRetries:         3,
		RetryDelay:         1 * time.Second,
		DialTimeout:        5 * time.Second,
		ReadTimeout:        3 * time.Second,
		WriteTimeout:       3 * time.Second,
		PoolTimeout:        4 * time.Second,
		IdleTimeout:        5 * time.Minute,
		IdleCheckFrequency: 1 * time.Minute,
	}

	client := NewClient(cfg)
	defer client.Close()

	// Test that client is created
	if client == nil {
		t.Fatal("Expected client to be created, got nil")
	}

	// Test that we can get the Redis client
	redisClient := client.GetClient()
	if redisClient == nil {
		t.Fatal("Expected Redis client to be available, got nil")
	}
}

func TestHealthCheck(t *testing.T) {
	cfg := config.RedisConfig{
		Host:               "localhost",
		Port:               "6379",
		Password:           "",
		DB:                 0,
		PoolSize:           10,
		MinIdleConns:       5,
		MaxRetries:         3,
		RetryDelay:         1 * time.Second,
		DialTimeout:        5 * time.Second,
		ReadTimeout:        3 * time.Second,
		WriteTimeout:       3 * time.Second,
		PoolTimeout:        4 * time.Second,
		IdleTimeout:        5 * time.Minute,
		IdleCheckFrequency: 1 * time.Minute,
	}

	client := NewClient(cfg)
	defer client.Close()

	// Give some time for initial connection
	time.Sleep(100 * time.Millisecond)

	status := client.HealthCheck()
	
	// Test that health check returns a status
	if status.ConnectionInfo == "" {
		t.Error("Expected connection info to be set")
	}

	if status.LastPing.IsZero() {
		t.Error("Expected LastPing to be set")
	}

	// Note: We don't test IsConnected = true because Redis might not be running in test environment
}

func TestGetConnectionStats(t *testing.T) {
	cfg := config.RedisConfig{
		Host:               "localhost",
		Port:               "6379",
		Password:           "",
		DB:                 0,
		PoolSize:           10,
		MinIdleConns:       5,
		MaxRetries:         3,
		RetryDelay:         1 * time.Second,
		DialTimeout:        5 * time.Second,
		ReadTimeout:        3 * time.Second,
		WriteTimeout:       3 * time.Second,
		PoolTimeout:        4 * time.Second,
		IdleTimeout:        5 * time.Minute,
		IdleCheckFrequency: 1 * time.Minute,
	}

	client := NewClient(cfg)
	defer client.Close()

	stats := client.GetConnectionStats()
	
	// Test that stats are returned
	if stats == nil {
		t.Fatal("Expected connection stats to be returned, got nil")
	}

	// Check that expected keys exist
	expectedKeys := []string{"hits", "misses", "timeouts", "totalConns", "idleConns", "staleConns", "isConnected"}
	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Expected key %s to exist in connection stats", key)
		}
	}
}