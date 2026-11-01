package fleet

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// ----------------- Vehicle Models -----------------

func (s *Service) CreateModel(m *FleetVehicleModel) error {
	if m.Brand == "" || m.ModelName == "" {
		return errors.New("brand and model name are required")
	}
	return s.db.Create(m).Error
}

func (s *Service) ListModels(companyID uint) ([]FleetVehicleModel, error) {
	var models []FleetVehicleModel
	err := s.db.Where("company_id = ?", companyID).Order("brand asc, model_name asc").Find(&models).Error
	return models, err
}

// ----------------- Vehicles -----------------

func (s *Service) CreateVehicle(v *FleetVehicle) error {
	if v.LicensePlate == "" {
		return errors.New("license plate is required")
	}
	if v.ModelID == 0 {
		return errors.New("vehicle model is required")
	}

	if v.Status == "" {
		v.Status = "active"
	}

	return s.db.Create(v).Error
}

func (s *Service) ListVehicles(companyID uint, status, vehicleType string) ([]FleetVehicle, error) {
	var list []FleetVehicle
	q := s.db.Where("company_id = ?", companyID)

	if status != "" && status != "all" {
		q = q.Where("status = ?", status)
	}

	if vehicleType != "" && vehicleType != "all" {
		q = q.Joins("JOIN fleet_vehicle_models ON fleet_vehicle_models.id = fleet_vehicles.model_id").
			Where("fleet_vehicle_models.vehicle_type = ?", vehicleType)
	}

	err := q.Preload("Model").
		Preload("Driver").
		Preload("Asset").
		Order("id desc").
		Find(&list).Error
	return list, err
}

func (s *Service) GetVehicle(id, companyID uint) (*FleetVehicle, error) {
	var v FleetVehicle
	err := s.db.Where("id = ? AND company_id = ?", id, companyID).
		Preload("Model").
		Preload("Driver").
		Preload("Asset").
		First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *Service) UpdateVehicle(v *FleetVehicle) error {
	return s.db.Save(v).Error
}

// ----------------- Odometer Logs -----------------

func (s *Service) LogOdometer(log *FleetOdometerLog, companyID, userID uint) error {
	if log.VehicleID == 0 {
		return errors.New("vehicle is required")
	}
	if log.Value <= 0 {
		return errors.New("odometer value must be positive")
	}

	if log.Date == "" {
		log.Date = time.Now().Format("2006-01-02")
	}

	log.CompanyID = companyID
	log.CreateUID = userID
	log.WriteUID = userID

	if err := s.db.Create(log).Error; err != nil {
		return err
	}

	// Update vehicle last odometer if this reading is higher
	var v FleetVehicle
	if err := s.db.First(&v, log.VehicleID).Error; err == nil {
		if log.Value > v.LastOdometer {
			v.LastOdometer = log.Value
			v.WriteUID = userID
			s.db.Save(&v)
		}
	}

	return nil
}

func (s *Service) ListOdometerLogs(vehicleID, companyID uint) ([]FleetOdometerLog, error) {
	var list []FleetOdometerLog
	q := s.db.Where("company_id = ?", companyID)
	if vehicleID > 0 {
		q = q.Where("vehicle_id = ?", vehicleID)
	}

	err := q.Preload("Vehicle").
		Preload("Vehicle.Model").
		Preload("Driver").
		Order("date desc, id desc").
		Find(&list).Error
	return list, err
}

// ----------------- Service Logs -----------------

func (s *Service) CreateServiceLog(srv *FleetServiceLog, companyID, userID uint) error {
	if srv.VehicleID == 0 {
		return errors.New("vehicle is required")
	}
	if srv.ServiceType == "" {
		return errors.New("service type is required")
	}

	if srv.Date == "" {
		srv.Date = time.Now().Format("2006-01-02")
	}
	if srv.Stage == "" {
		srv.Stage = "done"
	}

	year := time.Now().Year()
	if srv.ServiceNumber == "" {
		var count int64
		s.db.Model(&FleetServiceLog{}).Count(&count)
		srv.ServiceNumber = fmt.Sprintf("SRV/FLEET/%d/%05d", year, count+1)
	}

	srv.CompanyID = companyID
	srv.CreateUID = userID
	srv.WriteUID = userID

	if err := s.db.Create(srv).Error; err != nil {
		return err
	}

	// If service odometer reported and higher, update vehicle
	if srv.Odometer > 0 {
		var v FleetVehicle
		if err := s.db.First(&v, srv.VehicleID).Error; err == nil {
			if srv.Odometer > v.LastOdometer {
				v.LastOdometer = srv.Odometer
				v.WriteUID = userID
				s.db.Save(&v)
			}
		}
	}

	return nil
}

func (s *Service) ListServiceLogs(vehicleID, companyID uint, serviceType, stage string) ([]FleetServiceLog, error) {
	var list []FleetServiceLog
	q := s.db.Where("company_id = ?", companyID)
	if vehicleID > 0 {
		q = q.Where("vehicle_id = ?", vehicleID)
	}
	if serviceType != "" && serviceType != "all" {
		q = q.Where("service_type = ?", serviceType)
	}
	if stage != "" && stage != "all" {
		q = q.Where("stage = ?", stage)
	}

	err := q.Preload("Vehicle").
		Preload("Vehicle.Model").
		Preload("Vendor").
		Order("date desc, id desc").
		Find(&list).Error
	return list, err
}

func (s *Service) CompleteServiceLog(id, companyID, userID uint) (*FleetServiceLog, error) {
	var srv FleetServiceLog
	err := s.db.Where("id = ? AND company_id = ?", id, companyID).First(&srv).Error
	if err != nil {
		return nil, errors.New("service log not found")
	}

	srv.Stage = "done"
	srv.WriteUID = userID
	if err := s.db.Save(&srv).Error; err != nil {
		return nil, err
	}

	return &srv, nil
}
