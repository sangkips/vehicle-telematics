package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)


type Vehicle struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name             string             `bson:"name" json:"name" validate:"required"`
	PlateNumber      string             `bson:"plate_number" json:"plateNumber" validate:"required"`
	Driver           string             `bson:"driver" json:"driver" validate:"required"`
	FuelLevel        float64            `bson:"fuel_level" json:"fuelLevel"`
	MaxFuelCapacity  float64            `bson:"max_fuel_capacity" json:"maxFuelCapacity"`
	Location         Location           `bson:"location" json:"location"`
	Speed            int                `bson:"speed" json:"speed"`
	Status           string             `bson:"status" json:"status"`
	LastUpdate       time.Time          `bson:"last_update" json:"lastUpdate"`
	Odometer         int                `bson:"odometer" json:"odometer"`
	FuelConsumption  float64            `bson:"fuel_consumption" json:"fuelConsumption"`
	Alerts           []Alert            `bson:"alerts" json:"alerts"`
	Make             string             `bson:"make" json:"make"`
	Model            string             `bson:"model" json:"model"`
	Year             int                `bson:"year" json:"year"`
	VIN              string             `bson:"vin" json:"vin"`
	CreatedAt        time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt        time.Time          `bson:"updated_at" json:"updatedAt"`
}

type Location struct {
	Lat     float64 `bson:"lat" json:"lat"`
	Lng     float64 `bson:"lng" json:"lng"`
	Address string  `bson:"address" json:"address"`
}