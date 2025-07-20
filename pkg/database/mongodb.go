package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
)

// Connect establishes a connection to MongoDB

func Connect(mongoURI string) (*mongo.Database, error) {
	// Parse the URI to extract database name
	cs, err := connstring.ParseAndValidate(mongoURI)
	if err != nil {
		return nil, fmt.Errorf("invalid MongoDB URI: %v", err)
	}

	// Set client options
	clientOptions := options.Client().ApplyURI(mongoURI)

	// Set connection timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping the database to verify connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	log.Println("Successfully connected to MongoDB")

	// Use database name from URI or default to "fleet_management"
	dbName := cs.Database
	if dbName == "" {
		dbName = "fleet_management"
	}

	db := client.Database(dbName)

	// Initialize indexes
	if err := createIndexes(db); err != nil {
		log.Printf("Warning: Failed to create indexes: %v", err)
	}

	return db, nil
}

// createIndexes creates necessary indexes for all collections
func createIndexes(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Users collection indexes
	usersCollection := db.Collection("users")
	userIndexes := []mongo.IndexModel{
		{
			Keys:    map[string]interface{}{"username": 1},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    map[string]interface{}{"email": 1},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: map[string]interface{}{"role": 1},
		},
		{
			Keys: map[string]interface{}{"status": 1},
		},
		{
			Keys: map[string]interface{}{"created_at": 1},
		},
	}

	if _, err := usersCollection.Indexes().CreateMany(ctx, userIndexes); err != nil {
		log.Printf("Failed to create user indexes: %v", err)
	}

	// Vehicles collection indexes
	vehiclesCollection := db.Collection("vehicles")
	vehicleIndexes := []mongo.IndexModel{
		{
			Keys:    map[string]interface{}{"plate_number": 1},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: map[string]interface{}{"status": 1},
		},
		{
			Keys: map[string]interface{}{"driver": 1},
		},
		{
			Keys: map[string]interface{}{"fuel_level": 1},
		},
		{
			Keys: map[string]interface{}{"last_update": -1},
		},
		{
			Keys: map[string]interface{}{
				"location.lat": "2d",
				"location.lng": "2d",
			},
		},
		{
			Keys: map[string]interface{}{"created_at": 1},
		},
	}

	if _, err := vehiclesCollection.Indexes().CreateMany(ctx, vehicleIndexes); err != nil {
		log.Printf("Failed to create vehicle indexes: %v", err)
	}

	// Alerts collection indexes
	alertsCollection := db.Collection("alerts")
	alertIndexes := []mongo.IndexModel{
		{
			Keys: map[string]interface{}{"vehicle_id": 1},
		},
		{
			Keys: map[string]interface{}{"type": 1},
		},
		{
			Keys: map[string]interface{}{"severity": 1},
		},
		{
			Keys: map[string]interface{}{"resolved": 1},
		},
		{
			Keys: map[string]interface{}{"timestamp": -1},
		},
		{
			Keys: map[string]interface{}{"resolved_at": -1},
		},
		{
			Keys: map[string]interface{}{
				"vehicle_id": 1,
				"resolved":   1,
			},
		},
		{
			Keys: map[string]interface{}{
				"type":     1,
				"resolved": 1,
			},
		},
	}

	if _, err := alertsCollection.Indexes().CreateMany(ctx, alertIndexes); err != nil {
		log.Printf("Failed to create alert indexes: %v", err)
	}

	log.Println("Database indexes created successfully")
	return nil
}

// Disconnect closes the MongoDB connection
func Disconnect(client *mongo.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %v", err)
	}

	log.Println("Disconnected from MongoDB")
	return nil
}

// Health checks the database connection health
func Health(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Client().Ping(ctx, nil)
}