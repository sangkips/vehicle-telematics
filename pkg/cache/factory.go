package cache

import (
	"fleet-backend/pkg/redis"
)

// NewCacheManager creates a new cache manager with the specified Redis client and configuration
func NewCacheManager(redisClient *redis.Client, config CacheConfig) CacheManager {
	return NewRedisCacheManager(redisClient, config)
}

// NewDefaultCacheManager creates a new cache manager with default configuration
func NewDefaultCacheManager(redisClient *redis.Client) CacheManager {
	return NewRedisCacheManager(redisClient, DefaultCacheConfig())
}