package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"fleet-backend/internal/models"
	"fleet-backend/pkg/redis"

	redisClient "github.com/redis/go-redis/v9"
)

// RedisCacheManager implements CacheManager using Redis
type RedisCacheManager struct {
	client *redis.Client
	config CacheConfig
	stats  *cacheStats
	ctx    context.Context
}

// cacheStats tracks cache performance metrics
type cacheStats struct {
	mu            sync.RWMutex
	totalHits     int64
	totalMisses   int64
	evictionCount int64
}

// NewRedisCacheManager creates a new Redis-backed cache manager
func NewRedisCacheManager(redisClient *redis.Client, config CacheConfig) *RedisCacheManager {
	return &RedisCacheManager{
		client: redisClient,
		config: config,
		stats:  &cacheStats{},
		ctx:    context.Background(),
	}
}

// GetVehicle retrieves a vehicle from cache
func (r *RedisCacheManager) GetVehicle(vehicleID string) (*models.Vehicle, error) {
	key := r.buildKey("vehicle", vehicleID)
	
	data, err := r.client.GetClient().Get(r.ctx, key).Result()
	if err != nil {
		if err == redisClient.Nil {
			r.recordMiss()
			return nil, nil // Cache miss, not an error
		}
		return nil, fmt.Errorf("failed to get vehicle from cache: %w", err)
	}
	
	var vehicle models.Vehicle
	if err := json.Unmarshal([]byte(data), &vehicle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal vehicle data: %w", err)
	}
	
	r.recordHit()
	return &vehicle, nil
}

// SetVehicle stores a vehicle in cache with TTL
func (r *RedisCacheManager) SetVehicle(vehicleID string, vehicle *models.Vehicle, ttl time.Duration) error {
	key := r.buildKey("vehicle", vehicleID)
	
	data, err := json.Marshal(vehicle)
	if err != nil {
		return fmt.Errorf("failed to marshal vehicle data: %w", err)
	}
	
	if err := r.client.GetClient().Set(r.ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set vehicle in cache: %w", err)
	}
	
	// Tag the key for intelligent invalidation
	tags := []string{
		fmt.Sprintf("vehicle:%s", vehicleID),
		fmt.Sprintf("driver:%s", vehicle.Driver),
		fmt.Sprintf("status:%s", vehicle.Status),
	}
	
	if err := r.TagKey(key, tags...); err != nil {
		// Log error but don't fail the cache operation
		fmt.Printf("Warning: failed to tag cache key %s: %v\n", key, err)
	}
	
	return nil
}

// InvalidateVehicle removes a specific vehicle from cache
func (r *RedisCacheManager) InvalidateVehicle(vehicleID string) error {
	key := r.buildKey("vehicle", vehicleID)
	return r.Delete(key)
}

// InvalidateVehiclesByTag removes all vehicles with a specific tag
func (r *RedisCacheManager) InvalidateVehiclesByTag(tag string) error {
	return r.InvalidateByTag(tag)
}

// GetVehicleList retrieves a list of vehicles from cache
func (r *RedisCacheManager) GetVehicleList(key string) ([]*models.Vehicle, error) {
	cacheKey := r.buildKey("vehicle_list", key)
	
	data, err := r.client.GetClient().Get(r.ctx, cacheKey).Result()
	if err != nil {
		if err == redisClient.Nil {
			r.recordMiss()
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get vehicle list from cache: %w", err)
	}
	
	var vehicles []*models.Vehicle
	if err := json.Unmarshal([]byte(data), &vehicles); err != nil {
		return nil, fmt.Errorf("failed to unmarshal vehicle list data: %w", err)
	}
	
	r.recordHit()
	return vehicles, nil
}

// SetVehicleList stores a list of vehicles in cache
func (r *RedisCacheManager) SetVehicleList(key string, vehicles []*models.Vehicle, ttl time.Duration) error {
	cacheKey := r.buildKey("vehicle_list", key)
	
	data, err := json.Marshal(vehicles)
	if err != nil {
		return fmt.Errorf("failed to marshal vehicle list data: %w", err)
	}
	
	if err := r.client.GetClient().Set(r.ctx, cacheKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set vehicle list in cache: %w", err)
	}
	
	// Tag the list with relevant tags
	var tags []string
	for _, vehicle := range vehicles {
		tags = append(tags, fmt.Sprintf("vehicle:%s", vehicle.ID.Hex()))
	}
	
	if err := r.TagKey(cacheKey, tags...); err != nil {
		fmt.Printf("Warning: failed to tag cache key %s: %v\n", cacheKey, err)
	}
	
	return nil
}

// Get retrieves a generic value from cache
func (r *RedisCacheManager) Get(key string, dest interface{}) error {
	cacheKey := r.buildKey("generic", key)
	
	data, err := r.client.GetClient().Get(r.ctx, cacheKey).Result()
	if err != nil {
		if err == redisClient.Nil {
			r.recordMiss()
			return nil // Cache miss
		}
		return fmt.Errorf("failed to get from cache: %w", err)
	}
	
	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}
	
	r.recordHit()
	return nil
}

// Set stores a generic value in cache
func (r *RedisCacheManager) Set(key string, value interface{}, ttl time.Duration) error {
	cacheKey := r.buildKey("generic", key)
	
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	
	return r.client.GetClient().Set(r.ctx, cacheKey, data, ttl).Err()
}

// Delete removes a key from cache
func (r *RedisCacheManager) Delete(key string) error {
	// Remove tags associated with this key first
	if err := r.removeKeyTags(key); err != nil {
		fmt.Printf("Warning: failed to remove tags for key %s: %v\n", key, err)
	}
	
	return r.client.GetClient().Del(r.ctx, key).Err()
}

// TagKey associates tags with a cache key for intelligent invalidation
func (r *RedisCacheManager) TagKey(key string, tags ...string) error {
	pipe := r.client.GetClient().Pipeline()
	
	// Store key-to-tags mapping
	keyTagsKey := r.buildTagKey("key_tags", key)
	pipe.SAdd(r.ctx, keyTagsKey, tags)
	pipe.Expire(r.ctx, keyTagsKey, r.config.VehicleDataTTL*2) // Tags live longer than data
	
	// Store tag-to-keys mapping
	for _, tag := range tags {
		tagKeysKey := r.buildTagKey("tag_keys", tag)
		pipe.SAdd(r.ctx, tagKeysKey, key)
		pipe.Expire(r.ctx, tagKeysKey, r.config.VehicleDataTTL*2)
	}
	
	_, err := pipe.Exec(r.ctx)
	return err
}

// InvalidateByTag removes all keys associated with a tag
func (r *RedisCacheManager) InvalidateByTag(tag string) error {
	tagKeysKey := r.buildTagKey("tag_keys", tag)
	
	// Get all keys associated with this tag
	keys, err := r.client.GetClient().SMembers(r.ctx, tagKeysKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get keys for tag %s: %w", tag, err)
	}
	
	if len(keys) == 0 {
		return nil // No keys to invalidate
	}
	
	pipe := r.client.GetClient().Pipeline()
	
	// Delete all keys
	for _, key := range keys {
		pipe.Del(r.ctx, key)
		// Also remove the key's tag associations
		keyTagsKey := r.buildTagKey("key_tags", key)
		pipe.Del(r.ctx, keyTagsKey)
	}
	
	// Remove the tag-to-keys mapping
	pipe.Del(r.ctx, tagKeysKey)
	
	_, err = pipe.Exec(r.ctx)
	if err != nil {
		return fmt.Errorf("failed to invalidate keys for tag %s: %w", tag, err)
	}
	
	r.stats.mu.Lock()
	r.stats.evictionCount += int64(len(keys))
	r.stats.mu.Unlock()
	
	return nil
}

// GetCacheStats returns cache performance statistics
func (r *RedisCacheManager) GetCacheStats() CacheStats {
	r.stats.mu.RLock()
	totalHits := r.stats.totalHits
	totalMisses := r.stats.totalMisses
	evictionCount := r.stats.evictionCount
	r.stats.mu.RUnlock()
	
	total := totalHits + totalMisses
	var hitRate, missRate float64
	if total > 0 {
		hitRate = float64(totalHits) / float64(total)
		missRate = float64(totalMisses) / float64(total)
	}
	
	// Get memory usage and key count from Redis
	info, err := r.client.GetClient().Info(r.ctx, "memory").Result()
	var memoryUsage int64
	if err == nil {
		if lines := strings.Split(info, "\n"); len(lines) > 0 {
			for _, line := range lines {
				if strings.HasPrefix(line, "used_memory:") {
					if val, err := strconv.ParseInt(strings.TrimPrefix(line, "used_memory:"), 10, 64); err == nil {
						memoryUsage = val
					}
				}
			}
		}
	}
	
	// Get approximate key count
	keyCount := 0
	if keys, err := r.client.GetClient().Keys(r.ctx, r.config.KeyPrefix+"*").Result(); err == nil {
		keyCount = len(keys)
	}
	
	return CacheStats{
		HitRate:       hitRate,
		MissRate:      missRate,
		MemoryUsage:   memoryUsage,
		KeyCount:      keyCount,
		EvictionCount: int(evictionCount),
		TotalHits:     totalHits,
		TotalMisses:   totalMisses,
	}
}

// HealthCheck verifies cache connectivity
func (r *RedisCacheManager) HealthCheck() error {
	return r.client.GetClient().Ping(r.ctx).Err()
}

// Close closes the cache manager
func (r *RedisCacheManager) Close() error {
	return r.client.Close()
}

// Helper methods

func (r *RedisCacheManager) buildKey(keyType, identifier string) string {
	return fmt.Sprintf("%s%s:%s", r.config.KeyPrefix, keyType, identifier)
}

func (r *RedisCacheManager) buildTagKey(keyType, identifier string) string {
	return fmt.Sprintf("%s%s:%s", r.config.TagPrefix, keyType, identifier)
}

func (r *RedisCacheManager) recordHit() {
	r.stats.mu.Lock()
	r.stats.totalHits++
	r.stats.mu.Unlock()
}

func (r *RedisCacheManager) recordMiss() {
	r.stats.mu.Lock()
	r.stats.totalMisses++
	r.stats.mu.Unlock()
}

func (r *RedisCacheManager) removeKeyTags(key string) error {
	keyTagsKey := r.buildTagKey("key_tags", key)
	
	// Get all tags for this key
	tags, err := r.client.GetClient().SMembers(r.ctx, keyTagsKey).Result()
	if err != nil {
		return err
	}
	
	pipe := r.client.GetClient().Pipeline()
	
	// Remove key from each tag's key set
	for _, tag := range tags {
		tagKeysKey := r.buildTagKey("tag_keys", tag)
		pipe.SRem(r.ctx, tagKeysKey, key)
	}
	
	// Remove the key's tag set
	pipe.Del(r.ctx, keyTagsKey)
	
	_, err = pipe.Exec(r.ctx)
	return err
}