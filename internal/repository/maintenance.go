package repository

import (
	"context"
	"fleet-backend/internal/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MaintenanceRepository struct {
	collection         *mongo.Collection
	scheduleCollection *mongo.Collection
	reminderCollection *mongo.Collection
}

func NewMaintenanceRepository(db *mongo.Database) *MaintenanceRepository {
	return &MaintenanceRepository{
		collection:         db.Collection("maintenance_records"),
		scheduleCollection: db.Collection("maintenance_schedules"),
		reminderCollection: db.Collection("service_reminders"),
	}
}

// Maintenance Records
func (r *MaintenanceRepository) Create(record *models.MaintenanceRecord) error {
	record.ID = primitive.NewObjectID()
	record.CreatedAt = time.Now()
	record.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(context.Background(), record)
	return err
}

func (r *MaintenanceRepository) FindByID(id string) (*models.MaintenanceRecord, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var record models.MaintenanceRecord
	err = r.collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&record)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (r *MaintenanceRepository) FindByVehicleID(vehicleID string) ([]*models.MaintenanceRecord, error) {
	objectID, err := primitive.ObjectIDFromHex(vehicleID)
	if err != nil {
		return nil, err
	}

	cursor, err := r.collection.Find(context.Background(), bson.M{"vehicle_id": objectID}, options.Find().SetSort(bson.D{{Key: "performed_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var records []*models.MaintenanceRecord
	for cursor.Next(context.Background()) {
		var record models.MaintenanceRecord
		if err := cursor.Decode(&record); err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	return records, nil
}

func (r *MaintenanceRepository) FindAll(limit, offset int) ([]*models.MaintenanceRecord, error) {
	opts := options.Find().SetSort(bson.D{{Key: "performed_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}

	cursor, err := r.collection.Find(context.Background(), bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var records []*models.MaintenanceRecord
	for cursor.Next(context.Background()) {
		var record models.MaintenanceRecord
		if err := cursor.Decode(&record); err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	return records, nil
}

func (r *MaintenanceRepository) Update(id string, record *models.MaintenanceRecord) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	record.UpdatedAt = time.Now()
	update := bson.M{"$set": record}

	_, err = r.collection.UpdateOne(context.Background(), bson.M{"_id": objectID}, update)
	return err
}

func (r *MaintenanceRepository) Delete(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(context.Background(), bson.M{"_id": objectID})
	return err
}

// Maintenance Schedules
func (r *MaintenanceRepository) CreateSchedule(schedule *models.MaintenanceSchedule) error {
	schedule.ID = primitive.NewObjectID()
	schedule.CreatedAt = time.Now()
	schedule.UpdatedAt = time.Now()

	_, err := r.scheduleCollection.InsertOne(context.Background(), schedule)
	return err
}

func (r *MaintenanceRepository) FindScheduleByID(id string) (*models.MaintenanceSchedule, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var schedule models.MaintenanceSchedule
	err = r.scheduleCollection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&schedule)
	if err != nil {
		return nil, err
	}

	return &schedule, nil
}

func (r *MaintenanceRepository) FindSchedulesByVehicleID(vehicleID string) ([]*models.MaintenanceSchedule, error) {
	objectID, err := primitive.ObjectIDFromHex(vehicleID)
	if err != nil {
		return nil, err
	}

	cursor, err := r.scheduleCollection.Find(context.Background(), bson.M{"vehicle_id": objectID}, options.Find().SetSort(bson.D{{Key: "next_service_date", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var schedules []*models.MaintenanceSchedule
	for cursor.Next(context.Background()) {
		var schedule models.MaintenanceSchedule
		if err := cursor.Decode(&schedule); err != nil {
			return nil, err
		}
		schedules = append(schedules, &schedule)
	}

	return schedules, nil
}

func (r *MaintenanceRepository) FindUpcomingSchedules(days int) ([]*models.MaintenanceSchedule, error) {
	now := time.Now()
	
	var filter bson.M
	
	if days > 0 {
		// If days is specified, find schedules within that timeframe
		futureDate := now.AddDate(0, 0, days)
		filter = bson.M{
			"is_active": true,
			"$or": []bson.M{
				{
					"next_service_date": bson.M{
						"$gte": now,
						"$lte": futureDate,
					},
				},
				{
					"next_service_date": bson.M{"$exists": false},
				},
				{
					"next_service_date": nil,
				},
			},
		}
	} else {
		// If no days specified, get all future schedules
		filter = bson.M{
			"is_active": true,
			"$or": []bson.M{
				{
					"next_service_date": bson.M{"$gte": now},
				},
				{
					"next_service_date": bson.M{"$exists": false},
				},
				{
					"next_service_date": nil,
				},
			},
		}
	}

	cursor, err := r.scheduleCollection.Find(context.Background(), filter, options.Find().SetSort(bson.D{{Key: "next_service_date", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var schedules []*models.MaintenanceSchedule
	for cursor.Next(context.Background()) {
		var schedule models.MaintenanceSchedule
		if err := cursor.Decode(&schedule); err != nil {
			return nil, err
		}
		schedules = append(schedules, &schedule)
	}

	return schedules, nil
}

func (r *MaintenanceRepository) FindAllSchedules() ([]*models.MaintenanceSchedule, error) {
	cursor, err := r.scheduleCollection.Find(context.Background(), bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var schedules []*models.MaintenanceSchedule
	for cursor.Next(context.Background()) {
		var schedule models.MaintenanceSchedule
		if err := cursor.Decode(&schedule); err != nil {
			return nil, err
		}
		schedules = append(schedules, &schedule)
	}

	return schedules, nil
}

func (r *MaintenanceRepository) UpdateSchedule(id string, schedule *models.MaintenanceSchedule) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	schedule.UpdatedAt = time.Now()
	update := bson.M{"$set": schedule}

	_, err = r.scheduleCollection.UpdateOne(context.Background(), bson.M{"_id": objectID}, update)
	return err
}

func (r *MaintenanceRepository) DeleteSchedule(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.scheduleCollection.DeleteOne(context.Background(), bson.M{"_id": objectID})
	return err
}

// Service Reminders
func (r *MaintenanceRepository) CreateReminder(reminder *models.ServiceReminder) error {
	reminder.ID = primitive.NewObjectID()
	reminder.CreatedAt = time.Now()
	reminder.UpdatedAt = time.Now()

	_, err := r.reminderCollection.InsertOne(context.Background(), reminder)
	return err
}

func (r *MaintenanceRepository) FindRemindersByVehicleID(vehicleID string) ([]*models.ServiceReminder, error) {
	objectID, err := primitive.ObjectIDFromHex(vehicleID)
	if err != nil {
		return nil, err
	}

	cursor, err := r.reminderCollection.Find(context.Background(), bson.M{"vehicle_id": objectID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var reminders []*models.ServiceReminder
	for cursor.Next(context.Background()) {
		var reminder models.ServiceReminder
		if err := cursor.Decode(&reminder); err != nil {
			return nil, err
		}
		reminders = append(reminders, &reminder)
	}

	return reminders, nil
}

func (r *MaintenanceRepository) FindOverdueReminders() ([]*models.ServiceReminder, error) {
	filter := bson.M{"is_overdue": true}

	cursor, err := r.reminderCollection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var reminders []*models.ServiceReminder
	for cursor.Next(context.Background()) {
		var reminder models.ServiceReminder
		if err := cursor.Decode(&reminder); err != nil {
			return nil, err
		}
		reminders = append(reminders, &reminder)
	}

	return reminders, nil
}

func (r *MaintenanceRepository) UpdateReminder(id string, reminder *models.ServiceReminder) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	reminder.UpdatedAt = time.Now()
	update := bson.M{"$set": reminder}

	_, err = r.reminderCollection.UpdateOne(context.Background(), bson.M{"_id": objectID}, update)
	return err
}

func (r *MaintenanceRepository) DeleteReminder(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.reminderCollection.DeleteOne(context.Background(), bson.M{"_id": objectID})
	return err
}