package ai

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
)

var (
	globalEnergyProfileCounter uint64
)

// IndustrialEnergyProfileRecord holds the 24-hour electrical load profile, WBP/LWBP tariff rules, and peak-shaving optimization results
type IndustrialEnergyProfileRecord struct {
	ID                         uint      `gorm:"primaryKey" json:"id"`
	ProfileCode                string    `gorm:"size:64;uniqueIndex;not null" json:"profile_code"`
	FacilityName               string    `gorm:"size:128;not null" json:"facility_name"`
	GridTariffLWBP_IDR         float64   `gorm:"column:grid_tariff_lwbp_idr" json:"grid_tariff_lwbp_idr"` // Rp/kWh during off-peak
	GridTariffWBP_IDR          float64   `gorm:"column:grid_tariff_wbp_idr" json:"grid_tariff_wbp_idr"`   // Rp/kWh during peak hours (18:00 - 22:00)
	PeakWindowStartHour        int       `json:"peak_window_start_hour"`                                   // default: 18
	PeakWindowEndHour          int       `json:"peak_window_end_hour"`                                     // default: 22
	TotalBaselinePowerKW       float64   `gorm:"column:total_baseline_power_kw" json:"total_baseline_power_kw"`
	OptimizedPowerKW           float64   `gorm:"column:optimized_power_kw" json:"optimized_power_kw"`
	PeakReductionKW            float64   `gorm:"column:peak_reduction_kw" json:"peak_reduction_kw"`
	PeakReductionPct           float64   `gorm:"column:peak_reduction_pct" json:"peak_reduction_pct"`
	MonthlyCostSavingsIDR      float64   `gorm:"column:monthly_cost_savings_idr" json:"monthly_cost_savings_idr"`
	MonthlyCarbonAvoidedKgCO2e float64   `gorm:"column:monthly_carbon_avoided_kg_co2e" json:"monthly_carbon_avoided_kg_co2e"`
	HourlyCurveJSON            string    `gorm:"column:hourly_curve_json;type:text" json:"hourly_curve_json"`
	Status                     string    `gorm:"size:32;default:'ACTIVE'" json:"status"` // "ACTIVE", "SIMULATED", "APPLIED"
	CompanyID                  uint      `gorm:"index" json:"company_id"`
	CreatedAt                  time.Time `json:"created_at"`
	UpdatedAt                  time.Time `json:"updated_at"`
}

func (IndustrialEnergyProfileRecord) TableName() string {
	return "ai_industrial_energy_profiles"
}

// MachineLoadScheduleRecord stores individual high-load machine rescheduling recipes
type MachineLoadScheduleRecord struct {
	ID                     uint      `gorm:"primaryKey" json:"id"`
	ProfileID              uint      `gorm:"index;not null" json:"profile_id"`
	MachineCode            string    `gorm:"size:64;index" json:"machine_code"`
	MachineName            string    `gorm:"size:128" json:"machine_name"`
	RatedPowerKW           float64   `gorm:"column:rated_power_kw" json:"rated_power_kw"`
	OriginalStartHour      int       `json:"original_start_hour"`
	OriginalEndHour        int       `json:"original_end_hour"`
	OptimizedStartHour     int       `json:"optimized_start_hour"`
	OptimizedEndHour       int       `json:"optimized_end_hour"`
	ShiftStrategy          string    `gorm:"size:64" json:"shift_strategy"` // "PEAK_SHAVING_SHIFT", "THERMAL_STORAGE_PRECOOL", "BATTERY_BESS_DISCHARGE"
	CostSavingsPerCycleIDR float64   `gorm:"column:cost_savings_per_cycle_idr" json:"cost_savings_per_cycle_idr"`
	Status                 string    `gorm:"size:32;default:'PROPOSED'" json:"status"` // "PROPOSED", "ACCEPTED", "DISMISSED"
	CompanyID              uint      `gorm:"index" json:"company_id"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

func (MachineLoadScheduleRecord) TableName() string {
	return "ai_machine_load_schedules"
}

// ProductCarbonFootprintRecord models real-time Scope 1 & Scope 2 carbon accounting per SKU
type ProductCarbonFootprintRecord struct {
	ID                       uint      `gorm:"primaryKey" json:"id"`
	ProductSKU               string    `gorm:"size:64;index;not null" json:"product_sku"`
	ProductName              string    `gorm:"size:255" json:"product_name"`
	ProductionUnit           string    `gorm:"size:32" json:"production_unit"` // "PCS", "KG", "TON"
	ElectricityKWhPerUnit    float64   `gorm:"column:electricity_kwh_per_unit" json:"electricity_kwh_per_unit"`
	ThermalMJPerUnit         float64   `gorm:"column:thermal_mj_per_unit" json:"thermal_mj_per_unit"`
	Scope1EmissionsKgCO2e    float64   `gorm:"column:scope1_emissions_kg_co2e" json:"scope1_emissions_kg_co2e"`
	Scope2EmissionsKgCO2e    float64   `gorm:"column:scope2_emissions_kg_co2e" json:"scope2_emissions_kg_co2e"`
	TotalCarbonPerUnitKgCO2e float64   `gorm:"column:total_carbon_per_unit_kg_co2e" json:"total_carbon_per_unit_kg_co2e"`
	CBAMTaxExposureEUR       float64   `gorm:"column:cbam_tax_exposure_eur" json:"cbam_tax_exposure_eur"`
	CompanyID                uint      `gorm:"index" json:"company_id"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

func (ProductCarbonFootprintRecord) TableName() string {
	return "ai_product_carbon_footprints"
}

// HourlyEnergyPoint represents one hour in the 24-hour factory power curve
type HourlyEnergyPoint struct {
	Hour             int     `json:"hour"` // 0 - 23
	BaselineKW       float64 `json:"baseline_kw"`
	OptimizedKW      float64 `json:"optimized_kw"`
	IsPeakWBP        bool    `json:"is_peak_wbp"`
	TariffRateIDR    float64 `json:"tariff_rate_idr"`
	CostBaselineIDR  float64 `json:"cost_baseline_idr"`
	CostOptimizedIDR float64 `json:"cost_optimized_idr"`
	HourlySavingsIDR float64 `json:"hourly_savings_idr"`
}

// EnergySentinelOverview compiles executive facility energy, peak reduction, and decarbonization KPIs
type EnergySentinelOverview struct {
	TotalActiveFacilities     int                             `json:"total_active_facilities"`
	TotalMonthlySavingsIDR    float64                         `json:"total_monthly_savings_idr"`
	TotalCarbonAvoidedTonCO2e float64                         `json:"total_carbon_avoided_ton_co2e"`
	AveragePeakReductionPct   float64                         `json:"average_peak_reduction_pct"`
	ActiveSchedulesCount      int                             `json:"active_schedules_count"`
	RecentProfiles            []IndustrialEnergyProfileRecord `json:"recent_profiles"`
	MachineSchedules          []MachineLoadScheduleRecord     `json:"machine_schedules"`
	TopCarbonFootprints       []ProductCarbonFootprintRecord  `json:"top_carbon_footprints"`
}

// RunOptimizationRequest defines parameters for running physics-informed peak shaving
type RunOptimizationRequest struct {
	FacilityName        string  `json:"facility_name"`
	GridTariffLWBP_IDR  float64 `json:"grid_tariff_lwbp_idr"`
	GridTariffWBP_IDR   float64 `json:"grid_tariff_wbp_idr"`
	PeakWindowStart     int     `json:"peak_window_start"` // default: 18
	PeakWindowEnd       int     `json:"peak_window_end"`   // default: 22
	ApplyThermalPrecool bool    `json:"apply_thermal_precool"`
}

// EnergySentinelEngine manages industrial energy profiles, load shifting algorithms, and carbon accounting
type EnergySentinelEngine struct {
	db *gorm.DB
}

var (
	energyEngineInstance *EnergySentinelEngine
	energyEngineOnce     sync.Once
)

// GetEnergySentinelEngine returns the singleton instance
func GetEnergySentinelEngine(db *gorm.DB) *EnergySentinelEngine {
	energyEngineOnce.Do(func() {
		_ = db.AutoMigrate(
			&IndustrialEnergyProfileRecord{},
			&MachineLoadScheduleRecord{},
			&ProductCarbonFootprintRecord{},
		)
		energyEngineInstance = &EnergySentinelEngine{
			db: db,
		}
	})
	return energyEngineInstance
}

// OptimizeFacilityEnergy executes physics-informed peak-shaving and machine load-shifting algorithms
func (e *EnergySentinelEngine) OptimizeFacilityEnergy(companyID uint, req RunOptimizationRequest) (*IndustrialEnergyProfileRecord, []MachineLoadScheduleRecord, error) {
	_ = e.db.AutoMigrate(
		&IndustrialEnergyProfileRecord{},
		&MachineLoadScheduleRecord{},
		&ProductCarbonFootprintRecord{},
	)

	now := time.Now()
	seq := atomic.AddUint64(&globalEnergyProfileCounter, 1)
	profileCode := fmt.Sprintf("NRG-OPT-%s-%04d-%04d", now.Format("060102150405"), seq, now.UnixNano()%10000)

	if req.FacilityName == "" {
		req.FacilityName = "Pabrik Utama Cikarang Plant 1"
	}
	if req.GridTariffLWBP_IDR <= 0 {
		req.GridTariffLWBP_IDR = 1114.74 // PLN Industri Golongan I-3/TM (LWBP)
	}
	if req.GridTariffWBP_IDR <= 0 {
		req.GridTariffWBP_IDR = 2229.48 // PLN Industri Golongan I-3/TM (WBP, ~2.0x factor)
	}
	if req.PeakWindowStart <= 0 {
		req.PeakWindowStart = 18
	}
	if req.PeakWindowEnd <= 0 {
		req.PeakWindowEnd = 22
	}

	// 1. Generate 24-hour Baseline Power Demand Curve
	// Typically: Base load ~450 kW, Morning shift ~950 kW, Second shift / Evening peak ~1450 kW
	hourlyPoints := make([]HourlyEnergyPoint, 24)
	totalBaselineKW := 0.0
	totalOptimizedKW := 0.0
	dailyCostBaseline := 0.0
	dailyCostOptimized := 0.0

	// Machines candidate for load-shifting:
	// M1: Induction Melting Furnace (350 kW, original 19:00 - 22:00 -> shift to 22:00 - 01:00 LWBP)
	// M2: Ball Mill Pulverizer (180 kW, original 18:00 - 21:00 -> shift to 13:00 - 16:00 LWBP)
	// M3: Central Chiller Bank (220 kW, precool at 15:00 - 17:00, cut 110 kW during WBP)
	// M4: Wastewater Aerator (75 kW, original 20:00 - 23:00 -> shift to 02:00 - 05:00 LWBP)

	shiftedKWDuringWBP := 350.0 + 180.0 + 75.0 // 605 kW reducible in peak hours
	if req.ApplyThermalPrecool {
		shiftedKWDuringWBP += 110.0 // 715 kW total peak cut
	}

	for h := 0; h < 24; h++ {
		isPeak := h >= req.PeakWindowStart && h < req.PeakWindowEnd
		tariff := req.GridTariffLWBP_IDR
		if isPeak {
			tariff = req.GridTariffWBP_IDR
		}

		// Physics synthetic demand curve
		var baseKW float64
		if h >= 0 && h < 6 {
			baseKW = 450.0 + math.Sin(float64(h))*30.0 // night shift baseline
		} else if h >= 6 && h < 12 {
			baseKW = 950.0 + float64(h)*25.0 // morning ramp-up
		} else if h >= 12 && h < 18 {
			baseKW = 1100.0 + math.Cos(float64(h))*40.0 // afternoon production
		} else if isPeak {
			baseKW = 1480.0 + math.Sin(float64(h))*50.0 // peak second shift
		} else {
			baseKW = 820.0 - float64(h-22)*60.0 // night ramp-down
		}

		optKW := baseKW
		if isPeak {
			// Shave peak load by shifting non-critical loads
			optKW = baseKW - shiftedKWDuringWBP*0.65
			if optKW < 750.0 {
				optKW = 750.0
			}
		} else if h >= 22 || (h >= 13 && h <= 16) {
			// Absorb shifted load during off-peak LWBP
			optKW = baseKW + (shiftedKWDuringWBP * 0.25)
		}

		baseKW = math.Round(baseKW*10) / 10
		optKW = math.Round(optKW*10) / 10

		costBase := baseKW * tariff
		costOpt := optKW * tariff
		savings := costBase - costOpt
		if savings < 0 {
			savings = 0
		}

		hourlyPoints[h] = HourlyEnergyPoint{
			Hour:             h,
			BaselineKW:       baseKW,
			OptimizedKW:      optKW,
			IsPeakWBP:        isPeak,
			TariffRateIDR:    tariff,
			CostBaselineIDR:  math.Round(costBase),
			CostOptimizedIDR: math.Round(costOpt),
			HourlySavingsIDR: math.Round(savings),
		}

		totalBaselineKW += baseKW
		totalOptimizedKW += optKW
		dailyCostBaseline += costBase
		dailyCostOptimized += costOpt
	}

	peakBaselineKW := 0.0
	peakOptimizedKW := 0.0
	for h := req.PeakWindowStart; h < req.PeakWindowEnd; h++ {
		if hourlyPoints[h].BaselineKW > peakBaselineKW {
			peakBaselineKW = hourlyPoints[h].BaselineKW
		}
		if hourlyPoints[h].OptimizedKW > peakOptimizedKW {
			peakOptimizedKW = hourlyPoints[h].OptimizedKW
		}
	}

	peakReductionKW := peakBaselineKW - peakOptimizedKW
	peakReductionPct := 0.0
	if peakBaselineKW > 0 {
		peakReductionPct = (peakReductionKW / peakBaselineKW) * 100.0
		peakReductionPct = math.Round(peakReductionPct*10) / 10
	}

	// 26 industrial operating days per month
	monthlySavingsIDR := (dailyCostBaseline - dailyCostOptimized) * 26.0
	monthlySavingsIDR = math.Round(monthlySavingsIDR)

	// Avoided Carbon from peak inefficient peaker plant dispatch: ~0.794 kg CO2e per kWh saved during peak
	dailyKWhShaved := peakReductionKW * float64(req.PeakWindowEnd-req.PeakWindowStart) * 0.7
	monthlyCarbonKgCO2e := dailyKWhShaved * 0.794 * 26.0
	monthlyCarbonKgCO2e = math.Round(monthlyCarbonKgCO2e*10) / 10

	curveJSONBytes, _ := json.Marshal(hourlyPoints)

	profile := IndustrialEnergyProfileRecord{
		ProfileCode:                profileCode,
		FacilityName:               req.FacilityName,
		GridTariffLWBP_IDR:         req.GridTariffLWBP_IDR,
		GridTariffWBP_IDR:          req.GridTariffWBP_IDR,
		PeakWindowStartHour:        req.PeakWindowStart,
		PeakWindowEndHour:          req.PeakWindowEnd,
		TotalBaselinePowerKW:       math.Round(totalBaselineKW),
		OptimizedPowerKW:           math.Round(totalOptimizedKW),
		PeakReductionKW:            math.Round(peakReductionKW*10) / 10,
		PeakReductionPct:           peakReductionPct,
		MonthlyCostSavingsIDR:      monthlySavingsIDR,
		MonthlyCarbonAvoidedKgCO2e: monthlyCarbonKgCO2e,
		HourlyCurveJSON:            string(curveJSONBytes),
		Status:                     "ACTIVE",
		CompanyID:                  companyID,
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}

	if err := e.db.Create(&profile).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to save energy profile: %w", err)
	}

	// 2. Build Machine Load Schedule Recommendations
	schedules := []MachineLoadScheduleRecord{
		{
			ProfileID:              profile.ID,
			MachineCode:            "MCH-INDUCTION-FURNACE-01",
			MachineName:            "Tanur Induksi Peleburan Baja 350kW",
			RatedPowerKW:           350.0,
			OriginalStartHour:      19,
			OriginalEndHour:        22,
			OptimizedStartHour:     22,
			OptimizedEndHour:       1,
			ShiftStrategy:          "PEAK_SHAVING_SHIFT",
			CostSavingsPerCycleIDR: 350.0 * 3.0 * (req.GridTariffWBP_IDR - req.GridTariffLWBP_IDR),
			Status:                 "PROPOSED",
			CompanyID:              companyID,
			CreatedAt:              now,
			UpdatedAt:              now,
		},
		{
			ProfileID:              profile.ID,
			MachineCode:            "MCH-BALL-MILL-03",
			MachineName:            "Mesin Penggiling Ball Mill Granit 180kW",
			RatedPowerKW:           180.0,
			OriginalStartHour:      18,
			OriginalEndHour:        21,
			OptimizedStartHour:     13,
			OptimizedEndHour:       16,
			ShiftStrategy:          "PEAK_SHAVING_SHIFT",
			CostSavingsPerCycleIDR: 180.0 * 3.0 * (req.GridTariffWBP_IDR - req.GridTariffLWBP_IDR),
			Status:                 "PROPOSED",
			CompanyID:              companyID,
			CreatedAt:              now,
			UpdatedAt:              now,
		},
		{
			ProfileID:              profile.ID,
			MachineCode:            "MCH-CHILLER-CENTRAL-02",
			MachineName:            "Chiller Sentral HVAC & Precooling 220kW",
			RatedPowerKW:           220.0,
			OriginalStartHour:      18,
			OriginalEndHour:        22,
			OptimizedStartHour:     15,
			OptimizedEndHour:       18,
			ShiftStrategy:          "THERMAL_STORAGE_PRECOOL",
			CostSavingsPerCycleIDR: 110.0 * 4.0 * (req.GridTariffWBP_IDR - req.GridTariffLWBP_IDR),
			Status:                 "PROPOSED",
			CompanyID:              companyID,
			CreatedAt:              now,
			UpdatedAt:              now,
		},
		{
			ProfileID:              profile.ID,
			MachineCode:            "MCH-AERATOR-WWTP-01",
			MachineName:            "Blower Aerasi Pengolahan Air Limbah 75kW",
			RatedPowerKW:           75.0,
			OriginalStartHour:      20,
			OriginalEndHour:        23,
			OptimizedStartHour:     2,
			OptimizedEndHour:       5,
			ShiftStrategy:          "PEAK_SHAVING_SHIFT",
			CostSavingsPerCycleIDR: 75.0 * 2.0 * (req.GridTariffWBP_IDR - req.GridTariffLWBP_IDR),
			Status:                 "PROPOSED",
			CompanyID:              companyID,
			CreatedAt:              now,
			UpdatedAt:              now,
		},
	}

	for i := range schedules {
		schedules[i].CostSavingsPerCycleIDR = math.Round(schedules[i].CostSavingsPerCycleIDR)
		_ = e.db.Create(&schedules[i]).Error
	}

	return &profile, schedules, nil
}

// CalculateProductCarbonFootprint computes Scope 1 & 2 GHG footprint and CBAM tax exposure per SKU
func (e *EnergySentinelEngine) CalculateProductCarbonFootprint(companyID uint, sku, name, unit string, kwhPerUnit, fuelMJ float64) (*ProductCarbonFootprintRecord, error) {
	_ = e.db.AutoMigrate(&ProductCarbonFootprintRecord{})

	if sku == "" {
		sku = "SKU-PROD-GENERIC-01"
	}
	if name == "" {
		name = "Komponen Presisi Manufaktur"
	}
	if unit == "" {
		unit = "PCS"
	}

	// Emission Factors:
	// Scope 1: Fuel combustion (natural gas/diesel) ~ 0.0561 kg CO2e per MJ
	// Scope 2: Indonesian Jamali Grid Electricity Factor ~ 0.794 kg CO2e per kWh
	scope1 := fuelMJ * 0.0561
	scope2 := kwhPerUnit * 0.794
	totalCarbon := scope1 + scope2

	// CBAM Tariff Exposure: ~85 EUR per Ton CO2e
	cbamTaxEUR := (totalCarbon / 1000.0) * 85.0

	record := ProductCarbonFootprintRecord{
		ProductSKU:               sku,
		ProductName:              name,
		ProductionUnit:           unit,
		ElectricityKWhPerUnit:    math.Round(kwhPerUnit*100) / 100,
		ThermalMJPerUnit:         math.Round(fuelMJ*100) / 100,
		Scope1EmissionsKgCO2e:    math.Round(scope1*1000) / 1000,
		Scope2EmissionsKgCO2e:    math.Round(scope2*1000) / 1000,
		TotalCarbonPerUnitKgCO2e: math.Round(totalCarbon*1000) / 1000,
		CBAMTaxExposureEUR:       math.Round(cbamTaxEUR*100) / 100,
		CompanyID:                companyID,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	if err := e.db.Create(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// GetOverview aggregates executive metrics for industrial energy optimization
func (e *EnergySentinelEngine) GetOverview(companyID uint) (*EnergySentinelOverview, error) {
	_ = e.db.AutoMigrate(
		&IndustrialEnergyProfileRecord{},
		&MachineLoadScheduleRecord{},
		&ProductCarbonFootprintRecord{},
	)

	var facilityCount int64
	var scheduleCount int64

	_ = e.db.Model(&IndustrialEnergyProfileRecord{}).Where("company_id = ?", companyID).Count(&facilityCount).Error
	_ = e.db.Model(&MachineLoadScheduleRecord{}).Where("company_id = ?", companyID).Count(&scheduleCount).Error

	type sumResult struct {
		TotalSavings float64 `gorm:"column:total_savings"`
		TotalCarbon  float64 `gorm:"column:total_carbon"`
		AvgReduction float64 `gorm:"column:avg_reduction"`
	}
	var sr sumResult
	_ = e.db.Table("ai_industrial_energy_profiles").
		Select("COALESCE(SUM(monthly_cost_savings_idr), 0) as total_savings, COALESCE(SUM(monthly_carbon_avoided_kg_co2e), 0) as total_carbon, COALESCE(AVG(peak_reduction_pct), 0) as avg_reduction").
		Where("company_id = ?", companyID).
		Scan(&sr).Error

	var recentProfiles []IndustrialEnergyProfileRecord
	_ = e.db.Where("company_id = ?", companyID).Order("id DESC").Limit(10).Find(&recentProfiles).Error

	var machineSchedules []MachineLoadScheduleRecord
	_ = e.db.Where("company_id = ?", companyID).Order("id DESC").Limit(15).Find(&machineSchedules).Error

	var topFootprints []ProductCarbonFootprintRecord
	_ = e.db.Where("company_id = ?", companyID).Order("total_carbon_per_unit_kg_co2e DESC").Limit(10).Find(&topFootprints).Error

	overview := &EnergySentinelOverview{
		TotalActiveFacilities:     int(facilityCount),
		TotalMonthlySavingsIDR:    sr.TotalSavings,
		TotalCarbonAvoidedTonCO2e: math.Round((sr.TotalCarbon/1000.0)*10) / 10,
		AveragePeakReductionPct:   math.Round(sr.AvgReduction*10) / 10,
		ActiveSchedulesCount:      int(scheduleCount),
		RecentProfiles:            recentProfiles,
		MachineSchedules:          machineSchedules,
		TopCarbonFootprints:       topFootprints,
	}

	return overview, nil
}

// GetProfiles returns all facility profiles
func (e *EnergySentinelEngine) GetProfiles(companyID uint) ([]IndustrialEnergyProfileRecord, error) {
	var profiles []IndustrialEnergyProfileRecord
	if err := e.db.Where("company_id = ?", companyID).Order("id DESC").Find(&profiles).Error; err != nil {
		return nil, err
	}
	return profiles, nil
}

// GetProfileByID returns specific facility profile with associated machine schedules
func (e *EnergySentinelEngine) GetProfileByID(companyID uint, profileID uint) (*IndustrialEnergyProfileRecord, []MachineLoadScheduleRecord, error) {
	var profile IndustrialEnergyProfileRecord
	if err := e.db.Where("id = ? AND company_id = ?", profileID, companyID).First(&profile).Error; err != nil {
		return nil, nil, err
	}

	var schedules []MachineLoadScheduleRecord
	_ = e.db.Where("profile_id = ? AND company_id = ?", profileID, companyID).Order("id ASC").Find(&schedules).Error

	return &profile, schedules, nil
}

// GetProductCarbonFootprints returns all product carbon footprints
func (e *EnergySentinelEngine) GetProductCarbonFootprints(companyID uint) ([]ProductCarbonFootprintRecord, error) {
	var records []ProductCarbonFootprintRecord
	if err := e.db.Where("company_id = ?", companyID).Order("total_carbon_per_unit_kg_co2e DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

// UpdateScheduleStatus updates whether a machine load shift was applied or dismissed
func (e *EnergySentinelEngine) UpdateScheduleStatus(companyID uint, scheduleID uint, status string) (*MachineLoadScheduleRecord, error) {
	var schedule MachineLoadScheduleRecord
	if err := e.db.Where("id = ? AND company_id = ?", scheduleID, companyID).First(&schedule).Error; err != nil {
		return nil, err
	}

	schedule.Status = status
	schedule.UpdatedAt = time.Now()
	if err := e.db.Save(&schedule).Error; err != nil {
		return nil, err
	}

	return &schedule, nil
}

// SeedDemoData seeds realistic manufacturing plant profiles, machine schedules, and product carbon footprints
func (e *EnergySentinelEngine) SeedDemoData(companyID uint) error {
	_ = e.db.AutoMigrate(
		&IndustrialEnergyProfileRecord{},
		&MachineLoadScheduleRecord{},
		&ProductCarbonFootprintRecord{},
	)

	// 1. Metal Stamping Plant Cikarang
	_, _, _ = e.OptimizeFacilityEnergy(companyID, RunOptimizationRequest{
		FacilityName:        "Pabrik Utama Cikarang Plant 1 (Metal Stamping & Casting)",
		GridTariffLWBP_IDR:  1114.74,
		GridTariffWBP_IDR:   2229.48,
		PeakWindowStart:     18,
		PeakWindowEnd:       22,
		ApplyThermalPrecool: true,
	})

	// 2. Cold Storage Hub Surabaya
	_, _, _ = e.OptimizeFacilityEnergy(companyID, RunOptimizationRequest{
		FacilityName:        "Cold Storage & Freezing Hub Regional Surabaya",
		GridTariffLWBP_IDR:  1114.74,
		GridTariffWBP_IDR:   2229.48,
		PeakWindowStart:     18,
		PeakWindowEnd:       22,
		ApplyThermalPrecool: true,
	})

	// 3. Product Carbon Footprints
	_, _ = e.CalculateProductCarbonFootprint(companyID, "SKU-STEEL-BRKT-01", "Braket Baja Presisi Otomotif", "PCS", 4.25, 12.5)
	_, _ = e.CalculateProductCarbonFootprint(companyID, "SKU-PLAST-HDPE-25L", "Jerigen Polimer HDPE 25 Liter", "PCS", 0.65, 2.1)
	_, _ = e.CalculateProductCarbonFootprint(companyID, "SKU-FOOD-TUNA-LOIN", "Frozen Yellowfin Tuna Loin Super", "KG", 2.10, 0.0)

	return nil
}
