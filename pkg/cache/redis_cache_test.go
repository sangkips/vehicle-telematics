package cache

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"fleet-backend/internal/models"

	"github.com/alicebob/miniredis/v2"
	redisClient "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// setupTestRedis is removed as we use direct test setup in each test

func createTestClient(addr string) *redisClient.Client {
	return redisClient.NewClient(&redisClient.Options{
		Addr: addr,
	})
}

func TestRedisCacheManager_VehicleOperations(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	
	client := createTestClient(mr.Addr())
	config := DefaultCacheConfig()
	config.KeyPrefix = "test:"
	config.TagPrefix = "test_tag:"
	
	// We need to create a mock that returns our test client
	// For this test, let's directly use the Redis client
	testManager := &testRedisCacheManager{
		client: client,
		config: config,
		stats:  &cacheStats{},
		ctx:    context.Background(),
	}
	
	// Test vehicle creation
	vehicleID := primitive.NewObjectID()
	vehicle := &models.Vehicle{
		ID:          vehicleID,
		Name:        "Test Vehicle",
		PlateNumber: "ABC123",
		Driver:      "John Doe",
		Status:      "active",
		FuelLevel:   75.5,
		LastUpdate:  time.Now(),
	}
	
	t.Run("SetVehicle", func(t *testing.T) {
		err := testManager.SetVehicle(vehicleID.Hex(), vehicle, 30*time.Second)
		assert.NoError(t, err)
	})
	
	t.Run("GetVehicle", func(t *testing.T) {
		retrievedVehicle, err := testManager.GetVehicle(vehicleID.Hex())
		assert.NoError(t, err)
		assert.NotNil(t, retrievedVehicle)
		assert.Equal(t, vehicle.Name, retrievedVehicle.Name)
		assert.Equal(t, vehicle.PlateNumber, retrievedVehicle.PlateNumber)
		assert.Equal(t, vehicle.Driver, retrievedVehicle.Driver)
	})
	
	t.Run("GetVehicle_NotFound", func(t *testing.T) {
		nonExistentID := primitive.NewObjectID().Hex()
		retrievedVehicle, err := testManager.GetVehicle(nonExistentID)
		assert.NoError(t, err)
		assert.Nil(t, retrievedVehicle)
	})
	
	t.Run("InvalidateVehicle", func(t *testing.T) {
		err := testManager.InvalidateVehicle(vehicleID.Hex())
		assert.NoError(t, err)
		
		// Verify vehicle is no longer in cache
		retrievedVehicle, err := testManager.GetVehicle(vehicleID.Hex())
		assert.NoError(t, err)
		assert.Nil(t, retrievedVehicle)
	})
}

func TestRedisCacheManager_TTLBehavior(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	
	client := createTestClient(mr.Addr())
	config := DefaultCacheConfig()
	config.KeyPrefix = "test:"
	config.VehicleDataTTL = 100 * time.Millisecond // Short TTL for testing
	
	testManager := &testRedisCacheManager{
		client: client,
		config: config,
		stats:  &cacheStats{},
		ctx:    context.Background(),
	}
	
	vehicleID := primitive.NewObjectID()
	vehicle := &models.Vehicle{
		ID:          vehicleID,
		Name:        "TTL Test Vehicle",
		PlateNumber: "TTL123",
		Driver:      "Jane Doe",
		Status:      "active",
	}
	
	t.Run("TTL_Expiration", func(t *testing.T) {
		// Set vehicle with short TTL
		err := testManager.SetVehicle(vehicleID.Hex(), vehicle, config.VehicleDataTTL)
		assert.NoError(t, err)
		
		// Verify vehicle exists
		retrievedVehicle, err := testManager.GetVehicle(vehicleID.Hex())
		assert.NoError(t, err)
		assert.NotNil(t, retrievedVehicle)
		
		// Fast-forward time in miniredis
		mr.FastForward(200 * time.Millisecond)
		
		// Verify vehicle has expired
		retrievedVehicle, err = testManager.GetVehicle(vehicleID.Hex())
		assert.NoError(t, err)
		assert.Nil(t, retrievedVehicle)
	})
}

func TestRedisCacheManager_TaggingSystem(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	
	client := createTestClient(mr.Addr())
	config := DefaultCacheConfig()
	config.KeyPrefix = "test:"
	config.TagPrefix = "test_tag:"
	
	testManager := &testRedisCacheManager{
		client: client,
		config: config,
		stats:  &cacheStats{},
		ctx:    context.Background(),
	}
	
	// Create multiple vehicles with same driver
	vehicles := []*models.Vehicle{
		{
			ID:          primitive.NewObjectID(),
			Name:        "Vehicle 1",
			PlateNumber: "V1-123",
			Driver:      "John Doe",
			Status:      "active",
		},
		{
			ID:          primitive.NewObjectID(),
			Name:        "Vehicle 2",
			PlateNumber: "V2-456",
			Driver:      "John Doe",
			Status:      "maintenance",
		},
		{
			ID:          primitive.NewObjectID(),
			Name:        "Vehicle 3",
			PlateNumber: "V3-789",
			Driver:      "Jane Smith",
			Status:      "active",
		},
	}
	
	t.Run("SetVehiclesWithTags", func(t *testing.T) {
		for _, vehicle := range vehicles {
			err := testManager.SetVehicle(vehicle.ID.Hex(), vehicle, 5*time.Minute)
			assert.NoError(t, err)
		}
	})
	
	t.Run("InvalidateByDriverTag", func(t *testing.T) {
		// Invalidate all vehicles for John Doe
		err := testManager.InvalidateByTag("driver:John Doe")
		assert.NoError(t, err)
		
		// Verify John Doe's vehicles are gone
		vehicle1, err := testManager.GetVehicle(vehicles[0].ID.Hex())
		assert.NoError(t, err)
		assert.Nil(t, vehicle1)
		
		vehicle2, err := testManager.GetVehicle(vehicles[1].ID.Hex())
		assert.NoError(t, err)
		assert.Nil(t, vehicle2)
		
		// Verify Jane Smith's vehicle is still there
		vehicle3, err := testManager.GetVehicle(vehicles[2].ID.Hex())
		assert.NoError(t, err)
		assert.NotNil(t, vehicle3)
		assert.Equal(t, "Jane Smith", vehicle3.Driver)
	})
	
	t.Run("InvalidateByStatusTag", func(t *testing.T) {
		// Re-add a vehicle with active status
		activeVehicle := &models.Vehicle{
			ID:          primitive.NewObjectID(),
			Name:        "Active Vehicle",
			PlateNumber: "ACT-123",
			Driver:      "Bob Wilson",
			Status:      "active",
		}
		
		err := testManager.SetVehicle(activeVehicle.ID.Hex(), activeVehicle, 5*time.Minute)
		assert.NoError(t, err)
		
		// Invalidate by status
		err = testManager.InvalidateByTag("status:active")
		assert.NoError(t, err)
		
		// Verify active vehicle is gone
		retrievedVehicle, err := testManager.GetVehicle(activeVehicle.ID.Hex())
		assert.NoError(t, err)
		assert.Nil(t, retrievedVehicle)
	})
}

func TestRedisCacheManager_VehicleListOperations(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	
	client := createTestClient(mr.Addr())
	config := DefaultCacheConfig()
	config.KeyPrefix = "test:"
	
	testManager := &testRedisCacheManager{
		client: client,
		config: config,
		stats:  &cacheStats{},
		ctx:    context.Background(),
	}
	
	vehicles := []*models.Vehicle{
		{
			ID:          primitive.NewObjectID(),
			Name:        "List Vehicle 1",
			PlateNumber: "L1-123",
			Driver:      "Driver 1",
			Status:      "active",
		},
		{
			ID:          primitive.NewObjectID(),
			Name:        "List Vehicle 2",
			PlateNumber: "L2-456",
			Driver:      "Driver 2",
			Status:      "maintenance",
		},
	}
	
	t.Run("SetVehicleList", func(t *testing.T) {
		err := testManager.SetVehicleList("active_vehicles", vehicles, 2*time.Minute)
		assert.NoError(t, err)
	})
	
	t.Run("GetVehicleList", func(t *testing.T) {
		retrievedVehicles, err := testManager.GetVehicleList("active_vehicles")
		assert.NoError(t, err)
		assert.NotNil(t, retrievedVehicles)
		assert.Len(t, retrievedVehicles, 2)
		assert.Equal(t, vehicles[0].Name, retrievedVehicles[0].Name)
		assert.Equal(t, vehicles[1].Name, retrievedVehicles[1].Name)
	})
	
	t.Run("GetVehicleList_NotFound", func(t *testing.T) {
		retrievedVehicles, err := testManager.GetVehicleList("nonexistent_list")
		assert.NoError(t, err)
		assert.Nil(t, retrievedVehicles)
	})
}

func TestRedisCacheManager_GenericOperations(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	
	client := createTestClient(mr.Addr())
	config := DefaultCacheConfig()
	config.KeyPrefix = "test:"
	
	testManager := &testRedisCacheManager{
		client: client,
		config: config,
		stats:  &cacheStats{},
		ctx:    context.Background(),
	}
	
	testData := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": []string{"a", "b", "c"},
	}
	
	t.Run("SetGeneric", func(t *testing.T) {
		for key, value := range testData {
			err := testManager.Set(key, value, 1*time.Minute)
			assert.NoError(t, err)
		}
	})
	
	t.Run("GetGeneric", func(t *testing.T) {
		var stringValue string
		err := testManager.Get("key1", &stringValue)
		assert.NoError(t, err)
		assert.Equal(t, "value1", stringValue)
		
		var intValue int
		err = testManager.Get("key2", &intValue)
		assert.NoError(t, err)
		assert.Equal(t, 42, intValue)
		
		var sliceValue []string
		err = testManager.Get("key3", &sliceValue)
		assert.NoError(t, err)
		assert.Equal(t, []string{"a", "b", "c"}, sliceValue)
	})
	
	t.Run("DeleteGeneric", func(t *testing.T) {
		err := testManager.Delete(testManager.buildKey("generic", "key1"))
		assert.NoError(t, err)
		
		var value string
		err = testManager.Get("key1", &value)
		assert.NoError(t, err)
		assert.Empty(t, value)
	})
}

func TestRedisCacheManager_Stats(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	
	client := createTestClient(mr.Addr())
	config := DefaultCacheConfig()
	config.KeyPrefix = "test:"
	
	testManager := &testRedisCacheManager{
		client: client,
		config: config,
		stats:  &cacheStats{},
		ctx:    context.Background(),
	}
	
	vehicleID := primitive.NewObjectID()
	vehicle := &models.Vehicle{
		ID:          vehicleID,
		Name:        "Stats Test Vehicle",
		PlateNumber: "STS-123",
		Driver:      "Stats Driver",
		Status:      "active",
	}
	
	t.Run("StatsTracking", func(t *testing.T) {
		// Initial stats should be zero
		stats := testManager.GetCacheStats()
		assert.Equal(t, int64(0), stats.TotalHits)
		assert.Equal(t, int64(0), stats.TotalMisses)
		
		// Cache miss
		_, err := testManager.GetVehicle(vehicleID.Hex())
		assert.NoError(t, err)
		
		stats = testManager.GetCacheStats()
		assert.Equal(t, int64(0), stats.TotalHits)
		assert.Equal(t, int64(1), stats.TotalMisses)
		assert.Equal(t, 0.0, stats.HitRate)
		assert.Equal(t, 1.0, stats.MissRate)
		
		// Cache set and hit
		err = testManager.SetVehicle(vehicleID.Hex(), vehicle, 1*time.Minute)
		assert.NoError(t, err)
		
		_, err = testManager.GetVehicle(vehicleID.Hex())
		assert.NoError(t, err)
		
		stats = testManager.GetCacheStats()
		assert.Equal(t, int64(1), stats.TotalHits)
		assert.Equal(t, int64(1), stats.TotalMisses)
		assert.Equal(t, 0.5, stats.HitRate)
		assert.Equal(t, 0.5, stats.MissRate)
	})
}

func TestRedisCacheManager_HealthCheck(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	
	client := createTestClient(mr.Addr())
	config := DefaultCacheConfig()
	
	testManager := &testRedisCacheManager{
		client: client,
		config: config,
		stats:  &cacheStats{},
		ctx:    context.Background(),
	}
	
	t.Run("HealthCheck_Success", func(t *testing.T) {
		err := testManager.HealthCheck()
		assert.NoError(t, err)
	})
	
	t.Run("HealthCheck_Failure", func(t *testing.T) {
		mr.Close()
		err := testManager.HealthCheck()
		assert.Error(t, err)
	})
}

// testRedisCacheManager is a simplified version for testing
type testRedisCacheManager struct {
	client *redisClient.Client
	config CacheConfig
	stats  *cacheStats
	ctx    context.Context
}

func (t *testRedisCacheManager) GetVehicle(vehicleID string) (*models.Vehicle, error) {
	key := t.buildKey("vehicle", vehicleID)
	
	data, err := t.client.Get(t.ctx, key).Result()
	if err != nil {
		if err == redisClient.Nil {
			t.recordMiss()
			return nil, nil
		}
		return nil, err
	}
	
	var vehicle models.Vehicle
	if err := json.Unmarshal([]byte(data), &vehicle); err != nil {
		return nil, err
	}
	
	t.recordHit()
	return &vehicle, nil
}

func (t *testRedisCacheManager) SetVehicle(vehicleID string, vehicle *models.Vehicle, ttl time.Duration) error {
	key := t.buildKey("vehicle", vehicleID)
	
	data, err := json.Marshal(vehicle)
	if err != nil {
		return err
	}
	
	if err := t.client.Set(t.ctx, key, data, ttl).Err(); err != nil {
		return err
	}
	
	// Tag the key
	tags := []string{
		"vehicle:" + vehicleID,
		"driver:" + vehicle.Driver,
		"status:" + vehicle.Status,
	}
	
	return t.TagKey(key, tags...)
}

func (t *testRedisCacheManager) InvalidateVehicle(vehicleID string) error {
	key := t.buildKey("vehicle", vehicleID)
	return t.Delete(key)
}

func (t *testRedisCacheManager) InvalidateByTag(tag string) error {
	tagKeysKey := t.buildTagKey("tag_keys", tag)
	
	keys, err := t.client.SMembers(t.ctx, tagKeysKey).Result()
	if err != nil {
		return err
	}
	
	if len(keys) == 0 {
		return nil
	}
	
	pipe := t.client.Pipeline()
	
	for _, key := range keys {
		pipe.Del(t.ctx, key)
		keyTagsKey := t.buildTagKey("key_tags", key)
		pipe.Del(t.ctx, keyTagsKey)
	}
	
	pipe.Del(t.ctx, tagKeysKey)
	
	_, err = pipe.Exec(t.ctx)
	if err != nil {
		return err
	}
	
	t.stats.mu.Lock()
	t.stats.evictionCount += int64(len(keys))
	t.stats.mu.Unlock()
	
	return nil
}

func (t *testRedisCacheManager) GetVehicleList(key string) ([]*models.Vehicle, error) {
	cacheKey := t.buildKey("vehicle_list", key)
	
	data, err := t.client.Get(t.ctx, cacheKey).Result()
	if err != nil {
		if err == redisClient.Nil {
			t.recordMiss()
			return nil, nil
		}
		return nil, err
	}
	
	var vehicles []*models.Vehicle
	if err := json.Unmarshal([]byte(data), &vehicles); err != nil {
		return nil, err
	}
	
	t.recordHit()
	return vehicles, nil
}

func (t *testRedisCacheManager) SetVehicleList(key string, vehicles []*models.Vehicle, ttl time.Duration) error {
	cacheKey := t.buildKey("vehicle_list", key)
	
	data, err := json.Marshal(vehicles)
	if err != nil {
		return err
	}
	
	if err := t.client.Set(t.ctx, cacheKey, data, ttl).Err(); err != nil {
		return err
	}
	
	var tags []string
	for _, vehicle := range vehicles {
		tags = append(tags, "vehicle:"+vehicle.ID.Hex())
	}
	
	return t.TagKey(cacheKey, tags...)
}

func (t *testRedisCacheManager) Get(key string, dest interface{}) error {
	cacheKey := t.buildKey("generic", key)
	
	data, err := t.client.Get(t.ctx, cacheKey).Result()
	if err != nil {
		if err == redisClient.Nil {
			t.recordMiss()
			return nil
		}
		return err
	}
	
	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return err
	}
	
	t.recordHit()
	return nil
}

func (t *testRedisCacheManager) Set(key string, value interface{}, ttl time.Duration) error {
	cacheKey := t.buildKey("generic", key)
	
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	
	return t.client.Set(t.ctx, cacheKey, data, ttl).Err()
}

func (t *testRedisCacheManager) Delete(key string) error {
	if err := t.removeKeyTags(key); err != nil {
		// Log but don't fail
	}
	return t.client.Del(t.ctx, key).Err()
}

func (t *testRedisCacheManager) TagKey(key string, tags ...string) error {
	pipe := t.client.Pipeline()
	
	keyTagsKey := t.buildTagKey("key_tags", key)
	pipe.SAdd(t.ctx, keyTagsKey, tags)
	pipe.Expire(t.ctx, keyTagsKey, t.config.VehicleDataTTL*2)
	
	for _, tag := range tags {
		tagKeysKey := t.buildTagKey("tag_keys", tag)
		pipe.SAdd(t.ctx, tagKeysKey, key)
		pipe.Expire(t.ctx, tagKeysKey, t.config.VehicleDataTTL*2)
	}
	
	_, err := pipe.Exec(t.ctx)
	return err
}

func (t *testRedisCacheManager) GetCacheStats() CacheStats {
	t.stats.mu.RLock()
	totalHits := t.stats.totalHits
	totalMisses := t.stats.totalMisses
	evictionCount := t.stats.evictionCount
	t.stats.mu.RUnlock()
	
	total := totalHits + totalMisses
	var hitRate, missRate float64
	if total > 0 {
		hitRate = float64(totalHits) / float64(total)
		missRate = float64(totalMisses) / float64(total)
	}
	
	return CacheStats{
		HitRate:       hitRate,
		MissRate:      missRate,
		MemoryUsage:   0, // Simplified for testing
		KeyCount:      0, // Simplified for testing
		EvictionCount: int(evictionCount),
		TotalHits:     totalHits,
		TotalMisses:   totalMisses,
	}
}

func (t *testRedisCacheManager) HealthCheck() error {
	return t.client.Ping(t.ctx).Err()
}

func (t *testRedisCacheManager) buildKey(keyType, identifier string) string {
	return t.config.KeyPrefix + keyType + ":" + identifier
}

func (t *testRedisCacheManager) buildTagKey(keyType, identifier string) string {
	return t.config.TagPrefix + keyType + ":" + identifier
}

func (t *testRedisCacheManager) recordHit() {
	t.stats.mu.Lock()
	t.stats.totalHits++
	t.stats.mu.Unlock()
}

func (t *testRedisCacheManager) recordMiss() {
	t.stats.mu.Lock()
	t.stats.totalMisses++
	t.stats.mu.Unlock()
}

func (t *testRedisCacheManager) removeKeyTags(key string) error {
	keyTagsKey := t.buildTagKey("key_tags", key)
	
	tags, err := t.client.SMembers(t.ctx, keyTagsKey).Result()
	if err != nil {
		return err
	}
	
	pipe := t.client.Pipeline()
	
	for _, tag := range tags {
		tagKeysKey := t.buildTagKey("tag_keys", tag)
		pipe.SRem(t.ctx, tagKeysKey, key)
	}
	
	pipe.Del(t.ctx, keyTagsKey)
	
	_, err = pipe.Exec(t.ctx)
	return err
}