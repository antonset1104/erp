package integration_hub

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ----------------- Sequence Generators -----------------

func (r *Repository) GenerateLogCode() string {
	now := time.Now()
	var count int64
	r.db.Model(&IntegrationLog{}).Count(&count)
	return fmt.Sprintf("INTG/%d/%05d", now.Year(), count+1)
}

func (r *Repository) GenerateJobCode() string {
	now := time.Now()
	var count int64
	r.db.Model(&IntegrationDataJob{}).Count(&count)
	return fmt.Sprintf("JOB/%d/%05d", now.Year(), count+1)
}

// ----------------- API Keys -----------------

func (r *Repository) CreateAPIKey(key *APIKey) error {
	return r.db.Create(key).Error
}

func (r *Repository) GetAPIKeyByHash(keyHash string) (*APIKey, error) {
	var key APIKey
	err := r.db.Where("key_hash = ? AND is_active = ?", keyHash, true).First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *Repository) GetAPIKeyByID(id uint, companyID uint) (*APIKey, error) {
	var key APIKey
	err := r.db.Where("id = ? AND company_id = ?", id, companyID).First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *Repository) ListAPIKeys(companyID uint) ([]APIKey, error) {
	var keys []APIKey
	err := r.db.Where("company_id = ?", companyID).Order("id desc").Find(&keys).Error
	return keys, err
}

func (r *Repository) RevokeAPIKey(id uint, companyID uint) error {
	return r.db.Model(&APIKey{}).Where("id = ? AND company_id = ?", id, companyID).Update("is_active", false).Error
}

func (r *Repository) UpdateAPIKeyUsage(id uint) {
	now := time.Now()
	r.db.Model(&APIKey{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_used_at":   &now,
		"total_requests": gorm.Expr("total_requests + ?", 1),
	})
}

// ----------------- Connectors -----------------

func (r *Repository) CreateConnector(conn *IntegrationConnector) error {
	return r.db.Create(conn).Error
}

func (r *Repository) GetConnectorByCode(code string, companyID uint) (*IntegrationConnector, error) {
	var conn IntegrationConnector
	err := r.db.Where("code = ? AND company_id = ?", code, companyID).First(&conn).Error
	if err != nil {
		return nil, err
	}
	return &conn, nil
}

func (r *Repository) ListConnectors(companyID uint) ([]IntegrationConnector, error) {
	var conns []IntegrationConnector
	err := r.db.Where("company_id = ?", companyID).Order("id asc").Find(&conns).Error
	return conns, err
}

func (r *Repository) UpdateConnector(conn *IntegrationConnector) error {
	return r.db.Save(conn).Error
}

// ----------------- Telemetry Logs -----------------

func (r *Repository) CreateLog(log *IntegrationLog) error {
	return r.db.Create(log).Error
}

func (r *Repository) ListLogs(companyID uint, limit int) ([]IntegrationLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var logs []IntegrationLog
	err := r.db.Where("company_id = ?", companyID).Order("id desc").Limit(limit).Find(&logs).Error
	return logs, err
}

// ----------------- Data Jobs -----------------

func (r *Repository) CreateDataJob(job *IntegrationDataJob) error {
	return r.db.Create(job).Error
}

func (r *Repository) GetDataJobByID(id uint, companyID uint) (*IntegrationDataJob, error) {
	var job IntegrationDataJob
	err := r.db.Where("id = ? AND company_id = ?", id, companyID).First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *Repository) UpdateDataJob(job *IntegrationDataJob) error {
	return r.db.Save(job).Error
}

func (r *Repository) ListDataJobs(companyID uint) ([]IntegrationDataJob, error) {
	var jobs []IntegrationDataJob
	err := r.db.Where("company_id = ?", companyID).Order("id desc").Find(&jobs).Error
	return jobs, err
}

// ----------------- Statistics -----------------

func (r *Repository) GetStats(companyID uint) (*IntegrationHubStats, error) {
	var stats IntegrationHubStats

	r.db.Model(&APIKey{}).Where("company_id = ?", companyID).Count(&stats.TotalAPIKeys)
	r.db.Model(&APIKey{}).Where("company_id = ? AND is_active = ?", companyID, true).Count(&stats.ActiveAPIKeys)

	r.db.Model(&IntegrationConnector{}).Where("company_id = ?", companyID).Count(&stats.TotalConnectors)
	r.db.Model(&IntegrationConnector{}).Where("company_id = ? AND status = ?", companyID, "connected").Count(&stats.ConnectedConnectors)

	since24h := time.Now().Add(-24 * time.Hour)
	r.db.Model(&IntegrationLog{}).Where("company_id = ? AND created_at >= ?", companyID, since24h).Count(&stats.TotalCalls24h)
	r.db.Model(&IntegrationLog{}).Where("company_id = ? AND created_at >= ? AND status_code >= 200 AND status_code < 400", companyID, since24h).Count(&stats.SuccessCalls24h)
	r.db.Model(&IntegrationLog{}).Where("company_id = ? AND created_at >= ? AND status_code >= 400", companyID, since24h).Count(&stats.FailedCalls24h)

	if stats.TotalCalls24h > 0 {
		stats.SuccessRatePercent = float64(stats.SuccessCalls24h) / float64(stats.TotalCalls24h) * 100.0
	} else {
		stats.SuccessRatePercent = 100.0
	}

	var avgDuration float64
	r.db.Model(&IntegrationLog{}).Where("company_id = ? AND created_at >= ?", companyID, since24h).Select("COALESCE(AVG(duration_ms), 0)").Scan(&avgDuration)
	stats.AvgLatencyMs = avgDuration

	r.db.Model(&IntegrationDataJob{}).Where("company_id = ?", companyID).Count(&stats.TotalDataJobs)

	return &stats, nil
}

// ----------------- Model Registry & Dynamic Data Access -----------------

var SupportedModelTables = map[string]string{
	"res.partner":             "partners",
	"product.product":         "products",
	"product.template":        "product_templates",
	"stock.warehouse":         "stock_warehouses",
	"stock.location":          "stock_locations",
	"stock.picking":           "stock_pickings",
	"stock.quant":             "stock_quants",
	"sale.order":              "sale_orders",
	"sale.order.line":         "sale_order_lines",
	"purchase.order":          "purchase_orders",
	"purchase.order.line":     "purchase_order_lines",
	"account.move":            "account_moves",
	"account.move.line":       "account_move_lines",
	"account.payment":         "account_payments",
	"mrp.production":          "mrp_productions",
	"pos.order":               "pos_orders",
	"hr.employee":             "employees",
	"crm.lead":                "crm_leads",
	"project.project":         "projects",
	"helpdesk.ticket":         "helpdesk_tickets",
	"maintenance.equipment":   "maintenance_equipments",
	"fleet.vehicle":           "fleet_vehicles",
	"hr.leave":                "leave_requests",
	"hr.expense":              "expenses",
	"crossovered.budget":      "budgets",
	"webhook.subscription":    "webhook_subscriptions",
	"inbound.webhook.endpoint": "inbound_webhook_endpoints",
	"wms.bin":                 "wms_bins",
	"portal.vendor.rfq":       "portal_vendor_rfqs",
	"customs.document":        "customs_documents",
	"workflow.definition":     "workflow_definitions",
}

func (r *Repository) ResolveTableName(model string) (string, error) {
	tbl, ok := SupportedModelTables[model]
	if !ok {
		return "", fmt.Errorf("unsupported or unrecognized ERP model: '%s'", model)
	}
	return tbl, nil
}

func (r *Repository) ExecuteSearchRead(tableName string, domain [][]interface{}, fields []string, limit int, offset int, order string, companyID uint) ([]map[string]interface{}, int64, error) {
	query := r.db.Table(tableName)

	// Filter by company_id if table has it
	if r.db.Migrator().HasColumn(tableName, "company_id") {
		query = query.Where("company_id = ?", companyID)
	}

	// Filter soft delete if table has it
	if r.db.Migrator().HasColumn(tableName, "deleted_at") {
		query = query.Where("deleted_at IS NULL")
	}

	// Apply domain filters
	for _, cond := range domain {
		if len(cond) == 3 {
			field, okField := cond[0].(string)
			operator, okOp := cond[1].(string)
			val := cond[2]
			if okField && okOp {
				// Sanitize field name: only alphanumeric and underscore
				field = sanitizeIdentifier(field)
				switch operator {
				case "=", "==":
					query = query.Where(fmt.Sprintf("%s = ?", field), val)
				case "!=", "<>":
					query = query.Where(fmt.Sprintf("%s != ?", field), val)
				case ">":
					query = query.Where(fmt.Sprintf("%s > ?", field), val)
				case ">=":
					query = query.Where(fmt.Sprintf("%s >= ?", field), val)
				case "<":
					query = query.Where(fmt.Sprintf("%s < ?", field), val)
				case "<=":
					query = query.Where(fmt.Sprintf("%s <= ?", field), val)
				case "like", "ilike":
					query = query.Where(fmt.Sprintf("%s LIKE ?", field), fmt.Sprintf("%%%v%%", val))
				case "in":
					query = query.Where(fmt.Sprintf("%s IN (?)", field), val)
				}
			}
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply selected fields
	if len(fields) > 0 {
		var sanitizedFields []string
		for _, f := range fields {
			sanitizedFields = append(sanitizedFields, sanitizeIdentifier(f))
		}
		query = query.Select(sanitizedFields)
	}

	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query = query.Limit(limit).Offset(offset)

	if order != "" {
		query = query.Order(order)
	} else {
		query = query.Order("id desc")
	}

	var results []map[string]interface{}
	err := query.Find(&results).Error
	return results, total, err
}

func (r *Repository) ExecuteCreateRecord(tableName string, values map[string]interface{}, companyID uint) (uint, map[string]interface{}, error) {
	// Inject standard fields
	if r.db.Migrator().HasColumn(tableName, "company_id") {
		values["company_id"] = companyID
	}
	now := time.Now()
	if r.db.Migrator().HasColumn(tableName, "created_at") {
		values["created_at"] = now
	}
	if r.db.Migrator().HasColumn(tableName, "updated_at") {
		values["updated_at"] = now
	}

	err := r.db.Table(tableName).Create(&values).Error
	if err != nil {
		return 0, nil, err
	}

	var id uint
	if idVal, ok := values["id"]; ok {
		switch v := idVal.(type) {
		case uint:
			id = v
		case int:
			id = uint(v)
		case int64:
			id = uint(v)
		case float64:
			id = uint(v)
		}
	}

	return id, values, nil
}

func (r *Repository) ExecuteUpdateRecord(tableName string, id uint, values map[string]interface{}, companyID uint) error {
	query := r.db.Table(tableName).Where("id = ?", id)
	if r.db.Migrator().HasColumn(tableName, "company_id") {
		query = query.Where("company_id = ?", companyID)
	}
	if r.db.Migrator().HasColumn(tableName, "updated_at") {
		values["updated_at"] = time.Now()
	}

	return query.Updates(values).Error
}

func (r *Repository) ExecuteDeleteRecord(tableName string, id uint, companyID uint) error {
	query := r.db.Table(tableName).Where("id = ?", id)
	if r.db.Migrator().HasColumn(tableName, "company_id") {
		query = query.Where("company_id = ?", companyID)
	}

	if r.db.Migrator().HasColumn(tableName, "deleted_at") {
		now := time.Now()
		return query.Update("deleted_at", &now).Error
	}
	return query.Delete(nil).Error
}

func sanitizeIdentifier(s string) string {
	var clean strings.Builder
	for _, ch := range s {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			clean.WriteRune(ch)
		}
	}
	return clean.String()
}
