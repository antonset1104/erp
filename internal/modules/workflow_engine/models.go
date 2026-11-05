package workflow_engine

import (
	"time"

	"erp-golang/internal/core/models"
)

// BPMN Node Types
const (
	NodeTypeStartEvent       = "start_event"
	NodeTypeUserTask         = "user_task"
	NodeTypeExclusiveGateway = "exclusive_gateway"
	NodeTypeParallelGateway  = "parallel_gateway"
	NodeTypeServiceTask      = "service_task"
	NodeTypeEndEvent         = "end_event"
)

// Workflow Instance Status
const (
	WorkflowStatusRunning   = "running"
	WorkflowStatusCompleted = "completed"
	WorkflowStatusRejected  = "rejected"
	WorkflowStatusCancelled = "cancelled"
)

// Workflow Actions
const (
	WorkflowActionSubmit  = "submit"
	WorkflowActionApprove = "approve"
	WorkflowActionReject  = "reject"
	WorkflowActionRework  = "rework"
	WorkflowActionAuto    = "auto"
)

// WorkflowDefinition represents a BPMN 2.0 executable business process model
type WorkflowDefinition struct {
	models.ModelBase
	Code           string               `gorm:"size:64;uniqueIndex;not null" json:"code"` // e.g. WF-PO-TIERED
	Name           string               `gorm:"size:255;not null" json:"name"`
	Description    string               `gorm:"type:text" json:"description"`
	TargetModel    string               `gorm:"size:128;index;not null" json:"target_model"` // e.g. purchase.order, sale.order, hr.expense, x_construction_epc
	InitialNodeKey string               `gorm:"size:64;not null;default:'draft'" json:"initial_node_key"`
	IsActive       bool                 `gorm:"default:true;index" json:"is_active"`
	Version        int                  `gorm:"default:1" json:"version"`
	Nodes          []WorkflowNode       `gorm:"foreignKey:WorkflowID;constraint:OnDelete:CASCADE" json:"nodes,omitempty"`
	Transitions    []WorkflowTransition `gorm:"foreignKey:WorkflowID;constraint:OnDelete:CASCADE" json:"transitions,omitempty"`
}

func (WorkflowDefinition) TableName() string {
	return "workflow_definitions"
}

// WorkflowNode represents a BPMN step/activity/gateway in the process
type WorkflowNode struct {
	models.ModelBase
	WorkflowID     uint   `gorm:"index;not null" json:"workflow_id"`
	NodeKey        string `gorm:"size:64;not null;index" json:"node_key"` // e.g. draft, dept_head_review, cfo_approval, done
	Name           string `gorm:"size:128;not null" json:"name"`
	NodeType       string `gorm:"size:32;not null;default:'user_task'" json:"node_type"` // start_event, user_task, exclusive_gateway, service_task, end_event
	ApproverRole   string `gorm:"size:64;default:'any'" json:"approver_role"`            // dept_head, finance_manager, director, admin, any
	ApproverUserID *uint  `gorm:"index" json:"approver_user_id,omitempty"`
	TimeoutHours   int    `gorm:"default:24" json:"timeout_hours"`
	ActionHook     string `gorm:"size:128" json:"action_hook,omitempty"` // e.g. post_gl, send_email, auto_validate
	PosX           int    `gorm:"default:100" json:"pos_x"`              // Visual Canvas coordinate X
	PosY           int    `gorm:"default:100" json:"pos_y"`              // Visual Canvas coordinate Y
}

func (WorkflowNode) TableName() string {
	return "workflow_nodes"
}

// WorkflowTransition represents a directed edge / sequence flow between two BPMN nodes
type WorkflowTransition struct {
	models.ModelBase
	WorkflowID          uint   `gorm:"index;not null" json:"workflow_id"`
	FromNodeKey         string `gorm:"size:64;not null;index" json:"from_node_key"`
	ToNodeKey           string `gorm:"size:64;not null;index" json:"to_node_key"`
	ActionName          string `gorm:"size:64;not null;default:'approve'" json:"action_name"` // submit, approve, reject, rework, auto
	ConditionExpression string `gorm:"size:255" json:"condition_expression,omitempty"`        // e.g. amount >= 50000000, is_capex == true
	Priority            int    `gorm:"default:10" json:"priority"`                            // Evaluated in ascending priority order
}

func (WorkflowTransition) TableName() string {
	return "workflow_transitions"
}

// WorkflowInstance tracks the active runtime execution of a workflow on a business document
type WorkflowInstance struct {
	models.ModelBase
	WorkflowID     uint                  `gorm:"index;not null" json:"workflow_id"`
	Workflow       *WorkflowDefinition   `gorm:"foreignKey:WorkflowID" json:"workflow,omitempty"`
	TargetModel    string                `gorm:"size:128;index;not null" json:"target_model"`
	RecordID       uint                  `gorm:"index;not null" json:"record_id"`
	CurrentNodeKey string                `gorm:"size:64;not null;index" json:"current_node_key"`
	Status         string                `gorm:"size:32;default:'running';index" json:"status"` // running, completed, rejected, cancelled
	InitiatorID    uint                  `gorm:"index;not null" json:"initiator_id"`
	InitiatorName  string                `gorm:"size:128" json:"initiator_name"`
	ContextJSON    string                `gorm:"type:text" json:"context_json"` // JSON map of evaluated attributes (amount, requester, tags, etc.)
	StartedAt      time.Time             `json:"started_at"`
	CompletedAt    *time.Time            `json:"completed_at,omitempty"`
	Logs           []WorkflowActivityLog `gorm:"foreignKey:InstanceID;constraint:OnDelete:CASCADE" json:"logs,omitempty"`
}

func (WorkflowInstance) TableName() string {
	return "workflow_instances"
}

// WorkflowActivityLog logs each discrete transition, approval, rejection, or automated step
type WorkflowActivityLog struct {
	models.ModelBase
	InstanceID               uint      `gorm:"index;not null" json:"instance_id"`
	FromNodeKey              string    `gorm:"size:64;not null" json:"from_node_key"`
	ToNodeKey                string    `gorm:"size:64;not null" json:"to_node_key"`
	ActionTaken              string    `gorm:"size:64;not null" json:"action_taken"` // submit, approve, reject, auto
	ActorID                  uint      `gorm:"index;not null" json:"actor_id"`
	ActorName                string    `gorm:"size:128;not null" json:"actor_name"`
	Comments                 string    `gorm:"type:text" json:"comments,omitempty"`
	ExecutionDurationSeconds int       `gorm:"default:0" json:"execution_duration_seconds"`
	Timestamp                time.Time `json:"timestamp"`
}

func (WorkflowActivityLog) TableName() string {
	return "workflow_activity_logs"
}

// --- DTO Requests & Responses ---

type CreateWorkflowDefinitionRequest struct {
	Code           string                            `json:"code" binding:"required"`
	Name           string                            `json:"name" binding:"required"`
	Description    string                            `json:"description"`
	TargetModel    string                            `json:"target_model" binding:"required"`
	InitialNodeKey string                            `json:"initial_node_key"`
	Nodes          []CreateWorkflowNodeRequest       `json:"nodes"`
	Transitions    []CreateWorkflowTransitionRequest `json:"transitions"`
}

type CreateWorkflowNodeRequest struct {
	NodeKey        string `json:"node_key" binding:"required"`
	Name           string `json:"name" binding:"required"`
	NodeType       string `json:"node_type"`
	ApproverRole   string `json:"approver_role"`
	ApproverUserID *uint  `json:"approver_user_id,omitempty"`
	TimeoutHours   int    `json:"timeout_hours"`
	ActionHook     string `json:"action_hook,omitempty"`
	PosX           int    `json:"pos_x"`
	PosY           int    `json:"pos_y"`
}

type CreateWorkflowTransitionRequest struct {
	FromNodeKey         string `json:"from_node_key" binding:"required"`
	ToNodeKey           string `json:"to_node_key" binding:"required"`
	ActionName          string `json:"action_name" binding:"required"`
	ConditionExpression string `json:"condition_expression,omitempty"`
	Priority            int    `json:"priority"`
}

type StartWorkflowRequest struct {
	WorkflowCode string                 `json:"workflow_code,omitempty"`
	TargetModel  string                 `json:"target_model" binding:"required"`
	RecordID     uint                   `json:"record_id" binding:"required"`
	ContextData  map[string]interface{} `json:"context_data"`
}

type ExecuteWorkflowActionRequest struct {
	Action         string                 `json:"action" binding:"required"` // approve, reject, rework, submit
	Comments       string                 `json:"comments"`
	AdditionalData map[string]interface{} `json:"additional_data,omitempty"`
}

type WorkflowEngineStats struct {
	TotalDefinitions   int64   `json:"total_definitions"`
	ActiveWorkflows    int64   `json:"active_workflows"`
	RunningInstances   int64   `json:"running_instances"`
	CompletedInstances int64   `json:"completed_instances"`
	RejectedInstances  int64   `json:"rejected_instances"`
	PendingApprovals   int64   `json:"pending_approvals"`
	AvgCycleHours      float64 `json:"avg_cycle_hours"`
}

type AIWorkflowDesignRequest struct {
	BusinessScenario string `json:"business_scenario" binding:"required"` // e.g. "Pengadaan barang proyek bernilai tinggi > 1 Milyar dengan audit K3"
	TargetModel      string `json:"target_model"`
}

type AIWorkflowReport struct {
	ScenarioName        string                            `json:"scenario_name"`
	TargetModel         string                            `json:"target_model"`
	RecommendedNodes    []CreateWorkflowNodeRequest       `json:"recommended_nodes"`
	RecommendedEdges    []CreateWorkflowTransitionRequest `json:"recommended_edges"`
	ComplianceGuideline string                            `json:"compliance_guideline"`
	EfficiencyRating    string                            `json:"efficiency_rating"`
	OptimizationTips    []string                          `json:"optimization_tips"`
	Provider            string                            `json:"provider"`
}
