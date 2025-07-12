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

type AlertRepository struct {
	collection *mongo.Collection
}

func NewAlertRepository(db *mongo.Database) *AlertRepository {
	return &AlertRepository{
		collection: db.Collection("alerts"),
	}
}

func (r *AlertRepository) Create(alert *models.Alert) (*models.Alert, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.collection.InsertOne(ctx, alert)
	if err != nil {
		return nil, err
	}

	alert.ID = result.InsertedID.(primitive.ObjectID)
	return alert, nil
}

func (r *AlertRepository) FindByID(id string) (*models.Alert, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid alert ID")
	}

	var alert models.Alert
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&alert)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("alert not found")
		}
		return nil, err
	}

	return &alert, nil
}

func (r *AlertRepository) FindAll() ([]*models.Alert, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Sort by timestamp descending to get most recent alerts first
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var alerts []*models.Alert
	for cursor.Next(ctx) {
		var alert models.Alert
		if err := cursor.Decode(&alert); err != nil {
			return nil, err
		}
		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

func (r *AlertRepository) FindByVehicleID(vehicleID string) ([]*models.Alert, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"vehicle_id": vehicleID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var alerts []*models.Alert
	for cursor.Next(ctx) {
		var alert models.Alert
		if err := cursor.Decode(&alert); err != nil {
			return nil, err
		}
		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

func (r *AlertRepository) FindByType(alertType string) ([]*models.Alert, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"type": alertType}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var alerts []*models.Alert
	for cursor.Next(ctx) {
		var alert models.Alert
		if err := cursor.Decode(&alert); err != nil {
			return nil, err
		}
		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

func (r *AlertRepository) FindBySeverity(severity string) ([]*models.Alert, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"severity": severity}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var alerts []*models.Alert
	for cursor.Next(ctx) {
		var alert models.Alert
		if err := cursor.Decode(&alert); err != nil {
			return nil, err
		}
		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

func (r *AlertRepository) FindUnresolved() ([]*models.Alert, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"resolved": false}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var alerts []*models.Alert
	for cursor.Next(ctx) {
		var alert models.Alert
		if err := cursor.Decode(&alert); err != nil {
			return nil, err
		}
		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

func (r *AlertRepository) FindResolved() ([]*models.Alert, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "resolved_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"resolved": true}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var alerts []*models.Alert
	for cursor.Next(ctx) {
		var alert models.Alert
		if err := cursor.Decode(&alert); err != nil {
			return nil, err
		}
		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

func (r *AlertRepository) FindByDateRange(startDate, endDate time.Time) ([]*models.Alert, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"timestamp": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var alerts []*models.Alert
	for cursor.Next(ctx) {
		var alert models.Alert
		if err := cursor.Decode(&alert); err != nil {
			return nil, err
		}
		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

func (r *AlertRepository) FindCriticalAlerts() ([]*models.Alert, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"severity": "critical",
		"resolved": false,
	}

	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var alerts []*models.Alert
	for cursor.Next(ctx) {
		var alert models.Alert
		if err := cursor.Decode(&alert); err != nil {
			return nil, err
		}
		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

func (r *AlertRepository) Update(id string, alert *models.Alert) (*models.Alert, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid alert ID")
	}

	update := bson.M{
		"$set": alert,
	}

	result := r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objectID},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var updatedAlert models.Alert
	if err := result.Decode(&updatedAlert); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("alert not found")
		}
		return nil, err
	}

	return &updatedAlert, nil
}

func (r *AlertRepository) MarkAsResolved(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid alert ID")
	}

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"resolved":    true,
			"resolved_at": now,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("alert not found")
	}

	return nil
}

func (r *AlertRepository) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid alert ID")
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("alert not found")
	}

	return nil
}

func (r *AlertRepository) DeleteResolvedBefore(cutoffDate time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"resolved": true,
		"resolved_at": bson.M{
			"$lt": cutoffDate,
		},
	}

	_, err := r.collection.DeleteMany(ctx, filter)
	return err
}

func (r *AlertRepository) Count() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{})
	return count, err
}

func (r *AlertRepository) CountUnresolved() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{"resolved": false})
	return count, err
}

func (r *AlertRepository) CountByType(alertType string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{"type": alertType})
	return count, err
}

func (r *AlertRepository) CountBySeverity(severity string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{"severity": severity})
	return count, err
}

func (r *AlertRepository) CountByVehicle(vehicleID string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{"vehicle_id": vehicleID})
	return count, err
}

func (r *AlertRepository) GetAlertStatistics() (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id": nil,
				"total_alerts": bson.M{"$sum": 1},
				"resolved_alerts": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							"$resolved",
							1,
							0,
						},
					},
				},
				"unresolved_alerts": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$resolved", false}},
							1,
							0,
						},
					},
				},
				"critical_alerts": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$severity", "critical"}},
							1,
							0,
						},
					},
				},
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

// CreateIndexes creates necessary indexes for the alerts collection
func (r *AlertRepository) CreateIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "vehicle_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "type", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "severity", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "resolved", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "timestamp", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "resolved_at", Value: -1}},
		},
		{
			Keys: bson.D{
				{Key: "vehicle_id", Value: 1},
				{Key: "resolved", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "type", Value: 1},
				{Key: "resolved", Value: 1},
			},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}