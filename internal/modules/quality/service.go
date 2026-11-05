package quality

import (
	"errors"
	"fmt"
	"math"
	"time"

	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/mrp"
	"erp-golang/internal/modules/stock"

	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	stockSvc   *stock.Service
	mrpSvc     *mrp.Service
	chatterSvc *chatter.Service
}

func NewService(db *gorm.DB, stockSvc *stock.Service, mrpSvc *mrp.Service, chatterSvc *chatter.Service) *Service {
	return &Service{
		db:         db,
		stockSvc:   stockSvc,
		mrpSvc:     mrpSvc,
		chatterSvc: chatterSvc,
	}
}

// ---------------------------------------------------------------------------
// Quality Points (quality.point)
// ---------------------------------------------------------------------------

func (s *Service) ListPoints(companyID uint, operationType string) ([]QualityPoint, error) {
	var points []QualityPoint
	query := s.db.Preload("Product").
		Where("company_id = ? OR company_id IS NULL OR company_id = 0", companyID)

	if operationType != "" {
		query = query.Where("operation_type = ?", operationType)
	}

	err := query.Order("id asc").Find(&points).Error
	return points, err
}

func (s *Service) GetPoint(id, companyID uint) (*QualityPoint, error) {
	var point QualityPoint
	err := s.db.Preload("Product").
		Where("id = ? AND (company_id = ? OR company_id IS NULL OR company_id = 0)", id, companyID).
		First(&point).Error
	if err != nil {
		return nil, errors.New("quality point not found")
	}
	return &point, nil
}

func (s *Service) CreatePoint(qp *QualityPoint) error {
	if qp.Title == "" {
		return errors.New("quality point title is required")
	}
	if qp.OperationType == "" {
		qp.OperationType = "incoming"
	}
	if qp.TestType == "" {
		qp.TestType = "passfail"
	}

	// Generate Code if empty
	if qp.Code == "" {
		var count int64
		s.db.Model(&QualityPoint{}).Count(&count)
		qp.Code = fmt.Sprintf("QP/%04d", count+1)
	}

	return s.db.Create(qp).Error
}

func (s *Service) UpdatePoint(qp *QualityPoint) error {
	return s.db.Save(qp).Error
}

func (s *Service) DeletePoint(id, companyID uint) error {
	return s.db.Delete(&QualityPoint{}, "id = ? AND company_id = ?", id, companyID).Error
}

// ---------------------------------------------------------------------------
// Quality Checks (quality.check)
// ---------------------------------------------------------------------------

func (s *Service) ListChecks(companyID uint, pickingID *uint, productionID *uint, state string) ([]QualityCheck, error) {
	var checks []QualityCheck
	query := s.db.Preload("Point").
		Preload("Product").
		Preload("Picking").
		Preload("Production").
		Preload("Inspector").
		Where("company_id = ?", companyID)

	if pickingID != nil && *pickingID > 0 {
		query = query.Where("picking_id = ?", *pickingID)
	}
	if productionID != nil && *productionID > 0 {
		query = query.Where("production_id = ?", *productionID)
	}
	if state != "" {
		query = query.Where("quality_state = ?", state)
	}

	err := query.Order("id desc").Find(&checks).Error
	return checks, err
}

func (s *Service) GetCheck(id, companyID uint) (*QualityCheck, error) {
	var check QualityCheck
	err := s.db.Preload("Point").
		Preload("Product").
		Preload("Picking").
		Preload("Production").
		Preload("Inspector").
		Where("id = ? AND company_id = ?", id, companyID).
		First(&check).Error
	if err != nil {
		return nil, errors.New("quality check not found")
	}
	return &check, nil
}

func (s *Service) CreateCheck(qc *QualityCheck, companyID, userID uint) error {
	if qc.Title == "" {
		return errors.New("quality check title is required")
	}
	if qc.ProductID == 0 {
		return errors.New("product is required for quality check")
	}
	if qc.QualityState == "" {
		qc.QualityState = "none"
	}
	qc.CompanyID = companyID

	// Generate Sequence QC/YYYY/00001
	if qc.Name == "" {
		year := time.Now().Format("2006")
		var count int64
		s.db.Model(&QualityCheck{}).Where("company_id = ?", companyID).Count(&count)
		qc.Name = fmt.Sprintf("QC/%s/%05d", year, count+1)
	}

	return s.db.Create(qc).Error
}

// GenerateChecksForPicking triggers quality checks for incoming shipments based on active quality points
func (s *Service) GenerateChecksForPicking(pickingID, companyID, userID uint) ([]QualityCheck, error) {
	var picking stock.StockPicking
	if err := s.db.Preload("MoveLines").Where("id = ? AND company_id = ?", pickingID, companyID).First(&picking).Error; err != nil {
		return nil, errors.New("stock picking not found")
	}

	opType := "incoming"
	if picking.PickingType == "outgoing" {
		opType = "outgoing"
	}

	// Find applicable points
	var points []QualityPoint
	if err := s.db.Where("operation_type = ? AND active = ?", opType, true).Find(&points).Error; err != nil {
		return nil, err
	}

	var generated []QualityCheck
	for _, move := range picking.MoveLines {
		for _, qp := range points {
			// Match if point has no product restriction OR matches the move's product
			if qp.ProductID == nil || *qp.ProductID == move.ProductID {
				qc := QualityCheck{
					PointID:      &qp.ID,
					Title:        qp.Title,
					ProductID:    move.ProductID,
					PickingID:    &picking.ID,
					TestType:     qp.TestType,
					Norm:         qp.Norm,
					ToleranceMin: qp.ToleranceMin,
					ToleranceMax: qp.ToleranceMax,
					NormUnit:     qp.NormUnit,
					QualityState: "none",
					Notes:        qp.Instructions,
				}
				if err := s.CreateCheck(&qc, companyID, userID); err == nil {
					generated = append(generated, qc)
				}
			}
		}
	}

	if len(generated) > 0 && s.chatterSvc != nil {
		msg := chatter.MailMessage{
			ModelName:   "stock.picking",
			ResID:       picking.ID,
			AuthorID:    userID,
			Body:        fmt.Sprintf("Dibuat otomatis %d Quality Check untuk inspeksi penerimaan barang ini.", len(generated)),
			MessageType: "notification",
		}
		msg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(&msg)
	}

	return generated, nil
}

// GenerateChecksForProduction triggers quality checks for manufacturing orders
func (s *Service) GenerateChecksForProduction(productionID, companyID, userID uint) ([]QualityCheck, error) {
	var mo mrp.MrpProduction
	if err := s.db.Where("id = ? AND company_id = ?", productionID, companyID).First(&mo).Error; err != nil {
		return nil, errors.New("manufacturing order not found")
	}

	var points []QualityPoint
	if err := s.db.Where("operation_type = 'mrp_operation' AND active = ?", true).Find(&points).Error; err != nil {
		return nil, err
	}

	var generated []QualityCheck
	for _, qp := range points {
		if qp.ProductID == nil || *qp.ProductID == mo.ProductID {
			qc := QualityCheck{
				PointID:      &qp.ID,
				Title:        qp.Title,
				ProductID:    mo.ProductID,
				ProductionID: &mo.ID,
				TestType:     qp.TestType,
				Norm:         qp.Norm,
				ToleranceMin: qp.ToleranceMin,
				ToleranceMax: qp.ToleranceMax,
				NormUnit:     qp.NormUnit,
				QualityState: "none",
				Notes:        qp.Instructions,
			}
			if err := s.CreateCheck(&qc, companyID, userID); err == nil {
				generated = append(generated, qc)
			}
		}
	}

	if len(generated) > 0 && s.chatterSvc != nil {
		msg := chatter.MailMessage{
			ModelName:   "mrp.production",
			ResID:       mo.ID,
			AuthorID:    userID,
			Body:        fmt.Sprintf("Dibuat otomatis %d Quality Check untuk inspeksi proses perakitan manufaktur ini.", len(generated)),
			MessageType: "notification",
		}
		msg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(&msg)
	}

	return generated, nil
}

// ActionPassCheck marks an inspection check as PASSED
func (s *Service) ActionPassCheck(checkID, companyID, inspectorID uint, measureVal *float64, notes string) (*QualityCheck, error) {
	qc, err := s.GetCheck(checkID, companyID)
	if err != nil {
		return nil, err
	}

	if qc.TestType == "measure" {
		if measureVal == nil {
			return nil, errors.New("quantitative measure value is required for measurement test type")
		}
		val := *measureVal
		// Validate against tolerances if specified
		if qc.ToleranceMin > 0 && val < qc.ToleranceMin {
			return nil, fmt.Errorf("measure value %.2f %s is below minimum tolerance %.2f %s", val, qc.NormUnit, qc.ToleranceMin, qc.NormUnit)
		}
		if qc.ToleranceMax > 0 && val > qc.ToleranceMax {
			return nil, fmt.Errorf("measure value %.2f %s exceeds maximum tolerance %.2f %s", val, qc.NormUnit, qc.ToleranceMax, qc.NormUnit)
		}
		qc.Measure = val
	}

	qc.QualityState = "pass"
	qc.InspectorID = &inspectorID
	qc.CheckDate = time.Now().Format("2006-01-02 15:04:05")
	if notes != "" {
		qc.Notes = notes
	}

	if err := s.db.Save(qc).Error; err != nil {
		return nil, err
	}

	// Post chatter log
	if s.chatterSvc != nil {
		var modelName string
		var resID uint
		if qc.PickingID != nil {
			modelName = "stock.picking"
			resID = *qc.PickingID
		} else if qc.ProductionID != nil {
			modelName = "mrp.production"
			resID = *qc.ProductionID
		}

		if modelName != "" {
			msg := chatter.MailMessage{
				ModelName:   modelName,
				ResID:       resID,
				AuthorID:    inspectorID,
				Body:        fmt.Sprintf("✓ Uji Kualitas '%s' (%s) DINYATAKAN LOLOS (Pass).", qc.Title, qc.Name),
				MessageType: "comment",
			}
			msg.CompanyID = companyID
			_ = s.chatterSvc.PostMessage(&msg)
		}
	}

	return s.GetCheck(qc.ID, companyID)
}

// ActionFailCheck marks an inspection check as FAILED and optionally raises a Quality Alert (NCR)
func (s *Service) ActionFailCheck(checkID, companyID, inspectorID uint, measureVal *float64, reason string, autoCreateAlert bool) (*QualityCheck, *QualityAlert, error) {
	qc, err := s.GetCheck(checkID, companyID)
	if err != nil {
		return nil, nil, err
	}
	if reason == "" {
		reason = "Hasil inspeksi tidak memenuhi standar mutu atau spesifikasi teknis."
	}

	if measureVal != nil {
		qc.Measure = *measureVal
	}
	qc.QualityState = "fail"
	qc.InspectorID = &inspectorID
	qc.CheckDate = time.Now().Format("2006-01-02 15:04:05")
	qc.FailureReason = reason

	if err := s.db.Save(qc).Error; err != nil {
		return nil, nil, err
	}

	var alert *QualityAlert
	if autoCreateAlert {
		qa := QualityAlert{
			Title:            fmt.Sprintf("Ketidaksesuaian Mutu: %s (%s)", qc.Title, qc.Product.Name),
			CheckID:          &qc.ID,
			ProductID:        &qc.ProductID,
			PickingID:        qc.PickingID,
			ProductionID:     qc.ProductionID,
			Priority:         "2", // High
			Stage:            "new",
			ActionType:       "quarantine",
			RootCause:        reason,
			PreventiveAction: "Karantina barang yang bersangkutan dan lakukan evaluasi bersama vendor/produksi.",
			UserID:           &inspectorID,
			Notes:            fmt.Sprintf("Dibuat otomatis dari kegagalan inspeksi %s.", qc.Name),
		}
		if err := s.CreateAlert(&qa, companyID, inspectorID); err == nil {
			alert = &qa
		}
	}

	// Post chatter log
	if s.chatterSvc != nil {
		var modelName string
		var resID uint
		if qc.PickingID != nil {
			modelName = "stock.picking"
			resID = *qc.PickingID
		} else if qc.ProductionID != nil {
			modelName = "mrp.production"
			resID = *qc.ProductionID
		}

		if modelName != "" {
			alertInfo := ""
			if alert != nil {
				alertInfo = fmt.Sprintf(" Diterbitkan Quality Alert '%s'.", alert.Name)
			}
			msg := chatter.MailMessage{
				ModelName:   modelName,
				ResID:       resID,
				AuthorID:    inspectorID,
				Body:        fmt.Sprintf("⚠️ Uji Kualitas '%s' (%s) GAGAL (Fail): %s.%s", qc.Title, qc.Name, reason, alertInfo),
				MessageType: "comment",
			}
			msg.CompanyID = companyID
			_ = s.chatterSvc.PostMessage(&msg)
		}
	}

	return qc, alert, nil
}

// ---------------------------------------------------------------------------
// Quality Alerts (quality.alert)
// ---------------------------------------------------------------------------

func (s *Service) ListAlerts(companyID uint, stage string, priority string) ([]QualityAlert, error) {
	var alerts []QualityAlert
	query := s.db.Preload("Check").
		Preload("Product").
		Preload("Picking").
		Preload("Production").
		Preload("Partner").
		Preload("User").
		Where("company_id = ?", companyID)

	if stage != "" {
		query = query.Where("stage = ?", stage)
	}
	if priority != "" {
		query = query.Where("priority = ?", priority)
	}

	err := query.Order("id desc").Find(&alerts).Error
	return alerts, err
}

func (s *Service) GetAlert(id, companyID uint) (*QualityAlert, error) {
	var alert QualityAlert
	err := s.db.Preload("Check").
		Preload("Product").
		Preload("Picking").
		Preload("Production").
		Preload("Partner").
		Preload("User").
		Where("id = ? AND company_id = ?", id, companyID).
		First(&alert).Error
	if err != nil {
		return nil, errors.New("quality alert not found")
	}
	return &alert, nil
}

func (s *Service) CreateAlert(qa *QualityAlert, companyID, userID uint) error {
	if qa.Title == "" {
		return errors.New("quality alert title is required")
	}
	if qa.Stage == "" {
		qa.Stage = "new"
	}
	if qa.Priority == "" {
		qa.Priority = "1"
	}
	if qa.ActionType == "" {
		qa.ActionType = "quarantine"
	}
	qa.CompanyID = companyID

	// Generate Sequence QA/YYYY/00001
	if qa.Name == "" {
		year := time.Now().Format("2006")
		var count int64
		s.db.Model(&QualityAlert{}).Where("company_id = ?", companyID).Count(&count)
		qa.Name = fmt.Sprintf("QA/%s/%05d", year, count+1)
	}

	if err := s.db.Create(qa).Error; err != nil {
		return err
	}

	if s.chatterSvc != nil {
		msg := chatter.MailMessage{
			ModelName:   "quality.alert",
			ResID:       qa.ID,
			AuthorID:    userID,
			Body:        fmt.Sprintf("Quality Alert '%s' (%s) diterbitkan dengan tindakan '%s'.", qa.Name, qa.Title, qa.ActionType),
			MessageType: "notification",
		}
		msg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(&msg)
	}

	return nil
}

func (s *Service) UpdateAlertStage(id, companyID uint, newStage string, rootCause, preventiveAction, actionType string, userID uint) (*QualityAlert, error) {
	qa, err := s.GetAlert(id, companyID)
	if err != nil {
		return nil, err
	}

	oldStage := qa.Stage
	qa.Stage = newStage
	if rootCause != "" {
		qa.RootCause = rootCause
	}
	if preventiveAction != "" {
		qa.PreventiveAction = preventiveAction
	}
	if actionType != "" {
		qa.ActionType = actionType
	}
	if newStage == "closed" {
		qa.CloseDate = time.Now().Format("2006-01-02 15:04:05")
	}

	if err := s.db.Save(qa).Error; err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		msg := chatter.MailMessage{
			ModelName:   "quality.alert",
			ResID:       qa.ID,
			AuthorID:    userID,
			Body:        fmt.Sprintf("Tahapan Quality Alert diperbarui dari '%s' ke '%s'.", oldStage, newStage),
			MessageType: "comment",
		}
		msg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(&msg)
	}

	return s.GetAlert(qa.ID, companyID)
}

// ---------------------------------------------------------------------------
// Analytics & Stats
// ---------------------------------------------------------------------------

func (s *Service) GetQualityStats(companyID uint) (map[string]interface{}, error) {
	var totalPoints int64
	s.db.Model(&QualityPoint{}).Where("company_id = ? OR company_id IS NULL OR company_id = 0", companyID).Count(&totalPoints)

	var totalChecks int64
	var pendingChecks int64
	var passedChecks int64
	var failedChecks int64

	s.db.Model(&QualityCheck{}).Where("company_id = ?", companyID).Count(&totalChecks)
	s.db.Model(&QualityCheck{}).Where("company_id = ? AND quality_state = 'none'", companyID).Count(&pendingChecks)
	s.db.Model(&QualityCheck{}).Where("company_id = ? AND quality_state = 'pass'", companyID).Count(&passedChecks)
	s.db.Model(&QualityCheck{}).Where("company_id = ? AND quality_state = 'fail'", companyID).Count(&failedChecks)

	passRate := float64(0)
	inspectedChecks := passedChecks + failedChecks
	if inspectedChecks > 0 {
		passRate = math.Round((float64(passedChecks)/float64(inspectedChecks))*1000) / 10
	}

	var totalAlerts int64
	var activeAlerts int64
	var criticalAlerts int64
	var closedAlerts int64

	s.db.Model(&QualityAlert{}).Where("company_id = ?", companyID).Count(&totalAlerts)
	s.db.Model(&QualityAlert{}).Where("company_id = ? AND stage NOT IN ('resolved', 'closed')", companyID).Count(&activeAlerts)
	s.db.Model(&QualityAlert{}).Where("company_id = ? AND priority = '3' AND stage != 'closed'", companyID).Count(&criticalAlerts)
	s.db.Model(&QualityAlert{}).Where("company_id = ? AND stage IN ('resolved', 'closed')", companyID).Count(&closedAlerts)

	return map[string]interface{}{
		"total_points":    totalPoints,
		"total_checks":    totalChecks,
		"pending_checks":  pendingChecks,
		"passed_checks":   passedChecks,
		"failed_checks":   failedChecks,
		"pass_rate_pct":   passRate,
		"total_alerts":    totalAlerts,
		"active_alerts":   activeAlerts,
		"critical_alerts": criticalAlerts,
		"closed_alerts":   closedAlerts,
	}, nil
}
