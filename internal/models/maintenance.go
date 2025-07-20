package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MaintenanceRecord struct {
	ID                   primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	VehicleID            primitive.ObjectID `json:"vehicleId" bson:"vehicle_id"`
	Types                []string           `json:"types" bson:"types"`
	Description          string             `json:"description" bson:"description"`
	Cost                 float64            `json:"cost" bson:"cost"`
	Currency             string             `json:"currency" bson:"currency"`
	ServiceCenter        string             `json:"serviceCenter" bson:"service_center"`
	PerformedAt          time.Time          `json:"performedAt" bson:"performed_at"`
	Odometer             int                `json:"odometer" bson:"odometer"`
	ServiceInterval      int                `json:"serviceInterval" bson:"service_interval"` // km between services
	NextServiceOdometer  int                `json:"nextServiceOdometer" bson:"next_service_odometer"`
	NextServiceDate      *time.Time         `json:"nextServiceDate,omitempty" bson:"next_service_date,omitempty"`
	PartsReplaced        []string           `json:"partsReplaced" bson:"parts_replaced"`
	Notes                string             `json:"notes,omitempty" bson:"notes,omitempty"`
	Status               string             `json:"status" bson:"status"`
	CreatedAt            time.Time          `json:"createdAt" bson:"created_at"`
	UpdatedAt            time.Time          `json:"updatedAt" bson:"updated_at"`
}

type MaintenanceSchedule struct {
	ID                   primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	VehicleID            primitive.ObjectID `json:"vehicleId" bson:"vehicle_id"`
	Types                []string           `json:"types" bson:"types"`
	Description          string             `json:"description" bson:"description"`
	IntervalKm           int                `json:"intervalKm" bson:"interval_km"`           // service every X km (required)
	IntervalDays         *int               `json:"intervalDays,omitempty" bson:"interval_days,omitempty"` // optional: service every X days
	LastServiceOdometer  int                `json:"lastServiceOdometer" bson:"last_service_odometer"`
	LastServiceDate      time.Time          `json:"lastServiceDate" bson:"last_service_date"`
	NextServiceOdometer  int                `json:"nextServiceOdometer" bson:"next_service_odometer"` // calculated
	NextServiceDate      *time.Time         `json:"nextServiceDate,omitempty" bson:"next_service_date,omitempty"` // estimated
	ServiceCenterName    string             `json:"serviceCenterName" bson:"service_center_name"`
	IsActive             bool               `json:"isActive" bson:"is_active"`
	CreatedAt            time.Time          `json:"createdAt" bson:"created_at"`
	UpdatedAt            time.Time          `json:"updatedAt" bson:"updated_at"`
}

type ServiceReminder struct {
	ID                primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	VehicleID         primitive.ObjectID `json:"vehicleId" bson:"vehicle_id"`
	Types             []string           `json:"types" bson:"types"`
	DueDate           *time.Time         `json:"dueDate,omitempty" bson:"due_date,omitempty"`
	DueOdometer       *int               `json:"dueOdometer,omitempty" bson:"due_odometer,omitempty"`
	CurrentOdometer   int                `json:"currentOdometer" bson:"current_odometer"`
	DaysUntilDue      *int               `json:"daysUntilDue,omitempty" bson:"days_until_due,omitempty"`
	OdometerUntilDue  *int               `json:"odometerUntilDue,omitempty" bson:"odometer_until_due,omitempty"`
	Priority          string             `json:"priority" bson:"priority"`
	IsOverdue         bool               `json:"isOverdue" bson:"is_overdue"`
	CreatedAt         time.Time          `json:"createdAt" bson:"created_at"`
	UpdatedAt         time.Time          `json:"updatedAt" bson:"updated_at"`
}

// Constants for maintenance types
const (
	MaintenanceTypeOilChange          = "oil_change"
	MaintenanceTypeTireRotation       = "tire_rotation"
	MaintenanceTypeBrakeService       = "brake_service"
	MaintenanceTypeTransmissionService = "transmission_service"
	MaintenanceTypeEngineTuneUp       = "engine_tune_up"
	MaintenanceTypeBatteryReplacement = "battery_replacement"
	MaintenanceTypeAirFilter          = "air_filter"
	MaintenanceTypeFuelFilter         = "fuel_filter"
	MaintenanceTypeCoolantFlush       = "coolant_flush"
	MaintenanceTypeSparkPlugs         = "spark_plugs"
	MaintenanceTypeBeltReplacement    = "belt_replacement"
	MaintenanceTypeInspection         = "inspection"
	MaintenanceTypeRepair             = "repair"
	MaintenanceTypeOther              = "other"
)

// Constants for maintenance status
const (
	MaintenanceStatusScheduled  = "scheduled"
	MaintenanceStatusInProgress = "in_progress"
	MaintenanceStatusCompleted  = "completed"
	MaintenanceStatusCancelled  = "cancelled"
)

// Constants for schedule status
const (
	ScheduleStatusPending   = "pending"
	ScheduleStatusScheduled = "scheduled"
	ScheduleStatusOverdue   = "overdue"
)

// Constants for priority levels
const (
	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"
)

// ServiceIntervalConfig defines default service intervals for different maintenance types
type ServiceIntervalConfig struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Types        []string           `json:"types" bson:"types"`
	IntervalKm   int                `json:"intervalKm" bson:"interval_km"`
	IntervalDays int                `json:"intervalDays" bson:"interval_days"`
	Description  string             `json:"description" bson:"description"`
	CreatedAt    time.Time          `json:"createdAt" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updatedAt" bson:"updated_at"`
}

// Constants for common parts that can be replaced
const (
	PartEngineOil        = "engine_oil"
	PartOilFilter        = "oil_filter"
	PartAirFilter        = "air_filter"
	PartFuelFilter       = "fuel_filter"
	PartSparkPlugs       = "spark_plugs"
	PartBrakePads        = "brake_pads"
	PartBrakeDiscs       = "brake_discs"
	PartBrakeFluid       = "brake_fluid"
	PartTransmissionOil  = "transmission_oil"
	PartCoolant          = "coolant"
	PartBattery          = "battery"
	PartTires            = "tires"
	PartTimingBelt       = "timing_belt"
	PartSerpentineBelt   = "serpentine_belt"
	PartAlternator       = "alternator"
	PartStarter          = "starter"
	PartRadiator         = "radiator"
	PartThermostat       = "thermostat"
	PartWaterPump        = "water_pump"
	PartFuelPump         = "fuel_pump"
	PartClutch           = "clutch"
	PartShockAbsorbers   = "shock_absorbers"
	PartStruts           = "struts"
	PartWiperBlades      = "wiper_blades"
	PartHeadlights       = "headlights"
	PartTaillights       = "taillights"
	PartExhaustSystem    = "exhaust_system"
	PartCatalyticConverter = "catalytic_converter"
	PartOxygenSensor     = "oxygen_sensor"
	PartMassAirflowSensor = "mass_airflow_sensor"
	PartOther            = "other"
)

// Default service intervals (in kilometers)
var DefaultServiceIntervals = map[string]int{
	MaintenanceTypeOilChange:           10000, // Every 10,000 km
	MaintenanceTypeTireRotation:        15000, // Every 15,000 km
	MaintenanceTypeBrakeService:        30000, // Every 30,000 km
	MaintenanceTypeTransmissionService: 60000, // Every 60,000 km
	MaintenanceTypeEngineTuneUp:        40000, // Every 40,000 km
	MaintenanceTypeBatteryReplacement:  50000, // Every 50,000 km or 3 years
	MaintenanceTypeAirFilter:           20000, // Every 20,000 km
	MaintenanceTypeFuelFilter:          40000, // Every 40,000 km
	MaintenanceTypeCoolantFlush:        80000, // Every 80,000 km
	MaintenanceTypeSparkPlugs:          30000, // Every 30,000 km
	MaintenanceTypeBeltReplacement:     100000, // Every 100,000 km
	MaintenanceTypeInspection:          20000, // Every 20,000 km
}

// CommonPartsForService maps service types to commonly replaced parts
var CommonPartsForService = map[string][]string{
	MaintenanceTypeOilChange: {
		PartEngineOil,
		PartOilFilter,
	},
	MaintenanceTypeBrakeService: {
		PartBrakePads,
		PartBrakeDiscs,
		PartBrakeFluid,
	},
	MaintenanceTypeTransmissionService: {
		PartTransmissionOil,
	},
	MaintenanceTypeEngineTuneUp: {
		PartSparkPlugs,
		PartAirFilter,
		PartFuelFilter,
	},
	MaintenanceTypeBatteryReplacement: {
		PartBattery,
	},
	MaintenanceTypeAirFilter: {
		PartAirFilter,
	},
	MaintenanceTypeFuelFilter: {
		PartFuelFilter,
	},
	MaintenanceTypeCoolantFlush: {
		PartCoolant,
		PartThermostat,
	},
	MaintenanceTypeSparkPlugs: {
		PartSparkPlugs,
	},
	MaintenanceTypeBeltReplacement: {
		PartTimingBelt,
		PartSerpentineBelt,
	},
	MaintenanceTypeTireRotation: {
		// Usually no parts replaced, just service
	},
}