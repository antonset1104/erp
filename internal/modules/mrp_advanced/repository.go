package mrp_advanced

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	// Sequences
	GetNextWorkCenterCode(tx *gorm.DB) (string, error)
	GetNextRoutingCode(tx *gorm.DB) (string, error)
	GetNextWorkOrderCode(tx *gorm.DB) (string, error)

	// Work Centers
	CreateWorkCenter(wc *WorkCenter) error
	UpdateWorkCenter(wc *WorkCenter) error
	GetWorkCenterByID(id uint) (*WorkCenter, error)
	ListWorkCenters(companyID uint) ([]WorkCenter, error)

	// Routings & Operations
	CreateRouting(r *Routing) error
	GetRoutingByID(id uint) (*Routing, error)
	GetRoutingByProduct(productID uint, companyID uint) (*Routing, error)
	ListRoutings(companyID uint) ([]Routing, error)

	// Work Orders
	CreateWorkOrder(wo *WorkOrder) error
	UpdateWorkOrder(wo *WorkOrder) error
	GetWorkOrderByID(id uint) (*WorkOrder, error)
	ListWorkOrders(companyID uint, state string) ([]WorkOrder, error)
	ListWorkOrdersByMO(moID uint) ([]WorkOrder, error)
	ListWorkOrdersForGantt(companyID uint, start time.Time, end time.Time) ([]WorkOrder, error)

	// Analytics
	GetStats(companyID uint) (*MRPAdvancedStats, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// ----------------------------------------------------------------------------
// SEQUENCES
// ----------------------------------------------------------------------------

func (r *repository) GetNextWorkCenterCode(tx *gorm.DB) (string, error) {
	if tx == nil {
		tx = r.db
	}
	year := time.Now().Year()
	var count int64
	if err := tx.Model(&WorkCenter{}).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("WC/%d/%03d", year, count+1), nil
}

func (r *repository) GetNextRoutingCode(tx *gorm.DB) (string, error) {
	if tx == nil {
		tx = r.db
	}
	year := time.Now().Year()
	var count int64
	if err := tx.Model(&Routing{}).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("ROUT/%d/%05d", year, count+1), nil
}

func (r *repository) GetNextWorkOrderCode(tx *gorm.DB) (string, error) {
	if tx == nil {
		tx = r.db
	}
	year := time.Now().Year()
	var count int64
	if err := tx.Model(&WorkOrder{}).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("WO/%d/%05d", year, count+1), nil
}

// ----------------------------------------------------------------------------
// WORK CENTERS
// ----------------------------------------------------------------------------

func (r *repository) CreateWorkCenter(wc *WorkCenter) error {
	return r.db.Create(wc).Error
}

func (r *repository) UpdateWorkCenter(wc *WorkCenter) error {
	return r.db.Save(wc).Error
}

func (r *repository) GetWorkCenterByID(id uint) (*WorkCenter, error) {
	var wc WorkCenter
	if err := r.db.First(&wc, id).Error; err != nil {
		return nil, err
	}
	return &wc, nil
}

func (r *repository) ListWorkCenters(companyID uint) ([]WorkCenter, error) {
	var list []WorkCenter
	query := r.db.Model(&WorkCenter{})
	if companyID > 0 {
		query = query.Where("company_id = ? OR company_id = 0", companyID)
	}
	if err := query.Order("code asc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ----------------------------------------------------------------------------
// ROUTINGS & OPERATIONS
// ----------------------------------------------------------------------------

func (r *repository) CreateRouting(routing *Routing) error {
	return r.db.Create(routing).Error
}

func (r *repository) GetRoutingByID(id uint) (*Routing, error) {
	var routing Routing
	err := r.db.Preload("Product").
		Preload("Operations", func(db *gorm.DB) *gorm.DB {
			return db.Order("sequence asc")
		}).
		Preload("Operations.WorkCenter").
		First(&routing, id).Error
	if err != nil {
		return nil, err
	}
	return &routing, nil
}

func (r *repository) GetRoutingByProduct(productID uint, companyID uint) (*Routing, error) {
	var routing Routing
	query := r.db.Preload("Operations", func(db *gorm.DB) *gorm.DB {
		return db.Order("sequence asc")
	}).Preload("Operations.WorkCenter").
		Where("product_id = ?", productID)
	if companyID > 0 {
		query = query.Where("company_id = ? OR company_id = 0", companyID)
	}
	if err := query.First(&routing).Error; err != nil {
		return nil, err
	}
	return &routing, nil
}

func (r *repository) ListRoutings(companyID uint) ([]Routing, error) {
	var list []Routing
	query := r.db.Preload("Product").
		Preload("Operations").
		Order("id desc")
	if companyID > 0 {
		query = query.Where("company_id = ? OR company_id = 0", companyID)
	}
	if err := query.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ----------------------------------------------------------------------------
// WORK ORDERS
// ----------------------------------------------------------------------------

func (r *repository) CreateWorkOrder(wo *WorkOrder) error {
	return r.db.Create(wo).Error
}

func (r *repository) UpdateWorkOrder(wo *WorkOrder) error {
	return r.db.Save(wo).Error
}

func (r *repository) GetWorkOrderByID(id uint) (*WorkOrder, error) {
	var wo WorkOrder
	if err := r.db.Preload("WorkCenter").First(&wo, id).Error; err != nil {
		return nil, err
	}
	return &wo, nil
}

func (r *repository) ListWorkOrders(companyID uint, state string) ([]WorkOrder, error) {
	var list []WorkOrder
	query := r.db.Preload("WorkCenter").Order("id desc")
	if companyID > 0 {
		query = query.Where("company_id = ? OR company_id = 0", companyID)
	}
	if state != "" && state != "all" {
		query = query.Where("state = ?", state)
	}
	if err := query.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *repository) ListWorkOrdersByMO(moID uint) ([]WorkOrder, error) {
	var list []WorkOrder
	err := r.db.Preload("WorkCenter").
		Where("production_order_id = ?", moID).
		Order("sequence asc").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *repository) ListWorkOrdersForGantt(companyID uint, start time.Time, end time.Time) ([]WorkOrder, error) {
	var list []WorkOrder
	query := r.db.Preload("WorkCenter").
		Where("state != 'cancelled'")
	if companyID > 0 {
		query = query.Where("company_id = ? OR company_id = 0", companyID)
	}
	if !start.IsZero() && !end.IsZero() {
		query = query.Where("(planned_start_time <= ? AND planned_end_time >= ?) OR (actual_start_time IS NOT NULL AND actual_start_time <= ?)", end, start, end)
	}
	if err := query.Order("planned_start_time asc, sequence asc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ----------------------------------------------------------------------------
// ANALYTICS & STATS
// ----------------------------------------------------------------------------

func (r *repository) GetStats(companyID uint) (*MRPAdvancedStats, error) {
	var stats MRPAdvancedStats

	wcQuery := r.db.Model(&WorkCenter{})
	woQuery := r.db.Model(&WorkOrder{})
	rtQuery := r.db.Model(&Routing{})

	if companyID > 0 {
		wcQuery = wcQuery.Where("company_id = ? OR company_id = 0", companyID)
		woQuery = woQuery.Where("company_id = ? OR company_id = 0", companyID)
		rtQuery = rtQuery.Where("company_id = ? OR company_id = 0", companyID)
	}

	wcQuery.Count(&stats.TotalWorkCenters)
	r.db.Model(&WorkCenter{}).Where("status = 'available' OR status = 'busy'").Count(&stats.ActiveWorkCenters)
	rtQuery.Count(&stats.TotalRoutings)

	woQuery.Count(&stats.TotalWorkOrders)
	r.db.Model(&WorkOrder{}).Where("state = 'pending' OR state = 'ready'").Count(&stats.PendingWorkOrders)
	r.db.Model(&WorkOrder{}).Where("state = 'in_progress'").Count(&stats.InProgressWorkOrders)
	r.db.Model(&WorkOrder{}).Where("state = 'done'").Count(&stats.CompletedWorkOrders)

	// OEE and Machine Hours
	var oeeSum float64
	var wcCount int64
	r.db.Model(&WorkCenter{}).Select("COALESCE(SUM(total_oee_percentage), 0), COUNT(id)").Row().Scan(&oeeSum, &wcCount)
	if wcCount > 0 {
		stats.OverallAvgOEE = oeeSum / float64(wcCount)
	}

	r.db.Model(&WorkOrder{}).Select("COALESCE(SUM(machine_hours), 0), COALESCE(SUM(total_machine_cost), 0)").Row().Scan(&stats.TotalMachineHours, &stats.TotalMachineCost)

	return &stats, nil
}
