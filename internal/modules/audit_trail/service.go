package audit_trail

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/chatter"

	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	repo       *Repository
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewService(db *gorm.DB, repo *Repository, chatterSvc *chatter.Service, aiClient *ai.Client) *Service {
	return &Service{
		db:         db,
		repo:       repo,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

// ---------------------------------------------------------------------------
// Rule Management
// ---------------------------------------------------------------------------

func (s *Service) CreateRule(req CreateRuleRequest, companyID uint) (*AuditLogRule, error) {
	if companyID == 0 {
		companyID = 1
	}

	logCreate := true
	if req.LogCreate != nil {
		logCreate = *req.LogCreate
	}
	logWrite := true
	if req.LogWrite != nil {
		logWrite = *req.LogWrite
	}
	logUnlink := true
	if req.LogUnlink != nil {
		logUnlink = *req.LogUnlink
	}
	logRead := false
	if req.LogRead != nil {
		logRead = *req.LogRead
	}
	captureDiff := true
	if req.CaptureDiff != nil {
		captureDiff = *req.CaptureDiff
	}
	retentionDays := 365
	if req.RetentionDays > 0 {
		retentionDays = req.RetentionDays
	}

	rule := &AuditLogRule{
		ModelBase: models.ModelBase{
			CompanyID: companyID,
			Active:    true,
		},
		Name:          req.Name,
		ModelName:     req.ModelName,
		LogCreate:     logCreate,
		LogWrite:      logWrite,
		LogUnlink:     logUnlink,
		LogRead:       logRead,
		TrackFields:   req.TrackFields,
		CaptureDiff:   captureDiff,
		RetentionDays: retentionDays,
		Description:   req.Description,
	}

	if err := s.repo.CreateRule(rule); err != nil {
		return nil, fmt.Errorf("failed to create audit rule: %w", err)
	}

	return rule, nil
}

func (s *Service) GetRuleByID(id uint, companyID uint) (*AuditLogRule, error) {
	return s.repo.GetRuleByID(id, companyID)
}

func (s *Service) ListRules(companyID uint) ([]AuditLogRule, error) {
	return s.repo.ListRules(companyID)
}

func (s *Service) UpdateRule(id uint, req CreateRuleRequest, companyID uint) (*AuditLogRule, error) {
	rule, err := s.repo.GetRuleByID(id, companyID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		rule.Name = req.Name
	}
	if req.ModelName != "" {
		rule.ModelName = req.ModelName
	}
	if req.LogCreate != nil {
		rule.LogCreate = *req.LogCreate
	}
	if req.LogWrite != nil {
		rule.LogWrite = *req.LogWrite
	}
	if req.LogUnlink != nil {
		rule.LogUnlink = *req.LogUnlink
	}
	if req.LogRead != nil {
		rule.LogRead = *req.LogRead
	}
	if req.TrackFields != "" {
		rule.TrackFields = req.TrackFields
	}
	if req.CaptureDiff != nil {
		rule.CaptureDiff = *req.CaptureDiff
	}
	if req.RetentionDays > 0 {
		rule.RetentionDays = req.RetentionDays
	}
	if req.Description != "" {
		rule.Description = req.Description
	}

	if err := s.repo.UpdateRule(rule); err != nil {
		return nil, err
	}
	return rule, nil
}

func (s *Service) DeleteRule(id uint, companyID uint) error {
	return s.repo.DeleteRule(id, companyID)
}

// ---------------------------------------------------------------------------
// CDC Engine: Change Recording & Diff Analysis
// ---------------------------------------------------------------------------

var sensitiveFields = map[string]bool{
	"wage":            true,
	"salary":          true,
	"basic_salary":    true,
	"price_unit":      true,
	"price":           true,
	"amount_total":    true,
	"amount_untaxed":  true,
	"balance":         true,
	"debit":           true,
	"credit":          true,
	"bank_account":    true,
	"bank_account_id": true,
	"bank_id":         true,
	"password":        true,
	"secret":          true,
	"state":           true,
	"status":          true,
	"mrr":             true,
	"arr":             true,
	"rate":            true,
}

// RecordChange captures an entity modification, computes field diffs, performs anomaly scoring, and saves an immutable audit log entry
func (s *Service) RecordChange(req RecordChangeRequest, userID uint, userName string, userEmail string, companyID uint) (*AuditLogEntry, error) {
	if companyID == 0 {
		companyID = 1
	}

	// 1. Check if audit rule applies for this model
	rule, _ := s.repo.GetRuleByModel(req.ModelName, companyID)
	if rule != nil && !rule.Active {
		// Auditing explicitly deactivated for this model
		return nil, nil
	}

	if rule != nil {
		switch req.Action {
		case ActionCreate:
			if !rule.LogCreate {
				return nil, nil
			}
		case ActionUpdate:
			if !rule.LogWrite {
				return nil, nil
			}
		case ActionDelete:
			if !rule.LogUnlink {
				return nil, nil
			}
		case ActionRead:
			if !rule.LogRead {
				return nil, nil
			}
		}
	}

	// 2. Compute field diffs
	diffs, oldJSON, newJSON := s.computeDiffs(req.OldValues, req.NewValues, rule)

	// 3. Anomaly Detection & Risk Scoring
	isSuspicious, anomalyReason, riskScore := s.detectAnomalies(req, diffs)

	// 4. Generate Log Code
	logCode, err := s.repo.GenerateLogCode(companyID)
	if err != nil {
		logCode = fmt.Sprintf("AUD/%d/%d", time.Now().Year(), time.Now().UnixNano()%100000)
	}

	nowStr := time.Now().Format("2006-01-02 15:04:05")
	summary := req.Summary
	if summary == "" {
		summary = fmt.Sprintf("%s pada %s #%d (%s) oleh %s", req.Action, req.ModelName, req.ResID, req.ResName, userName)
	}

	entry := &AuditLogEntry{
		ModelBase: models.ModelBase{
			CompanyID: companyID,
			CreateUID: userID,
			WriteUID:  userID,
			Active:    true,
		},
		LogCode:       logCode,
		ModelName:     req.ModelName,
		ResID:         req.ResID,
		ResName:       req.ResName,
		Action:        req.Action,
		UserID:        userID,
		UserName:      userName,
		UserEmail:     userEmail,
		IPAddress:     req.IPAddress,
		UserAgent:     req.UserAgent,
		Timestamp:     nowStr,
		Summary:       summary,
		OldValuesJSON: oldJSON,
		NewValuesJSON: newJSON,
		Diffs:         diffs,
		IsSuspicious:  isSuspicious,
		AnomalyReason: anomalyReason,
		RiskScore:     riskScore,
	}

	if err := s.repo.CreateEntry(entry); err != nil {
		return nil, fmt.Errorf("failed to save audit log entry: %w", err)
	}

	// 5. Integrate with Chatter tracking if service available
	if s.chatterSvc != nil && req.ModelName != "" && req.ResID > 0 {
		trackingBytes, _ := json.Marshal(diffs)
		chatterText := fmt.Sprintf("[%s] Audit Trail: %s", req.Action, summary)
		if isSuspicious {
			chatterText = fmt.Sprintf("⚠️ [SUSPICIOUS %d/100] %s (Alasan: %s)", riskScore, chatterText, anomalyReason)
		}
		_ = s.chatterSvc.LogTracking(req.ModelName, req.ResID, userID, userName, chatterText, string(trackingBytes), companyID)
	}

	return entry, nil
}

func (s *Service) computeDiffs(oldVals, newVals map[string]interface{}, rule *AuditLogRule) ([]AuditLogFieldDiff, string, string) {
	var diffs []AuditLogFieldDiff
	var allowedFields map[string]bool
	if rule != nil && rule.TrackFields != "" && rule.TrackFields != "*" {
		allowedFields = make(map[string]bool)
		parts := strings.Split(rule.TrackFields, ",")
		for _, p := range parts {
			allowedFields[strings.TrimSpace(p)] = true
		}
	}

	allKeys := make(map[string]bool)
	for k := range oldVals {
		allKeys[k] = true
	}
	for k := range newVals {
		allKeys[k] = true
	}

	for k := range allKeys {
		if k == "created_at" || k == "updated_at" || k == "deleted_at" || k == "write_uid" || k == "write_date" {
			continue
		}
		if allowedFields != nil && !allowedFields[k] {
			continue
		}

		oldV, oldOk := oldVals[k]
		newV, newOk := newVals[k]

		oldStr := fmt.Sprintf("%v", oldV)
		newStr := fmt.Sprintf("%v", newV)
		if !oldOk {
			oldStr = "<nil>"
		}
		if !newOk {
			newStr = "<nil>"
		}

		if oldStr != newStr {
			valType := detectValueType(newV, oldV)
			isSens := sensitiveFields[strings.ToLower(k)]
			diffs = append(diffs, AuditLogFieldDiff{
				FieldName:   k,
				OldValue:    oldStr,
				NewValue:    newStr,
				ValueType:   valType,
				IsSensitive: isSens,
			})
		}
	}

	oldBytes, _ := json.Marshal(oldVals)
	newBytes, _ := json.Marshal(newVals)
	return diffs, string(oldBytes), string(newBytes)
}

func detectValueType(newV, oldV interface{}) string {
	v := newV
	if v == nil {
		v = oldV
	}
	if v == nil {
		return "string"
	}
	switch v.(type) {
	case float32, float64:
		return "float"
	case int, int32, int64, uint, uint32, uint64:
		return "int"
	case bool:
		return "bool"
	default:
		return "string"
	}
}

func (s *Service) detectAnomalies(req RecordChangeRequest, diffs []AuditLogFieldDiff) (bool, string, int) {
	riskScore := 0
	var reasons []string

	// 1. Off-hours activity check (e.g., 23:00 - 05:00 WIB)
	hour := time.Now().Hour()
	if hour >= 23 || hour < 5 {
		riskScore += 35
		reasons = append(reasons, fmt.Sprintf("Aktivitas dilakukan di luar jam kerja resmi (%02d:00 WIB)", hour))
	}

	// 2. Sensitive fields modified
	hasSensitive := false
	for _, d := range diffs {
		if d.IsSensitive {
			hasSensitive = true
			riskScore += 25
			reasons = append(reasons, fmt.Sprintf("Modifikasi data sensitif/finansial pada kolom '%s'", d.FieldName))
			break
		}
	}

	// 3. Modifying financial journal in posted/closed state
	if req.ModelName == "account.move" {
		for _, d := range diffs {
			if d.FieldName == "state" && d.OldValue == "posted" && d.NewValue == "draft" {
				riskScore += 45
				reasons = append(reasons, "Pembatalan (unpost) transaksi akuntansi yang telah dibukukan")
			}
			if d.FieldName == "amount_total" {
				riskScore += 20
			}
		}
	}

	// 4. Large value shift (> 100% or absolute > 50jt)
	for _, d := range diffs {
		if d.ValueType == "float" || d.ValueType == "int" {
			oldNum, err1 := strconv.ParseFloat(d.OldValue, 64)
			newNum, err2 := strconv.ParseFloat(d.NewValue, 64)
			if err1 == nil && err2 == nil && oldNum > 0 {
				pct := math.Abs((newNum - oldNum) / oldNum)
				if pct >= 1.0 { // >= 100% change
					riskScore += 30
					reasons = append(reasons, fmt.Sprintf("Lonjakan nilai drastis (%.0f%%) pada '%s'", pct*100, d.FieldName))
				}
				if math.Abs(newNum-oldNum) >= 50000000 {
					riskScore += 25
					reasons = append(reasons, fmt.Sprintf("Perubahan nominal signifikan (Rp %.0f) pada '%s'", math.Abs(newNum-oldNum), d.FieldName))
				}
			}
		}
	}

	// 5. Deletion of critical master data
	if req.Action == ActionDelete {
		riskScore += 30
		reasons = append(reasons, fmt.Sprintf("Penghapusan record pada model '%s'", req.ModelName))
	}

	// 6. Many fields changed simultaneously (> 8 fields)
	if len(diffs) > 8 && hasSensitive {
		riskScore += 20
		reasons = append(reasons, fmt.Sprintf("Perubahan massal %d kolom secara simultan", len(diffs)))
	}

	if riskScore > 100 {
		riskScore = 100
	}

	isSuspicious := riskScore >= 50
	anomalyReason := strings.Join(reasons, "; ")
	return isSuspicious, anomalyReason, riskScore
}

// ---------------------------------------------------------------------------
// Rollback Engine: Immutable Reversion with Compensating Log
// ---------------------------------------------------------------------------

func resolveTableName(modelName string) string {
	switch modelName {
	case "account.move":
		return "account_moves"
	case "account.move.line":
		return "account_move_lines"
	case "hr.employee":
		return "employees"
	case "sale.order":
		return "sale_orders"
	case "sale.order.line":
		return "sale_order_lines"
	case "purchase.order":
		return "purchase_orders"
	case "stock.picking":
		return "stock_pickings"
	case "product.product":
		return "products"
	case "res.partner":
		return "partners"
	case "sale.subscription":
		return "subscriptions"
	case "bank.statement":
		return "account_bank_statements"
	case "stock.lot":
		return "stock_production_lots"
	default:
		clean := strings.ReplaceAll(modelName, ".", "_")
		if strings.HasSuffix(clean, "s") {
			return clean
		}
		return clean + "s"
	}
}

// RollbackRecord restores previous values of a target record and issues an immutable ActionRollback audit log
func (s *Service) RollbackRecord(entryID uint, rollbackReason string, userID uint, userName string, userEmail string, companyID uint) (*AuditLogEntry, error) {
	if companyID == 0 {
		companyID = 1
	}

	targetEntry, err := s.repo.GetEntryByID(entryID, companyID)
	if err != nil {
		return nil, fmt.Errorf("audit log #%d tidak ditemukan: %w", entryID, err)
	}

	if targetEntry.Action == ActionRollback {
		return nil, errors.New("tidak dapat melakukan rollback pada entri yang sudah berstatus ROLLBACK")
	}

	tableName := resolveTableName(targetEntry.ModelName)

	var oldVals map[string]interface{}
	if targetEntry.OldValuesJSON != "" && targetEntry.OldValuesJSON != "{}" && targetEntry.OldValuesJSON != "null" {
		if err := json.Unmarshal([]byte(targetEntry.OldValuesJSON), &oldVals); err != nil {
			return nil, fmt.Errorf("gagal membaca snapshot nilai lama: %w", err)
		}
	}

	var newVals map[string]interface{}
	if targetEntry.NewValuesJSON != "" && targetEntry.NewValuesJSON != "{}" && targetEntry.NewValuesJSON != "null" {
		_ = json.Unmarshal([]byte(targetEntry.NewValuesJSON), &newVals)
	}

	// Execute rollback in transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	switch targetEntry.Action {
	case ActionUpdate:
		if len(oldVals) == 0 {
			tx.Rollback()
			return nil, errors.New("nilai lama tidak tersedia untuk melakukan update rollback")
		}
		// Strip unmodifiable base fields
		filtered := make(map[string]interface{})
		for k, v := range oldVals {
			if k != "id" && k != "created_at" && k != "deleted_at" {
				filtered[k] = v
			}
		}
		filtered["updated_at"] = time.Now()
		if err := tx.Table(tableName).Where("id = ?", targetEntry.ResID).Updates(filtered).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("gagal merestorasi tabel %s: %w", tableName, err)
		}

	case ActionCreate:
		// For created records, rollback by deactivating or deleting
		if err := tx.Table(tableName).Where("id = ?", targetEntry.ResID).Update("active", false).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("gagal menonaktifkan record baru pada tabel %s: %w", tableName, err)
		}

	case ActionDelete:
		// For deleted records, reactivate
		if err := tx.Table(tableName).Where("id = ?", targetEntry.ResID).Update("active", true).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("gagal mengaktifkan kembali record pada tabel %s: %w", tableName, err)
		}
	}

	// Generate compensating audit log
	logCode, err := s.repo.GenerateLogCode(companyID)
	if err != nil {
		logCode = fmt.Sprintf("AUD/%d/%d", time.Now().Year(), time.Now().UnixNano()%100000)
	}

	nowStr := time.Now().Format("2006-01-02 15:04:05")
	summary := fmt.Sprintf("ROLLBACK atas Log %s (%s #%d). Alasan: %s", targetEntry.LogCode, targetEntry.ModelName, targetEntry.ResID, rollbackReason)

	// Invert diffs for the compensating log
	var compDiffs []AuditLogFieldDiff
	for _, d := range targetEntry.Diffs {
		compDiffs = append(compDiffs, AuditLogFieldDiff{
			FieldName:   d.FieldName,
			OldValue:    d.NewValue,
			NewValue:    d.OldValue,
			ValueType:   d.ValueType,
			IsSensitive: d.IsSensitive,
		})
	}
	if len(compDiffs) == 0 && len(oldVals) > 0 {
		for k, oldV := range oldVals {
			newV := newVals[k]
			oldStr := fmt.Sprintf("%v", oldV)
			newStr := fmt.Sprintf("%v", newV)
			if oldStr != newStr {
				compDiffs = append(compDiffs, AuditLogFieldDiff{
					FieldName:   k,
					OldValue:    newStr,
					NewValue:    oldStr,
					ValueType:   detectValueType(oldV, newV),
					IsSensitive: sensitiveFields[strings.ToLower(k)],
				})
			}
		}
	}

	compEntry := &AuditLogEntry{
		ModelBase: models.ModelBase{
			CompanyID: companyID,
			CreateUID: userID,
			WriteUID:  userID,
			Active:    true,
		},
		LogCode:       logCode,
		ModelName:     targetEntry.ModelName,
		ResID:         targetEntry.ResID,
		ResName:       targetEntry.ResName,
		Action:        ActionRollback,
		UserID:        userID,
		UserName:      userName,
		UserEmail:     userEmail,
		Timestamp:     nowStr,
		Summary:       summary,
		OldValuesJSON: targetEntry.NewValuesJSON,
		NewValuesJSON: targetEntry.OldValuesJSON,
		Diffs:         compDiffs,
		IsSuspicious:  false,
		RiskScore:     10,
	}

	if err := tx.Create(compEntry).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("gagal mencatat compensating audit log: %w", err)
	}

	for i := range compDiffs {
		compDiffs[i].AuditLogID = compEntry.ID
		compDiffs[i].CompanyID = companyID
		compDiffs[i].Active = true
		_ = tx.Create(&compDiffs[i]).Error
	}
	compEntry.Diffs = compDiffs

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("gagal commit transaksi rollback: %w", err)
	}

	// Chatter log
	if s.chatterSvc != nil && targetEntry.ModelName != "" && targetEntry.ResID > 0 {
		_ = s.chatterSvc.LogTracking(targetEntry.ModelName, targetEntry.ResID, userID, userName, fmt.Sprintf("🔄 %s", summary), "", companyID)
	}

	return compEntry, nil
}

// ---------------------------------------------------------------------------
// Security & Anomaly Audit Copilot (Cloudflare Workers AI + Local Heuristics)
// ---------------------------------------------------------------------------

func (s *Service) AnalyzeAnomalies(companyID uint) (*AIAuditAnomalyReport, error) {
	if companyID == 0 {
		companyID = 1
	}

	logs, err := s.repo.ListRecentEntries(companyID, 100)
	if err != nil {
		return nil, err
	}

	var suspiciousLogs []AuditLogEntry
	for _, l := range logs {
		if l.IsSuspicious || l.RiskScore >= 50 {
			suspiciousLogs = append(suspiciousLogs, l)
		}
	}

	totalAnalyzed := len(logs)
	suspiciousCount := len(suspiciousLogs)

	// Calculate baseline security score
	securityScore := 100
	if totalAnalyzed > 0 {
		penalty := (suspiciousCount * 100) / totalAnalyzed
		securityScore = 100 - penalty
		if securityScore < 20 && suspiciousCount > 0 {
			securityScore = 20
		}
	}

	complianceStatus := "COMPLIANT"
	if securityScore < 70 {
		complianceStatus = "AUDIT_WARNING"
	}
	if securityScore < 50 {
		complianceStatus = "NON_COMPLIANT"
	}

	// If Cloudflare Workers AI configured, query it for expert auditor review
	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() && suspiciousCount > 0 {
		report, err := s.callAIAnomalyAudit(logs, suspiciousLogs, totalAnalyzed, securityScore, complianceStatus)
		if err == nil && report != nil {
			return report, nil
		}
	}

	// Fallback to rich rule-based heuristics
	var keyObs []string
	var actions []string

	if suspiciousCount == 0 {
		keyObs = append(keyObs, "Seluruh transaksi dan perubahan data berada dalam koridor operasional normal (Zero High-Risk Anomalies).")
		keyObs = append(keyObs, "Tidak ditemukan aktivitas modifikasi di luar jam kerja resmi pada record finansial.")
		actions = append(actions, "Pertahankan audit rule aktif untuk seluruh modul finansial (account.move) dan kepegawaian (hr.employee).")
	} else {
		keyObs = append(keyObs, fmt.Sprintf("Terdeteksi %d aktivitas mencurigakan dari %d transaksi yang dianalisis.", suspiciousCount, totalAnalyzed))
		for i, sl := range suspiciousLogs {
			if i >= 4 {
				break
			}
			keyObs = append(keyObs, fmt.Sprintf("Log %s: %s (Skor Risiko: %d) - %s", sl.LogCode, sl.Summary, sl.RiskScore, sl.AnomalyReason))
		}
		actions = append(actions, "Lakukan verifikasi 2-eye principle terhadap log yang memiliki RiskScore >= 70.")
		actions = append(actions, "Gunakan fitur 'Rollback' jika ditemukan manipulasi atau ketidaksengajaan data.")
		actions = append(actions, "Pastikan akun pengguna dengan aktivitas off-hours diautentikasi dengan MFA.")
	}

	execSummary := fmt.Sprintf("Evaluasi audit internal mengidentifikasi skor keamanan %d/100 dengan status kepatuhan %s. Terpantau %d anomali dari total %d log aktivitas terkini.",
		securityScore, complianceStatus, suspiciousCount, totalAnalyzed)

	return &AIAuditAnomalyReport{
		TotalAnalyzed:        totalAnalyzed,
		SuspiciousCount:      suspiciousCount,
		OverallSecurityScore: securityScore,
		ComplianceStatus:     complianceStatus,
		FlaggedEvents:        suspiciousLogs,
		KeyObservations:      keyObs,
		RemediationActions:   actions,
		ExecutiveSummary:     execSummary,
		AIProvider:           "Local Heuristic Engine (SOX / ISO 27001 Baseline)",
	}, nil
}

func (s *Service) callAIAnomalyAudit(allLogs []AuditLogEntry, suspicious []AuditLogEntry, totalAnalyzed int, baseScore int, baseCompliance string) (*AIAuditAnomalyReport, error) {
	type miniLog struct {
		Code   string `json:"code"`
		Model  string `json:"model"`
		Action string `json:"action"`
		User   string `json:"user"`
		Time   string `json:"time"`
		Reason string `json:"reason"`
		Score  int    `json:"score"`
	}

	var samples []miniLog
	for i, l := range suspicious {
		if i >= 10 {
			break
		}
		samples = append(samples, miniLog{
			Code:   l.LogCode,
			Model:  l.ModelName,
			Action: l.Action,
			User:   l.UserName,
			Time:   l.Timestamp,
			Reason: l.AnomalyReason,
			Score:  l.RiskScore,
		})
	}
	samplesBytes, _ := json.Marshal(samples)

	systemPrompt := "Anda adalah Lead IT Auditor & Forensic Security Specialist (CISA, CISSP) bersertifikasi SOX 404 & ISO 27001. Analisis log aktivitas ERP berikut dan berikan evaluasi forensik terstruktur dalam format JSON murni tanpa markdown pembungkus."

	userPrompt := fmt.Sprintf(`Tinjau data log audit berikut:
Total log dianalisis: %d
Jumlah anomali terdeteksi: %d
Contoh anomali terflag:
%s

Berikan JSON persis seperti format berikut:
{
  "overall_security_score": 85,
  "compliance_status": "COMPLIANT|AUDIT_WARNING|NON_COMPLIANT",
  "key_observations": ["observasi 1", "observasi 2"],
  "remediation_actions": ["remediasi 1", "remediasi 2"],
  "executive_summary": "Ringkasan eksekutif kepatuhan audit"
}`, totalAnalyzed, len(suspicious), string(samplesBytes))

	respStr, err := s.aiClient.CallRaw(userPrompt, systemPrompt)
	if err != nil {
		return nil, err
	}

	cleanJSON := strings.TrimSpace(respStr)
	if idx := strings.Index(cleanJSON, "{"); idx != -1 {
		if last := strings.LastIndex(cleanJSON, "}"); last != -1 && last > idx {
			cleanJSON = cleanJSON[idx : last+1]
		}
	}
	var aiResp struct {
		OverallSecurityScore int      `json:"overall_security_score"`
		ComplianceStatus     string   `json:"compliance_status"`
		KeyObservations      []string `json:"key_observations"`
		RemediationActions   []string `json:"remediation_actions"`
		ExecutiveSummary     string   `json:"executive_summary"`
	}

	if err := json.Unmarshal([]byte(cleanJSON), &aiResp); err != nil {
		return nil, err
	}

	score := aiResp.OverallSecurityScore
	if score <= 0 || score > 100 {
		score = baseScore
	}
	status := aiResp.ComplianceStatus
	if status == "" {
		status = baseCompliance
	}

	return &AIAuditAnomalyReport{
		TotalAnalyzed:        totalAnalyzed,
		SuspiciousCount:      len(suspicious),
		OverallSecurityScore: score,
		ComplianceStatus:     status,
		FlaggedEvents:        suspicious,
		KeyObservations:      aiResp.KeyObservations,
		RemediationActions:   aiResp.RemediationActions,
		ExecutiveSummary:     aiResp.ExecutiveSummary,
		AIProvider:           "Cloudflare Workers AI (@cf/meta/llama-3-8b-instruct)",
	}, nil
}

// ---------------------------------------------------------------------------
// CSV Export Engine
// ---------------------------------------------------------------------------

func (s *Service) ExportCSV(filters AuditLogFilters) ([]byte, error) {
	filters.Limit = 5000 // Allow up to 5,000 logs in single export
	filters.Offset = 0
	entries, _, err := s.repo.ListEntries(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch audit logs for CSV export: %w", err)
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write CSV Header
	header := []string{
		"Log Code",
		"Timestamp",
		"Model Name",
		"Resource ID",
		"Resource Name",
		"Action",
		"User Name",
		"User Email",
		"IP Address",
		"Suspicious",
		"Risk Score",
		"Summary",
		"Anomaly Reason",
		"Field Changes",
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	for _, e := range entries {
		var diffSummaries []string
		for _, d := range e.Diffs {
			diffSummaries = append(diffSummaries, fmt.Sprintf("%s: '%s' -> '%s'", d.FieldName, d.OldValue, d.NewValue))
		}
		diffStr := strings.Join(diffSummaries, " | ")

		suspiciousStr := "No"
		if e.IsSuspicious {
			suspiciousStr = "Yes"
		}

		row := []string{
			e.LogCode,
			e.Timestamp,
			e.ModelName,
			strconv.FormatUint(uint64(e.ResID), 10),
			e.ResName,
			e.Action,
			e.UserName,
			e.UserEmail,
			e.IPAddress,
			suspiciousStr,
			strconv.Itoa(e.RiskScore),
			e.Summary,
			e.AnomalyReason,
			diffStr,
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ---------------------------------------------------------------------------
// Query Helpers
// ---------------------------------------------------------------------------

func (s *Service) GetEntryByID(id uint, companyID uint) (*AuditLogEntry, error) {
	return s.repo.GetEntryByID(id, companyID)
}

func (s *Service) ListEntries(filters AuditLogFilters) ([]AuditLogEntry, int64, error) {
	return s.repo.ListEntries(filters)
}

func (s *Service) GetStats(companyID uint) (*AuditStats, error) {
	return s.repo.GetStats(companyID)
}
