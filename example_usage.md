# Multiple Service Types - Usage Examples

## Overview
The maintenance system now supports multiple service types per maintenance record, allowing you to perform multiple services during a single visit.

## API Changes

### Creating a Maintenance Record (Before)
```json
{
  "vehicleId": "507f1f77bcf86cd799439011",
  "type": "oil_change",
  "description": "Regular oil change",
  "cost": 50.00,
  "currency": "USD",
  "serviceCenter": "AutoCare Plus",
  "performedAt": "2024-01-15T10:00:00Z",
  "odometer": 45000,
  "partsReplaced": ["oil_filter", "engine_oil"],
  "status": "completed"
}
```

### Creating a Maintenance Record (After)
```json
{
  "vehicleId": "507f1f77bcf86cd799439011",
  "types": ["oil_change", "tire_rotation", "brake_service"],
  "description": "Comprehensive service - oil change, tire rotation, and brake inspection",
  "cost": 150.00,
  "currency": "USD",
  "serviceCenter": "AutoCare Plus",
  "performedAt": "2024-01-15T10:00:00Z",
  "odometer": 45000,
  "partsReplaced": ["engine_oil", "oil_filter", "brake_pads", "brake_fluid"],
  "notes": "Replaced worn brake pads and performed routine oil change",
  "status": "completed"
}
```

## Benefits

1. **Efficiency**: Perform multiple services in one visit
2. **Cost Tracking**: Track combined costs for multiple services
3. **Better Scheduling**: Service intervals are calculated based on the shortest interval among selected services
4. **Comprehensive Records**: Single record captures all work performed

## Service Interval Calculation

When multiple service types are selected, the system uses the **shortest interval** among all types to ensure no service is missed:

- Oil Change: 10,000 km
- Tire Rotation: 15,000 km  
- Brake Service: 30,000 km

If you select all three, the next service will be scheduled at 10,000 km (shortest interval).

## Example Service Combinations

### Basic Maintenance
```json
{
  "types": ["oil_change", "air_filter"],
  "description": "Basic maintenance package"
}
```

### Comprehensive Service
```json
{
  "types": ["oil_change", "tire_rotation", "brake_service", "transmission_service"],
  "description": "Full vehicle inspection and service"
}
```

### Seasonal Preparation
```json
{
  "types": ["coolant_flush", "battery_replacement", "inspection"],
  "description": "Winter preparation service"
}
```

## Parts Replacement Tracking

The `partsReplaced` field allows you to document exactly what parts were replaced during maintenance. This provides detailed records of what work was performed.

### Example with Detailed Parts Tracking
```json
{
  "vehicleId": "507f1f77bcf86cd799439011",
  "types": ["oil_change", "brake_service", "engine_tune_up"],
  "description": "Major service - oil change, brake service, and engine tune-up",
  "cost": 350.00,
  "currency": "USD",
  "serviceCenter": "Premium Auto Service",
  "performedAt": "2024-01-15T10:00:00Z",
  "odometer": 45000,
  "partsReplaced": [
    "engine_oil",
    "oil_filter", 
    "brake_pads",
    "brake_discs",
    "brake_fluid",
    "spark_plugs",
    "air_filter"
  ],
  "notes": "Replaced worn brake components and performed comprehensive engine service",
  "status": "completed"
}
```

### Common Parts by Service Type

**Oil Change:**
- `engine_oil`
- `oil_filter`

**Brake Service:**
- `brake_pads`
- `brake_discs`
- `brake_fluid`

**Engine Tune-up:**
- `spark_plugs`
- `air_filter`
- `fuel_filter`

**Transmission Service:**
- `transmission_oil`

**Battery Replacement:**
- `battery`

**Belt Replacement:**
- `timing_belt`
- `serpentine_belt`

**Coolant Service:**
- `coolant`
- `thermostat`
- `water_pump`

### Available Part Constants

Engine Components:
- `engine_oil`, `oil_filter`, `air_filter`, `fuel_filter`, `spark_plugs`

Brake System:
- `brake_pads`, `brake_discs`, `brake_fluid`

Transmission:
- `transmission_oil`, `clutch`

Cooling System:
- `coolant`, `radiator`, `thermostat`, `water_pump`

Electrical:
- `battery`, `alternator`, `starter`

Belts & Hoses:
- `timing_belt`, `serpentine_belt`

Suspension:
- `shock_absorbers`, `struts`

Tires & Wheels:
- `tires`

Other:
- `exhaust_system`, `catalytic_converter`, `wiper_blades`, `headlights`, `taillights`

## Available Service Types

- `oil_change` (10,000 km)
- `tire_rotation` (15,000 km)
- `brake_service` (30,000 km)
- `transmission_service` (60,000 km)
- `engine_tune_up` (40,000 km)
- `battery_replacement` (50,000 km)
- `air_filter` (20,000 km)
- `fuel_filter` (40,000 km)
- `coolant_flush` (80,000 km)
- `spark_plugs` (30,000 km)
- `belt_replacement` (100,000 km)
- `inspection` (20,000 km)
- `repair`
- `other`