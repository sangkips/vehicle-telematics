package repository

import (
	"context"
	"errors"
	"fleet-backend/internal/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type VehicleRepository struct {
	collection *mongo.Collection
}

func NewVehicleRepository(db *mongo.Database) *VehicleRepository {
	return &VehicleRepository{
		collection: db.Collection("vehicles"),
	}
}

func (r *VehicleRepository) Create(vehicle *models.Vehicle) (*models.Vehicle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.collection.InsertOne(ctx, vehicle)
	if err != nil {
		return nil, err
	}

	vehicle.ID = result.InsertedID.(primitive.ObjectID)
	return vehicle, nil
}

func (r *VehicleRepository) FindByID(id string) (*models.Vehicle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid vehicle ID")
	}

	var vehicle models.Vehicle
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&vehicle)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("vehicle not found")
		}
		return nil, err
	}

	return &vehicle, nil
}

func (r *VehicleRepository) FindByPlateNumber(plateNumber string) (*models.Vehicle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var vehicle models.Vehicle
	err := r.collection.FindOne(ctx, bson.M{"plate_number": plateNumber}).Decode(&vehicle)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("vehicle not found")
		}
		return nil, err
	}

	return &vehicle, nil
}

func (r *VehicleRepository) FindAll() ([]*models.Vehicle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Sort by last_update descending to get most recent updates first
	opts := options.Find().SetSort(bson.D{{Key: "last_update", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var vehicles []*models.Vehicle
	for cursor.Next(ctx) {
		var vehicle models.Vehicle
		if err := cursor.Decode(&vehicle); err != nil {
			return nil, err
		}
		vehicles = append(vehicles, &vehicle)
	}

	return vehicles, nil
}

func (r *VehicleRepository) FindByStatus(status string) ([]*models.Vehicle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"status": status})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var vehicles []*models.Vehicle
	for cursor.Next(ctx) {
		var vehicle models.Vehicle
		if err := cursor.Decode(&vehicle); err != nil {
			return nil, err
		}
		vehicles = append(vehicles, &vehicle)
	}

	return vehicles, nil
}

func (r *VehicleRepository) FindByDriver(driver string) ([]*models.Vehicle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"driver": driver})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var vehicles []*models.Vehicle
	for cursor.Next(ctx) {
		var vehicle models.Vehicle
		if err := cursor.Decode(&vehicle); err != nil {
			return nil, err
		}
		vehicles = append(vehicles, &vehicle)
	}

	return vehicles, nil
}

func (r *VehicleRepository) FindByFuelLevelBelow(threshold float64) ([]*models.Vehicle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"fuel_level": bson.M{"$lt": threshold}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var vehicles []*models.Vehicle
	for cursor.Next(ctx) {
		var vehicle models.Vehicle
		if err := cursor.Decode(&vehicle); err != nil {
			return nil, err
		}
		vehicles = append(vehicles, &vehicle)
	}

	return vehicles, nil
}

func (r *VehicleRepository) FindInLocationRadius(lat, lng, radiusKm float64) ([]*models.Vehicle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Simple radius calculation (for more precise geospatial queries, use MongoDB's geospatial features)
	latRange := radiusKm / 111.0 // Approximate km per degree latitude
	lngRange := radiusKm / (111.0 * 0.7)  // Approximate, varies by latitude

	filter := bson.M{
		"location.lat": bson.M{
			"$gte": lat - latRange,
			"$lte": lat + latRange,
		},
		"location.lng": bson.M{
			"$gte": lng - lngRange,
			"$lte": lng + lngRange,
		},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var vehicles []*models.Vehicle
	for cursor.Next(ctx) {
		var vehicle models.Vehicle
		if err := cursor.Decode(&vehicle); err != nil {
			return nil, err
		}
		vehicles = append(vehicles, &vehicle)
	}

	return vehicles, nil
}

func (r *VehicleRepository) Update(id string, vehicle *models.Vehicle) (*models.Vehicle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid vehicle ID")
	}

	vehicle.UpdatedAt = time.Now()

	update := bson.M{
		"$set": vehicle,
	}

	result := r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objectID},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var updatedVehicle models.Vehicle
	if err := result.Decode(&updatedVehicle); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("vehicle not found")
		}
		return nil, err
	}

	return &updatedVehicle, nil
}

func (r *VehicleRepository) UpdateLocation(id string, location models.Location) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid vehicle ID")
	}

	update := bson.M{
		"$set": bson.M{
			"location":    location,
			"last_update": time.Now(),
			"updated_at":  time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("vehicle not found")
	}

	return nil
}

func (r *VehicleRepository) UpdateFuelLevel(id string, fuelLevel float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid vehicle ID")
	}

	update := bson.M{
		"$set": bson.M{
			"fuel_level":  fuelLevel,
			"last_update": time.Now(),
			"updated_at":  time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("vehicle not found")
	}

	return nil
}

func (r *VehicleRepository) UpdateStatus(id string, status string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid vehicle ID")
	}

	update := bson.M{
		"$set": bson.M{
			"status":      status,
			"last_update": time.Now(),
			"updated_at":  time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("vehicle not found")
	}

	return nil
}

func (r *VehicleRepository) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid vehicle ID")
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("vehicle not found")
	}

	return nil
}

func (r *VehicleRepository) Count() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{})
	return count, err
}

func (r *VehicleRepository) CountByStatus(status string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{"status": status})
	return count, err
}

func (r *VehicleRepository) GetFleetStatistics() (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id": nil,
				"total_vehicles": bson.M{"$sum": 1},
				"avg_fuel_level": bson.M{"$avg": "$fuel_level"},
				"total_distance": bson.M{"$sum": "$odometer"},
				"avg_fuel_consumption": bson.M{"$avg": "$fuel_consumption"},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result map[string]interface{}
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// CreateIndexes creates necessary indexes for the vehicles collection
func (r *VehicleRepository) CreateIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "plate_number", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "driver", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "fuel_level", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "last_update", Value: -1}},
		},
		{
			Keys: bson.D{
				{Key: "location.lat", Value: "2d"},
				{Key: "location.lng", Value: "2d"},
			},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: 1}},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}