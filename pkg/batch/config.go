package batch

import (
	"os"
	"strconv"
	"time"
)

// DefaultBatchConfig returns the default configuration for batch processing
func DefaultBatchConfig() BatchConfig {
	return BatchConfig{
		MaxBatchSize:  50,                    // 50 vehicles per batch (as per requirements)
		BatchInterval: 30 * time.Second,     // 30 seconds interval
		MaxWaitTime:   5 * time.Minute,      // 5 minutes max wait time
		RetryAttempts: 3,                    // 3 retry attempts
		RetryBackoff:  1 * time.Second,      // 1 second initial backoff
	}
}

// LoadBatchConfigFromEnv loads batch configuration from environment variables
func LoadBatchConfigFromEnv() BatchConfig {
	config := DefaultBatchConfig()

	// Load max batch size
	if val := os.Getenv("BATCH_MAX_SIZE"); val != "" {
		if size, err := strconv.Atoi(val); err == nil && size > 0 {
			config.MaxBatchSize = size
		}
	}

	// Load batch interval
	if val := os.Getenv("BATCH_INTERVAL"); val != "" {
		if interval, err := time.ParseDuration(val); err == nil {
			config.BatchInterval = interval
		}
	}

	// Load max wait time
	if val := os.Getenv("BATCH_MAX_WAIT_TIME"); val != "" {
		if waitTime, err := time.ParseDuration(val); err == nil {
			config.MaxWaitTime = waitTime
		}
	}

	// Load retry attempts
	if val := os.Getenv("BATCH_RETRY_ATTEMPTS"); val != "" {
		if attempts, err := strconv.Atoi(val); err == nil && attempts >= 0 {
			config.RetryAttempts = attempts
		}
	}

	// Load retry backoff
	if val := os.Getenv("BATCH_RETRY_BACKOFF"); val != "" {
		if backoff, err := time.ParseDuration(val); err == nil {
			config.RetryBackoff = backoff
		}
	}

	return config
}

// ValidateConfig validates the batch configuration
func ValidateConfig(config BatchConfig) error {
	if config.MaxBatchSize <= 0 {
		return ErrInvalidBatchSize
	}

	if config.BatchInterval <= 0 {
		return ErrInvalidBatchInterval
	}

	if config.MaxWaitTime <= 0 {
		return ErrInvalidMaxWaitTime
	}

	if config.RetryAttempts < 0 {
		return ErrInvalidRetryAttempts
	}

	if config.RetryBackoff < 0 {
		return ErrInvalidRetryBackoff
	}

	return nil
}