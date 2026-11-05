package fleet

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/asset"
	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/hr"
)

// FleetVehicleModel corresponds to standard enterprise fleet.vehicle.model
type FleetVehicleModel struct {
	models.ModelBase
	Brand       string `gorm:"size:64;not null;index" json:"brand"`       // e.g. "Toyota", "Mitsubishi", "Isuzu"
	ModelName   string `gorm:"size:128;not null" json:"model_name"`       // e.g. "Canter FE 74 HD", "Gran Max Blind Van"
	VehicleType string `gorm:"size:32;default:'van'" json:"vehicle_type"` // "car", "truck", "van", "motorcycle"
}

// FleetVehicle corresponds to standard enterprise fleet.vehicle
type FleetVehicle struct {
	models.ModelBase
	LicensePlate     string             `gorm:"size:32;not null;uniqueIndex" json:"license_plate"` // e.g. "B 1234 XYZ"
	ModelID          uint               `gorm:"not null;index" json:"model_id"`
	Model            *FleetVehicleModel `gorm:"foreignKey:ModelID" json:"model,omitempty"`
	DriverEmployeeID *uint              `gorm:"index" json:"driver_employee_id,omitempty"`
	Driver           *hr.Employee       `gorm:"foreignKey:DriverEmployeeID" json:"driver,omitempty"`
	PlanPermitNumber string             `gorm:"size:64" json:"plan_permit_number,omitempty"` // STNK / Uji KIR
	AcquisitionDate  string             `gorm:"size:32" json:"acquisition_date,omitempty"`
	VinSN            string             `gorm:"size:64" json:"vin_sn,omitempty"`        // Chassis / No Rangka
	EngineNumber     string             `gorm:"size:64" json:"engine_number,omitempty"` // No Mesin
	Location         string             `gorm:"size:255" json:"location,omitempty"`     // e.g. "Gudang Utama Jakarta"
	AssetID          *uint              `gorm:"index" json:"asset_id,omitempty"`        // Fixed Asset link
	Asset            *asset.Asset       `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
	Status           string             `gorm:"size:32;default:'active';index" json:"status"` // "active", "in_service", "inactive"
	LastOdometer     float64            `gorm:"default:0" json:"last_odometer"`               // in KM
	FuelType         string             `gorm:"size:32;default:'diesel'" json:"fuel_type"`    // "diesel", "gasoline", "electric"
	Color            string             `gorm:"size:32;default:'White'" json:"color,omitempty"`
	Seats            int                `gorm:"default:2" json:"seats"`
	Note             string             `gorm:"type:text" json:"note,omitempty"`
}

// FleetOdometerLog corresponds to standard enterprise fleet.vehicle.odometer
type FleetOdometerLog struct {
	models.ModelBase
	VehicleID        uint          `gorm:"not null;index" json:"vehicle_id"`
	Vehicle          *FleetVehicle `gorm:"foreignKey:VehicleID" json:"vehicle,omitempty"`
	Date             string        `gorm:"size:32;not null" json:"date"`
	Value            float64       `gorm:"not null" json:"value"` // KM Reading
	DriverEmployeeID *uint         `json:"driver_employee_id,omitempty"`
	Driver           *hr.Employee  `gorm:"foreignKey:DriverEmployeeID" json:"driver,omitempty"`
	Note             string        `gorm:"size:255" json:"note,omitempty"`
}

// FleetServiceLog corresponds to standard enterprise fleet.vehicle.log.services
type FleetServiceLog struct {
	models.ModelBase
	ServiceNumber   string        `gorm:"size:64;not null;uniqueIndex" json:"service_number"` // e.g. SRV/FLEET/2026/00001
	VehicleID       uint          `gorm:"not null;index" json:"vehicle_id"`
	Vehicle         *FleetVehicle `gorm:"foreignKey:VehicleID" json:"vehicle,omitempty"`
	Date            string        `gorm:"size:32;not null" json:"date"`
	ServiceType     string        `gorm:"size:64;not null;index" json:"service_type"` // "oil_change", "tire_replacement", "general_servicing", "periodic_inspection_kir", "repair"
	Cost            float64       `gorm:"default:0" json:"cost"`
	VendorPartnerID *uint         `gorm:"index" json:"vendor_partner_id,omitempty"` // Workshop / bengkel
	Vendor          *base.Partner `gorm:"foreignKey:VendorPartnerID" json:"vendor,omitempty"`
	Odometer        float64       `gorm:"default:0" json:"odometer"`
	Stage           string        `gorm:"size:32;default:'done';index" json:"stage"` // "draft", "done", "cancelled"
	Description     string        `gorm:"size:255" json:"description"`
	Notes           string        `gorm:"type:text" json:"notes,omitempty"`
}
