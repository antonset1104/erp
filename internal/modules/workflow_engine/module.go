package workflow_engine

import (
	"log"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/chatter"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db       *gorm.DB
	service  *Service
	handler  *Handler
	aiClient *ai.Client
}

func NewModule(db *gorm.DB, chatterSvc *chatter.Service, aiClient *ai.Client) *Module {
	repo := NewRepository(db)
	service := NewService(db, repo, chatterSvc, aiClient)
	handler := NewHandler(service)

	return &Module{
		db:       db,
		service:  service,
		handler:  handler,
		aiClient: aiClient,
	}
}

func (m *Module) Name() string {
	return "workflow_engine"
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	m.handler.RegisterRoutes(api, authMiddleware)
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&WorkflowDefinition{},
		&WorkflowNode{},
		&WorkflowTransition{},
		&WorkflowInstance{},
		&WorkflowActivityLog{},
	)
}

func (m *Module) Seed(db *gorm.DB) error {
	return m.InitMasterData(db)
}

func (m *Module) InitMasterData(db *gorm.DB) error {
	var count int64
	db.Model(&WorkflowDefinition{}).Count(&count)
	if count > 0 {
		return nil
	}

	log.Println("🌱 Seeding BPMN 2.0 Dynamic Workflow & State Machine Engine Master Blueprints...")

	// 1. Tiered Purchase Order Approval Blueprint (SOX Compliant)
	poWf := WorkflowDefinition{
		Code:           "WF-PO-TIERED",
		Name:           "Pengadaan Barang & Jasa Bertingkat (Tiered Purchase Approval)",
		Description:    "Alur persetujuan PO otomatis: Nilai < 50 Juta langsung disetujui Dept Head; 50-500 Juta butuh CFO; > 500 Juta butuh Direktur Utama.",
		TargetModel:    "purchase.order",
		InitialNodeKey: "draft",
		IsActive:       true,
		Version:        1,
		Nodes: []WorkflowNode{
			{NodeKey: "draft", Name: "Draft Pesanan Pembelian", NodeType: NodeTypeStartEvent, ApproverRole: "initiator", PosX: 50, PosY: 150},
			{NodeKey: "dept_head_review", Name: "Review Kepala Departemen", NodeType: NodeTypeUserTask, ApproverRole: "dept_head", TimeoutHours: 24, PosX: 250, PosY: 150},
			{NodeKey: "amount_gateway", Name: "Gateway Pengecekan Nilai Nominal", NodeType: NodeTypeExclusiveGateway, PosX: 450, PosY: 150},
			{NodeKey: "cfo_approval", Name: "Persetujuan Direktur Keuangan (CFO)", NodeType: NodeTypeUserTask, ApproverRole: "finance_manager", TimeoutHours: 48, PosX: 650, PosY: 100},
			{NodeKey: "director_approval", Name: "Persetujuan Direktur Utama (CEO)", NodeType: NodeTypeUserTask, ApproverRole: "director", TimeoutHours: 72, PosX: 850, PosY: 100},
			{NodeKey: "approved", Name: "Pesanan Disetujui (Approved)", NodeType: NodeTypeEndEvent, PosX: 1050, PosY: 150},
			{NodeKey: "rejected", Name: "Pesanan Ditolak (Rejected)", NodeType: NodeTypeEndEvent, PosX: 650, PosY: 300},
		},
		Transitions: []WorkflowTransition{
			{FromNodeKey: "draft", ToNodeKey: "dept_head_review", ActionName: "submit", Priority: 1},
			{FromNodeKey: "dept_head_review", ToNodeKey: "amount_gateway", ActionName: "approve", Priority: 1},
			{FromNodeKey: "dept_head_review", ToNodeKey: "rejected", ActionName: "reject", Priority: 2},
			{FromNodeKey: "amount_gateway", ToNodeKey: "cfo_approval", ActionName: "auto", ConditionExpression: "amount >= 50000000", Priority: 1},
			{FromNodeKey: "amount_gateway", ToNodeKey: "approved", ActionName: "auto", ConditionExpression: "amount < 50000000", Priority: 2},
			{FromNodeKey: "cfo_approval", ToNodeKey: "director_approval", ActionName: "approve", ConditionExpression: "amount >= 500000000", Priority: 1},
			{FromNodeKey: "cfo_approval", ToNodeKey: "approved", ActionName: "approve", ConditionExpression: "amount < 500000000", Priority: 2},
			{FromNodeKey: "cfo_approval", ToNodeKey: "rejected", ActionName: "reject", Priority: 3},
			{FromNodeKey: "director_approval", ToNodeKey: "approved", ActionName: "approve", Priority: 1},
			{FromNodeKey: "director_approval", ToNodeKey: "rejected", ActionName: "reject", Priority: 2},
		},
	}
	poWf.CompanyID = 1
	for i := range poWf.Nodes {
		poWf.Nodes[i].CompanyID = 1
	}
	for i := range poWf.Transitions {
		poWf.Transitions[i].CompanyID = 1
	}
	db.Create(&poWf)

	// 2. Low-Code Construction Project Dynamic Workflow
	epcWf := WorkflowDefinition{
		Code:           "WF-EPC-CONSTRUCTION",
		Name:           "Alur Persetujuan Proyek EPC Konstruksi & Sipil",
		Description:    "Persetujuan multi-tahap proyek EPC: Verifikasi Lapangan -> Audit K3 (HSE) -> Pengesahan Project Director.",
		TargetModel:    "x_construction_epc",
		InitialNodeKey: "draft",
		IsActive:       true,
		Version:        1,
		Nodes: []WorkflowNode{
			{NodeKey: "draft", Name: "Inisiasi Proposal Proyek", NodeType: NodeTypeStartEvent, ApproverRole: "initiator", PosX: 50, PosY: 150},
			{NodeKey: "site_survey", Name: "Verifikasi Lapangan & Geoteknik", NodeType: NodeTypeUserTask, ApproverRole: "project_manager", TimeoutHours: 48, PosX: 250, PosY: 150},
			{NodeKey: "hse_audit", Name: "Audit Kepatuhan K3 & Lingkungan (HSE)", NodeType: NodeTypeUserTask, ApproverRole: "hse_officer", TimeoutHours: 24, PosX: 500, PosY: 150},
			{NodeKey: "project_director", Name: "Pengesahan Project Director", NodeType: NodeTypeUserTask, ApproverRole: "director", TimeoutHours: 72, PosX: 750, PosY: 150},
			{NodeKey: "approved", Name: "Proyek Resmi Berjalan (Active EPC)", NodeType: NodeTypeEndEvent, PosX: 1000, PosY: 150},
			{NodeKey: "rejected", Name: "Proposal Ditolak", NodeType: NodeTypeEndEvent, PosX: 500, PosY: 300},
		},
		Transitions: []WorkflowTransition{
			{FromNodeKey: "draft", ToNodeKey: "site_survey", ActionName: "submit", Priority: 1},
			{FromNodeKey: "site_survey", ToNodeKey: "hse_audit", ActionName: "approve", Priority: 1},
			{FromNodeKey: "site_survey", ToNodeKey: "rejected", ActionName: "reject", Priority: 2},
			{FromNodeKey: "hse_audit", ToNodeKey: "project_director", ActionName: "approve", Priority: 1},
			{FromNodeKey: "hse_audit", ToNodeKey: "rejected", ActionName: "reject", Priority: 2},
			{FromNodeKey: "project_director", ToNodeKey: "approved", ActionName: "approve", Priority: 1},
			{FromNodeKey: "project_director", ToNodeKey: "rejected", ActionName: "reject", Priority: 2},
		},
	}
	epcWf.CompanyID = 1
	for i := range epcWf.Nodes {
		epcWf.Nodes[i].CompanyID = 1
	}
	for i := range epcWf.Transitions {
		epcWf.Transitions[i].CompanyID = 1
	}
	db.Create(&epcWf)

	return nil
}
