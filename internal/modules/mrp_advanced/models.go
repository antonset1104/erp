package mrp_advanced

import (
	"time"

	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/product"
)

// WorkCenter represents Odoo's mrp.workcenter (Pusat Kerja / Stasiun Mesin Produksi)
type WorkCenter struct {
	models.ModelBase
	Code               string  `gorm:"size:32;not null;uniqueIndex" json:"code"`       // e.g. WC/2026/001
	Name               string  `gorm:"size:128;not null" json:"name"`                  // e.g. "Mesin CNC Milling 5-Axis"
	WorkCenterType     string  `gorm:"size:64;default:'machining';index" json:"work_center_type"` // machining, assembly, welding, packaging, qc
	CapacityPerHour    float64 `gorm:"default:10.0" json:"capacity_per_hour"`          // Output kapasitas unit per jam
	WorkingHoursPerDay float64 `gorm:"default:8.0" json:"working_hours_per_day"`       // Jam kerja efektif per hari (shift standar)
	TotalOEEPercentage float64 `gorm:"default:85.0" json:"total_oee_percentage"`      // Overall Equipment Effectiveness (%)
	CostPerHour        float64 `gorm:"default:150000.0" json:"cost_per_hour"`         // Biaya operasional mesin + teknisi per jam (IDR)
	MachinePowerKW     float64 `gorm:"default:15.0" json:"machine_power_kw"`          // Konsumsi daya listrik kW
	Status             string  `gorm:"size:32;default:'available';index" json:"status"` // available, busy, maintenance, breakdown
	Notes              string  `gorm:"type:text" json:"notes,omitempty"`
}

// Routing represents Odoo's mrp.routing (Rute Tahapan Proses Produksi)
type Routing struct {
	models.ModelBase
	RoutingCode string             `gorm:"size:64;not null;uniqueIndex" json:"routing_code"` // e.g. ROUT/2026/00001
	Name        string             `gorm:"size:128;not null" json:"name"`                   // e.g. "Rute Perakitan Chassis & Dinamo"
	ProductID   uint               `gorm:"index;not null" json:"product_id"`
	Product     product.Product    `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	BomID       uint               `gorm:"index" json:"bom_id,omitempty"`
	Operations  []RoutingOperation `gorm:"foreignKey:RoutingID" json:"operations,omitempty"`
	Notes       string             `gorm:"type:text" json:"notes,omitempty"`
}

// RoutingOperation represents Odoo's mrp.routing.workcenter (Langkah Operasi dalam Rute)
type RoutingOperation struct {
	models.ModelBase
	RoutingID               uint       `gorm:"index;not null" json:"routing_id"`
	Sequence                int        `gorm:"default:10;index" json:"sequence"` // 10, 20, 30...
	Name                    string     `gorm:"size:128;not null" json:"name"`   // e.g. "Pemotongan & Bubut Presisi"
	WorkCenterID            uint       `gorm:"index;not null" json:"work_center_id"`
	WorkCenter              WorkCenter `gorm:"foreignKey:WorkCenterID" json:"work_center,omitempty"`
	SetupTimeMinutes        float64    `gorm:"default:15.0" json:"setup_time_minutes"`       // Waktu persiapan mesin/mold
	CycleTimePerUnitMinutes float64    `gorm:"default:5.0" json:"cycle_time_per_unit_minutes"` // Durasi proses per 1 unit barang
	TeardownTimeMinutes     float64    `gorm:"default:10.0" json:"teardown_time_minutes"`    // Waktu pembersihan & pendinginan
	BatchSize               float64    `gorm:"default:50.0" json:"batch_size"`               // Ukuran batch rekomendasi
	Description             string     `gorm:"type:text" json:"description,omitempty"`
}

// WorkOrder represents Odoo's mrp.workorder (Perintah Kerja Operasi Aktual per SPK)
type WorkOrder struct {
	models.ModelBase
	WorkOrderCode          string           `gorm:"size:64;not null;uniqueIndex" json:"work_order_code"` // e.g. WO/2026/00001
	ProductionOrderID      uint             `gorm:"index;not null" json:"production_order_id"`          // Link ke mrp.production (MO)
	ProductionOrderName    string           `gorm:"size:64;index" json:"production_order_name"`         // e.g. MO/2026/00001
	RoutingOperationID     uint             `gorm:"index" json:"routing_operation_id,omitempty"`
	WorkCenterID           uint             `gorm:"index;not null" json:"work_center_id"`
	WorkCenter             WorkCenter       `gorm:"foreignKey:WorkCenterID" json:"work_center,omitempty"`
	Sequence               int              `gorm:"default:10" json:"sequence"`
	OperationName          string           `gorm:"size:128;not null" json:"operation_name"`
	PlannedStartTime       *time.Time       `json:"planned_start_time,omitempty"`
	PlannedEndTime         *time.Time       `json:"planned_end_time,omitempty"`
	ActualStartTime        *time.Time       `json:"actual_start_time,omitempty"`
	ActualEndTime          *time.Time       `json:"actual_end_time,omitempty"`
	PlannedDurationMinutes float64          `gorm:"default:0" json:"planned_duration_minutes"`
	ActualDurationMinutes  float64          `gorm:"default:0" json:"actual_duration_minutes"`
	QtyPlanned             float64          `gorm:"default:1.0" json:"qty_planned"`
	QtyProduced            float64          `gorm:"default:0.0" json:"qty_produced"`
	QtyScrap               float64          `gorm:"default:0.0" json:"qty_scrap"`
	MachineHours           float64          `gorm:"default:0.0" json:"machine_hours"`      // Jam mesin terpakai
	TotalMachineCost       float64          `gorm:"default:0.0" json:"total_machine_cost"` // IDR machine cost
	State                  string           `gorm:"size:32;default:'pending';index" json:"state"` // pending, ready, in_progress, done, paused, cancelled
	Notes                  string           `gorm:"type:text" json:"notes,omitempty"`
}

// DTOs for API requests & responses
type CreateWorkCenterRequest struct {
	Name               string  `json:"name" binding:"required"`
	WorkCenterType     string  `json:"work_center_type"`
	CapacityPerHour    float64 `json:"capacity_per_hour"`
	WorkingHoursPerDay float64 `json:"working_hours_per_day"`
	TotalOEEPercentage float64 `json:"total_oee_percentage"`
	CostPerHour        float64 `json:"cost_per_hour"`
	MachinePowerKW     float64 `json:"machine_power_kw"`
	Notes              string  `json:"notes"`
}

type CreateRoutingOperationDTO struct {
	Sequence                int     `json:"sequence"`
	Name                    string  `json:"name" binding:"required"`
	WorkCenterID            uint    `json:"work_center_id" binding:"required"`
	SetupTimeMinutes        float64 `json:"setup_time_minutes"`
	CycleTimePerUnitMinutes float64 `json:"cycle_time_per_unit_minutes"`
	TeardownTimeMinutes     float64 `json:"teardown_time_minutes"`
	BatchSize               float64 `json:"batch_size"`
	Description             string  `json:"description"`
}

type CreateRoutingRequest struct {
	Name        string                      `json:"name" binding:"required"`
	ProductID   uint                        `json:"product_id" binding:"required"`
	BomID       uint                        `json:"bom_id"`
	Operations  []CreateRoutingOperationDTO `json:"operations"`
	Notes       string                      `json:"notes"`
}

type GenerateWorkOrdersRequest struct {
	ProductionOrderID uint `json:"production_order_id" binding:"required"`
	RoutingID         uint `json:"routing_id" binding:"required"`
}

type CompleteWorkOrderRequest struct {
	QtyProduced            float64 `json:"qty_produced"`
	QtyScrap               float64 `json:"qty_scrap"`
	ActualDurationMinutes  float64 `json:"actual_duration_minutes"`
	Notes                  string  `json:"notes"`
}

// GanttCapacitySlot represents a visualized task in the Gantt Chart capacity timeline
type GanttCapacitySlot struct {
	WorkCenterID    uint      `json:"work_center_id"`
	WorkCenterName  string    `json:"work_center_name"`
	WorkCenterCode  string    `json:"work_center_code"`
	WorkOrderID     uint      `json:"work_order_id"`
	WorkOrderCode   string    `json:"work_order_code"`
	OperationName   string    `json:"operation_name"`
	MOName          string    `json:"mo_name"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	DurationMinutes float64   `json:"duration_minutes"`
	State           string    `json:"state"`
	ColorStatus     string    `json:"color_status"`
}

type WorkCenterCapacitySummary struct {
	WorkCenterID          uint    `json:"work_center_id"`
	WorkCenterName        string  `json:"work_center_name"`
	CapacityHoursPerDay   float64 `json:"capacity_hours_per_day"`
	ScheduledHoursTotal   float64 `json:"scheduled_hours_total"`
	UtilizationPercentage float64 `json:"utilization_percentage"`
	IsBottleneck          bool    `json:"is_bottleneck"`
	ActiveWorkOrdersCount int     `json:"active_work_orders_count"`
}

type GanttCapacityResponse struct {
	Slots             []GanttCapacitySlot          `json:"slots"`
	CapacitySummaries []WorkCenterCapacitySummary `json:"capacity_summaries"`
	HorizonDays       int                          `json:"horizon_days"`
	StartDate         string                       `json:"start_date"`
	EndDate           string                       `json:"end_date"`
}

type MRPAdvancedStats struct {
	TotalWorkCenters       int64   `json:"total_work_centers"`
	ActiveWorkCenters      int64   `json:"active_work_centers"`
	OverallAvgOEE          float64 `json:"overall_avg_oee"`
	TotalRoutings          int64   `json:"total_routings"`
	TotalWorkOrders        int64   `json:"total_work_orders"`
	PendingWorkOrders      int64   `json:"pending_work_orders"`
	InProgressWorkOrders   int64   `json:"in_progress_work_orders"`
	CompletedWorkOrders    int64   `json:"completed_work_orders"`
	TotalMachineHours      float64 `json:"total_machine_hours"`
	TotalMachineCost       float64 `json:"total_machine_cost"`
	CriticalBottlenecksNum int     `json:"critical_bottlenecks_num"`
}

// AIMRPCapacityReport represents Cloudflare Workers AI + Local Heuristic Analysis
type AIMRPCapacityReport struct {
	ScheduleEfficiencyScore int      `json:"schedule_efficiency_score"` // 0-100
	OverallStatus            string   `json:"overall_status"`            // BALANCED, BOTTLENECK_DETECTED, OVERLOADED
	BottleneckWorkCenter     string   `json:"bottleneck_work_center"`
	BottleneckUtilization    float64  `json:"bottleneck_utilization"`
	RecommendedLeadTimeDays  int      `json:"recommended_lead_time_days"`
	ExecutiveSummary         string   `json:"executive_summary"`
	LineBalancingAdvice      []string `json:"line_balancing_advice"`
	CriticalPathOperations   []string `json:"critical_path_operations"`
	AIProvider               string   `json:"ai_provider"`
}
