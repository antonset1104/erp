package workflow_engine

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/chatter"
	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	repo       Repository
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewService(db *gorm.DB, repo Repository, chatterSvc *chatter.Service, aiClient *ai.Client) *Service {
	return &Service{
		db:         db,
		repo:       repo,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

// ---------------- Condition Evaluator ----------------

// EvaluateCondition evaluates simple BPMN conditional expressions like:
// "amount >= 50000000", "amount < 10000000", "category == 'CAPEX'", "urgency == 'HIGH'", "is_capex == true"
func EvaluateCondition(expr string, ctx map[string]interface{}) bool {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return true // Empty condition means unconditional transition
	}

	operators := []string{">=", "<=", "!=", "==", ">", "<"}
	for _, op := range operators {
		if strings.Contains(expr, op) {
			parts := strings.SplitN(expr, op, 2)
			if len(parts) != 2 {
				continue
			}
			leftKey := strings.TrimSpace(parts[0])
			rightValStr := strings.TrimSpace(parts[1])
			rightValStr = strings.Trim(rightValStr, `"'`) // Remove quotes

			rawVal, exists := ctx[leftKey]
			if !exists {
				return false
			}

			// Numeric comparison
			if rightNum, err := strconv.ParseFloat(rightValStr, 64); err == nil {
				var leftNum float64
				switch v := rawVal.(type) {
				case float64:
					leftNum = v
				case float32:
					leftNum = float64(v)
				case int:
					leftNum = float64(v)
				case int64:
					leftNum = float64(v)
				case uint:
					leftNum = float64(v)
				case string:
					leftNum, _ = strconv.ParseFloat(v, 64)
				}

				switch op {
				case ">=":
					return leftNum >= rightNum
				case "<=":
					return leftNum <= rightNum
				case ">":
					return leftNum > rightNum
				case "<":
					return leftNum < rightNum
				case "==":
					return leftNum == rightNum
				case "!=":
					return leftNum != rightNum
				}
			}

			// Boolean comparison
			if rightValStr == "true" || rightValStr == "false" {
				rightBool := rightValStr == "true"
				var leftBool bool
				switch v := rawVal.(type) {
				case bool:
					leftBool = v
				case string:
					leftBool = strings.ToLower(v) == "true"
				}
				if op == "==" {
					return leftBool == rightBool
				}
				if op == "!=" {
					return leftBool != rightBool
				}
			}

			// String comparison
			leftStr := fmt.Sprintf("%v", rawVal)
			if op == "==" {
				return strings.EqualFold(leftStr, rightValStr)
			}
			if op == "!=" {
				return !strings.EqualFold(leftStr, rightValStr)
			}
		}
	}

	return true
}

// ---------------- Definition Management ----------------

func (s *Service) CreateDefinition(companyID uint, req CreateWorkflowDefinitionRequest) (*WorkflowDefinition, error) {
	initKey := req.InitialNodeKey
	if initKey == "" {
		initKey = "draft"
	}

	wf := &WorkflowDefinition{
		Code:           req.Code,
		Name:           req.Name,
		Description:    req.Description,
		TargetModel:    req.TargetModel,
		InitialNodeKey: initKey,
		IsActive:       true,
		Version:        1,
	}
	wf.CompanyID = companyID

	// Populate Nodes
	for _, nr := range req.Nodes {
		nodeType := nr.NodeType
		if nodeType == "" {
			nodeType = NodeTypeUserTask
		}
		appRole := nr.ApproverRole
		if appRole == "" {
			appRole = "any"
		}
		timeout := nr.TimeoutHours
		if timeout <= 0 {
			timeout = 24
		}
		posX := nr.PosX
		if posX == 0 {
			posX = 100
		}
		posY := nr.PosY
		if posY == 0 {
			posY = 100
		}

		node := WorkflowNode{
			NodeKey:        nr.NodeKey,
			Name:           nr.Name,
			NodeType:       nodeType,
			ApproverRole:   appRole,
			ApproverUserID: nr.ApproverUserID,
			TimeoutHours:   timeout,
			ActionHook:     nr.ActionHook,
			PosX:           posX,
			PosY:           posY,
		}
		node.CompanyID = companyID
		wf.Nodes = append(wf.Nodes, node)
	}

	// Populate Transitions
	for _, tr := range req.Transitions {
		prio := tr.Priority
		if prio == 0 {
			prio = 10
		}
		t := WorkflowTransition{
			FromNodeKey:         tr.FromNodeKey,
			ToNodeKey:           tr.ToNodeKey,
			ActionName:          tr.ActionName,
			ConditionExpression: tr.ConditionExpression,
			Priority:            prio,
		}
		t.CompanyID = companyID
		wf.Transitions = append(wf.Transitions, t)
	}

	if err := s.repo.CreateDefinition(wf); err != nil {
		return nil, fmt.Errorf("gagal membuat definisi alur kerja: %w", err)
	}

	return wf, nil
}

func (s *Service) GetDefinitionByID(id uint) (*WorkflowDefinition, error) {
	return s.repo.GetDefinitionByID(id)
}

func (s *Service) ListDefinitions(companyID uint) ([]WorkflowDefinition, error) {
	return s.repo.ListDefinitions(companyID)
}

// ---------------- Instance Execution ----------------

func (s *Service) StartWorkflow(companyID uint, req StartWorkflowRequest, initiatorID uint, initiatorName string) (*WorkflowInstance, error) {
	var wf *WorkflowDefinition
	var err error

	if req.WorkflowCode != "" {
		wf, err = s.repo.GetDefinitionByCode(companyID, req.WorkflowCode)
	} else {
		wf, err = s.repo.GetActiveDefinitionForModel(companyID, req.TargetModel)
	}

	if err != nil || wf == nil {
		return nil, fmt.Errorf("definisi alur kerja untuk model '%s' tidak ditemukan: %w", req.TargetModel, err)
	}

	ctxBytes, _ := json.Marshal(req.ContextData)

	inst := &WorkflowInstance{
		WorkflowID:     wf.ID,
		TargetModel:    req.TargetModel,
		RecordID:       req.RecordID,
		CurrentNodeKey: wf.InitialNodeKey,
		Status:         WorkflowStatusRunning,
		InitiatorID:    initiatorID,
		InitiatorName:  initiatorName,
		ContextJSON:    string(ctxBytes),
		StartedAt:      time.Now(),
	}
	inst.CompanyID = companyID

	if err := s.repo.CreateInstance(inst); err != nil {
		return nil, fmt.Errorf("gagal memulai instansi workflow: %w", err)
	}

	// Log initial start
	actLog := &WorkflowActivityLog{
		InstanceID:  inst.ID,
		FromNodeKey: "none",
		ToNodeKey:   inst.CurrentNodeKey,
		ActionTaken: WorkflowActionSubmit,
		ActorID:     initiatorID,
		ActorName:   initiatorName,
		Comments:    "Memulai proses bisnis workflow",
		Timestamp:   time.Now(),
	}
	actLog.CompanyID = companyID
	_ = s.repo.CreateActivityLog(actLog)

	// Post message to document chatter if chatter service is available
	if s.chatterSvc != nil {
		_ = s.chatterSvc.PostMessage(&chatter.MailMessage{
			ModelName:  req.TargetModel,
			ResID:      req.RecordID,
			AuthorName: initiatorName,
			Body: fmt.Sprintf(
				"🚀 <strong>[BPMN Workflow Dimulai]</strong> Proses <em>%s (%s)</em> aktif. Dokumen berada pada node: <code>%s</code>.",
				wf.Name, wf.Code, inst.CurrentNodeKey,
			),
			MessageType: "comment",
		})
	}

	// Auto-advance if initial node has automatic transitions
	inst = s.checkAndAdvanceAutoTransitions(inst)

	return inst, nil
}

func (s *Service) ExecuteAction(companyID uint, instanceID uint, action string, actorID uint, actorName string, comments string, additionalData map[string]interface{}) (*WorkflowInstance, error) {
	inst, err := s.repo.GetInstanceByID(instanceID)
	if err != nil || inst == nil {
		return nil, fmt.Errorf("instansi alur kerja #%d tidak ditemukan", instanceID)
	}

	if inst.Status != WorkflowStatusRunning {
		return nil, fmt.Errorf("alur kerja tidak lagi aktif (status saat ini: %s)", inst.Status)
	}

	if inst.Workflow == nil {
		return nil, fmt.Errorf("definisi alur kerja terhubung tidak valid")
	}

	// Parse and merge context
	var ctx map[string]interface{}
	if err := json.Unmarshal([]byte(inst.ContextJSON), &ctx); err != nil {
		ctx = make(map[string]interface{})
	}
	for k, v := range additionalData {
		ctx[k] = v
	}

	// Find eligible transition from CurrentNodeKey matching action & conditions
	var matchedTransition *WorkflowTransition
	for _, t := range inst.Workflow.Transitions {
		if t.FromNodeKey == inst.CurrentNodeKey && strings.EqualFold(t.ActionName, action) {
			if EvaluateCondition(t.ConditionExpression, ctx) {
				matchedTransition = &t
				break
			}
		}
	}

	if matchedTransition == nil {
		return nil, fmt.Errorf("transisi '%s' tidak valid dari node '%s' dengan parameter saat ini", action, inst.CurrentNodeKey)
	}

	prevNode := inst.CurrentNodeKey
	targetNodeKey := matchedTransition.ToNodeKey

	// Identify target node type
	var targetNodeType string
	for _, n := range inst.Workflow.Nodes {
		if n.NodeKey == targetNodeKey {
			targetNodeType = n.NodeType
			break
		}
	}

	// Update instance state
	inst.CurrentNodeKey = targetNodeKey
	now := time.Now()

	if targetNodeType == NodeTypeEndEvent || action == WorkflowActionReject {
		if action == WorkflowActionReject {
			inst.Status = WorkflowStatusRejected
		} else {
			inst.Status = WorkflowStatusCompleted
		}
		inst.CompletedAt = &now
	}

	// Update context with any additions
	updatedCtxBytes, _ := json.Marshal(ctx)
	inst.ContextJSON = string(updatedCtxBytes)

	if err := s.repo.UpdateInstance(inst); err != nil {
		return nil, fmt.Errorf("gagal memperbarui instansi workflow: %w", err)
	}

	// Log activity
	duration := int(now.Sub(inst.StartedAt).Seconds())
	actLog := &WorkflowActivityLog{
		InstanceID:               inst.ID,
		FromNodeKey:              prevNode,
		ToNodeKey:                targetNodeKey,
		ActionTaken:              action,
		ActorID:                  actorID,
		ActorName:                actorName,
		Comments:                 comments,
		ExecutionDurationSeconds: duration,
		Timestamp:                now,
	}
	actLog.CompanyID = companyID
	_ = s.repo.CreateActivityLog(actLog)

	// Post message to document chatter
	if s.chatterSvc != nil {
		statusLabel := "Node Selanjutnya: <code>" + targetNodeKey + "</code>"
		if inst.Status == WorkflowStatusCompleted {
			statusLabel = "<span style='color:#16a34a; font-weight:bold;'>SELESAI (APPROVED)</span>"
		} else if inst.Status == WorkflowStatusRejected {
			statusLabel = "<span style='color:#dc2626; font-weight:bold;'>DITOLAK (REJECTED)</span>"
		}

		body := fmt.Sprintf(
			"⚡ <strong>[BPMN Workflow Action: %s]</strong> Oleh: <em>%s</em><br/>%s &rarr; %s<br/>Catatan: <em>%s</em>",
			strings.ToUpper(action), actorName, prevNode, statusLabel, comments,
		)
		_ = s.chatterSvc.PostMessage(&chatter.MailMessage{
			ModelName:   inst.TargetModel,
			ResID:       inst.RecordID,
			AuthorName:  actorName,
			Body:        body,
			MessageType: "comment",
		})
	}

	// Auto-advance if target node has automatic transitions
	inst = s.checkAndAdvanceAutoTransitions(inst)

	return inst, nil
}

func (s *Service) checkAndAdvanceAutoTransitions(inst *WorkflowInstance) *WorkflowInstance {
	if inst.Status != WorkflowStatusRunning || inst.Workflow == nil {
		return inst
	}

	var ctx map[string]interface{}
	_ = json.Unmarshal([]byte(inst.ContextJSON), &ctx)

	for _, t := range inst.Workflow.Transitions {
		if t.FromNodeKey == inst.CurrentNodeKey && strings.EqualFold(t.ActionName, WorkflowActionAuto) {
			if EvaluateCondition(t.ConditionExpression, ctx) {
				advancedInst, err := s.ExecuteAction(inst.CompanyID, inst.ID, WorkflowActionAuto, 1, "Sistem BPMN Automator", "Transisi gateway otomatis", nil)
				if err == nil && advancedInst != nil {
					return advancedInst
				}
				break
			}
		}
	}
	return inst
}

func (s *Service) GetInstanceByID(id uint) (*WorkflowInstance, error) {
	return s.repo.GetInstanceByID(id)
}

func (s *Service) GetInstanceByRecord(companyID uint, targetModel string, recordID uint) (*WorkflowInstance, error) {
	return s.repo.GetInstanceByRecord(companyID, targetModel, recordID)
}

func (s *Service) ListPendingTasks(companyID uint, role string, userID uint) ([]WorkflowInstance, error) {
	return s.repo.ListPendingTasks(companyID, role, userID)
}

func (s *Service) GetActivityLogs(instanceID uint) ([]WorkflowActivityLog, error) {
	return s.repo.GetActivityLogs(instanceID)
}

func (s *Service) GetStats(companyID uint) (*WorkflowEngineStats, error) {
	return s.repo.GetStats(companyID)
}

// ---------------- AI Workflow Copilot ----------------

func (s *Service) AIAnalyzeWorkflow(req AIWorkflowDesignRequest) (*AIWorkflowReport, error) {
	scenario := strings.ToLower(req.BusinessScenario)
	targetModel := req.TargetModel
	if targetModel == "" {
		targetModel = "purchase.order"
	}

	// Fallback Smart Heuristic Engine for BPMN 2.0 Architectural Design
	report := &AIWorkflowReport{
		ScenarioName:        req.BusinessScenario,
		TargetModel:         targetModel,
		ComplianceGuideline: "ISO 9001:2015 Clause 8.4 (Control of Externally Provided Processes) & Sarbanes-Oxley (SOX) Section 404 Segregation of Duties",
		EfficiencyRating:    "OPTIMAL (Lean 3-Stage Gateway)",
		Provider:            "Local Smart Heuristic Engine (BPMN 2.0 Enterprise Designer)",
		OptimizationTips: []string{
			"Gunakan Exclusive Gateway (XOR) untuk membedakan jalur cepat (Fast-Track) nominal rendah vs verifikasi ketat nominal tinggi",
			"Terapkan batas waktu SLA (Timeout) maksimal 24 jam untuk mencegah terjadinya bottleneck antrean persetujuan",
			"Sertakan Service Task otomatis untuk integrasi general ledger akuntansi dan webhook notifikasi ke mitra",
		},
	}

	if strings.Contains(scenario, "pengadaan") || strings.Contains(scenario, "purchase") || strings.Contains(scenario, "milyar") || strings.Contains(scenario, "high") {
		report.RecommendedNodes = []CreateWorkflowNodeRequest{
			{NodeKey: "draft", Name: "Draft Pengadaan", NodeType: NodeTypeStartEvent, ApproverRole: "initiator", PosX: 50, PosY: 150},
			{NodeKey: "dept_head_review", Name: "Review Kepala Departemen", NodeType: NodeTypeUserTask, ApproverRole: "dept_head", TimeoutHours: 24, PosX: 250, PosY: 150},
			{NodeKey: "amount_gateway", Name: "Pengecekan Nilai Anggaran", NodeType: NodeTypeExclusiveGateway, PosX: 450, PosY: 150},
			{NodeKey: "cfo_approval", Name: "Persetujuan Direktur Keuangan (CFO)", NodeType: NodeTypeUserTask, ApproverRole: "finance_manager", TimeoutHours: 48, PosX: 650, PosY: 100},
			{NodeKey: "bod_approval", Name: "Persetujuan Direktur Utama (CEO)", NodeType: NodeTypeUserTask, ApproverRole: "director", TimeoutHours: 72, PosX: 850, PosY: 100},
			{NodeKey: "approved", Name: "Disetujui & Siap Terbit", NodeType: NodeTypeEndEvent, PosX: 1050, PosY: 150},
			{NodeKey: "rejected", Name: "Ditolak / Pembatalan", NodeType: NodeTypeEndEvent, PosX: 650, PosY: 300},
		}

		report.RecommendedEdges = []CreateWorkflowTransitionRequest{
			{FromNodeKey: "draft", ToNodeKey: "dept_head_review", ActionName: "submit", Priority: 1},
			{FromNodeKey: "dept_head_review", ToNodeKey: "amount_gateway", ActionName: "approve", Priority: 1},
			{FromNodeKey: "dept_head_review", ToNodeKey: "rejected", ActionName: "reject", Priority: 2},
			{FromNodeKey: "amount_gateway", ToNodeKey: "cfo_approval", ActionName: "auto", ConditionExpression: "amount >= 50000000", Priority: 1},
			{FromNodeKey: "amount_gateway", ToNodeKey: "approved", ActionName: "auto", ConditionExpression: "amount < 50000000", Priority: 2},
			{FromNodeKey: "cfo_approval", ToNodeKey: "bod_approval", ActionName: "approve", ConditionExpression: "amount >= 500000000", Priority: 1},
			{FromNodeKey: "cfo_approval", ToNodeKey: "approved", ActionName: "approve", ConditionExpression: "amount < 500000000", Priority: 2},
			{FromNodeKey: "cfo_approval", ToNodeKey: "rejected", ActionName: "reject", Priority: 3},
			{FromNodeKey: "bod_approval", ToNodeKey: "approved", ActionName: "approve", Priority: 1},
			{FromNodeKey: "bod_approval", ToNodeKey: "rejected", ActionName: "reject", Priority: 2},
		}
	} else {
		// Generic lean approval workflow
		report.RecommendedNodes = []CreateWorkflowNodeRequest{
			{NodeKey: "draft", Name: "Draft Dokumen", NodeType: NodeTypeStartEvent, ApproverRole: "initiator", PosX: 100, PosY: 150},
			{NodeKey: "manager_review", Name: "Review Manager", NodeType: NodeTypeUserTask, ApproverRole: "manager", TimeoutHours: 24, PosX: 300, PosY: 150},
			{NodeKey: "director_approval", Name: "Persetujuan Direksi", NodeType: NodeTypeUserTask, ApproverRole: "director", TimeoutHours: 48, PosX: 550, PosY: 150},
			{NodeKey: "approved", Name: "Selesai (Approved)", NodeType: NodeTypeEndEvent, PosX: 800, PosY: 150},
			{NodeKey: "rejected", Name: "Ditolak (Rejected)", NodeType: NodeTypeEndEvent, PosX: 425, PosY: 300},
		}
		report.RecommendedEdges = []CreateWorkflowTransitionRequest{
			{FromNodeKey: "draft", ToNodeKey: "manager_review", ActionName: "submit", Priority: 1},
			{FromNodeKey: "manager_review", ToNodeKey: "director_approval", ActionName: "approve", Priority: 1},
			{FromNodeKey: "manager_review", ToNodeKey: "rejected", ActionName: "reject", Priority: 2},
			{FromNodeKey: "director_approval", ToNodeKey: "approved", ActionName: "approve", Priority: 1},
			{FromNodeKey: "director_approval", ToNodeKey: "rejected", ActionName: "reject", Priority: 2},
		}
	}

	return report, nil
}
