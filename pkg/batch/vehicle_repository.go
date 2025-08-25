package batch

import (
	"context"
	"errors"
	"fmt"
	"time"

	"fleet-backend/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// VehicleRepositoryAdapter adapts the existing VehicleRepository to support batch operations
type VehicleRepositoryAdapter struct {
	repo       *repository.VehicleRepository
	collection *mongo.Collection
}

// NewVehicleRepositoryAdapter creates a new adapter for batch operations
func NewVehicleRepositoryAdapter(repo *repository.VehicleRepository, db *mongo.Database) *VehicleRepositoryAdapter {
	return &VehicleRepositoryAdapter{
		repo:       repo,
		collection: db.Collection("vehicles"),
	}
}

// UpdateVehicle updates a single vehicle with the provided data
func (vra *VehicleRepositoryAdapter) UpdateVehicle(vehicleID string, update VehicleUpdateData) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(vehicleID)
	if err != nil {
		return fmt.Errorf("invalid vehicle ID %s: %w", vehicleID, err)
	}

	// Build update document with only non-nil fields
	updateDoc := bson.M{
		"last_update": update.Timestamp,
		"updated_at":  time.Now(),
	}

	if update.FuelLevel != nil {
		updateDoc["fuel_level"] = *update.FuelLevel
	}
	if update.Location != nil {
		updateDoc["location"] = *update.Location
	}
	if update.Speed != nil {
		updateDoc["speed"] = *update.Speed
	}
	if update.Status != nil {
		updateDoc["status"] = *update.Status
	}
	if update.Odometer != nil {
		updateDoc["odometer"] = *update.Odometer
	}

	result, err := vra.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": updateDoc},
	)
	if err != nil {
		return fmt.Errorf("failed to update vehicle %s: %w", vehicleID, err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("vehicle %s not found", vehicleID)
	}

	return nil
}

// UpdateVehiclesBatch updates multiple vehicles in a single batch operation
func (vra *VehicleRepositoryAdapter) UpdateVehiclesBatch(updates map[string]VehicleUpdateData) error {
	if len(updates) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use MongoDB bulk write operations for efficiency
	var operations []mongo.WriteModel

	for vehicleID, update := range updates {
		objectID, err := primitive.ObjectIDFromHex(vehicleID)
		if err != nil {
			return fmt.Errorf("invalid vehicle ID %s: %w", vehicleID, err)
		}

		// Build update document with only non-nil fields
		updateDoc := bson.M{
			"last_update": update.Timestamp,
			"updated_at":  time.Now(),
		}

		if update.FuelLevel != nil {
			updateDoc["fuel_level"] = *update.FuelLevel
		}
		if update.Location != nil {
			updateDoc["location"] = *update.Location
		}
		if update.Speed != nil {
			updateDoc["speed"] = *update.Speed
		}
		if update.Status != nil {
			updateDoc["status"] = *update.Status
		}
		if update.Odometer != nil {
			updateDoc["odometer"] = *update.Odometer
		}

		operation := mongo.NewUpdateOneModel().
			SetFilter(bson.M{"_id": objectID}).
			SetUpdate(bson.M{"$set": updateDoc}).
			SetUpsert(false)

		operations = append(operations, operation)
	}

	// Execute bulk write
	result, err := vra.collection.BulkWrite(ctx, operations)
	if err != nil {
		return fmt.Errorf("bulk write failed: %w", err)
	}

	// Check if all updates were successful
	expectedUpdates := int64(len(updates))
	if result.ModifiedCount != expectedUpdates {
		return fmt.Errorf("expected %d updates, but only %d were modified", expectedUpdates, result.ModifiedCount)
	}

	return nil
}

// ValidateVehicleExists checks if a vehicle exists before processing updates
func (vra *VehicleRepositoryAdapter) ValidateVehicleExists(vehicleID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(vehicleID)
	if err != nil {
		return fmt.Errorf("invalid vehicle ID %s: %w", vehicleID, err)
	}

	count, err := vra.collection.CountDocuments(ctx, bson.M{"_id": objectID})
	if err != nil {
		return fmt.Errorf("failed to validate vehicle %s: %w", vehicleID, err)
	}

	if count == 0 {
		return errors.New("vehicle not found")
	}

	return nil
}

// GetVehicleLastUpdate returns the last update timestamp for a vehicle
func (vra *VehicleRepositoryAdapter) GetVehicleLastUpdate(vehicleID string) (time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(vehicleID)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid vehicle ID %s: %w", vehicleID, err)
	}

	var result struct {
		LastUpdate time.Time `bson:"last_update"`
	}

	err = vra.collection.FindOne(
		ctx,
		bson.M{"_id": objectID},
		options.FindOne().SetProjection(bson.M{"last_update": 1}),
	).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return time.Time{}, errors.New("vehicle not found")
		}
		return time.Time{}, fmt.Errorf("failed to get last update for vehicle %s: %w", vehicleID, err)
	}

	return result.LastUpdate, nil
}