package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Alert struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	VehicleID  string             `bson:"vehicle_id" json:"vehicleId" validate:"required"`
	Type       string             `bson:"type" json:"type" validate:"required,oneof=fuel_theft maintenance speeding unauthorized low_fuel"`
	Message    string             `bson:"message" json:"message" validate:"required"`
	Severity   string             `bson:"severity" json:"severity" validate:"required,oneof=low medium high critical"`
	Timestamp  time.Time          `bson:"timestamp" json:"timestamp"`
	Resolved   bool               `bson:"resolved" json:"resolved"`
	ResolvedAt *time.Time         `bson:"resolved_at,omitempty" json:"resolvedAt,omitempty"`
}