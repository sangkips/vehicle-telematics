package services

import (
	"fleet-backend/internal/models"
	"fleet-backend/pkg/cache"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MockCacheManager is a mock implementation of the CacheManager interface
type MockCacheManager struct {
	mock.Mock
}

func (m *MockCacheManager) GetVehicle(vehicleID string) (*models.Vehicle, error) {
	args := m.Called(vehicleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Vehicle), args.Error(1)
}

func (m *MockCacheManager) SetVehicle(vehicleID string, vehicle *models.Vehicle, ttl time.Duration) error {
	args := m.Called(vehicleID, vehicle, ttl)
	return args.Error(0)
}

func (m *MockCacheManager) InvalidateVehicle(vehicleID string) error {
	args := m.Called(vehicleID)
	return args.Error(0)
}

func (m *MockCacheManager) InvalidateVehiclesByTag(tag string) error {
	args := m.Called(tag)
	return args.Error(0)
}

func (m *MockCacheManager) GetVehicleList(key string) ([]*models.Vehicle, error) {
	args := m.Called(key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Vehicle), args.Error(1)
}

func (m *MockCacheManager) SetVehicleList(key string, vehicles []*models.Vehicle, ttl time.Duration) error {
	args := m.Called(key, vehicles, ttl)
	return args.Error(0)
}

func (m *MockCacheManager) Get(key string, dest interface{}) error {
	args := m.Called(key, dest)
	return args.Error(0)
}

func (m *MockCacheManager) Set(key string, value interface{}, ttl time.Duration) error {
	args := m.Called(key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheManager) Delete(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *MockCacheManager) TagKey(key string, tags ...string) error {
	args := m.Called(key, tags)
	return args.Error(0)
}

func (m *MockCacheManager) InvalidateByTag(tag string) error {
	args := m.Called(tag)
	return args.Error(0)
}

func (m *MockCacheManager) GetCacheStats() cache.CacheStats {
	args := m.Called()
	return args.Get(0).(cache.CacheStats)
}

func (m *MockCacheManager) HealthCheck() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockCacheManager) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Test cache-first strategy for GetVehicleByID with cache hit
func TestVehicleService_GetVehicleByID_CacheHit(t *testing.T) {
	mockCache := new(MockCacheManager)
	service := &VehicleService{
		cacheManager: mockCache,
		cacheConfig:  cache.DefaultCacheConfig(),
	}

	testVehicle := &models.Vehicle{
		ID:          primitive.NewObjectID(),
		Name:        "Test Vehicle",
		PlateNumber: "TEST123",
		Driver:      "John Doe",
		Status:      "active",
	}

	vehicleID := testVehicle.ID.Hex()

	// Mock cache hit
	mockCache.On("GetVehicle", vehicleID).Return(testVehicle, nil)

	result, err := service.GetVehicleByID(vehicleID)

	assert.NoError(t, err)
	assert.Equal(t, testVehicle, result)
	mockCache.AssertExpectations(t)
}

// Test cache-first strategy for GetAllVehicles with cache hit
func TestVehicleService_GetAllVehicles_CacheHit(t *testing.T) {
	mockCache := new(MockCacheManager)
	service := &VehicleService{
		cacheManager: mockCache,
		cacheConfig:  cache.DefaultCacheConfig(),
	}

	testVehicles := []*models.Vehicle{
		{
			ID:          primitive.NewObjectID(),
			Name:        "Test Vehicle 1",
			PlateNumber: "TEST123",
			Driver:      "John Doe",
			Status:      "active",
		},
		{
			ID:          primitive.NewObjectID(),
			Name:        "Test Vehicle 2",
			PlateNumber: "TEST456",
			Driver:      "Jane Doe",
			Status:      "idle",
		},
	}

	// Mock cache hit
	mockCache.On("GetVehicleList", "all_vehicles").Return(testVehicles, nil)

	result, err := service.GetAllVehicles()

	assert.NoError(t, err)
	assert.Equal(t, testVehicles, result)
	mockCache.AssertExpectations(t)
}

// Test cache invalidation helper methods
func TestVehicleService_CacheInvalidationHelpers(t *testing.T) {
	mockCache := new(MockCacheManager)
	service := &VehicleService{
		cacheManager: mockCache,
		cacheConfig:  cache.DefaultCacheConfig(),
	}

	testVehicle := &models.Vehicle{
		ID:          primitive.NewObjectID(),
		Name:        "Test Vehicle",
		PlateNumber: "TEST123",
		Driver:      "John Doe",
		Status:      "active",
	}

	t.Run("invalidateCacheOnCreate", func(t *testing.T) {
		// Mock cache invalidation calls for create
		mockCache.On("Delete", "fleet:vehicle_list:all_vehicles").Return(nil)
		mockCache.On("Delete", "fleet:vehicle_list:vehicles_by_status_active").Return(nil)
		mockCache.On("Delete", "fleet:vehicle_list:vehicles_by_driver_John Doe").Return(nil)
		mockCache.On("SetVehicle", testVehicle.ID.Hex(), testVehicle, service.cacheConfig.VehicleDataTTL).Return(nil)

		service.invalidateCacheOnCreate(testVehicle)

		mockCache.AssertExpectations(t)
	})

	t.Run("invalidateCacheOnUpdate", func(t *testing.T) {
		vehicleID := testVehicle.ID.Hex()
		previousDriver := "Old Driver"
		previousStatus := "idle"

		// Mock cache invalidation calls for update
		mockCache.On("InvalidateVehicle", vehicleID).Return(nil)
		mockCache.On("Delete", "fleet:vehicle_list:all_vehicles").Return(nil)
		mockCache.On("Delete", "fleet:vehicle_list:vehicles_by_status_active").Return(nil)
		mockCache.On("Delete", "fleet:vehicle_list:vehicles_by_status_idle").Return(nil) // previous status
		mockCache.On("Delete", "fleet:vehicle_list:vehicles_by_driver_John Doe").Return(nil)
		mockCache.On("Delete", "fleet:vehicle_list:vehicles_by_driver_Old Driver").Return(nil) // previous driver
		mockCache.On("SetVehicle", vehicleID, testVehicle, service.cacheConfig.VehicleDataTTL).Return(nil)

		service.invalidateCacheOnUpdate(testVehicle, previousDriver, previousStatus)

		mockCache.AssertExpectations(t)
	})

	t.Run("invalidateCacheOnDelete", func(t *testing.T) {
		vehicleID := testVehicle.ID.Hex()

		// Mock cache invalidation calls for delete
		mockCache.On("InvalidateVehicle", vehicleID).Return(nil)
		mockCache.On("Delete", "fleet:vehicle_list:all_vehicles").Return(nil)
		mockCache.On("Delete", "fleet:vehicle_list:vehicles_by_status_active").Return(nil)
		mockCache.On("Delete", "fleet:vehicle_list:vehicles_by_driver_John Doe").Return(nil)

		service.invalidateCacheOnDelete(testVehicle)

		mockCache.AssertExpectations(t)
	})
}

// Test cache fallback when cache is unavailable
func TestVehicleService_CacheFallback(t *testing.T) {
	// Test that the service handles gracefully when cache manager is nil
	service := &VehicleService{
		cacheManager: nil, // No cache manager
		cacheConfig:  cache.DefaultCacheConfig(),
	}

	// Test that cache manager can be set to nil without issues
	service.SetCacheManager(nil)
	assert.Nil(t, service.cacheManager)

	// Test that cache configuration works without cache manager
	customConfig := cache.CacheConfig{
		VehicleDataTTL: 60 * time.Second,
	}
	service.SetCacheConfig(customConfig)
	assert.Equal(t, customConfig, service.cacheConfig)
}

// Test cache configuration
func TestVehicleService_CacheConfiguration(t *testing.T) {
	service := &VehicleService{
		cacheConfig: cache.DefaultCacheConfig(),
	}

	// Test setting cache manager
	mockCache := new(MockCacheManager)
	service.SetCacheManager(mockCache)
	assert.Equal(t, mockCache, service.cacheManager)

	// Test setting cache config
	customConfig := cache.CacheConfig{
		VehicleDataTTL: 60 * time.Second,
		VehicleListTTL: 5 * time.Minute,
	}
	service.SetCacheConfig(customConfig)
	assert.Equal(t, customConfig, service.cacheConfig)
}