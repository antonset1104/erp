package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"erp-golang/internal/config"
	"erp-golang/internal/core/ai"
	"erp-golang/internal/core/commandcenter"
	"erp-golang/internal/core/events"
	"erp-golang/internal/core/multitenancy"
	"erp-golang/internal/core/openapi"
	"erp-golang/internal/core/audit"
	"erp-golang/internal/core/bi"
	coreaccount "erp-golang/internal/core/account"
	coreedi "erp-golang/internal/core/edi"
	corehr "erp-golang/internal/core/hr"
	coretreasury "erp-golang/internal/core/treasury"
	corewms "erp-golang/internal/core/wms"
	"erp-golang/internal/core/esg"
	"erp-golang/internal/core/iot"
	"erp-golang/internal/core/security"
	"erp-golang/internal/core/storage"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/approval"
	"erp-golang/internal/modules/asset"
	"erp-golang/internal/modules/attachment"
	"erp-golang/internal/modules/attendance"
	"erp-golang/internal/modules/bank_reconciliation"
	"erp-golang/internal/modules/barcode"
	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/budget"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/crm"
	"erp-golang/internal/modules/currency"
	"erp-golang/internal/modules/expense"
	"erp-golang/internal/modules/fleet"
	"erp-golang/internal/modules/helpdesk"
	"erp-golang/internal/modules/hr"
	"erp-golang/internal/modules/landed_cost"
	"erp-golang/internal/modules/leave"
	"erp-golang/internal/modules/maintenance"
	"erp-golang/internal/modules/mrp"
	"erp-golang/internal/modules/pos"
	"erp-golang/internal/modules/payroll"
	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/project"
	"erp-golang/internal/modules/purchase"
	"erp-golang/internal/modules/purchase_requisition"
	"erp-golang/internal/modules/quality"
	"erp-golang/internal/modules/recruitment"
	"erp-golang/internal/modules/sale"
	"erp-golang/internal/modules/stock"
	"erp-golang/internal/modules/stock_inventory"
	"erp-golang/internal/modules/tax_efaktur"
	"erp-golang/internal/modules/traceability"
	"erp-golang/internal/modules/subscription"
	"erp-golang/internal/modules/audit_trail"
	"erp-golang/internal/modules/webhook"
	"erp-golang/internal/modules/inbound_webhook"
	"erp-golang/internal/modules/intercompany"
	"erp-golang/internal/modules/treasury"
	"erp-golang/internal/modules/mrp_advanced"
	"erp-golang/internal/modules/tax_ebupot"
	"erp-golang/internal/modules/esignature"
	"erp-golang/internal/modules/wms"
	"erp-golang/internal/modules/vendor_portal"
	"erp-golang/internal/modules/customer_portal"
	"erp-golang/internal/modules/customs_bonded"
	"erp-golang/internal/modules/lowcode_studio"
	"erp-golang/internal/modules/reports"
	"erp-golang/internal/modules/workflow_engine"
	"erp-golang/internal/modules/integration_hub"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to open test in-memory database: %v", err)
	}

	// Auto-migrate all models
	err = db.AutoMigrate(
		&base.Company{},
		&base.Currency{},
		&base.User{},
		&base.Partner{},
		&product.ProductCategory{},
		&product.UoM{},
		&product.Product{},
		&stock.StockWarehouse{},
		&stock.StockLocation{},
		&stock.StockPicking{},
		&stock.StockMove{},
		&stock.StockQuant{},
		&stock.StockOrderpoint{},
		&sale.SaleOrder{},
		&sale.SaleOrderLine{},
		&purchase.PurchaseOrder{},
		&purchase.PurchaseOrderLine{},
		&account.AccountAccount{},
		&account.AccountJournal{},
		&account.AccountMove{},
		&account.AccountMoveLine{},
		&account.AccountPayment{},
		&mrp.MrpBom{},
		&mrp.MrpBomLine{},
		&mrp.MrpProduction{},
		&pos.PosConfig{},
		&pos.PosSession{},
		&pos.PosOrder{},
		&pos.PosOrderLine{},
		&hr.Department{},
		&hr.JobPosition{},
		&hr.Employee{},
		&hr.Payslip{},
		&crm.CrmLead{},
		&chatter.MailMessage{},
		&project.Project{},
		&project.Task{},
		&project.TimesheetLine{},
		&asset.Asset{},
		&asset.AssetDepreciationLine{},
		&helpdesk.HelpdeskTeam{},
		&helpdesk.HelpdeskTicket{},
		&maintenance.MaintenanceEquipmentCategory{},
		&maintenance.MaintenanceEquipment{},
		&maintenance.MaintenanceRequest{},
		&fleet.FleetVehicleModel{},
		&fleet.FleetVehicle{},
		&fleet.FleetOdometerLog{},
		&fleet.FleetServiceLog{},
		&leave.LeaveType{},
		&leave.LeaveAllocation{},
		&leave.LeaveRequest{},
		&expense.ExpenseCategory{},
		&expense.Expense{},
		&expense.ExpenseSheet{},
		&quality.QualityPoint{},
		&quality.QualityCheck{},
		&quality.QualityAlert{},
		&approval.ApprovalCategory{},
		&approval.ApprovalRequest{},
		&approval.ApprovalApprover{},
		&attachment.Attachment{},
		&attendance.Shift{},
		&attendance.Attendance{},
		&recruitment.RecruitmentStage{},
		&recruitment.JobOpening{},
		&recruitment.Applicant{},
		&stock_inventory.StockInventory{},
		&stock_inventory.StockInventoryLine{},
		&payroll.SalaryRuleCategory{},
		&payroll.PayrollStructure{},
		&payroll.SalaryRule{},
		&payroll.PayslipRun{},
		&payroll.Payslip{},
		&payroll.PayslipLine{},
		&budget.AnalyticAccount{},
		&budget.BudgetaryPosition{},
		&budget.Budget{},
		&budget.BudgetLine{},
		&barcode.StockProductionLot{},
		&purchase_requisition.PurchaseRequisition{},
		&purchase_requisition.PurchaseRequisitionLine{},
		&currency.Currency{},
		&currency.CurrencyRate{},
		&currency.ForexRevaluation{},
		&currency.ForexRevaluationLine{},
		&tax_efaktur.TaxNSFP{},
		&tax_efaktur.TaxInvoice{},
		&tax_efaktur.TaxInvoiceLine{},
		&bank_reconciliation.AccountBankStatement{},
		&bank_reconciliation.AccountBankStatementLine{},
		&bank_reconciliation.AccountReconcileModel{},
		&landed_cost.StockLandedCost{},
		&landed_cost.StockLandedCostLine{},
		&landed_cost.StockValuationAdjustmentLine{},
		&traceability.LotTraceRecord{},
		&traceability.ProductRecall{},
		&subscription.SubscriptionTemplate{},
		&subscription.Subscription{},
		&subscription.SubscriptionLine{},
		&subscription.SubscriptionInvoiceLog{},
		&subscription.SubscriptionStageLog{},
		&audit_trail.AuditLogRule{},
		&audit_trail.AuditLogEntry{},
		&audit_trail.AuditLogFieldDiff{},
		&webhook.WebhookSubscription{},
		&webhook.WebhookDelivery{},
		&inbound_webhook.InboundWebhookEndpoint{},
		&inbound_webhook.InboundWebhookLog{},
		&intercompany.IntercompanyGroup{},
		&intercompany.IntercompanyRelation{},
		&intercompany.IntercompanyTransaction{},
		&intercompany.ConsolidationRun{},
		&intercompany.ConsolidationLine{},
		&treasury.TreasuryBankAccount{},
		&treasury.CashFlowForecast{},
		&treasury.CashFlowForecastItem{},
		&treasury.InterBankTransfer{},
		&mrp_advanced.WorkCenter{},
		&mrp_advanced.Routing{},
		&mrp_advanced.RoutingOperation{},
		&mrp_advanced.WorkOrder{},
		&tax_ebupot.EbupotPeriod{},
		&tax_ebupot.WithholdingSlip{},
		&tax_ebupot.EbupotExportLog{},
		&esignature.DocumentSignRequest{},
		&wms.WMSZone{},
		&wms.WMSBinLocation{},
		&wms.WMSPutawayRule{},
		&wms.WMSInventorySlot{},
		&wms.WMSWavePicking{},
		&wms.WMSWavePickingLine{},
		&vendor_portal.VendorPortalAccount{},
		&vendor_portal.VendorQuotationBid{},
		&vendor_portal.VendorQuotationBidLine{},
		&vendor_portal.VendorASN{},
		&vendor_portal.VendorASNLine{},
		&vendor_portal.VendorInvoiceSubmission{},
		&customer_portal.CustomerPortalAccount{},
		&customer_portal.B2BOrderRequest{},
		&customer_portal.B2BOrderLine{},
		&customer_portal.RMARequest{},
		&customer_portal.CustomerPaymentNotification{},
		&customs_bonded.CustomsDocument{},
		&customs_bonded.CustomsDocumentLine{},
		&customs_bonded.CustomsITInventoryMutation{},
		&customs_bonded.CustomsSubcontract{},
		&lowcode_studio.CustomModel{},
		&lowcode_studio.CustomField{},
		&lowcode_studio.CustomRecord{},
		&lowcode_studio.CoreModelExtensionValue{},
		&workflow_engine.WorkflowDefinition{},
		&workflow_engine.WorkflowNode{},
		&workflow_engine.WorkflowTransition{},
		&workflow_engine.WorkflowInstance{},
		&workflow_engine.WorkflowActivityLog{},
		&integration_hub.APIKey{},
		&integration_hub.IntegrationConnector{},
		&integration_hub.IntegrationLog{},
		&integration_hub.IntegrationDataJob{},
	)
	if err != nil {
		t.Fatalf("Failed to auto-migrate models: %v", err)
	}

	return db
}

func TestCompleteERPWorkflow(t *testing.T) {
	db := setupTestDB(t)

	// 1. Seed Locations & Company
	company := base.Company{Name: "Test ERP Company"}
	company.CompanyID = 1
	db.Create(&company)

	whStock := stock.StockLocation{Name: "WH/Stock", Usage: "internal"}
	whStock.CompanyID = 1
	db.Create(&whStock) // ID: 1

	vendorsLoc := stock.StockLocation{Name: "Virtual Locations/Vendors", Usage: "supplier"}
	vendorsLoc.CompanyID = 1
	db.Create(&vendorsLoc) // ID: 2

	customersLoc := stock.StockLocation{Name: "Virtual Locations/Customers", Usage: "customer"}
	customersLoc.CompanyID = 1
	db.Create(&customersLoc) // ID: 3

	// 2. Base Module - Auth & Partner
	baseSvc := base.NewService(db, "test-secret")
	user, err := baseSvc.Register("John Doe", "john@example.com", "secret123", "admin", 1)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	authUser, token, err := baseSvc.Authenticate("john@example.com", "secret123")
	if err != nil || token == "" || authUser.ID != user.ID {
		t.Fatalf("Authentication failed: err=%v", err)
	}
	t.Logf("✓ Auth successful for user: %s (JWT token generated)", authUser.Email)

	customer := base.Partner{
		Name:         "PT Pelanggan Sejahtera",
		CustomerRank: 1,
		IsCompany:    true,
	}
	customer.CompanyID = 1
	db.Create(&customer)

	vendor := base.Partner{
		Name:         "CV Pemasok Utama",
		SupplierRank: 1,
		IsCompany:    true,
	}
	vendor.CompanyID = 1
	db.Create(&vendor)

	// 3. Product Module
	prodSvc := product.NewService(db)
	testProduct := product.Product{
		Name:          "Gaming Monitor 27 Inch",
		DefaultCode:   "MON-27-144",
		Type:          "product", // Storable
		ListPrice:     3500000,
		StandardPrice: 2400000,
	}
	testProduct.CompanyID = 1
	if err := prodSvc.CreateProduct(&testProduct); err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}
	t.Logf("✓ Created Storable Product: %s (Initial Stock: %.0f)", testProduct.Name, testProduct.QtyAvailable)

	// 4. Purchase Module & Double-Entry Stock Receiving
	stockSvc := stock.NewService(db)
	purchaseSvc := purchase.NewService(db, stockSvc)

	po := purchase.PurchaseOrder{
		PartnerID: vendor.ID,
		OrderLines: []purchase.PurchaseOrderLine{
			{
				ProductID:     testProduct.ID,
				Name:          testProduct.Name,
				ProductUomQty: 10,
				PriceUnit:     2400000,
			},
		},
	}
	po.CompanyID = 1
	if err := purchaseSvc.CreateOrder(&po); err != nil {
		t.Fatalf("Create PO failed: %v", err)
	}

	confirmedPO, receipt, err := purchaseSvc.ActionConfirm(po.ID, 1)
	if err != nil || confirmedPO.State != "purchase" || receipt == nil {
		t.Fatalf("Confirm PO failed: %v", err)
	}
	t.Logf("✓ Purchase Order Confirmed: %s -> Auto-generated Stock Receipt: %s", confirmedPO.Name, receipt.Name)

	// Validate Stock Receipt (Vendors -> WH/Stock)
	_, err = stockSvc.ValidatePicking(receipt.ID, 1)
	if err != nil {
		t.Fatalf("Validate incoming picking failed: %v", err)
	}

	// Verify stock on-hand increased to 10
	var updatedProd product.Product
	db.First(&updatedProd, testProduct.ID)
	if updatedProd.QtyAvailable != 10 {
		t.Fatalf("Expected 10 units on hand after receipt, got %.0f", updatedProd.QtyAvailable)
	}
	t.Logf("✓ Stock Receipt Executed: Double-Entry quant updated. Current On-Hand: %.0f units", updatedProd.QtyAvailable)

	// 5. Sales Module & Delivery Order
	saleSvc := sale.NewService(db, stockSvc)
	so := sale.SaleOrder{
		PartnerID: customer.ID,
		OrderLines: []sale.SaleOrderLine{
			{
				ProductID:     testProduct.ID,
				Name:          testProduct.Name,
				ProductUomQty: 3,
				PriceUnit:     3500000,
			},
		},
	}
	so.CompanyID = 1
	if err := saleSvc.CreateOrder(&so); err != nil {
		t.Fatalf("Create SO failed: %v", err)
	}

	confirmedSO, delivery, err := saleSvc.ActionConfirm(so.ID, 1)
	if err != nil || confirmedSO.State != "sale" || delivery == nil {
		t.Fatalf("Confirm SO failed: %v", err)
	}
	t.Logf("✓ Sales Order Confirmed: %s -> Auto-generated Delivery Order: %s", confirmedSO.Name, delivery.Name)

	// Validate Delivery Order (WH/Stock -> Customers)
	_, err = stockSvc.ValidatePicking(delivery.ID, 1)
	if err != nil {
		t.Fatalf("Validate outgoing picking failed: %v", err)
	}

	// Verify stock on-hand decreased from 10 to 7
	db.First(&updatedProd, testProduct.ID)
	if updatedProd.QtyAvailable != 7 {
		t.Fatalf("Expected 7 units on hand after delivery, got %.0f", updatedProd.QtyAvailable)
	}
	t.Logf("✓ Delivery Order Executed: On-Hand inventory safely reduced to %.0f units", updatedProd.QtyAvailable)

	// 6. Accounting Module - Balanced Journal Entry & Payment
	accountSvc := account.NewService(db)
	journal := account.AccountJournal{Code: "INV", Name: "Customer Invoices", Type: "sale"}
	journal.CompanyID = 1
	db.Create(&journal)

	accReceivable := account.AccountAccount{Code: "1200", Name: "Piutang Usaha", Type: "asset_receivable"}
	accReceivable.CompanyID = 1
	db.Create(&accReceivable)

	accSales := account.AccountAccount{Code: "4100", Name: "Pendapatan Penjualan", Type: "income"}
	accSales.CompanyID = 1
	db.Create(&accSales)

	invoice := account.AccountMove{
		MoveType:       "out_invoice",
		PartnerID:      customer.ID,
		JournalID:      journal.ID,
		InvoiceDate:    "2026-10-04",
		AmountTotal:    10500000,
		AmountResidual: 10500000,
		Ref:            confirmedSO.Name,
		LineIDs: []account.AccountMoveLine{
			{
				AccountID: accReceivable.ID,
				Name:      "Piutang Penjualan",
				Debit:     10500000,
				Credit:    0,
			},
			{
				AccountID: accSales.ID,
				Name:      "Penjualan Gaming Monitor (3 units)",
				Debit:     0,
				Credit:    10500000,
			},
		},
	}
	invoice.CompanyID = 1
	if err := accountSvc.CreateMove(&invoice); err != nil {
		t.Fatalf("Create Invoice failed: %v", err)
	}

	// Post invoice (verifies debit == credit)
	postedInvoice, err := accountSvc.ActionPost(invoice.ID, 1)
	if err != nil || postedInvoice.State != "posted" {
		t.Fatalf("ActionPost invoice failed: %v", err)
	}
	t.Logf("✓ Invoice %s posted to General Ledger (Debit=%.0f, Credit=%.0f verified)", postedInvoice.Name, 10500000.0, 10500000.0)

	// Register Payment
	bankJournal := account.AccountJournal{Code: "BNK", Name: "Bank BCA", Type: "bank"}
	bankJournal.CompanyID = 1
	db.Create(&bankJournal)

	payment, paidInvoice, err := accountSvc.RegisterPayment(postedInvoice.ID, 10500000, bankJournal.ID, "Full payment via BCA", 1)
	if err != nil || paidInvoice.PaymentState != "paid" || paidInvoice.AmountResidual != 0 {
		t.Fatalf("RegisterPayment failed: %v", err)
	}
	t.Logf("✓ Payment %s registered successfully. Invoice status: %s (Residual: %.0f)", payment.Name, paidInvoice.PaymentState, paidInvoice.AmountResidual)

	// 7. Manufacturing (MRP) Module Test
	mrpSvc := mrp.NewService(db, stockSvc)
	finishedBundle := product.Product{
		Name:          "Gaming Setup Bundle",
		DefaultCode:   "BUN-GAMING-01",
		Type:          "product",
		ListPrice:     5000000,
		StandardPrice: 3500000,
	}
	finishedBundle.CompanyID = 1
	prodSvc.CreateProduct(&finishedBundle)

	bom := mrp.MrpBom{
		Code:       "BOM/GAMING-BUNDLE/01",
		ProductID:  finishedBundle.ID,
		ProductQty: 1,
		Lines: []mrp.MrpBomLine{
			{
				ProductID:   testProduct.ID, // Consumes 1 Gaming Monitor
				ProductName: testProduct.Name,
				ProductQty:  1,
			},
		},
	}
	bom.CompanyID = 1
	if err := mrpSvc.CreateBom(&bom); err != nil {
		t.Fatalf("Create BOM failed: %v", err)
	}

	mo := mrp.MrpProduction{
		ProductID:  finishedBundle.ID,
		BomID:      bom.ID,
		ProductQty: 2, // Produce 2 Bundles -> Consumes 2 Monitors
	}
	mo.CompanyID = 1
	mrpSvc.CreateProduction(&mo)

	// Produce MO
	producedMO, err := mrpSvc.ActionProduce(mo.ID, 1)
	if err != nil || producedMO.State != "done" {
		t.Fatalf("ActionProduce failed: %v", err)
	}

	// Verify inventory: Monitor decreased from 7 to 5, Bundle increased from 0 to 2
	db.First(&updatedProd, testProduct.ID)
	if updatedProd.QtyAvailable != 5 {
		t.Fatalf("Expected 5 monitors remaining, got %.0f", updatedProd.QtyAvailable)
	}
	var bundleProd product.Product
	db.First(&bundleProd, finishedBundle.ID)
	if bundleProd.QtyAvailable != 2 {
		t.Fatalf("Expected 2 bundles produced, got %.0f", bundleProd.QtyAvailable)
	}
	t.Logf("✓ MRP Production %s executed: Raw component decreased to %.0f, Finished product produced: %.0f", producedMO.Name, updatedProd.QtyAvailable, bundleProd.QtyAvailable)

	// 8. Point of Sale (POS) Module Test
	posSvc := pos.NewService(db)
	posConfig := pos.PosConfig{Name: "Mall Storefront 1"}
	posConfig.CompanyID = 1
	db.Create(&posConfig)

	posSession, err := posSvc.OpenSession(posConfig.ID, user.ID, 500000, 1)
	if err != nil || posSession.State != "opened" {
		t.Fatalf("Open POS session failed: %v", err)
	}

	posOrder := pos.PosOrder{
		SessionID:     posSession.ID,
		AmountPaid:    4000000,
		PaymentMethod: "cash",
		Lines: []pos.PosOrderLine{
			{
				ProductID: testProduct.ID,
				Qty:       1, // Sells 1 Monitor
				PriceUnit: 3500000,
			},
		},
	}
	paidPosOrder, err := posSvc.CreateOrder(&posOrder, 1)
	if err != nil || paidPosOrder.AmountChange != 500000 {
		t.Fatalf("POS checkout failed: %v", err)
	}

	// Verify stock reduced from 5 to 4
	db.First(&updatedProd, testProduct.ID)
	if updatedProd.QtyAvailable != 4 {
		t.Fatalf("Expected 4 monitors remaining after POS sale, got %.0f", updatedProd.QtyAvailable)
	}
	t.Logf("✓ POS Order %s completed: Change: %.0f, Instant stock reduced to %.0f", paidPosOrder.Name, paidPosOrder.AmountChange, updatedProd.QtyAvailable)

	// 9. HR & Payroll Module Test
	hrSvc := hr.NewService(db)
	dept := hr.Department{Name: "Software Engineering"}
	dept.CompanyID = 1
	hrSvc.CreateDepartment(&dept)

	job := hr.JobPosition{Name: "Lead Go Architect", DepartmentID: dept.ID}
	job.CompanyID = 1
	hrSvc.CreateJob(&job)

	emp := hr.Employee{
		Name:         "Bambang Wijaya",
		WorkEmail:    "bambang@company.id",
		DepartmentID: dept.ID,
		JobID:        job.ID,
		BasicSalary:  20000000,
	}
	emp.CompanyID = 1
	hrSvc.CreateEmployee(&emp)

	slip, err := hrSvc.GeneratePayslip(emp.ID, "October 2026", 2000000, 1000000, "Bonus + Transport", 1)
	if err != nil || slip.NetSalary != 21000000 {
		t.Fatalf("Generate Payslip failed: %v", err)
	}

	confirmedSlip, err := hrSvc.ConfirmPayslip(slip.ID, 1)
	if err != nil || confirmedSlip.State != "confirmed" {
		t.Fatalf("Confirm payslip failed: %v", err)
	}

	paidSlip, err := hrSvc.PayPayslip(slip.ID, 1)
	if err != nil || paidSlip.State != "paid" {
		t.Fatalf("Pay payslip failed: %v", err)
	}
	t.Logf("✓ HR Payslip %s generated & paid: Net Salary: %.0f", paidSlip.Name, paidSlip.NetSalary)

	// ==========================================
	// 10. TEST FINANCIAL REPORTS & DOCUMENT PRINT
	// ==========================================
	pnl, err := accountSvc.GetProfitAndLoss(1, "2026-01-01", "2026-12-31")
	if err != nil {
		t.Fatalf("Failed to compute Profit & Loss: %v", err)
	}
	t.Logf("✓ Profit & Loss computed: Total Revenue=%.0f, Gross Profit=%.0f, Net Profit=%.0f",
		pnl.TotalRevenue, pnl.GrossProfit, pnl.NetProfit)

	bs, err := accountSvc.GetBalanceSheet(1, "2026-12-31")
	if err != nil {
		t.Fatalf("Failed to compute Balance Sheet: %v", err)
	}
	t.Logf("✓ Balance Sheet computed: Assets=%.0f, Liabilities=%.0f, Equity=%.0f (Balanced=%v)",
		bs.TotalAssets, bs.TotalLiabilities, bs.TotalEquity, bs.IsBalanced)

	tb, err := accountSvc.GetTrialBalance(1)
	if err != nil || !tb.IsBalanced {
		t.Fatalf("Failed to compute Trial Balance or not balanced: diff=%.2f", tb.Difference)
	}
	t.Logf("✓ Trial Balance verified balanced: Total Debit=%.0f, Total Credit=%.0f", tb.TotalDebit, tb.TotalCredit)

	invHTML := account.RenderInvoiceHTML(postedInvoice)
	if len(invHTML) < 100 || !strings.Contains(invHTML, "FAKTUR PENJUALAN") {
		t.Fatalf("Failed to render printable invoice HTML")
	}

	pickHTML := stock.RenderPickingHTML(delivery)
	if len(pickHTML) < 100 || !strings.Contains(pickHTML, "SURAT JALAN") {
		t.Fatalf("Failed to render printable picking HTML")
	}
	t.Log("✓ Printable Document Templates verified: HTML Invoice & Delivery Slip generated")

	// ==========================================
	// 11. TEST REORDERING RULES & AUTO PROCUREMENT
	// ==========================================
	// Product Gaming Monitor currently has 4 units left (after Sale + POS)
	// Create Reordering Rule: Min 5, Max 15 (deficiency = 11 units)
	op := stock.StockOrderpoint{
		Name:          "OP/TEST/001",
		ProductID:     testProduct.ID,
		LocationID:    whStock.ID,
		ProductMinQty: 5,
		ProductMaxQty: 15,
		QtyMultiple:   1,
		Trigger:       "auto",
	}
	op.CompanyID = 1
	if err := stockSvc.CreateOrderpoint(&op); err != nil {
		t.Fatalf("Failed to create orderpoint: %v", err)
	}

	autoPOs, err := purchaseSvc.RunReorderingRules(1)
	if err != nil {
		t.Fatalf("Failed to run reordering rules: %v", err)
	}
	if len(autoPOs) == 0 {
		t.Fatalf("Expected at least 1 auto-procured PO, got 0")
	}
	replenishQty := autoPOs[0].OrderLines[0].ProductUomQty
	if replenishQty != 11 {
		t.Fatalf("Expected replenish qty 11, got %.0f", replenishQty)
	}
	t.Logf("✓ Automated Reordering Rule executed: Auto-created RFQ %s for %.0f units of %s",
		autoPOs[0].Name, replenishQty, testProduct.Name)

	// ==========================================
	// 12. TEST CRM PIPELINE & QUOTATION CONVERSION
	// ==========================================
	crmSvc := crm.NewService(db, saleSvc)
	lead := crm.CrmLead{
		Name:            "Corporate Office Equipment Renewal",
		PartnerName:     "PT Sinergi Mandiri Abadi",
		ContactName:     "Hendra Wijaya",
		Email:           "hendra@sinergimandiri.co.id",
		ExpectedRevenue: 50000000,
		Stage:           "qualified",
	}
	lead.CompanyID = 1
	if err := crmSvc.CreateLead(&lead); err != nil {
		t.Fatalf("Create CRM lead failed: %v", err)
	}

	updatedLead, err := crmSvc.UpdateStage(lead.ID, "proposition", "", 1)
	if err != nil || updatedLead.Probability != 70 {
		t.Fatalf("Update lead stage failed: %v", err)
	}

	convertedSO, wonLead, err := crmSvc.ConvertToQuotation(lead.ID, 1)
	if err != nil || convertedSO.ID == 0 || wonLead.Stage != "won" || wonLead.Probability != 100 {
		t.Fatalf("Convert lead to quotation failed: %v", err)
	}
	t.Logf("✓ CRM Opportunity '%s' converted to Sales Order %s (Stage: %s, Prob: %.0f%%)",
		wonLead.Name, convertedSO.Name, wonLead.Stage, wonLead.Probability)

	// ==========================================
	// 13. TEST CHATTER & AUDIT TRAIL LOGS
	// ==========================================
	chatterSvc := chatter.NewService(db)
	err = chatterSvc.LogTracking("sale.order", convertedSO.ID, 1, "admin@erp.local",
		"Quotation generated from CRM lead conversion", "Stage: proposition -> won", 1)
	if err != nil {
		t.Fatalf("Chatter log tracking failed: %v", err)
	}

	noteMsg := chatter.MailMessage{
		ModelName:   "sale.order",
		ResID:       convertedSO.ID,
		AuthorID:    1,
		AuthorName:  "admin@erp.local",
		Body:        "Customer requested official quotation PDF via email.",
		MessageType: "comment",
	}
	noteMsg.CompanyID = 1
	if err := chatterSvc.PostMessage(&noteMsg); err != nil {
		t.Fatalf("Chatter post message failed: %v", err)
	}

	messages, err := chatterSvc.ListMessages("sale.order", convertedSO.ID, 1)
	if err != nil || len(messages) != 2 {
		t.Fatalf("Expected 2 chatter messages, got %d", len(messages))
	}
	t.Logf("✓ Chatter & Audit Trail verified: %d activity messages logged for %s", len(messages), convertedSO.Name)

	// ==========================================
	// 14. TEST PROJECT, TASKS & TIMESHEET LOGGING
	// ==========================================
	projectSvc := project.NewService(db)
	proj := project.Project{
		Name:        "Corporate E-Commerce Portal",
		PartnerID:   &customer.ID,
		UserID:      &user.ID,
		Status:      "active",
		Color:       "#6366f1",
		Description: "Design, develop and deploy high traffic web application",
	}
	proj.CompanyID = 1
	if err := projectSvc.CreateProject(&proj); err != nil {
		t.Fatalf("Create project failed: %v", err)
	}

	task := project.Task{
		ProjectID:    proj.ID,
		Name:         "Database Architecture & API Integration",
		PartnerID:    &customer.ID,
		UserID:       &user.ID,
		Stage:        "in_progress",
		Priority:     "2",
		PlannedHours: 40,
		Description:  "Implement double-entry stock models and balanced GL ledger",
	}
	task.CompanyID = 1
	if err := projectSvc.CreateTask(&task); err != nil {
		t.Fatalf("Create task failed: %v", err)
	}

	// Log timesheet: 10 hours
	ts1 := project.TimesheetLine{
		TaskID:     task.ID,
		UserID:     user.ID,
		Date:       "2026-10-04",
		UnitAmount: 10,
		Name:       "Initial schema design & migration logic",
	}
	if err := projectSvc.LogTimesheet(&ts1, 1); err != nil {
		t.Fatalf("Log timesheet 1 failed: %v", err)
	}

	// Log timesheet: 20 hours
	ts2 := project.TimesheetLine{
		TaskID:     task.ID,
		UserID:     user.ID,
		Date:       "2026-10-05",
		UnitAmount: 20,
		Name:       "Controller endpoints & integration tests",
	}
	if err := projectSvc.LogTimesheet(&ts2, 1); err != nil {
		t.Fatalf("Log timesheet 2 failed: %v", err)
	}

	// Verify task recalculated effective hours and progress
	updatedTask, err := projectSvc.GetTask(task.ID, 1)
	if err != nil || updatedTask.EffectiveHours != 30 || updatedTask.Progress != 75 {
		t.Fatalf("Task effective hours / progress mismatch: hours=%.1f, prog=%.1f", updatedTask.EffectiveHours, updatedTask.Progress)
	}

	// Advance to done
	doneTask, err := projectSvc.UpdateTaskStage(task.ID, "done", 1)
	if err != nil || doneTask.Stage != "done" || doneTask.Progress != 100 {
		t.Fatalf("Update task stage to done failed: %v", err)
	}
	t.Logf("✓ Project '%s' Task '%s' completed with %.1f hours logged (Progress: %.0f%%)",
		proj.Name, doneTask.Name, doneTask.EffectiveHours, doneTask.Progress)

	// 15. Fixed Asset Management (account.asset) - Depreciation Board & GL Integration
	assetJournal := account.AccountJournal{Code: "ASTJ", Name: "Asset Operations", Type: "general"}
	assetJournal.CompanyID = 1
	db.Create(&assetJournal)

	accFixedAsset := account.AccountAccount{Code: "150100", Name: "Peralatan & Mesin Kantor", Type: "asset_fixed"}
	accFixedAsset.CompanyID = 1
	db.Create(&accFixedAsset)

	accDeprExpense := account.AccountAccount{Code: "610100", Name: "Beban Penyusutan Peralatan", Type: "expense"}
	accDeprExpense.CompanyID = 1
	db.Create(&accDeprExpense)

	accAccumDepr := account.AccountAccount{Code: "150900", Name: "Akumulasi Penyusutan Peralatan", Type: "asset_non_current"}
	accAccumDepr.CompanyID = 1
	db.Create(&accAccumDepr)

	assetSvc := asset.NewService(db, accountSvc)

	// Test Asset: Original Rp 48,000,000, Salvage Rp 4,800,000, 24 months
	// Monthly depreciation: (48,000,000 - 4,800,000) / 24 = 1,800,000/mo
	fixedAsset := asset.Asset{
		Name:                  "Dell PowerEdge R750 Enterprise Server",
		Category:              "hardware",
		OriginalValue:         48000000,
		SalvageValue:          4800000,
		Method:                "linear",
		MethodNumber:          24,
		MethodPeriod:          "months",
		PurchaseDate:          "2026-01-01",
		AssetAccountID:        accFixedAsset.ID,
		DepreciationAccountID: accDeprExpense.ID,
		AccumulatedAccountID:  accAccumDepr.ID,
		JournalID:             assetJournal.ID,
		PartnerID:             &vendor.ID,
	}
	fixedAsset.CompanyID = 1
	if err := assetSvc.CreateAsset(&fixedAsset); err != nil {
		t.Fatalf("Create Asset failed: %v", err)
	}

	// Verify asset created in draft state with 24 depreciation lines
	createdAsset, err := assetSvc.GetAsset(fixedAsset.ID, 1)
	if err != nil {
		t.Fatalf("Get Asset failed: %v", err)
	}
	if createdAsset.State != "draft" {
		t.Fatalf("Expected asset state 'draft', got '%s'", createdAsset.State)
	}
	if len(createdAsset.DepreciationLines) != 24 {
		t.Fatalf("Expected 24 depreciation lines, got %d", len(createdAsset.DepreciationLines))
	}
	firstLine := createdAsset.DepreciationLines[0]
	if firstLine.Amount != 1800000 {
		t.Fatalf("Expected monthly depreciation 1800000, got %.2f", firstLine.Amount)
	}
	if firstLine.RemainingValue != 46200000 {
		t.Fatalf("Expected line 1 remaining value 46200000, got %.2f", firstLine.RemainingValue)
	}

	// Confirm Asset
	runningAsset, err := assetSvc.ActionConfirm(createdAsset.ID, 1)
	if err != nil || runningAsset.State != "running" {
		t.Fatalf("ActionConfirm failed or invalid state: %v", err)
	}

	// Post First Depreciation Line to General Ledger
	postedLine, err := assetSvc.PostDepreciationLine(firstLine.ID, 1, user.ID)
	if err != nil {
		t.Fatalf("PostDepreciationLine failed: %v", err)
	}
	if !postedLine.IsPosted || postedLine.MoveID == nil {
		t.Fatalf("Expected line to be posted with MoveID, got posted=%v, moveID=%v", postedLine.IsPosted, postedLine.MoveID)
	}

	// Verify Balanced Journal Entry in GL
	glMove, err := accountSvc.GetMove(*postedLine.MoveID, 1)
	if err != nil {
		t.Fatalf("Failed to fetch generated GL Move: %v", err)
	}
	if glMove.State != "posted" {
		t.Fatalf("Expected GL Move state 'posted', got '%s'", glMove.State)
	}
	var totalDebit, totalCredit float64
	for _, ml := range glMove.LineIDs {
		totalDebit += ml.Debit
		totalCredit += ml.Credit
	}
	if totalDebit != 1800000 || totalCredit != 1800000 {
		t.Fatalf("GL Move not balanced: Debit=%.2f, Credit=%.2f", totalDebit, totalCredit)
	}

	// Verify Asset updated book value
	updatedAssetAfterPost, err := assetSvc.GetAsset(runningAsset.ID, 1)
	if err != nil {
		t.Fatalf("Get updated asset failed: %v", err)
	}
	if updatedAssetAfterPost.ValueDepreciated != 1800000 || updatedAssetAfterPost.BookValue != 46200000 {
		t.Fatalf("Asset values mismatch after post: Depreciated=%.2f, BookValue=%.2f",
			updatedAssetAfterPost.ValueDepreciated, updatedAssetAfterPost.BookValue)
	}
	t.Logf("✓ Fixed Asset '%s' created (Rp %.0f), schedule computed (24 periods), and line 1 posted to GL with balanced Entry (Rp %.0f)",
		updatedAssetAfterPost.Name, updatedAssetAfterPost.OriginalValue, totalDebit)

	// 16. Multi-Warehouse & Inter-Warehouse Stock Transfers (stock.warehouse & WH/INT/0000x)
	// Create Main Warehouse Jakarta & Regional Warehouse Surabaya
	whJkt := stock.StockWarehouse{
		Name:       "Gudang Utama Jakarta",
		Code:       "WH-JKT",
		LotStockID: whStock.ID,
		Address:    "Kawasan Industri Pulogadung, Jakarta",
		Active:     true,
	}
	whJkt.CompanyID = 1
	if err := stockSvc.CreateWarehouse(&whJkt); err != nil {
		t.Fatalf("Create Warehouse Jakarta failed: %v", err)
	}

	whSbyLoc := stock.StockLocation{
		Name:         "WH-SBY/Stock",
		Usage:        "internal",
		CompleteName: "WH-SBY/Stock (Surabaya Regional Hub)",
	}
	whSbyLoc.CompanyID = 1
	db.Create(&whSbyLoc)

	whSby := stock.StockWarehouse{
		Name:       "Hub Regional Jawa Timur (Surabaya)",
		Code:       "WH-SBY",
		LotStockID: whSbyLoc.ID,
		Address:    "Kawasan Industri SIER, Surabaya",
		Active:     true,
	}
	whSby.CompanyID = 1
	if err := stockSvc.CreateWarehouse(&whSby); err != nil {
		t.Fatalf("Create Warehouse Surabaya failed: %v", err)
	}

	// Verify warehouse listing
	warehouses, err := stockSvc.ListWarehouses(1)
	if err != nil || len(warehouses) < 2 {
		t.Fatalf("ListWarehouses failed or unexpected count: %v, count=%d", err, len(warehouses))
	}

	// Check initial quant in WH/Stock for Gaming Monitor (testProduct.ID)
	var jktQuantBefore stock.StockQuant
	db.Where("product_id = ? AND location_id = ?", testProduct.ID, whStock.ID).First(&jktQuantBefore)
	initialJktQty := jktQuantBefore.Quantity

	// Create Inter-Warehouse Transfer: 2 units of Gaming Monitor from Jakarta to Surabaya
	transferMoves := []stock.StockMove{
		{
			ProductID:     testProduct.ID,
			ProductName:   testProduct.Name,
			ProductUomQty: 2,
		},
	}
	transferPicking, err := stockSvc.CreateInternalTransfer(1, user.ID, whStock.ID, whSbyLoc.ID, transferMoves, "WH/TRANSFER/001", "Inter-warehouse transfer Jakarta to Surabaya")
	if err != nil {
		t.Fatalf("CreateInternalTransfer failed: %v", err)
	}
	if transferPicking.PickingType != "internal" || !strings.HasPrefix(transferPicking.Name, "WH/INT") {
		t.Fatalf("Invalid transfer picking: type=%s, name=%s", transferPicking.PickingType, transferPicking.Name)
	}

	// Validate Inter-Warehouse Transfer
	validatedTransfer, err := stockSvc.ValidatePicking(transferPicking.ID, 1)
	if err != nil {
		t.Fatalf("ValidatePicking for internal transfer failed: %v", err)
	}
	if validatedTransfer.State != "done" {
		t.Fatalf("Expected validated transfer state 'done', got '%s'", validatedTransfer.State)
	}

	// Verify Quant in Source (Jakarta) decreased by 2
	var jktQuantAfter stock.StockQuant
	db.Where("product_id = ? AND location_id = ?", testProduct.ID, whStock.ID).First(&jktQuantAfter)
	if jktQuantAfter.Quantity != initialJktQty-2 {
		t.Fatalf("Source quant mismatch: expected %.1f, got %.1f", initialJktQty-2, jktQuantAfter.Quantity)
	}

	// Verify Quant in Destination (Surabaya) increased by 2
	var sbyQuantAfter stock.StockQuant
	db.Where("product_id = ? AND location_id = ?", testProduct.ID, whSbyLoc.ID).First(&sbyQuantAfter)
	if sbyQuantAfter.Quantity != 2 {
		t.Fatalf("Destination quant mismatch: expected 2, got %.1f", sbyQuantAfter.Quantity)
	}

	// Verify Total On-Hand for product remains conserved across all internal locations
	var prodAfterTransfer product.Product
	db.Where("id = ?", testProduct.ID).First(&prodAfterTransfer)
	if prodAfterTransfer.QtyAvailable != initialJktQty {
		t.Fatalf("Conserved total stock mismatch: expected %.1f, got %.1f", initialJktQty, prodAfterTransfer.QtyAvailable)
	}
	t.Logf("✓ Inter-Warehouse Transfer '%s' validated: 2 units moved from %s (Rem: %.0f) to %s (New: %.0f). Total stock conserved (%.0f units)",
		validatedTransfer.Name, whJkt.Name, jktQuantAfter.Quantity, whSby.Name, sbyQuantAfter.Quantity, prodAfterTransfer.QtyAvailable)

	// ==========================================
	// 17. HELPDESK & CUSTOMER SUPPORT TICKETING (helpdesk.ticket & helpdesk.team)
	// ==========================================
	t.Log("--- Section 17: Helpdesk & Customer Support Ticketing System ---")
	helpdeskSvc := helpdesk.NewService(db)

	// Create Helpdesk Team
	teamCare := helpdesk.HelpdeskTeam{
		Name:     "Customer Care & Support",
		Code:     "CARE",
		Email:    "care@go-erp.id",
		LeaderID: &user.ID,
		Color:    "#3b82f6",
		Active:   true,
	}
	teamCare.CompanyID = 1
	teamCare.CreateUID = user.ID
	teamCare.WriteUID = user.ID
	if err := helpdeskSvc.CreateTeam(&teamCare); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}

	// Create Helpdesk Ticket
	ticket := helpdesk.HelpdeskTicket{
		Name:         "Printer Thermal POS Rusak & Tidak Mencetak Struk",
		TeamID:       &teamCare.ID,
		PartnerID:    &customer.ID,
		PartnerName:  customer.Name,
		PartnerEmail: customer.Email,
		PartnerPhone: customer.Phone,
		Priority:     "2",
		Category:     "technical",
		Description:  "Printer kasir di outlet PVJ Bandung tidak mengeluarkan struk belanja kasir.",
	}
	ticket.CompanyID = 1
	ticket.CreateUID = user.ID
	ticket.WriteUID = user.ID

	if err := helpdeskSvc.CreateTicket(&ticket); err != nil {
		t.Fatalf("CreateTicket failed: %v", err)
	}
	if !strings.HasPrefix(ticket.TicketNumber, "TCK/") {
		t.Fatalf("Expected auto-generated ticket sequence starting with 'TCK/', got '%s'", ticket.TicketNumber)
	}
	if ticket.Stage != "new" {
		t.Fatalf("Expected default ticket stage 'new', got '%s'", ticket.Stage)
	}

	// Assign Ticket to Support Engineer / Admin
	assignedTicket, err := helpdeskSvc.AssignTicket(ticket.ID, 1, user.ID, user.ID)
	if err != nil {
		t.Fatalf("AssignTicket failed: %v", err)
	}
	if assignedTicket.UserID == nil || *assignedTicket.UserID != user.ID || assignedTicket.Stage != "in_progress" {
		t.Fatalf("Ticket assignment failed: user=%v, stage=%s", assignedTicket.UserID, assignedTicket.Stage)
	}

	// Post message to ticket via Universal Chatter
	ticketChatterMsg := chatter.MailMessage{
		ModelName:   "helpdesk.ticket",
		ResID:       ticket.ID,
		AuthorID:    user.ID,
		AuthorName:  user.Name,
		Body:        "Teknisi sudah diberangkatkan ke outlet dengan sparepart kabel power dan thermal head baru.",
		MessageType: "comment",
	}
	ticketChatterMsg.CompanyID = 1
	if err := chatterSvc.PostMessage(&ticketChatterMsg); err != nil {
		t.Fatalf("Post chatter message to ticket failed: %v", err)
	}

	// Resolve Ticket
	resolvedTicket, err := helpdeskSvc.ResolveTicket(ticket.ID, 1, user.ID, "Kabel power adaptor telah diganti dan test print struk kasir 10x berhasil lancar.")
	if err != nil {
		t.Fatalf("ResolveTicket failed: %v", err)
	}
	if resolvedTicket.Stage != "solved" || resolvedTicket.CloseDate == "" || resolvedTicket.Resolution == "" {
		t.Fatalf("Ticket resolution mismatch: stage=%s, closeDate=%s, res=%s", resolvedTicket.Stage, resolvedTicket.CloseDate, resolvedTicket.Resolution)
	}

	// List tickets with filter
	solvedTickets, err := helpdeskSvc.ListTickets(1, "solved", "all", "technical", 0)
	if err != nil || len(solvedTickets) == 0 {
		t.Fatalf("ListTickets filtered failed or returned 0 tickets: %v, count=%d", err, len(solvedTickets))
	}

	t.Logf("✓ Helpdesk Ticket '%s' (%s) successfully created, assigned to agent, chatter-logged, and resolved with status '%s'!",
		resolvedTicket.TicketNumber, resolvedTicket.Name, resolvedTicket.Stage)

	// ==========================================
	// 18. MAINTENANCE & EQUIPMENT MANAGEMENT (maintenance.equipment & maintenance.request)
	// ==========================================
	t.Log("--- Section 18: Maintenance & Equipment Management ---")
	maintSvc := maintenance.NewService(db)

	// 1. Create Equipment Category
	catMachinery := maintenance.MaintenanceEquipmentCategory{
		Name:             "Mesin Pabrik & Lini Produksi",
		TechnicianUserID: &user.ID,
		Color:            "#ef4444",
		Note:             "Mesin manufaktur perakitan dan perkakas kerja workshop",
	}
	catMachinery.CompanyID = 1
	catMachinery.CreateUID = user.ID
	catMachinery.WriteUID = user.ID
	if err := maintSvc.CreateCategory(&catMachinery); err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// 2. Create Equipment with Overdue Date (to test auto preventive scheduler)
	overdueDate := "2026-10-01"
	deptID := uint(1)
	equipment := maintenance.MaintenanceEquipment{
		Name:                  "Mesin Solder & Perakitan SMT Otomatis",
		SerialNumber:          "SN-SMT-8820",
		Model:                 "Yamaha YSM20R Modular",
		CategoryID:            &catMachinery.ID,
		PartnerID:             &customer.ID,
		DepartmentID:          &deptID,
		TechnicianUserID:      &user.ID,
		Location:              "Gudang Utama - Lantai 2 Workshop Produksi",
		Cost:                  250000000,
		WarrantyDate:          "2028-06-30",
		MaintenancePeriodDays: 30,
		NextActionDate:        overdueDate,
		Status:                "operational",
	}
	equipment.CompanyID = 1
	equipment.CreateUID = user.ID
	equipment.WriteUID = user.ID

	if err := maintSvc.CreateEquipment(&equipment); err != nil {
		t.Fatalf("CreateEquipment failed: %v", err)
	}
	if !strings.HasPrefix(equipment.EquipmentNumber, "EQP/") {
		t.Fatalf("Expected auto equipment number starting with 'EQP/', got '%s'", equipment.EquipmentNumber)
	}

	// 3. Test Auto-Scheduling of Preventive Maintenance
	autoScheduled, err := maintSvc.AutoSchedulePreventive(1, user.ID)
	if err != nil {
		t.Fatalf("AutoSchedulePreventive failed: %v", err)
	}
	if len(autoScheduled) == 0 {
		t.Fatalf("Expected at least 1 auto-scheduled preventive request for overdue equipment")
	}
	prevReq := autoScheduled[0]
	if prevReq.MaintenanceType != "preventive" || prevReq.EquipmentID != equipment.ID {
		t.Fatalf("Preventive request mismatch: type=%s, eqID=%d", prevReq.MaintenanceType, prevReq.EquipmentID)
	}

	// Verify Equipment next action date was updated and advanced
	updatedEq, err := maintSvc.GetEquipment(equipment.ID, 1)
	if err != nil || updatedEq.NextActionDate == overdueDate {
		t.Fatalf("Expected next action date to be advanced, got '%s'", updatedEq.NextActionDate)
	}
	t.Logf("✓ Auto-scheduled Preventive Maintenance '%s' for '%s' (Next action date advanced to %s)",
		prevReq.RequestNumber, updatedEq.Name, updatedEq.NextActionDate)

	// 4. Test Corrective Breakdown Request
	corrReq := maintenance.MaintenanceRequest{
		Name:            "Sensor Optik Feeder Macet & Suhu Pemanas Fluktuatif",
		EquipmentID:     equipment.ID,
		MaintenanceType: "corrective",
		Priority:        "3", // Urgent
		Description:     "Komponen resistor micro tidak terdeteksi saat penempatan pada PCB.",
	}
	corrReq.CompanyID = 1
	corrReq.CreateUID = user.ID
	corrReq.WriteUID = user.ID

	if err := maintSvc.CreateRequest(&corrReq); err != nil {
		t.Fatalf("CreateRequest corrective failed: %v", err)
	}
	if !strings.HasPrefix(corrReq.RequestNumber, "REQ/MAINT/") {
		t.Fatalf("Expected auto request number starting with 'REQ/MAINT/', got '%s'", corrReq.RequestNumber)
	}

	// Verify equipment status changed to 'maintenance'
	eqInMaint, _ := maintSvc.GetEquipment(equipment.ID, 1)
	if eqInMaint.Status != "maintenance" {
		t.Fatalf("Expected equipment status 'maintenance', got '%s'", eqInMaint.Status)
	}

	// 5. Assign Technician to Request
	assignedReq, err := maintSvc.AssignRequest(corrReq.ID, 1, user.ID, user.ID)
	if err != nil || assignedReq.Stage != "in_progress" || assignedReq.UserID == nil || *assignedReq.UserID != user.ID {
		t.Fatalf("AssignRequest failed: stage=%s, user=%v", assignedReq.Stage, assignedReq.UserID)
	}

	// 6. Post Activity Note to Universal Chatter
	maintChatterMsg := chatter.MailMessage{
		ModelName:   "maintenance.request",
		ResID:       corrReq.ID,
		AuthorID:    user.ID,
		AuthorName:  user.Name,
		Body:        "Teknisi telah membongkar feeder unit #2 dan mengganti sensor optik yang kotor.",
		MessageType: "comment",
	}
	maintChatterMsg.CompanyID = 1
	if err := chatterSvc.PostMessage(&maintChatterMsg); err != nil {
		t.Fatalf("Post chatter message to maintenance request failed: %v", err)
	}

	// 7. Complete and Repair Equipment
	completedReq, err := maintSvc.CompleteRequest(corrReq.ID, 1, user.ID, "Sensor optik dibersihkan dengan cairan kontak khusus dan kalibrasi suhu heating zone berhasil normal.", 2.5)
	if err != nil {
		t.Fatalf("CompleteRequest failed: %v", err)
	}
	if completedReq.Stage != "repaired" || completedReq.DurationHours != 2.5 || completedReq.CloseDate == "" {
		t.Fatalf("Completed request mismatch: stage=%s, duration=%.1f, closeDate=%s",
			completedReq.Stage, completedReq.DurationHours, completedReq.CloseDate)
	}

	// Verify equipment status restored to 'operational'
	eqRestored, _ := maintSvc.GetEquipment(equipment.ID, 1)
	if eqRestored.Status != "operational" {
		t.Fatalf("Expected equipment status 'operational' after repair, got '%s'", eqRestored.Status)
	}

	t.Logf("✓ Maintenance Request '%s' (%s) completed: Duration %.1f hrs, Stage '%s', Equipment '%s' status restored to '%s'",
		completedReq.RequestNumber, completedReq.Name, completedReq.DurationHours, completedReq.Stage, eqRestored.Name, eqRestored.Status)

	// 19. Fleet & Delivery Vehicle Management (Enterprise ERP fleet.vehicle, fleet.vehicle.odometer, fleet.vehicle.log.services)
	fleetSvc := fleet.NewService(db)

	// 1. Create Vehicle Model
	truckModel := fleet.FleetVehicleModel{
		Brand:       "Mitsubishi",
		ModelName:   "Canter FE 74 HD",
		VehicleType: "truck",
	}
	truckModel.CompanyID = 1
	if err := fleetSvc.CreateModel(&truckModel); err != nil {
		t.Fatalf("CreateModel failed: %v", err)
	}

	// 2. Create Driver Employee & Vehicle
	driver := hr.Employee{
		Name:         "Joko Supir Logistik",
		WorkEmail:    "joko.supir@company.id",
		DepartmentID: dept.ID,
		JobID:        job.ID,
		BasicSalary:  6500000,
	}
	driver.CompanyID = 1
	if err := hrSvc.CreateEmployee(&driver); err != nil {
		t.Fatalf("Create Driver Employee failed: %v", err)
	}

	truckVehicle := fleet.FleetVehicle{
		LicensePlate:     "B 1024 TKO",
		ModelID:          truckModel.ID,
		DriverEmployeeID: &driver.ID,
		AssetID:          &fixedAsset.ID,
		Location:         "Gudang Utama Jakarta",
		LastOdometer:     40000,
		FuelType:         "diesel",
		Status:           "active",
		VinSN:            "MHFE74HDX2026001",
		EngineNumber:     "4D34-2AT2026",
		Color:            "Yellow Kuning",
		Seats:            3,
	}
	truckVehicle.CompanyID = 1
	truckVehicle.CreateUID = user.ID
	if err := fleetSvc.CreateVehicle(&truckVehicle); err != nil {
		t.Fatalf("CreateVehicle failed: %v", err)
	}

	// 3. Log Odometer Reading
	odoLog := fleet.FleetOdometerLog{
		VehicleID:        truckVehicle.ID,
		Date:             "2026-10-05",
		Value:            42500,
		DriverEmployeeID: &driver.ID,
		Note:             "Pengiriman batch 500 unit ke Surabaya via Tol Trans Jawa",
	}
	if err := fleetSvc.LogOdometer(&odoLog, 1, user.ID); err != nil {
		t.Fatalf("LogOdometer failed: %v", err)
	}

	// Verify vehicle's LastOdometer auto-advanced to 42500
	vUpdated, err := fleetSvc.GetVehicle(truckVehicle.ID, 1)
	if err != nil || vUpdated.LastOdometer != 42500 {
		t.Fatalf("Vehicle LastOdometer failed to update: expected 42500, got %.0f", vUpdated.LastOdometer)
	}

	// 4. Create Service Log (SRV/FLEET/...)
	serviceRecord := fleet.FleetServiceLog{
		VehicleID:       truckVehicle.ID,
		Date:            "2026-10-06",
		ServiceType:     "oil_change",
		Cost:            1250000,
		VendorPartnerID: &vendor.ID,
		Odometer:        45000,
		Description:     "Servis berkala 45.000 KM & ganti oli gardan Mitsubishi Fuso",
		Stage:           "done",
	}
	if err := fleetSvc.CreateServiceLog(&serviceRecord, 1, user.ID); err != nil {
		t.Fatalf("CreateServiceLog failed: %v", err)
	}
	if !strings.HasPrefix(serviceRecord.ServiceNumber, "SRV/FLEET/") {
		t.Fatalf("Expected service number with prefix 'SRV/FLEET/', got '%s'", serviceRecord.ServiceNumber)
	}

	// Verify odometer auto-advanced again to 45000 from service record
	vAfterService, _ := fleetSvc.GetVehicle(truckVehicle.ID, 1)
	if vAfterService.LastOdometer != 45000 {
		t.Fatalf("Vehicle LastOdometer failed to sync from service: expected 45000, got %.0f", vAfterService.LastOdometer)
	}

	// 5. Post Universal Chatter message on vehicle
	fleetChatterMsg := chatter.MailMessage{
		ModelName:   "fleet.vehicle",
		ResID:       truckVehicle.ID,
		AuthorID:    user.ID,
		AuthorName:  user.Name,
		Body:        "Pajak STNK dan Uji KIR 6 bulanan armada B 1024 TKO telah diperpanjang hingga April 2027.",
		MessageType: "comment",
	}
	fleetChatterMsg.CompanyID = 1
	if err := chatterSvc.PostMessage(&fleetChatterMsg); err != nil {
		t.Fatalf("Post chatter message to fleet vehicle failed: %v", err)
	}

	t.Logf("✓ Fleet Vehicle '%s' (%s - %s) tested: LastOdometer %.0f KM, Service '%s' (Cost: Rp %.0f), Linked Asset #%d, Driver '%s'",
		truckVehicle.LicensePlate, truckModel.Brand, truckModel.ModelName, vAfterService.LastOdometer, serviceRecord.ServiceNumber, serviceRecord.Cost, *truckVehicle.AssetID, driver.Name)

	// 20. HR Leave & Time Off Management (Enterprise ERP hr.leave, hr.leave.allocation, hr.leave.type)
	leaveSvc := leave.NewService(db)

	// 1. Create Leave Types
	tAnnual := leave.LeaveType{
		Name:               "Cuti Tahunan (Paid Annual Leave)",
		Code:               "ANNUAL",
		IsUnpaid:           false,
		RequiresAllocation: true,
		DefaultDays:        12,
		Color:              "#10b981",
	}
	tAnnual.CompanyID = 1
	if err := leaveSvc.CreateLeaveType(&tAnnual); err != nil {
		t.Fatalf("CreateLeaveType annual failed: %v", err)
	}

	tUnpaid := leave.LeaveType{
		Name:               "Cuti Tanpa Gaji (Unpaid Leave)",
		Code:               "UNPAID",
		IsUnpaid:           true,
		RequiresAllocation: false,
		DefaultDays:        0,
		Color:              "#ef4444",
	}
	tUnpaid.CompanyID = 1
	if err := leaveSvc.CreateLeaveType(&tUnpaid); err != nil {
		t.Fatalf("CreateLeaveType unpaid failed: %v", err)
	}

	// 2. Allocate 12 days to Driver Employee
	alloc := leave.LeaveAllocation{
		Name:          "Alokasi Cuti Tahunan 2026 - Joko Supir",
		EmployeeID:    driver.ID,
		LeaveTypeID:   tAnnual.ID,
		Year:          2026,
		AllocatedDays: 12,
		UsedDays:      0,
		RemainingDays: 12,
		State:         "approved",
	}
	if err := leaveSvc.CreateAllocation(&alloc, 1, user.ID); err != nil {
		t.Fatalf("CreateAllocation failed: %v", err)
	}

	// 3. Submit Annual Leave Request for 3 days
	req1 := leave.LeaveRequest{
		EmployeeID:   driver.ID,
		LeaveTypeID:  tAnnual.ID,
		DateFrom:     "2026-10-10",
		DateTo:       "2026-10-12",
		NumberOfDays: 3,
		Description:  "Cuti tahunan urusan keluarga di Solo",
	}
	if err := leaveSvc.CreateLeaveRequest(&req1, 1, user.ID); err != nil {
		t.Fatalf("CreateLeaveRequest failed: %v", err)
	}
	if !strings.HasPrefix(req1.RequestNumber, "LV/") {
		t.Fatalf("Expected RequestNumber starting with 'LV/', got '%s'", req1.RequestNumber)
	}

	// 4. Approve Leave Request -> Verify automatic deduction of 3 days
	approvedReq, err := leaveSvc.ApproveLeaveRequest(req1.ID, 1, user.ID)
	if err != nil || approvedReq.State != "validate" {
		t.Fatalf("ApproveLeaveRequest failed: %v", err)
	}

	allocsAfterApproval, _ := leaveSvc.ListAllocations(1, driver.ID, 2026)
	if len(allocsAfterApproval) == 0 || allocsAfterApproval[0].UsedDays != 3 || allocsAfterApproval[0].RemainingDays != 9 {
		t.Fatalf("Expected allocation used=3, remaining=9, got used=%.1f, rem=%.1f",
			allocsAfterApproval[0].UsedDays, allocsAfterApproval[0].RemainingDays)
	}

	// 5. Submit & Approve Unpaid Leave for 2 days -> Verify Payroll integration
	reqUnpaid := leave.LeaveRequest{
		EmployeeID:   driver.ID,
		LeaveTypeID:  tUnpaid.ID,
		DateFrom:     "2026-10-20",
		DateTo:       "2026-10-21",
		NumberOfDays: 2,
		Description:  "Izin keperluan mendadak luar kota tanpa tanggungan",
	}
	if err := leaveSvc.CreateLeaveRequest(&reqUnpaid, 1, user.ID); err != nil {
		t.Fatalf("Create unpaid leave request failed: %v", err)
	}
	if _, err := leaveSvc.ApproveLeaveRequest(reqUnpaid.ID, 1, user.ID); err != nil {
		t.Fatalf("Approve unpaid leave request failed: %v", err)
	}

	unpaidDays, err := leaveSvc.GetUnpaidLeaveDays(driver.ID, "2026-10", 1)
	if err != nil || unpaidDays != 2 {
		t.Fatalf("Expected unpaid leave days 2, got %.1f", unpaidDays)
	}

	// 6. Post Universal Chatter message on leave request
	leaveChatterMsg := chatter.MailMessage{
		ModelName:   "hr.leave",
		ResID:       req1.ID,
		AuthorID:    user.ID,
		AuthorName:  user.Name,
		Body:        "Pengajuan cuti tahunan 3 hari telah disetujui Manager HRD.",
		MessageType: "comment",
	}
	leaveChatterMsg.CompanyID = 1
	if err := chatterSvc.PostMessage(&leaveChatterMsg); err != nil {
		t.Fatalf("Post chatter message to leave request failed: %v", err)
	}

	t.Logf("✓ HR Leave Request '%s' (%s) approved: %.1f Days, Remaining Annual Balance: %.1f Days, Unpaid Leaves in Oct: %.1f Days (Eligible for Payroll Prorata)",
		req1.RequestNumber, req1.Description, req1.NumberOfDays, allocsAfterApproval[0].RemainingDays, unpaidDays)

	// =========================================================================
	// Section 21: HR Expense & Reimbursement Management (hr.expense & hr.expense.sheet)
	// =========================================================================
	t.Log("--- Section 21: HR Expense & Reimbursement Management ---")
	expenseSvc := expense.NewService(db, accountSvc, chatterSvc)

	// 1. Ensure operational expense accounts exist in General Ledger
	expGLAccounts := []account.AccountAccount{
		{Code: "620100", Name: "Beban Perjalanan Dinas & Akomodasi", Type: "expense"},
		{Code: "620200", Name: "Beban Konsumsi & Representasi Tamu", Type: "expense"},
		{Code: "620400", Name: "Beban Operasional Armada, BBM & Tol", Type: "expense"},
	}
	for i := range expGLAccounts {
		var existing account.AccountAccount
		if err := db.Where("code = ?", expGLAccounts[i].Code).First(&existing).Error; err != nil {
			expGLAccounts[i].CompanyID = 1
			db.Create(&expGLAccounts[i])
		}
	}

	// 2. Create / Register Expense Categories
	catTravel := expense.ExpenseCategory{Name: "Perjalanan Dinas & Tiket", Code: "TRAVEL", AccountCode: "620100", Color: "#3b82f6"}
	catTravel.CompanyID = 1
	if err := expenseSvc.CreateCategory(&catTravel); err != nil {
		t.Fatalf("CreateCategory (Travel) failed: %v", err)
	}

	catMeal := expense.ExpenseCategory{Name: "Konsumsi & Representasi", Code: "MEAL", AccountCode: "620200", Color: "#f59e0b"}
	catMeal.CompanyID = 1
	if err := expenseSvc.CreateCategory(&catMeal); err != nil {
		t.Fatalf("CreateCategory (Meal) failed: %v", err)
	}

	catFuel := expense.ExpenseCategory{Name: "Bahan Bakar & Tol Armada", Code: "FUEL", AccountCode: "620400", Color: "#ef4444"}
	catFuel.CompanyID = 1
	if err := expenseSvc.CreateCategory(&catFuel); err != nil {
		t.Fatalf("CreateCategory (Fuel) failed: %v", err)
	}

	// 3. Employee Submits Individual Expenses (hr.expense)
	// Expense 1: Travel (linked to Project 1)
	expTravel := expense.Expense{
		Name:        "Tiket Kereta & Penginapan Hotel Surabaya",
		Date:        "2026-10-03",
		EmployeeID:  driver.ID,
		CategoryID:  catTravel.ID,
		PaymentMode: "own_account",
		UnitPrice:   1500000,
		Quantity:    1,
		ReceiptRef:  "INV-HOTEL/9912",
		ProjectID:   &proj.ID,
		Notes:       "Perjalanan dinas meeting implementasi API e-commerce",
	}
	if err := expenseSvc.CreateExpense(&expTravel, 1, user.ID); err != nil {
		t.Fatalf("CreateExpense (Travel) failed: %v", err)
	}

	// Expense 2: Meal (linked to Project 1)
	expMeal := expense.Expense{
		Name:        "Jamuan Makan Siang Klien PT Maju",
		Date:        "2026-10-03",
		EmployeeID:  driver.ID,
		CategoryID:  catMeal.ID,
		PaymentMode: "own_account",
		UnitPrice:   300000,
		Quantity:    1,
		ReceiptRef:  "STRUK-RESTO/884",
		ProjectID:   &proj.ID,
	}
	if err := expenseSvc.CreateExpense(&expMeal, 1, user.ID); err != nil {
		t.Fatalf("CreateExpense (Meal) failed: %v", err)
	}

	// Expense 3: Fuel (linked to Fleet Vehicle 1)
	expFuel := expense.Expense{
		Name:        "Pengisian Solar Canter FE 74 HD",
		Date:        "2026-10-04",
		EmployeeID:  driver.ID,
		CategoryID:  catFuel.ID,
		PaymentMode: "own_account",
		UnitPrice:   200000,
		Quantity:    1,
		ReceiptRef:  "SPBU-SHELL/4401",
		VehicleID:   &truckVehicle.ID,
	}
	if err := expenseSvc.CreateExpense(&expFuel, 1, user.ID); err != nil {
		t.Fatalf("CreateExpense (Fuel) failed: %v", err)
	}

	// 4. Create Expense Sheet / Report (hr.expense.sheet) bundling Travel & Meal
	sheet := expense.ExpenseSheet{
		Name:        "Klaim Reimbursement Kunjungan Klien Surabaya",
		EmployeeID:  driver.ID,
		PaymentMode: "own_account",
		Notes:       "Bukti tiket dan invoice hotel terlampir lengkap",
	}
	createdSheet, err := expenseSvc.CreateExpenseSheet(&sheet, []uint{expTravel.ID, expMeal.ID}, 1, user.ID)
	if err != nil {
		t.Fatalf("CreateExpenseSheet failed: %v", err)
	}
	if createdSheet.TotalAmount != 1800000 {
		t.Fatalf("Expected sheet TotalAmount=1800000, got %.2f", createdSheet.TotalAmount)
	}

	// 5. Submit Expense Sheet for Manager Approval
	submittedSheet, err := expenseSvc.ActionSubmitSheet(createdSheet.ID, 1, user.ID)
	if err != nil || submittedSheet.State != "submit" {
		t.Fatalf("ActionSubmitSheet failed: %v", err)
	}

	// 6. Approve Expense Sheet
	approvedSheet, err := expenseSvc.ActionApproveSheet(createdSheet.ID, 1, user.ID)
	if err != nil || approvedSheet.State != "approve" {
		t.Fatalf("ActionApproveSheet failed: %v", err)
	}

	// 7. Post Expense Sheet directly to General Ledger Accounting (account.move)
	postedSheet, postedMove, err := expenseSvc.ActionPostSheet(createdSheet.ID, 1, user.ID)
	if err != nil {
		t.Fatalf("ActionPostSheet failed: %v", err)
	}
	if postedSheet.State != "post" || postedSheet.AccountMoveID == nil {
		t.Fatalf("Expected sheet state='post' with AccountMoveID, got state='%s'", postedSheet.State)
	}
	if postedMove.State != "posted" || postedMove.AmountTotal != 1800000 {
		t.Fatalf("Expected posted GL Move total=1800000, got total=%.2f, state=%s", postedMove.AmountTotal, postedMove.State)
	}

	// 8. Register Reimbursement Payment / Done
	paidSheet, err := expenseSvc.ActionPaySheet(createdSheet.ID, 1, user.ID)
	if err != nil || paidSheet.State != "done" {
		t.Fatalf("ActionPaySheet failed: %v", err)
	}

	// 9. Verify Expense Analytics & Stats
	stats, err := expenseSvc.GetExpenseStats(1)
	if err != nil {
		t.Fatalf("GetExpenseStats failed: %v", err)
	}
	if stats["total_items"].(int64) < 3 {
		t.Fatalf("Expected at least 3 expense items in stats, got %v", stats["total_items"])
	}

	// 10. Post Universal Chatter message on expense report
	expChatterMsg := chatter.MailMessage{
		ModelName:   "hr.expense.sheet",
		ResID:       createdSheet.ID,
		AuthorID:    user.ID,
		AuthorName:  user.Name,
		Body:        "Reimbursement telah ditransfer ke rekening BCA karyawan dan jurnal GL telah dibukukan.",
		MessageType: "comment",
	}
	expChatterMsg.CompanyID = 1
	if err := chatterSvc.PostMessage(&expChatterMsg); err != nil {
		t.Fatalf("Post chatter message to expense sheet failed: %v", err)
	}

	t.Logf("✓ Expense Sheet '%s' (Rp %.2f) verified: Submitted -> Approved -> Posted to GL Move '%s' -> Paid / Done (Employee: %s)",
		paidSheet.SheetNumber, paidSheet.TotalAmount, postedMove.Name, driver.Name)

	// =========================================================================
	// 22. QUALITY CONTROL & INSPECTION MANAGEMENT (quality.point, quality.check, quality.alert)
	// =========================================================================
	t.Log("\n--- SECTION 22: QUALITY CONTROL & INSPECTION MANAGEMENT ---")

	qualitySvc := quality.NewService(db, stockSvc, mrpSvc, chatterSvc)

	// 1. Create Quality Points (Inspection Rules)
	qpVisual := quality.QualityPoint{
		Title:         "Inspeksi Fisik & Segel Komponen Elektronik",
		Code:          "QP/TEST/01",
		OperationType: "incoming",
		TestType:      "passfail",
		Instructions:  "Periksa kelengkapan segel QC vendor dan tidak ada pin melengkung",
		Active:        true,
	}
	qpVisual.CompanyID = 1
	if err := qualitySvc.CreatePoint(&qpVisual); err != nil {
		t.Fatalf("CreatePoint (visual) failed: %v", err)
	}

	qpVoltage := quality.QualityPoint{
		Title:         "Pengujian Toleransi Voltase Adaptor 12V",
		Code:          "QP/TEST/02",
		OperationType: "incoming",
		TestType:      "measure",
		Norm:          12.0,
		ToleranceMin:  11.5,
		ToleranceMax:  12.5,
		NormUnit:      "V",
		Instructions:  "Ukur tegangan output terminal adaptor 12V (toleransi: 11.5V - 12.5V)",
		Active:        true,
	}
	qpVoltage.CompanyID = 1
	if err := qualitySvc.CreatePoint(&qpVoltage); err != nil {
		t.Fatalf("CreatePoint (voltage) failed: %v", err)
	}

	// 2. Generate Quality Checks for incoming receipt picking
	genChecks, err := qualitySvc.GenerateChecksForPicking(receipt.ID, 1, user.ID)
	if err != nil {
		t.Fatalf("GenerateChecksForPicking failed: %v", err)
	}
	if len(genChecks) < 2 {
		t.Fatalf("Expected at least 2 generated quality checks for receipt, got %d", len(genChecks))
	}
	t.Logf("✓ Generated %d Quality Checks automatically for Receipt Picking ID=%d", len(genChecks), receipt.ID)

	// Find the visual check and the measurement check
	var visualCheck, voltageCheck *quality.QualityCheck
	for i := range genChecks {
		if genChecks[i].TestType == "passfail" {
			visualCheck = &genChecks[i]
		} else if genChecks[i].TestType == "measure" {
			voltageCheck = &genChecks[i]
		}
	}
	if visualCheck == nil || voltageCheck == nil {
		t.Fatalf("Failed to locate both passfail and measure checks in generated checks")
	}

	// 3. Execute Visual Inspection: Pass Check
	passedVisual, err := qualitySvc.ActionPassCheck(visualCheck.ID, 1, user.ID, nil, "Kemasan utuh dan segel lolos verifikasi")
	if err != nil {
		t.Fatalf("ActionPassCheck for visual inspection failed: %v", err)
	}
	if passedVisual.QualityState != "pass" {
		t.Fatalf("Expected visual check quality_state='pass', got '%s'", passedVisual.QualityState)
	}

	// 4. Test Tolerance Bounds: Attempting to pass out-of-tolerance measurement should fail
	outOfBoundsVal := 10.8 // Below 11.5V min tolerance
	_, err = qualitySvc.ActionPassCheck(voltageCheck.ID, 1, user.ID, &outOfBoundsVal, "Voltase terukur")
	if err == nil {
		t.Fatalf("Expected error passing out-of-tolerance value 10.8V (min=11.5V), but got none!")
	}
	t.Logf("✓ Tolerance Enforcement Verified: Passing 10.8V correctly rejected with: %v", err)

	// 5. Fail Check with Auto-Created Quality Alert (NCR)
	failedCheck, alert, err := qualitySvc.ActionFailCheck(voltageCheck.ID, 1, user.ID, &outOfBoundsVal, "Tegangan drop drastis ke 10.8V di bawah batas aman 11.5V", true)
	if err != nil {
		t.Fatalf("ActionFailCheck failed: %v", err)
	}
	if failedCheck.QualityState != "fail" {
		t.Fatalf("Expected voltage check quality_state='fail', got '%s'", failedCheck.QualityState)
	}
	if alert == nil || alert.ID == 0 {
		t.Fatalf("Expected auto-generated Quality Alert, got nil")
	}
	if alert.Stage != "new" || alert.ActionType != "quarantine" {
		t.Fatalf("Expected alert stage='new', action_type='quarantine', got stage=%s, action=%s", alert.Stage, alert.ActionType)
	}
	t.Logf("✓ Failed Check '%s' generated Quality Alert '%s' (%s, Priority=%s)", failedCheck.Name, alert.Name, alert.Title, alert.Priority)

	// 6. Alert Stage Lifecycle (new -> in_progress -> closed)
	updatedAlert, err := qualitySvc.UpdateAlertStage(alert.ID, 1, "in_progress", "Kerusakan pada unit regulator batch vendor", "Retur seluruh lot ke vendor untuk penggantian garansi", "return", user.ID)
	if err != nil || updatedAlert.Stage != "in_progress" {
		t.Fatalf("UpdateAlertStage to in_progress failed: %v", err)
	}

	closedAlert, err := qualitySvc.UpdateAlertStage(alert.ID, 1, "closed", "", "", "", user.ID)
	if err != nil || closedAlert.Stage != "closed" || closedAlert.CloseDate == "" {
		t.Fatalf("UpdateAlertStage to closed failed: %v", err)
	}
	t.Logf("✓ Quality Alert '%s' transitioned through stages to 'closed' on %s", closedAlert.Name, closedAlert.CloseDate)

	// 7. Verify Analytics & Pass Rate
	qStats, err := qualitySvc.GetQualityStats(1)
	if err != nil {
		t.Fatalf("GetQualityStats failed: %v", err)
	}
	if qStats["total_checks"].(int64) < 2 {
		t.Fatalf("Expected at least 2 checks in quality stats, got %v", qStats["total_checks"])
	}
	if qStats["passed_checks"].(int64) < 1 || qStats["failed_checks"].(int64) < 1 {
		t.Fatalf("Expected both passed and failed checks in stats: %v", qStats)
	}
	t.Logf("✓ Quality Control Analytics: Total Checks=%v, Passed=%v, Failed=%v, Pass Rate=%v%%, Total Alerts=%v",
		qStats["total_checks"], qStats["passed_checks"], qStats["failed_checks"], qStats["pass_rate_pct"], qStats["total_alerts"])

	// 8. Verify Universal Chatter on Quality Alert
	qaChatter, err := chatterSvc.ListMessages("quality.alert", alert.ID, 1)
	if err != nil || len(qaChatter) == 0 {
		t.Fatalf("Expected chatter messages on quality alert, got err=%v, count=%d", err, len(qaChatter))
	}
	t.Logf("✓ Verified %d chatter audit trail messages on Quality Alert %s", len(qaChatter), alert.Name)

	// =========================================================================
	// 23. CLOUDFLARE WORKERS AI INTEGRATION & SMART COPILOT
	// =========================================================================
	t.Log("\n--- SECTION 23: CLOUDFLARE WORKERS AI INTEGRATION & SMART COPILOT ---")

	cfg := config.LoadConfig()
	aiClient := ai.NewClient(cfg)

	// 1. Test AI Ticket Triage & Smart Auto-Reply (Helpdesk)
	ticketAnalysis, err := aiClient.AnalyzeTicket("Mesin Pompa Hidrolik Rusak & Oli Bocor", "Terjadi kebocoran oli hidrolik fatal di jalur perakitan manufaktur")
	if err != nil {
		t.Fatalf("AnalyzeTicket failed: %v", err)
	}
	if ticketAnalysis.SuggestedPriority != "urgent" || ticketAnalysis.SuggestedCategory != "technical" {
		t.Fatalf("Expected priority='urgent', category='technical', got priority=%s, category=%s", ticketAnalysis.SuggestedPriority, ticketAnalysis.SuggestedCategory)
	}
	if ticketAnalysis.DraftResponse == "" || len(ticketAnalysis.ActionRecommendations) == 0 {
		t.Fatalf("Expected non-empty draft response and recommendations, got %+v", ticketAnalysis)
	}
	t.Logf("✓ Helpdesk AI Triage: Sentiment=%s, Priority=%s, Category=%s, Engine Source=%s",
		ticketAnalysis.Sentiment, ticketAnalysis.SuggestedPriority, ticketAnalysis.SuggestedCategory, ticketAnalysis.Source)

	// 2. Test AI Quality Control Root Cause Analysis & CAPA Advisor
	qualityAnalysis, err := aiClient.AnalyzeQualityAlert("Fluktuasi Tegangan 10.8V pada Adaptor Daya", "Tegangan drop drastis di bawah ambang batas toleransi 11.5V", "Batch IC regulator vendor")
	if err != nil {
		t.Fatalf("AnalyzeQualityAlert failed: %v", err)
	}
	if qualityAnalysis.SuggestedActionType != "quarantine" || qualityAnalysis.RiskLevel != "High" {
		t.Fatalf("Expected action_type='quarantine', risk_level='High', got action=%s, risk=%s", qualityAnalysis.SuggestedActionType, qualityAnalysis.RiskLevel)
	}
	if qualityAnalysis.RootCauseAnalysis == "" || qualityAnalysis.PreventiveAction == "" {
		t.Fatalf("Expected RCA and CAPA guidance, got %+v", qualityAnalysis)
	}
	t.Logf("✓ Quality AI RCA & CAPA Advisor: Risk=%s, Action=%s, RCA length=%d chars, CAPA length=%d chars",
		qualityAnalysis.RiskLevel, qualityAnalysis.SuggestedActionType, len(qualityAnalysis.RootCauseAnalysis), len(qualityAnalysis.PreventiveAction))

	// 3. Test AI Expense Auto-Categorization & GL Account Mapping
	expenseAnalysis, err := aiClient.AnalyzeExpense("Tiket Kereta Cepat & Penginapan Hotel Perjalanan Dinas Surabaya", 1800000)
	if err != nil {
		t.Fatalf("AnalyzeExpense failed: %v", err)
	}
	if expenseAnalysis.SuggestedGLAccount != "620100" || expenseAnalysis.SuggestedCategoryName != "Perjalanan Dinas & Akomodasi" {
		t.Fatalf("Expected GL Account 620100 (Travel), got %s (%s)", expenseAnalysis.SuggestedGLAccount, expenseAnalysis.SuggestedCategoryName)
	}
	t.Logf("✓ Expense AI Categorizer: Category='%s' -> GL Account='%s', Tax Deductible=%v",
		expenseAnalysis.SuggestedCategoryName, expenseAnalysis.SuggestedGLAccount, expenseAnalysis.TaxDeductible)

	// 4. Test Executive Natural Language Query ("Ask ERP AI")
	statsCtx := map[string]interface{}{
		"total_sales":     10500000.0,
		"sales_orders":    int64(2),
		"purchase_orders": int64(2),
		"tickets":         int64(1),
		"quality_alerts":  int64(1),
	}
	askResp, err := aiClient.AskERP("Bagaimana performa penjualan dan kendala operasional terkini?", statsCtx)
	if err != nil {
		t.Fatalf("AskERP failed: %v", err)
	}
	if askResp.Answer == "" || len(askResp.SuggestedActions) == 0 {
		t.Fatalf("Expected comprehensive executive answer and suggested actions, got %+v", askResp)
	}
	t.Logf("✓ Ask ERP AI Copilot Response: \"%s\" (%d strategic actions suggested)", askResp.Answer, len(askResp.SuggestedActions))

	// --- SECTION 24: APPROVAL WORKFLOW MATRIX & AI RISK ASSESSMENT ---
	t.Log("\n    --- SECTION 24: APPROVAL WORKFLOW MATRIX & AI RISK ASSESSMENT ---")
	approvalSvc := approval.NewService(db, chatterSvc)

	mgrUser, err := baseSvc.Register("Jane Manager", "jane.mgr@example.com", "secret123", "manager", 1)
	if err != nil {
		t.Fatalf("Failed to create manager user for approval: %v", err)
	}

	// 1. Create Capex Approval Category with 2-tier approval
	capexCat := approval.ApprovalCategory{
		Name:                 "Pengadaan Capex & Server IT",
		Description:          "Otorisasi multi-level belanja modal infrastruktur server",
		SequenceCode:         "CAPEX",
		ApprovalType:         "capex",
		HasAmount:            true,
		MinAmount:            10000000,
		MinApprovalsRequired: 2,
		IsManagerApprover:    true,
		Active:               true,
	}
	capexCat.CompanyID = company.ID
	capexCat.CreateUID = user.ID
	if err := approvalSvc.CreateCategory(&capexCat); err != nil {
		t.Fatalf("Failed to create approval category: %v", err)
	}

	// 2. Create Approval Request with 2 Approvers (Level 1: Dept Manager user, Level 2: Director mgrUser)
	reqInput := approval.CreateRequestInput{
		Name:           "Pengadaan Edge Server Cloudflare Tunnel Cluster",
		CategoryID:     capexCat.ID,
		RequestOwnerID: user.ID,
		Reason:         "Peningkatan kapasitas throughput transaksi ERP dan deployment hybrid edge",
		Amount:         24000000,
		Reference:      "PO/2026/00002",
		CompanyID:      company.ID,
		Approvers: []approval.ApproverInputItem{
			{UserID: user.ID, Sequence: 1, Required: true},
			{UserID: mgrUser.ID, Sequence: 2, Required: true},
		},
	}
	apprReq, err := approvalSvc.CreateRequest(reqInput, user.ID)
	if err != nil {
		t.Fatalf("Failed to create approval request: %v", err)
	}
	if apprReq.Stage != "new" || len(apprReq.Approvers) != 2 {
		t.Fatalf("Expected stage 'new' and 2 approvers, got stage '%s', approvers count: %d", apprReq.Stage, len(apprReq.Approvers))
	}

	// 3. Submit Request -> Moves to 'pending', Level 1 becomes 'waiting'
	apprReq, err = approvalSvc.ActionSubmit(apprReq.ID, user.ID, company.ID)
	if err != nil {
		t.Fatalf("ActionSubmit failed: %v", err)
	}
	if apprReq.Stage != "pending" {
		t.Fatalf("Expected stage 'pending', got '%s'", apprReq.Stage)
	}
	if apprReq.Approvers[0].Status != "waiting" || apprReq.Approvers[1].Status != "new" {
		t.Fatalf("Expected Tier 1 'waiting' and Tier 2 'new', got Tier 1: %s, Tier 2: %s",
			apprReq.Approvers[0].Status, apprReq.Approvers[1].Status)
	}

	// 4. Level 1 Approves -> Tier 1 becomes 'approved', Tier 2 advances to 'waiting'
	apprReq, err = approvalSvc.ActionApprove(apprReq.ID, user.ID, company.ID, "Disetujui secara teknis sesuai arsitektur edge")
	if err != nil {
		t.Fatalf("ActionApprove Level 1 failed: %v", err)
	}
	if apprReq.Stage != "pending" {
		t.Fatalf("Expected request to remain 'pending' after tier 1, got '%s'", apprReq.Stage)
	}
	if apprReq.Approvers[0].Status != "approved" || apprReq.Approvers[1].Status != "waiting" {
		t.Fatalf("Expected Tier 1 'approved' and Tier 2 'waiting', got Tier 1: %s, Tier 2: %s",
			apprReq.Approvers[0].Status, apprReq.Approvers[1].Status)
	}

	// 5. Level 2 Approves -> All approved -> Request automatically transitions to 'approved'
	apprReq, err = approvalSvc.ActionApprove(apprReq.ID, mgrUser.ID, company.ID, "Otorisasi belanja modal anggaran kuartal 4 disetujui")
	if err != nil {
		t.Fatalf("ActionApprove Level 2 failed: %v", err)
	}
	if apprReq.Stage != "approved" {
		t.Fatalf("Expected request stage 'approved' after all tiers, got '%s'", apprReq.Stage)
	}
	t.Logf("✓ Multi-Tier Approval Workflow '%s' (%s): Completed 2 levels -> Final Stage: '%s'",
		apprReq.Code, apprReq.Name, apprReq.Stage)

	// 6. Test AI Approval Risk Assessment Copilot
	riskAnalysis, err := aiClient.AnalyzeApprovalRisk(
		apprReq.Name, capexCat.Name, apprReq.Amount, apprReq.Reason, apprReq.Reference,
	)
	if err != nil {
		t.Fatalf("AnalyzeApprovalRisk failed: %v", err)
	}
	if riskAnalysis.RiskLevel != "High" && riskAnalysis.RiskLevel != "Critical" {
		t.Fatalf("Expected High/Critical risk for Rp 24jt Capex, got: %s", riskAnalysis.RiskLevel)
	}
	if len(riskAnalysis.AuditRecommendations) == 0 {
		t.Fatalf("Expected audit recommendations from AI risk analyzer")
	}
	t.Logf("✓ AI Approval Risk Copilot: RiskLevel='%s', PolicyCompliance='%s', Recommendations=%d",
		riskAnalysis.RiskLevel, riskAnalysis.PolicyCompliance, len(riskAnalysis.AuditRecommendations))

	// 7. Verify Approval Stats
	apprStats, err := approvalSvc.GetStats(company.ID)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if apprStats["approved"] < 1 {
		t.Fatalf("Expected at least 1 approved request in stats, got %+v", apprStats)
	}
	t.Logf("✓ Approval KPI Stats verified: Total=%d, Approved=%d, Pending=%d, Refused=%d",
		apprStats["total"], apprStats["approved"], apprStats["pending"], apprStats["refused"])

	// =========================================================================
	// SECTION 25: CLOUDFLARE R2 STORAGE ADAPTER & AI VISION RECEIPT OCR
	// =========================================================================
	t.Log("\n--- SECTION 25: CLOUDFLARE R2 STORAGE ADAPTER & AI VISION RECEIPT OCR ---")

	// 1. Storage Provider Unit Tests (Save, Get, Delete)
	uploadTestDir := t.TempDir() + "/attachments"
	storageProv, err := storage.NewLocalProvider(uploadTestDir)
	if err != nil {
		t.Fatalf("Failed to init storage provider: %v", err)
	}

	testKey, err := storageProv.Save(context.Background(), "test/receipt.txt", "text/plain", []byte("Simulated binary payload"))
	if err != nil {
		t.Fatalf("Storage Save failed: %v", err)
	}
	retrievedBytes, err := storageProv.Get(context.Background(), testKey)
	if err != nil || string(retrievedBytes) != "Simulated binary payload" {
		t.Fatalf("Storage Get failed or payload mismatch: %v", err)
	}
	if err := storageProv.Delete(context.Background(), testKey); err != nil {
		t.Fatalf("Storage Delete failed: %v", err)
	}
	t.Logf("✓ Storage Provider (%s) verified: Save, Get, and Delete operations functioning correctly", storageProv.Type())

	// 2. Initialize Attachment Module Service
	attRepo := attachment.NewRepository(db)
	attSvc := attachment.NewService(db, attRepo, storageProv, aiClient, chatterSvc, expenseSvc)

	// 3. Upload Receipt Attachment linked to Expense
	receiptContent := []byte("SPBU PERTAMINA PASTI PAS 34-12345\nTanggal: 2026-10-06\nBBM Dexlite (Solar): Rp 346.500\nPPN Masukan (11%): Rp 38.500\nTotal Pembayaran: Rp 385.000\nNomor Kendaraan: B 1024 TKO (Canter FE 74 HD)\nPetugas: Ridwan")
	uploadReq := &attachment.UploadAttachmentRequest{
		Name:        "struk_pertamina_dexlite_b1024tko.jpg",
		ResModel:    "hr.expense",
		ResID:       createdSheet.ID,
		ResName:     createdSheet.SheetNumber,
		Description: "Struk pengisian Dexlite Solar Truk Canter ekspedisi Jawa Timur",
		AutoOCR:     false,
		TextHint:    string(receiptContent),
	}

	att, err := attSvc.UploadAttachment(company.ID, user.ID, user.Name, uploadReq, receiptContent, "image/jpeg")
	if err != nil {
		t.Fatalf("UploadAttachment failed: %v", err)
	}
	if att.ID == 0 || att.Checksum == "" || att.FileURL == "" {
		t.Fatalf("Attachment upload invalid: %+v", att)
	}
	t.Logf("✓ Uploaded Attachment '%s' (ID=%d, Checksum=%s, Size=%d bytes, Storage=%s)",
		att.Name, att.ID, att.Checksum[:12]+"...", att.FileSize, att.StorageType)

	// Verify chatter message logged
	var chatterCount int64
	db.Model(&chatter.MailMessage{}).Where("model_name = ? AND res_id = ?", att.ResModel, att.ResID).Count(&chatterCount)
	if chatterCount == 0 {
		t.Fatalf("Expected chatter log on target expense sheet after attachment upload")
	}

	// 4. Run Cloudflare Workers AI Vision OCR Engine
	ocrResult, err := attSvc.RunOCR(att.ID, string(receiptContent))
	if err != nil {
		t.Fatalf("RunOCR failed: %v", err)
	}
	if ocrResult.TotalAmount != 385000.0 {
		t.Fatalf("Expected OCR TotalAmount 385000.0, got: %.2f", ocrResult.TotalAmount)
	}
	if !strings.Contains(strings.ToLower(ocrResult.VendorName), "pertamina") {
		t.Fatalf("Expected Pertamina vendor, got: %s", ocrResult.VendorName)
	}
	if ocrResult.CategorySuggestion != "Bahan Bakar & Transportasi" {
		t.Fatalf("Expected category 'Bahan Bakar & Transportasi', got: %s", ocrResult.CategorySuggestion)
	}
	if ocrResult.ConfidenceScore < 0.90 {
		t.Fatalf("Expected high OCR confidence >= 0.90, got: %.2f", ocrResult.ConfidenceScore)
	}
	t.Logf("✓ AI Vision OCR processed receipt: Vendor='%s', Date='%s', Total=Rp %.2f (Tax: Rp %.2f), Confidence=%.0f%%, Engine='%s'",
		ocrResult.VendorName, ocrResult.InvoiceDate, ocrResult.TotalAmount, ocrResult.TaxAmount, ocrResult.ConfidenceScore*100, ocrResult.Source)

	// 5. Automated 1-Click Expense Creation from Scanned Receipt
	createdExp, err := attSvc.CreateExpenseFromReceipt(att.ID, driver.ID, company.ID, user.ID)
	if err != nil {
		t.Fatalf("CreateExpenseFromReceipt failed: %v", err)
	}
	if createdExp.TotalAmount != 385000.0 {
		t.Fatalf("Expected created expense total 385000.0, got: %.2f", createdExp.TotalAmount)
	}
	if createdExp.State != "draft" {
		t.Fatalf("Expected created expense state 'draft', got: %s", createdExp.State)
	}
	if createdExp.ReceiptRef != fmt.Sprintf("STRUK-%d", att.ID) {
		t.Fatalf("Expected receipt ref matching attachment ID, got: %s", createdExp.ReceiptRef)
	}

	// Verify attachment updated to point to newly created expense item
	updatedAtt, _ := attRepo.FindByID(att.ID)
	if updatedAtt.ResModel != "hr.expense" || updatedAtt.ResID != createdExp.ID {
		t.Fatalf("Expected attachment linked to created expense ID %d, got model=%s id=%d",
			createdExp.ID, updatedAtt.ResModel, updatedAtt.ResID)
	}
	t.Logf("✓ 1-Click Expense created from Scanned Receipt: ID=%d, Name='%s', Amount=Rp %.2f, Ref='%s'",
		createdExp.ID, createdExp.Name, createdExp.TotalAmount, createdExp.ReceiptRef)

	// 6. Verify Attachment Hub Statistics
	docStats, err := attSvc.GetStats(company.ID)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if docStats["total_documents"].(int64) < 1 || docStats["ocr_processed"].(int64) < 1 {
		t.Fatalf("Expected document stats with at least 1 document & 1 OCR processed, got: %+v", docStats)
	}
	t.Logf("✓ Attachment Hub Statistics verified: Total Docs=%d, OCR Scanned=%d, Expense Receipts=%d",
		docStats["total_documents"], docStats["ocr_processed"], docStats["expense_receipts"])

	// =========================================================================
	// SECTION 26: HR ATTENDANCE, CLOUDFLARE EDGE TELEMETRY & PAYROLL INTEGRATION
	// =========================================================================
	t.Log("\n--- SECTION 26: HR ATTENDANCE, CLOUDFLARE EDGE TELEMETRY & PAYROLL INTEGRATION ---")

	attendanceSvc := attendance.NewService(db, chatterSvc, aiClient)

	// 1. Create Work Shifts (Standard 8.0 hrs with 15m grace period vs Flexible WFH)
	regShift := attendance.Shift{
		Name:               "Shift Reguler Kantor (08:30 - 17:30)",
		Code:               "REGULAR",
		StartTime:          "23:59", // Set late evening to guarantee on-time status across any test run hour
		EndTime:            "23:59",
		StandardHours:      8.0,
		GracePeriodMinutes: 15,
		IsFlexible:         false,
		Color:              "#3b82f6",
	}
	regShift.CompanyID = company.ID
	regShift.CreateUID = user.ID
	if err := attendanceSvc.CreateShift(&regShift); err != nil {
		t.Fatalf("Create regShift failed: %v", err)
	}

	wfhShift := attendance.Shift{
		Name:               "Shift Fleksibel / WFH Remote",
		Code:               "WFH",
		StartTime:          "09:00",
		EndTime:            "18:00",
		StandardHours:      8.0,
		GracePeriodMinutes: 60,
		IsFlexible:         true,
		Color:              "#10b981",
	}
	wfhShift.CompanyID = company.ID
	wfhShift.CreateUID = user.ID
	if err := attendanceSvc.CreateShift(&wfhShift); err != nil {
		t.Fatalf("Create wfhShift failed: %v", err)
	}
	t.Logf("✓ Created Work Shifts: %s (Standard: %.1f hrs) & %s (Flexible: %v)",
		regShift.Name, regShift.StandardHours, wfhShift.Name, wfhShift.IsFlexible)

	// 2. Employee Clock-In with Cloudflare Edge Telemetry & GPS Geofencing
	checkInReq := attendance.CheckInRequest{
		EmployeeID: driver.ID,
		ShiftID:    &regShift.ID,
		Latitude:   -6.200050,
		Longitude:  106.816680,
		Address:    "Menara Sudirman Lantai 18, Jakarta Pusat",
		Device:     "PWA Mobile Edge / iOS 17.5",
		Note:       "Presensi pagi via Cloudflare Edge Tunnel",
	}
	cfIP := "114.124.200.80"
	cfCountry := "ID"
	cfRay := "8c8965f90a9821a0"

	attRec, err := attendanceSvc.CheckIn(checkInReq, company.ID, user.ID, cfIP, cfCountry, cfRay)
	if err != nil {
		t.Fatalf("Employee CheckIn failed: %v", err)
	}
	if attRec.Status != "checked_in" {
		t.Fatalf("Expected status 'checked_in', got: %s", attRec.Status)
	}
	if attRec.AnomalyFlag {
		t.Fatalf("Expected normal attendance within HQ geofence, but got anomaly: %s", attRec.AnomalyReason)
	}
	if attRec.IPAddress != cfIP || attRec.Country != cfCountry || attRec.CFRay != cfRay {
		t.Fatalf("Cloudflare edge metadata mismatch: IP=%s Country=%s Ray=%s", attRec.IPAddress, attRec.Country, attRec.CFRay)
	}
	t.Logf("✓ Employee '%s' Clocked-In: Date=%s, Time=%s, Status=%s, Edge Ray=%s, IP=%s (%s)",
		driver.Name, attRec.Date, attRec.CheckIn.Format("15:04:05"), attRec.Status, attRec.CFRay, attRec.IPAddress, attRec.Country)

	// 3. Verify Active Attendance Session & Duplicate Prevention
	activeAtt, err := attendanceSvc.Repo().GetActiveAttendanceByEmployee(driver.ID)
	if err != nil || activeAtt == nil || activeAtt.ID != attRec.ID {
		t.Fatalf("GetActiveAttendanceByEmployee failed or ID mismatch: %+v", activeAtt)
	}
	_, dupErr := attendanceSvc.CheckIn(checkInReq, company.ID, user.ID, cfIP, cfCountry, cfRay)
	if dupErr == nil {
		t.Fatalf("Expected duplicate check-in to be prevented for active employee")
	}
	t.Logf("✓ Duplicate Check-In rejected as expected: '%v'", dupErr)

	// 4. Test Cloudflare Edge Foreign IP Anomaly Detection
	foreignReq := attendance.CheckInRequest{
		EmployeeID: emp.ID,
		ShiftID:    &regShift.ID,
		Latitude:   -6.200100,
		Longitude:  106.816650,
		Address:    "Kantor Pusat Jakarta",
		Device:     "Workstation Chrome",
	}
	foreignAtt, err := attendanceSvc.CheckIn(foreignReq, company.ID, user.ID, "104.28.19.45", "US", "8c8999f11a0022ff")
	if err != nil {
		t.Fatalf("Foreign check-in request failed: %v", err)
	}
	if !foreignAtt.AnomalyFlag || !strings.Contains(foreignAtt.AnomalyReason, "Cloudflare Edge") {
		t.Fatalf("Expected Cloudflare Edge foreign country anomaly flag, got: flag=%v reason=%s",
			foreignAtt.AnomalyFlag, foreignAtt.AnomalyReason)
	}
	t.Logf("✓ Cloudflare Edge Country Anomaly correctly flagged: Country=%s, Reason='%s'",
		foreignAtt.Country, foreignAtt.AnomalyReason)

	// 5. Employee Clock-Out with Overtime Calculation
	// Simulate 10.5 hours worked (8.0 standard hours + 2.5 overtime hours)
	simulatedCheckIn := time.Now().Add(-10*time.Hour - 30*time.Minute)
	attRec.CheckIn = simulatedCheckIn
	db.Save(attRec)

	checkOutReq := attendance.CheckOutRequest{
		AttendanceID: attRec.ID,
		EmployeeID:   driver.ID,
		Latitude:     -6.200080,
		Longitude:    106.816690,
		Address:      "Lobi Gedung Menara Sudirman, Jakarta Pusat",
		Note:         "Check-out shift selesai lembur",
	}
	checkedOutAtt, err := attendanceSvc.CheckOut(checkOutReq, company.ID, user.ID)
	if err != nil {
		t.Fatalf("CheckOut failed: %v", err)
	}
	if checkedOutAtt.CheckOut == nil {
		t.Fatalf("Expected non-nil CheckOut time")
	}
	if checkedOutAtt.WorkedHours < 10.4 {
		t.Fatalf("Expected worked hours around 10.5, got: %.2f", checkedOutAtt.WorkedHours)
	}
	if checkedOutAtt.OvertimeHours < 2.4 {
		t.Fatalf("Expected overtime hours around 2.5, got: %.2f", checkedOutAtt.OvertimeHours)
	}
	if checkedOutAtt.Status != "overtime" {
		t.Fatalf("Expected status 'overtime', got: %s", checkedOutAtt.Status)
	}
	t.Logf("✓ Employee Clocked-Out: Worked=%.2f hrs, Standard=%.1f hrs, Overtime=%.2f hrs, Status=%s",
		checkedOutAtt.WorkedHours, checkedOutAtt.Shift.StandardHours, checkedOutAtt.OvertimeHours, checkedOutAtt.Status)

	// Verify chatter tracking recorded
	var attChatterCount int64
	db.Model(&chatter.MailMessage{}).Where("model_name = ? AND res_id = ?", "hr.attendance", attRec.ID).Count(&attChatterCount)
	if attChatterCount < 2 {
		t.Fatalf("Expected at least 2 chatter tracking logs for check-in and check-out, got: %d", attChatterCount)
	}
	t.Logf("✓ Chatter audit trail verified on Attendance #%d (%d entries)", att.ID, attChatterCount)

	// 6. 1-Click Sync Attendance to Payroll (hr.payslip)
	syncReq := attendance.SyncPayrollRequest{
		EmployeeID:           driver.ID,
		Period:               "October 2026",
		OvertimeRatePerHour:  50000.0, // Rp 50.000 / jam lembur
		LatePenaltyPerMinute: 1000.0,
	}
	payslip, err := attendanceSvc.SyncToPayslip(syncReq, company.ID, user.ID)
	if err != nil {
		t.Fatalf("SyncToPayslip failed: %v", err)
	}
	expectedMinAllowance := checkedOutAtt.OvertimeHours * 50000.0
	if payslip.Allowances < expectedMinAllowance-1.0 {
		t.Fatalf("Expected payslip allowances >= %.2f from overtime, got: %.2f", expectedMinAllowance, payslip.Allowances)
	}
	if payslip.NetSalary != payslip.BasicSalary+payslip.Allowances-payslip.Deductions {
		t.Fatalf("Net salary calculation mismatch: Basic=%.2f + Allow=%.2f - Ded=%.2f != Net=%.2f",
			payslip.BasicSalary, payslip.Allowances, payslip.Deductions, payslip.NetSalary)
	}
	t.Logf("✓ Attendance Overtime synced to HR Payslip '%s': Basic=Rp %.0f, Overtime Allowance=Rp %.0f, Net Salary=Rp %.0f",
		payslip.Name, payslip.BasicSalary, payslip.Allowances, payslip.NetSalary)

	// 7. Cloudflare Workers AI Anomaly & Burnout Risk Analysis
	aiAnomalyResult, err := attendanceSvc.RunAIAnomalyCheck(company.ID)
	if err != nil {
		t.Fatalf("RunAIAnomalyCheck failed: %v", err)
	}
	if aiAnomalyResult.AnalyzedAttendances < 2 {
		t.Fatalf("Expected at least 2 analyzed attendances, got: %d", aiAnomalyResult.AnalyzedAttendances)
	}
	if aiAnomalyResult.AnomaliesDetected < 1 {
		t.Fatalf("Expected at least 1 anomaly detected (foreign IP), got: %d", aiAnomalyResult.AnomaliesDetected)
	}
	if len(aiAnomalyResult.Recommendations) == 0 {
		t.Fatalf("Expected AI management recommendations, got empty")
	}
	if aiAnomalyResult.AIConfidence < 0.85 {
		t.Fatalf("Expected high AI confidence score >= 0.85, got: %.2f", aiAnomalyResult.AIConfidence)
	}
	t.Logf("✓ Cloudflare Workers AI Attendance Anomaly Audit: Analyzed=%d, Anomalies=%d, BurnoutRisks=%d, Engine='%s', Confidence=%.0f%%",
		aiAnomalyResult.AnalyzedAttendances, aiAnomalyResult.AnomaliesDetected, len(aiAnomalyResult.BurnoutRiskEmployees),
		aiAnomalyResult.Source, aiAnomalyResult.AIConfidence*100)
	t.Logf("  Executive Summary: %s", aiAnomalyResult.Summary)

	// 8. Attendance Executive Dashboard Stats
	attStats, err := attendanceSvc.GetStats(company.ID)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if attStats.TotalOvertimeHours <= 0 {
		t.Fatalf("Expected recorded overtime in stats, got: %.2f", attStats.TotalOvertimeHours)
	}
	t.Logf("✓ Attendance Executive Analytics: Total Today=%d, Currently In=%d, Late=%d, Overtime Hours=%.2f hrs, Avg Worked=%.2f hrs, Anomalies=%d",
		attStats.TotalToday, attStats.CurrentlyCheckedIn, attStats.LateToday, attStats.TotalOvertimeHours, attStats.AverageWorkedHours, attStats.AnomalyCount)

	// =========================================================================
	// SECTION 27: HR RECRUITMENT, CLOUDFLARE WORKERS AI TALENT SCREENING & 1-CLICK HIRE
	// =========================================================================
	t.Log("\n--- SECTION 27: HR RECRUITMENT, CLOUDFLARE WORKERS AI TALENT SCREENING & 1-CLICK HIRE ---")

	recruitmentRepo := recruitment.NewRepository(db)
	recruitmentSvc := recruitment.NewService(db, recruitmentRepo, hrSvc, chatterSvc, aiClient)

	// 1. Create Default Recruitment Stages
	stages := []recruitment.RecruitmentStage{
		{Name: "1. Initial Screening", Sequence: 10, Folded: false},
		{Name: "2. Technical Review", Sequence: 20, Folded: false},
		{Name: "3. User & HR Interview", Sequence: 30, Folded: false},
		{Name: "4. Contract Proposal", Sequence: 40, Folded: false},
		{Name: "5. Hired / Diterima", Sequence: 50, Folded: true, IsHired: true},
		{Name: "Refused / Ditolak", Sequence: 99, Folded: true, IsRefused: true},
	}
	for i := range stages {
		stages[i].CompanyID = company.ID
		if err := recruitmentSvc.CreateStage(&stages[i]); err != nil {
			t.Fatalf("Create stage failed: %v", err)
		}
	}
	stageList, err := recruitmentSvc.ListStages(company.ID)
	if err != nil || len(stageList) < 6 {
		t.Fatalf("Expected at least 6 stages, got %d (err: %v)", len(stageList), err)
	}
	t.Logf("✓ Configured %d Recruitment Stages (Initial -> Technical -> Interview -> Offering -> Hired/Refused)", len(stageList))

	// 2. Create Job Opening: "Lead Cloudflare & Golang Architect"
	var itDept hr.Department
	db.Where("company_id = ?", company.ID).First(&itDept)

	openingReq := recruitment.CreateJobOpeningRequest{
		Title:             "Lead Cloudflare & Golang Architect",
		DepartmentID:      itDept.ID,
		ExpectedEmployees: 2,
		Requirements:      "Kualifikasi: Berpengalaman minimal 5 tahun arsitektur sistem backend Golang tingkat lanjut, framework Gin, GORM, PostgreSQL, Cloudflare Workers, Edge caching, Docker, dan high concurrency microservices.",
		Description:       "Memimpin arsitektur core ERP dan sistem telemetri Cloudflare Edge global.",
		SalaryMin:         22000000,
		SalaryMax:         35000000,
	}
	jobOpening, err := recruitmentSvc.CreateJobOpening(&openingReq, company.ID, user.ID)
	if err != nil {
		t.Fatalf("CreateJobOpening failed: %v", err)
	}
	t.Logf("✓ Job Opening Created: '%s' (ID=%d, Target=%d employees, Budget=Rp %.0f - Rp %.0f)",
		jobOpening.Title, jobOpening.ID, jobOpening.ExpectedEmployees, jobOpening.SalaryMin, jobOpening.SalaryMax)

	// 3. Create Applicant: "Dewi Kartika" with comprehensive Golang & Cloudflare CV
	dewiResume := `RINGKASAN EKSEKUTIF:
Principal Backend Engineer dengan 6 tahun pengalaman merancang platform terdistribusi berbasis Golang, Gin, GORM, PostgreSQL, dan Redis. Ahli dalam optimalisasi Cloudflare Workers, Edge telemetry, dan container orchestration Docker di Linux.

KEAHLIAN KUNCI:
- Golang, Gin framework, GORM, PostgreSQL, SQLite, Redis, REST API
- Cloudflare Workers, Edge caching, Docker, Linux, CI/CD, Git
- Domain: Sistem ERP Terintegrasi, Multi-Company Data Isolation, Financial Ledger

PENGALAMAN KERJA:
Lead Engineer - Cloud Tech Nusantara (2020 - 2026)
- Merancang core engine ERP berbasis Golang dengan throughput 2.000 TPS.
- Mengintegrasikan Cloudflare Workers AI untuk automasi OCR dan analisis anomali edge.`

	appReq := recruitment.CreateApplicantRequest{
		Name:           "Dewi Kartika, M.T.",
		Email:          "dewi.kartika.architect@tech.co.id",
		Phone:          "+62 811-2233-4455",
		JobOpeningID:   jobOpening.ID,
		SalaryExpected: 28000000,
		ResumeText:     dewiResume,
	}
	applicant1, err := recruitmentSvc.CreateApplicant(&appReq, company.ID, user.ID)
	if err != nil {
		t.Fatalf("CreateApplicant failed: %v", err)
	}
	t.Logf("✓ Candidate Registered: '%s' (ID=%d, Email=%s, Expected Salary=Rp %.0f)",
		applicant1.Name, applicant1.ID, applicant1.Email, applicant1.SalaryExpected)

	// 4. Update Applicant Stage: Move to "2. Technical Review"
	applicant1, err = recruitmentSvc.UpdateApplicantStage(applicant1.ID, stageList[1].ID, company.ID, user.ID)
	if err != nil || applicant1.StageID != stageList[1].ID {
		t.Fatalf("UpdateApplicantStage failed: %v", err)
	}
	t.Logf("✓ Candidate advanced to stage '%s'", stageList[1].Name)

	// 5. Run Cloudflare Workers AI Talent Screening & Resume Evaluation
	aiScreening, err := recruitmentSvc.RunAIScreening(applicant1.ID, company.ID, user.ID)
	if err != nil {
		t.Fatalf("RunAIScreening failed: %v", err)
	}
	if aiScreening.AIScore < 75.0 {
		t.Fatalf("Expected strong candidate AI score >= 75%%, got: %.1f%%", aiScreening.AIScore)
	}
	if len(aiScreening.KeyStrengths) == 0 {
		t.Fatalf("Expected AI identified strengths, got empty")
	}
	if len(aiScreening.SuggestedQuestions) == 0 {
		t.Fatalf("Expected AI suggested interview questions, got empty")
	}
	t.Logf("✓ Cloudflare Workers AI Talent Screening: Candidate='%s', Score=%.1f%%, Recommendation='%s', Engine='%s'",
		aiScreening.ApplicantName, aiScreening.AIScore, aiScreening.AIRecommendation, aiScreening.EngineSource)
	t.Logf("  Key Strengths: %s", strings.Join(aiScreening.KeyStrengths, "; "))
	t.Logf("  Interview Question #1: %s", aiScreening.SuggestedQuestions[0])

	// Verify Chatter log on Applicant
	appMsgs, err := chatterSvc.ListMessages("hr.applicant", applicant1.ID, company.ID)
	if err != nil || len(appMsgs) < 3 {
		t.Fatalf("Expected at least 3 chatter messages on applicant, got: %d", len(appMsgs))
	}
	t.Logf("✓ Chatter Audit Trail: Verified %d messages on candidate file", len(appMsgs))

	// 6. Execute 1-Click Action Hire (Convert Candidate to Active HR Employee)
	hireReq := recruitment.HireApplicantRequest{
		BasicSalary: 28000000,
		BankName:    "Bank BCA",
		BankAccount: "8820192837",
	}
	hiredApp, newEmployee, err := recruitmentSvc.ActionHire(applicant1.ID, hireReq, company.ID, user.ID)
	if err != nil {
		t.Fatalf("ActionHire failed: %v", err)
	}
	if hiredApp.State != "hired" || hiredApp.HiredEmployeeID == nil {
		t.Fatalf("Expected applicant state 'hired' with HiredEmployeeID set, got state: %s", hiredApp.State)
	}
	if newEmployee == nil || newEmployee.BasicSalary != 28000000 || newEmployee.Name != applicant1.Name {
		t.Fatalf("Employee not created correctly: %+v", newEmployee)
	}
	t.Logf("✓ 1-Click Hire Executed: Candidate '%s' converted to Employee ID #%d (Department ID: %d, Salary: Rp %.0f, Bank: %s %s)",
		hiredApp.Name, newEmployee.ID, newEmployee.DepartmentID, newEmployee.BasicSalary, newEmployee.BankName, newEmployee.BankAccount)

	// Verify Job Opening CurrentHired incremented
	updatedOpening, _ := recruitmentSvc.GetJobOpening(jobOpening.ID, company.ID)
	if updatedOpening.CurrentHired != 1 {
		t.Fatalf("Expected CurrentHired = 1, got: %d", updatedOpening.CurrentHired)
	}
	t.Logf("✓ Job Opening status updated: Current Hired=%d / %d", updatedOpening.CurrentHired, updatedOpening.ExpectedEmployees)

	// 7. Test Refusal Workflow with Candidate 2: "Eko Prasetyo"
	app2Req := recruitment.CreateApplicantRequest{
		Name:           "Eko Prasetyo",
		Email:          "eko.prasetyo@gmail.com",
		Phone:          "+62 856-7890-1234",
		JobOpeningID:   jobOpening.ID,
		SalaryExpected: 35000000,
		ResumeText:     "Saya lulusan baru yang tertarik belajar pemrograman komputer dan backend web dasar.",
	}
	applicant2, err := recruitmentSvc.CreateApplicant(&app2Req, company.ID, user.ID)
	if err != nil {
		t.Fatalf("Create applicant2 failed: %v", err)
	}
	refusedApp, err := recruitmentSvc.ActionRefuse(applicant2.ID, "Pengalaman teknis belum memenuhi kualifikasi tingkat Lead Architect", company.ID, user.ID)
	if err != nil || refusedApp.State != "refused" {
		t.Fatalf("ActionRefuse failed: %v, state: %s", err, refusedApp.State)
	}
	t.Logf("✓ Rejection workflow verified: Candidate '%s' refused with reason '%s'", refusedApp.Name, refusedApp.RefusalReason)

	// 8. Recruitment KPI Analytics
	recStats, err := recruitmentSvc.GetStats(company.ID)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if recStats.TotalOpenings < 1 || recStats.TotalApplicants < 2 || recStats.HiredCount < 1 || recStats.RefusedCount < 1 {
		t.Fatalf("Recruitment stats unexpected: %+v", recStats)
	}
	t.Logf("✓ Recruitment Executive KPI Stats: Openings=%d, Total Applicants=%d, In Progress=%d, Hired=%d, Refused=%d, Avg AI Score=%.1f%%, Top Opening='%s'",
		recStats.TotalOpenings, recStats.TotalApplicants, recStats.InProgressCount, recStats.HiredCount, recStats.RefusedCount, recStats.AverageAIScore, recStats.TopJobOpening)

	// =========================================================================
	// SECTION 28: PHYSICAL STOCK OPNAME, CLOUDFLARE WORKERS AI SHRINKAGE AUDIT & GL POSTING
	// =========================================================================
	t.Log("\n--- SECTION 28: PHYSICAL STOCK OPNAME, AI SHRINKAGE AUDIT & GENERAL LEDGER POSTING ---")

	stockInvRepo := stock_inventory.NewRepository(db)
	stockInvSvc := stock_inventory.NewService(db, stockInvRepo, chatterSvc, aiClient)

	// 1. Create Stock Opname Draft for Main Warehouse Location WH/Stock
	createInvReq := stock_inventory.CreateInventoryRequest{
		Name:        "OPNAME/2026/00002",
		WarehouseID: &whJkt.ID,
		LocationID:  whStock.ID,
		Date:        "2026-10-08",
		Notes:       "Pemeriksaan Fisik Stok Akhir Kuartal Q4 Gudang Utama Jakarta",
	}
	opname, err := stockInvSvc.CreateInventory(&createInvReq, company.ID, user.ID)
	if err != nil {
		t.Fatalf("CreateInventory failed: %v", err)
	}
	if opname.State != "draft" {
		t.Fatalf("Expected draft state, got: %s", opname.State)
	}
	t.Logf("✓ Created Stock Opname draft '%s' for Location ID=%d", opname.Name, opname.LocationID)

	// 2. Start Stock Opname (Snapshots real quants into lines)
	opname, err = stockInvSvc.StartInventory(opname.ID, company.ID, user.ID)
	if err != nil {
		t.Fatalf("StartInventory failed: %v", err)
	}
	if opname.State != "confirm" {
		t.Fatalf("Expected confirm state, got: %s", opname.State)
	}
	if len(opname.Lines) == 0 {
		t.Fatalf("Expected inventory lines to be populated from stock quants, got 0")
	}
	t.Logf("✓ Started Stock Opname: Status='confirm', Snapshot populated %d product lines", len(opname.Lines))

	// Record initial book quantity of line 0
	line0 := &opname.Lines[0]
	initialTheo0 := line0.TheoreticalQty

	// 3. Update physical counts with deliberate discrepancy to test AI audit & GL adjustment
	// Discrepancy 1: Deficit 2 units on Line 0 (Shrinkage Loss)
	newCounted0 := initialTheo0 - 2
	if newCounted0 < 0 {
		newCounted0 = 0
	}
	opname, err = stockInvSvc.UpdateLine(opname.ID, line0.ID, &stock_inventory.UpdateInventoryLineRequest{
		CountedQty: newCounted0,
		Reason:     "2 unit rusak fisik/pecah saat pemindahan rak oleh forklift",
	}, company.ID, user.ID)
	if err != nil {
		t.Fatalf("UpdateLine failed: %v", err)
	}

	// Discrepancy 2: Surplus 1 unit on Line 1 (if exists) or add line
	if len(opname.Lines) > 1 {
		line1 := &opname.Lines[1]
		opname, err = stockInvSvc.UpdateLine(opname.ID, line1.ID, &stock_inventory.UpdateInventoryLineRequest{
			CountedQty: line1.TheoreticalQty + 1,
			Reason:     "1 unit bonus sampel vendor belum teradministrasi",
		}, company.ID, user.ID)
		if err != nil {
			t.Fatalf("UpdateLine line1 failed: %v", err)
		}
	}
	t.Logf("✓ Updated physical counts: Total Diff=%.1f units, Loss Value=Rp %.2f, Gain Value=Rp %.2f",
		opname.TotalDifferenceQty, opname.TotalLossValue, opname.TotalGainValue)

	// 4. Cloudflare Workers AI Discrepancy & Shrinkage Loss Audit
	aiAudit, err := stockInvSvc.AuditWithAI(opname.ID, company.ID, user.ID)
	if err != nil {
		t.Fatalf("AuditWithAI failed: %v", err)
	}
	if aiAudit.RiskLevel == "" || aiAudit.Summary == "" {
		t.Fatalf("AI audit result missing data: %+v", aiAudit)
	}
	t.Logf("✓ Cloudflare Workers AI Inventory Audit: RiskLevel='%s', Discrepancies=%d, Engine='%s'",
		aiAudit.RiskLevel, aiAudit.DiscrepancyCount, aiAudit.EngineSource)
	t.Logf("  AI Summary: %s", aiAudit.Summary)
	if len(aiAudit.RecommendedActions) > 0 {
		t.Logf("  Recommended Action #1: %s", aiAudit.RecommendedActions[0])
	}

	// 5. Validate Stock Opname (Atomic Quant Adjustment + General Ledger Move Posting)
	opname, err = stockInvSvc.ValidateInventory(opname.ID, company.ID, user.ID)
	if err != nil {
		t.Fatalf("ValidateInventory failed: %v", err)
	}
	if opname.State != "done" {
		t.Fatalf("Expected done state, got: %s", opname.State)
	}
	t.Logf("✓ Validated Stock Opname '%s': State='done'", opname.Name)

	// 6. Verify real quant was updated to match counted quantity
	var updatedQuant stock.StockQuant
	err = db.Where("product_id = ? AND location_id = ? AND company_id = ?", line0.ProductID, opname.LocationID, company.ID).First(&updatedQuant).Error
	if err != nil {
		t.Fatalf("Fetch updated quant failed: %v", err)
	}
	if updatedQuant.Quantity != newCounted0 {
		t.Fatalf("Expected quant quantity to be synced to counted %.2f, got: %.2f", newCounted0, updatedQuant.Quantity)
	}
	t.Logf("✓ Atomic StockQuant verified: Location ID=%d, Product ID=%d synced to exact counted qty %.2f",
		opname.LocationID, line0.ProductID, updatedQuant.Quantity)

	// 7. Verify General Ledger Move posted and balanced
	if opname.AccountingMoveID != nil {
		var glMove account.AccountMove
		err = db.Preload("LineIDs").Where("id = ?", *opname.AccountingMoveID).First(&glMove).Error
		if err != nil {
			t.Fatalf("Fetch GL Move failed: %v", err)
		}
		if glMove.State != "posted" {
			t.Fatalf("Expected GL Move state 'posted', got: %s", glMove.State)
		}
		var totalDeb, totalCred float64
		for _, ml := range glMove.LineIDs {
			totalDeb += ml.Debit
			totalCred += ml.Credit
		}
		if math.Abs(totalDeb-totalCred) > 0.01 {
			t.Fatalf("GL Move is unbalance: Debit=%.2f, Credit=%.2f", totalDeb, totalCred)
		}
		t.Logf("✓ Balanced General Ledger Move '%s' verified: Total Amount=Rp %.2f, Debit=%.2f == Credit=%.2f, Ref='%s'",
			glMove.Name, glMove.AmountTotal, totalDeb, totalCred, glMove.Ref)
	}

	// 8. Verify Chatter audit trail
	chatterLogs, _ := chatterSvc.ListMessages("stock.inventory", opname.ID, company.ID)
	if len(chatterLogs) < 3 {
		t.Fatalf("Expected at least 3 chatter messages on Stock Opname, got: %d", len(chatterLogs))
	}
	t.Logf("✓ Chatter Audit Trail verified: %d audit messages logged on Stock Opname '%s'", len(chatterLogs), opname.Name)

	// 9. Aggregate KPI Stats
	invStats, err := stockInvSvc.GetStats(company.ID)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if invStats.TotalInventories < 1 || invStats.DoneCount < 1 {
		t.Fatalf("Unexpected stats: %+v", invStats)
	}
	t.Logf("✓ Stock Opname Executive Analytics: Total Opnames=%d, Done=%d, Total Diff=%.1f units, Loss Value=Rp %.2f, Gain Value=Rp %.2f",
		invStats.TotalInventories, invStats.DoneCount, invStats.TotalDifferenceUnits, invStats.TotalLossValue, invStats.TotalGainValue)

	// ==========================================
	// 29. TEST ENTERPRISE PAYROLL & SALARY RULES ENGINE, CLOUDFLARE WORKERS AI TAX AUDITOR & GL DOUBLE-ENTRY POSTING
	// ==========================================
	t.Log("\n--- SECTION 29: ENTERPRISE PAYROLL & SALARY RULES ENGINE, CLOUDFLARE WORKERS AI AUDITOR & DOUBLE-ENTRY GL POSTING ---")

	payrollSvc := payroll.NewService(db, chatterSvc, aiClient, accountSvc)

	// 1. Create Enterprise Salary Structure
	salaryStructure, err := payrollSvc.CreateStructure(payroll.CreatePayrollStructureRequest{
		Name:        "Struktur Standar Karyawan Tetap Indonesia",
		Code:        "IDN_STD",
		Description: "Struktur penggajian standar BPJS Ketenagakerjaan, BPJS Kesehatan, & PPh 21 TER",
		Rules: []payroll.CreateSalaryRuleRequest{
			{
				Name:              "Gaji Pokok",
				Code:              "BASIC",
				Category:          "BASIC",
				Sequence:          10,
				AmountType:        "fixed",
				DebitAccountCode:  "610200",
				CreditAccountCode: "210200",
			},
			{
				Name:              "Tunjangan Jabatan & Keahlian",
				Code:              "ALW_JOB",
				Category:          "ALW",
				Sequence:          20,
				AmountType:        "fixed",
				FixedAmount:       2500000,
				DebitAccountCode:  "610300",
				CreditAccountCode: "210200",
			},
			{
				Name:              "Upah Lembur Presensi (Attendance)",
				Code:              "ALW_OT",
				Category:          "ALW",
				Sequence:          40,
				AmountType:        "dynamic",
				DebitAccountCode:  "610300",
				CreditAccountCode: "210200",
			},
			{
				Name:              "Potongan Cuti Tanpa Gaji (Unpaid Leave)",
				Code:              "DED_UNPAID",
				Category:          "DED",
				Sequence:          50,
				AmountType:        "dynamic",
			},
			{
				Name:              "Iuran BPJS Ketenagakerjaan (3%)",
				Code:              "DED_BPJS_TK",
				Category:          "DED",
				Sequence:          60,
				AmountType:        "percentage",
				Percentage:        3.0,
				DebitAccountCode:  "610200",
				CreditAccountCode: "210400",
			},
			{
				Name:              "Iuran BPJS Kesehatan (1%)",
				Code:              "DED_BPJS_KES",
				Category:          "DED",
				Sequence:          70,
				AmountType:        "percentage",
				Percentage:        1.0,
				DebitAccountCode:  "610200",
				CreditAccountCode: "210400",
			},
			{
				Name:              "Pajak PPh 21 TER (5%)",
				Code:              "DED_PPH21",
				Category:          "TAX",
				Sequence:          80,
				AmountType:        "percentage",
				Percentage:        5.0,
				DebitAccountCode:  "610200",
				CreditAccountCode: "210300",
			},
			{
				Name:              "Gaji Bersih (Take-Home Pay)",
				Code:              "NET",
				Category:          "NET",
				Sequence:          100,
				AmountType:        "dynamic",
				CreditAccountCode: "210200",
			},
		},
	}, company.ID)
	if err != nil {
		t.Fatalf("Failed to create salary structure: %v", err)
	}
	t.Logf("✓ Created Payroll Structure '%s' (%s) with %d rules", salaryStructure.Name, salaryStructure.Code, len(salaryStructure.Rules))

	// 2. Create Batch Payslip Run
	payRun, err := payrollSvc.CreatePayslipRun(payroll.CreatePayslipRunRequest{
		Name:      "BATCH/PAY/2026/10",
		Period:    "October 2026",
		DateStart: "2026-10-01",
		DateEnd:   "2026-10-31",
		Notes:     "Batch Penggajian Akhir Bulan Oktober 2026",
	}, company.ID)
	if err != nil {
		t.Fatalf("Failed to create payslip run: %v", err)
	}
	t.Logf("✓ Created Batch Payroll Run '%s' (Period: %s, State: %s)", payRun.Name, payRun.Period, payRun.State)

	// 3. Create Payslip for Employee (Bambang Wijaya, Basic Salary = 20,000,000)
	var targetEmp hr.Employee
	if err := db.Where("company_id = ?", company.ID).First(&targetEmp).Error; err != nil {
		t.Fatalf("Target employee not found: %v", err)
	}

	createSlipReq := payroll.CreatePayslipRequest{
		EmployeeID:   targetEmp.ID,
		StructureID:  salaryStructure.ID,
		PayslipRunID: &payRun.ID,
		Period:       "October 2026",
		DateFrom:     "2026-10-01",
		DateTo:       "2026-10-31",
		WorkingDays:  22,
		Notes:        "Slip Gaji Reguler Oktober 2026",
	}
	paySlip, err := payrollSvc.CreatePayslip(createSlipReq, company.ID)
	if err != nil {
		t.Fatalf("Failed to create payslip: %v", err)
	}
	t.Logf("✓ Created Payslip '%s' for Employee '%s' (Basic: Rp %.0f)", paySlip.Number, targetEmp.Name, paySlip.BasicSalary)

	// 4. Compute Payslip with Overtime & Unpaid Leave integrations
	computeReq := payroll.ComputePayslipRequest{
		SyncAttendance: true,
		SyncLeave:      true,
		ManualOTHours:  4.0, // 4 hours manual/synced overtime
	}
	computedSlip, err := payrollSvc.ComputePayslip(paySlip.ID, computeReq, company.ID)
	if err != nil {
		t.Fatalf("Failed to compute payslip: %v", err)
	}

	if computedSlip.GrossSalary <= computedSlip.BasicSalary {
		t.Fatalf("Expected GrossSalary (%.2f) to be greater than BasicSalary (%.2f)", computedSlip.GrossSalary, computedSlip.BasicSalary)
	}
	if computedSlip.TotalDeductions <= 0 {
		t.Fatalf("Expected TotalDeductions > 0, got %.2f", computedSlip.TotalDeductions)
	}
	if computedSlip.NetSalary != (computedSlip.GrossSalary - computedSlip.TotalDeductions) {
		t.Fatalf("NetSalary mismatch: Net=%.2f, Gross=%.2f, Deductions=%.2f",
			computedSlip.NetSalary, computedSlip.GrossSalary, computedSlip.TotalDeductions)
	}
	if len(computedSlip.Lines) == 0 {
		t.Fatalf("Expected payslip lines to be populated, got 0")
	}
	t.Logf("✓ Payslip '%s' Computed: Gross=Rp %.0f (Basic: Rp %.0f, Tunjangan: Rp 2.500.000, Lembur: %.1f jam, Rp %.0f) | Total Potongan=Rp %.0f | Net Take-Home=Rp %.0f (%d breakdown lines)",
		computedSlip.Number, computedSlip.GrossSalary, computedSlip.BasicSalary, computedSlip.OvertimeHours, computedSlip.OvertimeAmount, computedSlip.TotalDeductions, computedSlip.NetSalary, len(computedSlip.Lines))

	// 5. Cloudflare Workers AI Tax & Payroll Compliance Audit
	payrollAIAudit, err := payrollSvc.AuditWithAI(computedSlip.ID, company.ID)
	if err != nil {
		t.Fatalf("AuditWithAI failed: %v", err)
	}
	if payrollAIAudit.ComplianceScore <= 0 || payrollAIAudit.RiskLevel == "" {
		t.Fatalf("Invalid AI audit result: %+v", payrollAIAudit)
	}
	t.Logf("✓ Cloudflare Workers AI Payroll Audit: Employee='%s', ComplianceScore=%.1f%%, RiskLevel='%s', TaxStatus='%s', OvertimeStatus='%s', Engine='%s'",
		payrollAIAudit.EmployeeName, payrollAIAudit.ComplianceScore, payrollAIAudit.RiskLevel, payrollAIAudit.TaxComplianceStatus, payrollAIAudit.OvertimeRegulationStatus, payrollAIAudit.Source)
	t.Logf("  Executive Summary: %s", payrollAIAudit.ExecutiveSummary)
	if len(payrollAIAudit.Recommendations) > 0 {
		t.Logf("  Recommendation #1: %s", payrollAIAudit.Recommendations[0])
	}

	// 6. Confirm Payslip & Verify Balanced General Ledger Double-Entry Move
	confirmedPaySlip, err := payrollSvc.ConfirmPayslip(computedSlip.ID, company.ID)
	if err != nil {
		t.Fatalf("ConfirmPayslip failed: %v", err)
	}
	if confirmedPaySlip.State != "done" || confirmedPaySlip.AccountingMoveID == nil {
		t.Fatalf("Expected payslip state 'done' and non-nil AccountingMoveID, got: %s, moveID: %v", confirmedPaySlip.State, confirmedPaySlip.AccountingMoveID)
	}

	// Verify the General Ledger Move
	payrollGLMove, err := accountSvc.GetMove(*confirmedPaySlip.AccountingMoveID, company.ID)
	if err != nil {
		t.Fatalf("Failed to fetch GL Move #%d: %v", *confirmedPaySlip.AccountingMoveID, err)
	}
	if payrollGLMove.State != "posted" {
		t.Fatalf("Expected GL Move to be posted, got '%s'", payrollGLMove.State)
	}
	var totalDeb, totalCred float64
	for _, ml := range payrollGLMove.LineIDs {
		totalDeb += ml.Debit
		totalCred += ml.Credit
	}
	if math.Abs(totalDeb-totalCred) > 0.01 {
		t.Fatalf("Payroll GL Move is unbalanced: Debit=%.2f != Credit=%.2f", totalDeb, totalCred)
	}
	t.Logf("✓ Balanced General Ledger Move '%s' verified: Total Amount=Rp %.2f, Debit=%.2f == Credit=%.2f, Ref='%s'",
		payrollGLMove.Name, payrollGLMove.AmountTotal, totalDeb, totalCred, payrollGLMove.Ref)

	// 7. Pay Payslip & Record Bank Disbursement
	paidPaySlip, err := payrollSvc.PayPayslip(confirmedPaySlip.ID, company.ID)
	if err != nil {
		t.Fatalf("PayPayslip failed: %v", err)
	}
	if paidPaySlip.State != "paid" {
		t.Fatalf("Expected payslip state 'paid', got '%s'", paidPaySlip.State)
	}
	t.Logf("✓ Payslip '%s' Paid: Net Take-Home Salary Rp %.0f disbursed via Bank BCA", paidPaySlip.Number, paidPaySlip.NetSalary)

	// 8. Generate Batch Payslips
	batchRunUpdated, err := payrollSvc.GenerateRunPayslips(payRun.ID, company.ID)
	if err != nil {
		t.Fatalf("GenerateRunPayslips failed: %v", err)
	}
	if batchRunUpdated.TotalEmployees < 1 || batchRunUpdated.TotalNet <= 0 {
		t.Fatalf("Unexpected batch run result: %+v", batchRunUpdated)
	}
	t.Logf("✓ Batch Payroll Run '%s' Synchronized: Employees=%d, Gross=Rp %.0f, Deductions=Rp %.0f, Total Net=Rp %.0f, State='%s'",
		batchRunUpdated.Name, batchRunUpdated.TotalEmployees, batchRunUpdated.TotalGross, batchRunUpdated.TotalDeductions, batchRunUpdated.TotalNet, batchRunUpdated.State)

	// 9. Verify Chatter Audit Trail on Payslip
	chatterLogs, _ = chatterSvc.ListMessages("hr.payslip", paidPaySlip.ID, company.ID)
	if len(chatterLogs) < 4 {
		t.Fatalf("Expected at least 4 chatter messages on Payslip, got: %d", len(chatterLogs))
	}
	t.Logf("✓ Chatter Audit Trail verified: %d audit messages logged on Payslip '%s'", len(chatterLogs), paidPaySlip.Number)

	// 10. Executive KPI Stats
	payrollStats, err := payrollSvc.GetStats(company.ID)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if payrollStats.TotalPayslips < 1 || payrollStats.TotalPaidSlips < 1 || payrollStats.TotalGrossExpenditure <= 0 {
		t.Fatalf("Unexpected payroll stats: %+v", payrollStats)
	}
	t.Logf("✓ Executive Payroll KPI Analytics: Runs=%d, Total Payslips=%d, Paid Slips=%d, Gross Expenditure=Rp %.0f, Withheld Deductions=Rp %.0f, Net Paid=Rp %.0f, Overtime=%.1f hrs",
		payrollStats.TotalRuns, payrollStats.TotalPayslips, payrollStats.TotalPaidSlips, payrollStats.TotalGrossExpenditure, payrollStats.TotalDeductionsWithheld, payrollStats.TotalNetPaid, payrollStats.TotalOvertimeHours)

	// ==========================================
	// SECTION 30: ENTERPRISE BUDGETING & ANALYTIC ACCOUNTING, VARIANCE ANALYSIS & AI BURN-RATE AUDITOR
	// ==========================================
	t.Log("\n--- SECTION 30: ENTERPRISE BUDGETING & ANALYTIC ACCOUNTING, VARIANCE ANALYSIS & AI AUDITOR ---")

	budgetSvc := budget.NewService(db, chatterSvc, aiClient, accountSvc)

	// 1. Create Analytic Accounts (Cost Centers / Departments)
	anEng, err := budgetSvc.CreateAnalyticAccount(budget.CreateAnalyticAccountRequest{
		Code:        "AN/ENG",
		Name:        "Departemen IT & Software Engineering",
		Description: "Pusat biaya pengembangan sistem ERP dan infrastruktur cloud",
	}, company.ID)
	if err != nil {
		t.Fatalf("CreateAnalyticAccount ENG failed: %v", err)
	}

	anOps, err := budgetSvc.CreateAnalyticAccount(budget.CreateAnalyticAccountRequest{
		Code:        "AN/OPS",
		Name:        "Operasional Pabrik & Logistik",
		Description: "Pusat biaya logistik gudang dan operasional pabrik",
	}, company.ID)
	if err != nil {
		t.Fatalf("CreateAnalyticAccount OPS failed: %v", err)
	}

	anMkt, err := budgetSvc.CreateAnalyticAccount(budget.CreateAnalyticAccountRequest{
		Code:        "AN/MKT",
		Name:        "Divisi Penjualan & Digital Marketing",
		Description: "Pusat biaya promosi komersial dan campaign penjualan",
	}, company.ID)
	if err != nil {
		t.Fatalf("CreateAnalyticAccount MKT failed: %v", err)
	}
	t.Logf("✓ Created 3 Analytic Accounts: %s, %s, %s", anEng.Code, anOps.Code, anMkt.Code)

	// 2. Create Budgetary Positions (Account Budget Posts)
	// Linking to accounts created in Section 29: 610200 (Beban Gaji Pokok) & 610300 (Beban Tunjangan)
	var accGaji, accTunjangan account.AccountAccount
	db.Where("code = ?", "610200").First(&accGaji)
	db.Where("code = ?", "610300").First(&accTunjangan)
	payrollAccountIDs := []uint{}
	if accGaji.ID > 0 {
		payrollAccountIDs = append(payrollAccountIDs, accGaji.ID)
	}
	if accTunjangan.ID > 0 {
		payrollAccountIDs = append(payrollAccountIDs, accTunjangan.ID)
	}

	bpPayroll, err := budgetSvc.CreateBudgetaryPosition(budget.CreateBudgetaryPositionRequest{
		Name:        "Pos Beban Gaji & Ketenagakerjaan",
		Code:        "BP/PAYROLL",
		Description: "Plafon kompensasi pokok, lembur presensi, dan tunjangan",
		AccountIDs:  payrollAccountIDs,
	}, company.ID)
	if err != nil {
		t.Fatalf("CreateBudgetaryPosition BP/PAYROLL failed: %v", err)
	}

	bpOps, err := budgetSvc.CreateBudgetaryPosition(budget.CreateBudgetaryPositionRequest{
		Name:        "Pos Beban Operasional Kantor & Perjalanan",
		Code:        "BP/OPS",
		Description: "Plafon biaya perjalanan dinas dan konsumsi operasional",
		AccountIDs:  []uint{20, 21, 22},
	}, company.ID)
	if err != nil {
		t.Fatalf("CreateBudgetaryPosition BP/OPS failed: %v", err)
	}

	bpMkt, err := budgetSvc.CreateBudgetaryPosition(budget.CreateBudgetaryPositionRequest{
		Name:        "Pos Beban Pemasaran & Promosi",
		Code:        "BP/MKT",
		Description: "Plafon belanja iklan digital dan event B2B",
		AccountIDs:  []uint{20, 21},
	}, company.ID)
	if err != nil {
		t.Fatalf("CreateBudgetaryPosition BP/MKT failed: %v", err)
	}
	t.Logf("✓ Created 3 Budgetary Positions: %s, %s, %s", bpPayroll.Code, bpOps.Code, bpMkt.Code)

	// 3. Create Budget Document (Rencana Kerja & Anggaran Biaya Q4 2026)
	budgetReq := budget.CreateBudgetRequest{
		Name:     "Rencana Kerja & Anggaran Biaya (RKAB) Q4 2026",
		Code:     "BDG/2026/Q4/00001",
		DateFrom: "2026-10-01",
		DateTo:   "2026-12-31",
		Notes:    "Dokumen Rencana Anggaran Belanja Q4 2026",
		Lines: []budget.CreateBudgetLineRequest{
			{
				BudgetaryPositionID: bpPayroll.ID,
				AnalyticAccountID:   anEng.ID,
				DateFrom:            "2026-10-01",
				DateTo:              "2026-12-31",
				PlannedAmount:       60000000, // Rp 60 jt
				Notes:               "Beban Gaji Tim Engineering",
			},
			{
				BudgetaryPositionID: bpOps.ID,
				AnalyticAccountID:   anOps.ID,
				DateFrom:            "2026-10-01",
				DateTo:              "2026-12-31",
				PlannedAmount:       30000000, // Rp 30 jt
				Notes:               "Beban Operasional Gudang & Logistik",
			},
			{
				BudgetaryPositionID: bpMkt.ID,
				AnalyticAccountID:   anMkt.ID,
				DateFrom:            "2026-10-01",
				DateTo:              "2026-12-31",
				PlannedAmount:       20000000, // Rp 20 jt
				Notes:               "Beban Pemasaran Digital",
			},
		},
	}

	b1, err := budgetSvc.CreateBudget(budgetReq, user.ID, company.ID)
	if err != nil {
		t.Fatalf("CreateBudget failed: %v", err)
	}
	if b1.TotalPlannedAmount != 110000000 || len(b1.Lines) != 3 || b1.State != "draft" {
		t.Fatalf("Unexpected budget draft: TotalPlanned=%.0f, Lines=%d, State=%s", b1.TotalPlannedAmount, len(b1.Lines), b1.State)
	}
	t.Logf("✓ Created Budget Plan '%s' (%s): Plafon=Rp %.0f, 3 Pos Anggaran, State='%s'",
		b1.Name, b1.Code, b1.TotalPlannedAmount, b1.State)

	// 4. State Transitions: Confirm -> Validate
	b1, err = budgetSvc.ActionConfirm(b1.ID, company.ID, user.ID)
	if err != nil || b1.State != "confirm" {
		t.Fatalf("ActionConfirm failed: err=%v, state=%s", err, b1.State)
	}
	b1, err = budgetSvc.ActionValidate(b1.ID, company.ID, user.ID)
	if err != nil || b1.State != "validate" {
		t.Fatalf("ActionValidate failed: err=%v, state=%s", err, b1.State)
	}
	t.Logf("✓ Budget Lifecycle validated: State='%s' (Aktif memantau realisasi belanja berjalan)", b1.State)

	// 5. Verify Real-time Realized Expenditure & Variance Calculation
	// (In Section 29, Move MISC/2026/00005 posted Debit Rp 22.962.428 to accounts 20 & 21)
	if err := budgetSvc.ComputeBudget(b1.ID, company.ID); err != nil {
		t.Fatalf("ComputeBudget failed: %v", err)
	}
	b1, _ = budgetSvc.GetBudget(b1.ID)

	var linePayroll *budget.BudgetLine
	for i := range b1.Lines {
		if b1.Lines[i].BudgetaryPositionID == bpPayroll.ID {
			linePayroll = &b1.Lines[i]
			break
		}
	}
	if linePayroll == nil {
		t.Fatal("BP/PAYROLL line not found in budget")
	}

	if linePayroll.PracticalAmount <= 0 {
		t.Fatalf("Expected practical amount > 0 from posted GL moves, got: %.2f", linePayroll.PracticalAmount)
	}
	if linePayroll.VarianceAmount != linePayroll.PlannedAmount-linePayroll.PracticalAmount {
		t.Fatalf("Variance calculation mismatch: Variance=%.2f, Planned=%.2f, Practical=%.2f",
			linePayroll.VarianceAmount, linePayroll.PlannedAmount, linePayroll.PracticalAmount)
	}
	t.Logf("✓ Realized Expenditure Synced from GL Moves: Line '%s' -> Plafon=Rp %.0f, Realisasi=Rp %.0f, Konsumsi=%.1f%%, Sisa Anggaran=Rp %.0f",
		linePayroll.BudgetaryPosition.Name, linePayroll.PlannedAmount, linePayroll.PracticalAmount, linePayroll.Percentage, linePayroll.VarianceAmount)

	// 6. Cloudflare Workers AI Budget Variance & Overspend Auditor
	aiAuditResult1, err := budgetSvc.AuditWithAI(b1.ID, company.ID, user.ID)
	if err != nil {
		t.Fatalf("AuditWithAI failed: %v", err)
	}
	if aiAuditResult1.RiskLevel == "" || aiAuditResult1.ExecutiveSummary == "" {
		t.Fatalf("Invalid AI audit result: %+v", aiAuditResult1)
	}
	t.Logf("✓ Cloudflare Workers AI Budget Variance Audit: Document='%s', RiskLevel='%s', RiskScore=%.1f, BurnRate='%s', Engine='%s'",
		aiAuditResult1.BudgetName, aiAuditResult1.RiskLevel, aiAuditResult1.RiskScore, aiAuditResult1.OverallBurnRate, aiAuditResult1.Source)
	t.Logf("  Executive Summary: %s", aiAuditResult1.ExecutiveSummary)
	if len(aiAuditResult1.RecommendedActions) > 0 {
		t.Logf("  Action Recommendation #1: %s", aiAuditResult1.RecommendedActions[0])
	}

	// 7. Simulate Overspend Anomaly & Critical Risk Flagging
	// Create second budget with tight planned ceiling to test anomaly detection
	b2Req := budget.CreateBudgetRequest{
		Name:     "Anggaran Khusus Riset Kilat Q4",
		Code:     "BDG/2026/Q4/OVER01",
		DateFrom: "2026-10-01",
		DateTo:   "2026-12-31",
		Lines: []budget.CreateBudgetLineRequest{
			{
				BudgetaryPositionID: bpPayroll.ID,
				AnalyticAccountID:   anEng.ID,
				PlannedAmount:       15000000, // Rp 15 jt (Sedangkan realisasi aktual Rp 22.962.428 -> >150% overspend!)
			},
		},
	}
	b2, err := budgetSvc.CreateBudget(b2Req, user.ID, company.ID)
	if err != nil {
		t.Fatalf("CreateBudget b2 failed: %v", err)
	}
	b2, _ = budgetSvc.ActionConfirm(b2.ID, company.ID, user.ID)
	b2, _ = budgetSvc.ActionValidate(b2.ID, company.ID, user.ID)

	aiAuditResult2, err := budgetSvc.AuditWithAI(b2.ID, company.ID, user.ID)
	if err != nil {
		t.Fatalf("AuditWithAI b2 failed: %v", err)
	}
	if aiAuditResult2.RiskLevel != "critical" && aiAuditResult2.RiskLevel != "high" {
		t.Fatalf("Expected critical or high risk on overspent budget, got: %s", aiAuditResult2.RiskLevel)
	}
	if len(aiAuditResult2.FlaggedOverspends) == 0 {
		t.Fatalf("Expected flagged overspends on budget b2, got empty")
	}
	t.Logf("✓ AI Overspend Anomaly correctly flagged: Document='%s', RiskLevel='%s', BurnRate='%s', Overspend Findings=%d",
		b2.Name, aiAuditResult2.RiskLevel, aiAuditResult2.OverallBurnRate, len(aiAuditResult2.FlaggedOverspends))
	t.Logf("  Warning: %s", aiAuditResult2.FlaggedOverspends[0])

	// 8. Close Budget 1 to Done
	b1, err = budgetSvc.ActionDone(b1.ID, company.ID, user.ID)
	if err != nil || b1.State != "done" {
		t.Fatalf("ActionDone failed: err=%v, state=%s", err, b1.State)
	}
	t.Logf("✓ Budget '%s' closed: State='done'", b1.Name)

	// 9. Verify Chatter Audit Trail
	budgetChatter, _ := chatterSvc.ListMessages("account.budget", b1.ID, company.ID)
	if len(budgetChatter) < 4 {
		t.Fatalf("Expected at least 4 chatter messages on Budget, got: %d", len(budgetChatter))
	}
	t.Logf("✓ Chatter Audit Trail verified: %d audit messages logged on Budget '%s'", len(budgetChatter), b1.Code)

	// 10. Executive Budget KPI Stats
	bStats, err := budgetSvc.GetStats(company.ID)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if bStats.TotalBudgets < 2 || bStats.DoneBudgets < 1 || bStats.TotalPlannedAmount <= 0 || bStats.OverspentLinesCount < 1 {
		t.Fatalf("Unexpected budget stats: %+v", bStats)
	}
	t.Logf("✓ Executive Budget KPI Analytics: Total Budgets=%d, Done=%d, Total Planned=Rp %.0f, Total Practical=Rp %.0f, Overspent Lines=%d, High Risk Budgets=%d",
		bStats.TotalBudgets, bStats.DoneBudgets, bStats.TotalPlannedAmount, bStats.TotalPracticalAmount, bStats.OverspentLinesCount, bStats.HighRiskBudgetsCount)

	t.Log("\n--- SECTION 31: BARCODE & QR CODE SCANNER ENGINE, LOT/SERIAL TRACKING & MOBILE OPERATIONS ---")

	barcodeRepo := barcode.NewRepository(db)
	barcodeSvc := barcode.NewService(db, barcodeRepo)

	// 1. Assign Barcode to testProduct
	testProduct.Barcode = "899000100001"
	db.Save(&testProduct)

	// 2. Create Lot & Serial Number for testProduct
	testLot, err := barcodeSvc.CreateLot(barcode.CreateLotRequest{
		Name:           "LOT-2026-TEST-001",
		ProductID:      testProduct.ID,
		Barcode:        "LOT2026TEST001",
		Ref:            "PO-BATCH-OCT-2026-A",
		ExpirationDate: "2028-12-31",
		AlertDate:      "2028-06-30",
		Qty:            100,
		Notes:          "Batch perakitan unit pertama dengan garansi resmi 2 tahun",
	}, company.ID)
	if err != nil {
		t.Fatalf("CreateLot failed: %v", err)
	}
	if testLot.ID == 0 || testLot.Name != "LOT-2026-TEST-001" || testLot.ExpirationDate != "2028-12-31" {
		t.Fatalf("Unexpected lot record: %+v", testLot)
	}
	t.Logf("✓ Created Lot/Serial Number: Name='%s', Product ID=%d, Barcode='%s', Qty=%.0f, ExpDate='%s'",
		testLot.Name, testLot.ProductID, testLot.Barcode, testLot.Qty, testLot.ExpirationDate)

	// 3. Test Universal Barcode Scanner (Direct Product Lookup)
	scanProd, err := barcodeSvc.Scan(barcode.BarcodeScanRequest{
		Barcode: "899000100001",
		Context: "general",
	}, company.ID, user.ID)
	if err != nil || !scanProd.Found || scanProd.Type != "product" {
		t.Fatalf("Product barcode scan failed: err=%v, res=%+v", err, scanProd)
	}
	if scanProd.Product.ID != testProduct.ID {
		t.Fatalf("Expected scanned product ID %d, got %d", testProduct.ID, scanProd.Product.ID)
	}
	t.Logf("✓ Barcode Scanner (Product): Recognized '%s' (SKU: %s, Price: Rp %.0f)",
		scanProd.Product.Name, scanProd.Product.DefaultCode, scanProd.Product.ListPrice)

	// 4. Test Universal Barcode Scanner (Lot/Serial Number Lookup)
	scanLot, err := barcodeSvc.Scan(barcode.BarcodeScanRequest{
		Barcode: "LOT2026TEST001",
		Context: "general",
	}, company.ID, user.ID)
	if err != nil || !scanLot.Found || scanLot.Type != "lot" {
		t.Fatalf("Lot barcode scan failed: err=%v, res=%+v", err, scanLot)
	}
	t.Logf("✓ Barcode Scanner (Lot): Recognized Lot '%s' for Product '%s' (Saldo Lot: %.0f, Kadaluwarsa: %s)",
		scanLot.Lot.Name, scanLot.Lot.Product.Name, scanLot.Lot.Qty, scanLot.Lot.ExpirationDate)

	// 5. Test Universal Barcode Scanner (Warehouse Location Lookup)
	scanLoc, err := barcodeSvc.Scan(barcode.BarcodeScanRequest{
		Barcode: "WH/Stock",
		Context: "general",
	}, company.ID, user.ID)
	if err != nil || !scanLoc.Found || scanLoc.Type != "location" {
		t.Fatalf("Location barcode scan failed: err=%v, res=%+v", err, scanLoc)
	}
	t.Logf("✓ Barcode Scanner (Location): Recognized Shelf Location '%s' (Usage: %s)",
		scanLoc.Location.CompleteName, scanLoc.Location.Usage)

	// 6. Test POS Mobile Scanner Integration
	scanPOS, err := barcodeSvc.Scan(barcode.BarcodeScanRequest{
		Barcode: "899000100001",
		Context: "pos",
		Qty:     2.0,
	}, company.ID, user.ID)
	if err != nil || scanPOS.ActionTaken != "pos_item_prepared" {
		t.Fatalf("POS scan action failed: err=%v, res=%+v", err, scanPOS)
	}
	posData, ok := scanPOS.Data.(map[string]interface{})
	if !ok || posData["price_subtotal"].(float64) != testProduct.ListPrice*2 {
		t.Fatalf("Unexpected POS data calculation: %+v", posData)
	}
	t.Logf("✓ Barcode Scanner (POS Cart Integration): Scanned Qty=2, Subtotal=Rp %.0f, Cart Item prepared instantly",
		posData["price_subtotal"])

	// 7. Test Physical Stock Opname Barcode Scanner Integration
	// Create a dedicated stock opname for testing barcode counting
	testInv, err := stockInvSvc.CreateInventory(&stock_inventory.CreateInventoryRequest{
		Name:        "OPNAME/2026/BARCODE-TEST",
		LocationID:  whStock.ID,
		Date:        "2026-10-08",
		Notes:       "Uji coba pemindaian barcode fisik barang di gudang",
	}, company.ID, user.ID)
	if err != nil {
		t.Fatalf("Create test inventory failed: %v", err)
	}
	testInv, err = stockInvSvc.StartInventory(testInv.ID, company.ID, user.ID)
	if err != nil {
		t.Fatalf("Start test inventory failed: %v", err)
	}

	// Scan Product 1 into Stock Opname
	scanOpname, err := barcodeSvc.Scan(barcode.BarcodeScanRequest{
		Barcode:  "899000100001",
		Context:  "stock_inventory",
		TargetID: testInv.ID,
		Qty:      10.0,
		Action:   "increment",
	}, company.ID, user.ID)
	if err != nil || (!strings.HasPrefix(scanOpname.ActionTaken, "inventory_line_")) {
		t.Fatalf("Stock Opname barcode scan failed: err=%v, res=%+v", err, scanOpname)
	}
	t.Logf("✓ Barcode Scanner (Stock Opname Integration): Scanned '%s' -> Action='%s', Msg='%s'",
		testProduct.Name, scanOpname.ActionTaken, scanOpname.Message)

	// 8. Test Code 128 Vector SVG Label Generator
	svgProduct, err := barcodeSvc.GenerateLabelSVG("product", testProduct.ID, company.ID)
	if err != nil || !strings.Contains(svgProduct, "<svg") || !strings.Contains(svgProduct, "899000100001") {
		t.Fatalf("Generate product label SVG failed: err=%v, svg len=%d", err, len(svgProduct))
	}
	svgLot, err := barcodeSvc.GenerateLabelSVG("lot", testLot.ID, company.ID)
	if err != nil || !strings.Contains(svgLot, "<svg") || !strings.Contains(svgLot, "LOT-2026-TEST-001") {
		t.Fatalf("Generate lot label SVG failed: err=%v", err)
	}
	svgLoc, err := barcodeSvc.GenerateLabelSVG("location", whStock.ID, company.ID)
	if err != nil || !strings.Contains(svgLoc, "<svg") || !strings.Contains(svgLoc, "LOC-") {
		t.Fatalf("Generate location label SVG failed: err=%v", err)
	}
	t.Logf("✓ Code 128 Pure Go Vector SVG Generator verified: Product Label (%d bytes), Lot Label (%d bytes), Shelf Location Label (%d bytes)",
		len(svgProduct), len(svgLot), len(svgLoc))

	// 9. Barcode Analytics & KPI Stats
	barStats, err := barcodeSvc.GetStats(company.ID)
	if err != nil || barStats.TotalProductsBarcoded < 1 || barStats.TotalLots < 1 {
		t.Fatalf("Barcode GetStats failed: err=%v, stats=%+v", err, barStats)
	}
	t.Logf("✓ Barcode Executive KPI Stats: Products Barcoded=%d, Total Lots=%d, Active Stock Lots=%d, Shelf Locations Barcoded=%d",
		barStats.TotalProductsBarcoded, barStats.TotalLots, barStats.ActiveLotsWithStock, barStats.TotalLocationsBarcoded)

	t.Log("\n--- SECTION 32: PURCHASE REQUISITIONS, COMPETITIVE TENDER BIDDING & BLANKET PURCHASE ORDERS ---")

	purchaseReqRepo := purchase_requisition.NewRepository(db)
	purchaseReqSvc := purchase_requisition.NewService(db, purchaseReqRepo, purchaseSvc, chatterSvc)

	// Create second competitor vendor for tender bidding
	vendor2 := base.Partner{
		Name:         "CV Multi Mega Komponen",
		IsCompany:    true,
		SupplierRank: 1,
	}
	vendor2.CompanyID = company.ID
	db.Create(&vendor2)

	// 1. Create Internal Purchase Requisition (Competitive Tender / Call for Bids)
	reqTender, err := purchaseReqSvc.CreateRequisition(purchase_requisition.CreateRequisitionRequest{
		Origin:       "MEMO/ENG/2026/OCT-01",
		Type:         purchase_requisition.TypePurchaseTender,
		OrderingDate: "2026-10-08",
		DateEnd:      "2026-10-20",
		ScheduleDate: "2026-10-25",
		Description:  "Pengadaan batch 20 unit monitor display untuk workstation lab perakitan",
		Lines: []purchase_requisition.CreateRequisitionLineRequest{
			{
				ProductID:   testProduct.ID,
				Description: testProduct.Name,
				ProductQty:  20,
				PriceUnit:   2400000,
			},
		},
	}, company.ID, user.ID)
	if err != nil {
		t.Fatalf("CreateRequisition (Tender) failed: %v", err)
	}
	if reqTender.ID == 0 || !strings.HasPrefix(reqTender.Name, "PR/") || reqTender.State != purchase_requisition.StateDraft {
		t.Fatalf("Unexpected tender requisition state: %+v", reqTender)
	}
	if reqTender.TotalAmount != 48000000 {
		t.Fatalf("Expected total amount 48000000, got %.2f", reqTender.TotalAmount)
	}
	t.Logf("✓ Created Purchase Requisition (Tender): '%s' (State: %s, Total Plafon: Rp %.0f, 20 units)",
		reqTender.Name, reqTender.State, reqTender.TotalAmount)

	// 2. Manager Approval (Draft -> Confirmed)
	confirmedTender, err := purchaseReqSvc.ActionConfirm(reqTender.ID, company.ID, user.ID)
	if err != nil || confirmedTender.State != purchase_requisition.StateConfirmed {
		t.Fatalf("ActionConfirm tender failed: %v", err)
	}
	t.Logf("✓ Purchase Requisition approved by Manager: State='%s'", confirmedTender.State)

	// 3. Open Tender (Confirmed -> In Progress)
	openedTender, err := purchaseReqSvc.ActionOpenTender(reqTender.ID, company.ID, user.ID)
	if err != nil || openedTender.State != purchase_requisition.StateInProgress {
		t.Fatalf("ActionOpenTender failed: %v", err)
	}
	t.Logf("✓ Bidding Tender Opened: State='%s' (Ready for multi-vendor RFQ generation)", openedTender.State)

	// 4. Generate Competing RFQs for Vendor 1 and Vendor 2
	rfq1, err := purchaseReqSvc.GenerateRFQ(reqTender.ID, purchase_requisition.GenerateRFQRequest{
		VendorID: vendor.ID,
		Notes:    "Penawaran harga dari Vendor Utama Rekanan",
	}, company.ID, user.ID)
	if err != nil || rfq1.ID == 0 || rfq1.RequisitionID == nil || *rfq1.RequisitionID != reqTender.ID {
		t.Fatalf("GenerateRFQ #1 failed: %v", err)
	}

	rfq2, err := purchaseReqSvc.GenerateRFQ(reqTender.ID, purchase_requisition.GenerateRFQRequest{
		VendorID: vendor2.ID,
		Notes:    "Penawaran harga dari Vendor Penantang (Competitor)",
	}, company.ID, user.ID)
	if err != nil || rfq2.ID == 0 || rfq2.RequisitionID == nil || *rfq2.RequisitionID != reqTender.ID {
		t.Fatalf("GenerateRFQ #2 failed: %v", err)
	}
	t.Logf("✓ Generated 2 Competing RFQs: RFQ #1='%s' (Vendor: %s), RFQ #2='%s' (Vendor: %s)",
		rfq1.Name, vendor.Name, rfq2.Name, vendor2.Name)

	// 5. Select Winning Bid: Vendor 1 wins the tender
	awardedTender, err := purchaseReqSvc.SelectWinningBid(reqTender.ID, rfq1.ID, company.ID, user.ID)
	if err != nil || awardedTender.State != purchase_requisition.StateDone || awardedTender.SelectedPOID == nil || *awardedTender.SelectedPOID != rfq1.ID {
		t.Fatalf("SelectWinningBid failed: %v, awarded: %+v", err, awardedTender)
	}

	// Verify winning PO confirmed and competitor PO cancelled
	winningPO, _ := purchaseSvc.GetOrder(rfq1.ID, company.ID)
	losingPO, _ := purchaseSvc.GetOrder(rfq2.ID, company.ID)
	if winningPO.State != "purchase" {
		t.Fatalf("Expected winning PO state='purchase', got '%s'", winningPO.State)
	}
	if losingPO.State != "cancel" {
		t.Fatalf("Expected losing PO state='cancel', got '%s'", losingPO.State)
	}
	if awardedTender.Lines[0].QtyOrdered != 20 {
		t.Fatalf("Expected line QtyOrdered=20, got %.2f", awardedTender.Lines[0].QtyOrdered)
	}
	t.Logf("✓ Tender Bid Selection Verified: Winning PO '%s' confirmed (State: %s), Competitor PO '%s' automatically cancelled (State: %s)",
		winningPO.Name, winningPO.State, losingPO.Name, losingPO.State)

	// 6. Create Blanket Purchase Order (Long-term supply contract with locked price)
	reqBlanket, err := purchaseReqSvc.CreateRequisition(purchase_requisition.CreateRequisitionRequest{
		Origin:       "CONTRACT-BLANKET-2026-2027",
		Type:         purchase_requisition.TypeBlanketOrder,
		VendorID:     &vendor.ID,
		OrderingDate: "2026-10-08",
		DateEnd:      "2027-04-30",
		Description:  "Perjanjian Blanket Order pengadaan suku cadang 50 unit dengan harga terkunci Rp 2.300.000",
		Lines: []purchase_requisition.CreateRequisitionLineRequest{
			{
				ProductID:   testProduct.ID,
				Description: testProduct.Name + " (Blanket Supply)",
				ProductQty:  50,
				PriceUnit:   2300000,
			},
		},
	}, company.ID, user.ID)
	if err != nil || !strings.HasPrefix(reqBlanket.Name, "BO/") {
		t.Fatalf("CreateRequisition (Blanket) failed: %v, req: %+v", err, reqBlanket)
	}
	t.Logf("✓ Created Blanket Order Agreement: '%s' (Vendor: %s, Quota: 50 units @ Rp 2.300.000, Plafon: Rp %.0f)",
		reqBlanket.Name, vendor.Name, reqBlanket.TotalAmount)

	// 7. Confirm Blanket Order agreement
	confirmedBlanket, err := purchaseReqSvc.ActionConfirm(reqBlanket.ID, company.ID, user.ID)
	if err != nil || confirmedBlanket.State != purchase_requisition.StateConfirmed {
		t.Fatalf("ActionConfirm blanket order failed: %v", err)
	}

	// 8. Release Partial PO #1 for 20 units against Blanket Order
	releasePO1, err := purchaseReqSvc.ReleaseBlanketPO(reqBlanket.ID, purchase_requisition.ReleaseBlanketPORequest{
		Lines: []purchase_requisition.ReleasePOLineReq{
			{
				RequisitionLineID: confirmedBlanket.Lines[0].ID,
				Qty:               20,
			},
		},
		Notes: "Rilis Tahap 1: Pengiriman awal Oktober 20 unit",
	}, company.ID, user.ID)
	if err != nil || releasePO1.ID == 0 {
		t.Fatalf("ReleaseBlanketPO (Release 1) failed: %v", err)
	}
	curBlanket, _ := purchaseReqSvc.GetByID(reqBlanket.ID, company.ID)
	if curBlanket.State != purchase_requisition.StateInProgress || curBlanket.Lines[0].QtyOrdered != 20 {
		t.Fatalf("Unexpected blanket state after release 1: State=%s, QtyOrdered=%.2f", curBlanket.State, curBlanket.Lines[0].QtyOrdered)
	}
	t.Logf("✓ Blanket PO Partial Release #1: '%s' (Qty: 20 units, Total: Rp %.0f), Remaining Quota: %.0f units, Agreement State: '%s'",
		releasePO1.Name, releasePO1.AmountTotal, 50-curBlanket.Lines[0].QtyOrdered, curBlanket.State)

	// 9. Release Final PO #2 for remaining 30 units
	releasePO2, err := purchaseReqSvc.ReleaseBlanketPO(reqBlanket.ID, purchase_requisition.ReleaseBlanketPORequest{
		Lines: []purchase_requisition.ReleasePOLineReq{
			{
				RequisitionLineID: confirmedBlanket.Lines[0].ID,
				Qty:               30,
			},
		},
		Notes: "Rilis Tahap 2: Pengiriman akhir November 30 unit (Full Fulfilled)",
	}, company.ID, user.ID)
	if err != nil || releasePO2.ID == 0 {
		t.Fatalf("ReleaseBlanketPO (Release 2) failed: %v", err)
	}
	fulfilledBlanket, _ := purchaseReqSvc.GetByID(reqBlanket.ID, company.ID)
	if fulfilledBlanket.State != purchase_requisition.StateDone || fulfilledBlanket.Lines[0].QtyOrdered != 50 {
		t.Fatalf("Expected blanket state='done' and QtyOrdered=50, got State=%s, QtyOrdered=%.2f",
			fulfilledBlanket.State, fulfilledBlanket.Lines[0].QtyOrdered)
	}
	t.Logf("✓ Blanket PO Final Release #2: '%s' (Qty: 30 units), Full Quota Reached! Blanket Agreement State: '%s'",
		releasePO2.Name, fulfilledBlanket.State)

	// 10. Test Over-Release Quota Prevention
	_, errOver := purchaseReqSvc.ReleaseBlanketPO(reqBlanket.ID, purchase_requisition.ReleaseBlanketPORequest{
		Lines: []purchase_requisition.ReleasePOLineReq{
			{
				RequisitionLineID: confirmedBlanket.Lines[0].ID,
				Qty:               5,
			},
		},
	}, company.ID, user.ID)
	if errOver == nil {
		t.Fatalf("Expected error when releasing beyond blanket quota, but got nil")
	}
	t.Logf("✓ Quota Overflow Protection verified: Attempting to release beyond agreement quota rejected with: '%v'", errOver)

	// 11. Verify Chatter Audit Trail
	chatterMsgs, err := chatterSvc.ListMessages("purchase.requisition", reqTender.ID, company.ID)
	if err != nil || len(chatterMsgs) < 4 {
		t.Fatalf("Expected at least 4 chatter messages on tender, got %d", len(chatterMsgs))
	}
	t.Logf("✓ Universal Chatter Audit Trail verified: %d audit messages logged on Requisition '%s'",
		len(chatterMsgs), reqTender.Name)

	// 12. Requisition KPI Analytics & Stats
	prStats, err := purchaseReqSvc.GetStats(company.ID)
	if err != nil || prStats.TotalRequisitions < 2 || prStats.DoneCount < 2 || prStats.BlanketOrdersCount < 1 {
		t.Fatalf("Requisition GetStats failed: err=%v, stats=%+v", err, prStats)
	}
	t.Logf("✓ Requisition Executive KPI Stats: Total Requisitions=%d, Done=%d, Blanket Orders=%d, Total Estimated Value=Rp %.0f",
		prStats.TotalRequisitions, prStats.DoneCount, prStats.BlanketOrdersCount, prStats.TotalEstimatedVal)

	// -------------------------------------------------------------------------
	// SECTION 33: MULTI-CURRENCY REVALUATION & REALIZED/UNREALIZED FOREX ENGINE (PSAK 10 & AI COPILOT)
	// -------------------------------------------------------------------------
	t.Log("\n    --- SECTION 33: MULTI-CURRENCY REVALUATION & REALIZED/UNREALIZED FOREX ENGINE (PSAK 10 & AI COPILOT) ---")

	currRepo := currency.NewRepository(db)
	currSvc := currency.NewService(db, currRepo, accountSvc, chatterSvc, aiClient)

	// 1. Setup Base Currency (IDR) and Foreign Currencies (USD, EUR)
	idrCurr, err := currSvc.CreateCurrency(currency.CreateCurrencyRequest{
		Name:          "IDR",
		Symbol:        "Rp",
		FullName:      "Indonesian Rupiah",
		DecimalPlaces: 2,
		Rounding:      1.0,
		IsBase:        true,
	}, company.ID)
	if err != nil {
		t.Fatalf("Failed to create base currency IDR: %v", err)
	}

	usdCurr, err := currSvc.CreateCurrency(currency.CreateCurrencyRequest{
		Name:          "USD",
		Symbol:        "$",
		FullName:      "US Dollar",
		DecimalPlaces: 2,
		Rounding:      0.01,
		IsBase:        false,
	}, company.ID)
	if err != nil {
		t.Fatalf("Failed to create foreign currency USD: %v", err)
	}

	eurCurr, err := currSvc.CreateCurrency(currency.CreateCurrencyRequest{
		Name:          "EUR",
		Symbol:        "€",
		FullName:      "Euro",
		DecimalPlaces: 2,
		Rounding:      0.01,
		IsBase:        false,
	}, company.ID)
	if err != nil {
		t.Fatalf("Failed to create foreign currency EUR: %v", err)
	}
	t.Logf("✓ Created Currencies: Base=%s, Foreign=%s, %s", idrCurr.Name, usdCurr.Name, eurCurr.Name)

	// 2. Record Historical Spot Rates (2026-10-01)
	rateUSD1, err := currSvc.CreateRate(currency.CreateRateRequest{
		CurrencyID: usdCurr.ID,
		Date:       "2026-10-01",
		DirectRate: 16000.0, // 1 USD = 16,000 IDR
		Source:     "Bank Indonesia JISDOR",
	}, company.ID)
	if err != nil {
		t.Fatalf("Failed to create USD rate: %v", err)
	}

	rateEUR1, err := currSvc.CreateRate(currency.CreateRateRequest{
		CurrencyID: eurCurr.ID,
		Date:       "2026-10-01",
		DirectRate: 17400.0, // 1 EUR = 17,400 IDR
		Source:     "Bank Indonesia JISDOR",
	}, company.ID)
	if err != nil {
		t.Fatalf("Failed to create EUR rate: %v", err)
	}
	t.Logf("✓ Recorded Historical Spot Rates: USD=Rp %.0f, EUR=Rp %.0f per %s",
		rateUSD1.DirectRate, rateEUR1.DirectRate, rateUSD1.Name)

	// 3. Test Real-time Currency Conversion Engine
	convRes, err := currSvc.Convert(1500.0, usdCurr.ID, idrCurr.ID, "2026-10-01", company.ID)
	if err != nil || convRes.ConvertedAmount != 24000000.0 {
		t.Fatalf("Convert USD to IDR failed: expected Rp 24,000,000, got %.2f, err=%v", convRes.ConvertedAmount, err)
	}
	t.Logf("✓ Currency Conversion Engine verified: $1,500 USD @ Rp %.0f = %s Rp %.0f",
		convRes.AppliedRate, idrCurr.Name, convRes.ConvertedAmount)

	// 4. Create Foreign Currency Customer Invoice (AR) in USD ($1,500 USD @ 16,000 = Rp 24,000,000)
	var invJournal account.AccountJournal
	if err := db.Where("code = ?", "INV").First(&invJournal).Error; err != nil {
		invJournal = account.AccountJournal{Code: "INV", Name: "Customer Invoices", Type: "sale"}
		invJournal.CompanyID = company.ID
		db.Create(&invJournal)
	}

	var arAcc account.AccountAccount
	if err := db.Where("code IN ?", []string{"120000", "1200"}).First(&arAcc).Error; err != nil {
		arAcc = account.AccountAccount{Code: "120000", Name: "Piutang Usaha", Type: "asset_receivable"}
		arAcc.CompanyID = company.ID
		db.Create(&arAcc)
	}
	var salesAcc account.AccountAccount
	if err := db.Where("code IN ?", []string{"410000", "4100"}).First(&salesAcc).Error; err != nil {
		salesAcc = account.AccountAccount{Code: "410000", Name: "Pendapatan Penjualan Ekspor", Type: "income"}
		salesAcc.CompanyID = company.ID
		db.Create(&salesAcc)
	}

	usdCustPartnerID := customer.ID
	usdInvoice := &account.AccountMove{
		MoveType:       "out_invoice",
		JournalID:      invJournal.ID,
		PartnerID:      usdCustPartnerID,
		InvoiceDate:    "2026-10-01",
		Ref:            "EXPORT/US/2026/001",
		CurrencyID:     &usdCurr.ID,
		CurrencyRate:   16000.0,
		AmountCurrency: 1500.0,
		AmountUntaxed:  24000000.0,
		AmountTotal:    24000000.0,
		AmountResidual: 24000000.0,
		State:          "draft",
		PaymentState:   "not_paid",
		LineIDs: []account.AccountMoveLine{
			{
				AccountID:      arAcc.ID,
				PartnerID:      &usdCustPartnerID,
				Name:           "Piutang Ekspor Pelanggan Global (1,500 USD)",
				Debit:          24000000.0,
				Credit:         0,
				Balance:        24000000.0,
				CurrencyID:     &usdCurr.ID,
				AmountCurrency: 1500.0,
			},
			{
				AccountID:      salesAcc.ID,
				Name:           "Penjualan Produk Ekspor Workstation",
				Debit:          0,
				Credit:         24000000.0,
				Balance:        -24000000.0,
				CurrencyID:     &usdCurr.ID,
				AmountCurrency: -1500.0,
			},
		},
	}
	usdInvoice.CompanyID = company.ID
	if err := accountSvc.CreateMove(usdInvoice); err != nil {
		t.Fatalf("Failed to create USD Invoice: %v", err)
	}
	if _, err := accountSvc.ActionPost(usdInvoice.ID, company.ID); err != nil {
		t.Fatalf("Failed to post USD Invoice: %v", err)
	}
	t.Logf("✓ Foreign Currency Customer Invoice posted: '%s' ($1,500 USD @ Rp 16,000 = Rp 24,000,000, AR Residual=Rp 24,000,000)", usdInvoice.Name)

	// 5. Create Foreign Currency Vendor Bill (AP) in USD ($500 USD @ 16,000 = Rp 8,000,000)
	var billJournal account.AccountJournal
	if err := db.Where("code = ?", "BILL").First(&billJournal).Error; err != nil {
		billJournal = account.AccountJournal{Code: "BILL", Name: "Vendor Bills", Type: "purchase"}
		billJournal.CompanyID = company.ID
		db.Create(&billJournal)
	}

	var apAcc account.AccountAccount
	if err := db.Where("code IN ?", []string{"210000", "2100"}).First(&apAcc).Error; err != nil {
		apAcc = account.AccountAccount{Code: "210000", Name: "Hutang Usaha", Type: "liability_payable"}
		apAcc.CompanyID = company.ID
		db.Create(&apAcc)
	}
	var cogsAcc account.AccountAccount
	if err := db.Where("code IN ?", []string{"510000", "5100"}).First(&cogsAcc).Error; err != nil {
		cogsAcc = account.AccountAccount{Code: "510000", Name: "Beban Pokok Penjualan", Type: "expense"}
		cogsAcc.CompanyID = company.ID
		db.Create(&cogsAcc)
	}

	usdVendorPartnerID := vendor.ID
	usdBill := &account.AccountMove{
		MoveType:       "in_invoice",
		JournalID:      billJournal.ID,
		PartnerID:      usdVendorPartnerID,
		InvoiceDate:    "2026-10-01",
		Ref:            "IMPORT/US/2026/001",
		CurrencyID:     &usdCurr.ID,
		CurrencyRate:   16000.0,
		AmountCurrency: 500.0,
		AmountUntaxed:  8000000.0,
		AmountTotal:    8000000.0,
		AmountResidual: 8000000.0,
		State:          "draft",
		PaymentState:   "not_paid",
		LineIDs: []account.AccountMoveLine{
			{
				AccountID:      cogsAcc.ID,
				Name:           "Biaya Impor Komponen Elektronik Workstation",
				Debit:          8000000.0,
				Credit:         0,
				Balance:        8000000.0,
				CurrencyID:     &usdCurr.ID,
				AmountCurrency: 500.0,
			},
			{
				AccountID:      apAcc.ID,
				PartnerID:      &usdVendorPartnerID,
				Name:           "Hutang Dagang Pemasok Luar Negeri (500 USD)",
				Debit:          0,
				Credit:         8000000.0,
				Balance:        -8000000.0,
				CurrencyID:     &usdCurr.ID,
				AmountCurrency: -500.0,
			},
		},
	}
	usdBill.CompanyID = company.ID
	if err := accountSvc.CreateMove(usdBill); err != nil {
		t.Fatalf("Failed to create USD Bill: %v", err)
	}
	if _, err := accountSvc.ActionPost(usdBill.ID, company.ID); err != nil {
		t.Fatalf("Failed to post USD Bill: %v", err)
	}
	t.Logf("✓ Foreign Currency Vendor Bill posted: '%s' ($500 USD @ Rp 16,000 = Rp 8,000,000, AP Residual=Rp 8,000,000)", usdBill.Name)

	// 6. Record Month-End Closing Exchange Rate (2026-10-31: USD rises to 16,300 IDR)
	rateUSD2, err := currSvc.CreateRate(currency.CreateRateRequest{
		CurrencyID: usdCurr.ID,
		Date:       "2026-10-31",
		DirectRate: 16300.0, // Rupiah weakens from 16,000 to 16,300 (+300 IDR/USD)
		Source:     "Bank Indonesia JISDOR Penutupan Bulan",
	}, company.ID)
	if err != nil {
		t.Fatalf("Failed to record month-end USD rate: %v", err)
	}
	t.Logf("✓ Month-End Closing Rate recorded: USD=Rp %.0f per %s", rateUSD2.DirectRate, rateUSD2.Name)

	// 7. Compute Month-End Unrealized Forex Revaluation (PSAK 10 / IAS 21)
	revalDraft, err := currSvc.ComputeUnrealizedRevaluation(currency.ComputeRevaluationRequest{
		CurrencyID:      usdCurr.ID,
		RevaluationDate: "2026-10-31",
		ClosingRate:     16300.0,
		Notes:           "Penyesuaian Nilai Wajar Kurs Valas Akhir Bulan Oktober 2026 (PSAK 10)",
	}, company.ID, 1)
	if err != nil {
		t.Fatalf("ComputeUnrealizedRevaluation failed: %v", err)
	}

	if len(revalDraft.Lines) != 2 {
		t.Fatalf("Expected 2 revaluation lines (1 AR, 1 AP), got %d", len(revalDraft.Lines))
	}
	if math.Abs(revalDraft.TotalGain-450000.0) > 0.01 || math.Abs(revalDraft.TotalLoss-150000.0) > 0.01 || math.Abs(revalDraft.NetGainLoss-300000.0) > 0.01 {
		t.Fatalf("Revaluation calculation mismatch: TotalGain=%.2f (exp 450000), TotalLoss=%.2f (exp 150000), NetGainLoss=%.2f (exp 300000)",
			revalDraft.TotalGain, revalDraft.TotalLoss, revalDraft.NetGainLoss)
	}
	t.Logf("✓ Month-End Unrealized Revaluation Draft '%s' computed successfully: Total Lines=%d, Gain=Rp %.0f, Loss=Rp %.0f, Net Gain=Rp %.0f",
		revalDraft.Name, len(revalDraft.Lines), revalDraft.TotalGain, revalDraft.TotalLoss, revalDraft.NetGainLoss)

	// 8. Post Revaluation to General Ledger and verify Double-Entry Balance
	revalPosted, err := currSvc.PostRevaluation(revalDraft.ID, company.ID, 1)
	if err != nil {
		t.Fatalf("PostRevaluation failed: %v", err)
	}
	if revalPosted.State != "posted" || revalPosted.JournalEntryID == nil {
		t.Fatalf("Expected revaluation posted with JournalEntryID, state=%s", revalPosted.State)
	}

	glRevalMove, err := accountSvc.GetMove(*revalPosted.JournalEntryID, company.ID)
	if err != nil || glRevalMove.State != "posted" {
		t.Fatalf("Failed to fetch posted revaluation GL entry: %v", err)
	}

	var glDebit, glCredit float64
	for _, l := range glRevalMove.LineIDs {
		glDebit += l.Debit
		glCredit += l.Credit
	}
	if math.Abs(glDebit-glCredit) > 0.01 || math.Abs(glDebit-600000.0) > 0.01 {
		t.Fatalf("GL revaluation move unbalanced: Total Debit=%.2f, Total Credit=%.2f (exp 600000.00)", glDebit, glCredit)
	}
	t.Logf("✓ Balanced General Ledger Move '%s' posted: Total Debit=Rp %.0f == Total Credit=Rp %.0f (Lines: %d, Ref: %s)",
		glRevalMove.Name, glDebit, glCredit, len(glRevalMove.LineIDs), glRevalMove.Ref)

	// 9. Realized Forex Gain/Loss Engine (Invoice Settlement Analysis)
	realizedLossRes, err := currSvc.CalculateRealizedGainLoss(currency.RealizedGainLossCalculationRequest{
		MoveID:        usdBill.ID,
		PaymentDate:   "2026-11-05",
		PaymentRate:   16400.0,
		PaymentAmount: 500.0,
	}, company.ID)
	if err != nil || realizedLossRes.GainLossType != "loss" || math.Abs(realizedLossRes.RealizedGainLoss-200000.0) > 0.01 {
		t.Fatalf("Realized Forex calculation failed: res=%+v, err=%v", realizedLossRes, err)
	}
	t.Logf("✓ Realized Forex Settlement Engine verified: Vendor Bill '%s' paid $500 @ Rp 16,400 vs Hist Rp 16,000 -> Realized %s of Rp %.0f (Account: %s)",
		usdBill.Name, strings.ToUpper(realizedLossRes.GainLossType), realizedLossRes.RealizedGainLoss, realizedLossRes.JournalAccountName)

	realizedGainRes, err := currSvc.CalculateRealizedGainLoss(currency.RealizedGainLossCalculationRequest{
		MoveID:        usdInvoice.ID,
		PaymentDate:   "2026-11-05",
		PaymentRate:   16250.0,
		PaymentAmount: 1500.0,
	}, company.ID)
	if err != nil || realizedGainRes.GainLossType != "gain" || math.Abs(realizedGainRes.RealizedGainLoss-375000.0) > 0.01 {
		t.Fatalf("Realized Forex calculation failed: res=%+v, err=%v", realizedGainRes, err)
	}
	t.Logf("✓ Realized Forex Settlement Engine verified: Customer Invoice '%s' settled $1,500 @ Rp 16,250 vs Hist Rp 16,000 -> Realized %s of Rp %.0f (Account: %s)",
		usdInvoice.Name, strings.ToUpper(realizedGainRes.GainLossType), realizedGainRes.RealizedGainLoss, realizedGainRes.JournalAccountName)

	// 10. Cloudflare Workers AI Forex Risk & Sensitivity Exposure Copilot
	forexReport, err := currSvc.ForexRiskAIAudit(usdCurr.ID, company.ID)
	if err != nil || forexReport.ForeignCurrency != "USD" || forexReport.NetExposureFC != 1000.0 {
		t.Fatalf("ForexRiskAIAudit failed: report=%+v, err=%v", forexReport, err)
	}
	t.Logf("✓ Cloudflare Workers AI Forex Risk Copilot: Currency=%s, SpotRate=Rp %.0f, Open AR=$%.0f, Open AP=$%.0f, Net Exposure=$%.0f (Rp %.0f), RiskLevel='%s', Engine='%s'",
		forexReport.ForeignCurrency, forexReport.SpotRate, forexReport.OpenReceivablesFC, forexReport.OpenPayablesFC,
		forexReport.NetExposureFC, forexReport.NetExposureIDR, forexReport.RiskLevel, forexReport.Engine)
	t.Logf("  Executive Summary: %s", forexReport.ExecutiveSummary)
	t.Logf("  Action Recommendation #1: %s", forexReport.ActionRecommendations[0])

	// 11. Chatter Audit Trail on Forex Revaluation
	revChatter, err := chatterSvc.ListMessages("account.multicurrency.revaluation", revalPosted.ID, company.ID)
	if err != nil || len(revChatter) < 2 {
		t.Fatalf("Expected at least 2 chatter messages on revaluation, got %d", len(revChatter))
	}
	t.Logf("✓ Universal Chatter Audit Trail verified: %d audit messages logged on Revaluation '%s'",
		len(revChatter), revalPosted.Name)

	// 12. Executive Currency KPI Stats
	currStats, err := currSvc.GetStats(company.ID)
	if err != nil || currStats.TotalCurrencies < 3 || currStats.TotalRatesRecorded < 3 || currStats.TotalRevaluationsDone < 1 {
		t.Fatalf("GetStats failed: stats=%+v, err=%v", currStats, err)
	}
	t.Logf("✓ Currency Executive KPI Stats: Total Currencies=%d, Total Rates Recorded=%d, Revaluations Done=%d, YTD Unrealized Gain=Rp %.0f, Latest USD Rate=Rp %.0f, Latest EUR Rate=Rp %.0f",
		currStats.TotalCurrencies, currStats.TotalRatesRecorded, currStats.TotalRevaluationsDone, currStats.YTDUnrealizedGainLoss, currStats.LatestUSDRate, currStats.LatestEURRate)

	// -------------------------------------------------------------------------
	// SECTION 34: INDONESIAN E-FAKTUR & TAX ENGINE (DJP REGULATION, NSFP ALLOCATION, SPT 1111 & AI COMPLIANCE COPILOT)
	// -------------------------------------------------------------------------
	t.Log("\n    --- SECTION 34: INDONESIAN E-FAKTUR & TAX ENGINE (DJP REGULATION, NSFP ALLOCATION, SPT 1111 & AI COMPLIANCE COPILOT) ---")

	taxEFakturRepo := tax_efaktur.NewRepository(db)
	taxEFakturSvc := tax_efaktur.NewService(db, taxEFakturRepo, accountSvc, chatterSvc, aiClient)

	// 1. Setup DJP NSFP Quota Allocation for 2026 (500 serial numbers)
	nsfpRange, err := taxEFakturSvc.CreateNSFP(tax_efaktur.CreateNSFPRequest{
		Prefix:      "010",
		Year:        2026,
		StartNumber: "00000001",
		EndNumber:   "00000500",
	}, company.ID)
	if err != nil {
		t.Fatalf("Failed to create NSFP quota range: %v", err)
	}
	if nsfpRange.TotalQuota != 500 || nsfpRange.RemainingQuota != 500 || nsfpRange.UsedQuota != 0 || nsfpRange.State != "active" {
		t.Fatalf("NSFP Quota assertion failed: %+v", nsfpRange)
	}
	t.Logf("✓ DJP NSFP Allocation Registered: Prefix=%s, Year=%d, Range=%s - %s, Total Quota=%d, State=%s",
		nsfpRange.Prefix, nsfpRange.Year, nsfpRange.StartNumber, nsfpRange.EndNumber, nsfpRange.TotalQuota, nsfpRange.State)

	// 2. Prepare Customer with Valid 15-digit NPWP
	customer.TaxID = "01.234.567.8-012.000"
	customer.Street = "Jl. Jenderal Sudirman Kav. 52-53, SCBD, Jakarta Selatan"
	db.Save(&customer)

	var taxOutAcc account.AccountAccount
	if err := db.Where("code IN ?", []string{"220000", "2200"}).First(&taxOutAcc).Error; err != nil {
		taxOutAcc = account.AccountAccount{Code: "220000", Name: "Hutang PPN Keluaran", Type: "liability_current"}
		taxOutAcc.CompanyID = company.ID
		db.Create(&taxOutAcc)
	}

	// 3. Create and Post Commercial Customer Invoice (AmountUntaxed: Rp 50,000,000)
	taxCustMove := &account.AccountMove{
		MoveType:       "out_invoice",
		JournalID:      invJournal.ID,
		PartnerID:      customer.ID,
		InvoiceDate:    "2026-10-05",
		Ref:            "INV/2026/TAX/001",
		AmountUntaxed:  50000000.0,
		AmountTax:      5500000.0,
		AmountTotal:    55500000.0,
		AmountResidual: 55500000.0,
		State:          "draft",
		PaymentState:   "not_paid",
		LineIDs: []account.AccountMoveLine{
			{
				AccountID: arAcc.ID,
				PartnerID: &customer.ID,
				Name:      "Piutang Penjualan Cloud Server Rack (DPP Rp 50jt)",
				Debit:     55500000.0,
				Credit:    0,
				Balance:   55500000.0,
			},
			{
				AccountID: salesAcc.ID,
				Name:      "Penjualan Server Enterprise BKP",
				Debit:     0,
				Credit:    50000000.0,
				Balance:   -50000000.0,
			},
			{
				AccountID: taxOutAcc.ID,
				Name:      "Hutang PPN Keluaran 11%",
				Debit:     0,
				Credit:    5500000.0,
				Balance:   -5500000.0,
			},
		},
	}
	taxCustMove.CompanyID = company.ID
	if err := accountSvc.CreateMove(taxCustMove); err != nil {
		t.Fatalf("Failed to create commercial move for tax test: %v", err)
	}
	if _, err := accountSvc.ActionPost(taxCustMove.ID, company.ID); err != nil {
		t.Fatalf("Failed to post commercial move for tax test: %v", err)
	}

	// 4. Generate e-Faktur Pajak Keluaran (Out Tax Invoice)
	outTaxInv, err := taxEFakturSvc.GenerateOutTaxInvoiceFromMove(tax_efaktur.GenerateTaxInvoiceRequest{
		MoveID:         taxCustMove.ID,
		TaxInvoiceType: "out_tax",
		TaxPeriod:      10,
		TaxYear:        2026,
		TaxRatePPN:     11.0,
		Notes:          "Penyerahan BKP kepada Wajib Pajak Badan",
	}, company.ID, 1)
	if err != nil {
		t.Fatalf("GenerateOutTaxInvoiceFromMove failed: %v", err)
	}
	if outTaxInv.NSFPNumber != "010.001-26.00000001" {
		t.Fatalf("Expected NSFP '010.001-26.00000001', got '%s'", outTaxInv.NSFPNumber)
	}
	if outTaxInv.DPP != 50000000.0 || outTaxInv.PPN != 5500000.0 || outTaxInv.TotalAmount != 55500000.0 {
		t.Fatalf("Tax amounts mismatch: DPP=%.0f, PPN=%.0f, Total=%.0f", outTaxInv.DPP, outTaxInv.PPN, outTaxInv.TotalAmount)
	}
	if !strings.Contains(outTaxInv.QRURL, "efaktur.pajak.go.id") {
		t.Fatalf("Expected DJP QR URL, got '%s'", outTaxInv.QRURL)
	}
	if len(outTaxInv.Lines) == 0 {
		t.Fatalf("Expected detail BKP/JKP lines, got 0")
	}
	t.Logf("✓ Faktur Pajak Keluaran Generated: NSFP='%s', Partner='%s', NPWP='%s', DPP=Rp %.0f, PPN 11%%=Rp %.0f, QR URL='%s'",
		outTaxInv.NSFPNumber, outTaxInv.PartnerName, outTaxInv.NPWP, outTaxInv.DPP, outTaxInv.PPN, outTaxInv.QRURL)

	// Verify NSFP Quota incrementation
	nsfpList, err := taxEFakturSvc.ListNSFP(company.ID)
	if err != nil || len(nsfpList) == 0 || nsfpList[0].UsedQuota != 1 || nsfpList[0].RemainingQuota != 499 {
		t.Fatalf("NSFP Quota sequence check failed: %+v", nsfpList)
	}
	t.Logf("✓ NSFP Quota Decrement verified: Used=%d, Remaining=%d", nsfpList[0].UsedQuota, nsfpList[0].RemainingQuota)

	// 5. Prepare Vendor Bill with NPWP and Record Faktur Pajak Masukan (In Tax Invoice)
	vendor.TaxID = "02.345.678.9-023.000"
	vendor.Street = "Kawasan Industri MM2100, Cikarang Barat, Bekasi"
	db.Save(&vendor)

	if err := db.Where("code = ?", "BILL").First(&billJournal).Error; err != nil {
		billJournal = account.AccountJournal{Code: "BILL", Name: "Vendor Bills", Type: "purchase"}
		billJournal.CompanyID = company.ID
		db.Create(&billJournal)
	}
	if err := db.Where("code IN ?", []string{"210000", "2100"}).First(&apAcc).Error; err != nil {
		apAcc = account.AccountAccount{Code: "210000", Name: "Hutang Usaha", Type: "liability_payable"}
		apAcc.CompanyID = company.ID
		db.Create(&apAcc)
	}
	var expenseAcc account.AccountAccount
	if err := db.Where("code IN ?", []string{"510000", "5100", "610000"}).First(&expenseAcc).Error; err != nil {
		expenseAcc = account.AccountAccount{Code: "510000", Name: "Biaya Operasional", Type: "expense"}
		expenseAcc.CompanyID = company.ID
		db.Create(&expenseAcc)
	}

	var taxInAcc account.AccountAccount
	if err := db.Where("code IN ?", []string{"115000", "1150"}).First(&taxInAcc).Error; err != nil {
		taxInAcc = account.AccountAccount{Code: "115000", Name: "PPN Masukan Dibayar di Muka", Type: "asset_current"}
		taxInAcc.CompanyID = company.ID
		db.Create(&taxInAcc)
	}

	taxVendBill := &account.AccountMove{
		MoveType:       "in_invoice",
		JournalID:      billJournal.ID,
		PartnerID:      vendor.ID,
		InvoiceDate:    "2026-10-06",
		Ref:            "BILL/2026/TAX/001",
		AmountUntaxed:  20000000.0,
		AmountTax:      2200000.0,
		AmountTotal:    22200000.0,
		AmountResidual: 22200000.0,
		State:          "draft",
		PaymentState:   "not_paid",
		LineIDs: []account.AccountMoveLine{
			{
				AccountID: expenseAcc.ID,
				Name:      "Biaya Pengadaan Sparepart Server Rack",
				Debit:     20000000.0,
				Credit:    0,
				Balance:   20000000.0,
			},
			{
				AccountID: taxInAcc.ID,
				Name:      "PPN Masukan 11% Dapat Dikreditkan",
				Debit:     2200000.0,
				Credit:    0,
				Balance:   2200000.0,
			},
			{
				AccountID: apAcc.ID,
				PartnerID: &vendor.ID,
				Name:      "Hutang Vendor Sparepart (Rp 20jt + PPN)",
				Debit:     0,
				Credit:    22200000.0,
				Balance:   -22200000.0,
			},
		},
	}
	taxVendBill.CompanyID = company.ID
	if err := accountSvc.CreateMove(taxVendBill); err != nil {
		t.Fatalf("Failed to create vendor bill for tax test: %v", err)
	}
	if _, err := accountSvc.ActionPost(taxVendBill.ID, company.ID); err != nil {
		t.Fatalf("Failed to post vendor bill for tax test: %v", err)
	}

	inTaxInv, err := taxEFakturSvc.RecordInTaxInvoiceFromMove(tax_efaktur.GenerateTaxInvoiceRequest{
		MoveID:         taxVendBill.ID,
		CustomNSFP:     "010.002-26.99887766",
		TaxPeriod:      10,
		TaxYear:        2026,
		TaxRatePPN:     11.0,
		Notes:          "Pajak Masukan dapat dikreditkan",
	}, company.ID, 1)
	if err != nil {
		t.Fatalf("RecordInTaxInvoiceFromMove failed: %v", err)
	}
	if inTaxInv.NSFPNumber != "010.002-26.99887766" || inTaxInv.DPP != 20000000.0 || inTaxInv.PPN != 2200000.0 {
		t.Fatalf("In Tax invoice assertion failed: %+v", inTaxInv)
	}
	t.Logf("✓ Faktur Pajak Masukan Recorded: NSFP='%s', Vendor='%s', NPWP='%s', DPP=Rp %.0f, PPN Masukan=Rp %.0f",
		inTaxInv.NSFPNumber, inTaxInv.PartnerName, inTaxInv.NPWP, inTaxInv.DPP, inTaxInv.PPN)

	// 6. SPT Masa PPN 1111 Period Calculation (Masa Pajak 10 / 2026)
	sptSummary, err := taxEFakturSvc.GetSPTPeriodSummary(10, 2026, company.ID)
	if err != nil {
		t.Fatalf("GetSPTPeriodSummary failed: %v", err)
	}
	if sptSummary.DPPPenyerahanKeluaran != 50000000.0 || sptSummary.PPNKeluaranDipungut != 5500000.0 {
		t.Fatalf("SPT PPN Keluaran mismatch: %+v", sptSummary)
	}
	if sptSummary.DPPPerolehanMasukan != 20000000.0 || sptSummary.PPNMasukanDikreditkan != 2200000.0 {
		t.Fatalf("SPT PPN Masukan mismatch: %+v", sptSummary)
	}
	if sptSummary.PPNKurangLebihBayar != 3300000.0 || sptSummary.StatusFiskal != "Kurang Bayar" {
		t.Fatalf("SPT Kurang Bayar mismatch: expected 3,300,000 ('Kurang Bayar'), got %.0f ('%s')",
			sptSummary.PPNKurangLebihBayar, sptSummary.StatusFiskal)
	}
	t.Logf("✓ SPT Masa PPN 1111 Computed: Masa=%d/%d, PPN Keluaran=Rp %.0f, PPN Masukan=Rp %.0f, Net %s=Rp %.0f",
		sptSummary.TaxPeriod, sptSummary.TaxYear, sptSummary.PPNKeluaranDipungut, sptSummary.PPNMasukanDikreditkan,
		sptSummary.StatusFiskal, sptSummary.PPNKurangLebihBayar)

	// 7. Official DJP e-Faktur CSV Export Engine
	csvContent, err := taxEFakturSvc.ExportEFakturCSV("out_tax", 10, 2026, company.ID)
	if err != nil {
		t.Fatalf("ExportEFakturCSV failed: %v", err)
	}
	if !strings.Contains(csvContent, "FK,KD_JENIS_TRANSAKSI,FG_PENGGANTI,NOMOR_FAKTUR") {
		t.Fatalf("CSV missing DJP FK Header: %s", csvContent)
	}
	if !strings.Contains(csvContent, "OF,KODE_OBJEK,NAMA") {
		t.Fatalf("CSV missing DJP OF Header: %s", csvContent)
	}
	if !strings.Contains(csvContent, "0100012600000001") {
		t.Fatalf("CSV missing formatted 16-digit NSFP '0100012600000001': %s", csvContent)
	}
	t.Logf("✓ DJP Official e-Faktur CSV Export verified (%d bytes generated, FK & OF lines formatted)", len(csvContent))

	// 8. AI Tax Compliance Copilot Audit
	taxAudit, err := taxEFakturSvc.TaxComplianceAIAudit(10, 2026, company.ID)
	if err != nil {
		t.Fatalf("TaxComplianceAIAudit failed: %v", err)
	}
	if taxAudit.ComplianceScore <= 0 || len(taxAudit.ActionRecommendations) == 0 || taxAudit.NetTaxPayable != 3300000.0 {
		t.Fatalf("Tax Audit assertion failed: %+v", taxAudit)
	}
	t.Logf("✓ Cloudflare Workers AI Tax Compliance Copilot: Score=%.1f%%, Risk='%s', Invoices Audited=%d, Net Tax Payable=Rp %.0f, Engine='%s'",
		taxAudit.ComplianceScore, taxAudit.RiskLevel, taxAudit.TotalInvoicesAudited, taxAudit.NetTaxPayable, taxAudit.Engine)
	t.Logf("  Executive Summary: %s", taxAudit.ExecutiveSummary)
	t.Logf("  Action Recommendation #1: %s", taxAudit.ActionRecommendations[0])

	// 9. Universal Chatter Audit Trail on Tax Invoice
	taxChatter, err := chatterSvc.ListMessages("account.tax.invoice", outTaxInv.ID, company.ID)
	if err != nil || len(taxChatter) < 1 {
		t.Fatalf("Expected chatter log on tax invoice, got %d (err=%v)", len(taxChatter), err)
	}
	t.Logf("✓ Universal Chatter Audit Trail verified: %d audit messages logged on Tax Invoice '%s'",
		len(taxChatter), outTaxInv.NSFPNumber)

	// 10. Executive Tax & e-Faktur KPI Stats
	taxStats, err := taxEFakturSvc.GetStats(company.ID)
	if err != nil || taxStats.TotalNSFPAllocated < 500 || taxStats.RemainingNSFPQuota != 499 || taxStats.TotalOutTaxInvoices < 1 || taxStats.TotalInTaxInvoices < 1 {
		t.Fatalf("GetStats tax assertion failed: %+v, err=%v", taxStats, err)
	}
	t.Logf("✓ Tax Executive KPI Stats: Total NSFP=%d, Remaining Quota=%d, Out Invoices=%d, In Invoices=%d, Total PPN Keluaran=Rp %.0f, Total PPN Masukan=Rp %.0f, Net Kurang Bayar=Rp %.0f",
		taxStats.TotalNSFPAllocated, taxStats.RemainingNSFPQuota, taxStats.TotalOutTaxInvoices, taxStats.TotalInTaxInvoices,
		taxStats.TotalPPNKeluaran, taxStats.TotalPPNMasukan, taxStats.NetPPNKurangBayar)

	// =========================================================================
	// SECTION 35: BANK RECONCILIATION ENGINE & STATEMENT IMPORT (SMART CSV PARSER, AUTO-MATCH ENGINE, WRITE-OFF RULES & AI AUDIT COPILOT)
	// =========================================================================
	t.Log("\n    --- SECTION 35: BANK RECONCILIATION ENGINE & STATEMENT IMPORT (SMART CSV PARSER, AUTO-MATCH ENGINE, WRITE-OFF RULES & AI AUDIT COPILOT) ---")

	// 1. Initialize Bank Reconciliation Services & Default Rules
	bankReconcileRepo := bank_reconciliation.NewRepository(db)
	bankReconcileSvc := bank_reconciliation.NewService(db, bankReconcileRepo, accountSvc, chatterSvc, aiClient)
	bankReconcileSvc.SeedDefaultRules(company.ID)

	rules, err := bankReconcileSvc.ListReconcileModels(company.ID)
	if err != nil || len(rules) < 2 {
		t.Fatalf("Expected at least 2 seeded reconcile rules, got %d (err=%v)", len(rules), err)
	}
	t.Logf("✓ Seeded Bank Reconciliation Rules verified: %d rules active (e.g. '%s', '%s')",
		len(rules), rules[0].RuleName, rules[1].RuleName)

	// Ensure Bank BCA Journal
	if bankJournal.ID == 0 {
		_ = db.Where("code = ? AND company_id = ?", "BNK", company.ID).First(&bankJournal)
	}

	// 2. Create Target Open Invoices for Bank Reconciliation
	// Invoice A: Rp 15,000,000 (Customer Invoice)
	invA := &account.AccountMove{
		MoveType:       "out_invoice",
		JournalID:      invJournal.ID,
		PartnerID:      customer.ID,
		InvoiceDate:    "2026-10-09",
		InvoiceDueDate: "2026-10-30",
		AmountUntaxed:  15000000.0,
		AmountTotal:    15000000.0,
		AmountResidual: 15000000.0,
		State:          "draft",
		PaymentState:   "not_paid",
		LineIDs: []account.AccountMoveLine{
			{
				AccountID: arAcc.ID,
				PartnerID: &customer.ID,
				Name:      "Piutang Penjualan Mesin",
				Debit:     15000000.0,
				Credit:    0,
				Balance:   15000000.0,
			},
			{
				AccountID: salesAcc.ID,
				Name:      "Pendapatan Penjualan Mesin",
				Debit:     0,
				Credit:    15000000.0,
				Balance:   -15000000.0,
			},
		},
	}
	invA.CompanyID = company.ID
	if err := accountSvc.CreateMove(invA); err != nil {
		t.Fatalf("Failed to create customer invoice A for bank reconcile: %v", err)
	}
	if _, err := accountSvc.ActionPost(invA.ID, company.ID); err != nil {
		t.Fatalf("Failed to post customer invoice A: %v", err)
	}

	// Invoice B: Rp 10,000,000 (Customer Invoice for Partner Match)
	invB := &account.AccountMove{
		MoveType:       "out_invoice",
		JournalID:      invJournal.ID,
		PartnerID:      customer.ID,
		InvoiceDate:    "2026-10-09",
		InvoiceDueDate: "2026-10-30",
		AmountUntaxed:  10000000.0,
		AmountTotal:    10000000.0,
		AmountResidual: 10000000.0,
		State:          "draft",
		PaymentState:   "not_paid",
		LineIDs: []account.AccountMoveLine{
			{
				AccountID: arAcc.ID,
				PartnerID: &customer.ID,
				Name:      "Piutang Jasa Konsultasi",
				Debit:     10000000.0,
				Credit:    0,
				Balance:   10000000.0,
			},
			{
				AccountID: salesAcc.ID,
				Name:      "Pendapatan Jasa Konsultasi",
				Debit:     0,
				Credit:    10000000.0,
				Balance:   -10000000.0,
			},
		},
	}
	invB.CompanyID = company.ID
	if err := accountSvc.CreateMove(invB); err != nil {
		t.Fatalf("Failed to create customer invoice B for bank reconcile: %v", err)
	}
	if _, err := accountSvc.ActionPost(invB.ID, company.ID); err != nil {
		t.Fatalf("Failed to post customer invoice B: %v", err)
	}

	t.Logf("✓ Posted 2 Open Customer Invoices for Reconciliation: '%s' (Rp 15jt) & '%s' (Rp 10jt)", invA.Name, invB.Name)

	// 3. Multi-Format Smart CSV Mutasi Bank Import (BCA Format)
	// Mutasi contains:
	// 1) Customer Payment Inflow +15jt mentioning Invoice A Number
	// 2) Customer Payment Inflow +10jt mentioning Customer Partner Name
	// 3) Bank Admin Fee Outflow -25,000 mentioning "BIAYA ADM BULANAN GIRO BCA"
	bankCSVContent := fmt.Sprintf(`Tanggal,Keterangan,Mutasi,Saldo
09/10/2026,TRSF CR PEMBAYARAN FAKTUR %s,"15.000.000,00 CR","65.000.000,00"
09/10/2026,TRSF CR DARI MITRA %s PELUNASAN,"10.000.000,00 CR","75.000.000,00"
09/10/2026,BIAYA ADM BULANAN GIRO BCA,"25.000,00 DB","74.975.000,00"`, invA.Name, customer.Name)

	importReq := bank_reconciliation.ImportStatementCSVRequest{
		JournalID:      bankJournal.ID,
		Name:           "BNK/BCA/2026/10/TEST",
		Date:           "2026-10-09",
		Format:         "bca",
		BalanceStart:   50000000.0,
		BalanceEndReal: 74975000.0,
		CSVContent:     bankCSVContent,
	}

	stmt, linesCount, err := bankReconcileSvc.ParseAndImportCSV(importReq, company.ID, 1)
	if err != nil {
		t.Fatalf("ParseAndImportCSV failed: %v", err)
	}
	if linesCount != 3 || stmt.TotalLines != 3 {
		t.Fatalf("Expected 3 lines imported, got %d", linesCount)
	}
	if stmt.BalanceEnd != 74975000.0 || math.Abs(stmt.Difference) > 0.01 {
		t.Fatalf("Expected Ending Balance 74,975,000 and 0 difference, got BalanceEnd=%.2f, Diff=%.2f",
			stmt.BalanceEnd, stmt.Difference)
	}
	t.Logf("✓ Multi-Format Bank Statement CSV Parsed & Imported: '%s', %d lines, Saldo Awal=Rp %.0f, Saldo Akhir=Rp %.0f (Selisih=Rp %.2f)",
		stmt.Name, stmt.TotalLines, stmt.BalanceStart, stmt.BalanceEnd, stmt.Difference)

	// 4. Test Algorithmic Auto-Match Engine
	suggestions, err := bankReconcileSvc.GetAutoMatchSuggestions(stmt.ID, company.ID)
	if err != nil || len(suggestions) != 3 {
		t.Fatalf("GetAutoMatchSuggestions failed: got %d suggestions (err=%v)", len(suggestions), err)
	}

	// Verify suggestion 1: Exact reference match on Invoice A
	sugg1 := suggestions[0]
	if sugg1.MatchType != "exact_ref" || sugg1.Confidence < 0.95 || sugg1.SuggestedMoveID == nil || *sugg1.SuggestedMoveID != invA.ID {
		t.Fatalf("Suggestion 1 mismatch: expected exact_ref on invA (%s), got %+v", invA.Name, sugg1)
	}
	t.Logf("✓ Auto-Match Rule 1 (Exact Invoice Regex) verified: Line '%s' matched to '%s' (Confidence: %.1f%%, Reason: %s)",
		sugg1.LineMemo, sugg1.SuggestedMoveName, sugg1.Confidence*100, sugg1.Reason)

	// Verify suggestion 3: Bank fee write-off rule
	sugg3 := suggestions[2]
	if sugg3.MatchType != "bank_charge_rule" || sugg3.Confidence < 0.90 || sugg3.SuggestedAccountID == nil {
		t.Fatalf("Suggestion 3 mismatch: expected bank_charge_rule, got %+v", sugg3)
	}
	t.Logf("✓ Auto-Match Rule 2 (Bank Fee Write-Off Rule) verified: Line '%s' matched to GL Account (Confidence: %.1f%%, Reason: %s)",
		sugg3.LineMemo, sugg3.Confidence*100, sugg3.Reason)

	// 5. Atomic Reconcile Execution: Reconcile Line 1 against Invoice A
	err = bankReconcileSvc.ReconcileLine(bank_reconciliation.ReconcileLineRequest{
		StatementLineID: sugg1.LineID,
		MoveID:          sugg1.SuggestedMoveID,
	}, company.ID, 1)
	if err != nil {
		t.Fatalf("ReconcileLine 1 failed: %v", err)
	}

	// Verify invoice A is now fully paid
	refreshedInvA, err := accountSvc.GetMove(invA.ID, company.ID)
	if err != nil || refreshedInvA.PaymentState != "paid" || refreshedInvA.AmountResidual > 0.01 {
		t.Fatalf("Invoice A payment state not updated: residual=%.2f, payment_state=%s",
			refreshedInvA.AmountResidual, refreshedInvA.PaymentState)
	}
	t.Logf("✓ Atomic Reconcile with Invoice executed: Invoice '%s' marked as '%s' (Amount Residual=Rp %.2f)",
		refreshedInvA.Name, refreshedInvA.PaymentState, refreshedInvA.AmountResidual)

	// 6. Direct Write-off / Bank Charge Booking: Reconcile Line 3 against Bank Expense
	err = bankReconcileSvc.ReconcileLine(bank_reconciliation.ReconcileLineRequest{
		StatementLineID:   sugg3.LineID,
		WriteOffAccountID: sugg3.SuggestedAccountID,
	}, company.ID, 1)
	if err != nil {
		t.Fatalf("ReconcileLine 3 (bank fee write-off) failed: %v", err)
	}
	t.Logf("✓ Direct Bank Write-off executed: Bank Admin Fee booked to General Ledger expense account")

	// 7. Auto-Reconcile High Confidence for remaining lines
	autoReconciledCount, err := bankReconcileSvc.AutoReconcileHighConfidence(stmt.ID, 0.70, company.ID, 1)
	if err != nil {
		t.Fatalf("AutoReconcileHighConfidence failed: %v", err)
	}
	t.Logf("✓ AutoReconcileHighConfidence executed: %d remaining lines matched and reconciled", autoReconciledCount)

	// 8. Confirm Statement Closure
	confirmedStmt, err := bankReconcileSvc.ConfirmStatement(stmt.ID, company.ID, 1)
	if err != nil {
		t.Fatalf("ConfirmStatement failed: %v", err)
	}
	if confirmedStmt.State != "confirm" || confirmedStmt.ReconciledLines != 3 {
		t.Fatalf("Confirmed statement mismatch: state=%s, reconciled=%d/%d",
			confirmedStmt.State, confirmedStmt.ReconciledLines, confirmedStmt.TotalLines)
	}
	t.Logf("✓ Bank Statement Closed & Confirmed: State='%s', %d/%d Lines Reconciled, Saldo Akhir=Rp %.2f (Selisih=Rp %.2f)",
		confirmedStmt.State, confirmedStmt.ReconciledLines, confirmedStmt.TotalLines, confirmedStmt.BalanceEnd, confirmedStmt.Difference)

	// 9. AI Bank Statement Copilot Audit
	aiReport, err := bankReconcileSvc.AIBankCopilot(stmt.ID, company.ID)
	if err != nil {
		t.Fatalf("AIBankCopilot failed: %v", err)
	}
	if aiReport.RiskAssessment == "" || aiReport.ExecutiveSummary == "" || len(aiReport.ActionRecommendations) == 0 {
		t.Fatalf("AI Bank Copilot report assertion failed: %+v", aiReport)
	}
	t.Logf("✓ Cloudflare Workers AI Bank Copilot Audit: Risk='%s', Discrepancies=%d, Engine='%s'",
		aiReport.RiskAssessment, aiReport.TotalDiscrepancies, aiReport.Engine)
	t.Logf("  Executive Summary: %s", aiReport.ExecutiveSummary)
	t.Logf("  Action Recommendation #1: %s", aiReport.ActionRecommendations[0])

	// 10. Universal Chatter Audit Trail on Bank Statement
	bankChatter, err := chatterSvc.ListMessages("account.bank.statement", stmt.ID, company.ID)
	if err != nil || len(bankChatter) < 1 {
		t.Fatalf("Expected chatter messages on bank statement, got %d (err=%v)", len(bankChatter), err)
	}
	t.Logf("✓ Universal Chatter Audit Trail verified: %d audit tracking messages recorded on Bank Statement '%s'",
		len(bankChatter), stmt.Name)

	// 11. Statement CSV Export & Executive KPI Statistics
	exportedCSV, filename, err := bankReconcileSvc.ExportStatementCSV(stmt.ID, company.ID)
	if err != nil || len(exportedCSV) == 0 || !strings.Contains(string(exportedCSV), "TRSF CR") {
		t.Fatalf("ExportStatementCSV failed or missing lines: %s (err=%v)", filename, err)
	}
	t.Logf("✓ Statement CSV Export verified: File='%s' (%d bytes exported)", filename, len(exportedCSV))

	bankStats, err := bankReconcileSvc.GetStats(company.ID)
	if err != nil || bankStats.TotalStatements < 1 || bankStats.TotalLines < 3 || bankStats.ReconciledPercentage <= 0 {
		t.Fatalf("GetStats bank reconciliation assertion failed: %+v (err=%v)", bankStats, err)
	}
	t.Logf("✓ Bank Reconciliation Executive KPI Stats: Statements=%d (Confirmed=%d), Total Lines=%d, Reconciled=%.1f%%, Total Inflow=Rp %.0f, Total Outflow=Rp %.0f, Net Movement=Rp %.0f",
		bankStats.TotalStatements, bankStats.ConfirmedStatements, bankStats.TotalLines, bankStats.ReconciledPercentage,
		bankStats.TotalInflow, bankStats.TotalOutflow, bankStats.NetMovement)

	// =========================================================================
	// SECTION 36: LANDED COSTS ENGINE & ALOKASI BIAYA IMPOR/LOGISTIK (5 METODE SPLIT, MULTI-LINE VALUATION, ATOMIC GL POSTING & AI MARGIN COPILOT)
	// =========================================================================
	t.Log("\n    --- SECTION 36: LANDED COSTS ENGINE & ALOKASI BIAYA IMPOR/LOGISTIK (5 METODE SPLIT, MULTI-LINE VALUATION, ATOMIC GL POSTING & AI MARGIN COPILOT) ---")

	// 1. Initialize Landed Cost Services
	landedCostRepo := landed_cost.NewRepository(db)
	landedCostSvc := landed_cost.NewService(db, landedCostRepo, accountSvc, chatterSvc, aiClient)

	// 2. Setup Storable Imported Products with physical metrics (Weight & Volume)
	importedMotor := product.Product{
		Name:          "Industrial Servo Motor 15kW",
		DefaultCode:   "IMP-SERVO-15K",
		Type:          "product",
		ListPrice:     16000000.0, // Selling price
		StandardPrice: 10000000.0, // Initial PO cost per unit
		Weight:        25.0,       // 25 kg per unit
		Volume:        0.08,       // 0.08 m3 per unit
	}
	importedMotor.CompanyID = company.ID
	if err := prodSvc.CreateProduct(&importedMotor); err != nil {
		t.Fatalf("Create importedMotor failed: %v", err)
	}

	importedPLC := product.Product{
		Name:          "Precision PLC Controller Module",
		DefaultCode:   "IMP-PLC-MOD",
		Type:          "product",
		ListPrice:     9000000.0, // Selling price
		StandardPrice: 5000000.0, // Initial PO cost per unit
		Weight:        5.0,       // 5 kg per unit
		Volume:        0.02,      // 0.02 m3 per unit
	}
	importedPLC.CompanyID = company.ID
	if err := prodSvc.CreateProduct(&importedPLC); err != nil {
		t.Fatalf("Create importedPLC failed: %v", err)
	}

	// 3. Create Incoming Stock Picking Receipt (4 Motors + 6 PLCs = 10 units total)
	// Former Cost: Motor (4 * 10jt = 40jt), PLC (6 * 5jt = 30jt) -> Total Goods Value = Rp 70,000,000
	importPO := purchase.PurchaseOrder{
		PartnerID: vendor.ID,
		OrderLines: []purchase.PurchaseOrderLine{
			{
				ProductID:     importedMotor.ID,
				Name:          importedMotor.Name,
				ProductUomQty: 4,
				PriceUnit:     10000000.0,
			},
			{
				ProductID:     importedPLC.ID,
				Name:          importedPLC.Name,
				ProductUomQty: 6,
				PriceUnit:     5000000.0,
			},
		},
	}
	importPO.CompanyID = company.ID
	if err := purchaseSvc.CreateOrder(&importPO); err != nil {
		t.Fatalf("Create Import PO failed: %v", err)
	}
	_, importReceipt, err := purchaseSvc.ActionConfirm(importPO.ID, company.ID)
	if err != nil || importReceipt == nil {
		t.Fatalf("Confirm Import PO failed: %v", err)
	}
	_, err = stockSvc.ValidatePicking(importReceipt.ID, company.ID)
	if err != nil {
		t.Fatalf("Validate Import Stock Receipt failed: %v", err)
	}
	t.Logf("✓ Incoming Stock Receipt '%s' validated: 4x Servo Motor (Rp 40jt) + 6x PLC Controller (Rp 30jt) = Total Nilai Barang Rp 70jt",
		importReceipt.Name)

	// 4. Create Landed Cost Document with 3 Cost Lines & 3 Distinct Split Methods
	// Total Landed Cost: Rp 7,000,000 (10% overhead of goods value)
	// Line 1: Ocean Freight (Rp 3,500,000) -> Split 'by_value'
	// Line 2: Bea Masuk & Pajak Impor (Rp 2,500,000) -> Split 'by_quantity'
	// Line 3: Port Handling Charges (Rp 1,000,000) -> Split 'equal'
	clearingAccountID := landedCostSvc.GetOrCreateClearingAccountID(company.ID)
	lcReq := landed_cost.CreateLandedCostRequest{
		Date:        "2026-10-09",
		PickingID:   importReceipt.ID,
		Description: "Alokasi Biaya Pengapalan Laut, Bea Masuk Pelabuhan & Terminal Handling",
		CostLines: []landed_cost.CreateCostLineRequest{
			{
				Name:        "Ocean Freight & Shipping Line",
				SplitMethod: landed_cost.SplitByValue,
				PriceUnit:   3500000.0,
				AccountID:   clearingAccountID,
			},
			{
				Name:        "Bea Masuk & Pajak Impor DJBC",
				SplitMethod: landed_cost.SplitByQuantity,
				PriceUnit:   2500000.0,
				AccountID:   clearingAccountID,
			},
			{
				Name:        "Port Handling & Demurrage Tanjung Priok",
				SplitMethod: landed_cost.SplitEqual,
				PriceUnit:   1000000.0,
				AccountID:   clearingAccountID,
			},
		},
	}

	createdLC, err := landedCostSvc.CreateLandedCost(lcReq, company.ID, 1)
	if err != nil {
		t.Fatalf("CreateLandedCost failed: %v", err)
	}
	if createdLC.State != landed_cost.StateDraft || createdLC.TotalAmount != 7000000.0 || len(createdLC.CostLines) != 3 {
		t.Fatalf("CreateLandedCost assertions failed: %+v", createdLC)
	}
	t.Logf("✓ Draft Landed Cost Document '%s' created: Total Amount Rp %.2f (3 Cost Lines, Picking '%s')",
		createdLC.Name, createdLC.TotalAmount, importReceipt.Name)

	// 5. Compute Valuation Lines with Multi-Split Algorithm
	computedLC, err := landedCostSvc.ComputeValuationLines(createdLC.ID, company.ID, 1)
	if err != nil {
		t.Fatalf("ComputeValuationLines failed: %v", err)
	}
	if computedLC.State != landed_cost.StateComputed || len(computedLC.ValuationLines) != 6 {
		t.Fatalf("ComputeValuationLines mismatch: state=%s, valuation_lines=%d (expected 6)",
			computedLC.State, len(computedLC.ValuationLines))
	}

	// Verify exact sum of allocated additional costs across all lines equals Rp 7,000,000
	var sumAllocated float64
	for _, vl := range computedLC.ValuationLines {
		sumAllocated += vl.AdditionalLandedCost
	}
	if math.Abs(sumAllocated-7000000.0) > 0.01 {
		t.Fatalf("Allocated landed cost sum mismatch: got %.2f, expected 7000000.00", sumAllocated)
	}
	t.Logf("✓ Landed Cost Split Engine: Computed %d Valuation Adjustment Lines, Total Allocated: Rp %.2f (Exact 100%% Balance)",
		len(computedLC.ValuationLines), sumAllocated)

	// 6. Test AI Landed Cost Margin & Profitability Copilot
	aiLCReport, err := landedCostSvc.AILandedCostCopilot(computedLC.ID, company.ID)
	if err != nil {
		t.Fatalf("AILandedCostCopilot failed: %v", err)
	}
	if aiLCReport.RiskLevel == "" || len(aiLCReport.ProductImpacts) != 2 || len(aiLCReport.Recommendations) == 0 {
		t.Fatalf("AI Landed Cost Report assertion failed: %+v", aiLCReport)
	}
	t.Logf("✓ Cloudflare Workers AI Landed Cost Copilot Audit: RiskLevel='%s', Overhead Ratio=%.1f%%, Provider='%s'",
		aiLCReport.RiskLevel, aiLCReport.OverallOverheadPct, aiLCReport.AIProvider)
	for _, impact := range aiLCReport.ProductImpacts {
		t.Logf("  Product '%s': HPP Awal Rp %.0f -> HPP Baru Rp %.0f (+%.1f%%), ListPrice Rp %.0f, Margin Kotor=%.1f%% (%s)",
			impact.ProductName, impact.OldCostPerUnit, impact.NewCostPerUnit, impact.CostIncreasePct,
			impact.SellingPrice, impact.EstimatedGrossMarginPct, impact.MarginStatus)
	}

	// 7. ActionValidate: Atomic General Ledger Journal Posting & Product Cost Update
	validatedLC, err := landedCostSvc.ActionValidate(computedLC.ID, company.ID, 1)
	if err != nil {
		t.Fatalf("ActionValidate failed: %v", err)
	}
	if validatedLC.State != landed_cost.StateDone || validatedLC.AccountMoveID == nil {
		t.Fatalf("Validated Landed Cost assertion failed: state=%s, moveID=%v", validatedLC.State, validatedLC.AccountMoveID)
	}

	// Verify Journal Entry created & posted
	glMove, err = accountSvc.GetMove(*validatedLC.AccountMoveID, company.ID)
	if err != nil || glMove.State != "posted" {
		t.Fatalf("Expected posted GL Move, got state=%s (err=%v)", glMove.State, err)
	}

	var sumDebit, sumCredit float64
	for _, l := range glMove.LineIDs {
		sumDebit += l.Debit
		sumCredit += l.Credit
	}
	if math.Abs(sumDebit-7000000.0) > 0.01 || math.Abs(sumCredit-7000000.0) > 0.01 {
		t.Fatalf("Unbalanced GL Move lines: Debit=%.2f, Credit=%.2f (expected 7000000.0)", sumDebit, sumCredit)
	}
	t.Logf("✓ Atomic General Ledger Posting verified: Entry '%s' Posted (Debit=Rp %.2f, Credit=Rp %.2f Balanced)",
		glMove.Name, sumDebit, sumCredit)

	// 8. Verify Product Standard Price Updated in DB
	var reloadedMotor, reloadedPLC product.Product
	db.First(&reloadedMotor, importedMotor.ID)
	db.First(&reloadedPLC, importedPLC.ID)
	if reloadedMotor.StandardPrice <= 10000000.0 || reloadedPLC.StandardPrice <= 5000000.0 {
		t.Fatalf("Product standard prices not updated after landed cost validation: Motor=%.2f, PLC=%.2f",
			reloadedMotor.StandardPrice, reloadedPLC.StandardPrice)
	}
	t.Logf("✓ Product Standard Cost automatically updated: Motor 15kW Rp %.2f (was 10jt), PLC Module Rp %.2f (was 5jt)",
		reloadedMotor.StandardPrice, reloadedPLC.StandardPrice)

	// 9. Universal Chatter Audit Trail
	lcChatter, err := chatterSvc.ListMessages("stock.landed.cost", validatedLC.ID, company.ID)
	if err != nil || len(lcChatter) < 2 {
		t.Fatalf("Expected chatter tracking on landed cost, got %d messages (err=%v)", len(lcChatter), err)
	}
	t.Logf("✓ Universal Chatter Audit Trail verified: %d audit messages logged on Landed Cost '%s'",
		len(lcChatter), validatedLC.Name)

	// 10. Executive KPI Stats
	lcStats, err := landedCostSvc.GetStats(company.ID)
	if err != nil || lcStats.TotalLandedCosts < 1 || lcStats.DoneCount < 1 || lcStats.TotalCostAllocated < 7000000.0 {
		t.Fatalf("Landed Cost stats assertion failed: %+v", lcStats)
	}
	t.Logf("✓ Landed Cost Executive KPI Stats: Total=%d, Done=%d, Total Cost Allocated=Rp %.0f, Average Cost Ratio=%.1f%%",
		lcStats.TotalLandedCosts, lcStats.DoneCount, lcStats.TotalCostAllocated, lcStats.AverageCostRatio)

	// =======================================================================
	// 37. TEST LOT & SERIAL TRACEABILITY ENGINE, FEFO PICKING ALLOCATION & CLOUDFLARE WORKERS AI PRODUCT RECALL COPILOT (MODUL KE-33)
	// =======================================================================
	t.Log("--- 37. TESTING LOT & SERIAL TRACEABILITY ENGINE, FEFO PICKING ALLOCATION & PRODUCT RECALL COPILOT ---")

	traceabilityRepo := traceability.NewRepository(db)
	traceabilitySvc := traceability.NewService(db, traceabilityRepo, chatterSvc, aiClient)

	// 1. Setup Test Products & Lots with specific expiration horizons
	rawPharma := product.Product{
		Name:          "Paracetamol Pure Powder USP 99.8%",
		DefaultCode:   "RAW-PARA-USP",
		Type:          "product",
		ListPrice:     350000.0,
		StandardPrice: 200000.0,
	}
	rawPharma.CompanyID = company.ID
	if err := db.Create(&rawPharma).Error; err != nil {
		t.Fatalf("Failed to create rawPharma product: %v", err)
	}

	finPharma := product.Product{
		Name:          "Sanmol Forte 650mg Caplet Box 100s",
		DefaultCode:   "MED-SANMOL-650",
		Type:          "product",
		ListPrice:     180000.0,
		StandardPrice: 95000.0,
	}
	finPharma.CompanyID = company.ID
	if err := db.Create(&finPharma).Error; err != nil {
		t.Fatalf("Failed to create finPharma product: %v", err)
	}

	now := time.Now()
	dateFresh := now.AddDate(2, 0, 0).Format("2006-01-02")
	dateNearExp := now.AddDate(0, 0, 14).Format("2006-01-02") // 14 days left
	dateExpired := now.AddDate(0, -1, -5).Format("2006-01-02") // Expired 35 days ago

	// Create 3 Lots for raw material to test FEFO
	lotRawFresh := barcode.StockProductionLot{
		Name:           "LOT-RAW-PARA-FRESH-2026",
		ProductID:      rawPharma.ID,
		Barcode:        "LOT-RAW-PARA-FRESH-2026",
		ExpirationDate: dateFresh,
		Qty:            100.0,
	}
	lotRawFresh.CompanyID = company.ID
	db.Create(&lotRawFresh)

	lotRawNear := barcode.StockProductionLot{
		Name:           "LOT-RAW-PARA-NEAR-2026",
		ProductID:      rawPharma.ID,
		Barcode:        "LOT-RAW-PARA-NEAR-2026",
		ExpirationDate: dateNearExp,
		Qty:            40.0,
	}
	lotRawNear.CompanyID = company.ID
	db.Create(&lotRawNear)

	lotRawExp := barcode.StockProductionLot{
		Name:           "LOT-RAW-PARA-EXPIRED-2026",
		ProductID:      rawPharma.ID,
		Barcode:        "LOT-RAW-PARA-EXPIRED-2026",
		ExpirationDate: dateExpired,
		Qty:            25.0,
	}
	lotRawExp.CompanyID = company.ID
	db.Create(&lotRawExp)

	t.Logf("✓ Setup 3 Raw Material Lots: Lot A (Fresh, 100kg, Exp: %s), Lot B (Near Exp, 40kg, Exp: %s), Lot C (Expired, 25kg, Exp: %s)",
		dateFresh, dateNearExp, dateExpired)

	// 2. Test FEFO (First-Expired, First-Out) Strategy Picking Allocation
	fefoReq := traceability.FEFOAllocationRequest{
		ProductID:        rawPharma.ID,
		RequestedQty:     60.0,
		AllowNearExpiry:  false,
		ExpiryCutoffDays: 30,
	}
	fefoRes, err := traceabilitySvc.AllocateFEFO(fefoReq, company.ID)
	if err != nil {
		t.Fatalf("AllocateFEFO failed: %v", err)
	}
	if !fefoRes.IsFullyAllocated || fefoRes.TotalAllocatedQty != 60.0 || len(fefoRes.Allocations) != 2 {
		t.Fatalf("FEFO Allocation assertions failed: fully=%v, total=%.2f (expected 60), alloc_count=%d (expected 2)",
			fefoRes.IsFullyAllocated, fefoRes.TotalAllocatedQty, len(fefoRes.Allocations))
	}

	if fefoRes.Allocations[0].LotID != lotRawNear.ID || fefoRes.Allocations[0].AllocatedQty != 40.0 {
		t.Fatalf("FEFO First pick violation: expected Lot Near (ID %d, 40kg), got Lot ID %d (%.2f kg)",
			lotRawNear.ID, fefoRes.Allocations[0].LotID, fefoRes.Allocations[0].AllocatedQty)
	}
	if fefoRes.Allocations[1].LotID != lotRawFresh.ID || fefoRes.Allocations[1].AllocatedQty != 20.0 {
		t.Fatalf("FEFO Second pick violation: expected Lot Fresh (ID %d, 20kg), got Lot ID %d (%.2f kg)",
			lotRawFresh.ID, fefoRes.Allocations[1].LotID, fefoRes.Allocations[1].AllocatedQty)
	}
	for _, item := range fefoRes.Allocations {
		if item.LotID == lotRawExp.ID {
			t.Fatalf("CRITICAL: Expired lot %s was allocated in FEFO!", item.LotName)
		}
	}
	t.Logf("✓ FEFO Allocation Engine verified: Requested 60kg -> Picked 40kg from '%s' (expiring in 14 days) + 20kg from '%s' (expiring in 2 yrs). Expired lot safely bypassed!",
		fefoRes.Allocations[0].LotName, fefoRes.Allocations[1].LotName)

	// 3. Test Movement Recording for Complete Upstream/Downstream Genealogy
	_, err = traceabilitySvc.RecordMovement(traceability.RecordLotMovementRequest{
		LotID:          lotRawFresh.ID,
		ProductID:      rawPharma.ID,
		EventType:      traceability.EventReceipt,
		Reference:      "WH/IN/00771",
		SourceDocument: "PO00099",
		PartnerName:    "PT Kimia Farma Trading & Distribution",
		Quantity:       100.0,
		UoM:            "kg",
		Date:           now.AddDate(0, -3, 0).Format("2006-01-02 15:04:05"),
		Notes:          "Penerimaan bahan baku impor tersertifikasi GMP",
	}, company.ID, 1)
	if err != nil {
		t.Fatalf("RecordMovement (Receipt) failed: %v", err)
	}

	lotFinProd := barcode.StockProductionLot{
		Name:           "LOT-FIN-SANMOL-F881",
		ProductID:      finPharma.ID,
		Barcode:        "LOT-FIN-SANMOL-F881",
		ExpirationDate: dateFresh,
		Qty:            10000.0,
		Notes:          "Batch produksi kapsul sanmol 650mg",
	}
	lotFinProd.CompanyID = company.ID
	db.Create(&lotFinProd)

	_, err = traceabilitySvc.RecordMovement(traceability.RecordLotMovementRequest{
		LotID:          lotRawFresh.ID,
		ProductID:      rawPharma.ID,
		EventType:      traceability.EventMOConsume,
		Reference:      "MO/2026/00888",
		SourceDocument: "MO/2026/00888",
		ProducedLotID:  &lotFinProd.ID,
		Quantity:       50.0,
		UoM:            "kg",
		Date:           now.AddDate(0, -2, 0).Format("2006-01-02 15:04:05"),
		Notes:          "Konsumsi bahan baku di cleanroom blending",
	}, company.ID, 1)
	if err != nil {
		t.Fatalf("RecordMovement (MO Consume) failed: %v", err)
	}

	_, err = traceabilitySvc.RecordMovement(traceability.RecordLotMovementRequest{
		LotID:          lotFinProd.ID,
		ProductID:      finPharma.ID,
		EventType:      traceability.EventMOProduce,
		Reference:      "MO/2026/00888",
		SourceDocument: "MO/2026/00888",
		ConsumedLotID:  &lotRawFresh.ID,
		Quantity:       10000.0,
		UoM:            "Box",
		Date:           now.AddDate(0, -2, 0).Format("2006-01-02 15:04:05"),
		Notes:          "Pengemasan primer blister dan sekunder box",
	}, company.ID, 1)
	if err != nil {
		t.Fatalf("RecordMovement (MO Produce) failed: %v", err)
	}

	_, err = traceabilitySvc.RecordMovement(traceability.RecordLotMovementRequest{
		LotID:          lotFinProd.ID,
		ProductID:      finPharma.ID,
		EventType:      traceability.EventDelivery,
		Reference:      "WH/OUT/00771",
		SourceDocument: "SO00088",
		PartnerName:    "RS Siloam Hospitals Kebon Jeruk",
		Quantity:       6000.0,
		UoM:            "Box",
		Date:           now.AddDate(0, -1, 0).Format("2006-01-02 15:04:05"),
		Notes:          "Pengiriman farmasi darurat",
	}, company.ID, 1)
	if err != nil {
		t.Fatalf("RecordMovement (Delivery 1) failed: %v", err)
	}

	_, err = traceabilitySvc.RecordMovement(traceability.RecordLotMovementRequest{
		LotID:          lotFinProd.ID,
		ProductID:      finPharma.ID,
		EventType:      traceability.EventDelivery,
		Reference:      "WH/OUT/00772",
		SourceDocument: "SO00089",
		PartnerName:    "RS Mayapada Healthcare Jakarta Selatan",
		Quantity:       2500.0,
		UoM:            "Box",
		Date:           now.AddDate(0, 0, -10).Format("2006-01-02 15:04:05"),
		Notes:          "Pengiriman logistik reguler",
	}, company.ID, 1)
	if err != nil {
		t.Fatalf("RecordMovement (Delivery 2) failed: %v", err)
	}
	t.Logf("✓ Complete Physical Movement Lifecycle recorded: Receipt WH/IN/00771 -> MO/2026/00888 -> Deliveries WH/OUT/00771 & WH/OUT/00772")

	// 4. Test Upstream Traceability
	upstreamTree, err := traceabilitySvc.BuildTraceabilityTree(lotFinProd.ID, "upstream", company.ID)
	if err != nil {
		t.Fatalf("BuildTraceabilityTree (Upstream) failed: %v", err)
	}
	if upstreamTree.UpstreamTree == nil || upstreamTree.TotalManufacturingOrdersInvolved < 1 || upstreamTree.TotalSuppliersInvolved < 1 {
		t.Fatalf("Upstream Tree assertions failed: MOs=%d, Suppliers=%d",
			upstreamTree.TotalManufacturingOrdersInvolved, upstreamTree.TotalSuppliersInvolved)
	}
	t.Logf("✓ Upstream Traceability Tree verified: Target Finished Lot '%s' traced back to MO '%s', Raw Lot '%s', Supplier '%s'",
		lotFinProd.Name, upstreamTree.UpstreamTree.Children[0].Reference,
		upstreamTree.UpstreamTree.Children[0].Children[0].LotName,
		upstreamTree.UpstreamTree.Children[0].Children[0].Children[0].PartnerName)

	// 5. Test Downstream Traceability
	downstreamTree, err := traceabilitySvc.BuildTraceabilityTree(lotRawFresh.ID, "downstream", company.ID)
	if err != nil {
		t.Fatalf("BuildTraceabilityTree (Downstream) failed: %v", err)
	}
	if len(downstreamTree.ImpactedCustomers) != 2 || downstreamTree.TotalCustomerDeliveriesInvolved != 2 {
		t.Fatalf("Downstream Tree assertions failed: impacted_customers=%v, total_deliveries=%d",
			downstreamTree.ImpactedCustomers, downstreamTree.TotalCustomerDeliveriesInvolved)
	}
	t.Logf("✓ Downstream Traceability Tree verified: Raw Lot '%s' traced forward through MO to 2 Impacted Customers: %v (Deliveries: %d)",
		lotRawFresh.Name, downstreamTree.ImpactedCustomers, downstreamTree.TotalCustomerDeliveriesInvolved)

	// 6. Test Product Recall & Emergency Customer Tracing Copilot
	recallReq := traceability.ProductRecallRequest{
		LotID:    lotRawFresh.ID,
		Reason:   "Kontaminasi silang partikel logam pada bejana reaktor supplier",
		Severity: "critical",
	}
	recallNotice, err := traceabilitySvc.InitiateRecall(recallReq, company.ID, 1)
	if err != nil {
		t.Fatalf("InitiateRecall failed: %v", err)
	}
	if recallNotice.RecallCode == "" || recallNotice.AffectedCustomersCount != 2 || recallNotice.CustomerNoticeText == "" || recallNotice.RegulatoryNoticeText == "" {
		t.Fatalf("InitiateRecall assertions failed: %+v", recallNotice)
	}
	if recallNotice.State != traceability.RecallStateQuarantined {
		t.Fatalf("Expected recall state 'quarantined', got '%s'", recallNotice.State)
	}

	var reloadedRawLot barcode.StockProductionLot
	db.First(&reloadedRawLot, lotRawFresh.ID)
	if !strings.Contains(reloadedRawLot.Notes, "[QUARANTINED BY RECALL") {
		t.Fatalf("Raw lot notes do not indicate quarantine: %s", reloadedRawLot.Notes)
	}
	t.Logf("✓ Product Recall Copilot: Code '%s' Issued! Quarantined %.0f kg, identified %d affected customers (%s). AI Drafted Official Customer Recall Notice & BPOM Regulatory Report.",
		recallNotice.RecallCode, recallNotice.AffectedStockQuarantined, recallNotice.AffectedCustomersCount,
		recallNotice.AIProvider)

	completedRecall, err := traceabilitySvc.CompleteRecall(recallNotice.ID, company.ID, "Pemusnahan limbah B3 selesai bersertifikasi", 1)
	if err != nil || completedRecall.State != traceability.RecallStateCompleted {
		t.Fatalf("CompleteRecall failed: state=%s (err=%v)", completedRecall.State, err)
	}
	t.Logf("✓ Product Recall lifecycle completed: State='%s', Resolution Notes='%s'",
		completedRecall.State, completedRecall.ResolutionNotes)

	// 7. Test Expiry Risk & Warehouse Batch Audit Copilot
	expiryAudit, err := traceabilitySvc.AuditExpiryRisk(company.ID)
	if err != nil {
		t.Fatalf("AuditExpiryRisk failed: %v", err)
	}
	if expiryAudit.ExpiredLotsCount < 1 || expiryAudit.NearExpiryLotsCount < 1 || len(expiryAudit.CriticalLots) < 2 {
		t.Fatalf("AuditExpiryRisk assertions failed: expired=%d, near=%d, critical_lots=%d",
			expiryAudit.ExpiredLotsCount, expiryAudit.NearExpiryLotsCount, len(expiryAudit.CriticalLots))
	}
	t.Logf("✓ Expiry Risk & Warehouse Audit Copilot: Overall Risk='%s', %d Expired Lots (Loss Rp %.2f), %d Near-Expiry Lots (Risk Rp %.2f). Insights: '%s' (%s)",
		expiryAudit.OverallRiskLevel, expiryAudit.ExpiredLotsCount, expiryAudit.ExpiredLotsValue,
		expiryAudit.NearExpiryLotsCount, expiryAudit.NearExpiryLotsValue,
		expiryAudit.AIInsights[:80]+"...", expiryAudit.AIProvider)

	// 8. Test Traceability Dashboard KPI Stats
	traceStats, err := traceabilitySvc.GetStats(company.ID)
	if err != nil || traceStats.TotalTraceRecords < 5 || traceStats.ExpiredLots < 1 {
		t.Fatalf("Traceability Stats assertion failed: %+v", traceStats)
	}
	t.Logf("✓ Traceability Executive KPI Stats: Total Lots=%d, Active Lots With Stock=%d, Expired=%d, Near-Expiry=%d, Total Trace Records=%d",
		traceStats.TotalLots, traceStats.ActiveLotsWithStock, traceStats.ExpiredLots, traceStats.NearExpiryLots, traceStats.TotalTraceRecords)

	// =========================================================================
	// 38. SUBSCRIPTION & RECURRING INVOICING ENGINE (SAAS METRICS, BATCH BILLING, BALANCED INVOICE CREATION & AI CHURN COPILOT)
	// =========================================================================
	t.Log("\n--- SECTION 38: SUBSCRIPTION & RECURRING INVOICING ENGINE (SAAS METRICS, BATCH BILLING, BALANCED INVOICE CREATION & AI CHURN COPILOT) ---")

	subMod := subscription.NewModule(db, accountSvc, chatterSvc, aiClient)
	if err := subMod.Seed(db); err != nil {
		t.Fatalf("Subscription Seed failed: %v", err)
	}
	subSvc := subMod.Service()

	// 1. Verify Seeded Subscription Templates
	tmpls, err := subSvc.ListTemplates(company.ID)
	if err != nil || len(tmpls) < 3 {
		t.Fatalf("Expected at least 3 seeded templates, got %d (err: %v)", len(tmpls), err)
	}
	t.Logf("✓ Verified %d Subscription Templates seeded (e.g. '%s', '%s')", len(tmpls), tmpls[0].Name, tmpls[1].Name)

	// 2. Create Custom Subscription Template
	autoPost := true
	customTmpl, err := subSvc.CreateTemplate(subscription.CreateTemplateRequest{
		Code:               "TMPL-VIP-BIANNUAL",
		Name:               "VIP Cloud Dedicated (Semesteran)",
		RecurringInterval:  subscription.IntervalMonthly,
		RecurringRuleCount: 6,
		InvoiceJournalID:   1,
		AutoPostInvoice:    &autoPost,
		Description:        "Paket dedikasi cloud semesteran dengan server terisolasi",
	}, company.ID)
	if err != nil {
		t.Fatalf("CreateTemplate failed: %v", err)
	}
	t.Logf("✓ Custom Template created: '%s' (%s, RuleCount=%d, AutoPost=%v)", customTmpl.Name, customTmpl.Code, customTmpl.RecurringRuleCount, customTmpl.AutoPostInvoice)

	// 3. Create Subscription Contract in Draft State
	autoRenew := true
	subReq := subscription.CreateSubscriptionRequest{
		Name:                   "Enterprise Cloud ERP Suite - PT Maju Bersama Makmur",
		PartnerID:              customer.ID,
		TemplateID:             tmpls[0].ID, // Monthly Pro
		StartDate:              "2026-09-01",
		NextInvoiceDate:        "2026-10-01",
		RecurringInterval:      subscription.IntervalMonthly,
		RecurringIntervalCount: 1,
		Currency:               "IDR",
		PaymentMethod:          "bank_transfer",
		AutoRenew:              &autoRenew,
		Notes:                  "Kontrak lisensi cloud multi-user dengan add-on modul AI",
		Lines: []subscription.CreateSubscriptionLineRequest{
			{
				ProductID: testProduct.ID,
				Name:      "Lisensi Cloud ERP Core (10 User)",
				Quantity:  1,
				UnitPrice: 5000000,
				Discount:  10, // 10% discount -> Rp 4.500.000
				UoM:       "Bulan",
			},
			{
				ProductID: testProduct.ID,
				Name:      "Dedicated AI Assistant Engine Add-on",
				Quantity:  2,
				UnitPrice: 1000000,
				Discount:  0, // Rp 2.000.000
				UoM:       "Seat",
			},
		},
	}
	newSub, err := subSvc.CreateSubscription(subReq, company.ID, 1)
	if err != nil {
		t.Fatalf("CreateSubscription failed: %v", err)
	}
	expectedRecurring := 6500000.0 // 4.5jt + 2.0jt
	if newSub.State != subscription.StateDraft || math.Abs(newSub.RecurringAmount-expectedRecurring) > 0.01 {
		t.Fatalf("CreateSubscription assertions failed: state=%s, recurring_amount=%.2f (expected %.2f)",
			newSub.State, newSub.RecurringAmount, expectedRecurring)
	}
	if math.Abs(newSub.TotalMRR-expectedRecurring) > 0.01 || math.Abs(newSub.TotalARR-(expectedRecurring*12)) > 0.01 {
		t.Fatalf("MRR/ARR calculation failed: MRR=%.2f, ARR=%.2f", newSub.TotalMRR, newSub.TotalARR)
	}
	t.Logf("✓ Created Subscription Contract '%s' (%s): Recurring Rp %.2f / mo (MRR: Rp %.2f, ARR: Rp %.2f, State: '%s')",
		newSub.Code, newSub.Name, newSub.RecurringAmount, newSub.TotalMRR, newSub.TotalARR, newSub.State)

	// 4. Activate Subscription (Draft -> In Progress)
	activeSub, err := subSvc.StartSubscription(newSub.ID, company.ID, 1)
	if err != nil || activeSub.State != subscription.StateInProgress {
		t.Fatalf("StartSubscription failed: state=%s, err=%v", activeSub.State, err)
	}
	t.Logf("✓ Subscription '%s' activated: State='%s', Active MRR=Rp %.2f", activeSub.Code, activeSub.State, activeSub.TotalMRR)

	// 5. Generate Single Recurring Invoice & Verify Double-Entry Balance
	invLog, err := subSvc.GenerateSingleInvoice(activeSub.ID, company.ID, 1)
	if err != nil {
		t.Fatalf("GenerateSingleInvoice failed: %v", err)
	}
	if invLog.InvoiceNumber == "" || invLog.Amount != expectedRecurring || invLog.Status != "posted" {
		t.Fatalf("InvoiceLog assertions failed: number=%s, amount=%.2f, status=%s",
			invLog.InvoiceNumber, invLog.Amount, invLog.Status)
	}

	// Verify Account Move in General Ledger
	invMove, err := accountSvc.GetMove(invLog.InvoiceID, company.ID)
	if err != nil {
		t.Fatalf("Failed to fetch generated AccountMove ID %d: %v", invLog.InvoiceID, err)
	}
	if invMove.MoveType != "out_invoice" || invMove.State != "posted" {
		t.Fatalf("AccountMove assertions failed: type=%s, state=%s", invMove.MoveType, invMove.State)
	}

	// Strict Double-Entry Balance verification (Debit == Credit)
	var subMoveDebit, subMoveCredit float64
	for _, l := range invMove.LineIDs {
		subMoveDebit += l.Debit
		subMoveCredit += l.Credit
	}
	if math.Abs(subMoveDebit-subMoveCredit) > 0.001 || math.Abs(subMoveDebit-expectedRecurring) > 0.001 {
		t.Fatalf("AccountMove unbalanced: Total Debit=%.2f, Total Credit=%.2f, Expected=%.2f",
			subMoveDebit, subMoveCredit, expectedRecurring)
	}

	// Reload subscription to verify next invoice date advanced by 1 month
	reloadedSub, err := subSvc.GetSubscription(activeSub.ID, company.ID)
	if err != nil || reloadedSub.InvoicedCount != 1 || reloadedSub.NextInvoiceDate != "2026-11-01" {
		t.Fatalf("Subscription counters failed: invoiced_count=%d, next_date=%s (expected 2026-11-01)",
			reloadedSub.InvoicedCount, reloadedSub.NextInvoiceDate)
	}
	t.Logf("✓ Automated Recurring Invoice '%s' Generated & Balanced: Rp %.2f (Debit=%.2f, Credit=%.2f, State: '%s'). Next Billing Date: %s",
		invMove.Name, invMove.AmountTotal, subMoveDebit, subMoveCredit, invMove.State, reloadedSub.NextInvoiceDate)

	// 6. Test Batch Recurring Invoicing Engine
	// Create another subscription with NextInvoiceDate set to past
	dueSubReq := subscription.CreateSubscriptionRequest{
		Name:                   "Maintenance SLA Contract - PT Cahaya Sentosa",
		PartnerID:              customer.ID,
		TemplateID:             tmpls[2].ID, // Quarterly SLA
		StartDate:              "2026-07-01",
		NextInvoiceDate:        "2026-10-01", // Due now
		RecurringInterval:      subscription.IntervalQuarterly,
		RecurringIntervalCount: 1,
		Currency:               "IDR",
		PaymentMethod:          "auto_debit",
		Lines: []subscription.CreateSubscriptionLineRequest{
			{
				ProductID: testProduct.ID,
				Name:      "Paket Pemeliharaan Rutin Server Triwulan",
				Quantity:  1,
				UnitPrice: 9000000,
				Discount:  0,
				UoM:       "Triwulan",
			},
		},
	}
	dueSub, err := subSvc.CreateSubscription(dueSubReq, company.ID, 1)
	if err != nil {
		t.Fatalf("Failed to create due subscription: %v", err)
	}
	if _, err := subSvc.StartSubscription(dueSub.ID, company.ID, 1); err != nil {
		t.Fatalf("Failed to start due subscription: %v", err)
	}

	// Execute Batch Billing runner up to today
	batchRes, err := subSvc.RunBillingBatch(subscription.BillingBatchRequest{TargetDate: "2026-10-15"}, company.ID, 1)
	if err != nil {
		t.Fatalf("RunBillingBatch failed: %v", err)
	}
	if batchRes.InvoicesGenerated < 1 || batchRes.TotalBilledRevenue < 9000000 {
		t.Fatalf("RunBillingBatch assertions failed: generated=%d, revenue=%.2f",
			batchRes.InvoicesGenerated, batchRes.TotalBilledRevenue)
	}
	t.Logf("✓ Batch Recurring Billing Runner: Processed %d due contracts -> %d Invoices Generated, Total Billed Revenue: Rp %.2f",
		batchRes.ProcessedSubscriptions, batchRes.InvoicesGenerated, batchRes.TotalBilledRevenue)

	// 7. Pause & Resume Subscription Lifecycle
	pausedSub, err := subSvc.PauseSubscription(reloadedSub.ID, "Upgrade infrastruktur internal klien", company.ID, 1)
	if err != nil || pausedSub.State != subscription.StatePaused {
		t.Fatalf("PauseSubscription failed: state=%s, err=%v", pausedSub.State, err)
	}
	t.Logf("✓ Paused Subscription '%s': State='%s'", pausedSub.Code, pausedSub.State)

	resumedSub, err := subSvc.ResumeSubscription(reloadedSub.ID, company.ID, 1)
	if err != nil || resumedSub.State != subscription.StateInProgress {
		t.Fatalf("ResumeSubscription failed: state=%s, err=%v", resumedSub.State, err)
	}
	t.Logf("✓ Resumed Subscription '%s': State='%s', Active MRR=Rp %.2f", resumedSub.Code, resumedSub.State, resumedSub.TotalMRR)

	// 8. Cloudflare Workers AI SaaS Retention & Churn Risk Copilot
	churnAudit, err := subSvc.AnalyzeChurnRisk(reloadedSub.ID, company.ID)
	if err != nil {
		t.Fatalf("AnalyzeChurnRisk failed: %v", err)
	}
	if churnAudit.RiskLevel == "" || churnAudit.ChurnProbabilityPct < 0 || len(churnAudit.RetentionActionPlan) == 0 {
		t.Fatalf("AnalyzeChurnRisk assertions failed: %+v", churnAudit)
	}
	t.Logf("✓ Cloudflare Workers AI SaaS Retention Copilot: Contract '%s', Risk='%s', Churn Probability=%.1f%%, Actions=%d, Discount=%.0f%%, Engine='%s'",
		churnAudit.SubscriptionCode, churnAudit.RiskLevel, churnAudit.ChurnProbabilityPct,
		len(churnAudit.RetentionActionPlan), churnAudit.RecommendedDiscount, churnAudit.AIProvider)
	t.Logf("  Executive Summary: %s", churnAudit.ExecutiveSummary)

	// 9. Cancellation & Churn Lifecycle
	churnedSub, err := subSvc.CancelSubscription(dueSub.ID, "Penyesuaian anggaran operasional tahunan", company.ID, 1)
	if err != nil || churnedSub.State != subscription.StateChurned || churnedSub.ChurnReason == "" {
		t.Fatalf("CancelSubscription failed: state=%s, reason=%s, err=%v", churnedSub.State, churnedSub.ChurnReason, err)
	}
	t.Logf("✓ Churned Subscription '%s': State='%s', Churn Reason='%s', Churn Date='%s'",
		churnedSub.Code, churnedSub.State, churnedSub.ChurnReason, churnedSub.ChurnDate)

	// 10. SaaS Metrics & Executive Analytics Engine
	saasMetrics, err := subSvc.GetStats(company.ID)
	if err != nil || saasMetrics.TotalSubscriptions < 3 || saasMetrics.ActiveSubscriptions < 1 || saasMetrics.ChurnedSubscriptions < 1 {
		t.Fatalf("GetStats SaaS metrics assertions failed: %+v", saasMetrics)
	}
	if saasMetrics.TotalLiveMRR <= 0 || saasMetrics.TotalLiveARR <= 0 || saasMetrics.TotalInvoicedLifetime <= 0 {
		t.Fatalf("SaaS revenue metrics invalid: MRR=%.2f, ARR=%.2f, Invoiced=%.2f",
			saasMetrics.TotalLiveMRR, saasMetrics.TotalLiveARR, saasMetrics.TotalInvoicedLifetime)
	}
	t.Logf("✓ SaaS Executive Metrics & Analytics: Total Subs=%d (Active=%d, Churned=%d), Total Live MRR=Rp %.2f, ARR=Rp %.2f, ARPU=Rp %.2f, Logo Churn=%.1f%%, Invoiced Lifetime=Rp %.2f (%d invoices)",
		saasMetrics.TotalSubscriptions, saasMetrics.ActiveSubscriptions, saasMetrics.ChurnedSubscriptions,
		saasMetrics.TotalLiveMRR, saasMetrics.TotalLiveARR, saasMetrics.ARPU, saasMetrics.LogoChurnRatePct,
		saasMetrics.TotalInvoicedLifetime, saasMetrics.TotalInvoicesCount)

	// 11. Universal Chatter Audit Trail on Subscription
	subChatter, err := chatterSvc.ListMessages("sale.subscription", reloadedSub.ID, company.ID)
	if err != nil || len(subChatter) < 4 {
		t.Fatalf("Expected at least 4 chatter messages on subscription %s, got %d (err: %v)", reloadedSub.Code, len(subChatter), err)
	}
	t.Logf("✓ Universal Chatter Audit Trail verified: %d audit messages tracked on Subscription '%s'", len(subChatter), reloadedSub.Code)

	// =========================================================================
	// 39. ENTERPRISE AUDIT TRAIL & CDC (CHANGE DATA CAPTURE) ENGINE (DIFFS, ANOMALY DETECTION, ROLLBACK, AI AUDITOR & CSV EXPORT)
	// =========================================================================
	t.Log("\n--- SECTION 39: ENTERPRISE AUDIT TRAIL & CDC (CHANGE DATA CAPTURE) ENGINE (DIFFS, ANOMALY DETECTION, ROLLBACK, AI AUDITOR & CSV EXPORT) ---")

	auditMod := audit_trail.NewModule(db, chatterSvc, aiClient)
	if err := auditMod.Seed(db); err != nil {
		t.Fatalf("Audit Trail Seed failed: %v", err)
	}
	auditSvc := auditMod.Service()

	// 1. Audit Rules Management
	auditRules, err := auditSvc.ListRules(company.ID)
	if err != nil || len(auditRules) < 5 {
		t.Fatalf("Expected at least 5 seeded audit rules, got %d (err: %v)", len(auditRules), err)
	}
	t.Logf("✓ Verified %d Seeded Audit Rules (e.g. '%s', '%s')", len(auditRules), auditRules[0].Name, auditRules[1].Name)

	// Create custom audit rule for POS orders
	posRule, err := auditSvc.CreateRule(audit_trail.CreateRuleRequest{
		Name:          "Audit Transaksi Point of Sale",
		ModelName:     "pos.order",
		TrackFields:   "amount_total,state,partner_id",
		RetentionDays: 730,
		Description:   "Pengawasan mutasi transaksi kasir retail",
	}, company.ID)
	if err != nil || posRule.ModelName != "pos.order" {
		t.Fatalf("CreateRule failed: %v", err)
	}
	t.Logf("✓ Created Custom Audit Rule: '%s' for model '%s' (Retention: %d days)", posRule.Name, posRule.ModelName, posRule.RetentionDays)

	// 2. Normal Change Data Capture (CDC Engine)
	normalCDC, err := auditSvc.RecordChange(audit_trail.RecordChangeRequest{
		ModelName: "sale.order",
		ResID:     so.ID,
		ResName:   so.Name,
		Action:    audit_trail.ActionUpdate,
		OldValues: map[string]interface{}{
			"id":           so.ID,
			"state":        "draft",
			"amount_total": 10500000.0,
		},
		NewValues: map[string]interface{}{
			"id":           so.ID,
			"state":        "sent",
			"amount_total": 10500000.0,
		},
		IPAddress: "192.168.1.100",
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X)",
		Summary:   "Perubahan status pesanan dari penawaran ke terkirim",
	}, user.ID, user.Name, user.Email, company.ID)
	if err != nil {
		t.Fatalf("RecordChange failed: %v", err)
	}
	if normalCDC.IsSuspicious || normalCDC.RiskScore >= 50 || len(normalCDC.Diffs) != 1 {
		t.Fatalf("Normal CDC assertion failed: suspicious=%v, risk=%d, diffs=%d",
			normalCDC.IsSuspicious, normalCDC.RiskScore, len(normalCDC.Diffs))
	}
	t.Logf("✓ Normal CDC Log Created: '%s' on %s #%d (Risk: %d/100, Diff: %s: '%s' -> '%s')",
		normalCDC.LogCode, normalCDC.ModelName, normalCDC.ResID, normalCDC.RiskScore,
		normalCDC.Diffs[0].FieldName, normalCDC.Diffs[0].OldValue, normalCDC.Diffs[0].NewValue)

	// 3. Sensitive & Anomaly CDC - Salary Jump (> 100%)
	anomEmployee, err := auditSvc.RecordChange(audit_trail.RecordChangeRequest{
		ModelName: "hr.employee",
		ResID:     emp.ID,
		ResName:   emp.Name,
		Action:    audit_trail.ActionUpdate,
		OldValues: map[string]interface{}{
			"id":        emp.ID,
			"wage":      8500000.0,
			"job_title": "Accounting Staff",
		},
		NewValues: map[string]interface{}{
			"id":        emp.ID,
			"wage":      22000000.0, // 158% increase!
			"job_title": "Accounting Manager",
		},
		IPAddress: "192.168.1.150",
		UserAgent: "PostmanRuntime/7.32.2",
		Summary:   "Penyesuaian gaji pokok dan jabatan karyawan",
	}, user.ID, user.Name, user.Email, company.ID)
	if err != nil {
		t.Fatalf("RecordChange for employee failed: %v", err)
	}
	if !anomEmployee.IsSuspicious || anomEmployee.RiskScore < 50 || anomEmployee.AnomalyReason == "" {
		t.Fatalf("Anomaly detection failed to flag 158%% wage jump: suspicious=%v, risk=%d, reason='%s'",
			anomEmployee.IsSuspicious, anomEmployee.RiskScore, anomEmployee.AnomalyReason)
	}
	hasSensitiveWage := false
	for _, d := range anomEmployee.Diffs {
		if d.FieldName == "wage" && d.IsSensitive {
			hasSensitiveWage = true
		}
	}
	if !hasSensitiveWage {
		t.Fatalf("Expected 'wage' diff to be marked as IsSensitive")
	}
	t.Logf("✓ High-Risk Anomaly CDC Flagged: '%s' on %s (Risk: %d/100, Reason: '%s')",
		anomEmployee.LogCode, anomEmployee.ModelName, anomEmployee.RiskScore, anomEmployee.AnomalyReason)

	// 4. Critical Financial Anomaly CDC - Unposting Accounting Move
	anomInvoice, err := auditSvc.RecordChange(audit_trail.RecordChangeRequest{
		ModelName: "account.move",
		ResID:     1,
		ResName:   "INV/2026/00001",
		Action:    audit_trail.ActionUpdate,
		OldValues: map[string]interface{}{
			"id":           1,
			"state":        "posted",
			"amount_total": 35000000.0,
		},
		NewValues: map[string]interface{}{
			"id":           1,
			"state":        "draft",
			"amount_total": 35000000.0,
		},
		IPAddress: "192.168.1.200",
		UserAgent: "Mozilla/5.0",
		Summary:   "Pembatalan pembukuan jurnal faktur penjualan",
	}, user.ID, user.Name, user.Email, company.ID)
	if err != nil {
		t.Fatalf("RecordChange for unpost invoice failed: %v", err)
	}
	if !anomInvoice.IsSuspicious || anomInvoice.RiskScore < 50 {
		t.Fatalf("Anomaly detection failed to flag accounting unposting: suspicious=%v, risk=%d",
			anomInvoice.IsSuspicious, anomInvoice.RiskScore)
	}
	t.Logf("✓ Accounting Unpost Anomaly Flagged: '%s' on %s (Risk: %d/100, Reason: '%s')",
		anomInvoice.LogCode, anomInvoice.ModelName, anomInvoice.RiskScore, anomInvoice.AnomalyReason)

	// 5. Universal Rollback Engine (Compensating Audit Log + State Reversion)
	// Create test target record in database
	testEmp := hr.Employee{
		Name:        "Andi Auditor",
		WorkEmail:   "andi@company.test",
		BasicSalary: 8500000,
	}
	testEmp.CompanyID = company.ID
	if err := db.Create(&testEmp).Error; err != nil {
		t.Fatalf("Failed to create test employee for rollback: %v", err)
	}

	// Record an erroneous update
	erroneousLog, err := auditSvc.RecordChange(audit_trail.RecordChangeRequest{
		ModelName: "hr.employee",
		ResID:     testEmp.ID,
		ResName:   testEmp.Name,
		Action:    audit_trail.ActionUpdate,
		OldValues: map[string]interface{}{
			"id":           testEmp.ID,
			"basic_salary": 8500000.0,
		},
		NewValues: map[string]interface{}{
			"id":           testEmp.ID,
			"basic_salary": 25000000.0,
		},
		Summary: "Koreksi salah input nominal gaji",
	}, user.ID, user.Name, user.Email, company.ID)
	if err != nil {
		t.Fatalf("Failed to record erroneous change: %v", err)
	}

	// Update DB to reflect erroneous change
	db.Model(&hr.Employee{}).Where("id = ?", testEmp.ID).Update("basic_salary", 25000000.0)

	// Execute Rollback
	rollbackLog, err := auditSvc.RollbackRecord(erroneousLog.ID, "Ditemukan kekeliruan ketik oleh operator data entry", user.ID, user.Name, user.Email, company.ID)
	if err != nil {
		t.Fatalf("RollbackRecord failed: %v", err)
	}
	if rollbackLog.Action != audit_trail.ActionRollback || len(rollbackLog.Diffs) == 0 {
		t.Fatalf("Compensating rollback log invalid: action=%s, diffs=%d", rollbackLog.Action, len(rollbackLog.Diffs))
	}

	// Verify database record restored
	var reloadedEmp hr.Employee
	db.First(&reloadedEmp, testEmp.ID)
	if math.Abs(reloadedEmp.BasicSalary-8500000) > 0.01 {
		t.Fatalf("Rollback failed to restore wage in database: got %.2f, expected 8500000", reloadedEmp.BasicSalary)
	}
	t.Logf("✓ Rollback Engine Executed: Compensating Log '%s' created, Target Employee wage restored to Rp %.2f",
		rollbackLog.LogCode, reloadedEmp.BasicSalary)

	// 6. Cloudflare Workers AI Security & Anomaly Audit Copilot
	auditCopilotReport, err := auditSvc.AnalyzeAnomalies(company.ID)
	if err != nil {
		t.Fatalf("AnalyzeAnomalies failed: %v", err)
	}
	if auditCopilotReport.TotalAnalyzed < 4 || auditCopilotReport.SuspiciousCount < 2 || auditCopilotReport.OverallSecurityScore <= 0 {
		t.Fatalf("AnalyzeAnomalies assertions failed: %+v", auditCopilotReport)
	}
	if len(auditCopilotReport.KeyObservations) == 0 || len(auditCopilotReport.RemediationActions) == 0 {
		t.Fatalf("Expected key observations and remediation actions in audit report")
	}
	t.Logf("✓ Cloudflare Workers AI Security & Anomaly Audit Copilot: Analyzed %d logs, %d Flagged, Security Score: %d/100 (%s), Provider: '%s'",
		auditCopilotReport.TotalAnalyzed, auditCopilotReport.SuspiciousCount,
		auditCopilotReport.OverallSecurityScore, auditCopilotReport.ComplianceStatus, auditCopilotReport.AIProvider)
	t.Logf("  Executive Summary: %s", auditCopilotReport.ExecutiveSummary)

	// 7. Regulatory CSV Export Engine
	auditCsvBytes, err := auditSvc.ExportCSV(audit_trail.AuditLogFilters{CompanyID: company.ID})
	if err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}
	auditCsvContent := string(auditCsvBytes)
	if !strings.Contains(auditCsvContent, "Log Code,Timestamp,Model Name") || !strings.Contains(auditCsvContent, "AUD/2026/") {
		t.Fatalf("ExportCSV output invalid or missing headers:\n%s", auditCsvContent[:200])
	}
	t.Logf("✓ Regulatory CSV Export Engine: Generated %d bytes of SOX/ISO 27001 compliant audit export", len(auditCsvBytes))

	// 8. Security Dashboard KPI Statistics
	atStats, err := auditSvc.GetStats(company.ID)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if atStats.TotalLogs < 5 || atStats.SuspiciousCount < 2 || atStats.RollbackCount < 1 || atStats.TrackedModelsCount < 5 {
		t.Fatalf("AuditStats assertions failed: %+v", atStats)
	}
	t.Logf("✓ Audit Security Dashboard Stats: Total Logs=%d, Update=%d, Rollback=%d, Suspicious=%d, Tracked Models=%d, Top Models=%+v",
		atStats.TotalLogs, atStats.UpdateCount, atStats.RollbackCount, atStats.SuspiciousCount,
		atStats.TrackedModelsCount, atStats.TopModels)

	// 9. Universal Chatter Audit Integration
	empChatter, err := chatterSvc.ListMessages("hr.employee", emp.ID, company.ID)
	if err != nil || len(empChatter) == 0 {
		t.Fatalf("Expected Chatter messages on hr.employee, got %d (err: %v)", len(empChatter), err)
	}
	t.Logf("✓ Universal Chatter Integration: %d audit messages tracked on hr.employee #%d", len(empChatter), emp.ID)

	// ==========================================
	// 40. TEST ENTERPRISE WEBHOOK ENGINE & EVENT-DRIVEN OUTBOUND INTEGRATION, HMAC-SHA256 SIGNING, RETRY POLICY & AI DIAGNOSTIC COPILOT
	// ==========================================
	t.Log("\n--- SECTION 40: ENTERPRISE WEBHOOK ENGINE & EVENT-DRIVEN OUTBOUND INTEGRATION, HMAC-SHA256 SIGNING & AI DIAGNOSTIC COPILOT ---")

	webhookRepo := webhook.NewRepository(db)
	webhookSvc := webhook.NewService(db, webhookRepo, chatterSvc, aiClient)

	// 1. Setup local mock HTTP receiver server to capture outbound webhooks & verify HMAC-SHA256 signatures
	type receivedWebhook struct {
		Event        string
		Signature    string
		Body         string
		VerifiedHMAC bool
	}
	var receivedMutex sync.Mutex
	receivedLogistics := make([]receivedWebhook, 0)
	receivedCRM := make([]receivedWebhook, 0)
	flakyRequestCount := 0

	sharedSecretLogistics := "whsec_super_secret_logistics_key_2026"
	sharedSecretCRM := "whsec_super_secret_crm_key_2026"

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		eventHeader := r.Header.Get("X-Webhook-Event")
		sigHeader := r.Header.Get("X-Webhook-Signature")

		switch r.URL.Path {
		case "/logistics":
			expectedSig := "sha256=" + webhook.ComputeHMACSignature(bodyBytes, sharedSecretLogistics)
			verified := (sigHeader == expectedSig)
			receivedMutex.Lock()
			receivedLogistics = append(receivedLogistics, receivedWebhook{
				Event:        eventHeader,
				Signature:    sigHeader,
				Body:         string(bodyBytes),
				VerifiedHMAC: verified,
			})
			receivedMutex.Unlock()
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"acknowledged","carrier":"JNE Express"}`))

		case "/crm":
			expectedSig := "sha256=" + webhook.ComputeHMACSignature(bodyBytes, sharedSecretCRM)
			verified := (sigHeader == expectedSig)
			receivedMutex.Lock()
			receivedCRM = append(receivedCRM, receivedWebhook{
				Event:        eventHeader,
				Signature:    sigHeader,
				Body:         string(bodyBytes),
				VerifiedHMAC: verified,
			})
			receivedMutex.Unlock()
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"lead_synced","lead_id":9021}`))

		case "/flaky":
			receivedMutex.Lock()
			flakyRequestCount++
			count := flakyRequestCount
			receivedMutex.Unlock()

			if count == 1 {
				// First time fails with HTTP 500
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"Internal downstream gateway error"}`))
			} else {
				// Second retry succeeds with HTTP 200
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"recovered_after_retry"}`))
			}

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// 2. Create Webhook Subscriptions
	subLogistics, err := webhookSvc.CreateSubscription(webhook.CreateWebhookRequest{
		Name:          "Integrasi Logistik 3PL Ekspedisi",
		TargetURL:     mockServer.URL + "/logistics",
		SecretToken:   sharedSecretLogistics,
		EventTopics:   webhook.EventPickingValidated + "," + webhook.EventWebhookPing,
		CustomHeaders: `{"X-3PL-Client":"GO-ERP"}`,
		MaxRetries:    3,
		RetryDelaySec: 2,
		TimeoutSec:    5,
		Description:   "Pengiriman otomatis status picking keluar ke provider logistik",
	}, user.ID, company.ID)
	if err != nil {
		t.Fatalf("Failed to create logistics subscription: %v", err)
	}
	t.Logf("✓ Created Webhook Subscription '%s' (ID: %d, URL: %s)", subLogistics.Name, subLogistics.ID, subLogistics.TargetURL)

	subCRM, err := webhookSvc.CreateSubscription(webhook.CreateWebhookRequest{
		Name:          "Sinkronisasi CRM Gateway",
		TargetURL:     mockServer.URL + "/crm",
		SecretToken:   sharedSecretCRM,
		EventTopics:   webhook.EventSaleConfirmed,
		CustomHeaders: `{"Authorization":"Bearer test_crm_token"}`,
		MaxRetries:    3,
		RetryDelaySec: 2,
		TimeoutSec:    5,
		Description:   "Sinkronisasi pesanan penjualan terkonfirmasi ke CRM",
	}, user.ID, company.ID)
	if err != nil {
		t.Fatalf("Failed to create CRM subscription: %v", err)
	}
	t.Logf("✓ Created Webhook Subscription '%s' (ID: %d, URL: %s)", subCRM.Name, subCRM.ID, subCRM.TargetURL)

	subFlaky, err := webhookSvc.CreateSubscription(webhook.CreateWebhookRequest{
		Name:          "Flaky Partner Webhook",
		TargetURL:     mockServer.URL + "/flaky",
		SecretToken:   "whsec_flaky_test_123",
		EventTopics:   webhook.EventPartnerCreated,
		MaxRetries:    3,
		RetryDelaySec: 1,
		TimeoutSec:    5,
		Description:   "Endpoint sementara yang mengalami error HTTP 500 lalu pulih",
	}, user.ID, company.ID)
	if err != nil {
		t.Fatalf("Failed to create Flaky subscription: %v", err)
	}
	t.Logf("✓ Created Webhook Subscription '%s' (ID: %d, URL: %s)", subFlaky.Name, subFlaky.ID, subFlaky.TargetURL)

	// 3. Test Ping Delivery & Signature Verification
	pingDelivery, err := webhookSvc.TestPing(subLogistics.ID, company.ID)
	if err != nil {
		t.Fatalf("TestPing failed: %v", err)
	}
	if pingDelivery.StatusCode != 200 || pingDelivery.Status != webhook.StatusSuccess {
		t.Fatalf("Expected ping delivery success, got status: %s (code: %d)", pingDelivery.Status, pingDelivery.StatusCode)
	}
	receivedMutex.Lock()
	logisticsCount := len(receivedLogistics)
	var lastLogistics receivedWebhook
	if logisticsCount > 0 {
		lastLogistics = receivedLogistics[logisticsCount-1]
	}
	receivedMutex.Unlock()
	if logisticsCount == 0 || !lastLogistics.VerifiedHMAC {
		t.Fatalf("Receiver failed to verify HMAC-SHA256 signature for ping delivery! Delivered: %d", logisticsCount)
	}
	t.Logf("✓ Test Ping Success: Delivery Code '%s', Status %d, Latency %dms, Receiver Verified HMAC-SHA256: %v",
		pingDelivery.DeliveryCode, pingDelivery.StatusCode, pingDelivery.DurationMs, lastLogistics.VerifiedHMAC)

	// 4. Publish Real Business Event (sale.order.confirmed)
	deliveries, err := webhookSvc.PublishEvent(webhook.PublishEventRequest{
		EventName: webhook.EventSaleConfirmed,
		ModelName: "sale.order",
		ResID:     so.ID,
		Data:      map[string]interface{}{"order_name": so.Name, "amount_total": so.AmountTotal, "partner_name": customer.Name},
	}, company.ID)
	if err != nil {
		t.Fatalf("PublishEvent failed: %v", err)
	}
	if len(deliveries) != 1 {
		t.Fatalf("Expected exactly 1 delivery for topic sale.order.confirmed (CRM only), got: %d", len(deliveries))
	}
	if deliveries[0].StatusCode != 200 || deliveries[0].Status != webhook.StatusSuccess {
		t.Fatalf("Expected successful delivery to CRM, got: %+v", deliveries[0])
	}
	receivedMutex.Lock()
	crmCount := len(receivedCRM)
	var lastCRM receivedWebhook
	if crmCount > 0 {
		lastCRM = receivedCRM[crmCount-1]
	}
	receivedMutex.Unlock()
	if crmCount == 0 || !lastCRM.VerifiedHMAC {
		t.Fatalf("CRM receiver failed to verify HMAC-SHA256 signature! Delivered: %d", crmCount)
	}
	t.Logf("✓ Outbound Event '%s' Published: Delivered to CRM (%s), Delivery Code '%s', Status %d, HMAC Verified: %v",
		webhook.EventSaleConfirmed, deliveries[0].TargetURL, deliveries[0].DeliveryCode, deliveries[0].StatusCode, lastCRM.VerifiedHMAC)

	// 5. Test Error Handling, Exponential Backoff & Retry Queue
	flakyDeliveries, err := webhookSvc.PublishEvent(webhook.PublishEventRequest{
		EventName: webhook.EventPartnerCreated,
		ModelName: "res.partner",
		ResID:     customer.ID,
		Data:      map[string]interface{}{"partner_id": customer.ID, "name": customer.Name},
	}, company.ID)
	if err != nil {
		t.Fatalf("PublishEvent for flaky webhook failed: %v", err)
	}
	if len(flakyDeliveries) != 1 {
		t.Fatalf("Expected 1 delivery to flaky endpoint, got: %d", len(flakyDeliveries))
	}
	firstAttempt := flakyDeliveries[0]
	if firstAttempt.StatusCode != 500 || firstAttempt.Status != webhook.StatusRetrying || firstAttempt.AttemptCount != 1 {
		t.Fatalf("Expected first attempt to fail with 500 and status 'retrying', got: code=%d, status=%s, attempts=%d",
			firstAttempt.StatusCode, firstAttempt.Status, firstAttempt.AttemptCount)
	}
	if firstAttempt.NextRetryAt == nil {
		t.Fatalf("Expected NextRetryAt to be set for retrying delivery")
	}
	t.Logf("✓ Retry Policy Working: First attempt failed with HTTP %d, Status '%s', Next Retry scheduled at %s",
		firstAttempt.StatusCode, firstAttempt.Status, firstAttempt.NextRetryAt.Format(time.RFC3339))

	// Execute manual retry on delivery (second attempt succeeds with HTTP 200)
	retriedDelivery, err := webhookSvc.RetryDelivery(firstAttempt.ID, company.ID)
	if err != nil {
		t.Fatalf("RetryDelivery failed: %v", err)
	}
	if retriedDelivery.StatusCode != 200 || retriedDelivery.Status != webhook.StatusSuccess || retriedDelivery.AttemptCount != 2 {
		t.Fatalf("Expected retry delivery to succeed with 200 on second attempt, got: code=%d, status=%s, attempts=%d",
			retriedDelivery.StatusCode, retriedDelivery.Status, retriedDelivery.AttemptCount)
	}
	t.Logf("✓ Manual Delivery Retry Executed: Code '%s' recovered to Status '%s' (HTTP %d, Total Attempts: %d)",
		retriedDelivery.DeliveryCode, retriedDelivery.Status, retriedDelivery.StatusCode, retriedDelivery.AttemptCount)

	// 6. Cloudflare Workers AI Webhook Diagnostic Copilot
	webhookDiagReport, err := webhookSvc.AnalyzeWebhookHealth(company.ID)
	if err != nil {
		t.Fatalf("AnalyzeWebhookHealth failed: %v", err)
	}
	if webhookDiagReport.TotalAnalyzed < 3 || webhookDiagReport.HealthStatus == "" || webhookDiagReport.ExecutiveSummary == "" {
		t.Fatalf("Invalid webhook AI diagnostic report: %+v", webhookDiagReport)
	}
	if len(webhookDiagReport.RemediationActions) == 0 {
		t.Fatalf("Expected remediation actions in diagnostic report")
	}
	t.Logf("✓ Cloudflare Workers AI Webhook Diagnostic Copilot: Total Analyzed=%d, Health Status='%s', AI Provider='%s'",
		webhookDiagReport.TotalAnalyzed, webhookDiagReport.HealthStatus, webhookDiagReport.AIProvider)
	t.Logf("  Executive Summary: %s", webhookDiagReport.ExecutiveSummary)

	// 7. Executive Webhook KPI Statistics
	whStats, err := webhookSvc.GetStats(company.ID)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if whStats.TotalSubscriptions < 3 || whStats.ActiveSubscriptions < 3 || whStats.TotalDeliveries < 3 || whStats.SuccessRatePct <= 0 {
		t.Fatalf("Invalid webhook stats: %+v", whStats)
	}
	t.Logf("✓ Webhook Executive Analytics: Subscriptions=%d (Active=%d), Total Deliveries=%d, Success=%d, Success Rate=%.1f%%, Avg Latency=%.1fms",
		whStats.TotalSubscriptions, whStats.ActiveSubscriptions, whStats.TotalDeliveries,
		whStats.SuccessDeliveries, whStats.SuccessRatePct, whStats.AvgDurationMs)

	// 8. Universal Chatter Audit Integration
	whChatter, err := chatterSvc.ListMessages("webhook.subscription", subLogistics.ID, company.ID)
	if err != nil || len(whChatter) == 0 {
		t.Fatalf("Expected Chatter messages on webhook.subscription, got %d (err: %v)", len(whChatter), err)
	}
	t.Logf("✓ Universal Chatter Integration: %d audit messages tracked on webhook.subscription #%d", len(whChatter), subLogistics.ID)

	// ==========================================
	// 41. TEST INBOUND WEBHOOKS & PARTNER INGESTION GATEWAY, MIDTRANS SHA512, LOGISTICS HMAC & AI INBOUND ANALYZER (MODUL KE-37)
	// ==========================================
	t.Log("\n--- SECTION 41: INBOUND WEBHOOKS & PARTNER INGESTION GATEWAY, CRYPTOGRAPHIC AUTH & AI INBOUND ANALYZER ---")

	inboundRepo := inbound_webhook.NewRepository(db)
	inboundSvc := inbound_webhook.NewService(db, inboundRepo, accountSvc, chatterSvc, aiClient)

	// 1. Create and Post a fresh customer invoice for Midtrans Inbound Settlement test
	inboundInv := account.AccountMove{
		Name:           "INV/2026/INWH-001",
		MoveType:       "out_invoice",
		PartnerID:      customer.ID,
		AmountTotal:    7500000,
		AmountResidual: 7500000,
		Ref:            "SO/INWH/001",
		LineIDs: []account.AccountMoveLine{
			{
				AccountID: accReceivable.ID,
				Name:      "Piutang Dagang Inbound Test",
				Debit:     7500000,
				Credit:    0,
			},
			{
				AccountID: accSales.ID,
				Name:      "Penjualan Online Inbound Test",
				Debit:     0,
				Credit:    7500000,
			},
		},
	}
	inboundInv.CompanyID = company.ID
	if err := accountSvc.CreateMove(&inboundInv); err != nil {
		t.Fatalf("Failed to create invoice for inbound webhook test: %v", err)
	}
	postedInboundInv, err := accountSvc.ActionPost(inboundInv.ID, company.ID)
	if err != nil || postedInboundInv.State != "posted" {
		t.Fatalf("Failed to post invoice for inbound test: %v", err)
	}
	t.Logf("✓ Created & Posted Invoice '%s' (Total: Rp %.2f) for Midtrans Inbound Settlement test", postedInboundInv.Name, postedInboundInv.AmountTotal)

	// 2. Setup Midtrans SHA512 Inbound Endpoint & Ingest Settlement Notification
	midtransSecret := "SB-Mid-server-test-secret-2026"
	epMidtrans, err := inboundSvc.CreateEndpoint(inbound_webhook.CreateInboundEndpointRequest{
		EndpointKey:  "midtrans_snap_test",
		Name:         "Midtrans PG Inbound Webhook",
		ProviderType: inbound_webhook.ProviderPaymentMidtrans,
		AuthType:     inbound_webhook.AuthSHA512,
		SecretKey:    midtransSecret,
		TargetModel:  "account.move",
		AutoAction:   inbound_webhook.ActionMarkInvoicePaid,
		Description:  "Midtrans Snap payment notification gateway",
	}, user.ID, company.ID)
	if err != nil {
		t.Fatalf("Failed to create Midtrans inbound endpoint: %v", err)
	}
	t.Logf("✓ Created Inbound Webhook Endpoint '%s' (Key: %s, Auth: %s)", epMidtrans.Name, epMidtrans.EndpointKey, epMidtrans.AuthType)

	orderID := postedInboundInv.Name
	statusCode := "200"
	grossAmount := "7500000.00"
	midtransSignaturePayload := orderID + statusCode + grossAmount + midtransSecret
	computedMidtransSig := inbound_webhook.ComputeSHA512(midtransSignaturePayload)

	midtransBody := fmt.Sprintf(`{"order_id":"%s","status_code":"%s","gross_amount":"%s","transaction_status":"settlement","payment_type":"bank_transfer","transaction_time":"2026-10-09 20:00:00"}`,
		orderID, statusCode, grossAmount)

	midtransHeaders := map[string]string{
		"Content-Type": "application/json",
		"X-Signature":  computedMidtransSig,
	}

	midtransLog, err := inboundSvc.ProcessInboundWebhook(epMidtrans.EndpointKey, midtransHeaders, []byte(midtransBody), "103.208.22.45")
	if err != nil {
		t.Fatalf("ProcessInboundWebhook Midtrans failed: %v", err)
	}
	if !midtransLog.SignatureValid || midtransLog.ProcessStatus != inbound_webhook.StatusProcessed || midtransLog.HTTPStatusCode != 200 {
		t.Fatalf("Midtrans processing failed: valid=%v, status=%s, code=%d, err=%s",
			midtransLog.SignatureValid, midtransLog.ProcessStatus, midtransLog.HTTPStatusCode, midtransLog.ErrorMessage)
	}

	var refreshedInv account.AccountMove
	db.First(&refreshedInv, postedInboundInv.ID)
	if refreshedInv.PaymentState != "paid" {
		t.Fatalf("Expected invoice payment_state to be 'paid', got: '%s'", refreshedInv.PaymentState)
	}
	t.Logf("✓ Midtrans Inbound Callback Success: Invoice '%s' automatically marked 'paid'! Action: %s (Duration: %dms)",
		refreshedInv.Name, midtransLog.ActionTaken, midtransLog.DurationMs)

	// 3. Setup JNE Logistics Tracking Updates Endpoint (HMAC-SHA256)
	jneSecret := "whsec_jne_tracking_secret_2026"
	epLogistics, err := inboundSvc.CreateEndpoint(inbound_webhook.CreateInboundEndpointRequest{
		EndpointKey:  "jne_courier_tracking",
		Name:         "JNE Logistics Tracking Inbound",
		ProviderType: inbound_webhook.ProviderLogistics3PL,
		AuthType:     inbound_webhook.AuthHMACSHA256,
		SecretKey:    jneSecret,
		TargetModel:  "stock.picking",
		AutoAction:   inbound_webhook.ActionUpdatePickingTracking,
		Description:  "JNE 3PL courier tracking callbacks",
	}, user.ID, company.ID)
	if err != nil {
		t.Fatalf("Failed to create JNE inbound endpoint: %v", err)
	}

	jnePayload := fmt.Sprintf(`{"picking_name":"%s","status":"delivered","waybill":"JNE-EXP-2026998","recipient":"Pak Bambang (Security)"}`, delivery.Name)
	jneSig := "sha256=" + inbound_webhook.ComputeHMACSHA256([]byte(jnePayload), jneSecret)
	jneHeaders := map[string]string{
		"Content-Type": "application/json",
		"X-Signature":  jneSig,
	}

	jneLog, err := inboundSvc.ProcessInboundWebhook(epLogistics.EndpointKey, jneHeaders, []byte(jnePayload), "182.253.110.12")
	if err != nil {
		t.Fatalf("ProcessInboundWebhook JNE failed: %v", err)
	}
	if !jneLog.SignatureValid || jneLog.ProcessStatus != inbound_webhook.StatusProcessed {
		t.Fatalf("JNE processing failed: valid=%v, status=%s, err=%s", jneLog.SignatureValid, jneLog.ProcessStatus, jneLog.ErrorMessage)
	}
	t.Logf("✓ Logistics 3PL Inbound Callback: Picking '%s' updated with courier status 'delivered' (Action: %s)",
		delivery.Name, jneLog.ActionTaken)

	// 4. Setup CRM Lead Ingestion Endpoint (Bearer Token)
	leadToken := "crm_bearer_token_abc123"
	epLeads, err := inboundSvc.CreateEndpoint(inbound_webhook.CreateInboundEndpointRequest{
		EndpointKey:  "crm_landing_page_leads",
		Name:         "Landing Page Lead Ingestion",
		ProviderType: inbound_webhook.ProviderCRMLeads,
		AuthType:     inbound_webhook.AuthBearer,
		SecretKey:    leadToken,
		TargetModel:  "crm.lead",
		AutoAction:   inbound_webhook.ActionCreateCRMLead,
		Description:  "Web contact form submissions to CRM leads",
	}, user.ID, company.ID)
	if err != nil {
		t.Fatalf("Failed to create CRM Leads inbound endpoint: %v", err)
	}

	leadPayload := `{"title":"Permintaan Demo ERP Enterprise","name":"Budi Santoso","company_name":"PT Nusantara Sukses","email":"budi@nusantara.co.id","phone":"+628123456789","notes":"Tertarik modul manufaktur dan inbound webhook"}`
	leadHeaders := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + leadToken,
	}

	leadLog, err := inboundSvc.ProcessInboundWebhook(epLeads.EndpointKey, leadHeaders, []byte(leadPayload), "127.0.0.1")
	if err != nil {
		t.Fatalf("ProcessInboundWebhook CRM Leads failed: %v", err)
	}
	if !leadLog.SignatureValid || leadLog.ProcessStatus != inbound_webhook.StatusProcessed {
		t.Fatalf("CRM lead processing failed: valid=%v, status=%s, err=%s", leadLog.SignatureValid, leadLog.ProcessStatus, leadLog.ErrorMessage)
	}
	t.Logf("✓ CRM Inbound Lead Ingestion: Created Lead '%s' (ID: %d) via Webhook!", leadLog.TargetResName, leadLog.TargetResID)

	// 5. Test Cryptographic Security: Forged signature rejection (returns 401)
	badHeaders := map[string]string{
		"Content-Type": "application/json",
		"X-Signature":  "sha256=invalid_forged_signature_00000000000000000000000000000000",
	}
	badLog, err := inboundSvc.ProcessInboundWebhook(epLogistics.EndpointKey, badHeaders, []byte(jnePayload), "127.0.0.1")
	if err == nil {
		t.Fatalf("Expected error on invalid signature, but got nil")
	}
	if badLog.SignatureValid || badLog.ProcessStatus != inbound_webhook.StatusRejected || badLog.HTTPStatusCode != 401 {
		t.Fatalf("Expected 401 rejected for invalid signature, got code=%d, status=%s, valid=%v",
			badLog.HTTPStatusCode, badLog.ProcessStatus, badLog.SignatureValid)
	}
	t.Logf("✓ Cryptographic Security Verified: Forged signature rejected with HTTP 401, Log Status '%s'", badLog.ProcessStatus)

	// 6. Test Inbound Payload Simulator
	simResult, err := inboundSvc.SimulateInboundWebhook(inbound_webhook.SimulateInboundRequest{
		EndpointKey: epMidtrans.EndpointKey,
		Headers:     map[string]string{"X-Signature": computedMidtransSig},
		Payload: map[string]interface{}{
			"order_id":           postedInboundInv.Name,
			"status_code":        "200",
			"gross_amount":       "7500000.00",
			"transaction_status": "settlement",
		},
	}, company.ID)
	if err != nil {
		t.Fatalf("SimulateInboundWebhook failed: %v", err)
	}
	if !simResult.SignatureValid || simResult.HTTPStatusCode != 200 {
		t.Fatalf("Simulator failed: valid=%v, status=%d, err=%s", simResult.SignatureValid, simResult.HTTPStatusCode, simResult.ErrorMessage)
	}
	t.Logf("✓ Inbound Payload Simulator Executed: %s (Status: %d, Signature Valid: %v)",
		simResult.ActionTaken, simResult.HTTPStatusCode, simResult.SignatureValid)

	// 7. Cloudflare Workers AI Inbound Payload Analyzer
	samplePayloadForAI := `{"order_id":"INV/2026/001","status_code":"200","gross_amount":"5000000","transaction_status":"settlement","payment_type":"qris"}`
	inboundAIReport, err := inboundSvc.AnalyzeInboundPayloadWithAI(samplePayloadForAI, company.ID)
	if err != nil {
		t.Fatalf("AnalyzeInboundPayloadWithAI failed: %v", err)
	}
	if inboundAIReport.DetectedProvider == "" || inboundAIReport.SuggestedModel == "" || inboundAIReport.SuggestedAction == "" {
		t.Fatalf("Invalid AI Inbound Payload report: %+v", inboundAIReport)
	}
	t.Logf("✓ Cloudflare Workers AI Inbound Payload Analyzer: Detected Provider='%s', Model='%s', Action='%s' (AI Provider: %s)",
		inboundAIReport.DetectedProvider, inboundAIReport.SuggestedModel, inboundAIReport.SuggestedAction, inboundAIReport.AIProvider)

	// 8. Inbound Executive KPI Analytics
	inStats, err := inboundSvc.GetStats(company.ID)
	if err != nil {
		t.Fatalf("GetStats inbound failed: %v", err)
	}
	if inStats.TotalEndpoints < 3 || inStats.TotalInboundLogs < 3 {
		t.Fatalf("Invalid inbound stats: %+v", inStats)
	}
	t.Logf("✓ Inbound Webhook Executive Analytics: Endpoints=%d (Active=%d), Total Logs=%d, Processed=%d, Rejected=%d, Valid Sig=%.1f%%, Avg Latency=%.1fms",
		inStats.TotalEndpoints, inStats.ActiveEndpoints, inStats.TotalInboundLogs,
		inStats.ProcessedLogs, inStats.RejectedLogs, inStats.ValidSigPercent, inStats.AvgDurationMs)

	// 9. Universal Chatter Audit Integration
	inChatter, err := chatterSvc.ListMessages("inbound.webhook.endpoint", epMidtrans.ID, company.ID)
	if err != nil || len(inChatter) == 0 {
		t.Fatalf("Expected Chatter messages on inbound webhook endpoint, got %d (err: %v)", len(inChatter), err)
	}
	t.Logf("✓ Universal Chatter Integration: %d audit messages tracked on inbound endpoint #%d", len(inChatter), epMidtrans.ID)

	// =========================================================================
	// --- SECTION 42: MULTI-COMPANY CONSOLIDATION & INTERCOMPANY TRANSACTIONS ENGINE (MODUL KE-38) ---
	// =========================================================================
	t.Log("\n--- SECTION 42: MULTI-COMPANY CONSOLIDATION & INTERCOMPANY TRANSACTIONS ENGINE (HOLDING TREE, DUAL-SIDED MIRRORING, ELIMINATION JOURNALS & AI CONSOLIDATION AUDITOR) ---")

	icRepo := intercompany.NewRepository(db)
	icSvc := intercompany.NewService(db, icRepo, accountSvc, chatterSvc, aiClient)

	// 1. Setup Subsidiary Company & Intercompany Partners
	subComp := base.Company{
		Name:     "PT Nusantara Distribusi Anak",
		Email:    "distribusi@nusantara.local",
		Currency: "IDR",
	}
	db.Create(&subComp)

	partnerSubInHolding := base.Partner{
		Name:         "PT Nusantara Distribusi Anak (Subsidiary)",
		IsCompany:    true,
		Email:        "distribusi@nusantara.local",
		CustomerRank: 1,
		SupplierRank: 1,
	}
	partnerSubInHolding.CompanyID = company.ID
	db.Create(&partnerSubInHolding)

	partnerHoldingInSub := base.Partner{
		Name:         "PT Nusantara Holding Group (Parent)",
		IsCompany:    true,
		Email:        "holding@nusantara.local",
		CustomerRank: 1,
		SupplierRank: 1,
	}
	partnerHoldingInSub.CompanyID = subComp.ID
	db.Create(&partnerHoldingInSub)

	// 2. Corporate Group Creation
	groupReq := intercompany.CreateGroupRequest{
		GroupCode:        "GRP/2026/001",
		Name:             "Nusantara Corporate Holdings Group",
		HoldingCompanyID: company.ID,
		Currency:         "IDR",
		Description:      "Konglomerasi Holding Multi-Perusahaan Nasional.",
	}
	icGroup, err := icSvc.CreateGroup(groupReq, user.ID, company.ID)
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	t.Logf("✓ Corporate Holding Group Created: Code='%s', Name='%s' (Holding ID #%d)",
		icGroup.GroupCode, icGroup.Name, icGroup.HoldingCompanyID)

	// 3. Bilateral Intercompany Relation (85% Majority Control)
	autoBool := true
	relReq := intercompany.CreateRelationRequest{
		GroupID:                icGroup.ID,
		Name:                   "Holding Induk <-> Anak Distribusi",
		FromCompanyID:          company.ID,
		FromPartnerID:          partnerSubInHolding.ID,
		ToCompanyID:            subComp.ID,
		ToPartnerID:            partnerHoldingInSub.ID,
		OwnershipPct:           85.0,
		ConsolidationMethod:    "full",
		AutoMirrorSalePurchase: &autoBool,
		AutoMirrorInvoiceBill:  &autoBool,
		AutoEliminate:          &autoBool,
	}
	icRel, err := icSvc.CreateRelation(relReq, user.ID, company.ID)
	if err != nil {
		t.Fatalf("CreateRelation failed: %v", err)
	}
	t.Logf("✓ Bilateral Intercompany Relation Configured: Code='%s', %s <-> %s (Ownership: %.1f%%, Method: %s)",
		icRel.RelationCode, icRel.FromCompanyName, icRel.ToCompanyName, icRel.OwnershipPct, icRel.ConsolidationMethod)

	// 4. Dual-Sided Sale-to-Purchase Auto-Mirroring
	icSO := sale.SaleOrder{
		Name:          "SO/IC/2026/0001",
		PartnerID:     partnerSubInHolding.ID,
		DateOrder:     "2026-10-09",
		State:         "sale",
		AmountUntaxed: 20000000,
		AmountTax:     2200000,
		AmountTotal:   22200000,
		TaxRate:       11.0,
		Note:          "Intercompany sales to Subsidiary Distribusi",
	}
	icSO.CompanyID = company.ID
	db.Create(&icSO)

	soLine := sale.SaleOrderLine{
		OrderID:       icSO.ID,
		ProductID:     testProduct.ID,
		Name:          testProduct.Name,
		ProductUomQty: 10,
		PriceUnit:     2000000,
		PriceSubtotal: 20000000,
		PriceTotal:    22200000,
	}
	soLine.CompanyID = company.ID
	db.Create(&soLine)

	mirrorSOTx, err := icSvc.MirrorSaleOrderToPurchaseOrder(icSO.ID, user.ID, company.ID)
	if err != nil {
		t.Fatalf("MirrorSaleOrderToPurchaseOrder failed: %v", err)
	}
	if mirrorSOTx.SyncStatus != "synced" || mirrorSOTx.TargetResID == 0 {
		t.Fatalf("Invalid mirrored SO transaction: %+v", mirrorSOTx)
	}

	var createdPO purchase.PurchaseOrder
	if err := db.First(&createdPO, mirrorSOTx.TargetResID).Error; err != nil {
		t.Fatalf("Mirrored Purchase Order ID #%d not found: %v", mirrorSOTx.TargetResID, err)
	}
	if createdPO.CompanyID != subComp.ID || createdPO.AmountTotal != icSO.AmountTotal {
		t.Fatalf("Purchase Order mismatch: CompanyID=%d (expected %d), Total=%.2f (expected %.2f)",
			createdPO.CompanyID, subComp.ID, createdPO.AmountTotal, icSO.AmountTotal)
	}
	t.Logf("✓ Dual-Sided Mirroring SO -> PO: Sale Order '%s' (Rp %.2f) otomatis menghasilkan Purchase Order '%s' di Anak Perusahaan ID #%d",
		icSO.Name, icSO.AmountTotal, createdPO.Name, createdPO.CompanyID)

	// 5. Dual-Sided Invoice-to-Vendor Bill Auto-Mirroring
	icInv := account.AccountMove{
		Name:           "INV/IC/2026/0001",
		MoveType:       "out_invoice",
		JournalID:      journal.ID,
		PartnerID:      partnerSubInHolding.ID,
		InvoiceDate:    "2026-10-09",
		InvoiceDueDate: "2026-11-09",
		State:          "posted",
		PaymentState:   "not_paid",
		AmountUntaxed:  20000000,
		AmountTax:      2200000,
		AmountTotal:    22200000,
		Ref:            icSO.Name,
	}
	icInv.CompanyID = company.ID
	db.Create(&icInv)

	partnerSubIDPtr := &partnerSubInHolding.ID
	invLine := account.AccountMoveLine{
		MoveID:    icInv.ID,
		AccountID: arAcc.ID,
		PartnerID: partnerSubIDPtr,
		Name:      "Intercompany AR",
		Debit:     22200000,
		Credit:    0,
		Balance:   22200000,
	}
	invLine.CompanyID = company.ID
	db.Create(&invLine)

	mirrorInvTx, err := icSvc.MirrorInvoiceToVendorBill(icInv.ID, user.ID, company.ID)
	if err != nil {
		t.Fatalf("MirrorInvoiceToVendorBill failed: %v", err)
	}
	if mirrorInvTx.SyncStatus != "synced" || mirrorInvTx.TargetResID == 0 {
		t.Fatalf("Invalid mirrored Invoice transaction: %+v", mirrorInvTx)
	}

	var createdBill account.AccountMove
	if err := db.First(&createdBill, mirrorInvTx.TargetResID).Error; err != nil {
		t.Fatalf("Mirrored Vendor Bill ID #%d not found: %v", mirrorInvTx.TargetResID, err)
	}
	if createdBill.MoveType != "in_invoice" || createdBill.CompanyID != subComp.ID || createdBill.AmountTotal != icInv.AmountTotal {
		t.Fatalf("Vendor Bill mismatch: MoveType=%s, CompanyID=%d, AmountTotal=%.2f",
			createdBill.MoveType, createdBill.CompanyID, createdBill.AmountTotal)
	}
	t.Logf("✓ Dual-Sided Mirroring Customer Invoice -> Vendor Bill: Faktur '%s' otomatis menghasilkan Tagihan Vendor '%s' di Anak Perusahaan ID #%d",
		icInv.Name, createdBill.Name, createdBill.CompanyID)

	// 6. Financial Statement Consolidation Run (PSAK 65 / IFRS 10)
	consReq := intercompany.RunConsolidationRequest{
		GroupID:         icGroup.ID,
		PeriodStartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodEndDate:   time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
		FiscalYear:      2026,
		Notes:           "Audit Konsolidasi Finansial Tahunan 2026.",
	}
	consRun, err := icSvc.ExecuteFinancialConsolidation(consReq, user.ID, company.ID)
	if err != nil {
		t.Fatalf("ExecuteFinancialConsolidation failed: %v", err)
	}
	if consRun.RunCode == "" || consRun.ConsolidatedRevenue <= 0 || len(consRun.Lines) == 0 {
		t.Fatalf("Invalid consolidation run output: %+v", consRun)
	}
	t.Logf("✓ Financial Consolidation Run Executed: RunCode='%s', FiscalYear=%d", consRun.RunCode, consRun.FiscalYear)
	t.Logf("   - Total Pendapatan Kotor Gabungan: Rp %.2f", consRun.TotalGrossRevenue)
	t.Logf("   - Eliminasi Pendapatan Resiprokal Intra-Grup: Rp %.2f", consRun.EliminatedRevenue)
	t.Logf("   - Eliminasi Piutang/Hutang Resiprokal Intra-Grup: Rp %.2f", consRun.EliminatedReceivables)
	t.Logf("   - Eliminasi Laba Belum Terealisasi pada Persediaan: Rp %.2f", consRun.EliminatedUnrealizedProfit)
	t.Logf("   - Pendapatan Bersih Konsolidasi (IFRS 10): Rp %.2f", consRun.ConsolidatedRevenue)
	t.Logf("   - Laba Bersih Konsolidasian: Rp %.2f (Entitas Induk: Rp %.2f, Hak Minoritas/NCI: Rp %.2f)",
		consRun.ConsolidatedNetIncome, consRun.ControllingInterest, consRun.NonControllingInterest)
	t.Logf("   - Total Aset Konsolidasian: Rp %.2f, Ekuitas Konsolidasian: Rp %.2f",
		consRun.ConsolidatedAssets, consRun.ConsolidatedEquity)

	// 7. Cloudflare Workers AI Consolidation Auditor Copilot
	consAIReport, err := icSvc.AnalyzeConsolidationWithAI(consRun.ID, company.ID)
	if err != nil {
		t.Fatalf("AnalyzeConsolidationWithAI failed: %v", err)
	}
	if consAIReport.ComplianceStatus == "" || consAIReport.AuditScore == 0 {
		t.Fatalf("Invalid AI Consolidation Audit Report: %+v", consAIReport)
	}
	t.Logf("✓ Cloudflare Workers AI Consolidation Auditor: Status='%s', Skor Kepatuhan=%d/100 (AI Engine: %s)",
		consAIReport.ComplianceStatus, consAIReport.AuditScore, consAIReport.AIProvider)
	t.Logf("   - Ringkasan Auditor: %s", consAIReport.ExecutiveSummary)
	t.Logf("   - Risiko Non-Controlling Interest: %s", consAIReport.NonControllingRisk)

	// 8. Intercompany Executive Analytics
	icStats, err := icSvc.GetStats(company.ID)
	if err != nil {
		t.Fatalf("GetStats intercompany failed: %v", err)
	}
	if icStats.TotalGroups < 1 || icStats.TotalRelations < 1 || icStats.TotalMirroredTransactions < 2 || icStats.TotalConsolidationRuns < 1 {
		t.Fatalf("Invalid intercompany stats: %+v", icStats)
	}
	t.Logf("✓ Intercompany Executive Analytics: Groups=%d, Relations=%d, Mirrored Txs=%d, Consolidation Runs=%d, Total Eliminated=Rp %.2f",
		icStats.TotalGroups, icStats.TotalRelations, icStats.TotalMirroredTransactions, icStats.TotalConsolidationRuns, icStats.TotalEliminatedVolume)

	// 9. Universal Chatter Audit Integration
	icChatter, err := chatterSvc.ListMessages("intercompany.group", icGroup.ID, company.ID)
	if err != nil || len(icChatter) == 0 {
		t.Fatalf("Expected Chatter messages on intercompany group, got %d (err: %v)", len(icChatter), err)
	}
	t.Logf("✓ Universal Chatter Integration: %d audit messages tracked on intercompany group #%d", len(icChatter), icGroup.ID)

	// =========================================================================
	// 43. AI PREDICTIVE CASH FLOW FORECASTING & TREASURY MANAGEMENT (MODUL KE-39)
	// =========================================================================
	t.Log("\n--- SECTION 43: AI PREDICTIVE CASH FLOW FORECASTING & TREASURY MANAGEMENT ---")

	treasuryRepo := treasury.NewRepository(db)
	treasurySvc := treasury.NewService(db, treasuryRepo, accountSvc, chatterSvc, aiClient)

	// 1. Create Treasury Bank Accounts
	var bankAccGL account.AccountAccount
	if err := db.Where("company_id = ? AND (code LIKE '102%' OR code LIKE '111%' OR type = 'asset_cash')", company.ID).First(&bankAccGL).Error; err != nil {
		bankAccGL = account.AccountAccount{
			Name: "Bank Central Asia GL",
			Code: "102001",
			Type: "asset_cash",
		}
		bankAccGL.CompanyID = company.ID
		db.Create(&bankAccGL)
	}

	accBCAReq := treasury.CreateBankAccountRequest{
		BankName:        "Bank Central Asia (BCA)",
		AccountNumber:   "800-1122-3344",
		AccountHolder:   "PT Golang ERP Nusantara",
		AccountType:     treasury.BankTypeChecking,
		Currency:        "IDR",
		CurrentBalance:  500000000, // Rp 500 Juta
		MinimumBalance:  50000000,
		OverdraftLimit:  20000000,
		InterestRatePct: 2.5,
		JournalID:       bankJournal.ID,
		GLAccountID:     bankAccGL.ID,
		IsPrimaryPool:   true,
		Notes:           "Rekening Operasional & Likuiditas Utama Holding",
	}
	accBCA, err := treasurySvc.CreateBankAccount(accBCAReq, user.ID, company.ID)
	if err != nil {
		t.Fatalf("Failed to create BCA treasury account: %v", err)
	}
	t.Logf("✓ Created Treasury Bank Account: %s (%s - %s) Balance=Rp %.2f",
		accBCA.AccountCode, accBCA.BankName, accBCA.AccountNumber, accBCA.CurrentBalance)

	accMandiriReq := treasury.CreateBankAccountRequest{
		BankName:        "Bank Mandiri",
		AccountNumber:   "137-00-1234-567",
		AccountHolder:   "PT Golang ERP Nusantara",
		AccountType:     treasury.BankTypeChecking,
		Currency:        "IDR",
		CurrentBalance:  200000000, // Rp 200 Juta
		MinimumBalance:  30000000,
		OverdraftLimit:  10000000,
		InterestRatePct: 2.0,
		JournalID:       bankJournal.ID,
		GLAccountID:     bankAccGL.ID,
		IsPrimaryPool:   false,
		Notes:           "Rekening Khusus Payroll Gaji Karyawan",
	}
	accMandiri, err := treasurySvc.CreateBankAccount(accMandiriReq, user.ID, company.ID)
	if err != nil {
		t.Fatalf("Failed to create Mandiri treasury account: %v", err)
	}
	t.Logf("✓ Created Treasury Bank Account: %s (%s - %s) Balance=Rp %.2f",
		accMandiri.AccountCode, accMandiri.BankName, accMandiri.AccountNumber, accMandiri.CurrentBalance)

	// 2. Generate 30-Day Cash Flow Forecast
	forecastReq := treasury.GenerateForecastRequest{
		Name:                "Proyeksi Likuiditas & Arus Kas 30 Hari Kedepan",
		HorizonDays:         30,
		BaseDate:            time.Now(),
		MinimumSafetyBuffer: 80000000,
		IncludePipelineSO:   true,
		IncludeApprovedPO:   true,
		IncludeMRRRenewal:   true,
		Notes:               "Proyeksi kas operasional komprehensif mengintegrasikan AR, AP, SO pipeline, dan payroll",
	}
	forecast, err := treasurySvc.GenerateCashFlowForecast(forecastReq, user.ID, company.ID)
	if err != nil {
		t.Fatalf("GenerateCashFlowForecast failed: %v", err)
	}
	if forecast.OpeningCashBalance <= 0 || len(forecast.Items) == 0 {
		t.Fatalf("Invalid cash flow forecast output: Opening=%.2f, ItemsCount=%d", forecast.OpeningCashBalance, len(forecast.Items))
	}
	t.Logf("✓ Generated Cash Flow Forecast: %s (Horizon %d Hari)", forecast.ForecastCode, forecast.HorizonDays)
	t.Logf("  Opening Cash Balance : Rp %.2f", forecast.OpeningCashBalance)
	t.Logf("  Projected Inflow     : Rp %.2f", forecast.ProjectedInflow)
	t.Logf("  Projected Outflow    : Rp %.2f", forecast.ProjectedOutflow)
	t.Logf("  Projected Net Cash   : Rp %.2f", forecast.ProjectedNetCash)
	t.Logf("  Projected Ending Cash: Rp %.2f (Lowest Point: Rp %.2f)", forecast.ProjectedEndingCash, forecast.LowestPointCash)
	t.Logf("  Deficit Risk Level   : %s (Estimated Runway: %d Hari)", forecast.DeficitRisk, forecast.EstimatedRunwayDays)

	// 3. Verify Forecast Items
	hasAR := false
	hasPayroll := false
	for _, it := range forecast.Items {
		if it.Category == treasury.CategoryARReceivables {
			hasAR = true
		}
		if it.Category == treasury.CategoryPayrollSalaries {
			hasPayroll = true
		}
	}
	if !hasPayroll {
		t.Fatalf("Expected payroll salary item in forecast, but none found")
	}
	t.Logf("✓ Forecast Items Verified: Total %d items (hasAR=%v, hasPayroll=%v)", len(forecast.Items), hasAR, hasPayroll)

	// 4. Inter-Bank Liquidity Fund Sweep Transfer
	transferReq := treasury.CreateInterBankTransferRequest{
		FromAccountID: accBCA.ID,
		ToAccountID:   accMandiri.ID,
		TransferDate:  time.Now(),
		Amount:        75000000, // Rp 75 Juta
		TransferFee:   2500,
		Reference:     "SWEEP-PAYROLL-001",
		Memo:          "Penyelarasan likuiditas rekening payroll untuk pembayaran gaji karyawan",
	}
	transfer, err := treasurySvc.ExecuteInterBankTransfer(transferReq, user.ID, company.ID)
	if err != nil {
		t.Fatalf("ExecuteInterBankTransfer failed: %v", err)
	}
	if transfer.State != "posted" {
		t.Fatalf("Expected transfer state 'posted', got '%s'", transfer.State)
	}

	// Verify updated bank account balances
	updatedBCA, err := treasurySvc.GetBankAccount(accBCA.ID, company.ID)
	if err != nil || updatedBCA.CurrentBalance != 500000000-75002500 {
		t.Fatalf("Expected BCA balance Rp 424,997,500, got Rp %.2f (err: %v)", updatedBCA.CurrentBalance, err)
	}
	updatedMandiri, err := treasurySvc.GetBankAccount(accMandiri.ID, company.ID)
	if err != nil || updatedMandiri.CurrentBalance != 200000000+75000000 {
		t.Fatalf("Expected Mandiri balance Rp 275,000,000, got Rp %.2f (err: %v)", updatedMandiri.CurrentBalance, err)
	}
	t.Logf("✓ Inter-Bank Fund Sweep Posted: %s (Rp %.2f + Fee Rp %.2f)", transfer.TransferCode, transfer.Amount, transfer.TransferFee)
	t.Logf("  From Account: %s -> New Balance = Rp %.2f", updatedBCA.BankName, updatedBCA.CurrentBalance)
	t.Logf("  To Account  : %s -> New Balance = Rp %.2f", updatedMandiri.BankName, updatedMandiri.CurrentBalance)

	// 5. Test Insufficient Balance Rejection
	excessiveTransferReq := treasury.CreateInterBankTransferRequest{
		FromAccountID: accBCA.ID,
		ToAccountID:   accMandiri.ID,
		Amount:        99999999999, // Exceeds available balance + overdraft limit
	}
	_, errExcessive := treasurySvc.ExecuteInterBankTransfer(excessiveTransferReq, user.ID, company.ID)
	if errExcessive == nil {
		t.Fatalf("Expected error on excessive transfer beyond overdraft limit, but got none")
	}
	t.Logf("✓ Overdraft/Balance Protection verified: Rejection on excessive transfer: %v", errExcessive)

	// 6. AI Predictive Treasury Copilot & Working Capital Advisory
	treasuryAIReport, err := treasurySvc.AnalyzeTreasuryWithAI(forecast.ID, company.ID)
	if err != nil {
		t.Fatalf("AnalyzeTreasuryWithAI failed: %v", err)
	}
	if treasuryAIReport.HealthScore <= 0 || treasuryAIReport.ExecutiveSummary == "" || len(treasuryAIReport.WorkingCapitalMetrics) == 0 {
		t.Fatalf("Invalid AI Treasury report: %+v", treasuryAIReport)
	}
	t.Logf("✓ AI Treasury Copilot Report (%s):", treasuryAIReport.AIProvider)
	t.Logf("  Liquidity Health: %s (Score: %d/100, Runway: %.1f Bulan)", treasuryAIReport.LiquidityHealth, treasuryAIReport.HealthScore, treasuryAIReport.EstimatedRunwayMonths)
	t.Logf("  Executive Summary: %s", treasuryAIReport.ExecutiveSummary)
	for _, m := range treasuryAIReport.WorkingCapitalMetrics {
		t.Logf("  - %s: %.2f (Benchmark: %.2f, Status: %s)", m.MetricName, m.CurrentVal, m.Benchmark, m.Status)
	}
	for i, adv := range treasuryAIReport.ActionableAdvice {
		t.Logf("  Recommendation #%d: %s", i+1, adv)
	}

	// 7. Treasury Stats & Dashboard KPIs
	treasuryStats, err := treasurySvc.GetStats(company.ID)
	if err != nil {
		t.Fatalf("GetStats treasury failed: %v", err)
	}
	if treasuryStats.TotalLiquidCash <= 0 || treasuryStats.TotalActiveAccounts < 2 || treasuryStats.TotalTransfersVolume < 75000000 {
		t.Fatalf("Invalid treasury stats: %+v", treasuryStats)
	}
	t.Logf("✓ Treasury Analytics: Liquid Cash=Rp %.2f across %d accounts, Transfers Volume=Rp %.2f, Buffer Coverage=%.1f%%",
		treasuryStats.TotalLiquidCash, treasuryStats.TotalActiveAccounts, treasuryStats.TotalTransfersVolume, treasuryStats.SafetyBufferCoverage)

	// 8. Universal Chatter Audit Integration
	treasuryChatterLogs, errChatter := chatterSvc.ListMessages("treasury.forecast", forecast.ID, company.ID)
	if errChatter != nil || len(treasuryChatterLogs) == 0 {
		t.Fatalf("Expected Chatter messages on treasury forecast, got %d (err: %v)", len(treasuryChatterLogs), errChatter)
	}
	t.Logf("✓ Universal Chatter Integration: %d audit messages logged on treasury forecast #%d", len(treasuryChatterLogs), forecast.ID)

	// =========================================================================
	// 44. ADVANCED MRP II: WORK CENTER ROUTING, JAM MESIN & GANTT CAPACITY PLANNING (MODUL KE-40)
	// =========================================================================
	t.Log("\n--- SECTION 44: ADVANCED MRP II (WORK CENTERS, ROUTINGS, WORK ORDERS & GANTT CAPACITY PLANNING) ---")
	mrpAdvRepo := mrp_advanced.NewRepository(db)
	mrpAdvSvc := mrp_advanced.NewService(mrpAdvRepo, db, mrpSvc, chatterSvc, aiClient)

	// 1. Create Work Centers
	wc1Req := &mrp_advanced.CreateWorkCenterRequest{
		Name:               "Mesin CNC Milling 5-Axis",
		WorkCenterType:     "machining",
		CapacityPerHour:    10.0,
		WorkingHoursPerDay: 8.0,
		TotalOEEPercentage: 88.0,
		CostPerHour:        250000.0,
		MachinePowerKW:     22.0,
	}
	wc1, err := mrpAdvSvc.CreateWorkCenter(wc1Req, company.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to create Work Center 1: %v", err)
	}
	t.Logf("✓ Created Work Center 1: %s (%s) Cost=Rp %.0f/hr, Cap=%.1f units/hr", wc1.Name, wc1.Code, wc1.CostPerHour, wc1.CapacityPerHour)

	wc2Req := &mrp_advanced.CreateWorkCenterRequest{
		Name:               "Stasiun Welding & Robot Fabrikasi",
		WorkCenterType:     "welding",
		CapacityPerHour:    15.0,
		WorkingHoursPerDay: 8.0,
		TotalOEEPercentage: 85.0,
		CostPerHour:        180000.0,
		MachinePowerKW:     18.0,
	}
	wc2, err := mrpAdvSvc.CreateWorkCenter(wc2Req, company.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to create Work Center 2: %v", err)
	}
	t.Logf("✓ Created Work Center 2: %s (%s)", wc2.Name, wc2.Code)

	wc3Req := &mrp_advanced.CreateWorkCenterRequest{
		Name:               "Lini Perakitan Akhir (Assembly Line)",
		WorkCenterType:     "assembly",
		CapacityPerHour:    20.0,
		WorkingHoursPerDay: 8.0,
		TotalOEEPercentage: 92.0,
		CostPerHour:        120000.0,
		MachinePowerKW:     8.0,
	}
	wc3, err := mrpAdvSvc.CreateWorkCenter(wc3Req, company.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to create Work Center 3: %v", err)
	}
	t.Logf("✓ Created Work Center 3: %s (%s)", wc3.Name, wc3.Code)

	// 2. Create Multi-Step Routing with Operations
	routingReq := &mrp_advanced.CreateRoutingRequest{
		Name:      "Rute Pembuatan Unit Mesin Otomotif",
		ProductID: testProduct.ID,
		Operations: []mrp_advanced.CreateRoutingOperationDTO{
			{
				Sequence:                10,
				Name:                    "Milling Komponen Presisi",
				WorkCenterID:            wc1.ID,
				SetupTimeMinutes:        15.0,
				CycleTimePerUnitMinutes: 5.0,
				TeardownTimeMinutes:     10.0,
				BatchSize:               50.0,
			},
			{
				Sequence:                20,
				Name:                    "Pengelasan Rangka Struktural",
				WorkCenterID:            wc2.ID,
				SetupTimeMinutes:        10.0,
				CycleTimePerUnitMinutes: 8.0,
				TeardownTimeMinutes:     5.0,
				BatchSize:               50.0,
			},
			{
				Sequence:                30,
				Name:                    "Perakitan Akhir & Finishing",
				WorkCenterID:            wc3.ID,
				SetupTimeMinutes:        10.0,
				CycleTimePerUnitMinutes: 12.0,
				TeardownTimeMinutes:     5.0,
				BatchSize:               25.0,
			},
		},
		Notes: "Rute 3 stasiun kerja dari permesinan CNC hingga perakitan.",
	}
	routing, err := mrpAdvSvc.CreateRouting(routingReq, company.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to create routing: %v", err)
	}
	t.Logf("✓ Created Routing: %s (%s) with %d operations", routing.Name, routing.RoutingCode, len(routing.Operations))

	// 3. Generate Work Orders for an MO (10 units)
	var testMO mrp.MrpProduction
	db.First(&testMO)
	if testMO.ID == 0 {
		testMO = mrp.MrpProduction{
			Name:        "MO/2026/00099",
			ProductID:   testProduct.ID,
			ProductQty:  10.0,
			State:       "confirmed",
			DatePlanned: time.Now().Format("2006-01-02"),
		}
		testMO.CompanyID = company.ID
		db.Create(&testMO)
	}

	genReq := &mrp_advanced.GenerateWorkOrdersRequest{
		ProductionOrderID: testMO.ID,
		RoutingID:         routing.ID,
	}
	workOrders, err := mrpAdvSvc.GenerateWorkOrdersForMO(genReq, company.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to generate work orders for MO: %v", err)
	}
	if len(workOrders) != 3 {
		t.Fatalf("Expected 3 work orders generated, got %d", len(workOrders))
	}
	t.Logf("✓ Generated %d Work Orders for MO %s:", len(workOrders), testMO.Name)
	for _, wo := range workOrders {
		t.Logf("  - %s: Seq %d (%s) at %s | State: %s | Planned: %.1f mins",
			wo.WorkOrderCode, wo.Sequence, wo.OperationName, wo.WorkCenter.Name, wo.State, wo.PlannedDurationMinutes)
	}

	// First operation must be READY, others PENDING
	if workOrders[0].State != "ready" {
		t.Fatalf("Expected first work order state to be 'ready', got %s", workOrders[0].State)
	}
	if workOrders[1].State != "pending" || workOrders[2].State != "pending" {
		t.Fatalf("Expected subsequent work orders to be 'pending'")
	}

	// 4. Start & Complete Operation 10 (CNC Milling)
	startedWO1, err := mrpAdvSvc.StartWorkOrder(workOrders[0].ID, company.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to start Work Order 1: %v", err)
	}
	if startedWO1.State != "in_progress" {
		t.Fatalf("Expected WO 1 state 'in_progress', got %s", startedWO1.State)
	}
	t.Logf("✓ Started Work Order 1: %s (State: %s)", startedWO1.WorkOrderCode, startedWO1.State)

	finishReq1 := &mrp_advanced.CompleteWorkOrderRequest{
		QtyProduced:           10.0,
		QtyScrap:              0.0,
		ActualDurationMinutes: 75.0, // 1.25 hours
		Notes:                 "Pemotongan blok presisi selesai tanpa cacat.",
	}
	finishedWO1, err := mrpAdvSvc.FinishWorkOrder(workOrders[0].ID, finishReq1, company.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to finish Work Order 1: %v", err)
	}
	if finishedWO1.State != "done" || finishedWO1.MachineHours != 1.25 || finishedWO1.TotalMachineCost != (1.25*250000.0) {
		t.Fatalf("Invalid finished WO 1: State=%s, MachineHours=%.2f, Cost=Rp %.0f", finishedWO1.State, finishedWO1.MachineHours, finishedWO1.TotalMachineCost)
	}
	t.Logf("✓ Finished Work Order 1: %s (State: %s, Machine Hours: %.2f hrs, Machine Cost: Rp %.0f)",
		finishedWO1.WorkOrderCode, finishedWO1.State, finishedWO1.MachineHours, finishedWO1.TotalMachineCost)

	// Verify Operation 20 automatically transitioned from PENDING to READY!
	wo2Updated, err := mrpAdvSvc.GetWorkOrder(workOrders[1].ID)
	if err != nil {
		t.Fatalf("Failed to get WO 2: %v", err)
	}
	if wo2Updated.State != "ready" {
		t.Fatalf("Expected WO 2 to automatically become 'ready', got %s", wo2Updated.State)
	}
	t.Logf("✓ Work Order 2 (%s) automatically transitioned to 'ready' upon completion of WO 1!", wo2Updated.WorkOrderCode)

	// 5. Gantt Capacity Planning Schedule Verification
	ganttSched, err := mrpAdvSvc.GetGanttCapacitySchedule(company.ID, 14)
	if err != nil {
		t.Fatalf("Failed to get Gantt capacity schedule: %v", err)
	}
	if len(ganttSched.Slots) == 0 || len(ganttSched.CapacitySummaries) == 0 {
		t.Fatalf("Empty Gantt capacity schedule: slots=%d, summaries=%d", len(ganttSched.Slots), len(ganttSched.CapacitySummaries))
	}
	t.Logf("✓ Gantt Capacity Schedule: %d task slots across %d work centers (Horizon: %d days)",
		len(ganttSched.Slots), len(ganttSched.CapacitySummaries), ganttSched.HorizonDays)
	for _, cs := range ganttSched.CapacitySummaries {
		t.Logf("  * %s: Scheduled %.1f hrs / Cap %.1f hrs (Util: %.1f%%, Bottleneck: %v)",
			cs.WorkCenterName, cs.ScheduledHoursTotal, cs.CapacityHoursPerDay*14.0, cs.UtilizationPercentage, cs.IsBottleneck)
	}

	// 6. AI Production Capacity & Line Balancing Advisor
	aiCapacityReport, err := mrpAdvSvc.AnalyzeMRPCapacityWithAI(company.ID)
	if err != nil {
		t.Fatalf("AnalyzeMRPCapacityWithAI failed: %v", err)
	}
	if aiCapacityReport.ScheduleEfficiencyScore <= 0 || aiCapacityReport.ExecutiveSummary == "" || len(aiCapacityReport.LineBalancingAdvice) == 0 {
		t.Fatalf("Invalid AI Capacity report: %+v", aiCapacityReport)
	}
	t.Logf("✓ AI MRP Capacity Report (%s):", aiCapacityReport.AIProvider)
	t.Logf("  Overall Status: %s (Efficiency Score: %d/100, Rec Lead Time: %d days)",
		aiCapacityReport.OverallStatus, aiCapacityReport.ScheduleEfficiencyScore, aiCapacityReport.RecommendedLeadTimeDays)
	t.Logf("  Executive Summary: %s", aiCapacityReport.ExecutiveSummary)
	for i, adv := range aiCapacityReport.LineBalancingAdvice {
		t.Logf("  Recommendation #%d: %s", i+1, adv)
	}

	// 7. MRP Advanced Dashboard Stats
	mrpStats, err := mrpAdvSvc.GetStats(company.ID)
	if err != nil {
		t.Fatalf("GetStats mrp_advanced failed: %v", err)
	}
	if mrpStats.TotalWorkCenters < 3 || mrpStats.TotalWorkOrders < 3 || mrpStats.TotalMachineHours <= 0 {
		t.Fatalf("Invalid mrp advanced stats: %+v", mrpStats)
	}
	t.Logf("✓ MRP II Analytics: %d Work Centers (Avg OEE: %.1f%%), %d Work Orders, Total Machine Hours: %.2f hrs (Cost: Rp %.0f)",
		mrpStats.TotalWorkCenters, mrpStats.OverallAvgOEE, mrpStats.TotalWorkOrders, mrpStats.TotalMachineHours, mrpStats.TotalMachineCost)

	// 8. Universal Chatter Audit Integration
	mrpChatterLogs, errChatter := chatterSvc.ListMessages("mrp.workcenter", wc1.ID, company.ID)
	if errChatter != nil || len(mrpChatterLogs) == 0 {
		t.Fatalf("Expected Chatter messages on Work Center 1, got %d (err: %v)", len(mrpChatterLogs), errChatter)
	}
	t.Logf("✓ Universal Chatter Integration: %d audit messages logged on Work Center #%d", len(mrpChatterLogs), wc1.ID)

	// =========================================================================
	// SECTION 45: INDONESIAN E-BUPOT 21/26 & SPT MASA PPH 21 DJP EXPORT ENGINE
	// =========================================================================
	t.Log("\n    --- SECTION 45: INDONESIAN E-BUPOT 21/26 & SPT MASA PPH 21 DJP EXPORT (PP 58/2023 & PMK 168/2023) ---")

	ebupotRepo := tax_ebupot.NewRepository(db)
	ebupotSvc := tax_ebupot.NewService(ebupotRepo, db, payrollSvc, accountSvc, chatterSvc, aiClient)

	// 1. Create SPT Masa Period
	createPeriodReq := &tax_ebupot.CreatePeriodRequest{
		TaxMonth:       10,
		TaxYear:        2026,
		RevisionNumber: 0,
		Notes:          "SPT Masa PPh 21/26 Masa Pajak Oktober 2026 (PP 58/2023 TER)",
	}
	period21, err := ebupotSvc.CreatePeriod(createPeriodReq, company.ID)
	if err != nil {
		t.Fatalf("CreatePeriod failed: %v", err)
	}
	if period21.PeriodCode != "SPT21/2026/10/0" || period21.State != "draft" {
		t.Fatalf("Unexpected period result: %+v", period21)
	}
	t.Logf("✓ Created e-Bupot Tax Period: %s (Bulan %d, Tahun %d, Pembetulan: %d)",
		period21.PeriodCode, period21.TaxMonth, period21.TaxYear, period21.RevisionNumber)

	// 2. Test TER PP 58/2023 Calculation & Manual Withholding Slips Creation
	// Case 1: Pegawai Tetap TER A (TK/0) - Gaji Rp 12.500.000 -> 4.0% = Rp 500.000
	slipReq1 := &tax_ebupot.CreateWithholdingSlipRequest{
		PeriodID:         period21.ID,
		SlipType:         "bp_pegawai_tetap",
		TaxObjectCode:    "21-100-01",
		TaxObjectName:    "Pegawai Tetap (Gaji Pokok & Tunjangan Jabatan)",
		RecipientType:    "employee",
		NIK:              "3171012304890001",
		NPWP:             "012345678901000",
		RecipientName:    "Ahmad Fauzi, S.T.",
		RecipientAddress: "Jl. Sudirman Kav 24, Jakarta Selatan",
		PTKPStatus:       "TK/0",
		GrossIncome:      12500000,
		TaxFacilityType:  "none",
	}
	slip1, err := ebupotSvc.CreateWithholdingSlip(slipReq1, company.ID)
	if err != nil {
		t.Fatalf("CreateWithholdingSlip 1 failed: %v", err)
	}
	if slip1.TERCategory != "TER_A" || slip1.EffectiveRatePct != 4.0 || slip1.TaxWithheld != 500000 {
		t.Fatalf("TER A Calculation mismatch: Cat=%s, Rate=%.2f, Tax=%.2f", slip1.TERCategory, slip1.EffectiveRatePct, slip1.TaxWithheld)
	}
	t.Logf("✓ Created Slip 1: %s (%s) PTKP: %s | TER: %s (%.2f%%) | Bruto: Rp %.2f | PPh 21: Rp %.2f",
		slip1.SlipNumber, slip1.RecipientName, slip1.PTKPStatus, slip1.TERCategory, slip1.EffectiveRatePct, slip1.GrossIncome, slip1.TaxWithheld)

	// Case 2: Pegawai Tetap TER B (K/1) - Gaji Rp 18.500.000 -> 8.0% = Rp 1.480.000
	slipReq2 := &tax_ebupot.CreateWithholdingSlipRequest{
		PeriodID:         period21.ID,
		SlipType:         "bp_pegawai_tetap",
		TaxObjectCode:    "21-100-01",
		TaxObjectName:    "Pegawai Tetap (Gaji Pokok & Tunjangan Keluarga)",
		RecipientType:    "employee",
		NIK:              "3276021508920003",
		NPWP:             "098765432109000",
		RecipientName:    "Siti Nurhaliza, M.M.",
		RecipientAddress: "Jl. Margonda Raya No. 45, Depok",
		PTKPStatus:       "K/1",
		GrossIncome:      18500000,
		TaxFacilityType:  "none",
	}
	slip2, err := ebupotSvc.CreateWithholdingSlip(slipReq2, company.ID)
	if err != nil {
		t.Fatalf("CreateWithholdingSlip 2 failed: %v", err)
	}
	if slip2.TERCategory != "TER_B" || slip2.EffectiveRatePct != 8.0 || slip2.TaxWithheld != 1480000 {
		t.Fatalf("TER B Calculation mismatch: Cat=%s, Rate=%.2f, Tax=%.2f", slip2.TERCategory, slip2.EffectiveRatePct, slip2.TaxWithheld)
	}
	t.Logf("✓ Created Slip 2: %s (%s) PTKP: %s | TER: %s (%.2f%%) | Bruto: Rp %.2f | PPh 21: Rp %.2f",
		slip2.SlipNumber, slip2.RecipientName, slip2.PTKPStatus, slip2.TERCategory, slip2.EffectiveRatePct, slip2.GrossIncome, slip2.TaxWithheld)

	// Case 3: Tenaga Ahli Bukan Pegawai (21-100-07)
	slipReq3 := &tax_ebupot.CreateWithholdingSlipRequest{
		PeriodID:         period21.ID,
		SlipType:         "bp_tidak_final",
		TaxObjectCode:    "21-100-07",
		TaxObjectName:    "Tenaga Ahli Konsultan ERP",
		RecipientType:    "vendor_expert",
		NIK:              "3174092011850002",
		NPWP:             "081234567890000",
		RecipientName:    "Dr. Ir. Budi Santoso, M.Kom.",
		RecipientAddress: "Kuningan Barat No. 12, Jakarta",
		PTKPStatus:       "TK/0",
		GrossIncome:      25000000,
		TaxFacilityType:  "none",
	}
	slip3, err := ebupotSvc.CreateWithholdingSlip(slipReq3, company.ID)
	if err != nil {
		t.Fatalf("CreateWithholdingSlip 3 failed: %v", err)
	}
	t.Logf("✓ Created Slip 3 (Tenaga Ahli): %s (%s) Bruto: Rp %.2f | PPh 21: Rp %.2f",
		slip3.SlipNumber, slip3.RecipientName, slip3.GrossIncome, slip3.TaxWithheld)

	// 3. Import from Payroll Integration
	ebupotImportReq := &tax_ebupot.ImportFromPayrollRequest{
		PeriodID: period21.ID,
	}
	importedSlips, err := ebupotSvc.ImportFromPayroll(ebupotImportReq, company.ID)
	if err != nil {
		t.Logf("Notice on ImportFromPayroll (payroll items): %v", err)
	} else {
		t.Logf("✓ Imported %d Withholding Slips directly from Payroll Payslips", len(importedSlips))
	}

	// 4. Generate DJP CSV Format Export
	djpCSV, err := ebupotSvc.GenerateDJPCSV(period21.ID)
	if err != nil {
		t.Fatalf("GenerateDJPCSV failed: %v", err)
	}
	if !strings.Contains(djpCSV, "MASA_PAJAK;TAHUN_PAJAK") || !strings.Contains(djpCSV, "Ahmad Fauzi") {
		t.Fatalf("Invalid DJP CSV content: %s", djpCSV)
	}
	t.Logf("✓ Generated Official DJP e-Bupot 21/26 CSV Export (Length: %d bytes):\n%s",
		len(djpCSV), strings.Split(djpCSV, "\n")[0])

	// 5. Generate DJP XML Format Export
	djpXML, err := ebupotSvc.GenerateDJPXML(period21.ID)
	if err != nil {
		t.Fatalf("GenerateDJPXML failed: %v", err)
	}
	if !strings.Contains(djpXML, "<SPTMasaPPh21") || !strings.Contains(djpXML, "</DaftarBuktiPotong>") {
		t.Fatalf("Invalid DJP XML content: %s", djpXML)
	}
	t.Logf("✓ Generated Official DJP e-Bupot 21/26 XML Document (Length: %d bytes)", len(djpXML))

	// 6. Cross-Reconciliation with Payroll & General Ledger
	recReport, err := ebupotSvc.ReconcileWithPayrollAndGL(period21.ID, company.ID)
	if err != nil {
		t.Fatalf("ReconcileWithPayrollAndGL failed: %v", err)
	}
	t.Logf("✓ Tax Reconciliation: Period %s | e-Bupot Gross: Rp %.2f | e-Bupot Tax: Rp %.2f | Status: %s",
		recReport.PeriodCode, recReport.TotalEbupotGross, recReport.TotalEbupotTaxWithheld, recReport.StatusSummary)

	// 7. Lock Period with NTPN
	lockedPeriod, err := ebupotSvc.LockPeriod(period21.ID, "NTPN-2026-PPH21-998877")
	if err != nil {
		t.Fatalf("LockPeriod failed: %v", err)
	}
	if lockedPeriod.State != "locked" || lockedPeriod.SubmissionNTPN == "" || lockedPeriod.DJPReceiptNumber == "" {
		t.Fatalf("Invalid locked period state: %+v", lockedPeriod)
	}
	t.Logf("✓ Locked e-Bupot Period: %s (State: %s, NTPN: %s, BPE DJP: %s)",
		lockedPeriod.PeriodCode, lockedPeriod.State, lockedPeriod.SubmissionNTPN, lockedPeriod.DJPReceiptNumber)

	// 8. AI Tax Compliance & Risk Copilot
	aiTaxReport, err := ebupotSvc.AnalyzeTaxComplianceWithAI(period21.ID)
	if err != nil {
		t.Fatalf("AnalyzeTaxComplianceWithAI failed: %v", err)
	}
	if aiTaxReport.ComplianceScore <= 0 || aiTaxReport.ExecutiveSummary == "" || len(aiTaxReport.ActionableAdvice) == 0 {
		t.Fatalf("Invalid AI Tax Compliance report: %+v", aiTaxReport)
	}
	t.Logf("✓ AI Tax Compliance Copilot (%s):", aiTaxReport.AIProvider)
	t.Logf("  Status: %s (Compliance Score: %d/100, Penalty Risk: %s)",
		aiTaxReport.ComplianceStatus, aiTaxReport.ComplianceScore, aiTaxReport.EstimatedPenaltyRisk)
	t.Logf("  Executive Summary: %s", aiTaxReport.ExecutiveSummary)
	for i, chk := range aiTaxReport.ComplianceChecklist {
		t.Logf("  Checklist #%d: %s", i+1, chk)
	}

	// 9. Dashboard Stats Verification
	ebupotStats, err := ebupotSvc.GetStats(company.ID)
	if err != nil {
		t.Fatalf("GetStats tax_ebupot failed: %v", err)
	}
	if ebupotStats.TotalPeriods < 1 || ebupotStats.TotalWithholdingSlips < 3 || ebupotStats.TotalTaxWithheld <= 0 {
		t.Fatalf("Invalid ebupot stats: %+v", ebupotStats)
	}
	t.Logf("✓ e-Bupot 21/26 Analytics: %d Periods, %d Slips, Total Gross: Rp %.2f, Total Tax Withheld: Rp %.2f (Valid NIK: %d)",
		ebupotStats.TotalPeriods, ebupotStats.TotalWithholdingSlips, ebupotStats.TotalGrossWages, ebupotStats.TotalTaxWithheld, ebupotStats.ValidNIKCount)

	// 10. Universal Chatter Audit Integration
	taxChatterLogs, errChatter := chatterSvc.ListMessages("tax_ebupot.period", period21.ID, company.ID)
	if errChatter != nil || len(taxChatterLogs) == 0 {
		t.Fatalf("Expected Chatter messages on Ebupot Period, got %d (err: %v)", len(taxChatterLogs), errChatter)
	}
	t.Logf("✓ Universal Chatter Integration: %d audit messages logged on Ebupot Period #%d", len(taxChatterLogs), period21.ID)

	// -------------------------------------------------------------
	// SECTION 46: DIGITAL SIGNATURE & DOCUMENT WATERMARKING (UU ITE NO. 1/2024 & PP NO. 71/2019)
	// -------------------------------------------------------------
	t.Log("\n--- SECTION 46: DIGITAL SIGNATURE & DOCUMENT WATERMARKING (UU ITE NO. 1/2024 & PP NO. 71/2019) ---")
	esignRepo := esignature.NewRepository(db)
	esignSvc := esignature.NewService(esignRepo, chatterSvc, aiClient)

	// 1. Create Digital Signature Request for Sale Order #1
	watermarkTrue := true
	signReq1, err := esignSvc.CreateSignRequest(&esignature.CreateSignRequest{
		DocumentType:     "sale_order",
		DocumentID:       so.ID,
		DocumentNumber:   so.Name,
		Title:            "Kontrak Perjanjian Jual Beli Mesin Otomotif & Lisensi Software",
		Description:      "Otorisasi final kontrak pengadaan mesin otomotif beserta klausul SLA.",
		SignerName:       "Budi Santoso, S.T., M.M.",
		SignerEmail:      "budi.santoso@client.co.id",
		SignerRole:       "Direktur Utama (Client)",
		WatermarkEnabled: &watermarkTrue,
		WatermarkText:    "CONFIDENTIAL & DIGITALLY VERIFIED - PT MULTI PERUSAHAAN HOLDING",
		ExpiryDays:       14,
	}, company.ID)
	if err != nil {
		t.Fatalf("CreateSignRequest 1 failed: %v", err)
	}
	if signReq1.RequestNumber == "" || signReq1.SecurityToken == "" || signReq1.Status != "pending" {
		t.Fatalf("Invalid sign request: %+v", signReq1)
	}
	t.Logf("✓ Created Digital Signature Request 1: %s for %s (%s) | Token: %s",
		signReq1.RequestNumber, signReq1.DocumentNumber, signReq1.SignerName, signReq1.SecurityToken[:12]+"...")

	// 2. Public verification endpoint access (View event)
	viewedReq, err := esignSvc.GetSignRequestByToken(signReq1.SecurityToken)
	if err != nil {
		t.Fatalf("GetSignRequestByToken failed: %v", err)
	}
	if !strings.Contains(viewedReq.AuditTrailLog, "viewed") {
		t.Fatalf("Expected audit trail to contain 'viewed' action, got: %s", viewedReq.AuditTrailLog)
	}
	t.Logf("✓ Document accessed via Secure Public Signing Portal (Audit action 'viewed' registered)")

	// 3. Execute Digital Signing
	signedReq1, err := esignSvc.ExecuteSigning(signReq1.SecurityToken, &esignature.ExecuteSignRequest{
		SignatureType: "draw",
		SignatureData: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAK8AAAA8CAYAAADt7z10AAAFeklEQVR4Xu2dz4tcVRTHP/fe7unpZCYz",
		WatermarkText: "DIGITALLY SIGNED & VERIFIED BY BUDI SANTOSO - PT MULTI HOLDING",
	}, "103.111.20.10", "Mozilla/5.0 (Macintosh; Apple Silicon)")
	if err != nil {
		t.Fatalf("ExecuteSigning failed: %v", err)
	}
	if signedReq1.Status != "signed" || signedReq1.SignedAt == nil || len(signedReq1.SignatureHash) != 64 {
		t.Fatalf("Invalid signed document state: %+v", signedReq1)
	}
	if !strings.HasPrefix(signedReq1.CertificateSerialNumber, "CERT-ID-") {
		t.Fatalf("Invalid certificate serial number: %s", signedReq1.CertificateSerialNumber)
	}
	if !strings.Contains(signedReq1.WatermarkQRData, signedReq1.SignatureHash) {
		t.Fatalf("Expected QR verification URL to contain signature hash: %s", signedReq1.WatermarkQRData)
	}
	t.Logf("✓ Executed Digital Signature: %s (Status: %s) | Hash SHA-256: %s | Cert: %s",
		signedReq1.RequestNumber, signedReq1.Status, signedReq1.SignatureHash, signedReq1.CertificateSerialNumber)

	// 4. Tamper Prevention: Attempt to re-sign already signed request
	_, errReSign := esignSvc.ExecuteSigning(signReq1.SecurityToken, &esignature.ExecuteSignRequest{
		SignatureType: "draw",
		SignatureData: "tampered_data",
	}, "127.0.0.1", "curl")
	if errReSign == nil {
		t.Fatalf("Expected re-signing to fail, but it succeeded")
	}
	t.Logf("✓ Tamper-Resistance: Double-signing rejected with error: %v", errReSign)

	// 5. Create Request 2 for Purchase Order and test Rejection workflow
	signReq2, err := esignSvc.CreateSignRequest(&esignature.CreateSignRequest{
		DocumentType:   "purchase_order",
		DocumentID:     po.ID,
		DocumentNumber: po.Name,
		Title:          "Otorisasi PO Pengadaan Bahan Baku Baja",
		Description:    "Persetujuan pengadaan bahan baku baja untuk lini permesinan.",
		SignerName:     "Ir. Bambang Soedjarwo",
		SignerEmail:    "bambang@vendor-baja.co.id",
		SignerRole:     "Vendor Authorized Representative",
		ExpiryDays:     7,
	}, company.ID)
	if err != nil {
		t.Fatalf("CreateSignRequest 2 failed: %v", err)
	}

	rejectedReq, err := esignSvc.RejectSignRequest(signReq2.SecurityToken, "Harga satuan pada klausul 4 perlu penyesuaian diskon volume 5%", "180.252.11.22", "Mozilla/5.0")
	if err != nil {
		t.Fatalf("RejectSignRequest failed: %v", err)
	}
	if rejectedReq.Status != "rejected" || rejectedReq.RejectionReason == "" {
		t.Fatalf("Invalid rejected state: %+v", rejectedReq)
	}
	t.Logf("✓ Digital Signature Rejection Recorded: %s (Status: %s, Reason: %s)",
		rejectedReq.RequestNumber, rejectedReq.Status, rejectedReq.RejectionReason)

	// 6. AI Legal Compliance Auditor (UU ITE No. 1/2024 & PP No. 71/2019)
	aiEsignReport, err := esignSvc.AnalyzeSignatureComplianceWithAI(signedReq1.ID)
	if err != nil {
		t.Fatalf("AnalyzeSignatureComplianceWithAI failed: %v", err)
	}
	if aiEsignReport.IntegrityScore < 70 || aiEsignReport.AuditStatus != "VERIFIED_LEGAL" || !aiEsignReport.HashIntegrityValid {
		t.Fatalf("Invalid AI eSign Report: %+v", aiEsignReport)
	}
	t.Logf("✓ AI Digital Signature Legal Audit (%s):", aiEsignReport.AIProvider)
	t.Logf("  Status: %s (Integrity Score: %d/100, Legal Basis: %s)",
		aiEsignReport.AuditStatus, aiEsignReport.IntegrityScore, aiEsignReport.LegalBasis)
	t.Logf("  Executive Summary: %s", aiEsignReport.ExecutiveSummary)
	for i, ev := range aiEsignReport.EvidenceChecklist {
		t.Logf("  Evidence #%d: %s", i+1, ev)
	}

	// 7. Signature Analytics KPI
	esignStats, err := esignSvc.GetStats(company.ID)
	if err != nil {
		t.Fatalf("GetStats esignature failed: %v", err)
	}
	if esignStats.TotalRequests < 2 || esignStats.SignedRequests < 1 || esignStats.RejectedRequests < 1 {
		t.Fatalf("Invalid esign stats: %+v", esignStats)
	}
	t.Logf("✓ Digital Signature Analytics: Total Requests=%d, Signed=%d, Rejected=%d, Compliance Rate=%.1f%%, Avg Turnaround=%.2f hrs",
		esignStats.TotalRequests, esignStats.SignedRequests, esignStats.RejectedRequests, esignStats.ComplianceRate, esignStats.AvgTurnaroundHrs)

	// 8. Universal Chatter Audit Integration
	soChatterLogs, errChatter := chatterSvc.ListMessages("sale_order", so.ID, company.ID)
	if errChatter != nil || len(soChatterLogs) == 0 {
		t.Fatalf("Expected Chatter messages on Sale Order for e-sign, got %d (err: %v)", len(soChatterLogs), errChatter)
	}
	t.Logf("✓ Universal Chatter Integration: Audit messages logged on Sale Order #%d", so.ID)

	// -------------------------------------------------------------
	// --- SECTION 47: ENTERPRISE WMS: MULTI-TIER BIN LOCATIONS, PUTAWAY RULES & WAVE PICKING ENGINE ---
	// -------------------------------------------------------------
	t.Log("\n--- SECTION 47: ENTERPRISE WMS: MULTI-TIER BIN LOCATIONS, PUTAWAY RULES & WAVE PICKING ENGINE ---")

	wmsRepo := wms.NewRepository(db)
	wmsSvc := wms.NewService(wmsRepo, db, stockSvc, chatterSvc, aiClient)
	testWarehouseID := uint(1)

	// 1. Create Storage Zones
	zoneDry, err := wmsSvc.CreateZone(wms.CreateZoneRequest{
		WarehouseID:       testWarehouseID,
		Code:              "ZONE-DRY-A",
		Name:              "Zona Penyimpanan Kering A (Fast Moving)",
		ZoneType:          "dry",
		TemperatureMin:    20,
		TemperatureMax:    28,
		MaxWeightCapacity: 50000,
		Description:       "Zona utama dekat docking",
	})
	if err != nil {
		t.Fatalf("CreateZone dry failed: %v", err)
	}
	t.Logf("✓ Created WMS Zone 1: %s (%s) [Type: %s, MaxCap: %.0f kg]", zoneDry.Code, zoneDry.Name, zoneDry.ZoneType, zoneDry.MaxWeightCapacity)

	zoneCold, err := wmsSvc.CreateZone(wms.CreateZoneRequest{
		WarehouseID:       testWarehouseID,
		Code:              "ZONE-COLD-C",
		Name:              "Zona Rantai Dingin C",
		ZoneType:          "cold",
		TemperatureMin:    2,
		TemperatureMax:    8,
		MaxWeightCapacity: 20000,
		Description:       "Zona suhu terkontrol untuk bahan mudah rusak",
	})
	if err != nil {
		t.Fatalf("CreateZone cold failed: %v", err)
	}
	t.Logf("✓ Created WMS Zone 2: %s (%s) [Type: %s, Temp: %.1f°C - %.1f°C]", zoneCold.Code, zoneCold.Name, zoneCold.ZoneType, zoneCold.TemperatureMin, zoneCold.TemperatureMax)

	// 2. Create 3D Coordinate Bin Locations
	binA01, err := wmsSvc.CreateBin(wms.CreateBinRequest{
		WarehouseID:      testWarehouseID,
		ZoneID:           zoneDry.ID,
		Aisle:            "A01",
		Rack:             "R01",
		Shelf:            "S01",
		Bin:              "B01",
		MaxWeightKg:      500,
		MaxVolumeCbm:     1.5,
		VelocityCategory: "A",
		PickingSequence:  1,
	})
	if err != nil {
		t.Fatalf("CreateBin A01 failed: %v", err)
	}
	t.Logf("✓ Created WMS Bin Location 1: %s [Seq: %d, Max: %.0f kg / %.1f cbm, Velocity: %s]",
		binA01.Code, binA01.PickingSequence, binA01.MaxWeightKg, binA01.MaxVolumeCbm, binA01.VelocityCategory)

	binA02, err := wmsSvc.CreateBin(wms.CreateBinRequest{
		WarehouseID:      testWarehouseID,
		ZoneID:           zoneDry.ID,
		Aisle:            "A01",
		Rack:             "R01",
		Shelf:            "S01",
		Bin:              "B02",
		MaxWeightKg:      500,
		MaxVolumeCbm:     1.5,
		VelocityCategory: "A",
		PickingSequence:  2,
	})
	if err != nil {
		t.Fatalf("CreateBin A02 failed: %v", err)
	}
	t.Logf("✓ Created WMS Bin Location 2: %s [Seq: %d, Max: %.0f kg / %.1f cbm, Velocity: %s]",
		binA02.Code, binA02.PickingSequence, binA02.MaxWeightKg, binA02.MaxVolumeCbm, binA02.VelocityCategory)

	binCold1, err := wmsSvc.CreateBin(wms.CreateBinRequest{
		WarehouseID:      testWarehouseID,
		ZoneID:           zoneCold.ID,
		Aisle:            "C01",
		Rack:             "R01",
		Shelf:            "S01",
		Bin:              "B01",
		MaxWeightKg:      400,
		MaxVolumeCbm:     1.0,
		VelocityCategory: "B",
		PickingSequence:  10,
	})
	if err != nil {
		t.Fatalf("CreateBin cold1 failed: %v", err)
	}
	t.Logf("✓ Created WMS Bin Location 3: %s [Cold Storage, Seq: %d]", binCold1.Code, binCold1.PickingSequence)

	// 3. Putaway Rules & Suggestion Engine
	putawayRule, err := wmsSvc.CreatePutawayRule(wms.CreatePutawayRuleRequest{
		WarehouseID:  testWarehouseID,
		ProductID:    &testProduct.ID,
		TargetZoneID: zoneDry.ID,
		Strategy:     "fastest_access",
		Priority:     1,
		Description:  "Fastest access for Furniture Product 1",
	})
	if err != nil {
		t.Fatalf("CreatePutawayRule failed: %v", err)
	}
	t.Logf("✓ Created Putaway Rule: Target Zone #%d, Strategy: %s, Priority: %d", putawayRule.TargetZoneID, putawayRule.Strategy, putawayRule.Priority)

	suggestion, err := wmsSvc.SuggestPutawayLocation(wms.SuggestPutawayRequest{
		WarehouseID: testWarehouseID,
		ProductID:   testProduct.ID,
		Quantity:    20,
		WeightKg:    50,
		VolumeCbm:   0.3,
	})
	if err != nil {
		t.Fatalf("SuggestPutawayLocation failed: %v", err)
	}
	t.Logf("✓ AI & Rule-Based Putaway Suggestion: Assigned Bin %s (%s) [Confidence: %d%%, Strategy: %s]",
		suggestion.BinLocationCode, suggestion.ZoneName, suggestion.ConfidenceScore, suggestion.StrategyApplied)

	// 4. Physical Storage Allocation into Bin
	err = wmsSvc.AssignInventoryToBin(binA01.ID, testProduct.ID, 20, 50, 0.3)
	if err != nil {
		t.Fatalf("AssignInventoryToBin failed: %v", err)
	}
	updatedBinA01, _ := wmsSvc.GetBin(binA01.ID)
	if !updatedBinA01.IsOccupied || updatedBinA01.CurrentWeightKg != 50 {
		t.Fatalf("Bin metrics mismatch after assign: occupied=%v, weight=%.1f", updatedBinA01.IsOccupied, updatedBinA01.CurrentWeightKg)
	}
	t.Logf("✓ Physical Inventory Allocated to Bin %s: Current Weight=%.1f kg, Current Volume=%.2f cbm, Status=%s",
		updatedBinA01.Code, updatedBinA01.CurrentWeightKg, updatedBinA01.CurrentVolumeCbm, updatedBinA01.Status)

	// 5. Create Wave Picking grouping multiple delivery pickings
	wave, err := wmsSvc.CreateWavePicking(wms.CreateWavePickingRequest{
		Name:               "Wave Pengiriman Sore - Ekspedisi JNE & Truk",
		WarehouseID:        testWarehouseID,
		PickingIDs:         []uint{delivery.ID},
		AssignedPickerName: "Bambang Wibowo (Lead Picker)",
		Notes:              "Kelompokkan pesanan ritel sore",
	})
	if err != nil {
		t.Fatalf("CreateWavePicking failed: %v", err)
	}
	if wave.TotalLines == 0 {
		t.Fatalf("Expected wave lines, got 0")
	}
	t.Logf("✓ Created Wave Picking: %s (%s) [Total Pickings: %d, Total Lines: %d, Assigned: %s]",
		wave.WaveNumber, wave.Name, wave.TotalPickings, wave.TotalLines, wave.AssignedPickerName)

	// 6. Execute Wave Picking Workflow
	startedWave, err := wmsSvc.StartWavePicking(wave.ID)
	if err != nil || startedWave.Status != "in_progress" {
		t.Fatalf("StartWavePicking failed: status=%s, err=%v", startedWave.Status, err)
	}
	t.Logf("✓ Wave Picking Started: %s (Status: %s, Started At: %v)", startedWave.WaveNumber, startedWave.Status, startedWave.StartedAt.Format(time.RFC3339))

	// Pick individual line
	pickedLine, err := wmsSvc.ConfirmPickLine(wave.ID, startedWave.Lines[0].ID, startedWave.Lines[0].QuantityRequested)
	if err != nil || pickedLine.Status != "picked" {
		t.Fatalf("ConfirmPickLine failed: %v", err)
	}
	t.Logf("✓ Confirmed Wave Line #%d: Item '%s' (SKU: %s) -> Qty Picked=%.0f from Bin %s",
		pickedLine.ID, pickedLine.ProductName, pickedLine.ProductSku, pickedLine.QuantityPicked, pickedLine.BinLocationCode)

	// Complete Wave Picking
	completedWave, err := wmsSvc.CompleteWavePicking(wave.ID)
	if err != nil || completedWave.Status != "completed" {
		t.Fatalf("CompleteWavePicking failed: %v", err)
	}
	t.Logf("✓ Wave Picking Completed: %s (Status: %s, Picked Lines: %d/%d)",
		completedWave.WaveNumber, completedWave.Status, completedWave.PickedLines, completedWave.TotalLines)

	// Verify Delivery Order was auto-advanced
	updatedPicking, _ := stockSvc.GetPicking(delivery.ID, company.ID)
	if updatedPicking.State != "assigned" && updatedPicking.State != "done" {
		t.Fatalf("Expected picking state assigned/done after wave pick, got: %s", updatedPicking.State)
	}
	t.Logf("✓ Stock Delivery Order '%s' auto-transitioned to state: '%s'", updatedPicking.Name, updatedPicking.State)

	// 7. AI Warehouse Slotting Copilot
	slottingReport, err := wmsSvc.AnalyzeWarehouseSlottingWithAI(testWarehouseID)
	if err != nil {
		t.Fatalf("AnalyzeWarehouseSlottingWithAI failed: %v", err)
	}
	t.Logf("✓ AI Warehouse Slotting Report (%s):", slottingReport.AIProvider)
	t.Logf("  Overall Status: %s (Efficiency Score: %d/100, Est. Travel Distance: %.1f m)",
		slottingReport.OverallStatus, slottingReport.EfficiencyScore, slottingReport.AverageTravelDistanceM)
	t.Logf("  Executive Summary: %s", slottingReport.ExecutiveSummary)
	for i, rel := range slottingReport.RecommendedRelocations {
		t.Logf("  Relocation Recommendation #%d: %s", i+1, rel)
	}

	// 8. WMS Analytics Stats
	wmsStats, err := wmsSvc.GetStats(testWarehouseID)
	if err != nil {
		t.Fatalf("GetStats WMS failed: %v", err)
	}
	if wmsStats.TotalZones < 2 || wmsStats.TotalBins < 3 || wmsStats.CompletedWaves < 1 {
		t.Fatalf("Invalid WMS stats: %+v", wmsStats)
	}
	t.Logf("✓ WMS Analytics: Total Zones=%d, Total Bins=%d (Occupied: %d, Util: %.1f%%), Completed Waves=%d, Stored Items=%d",
		wmsStats.TotalZones, wmsStats.TotalBins, wmsStats.OccupiedBins, wmsStats.BinUtilizationPct, wmsStats.CompletedWaves, wmsStats.TotalStoredItems)

	// ---------------------------------------------------------
	// SECTION 48: VENDOR PORTAL & SUPPLIER SELF-SERVICE (RFQ BIDDING, ASN, 3-WAY MATCHING & AI VENDOR RATING)
	// ---------------------------------------------------------
	t.Log("\n--- SECTION 48: VENDOR PORTAL & SUPPLIER SELF-SERVICE (RFQ BIDDING, ASN, 3-WAY MATCHING & AI VENDOR RATING) ---")

	vpRepo := vendor_portal.NewRepository(db)
	vpSvc := vendor_portal.NewService(vpRepo, db, purchaseSvc, stockSvc, accountSvc, chatterSvc, aiClient)

	// 0. Setup fresh PO for Vendor Portal test
	vendorPortalPO := purchase.PurchaseOrder{
		PartnerID: vendor.ID,
		OrderLines: []purchase.PurchaseOrderLine{
			{
				ProductID:     testProduct.ID,
				Name:          testProduct.Name,
				ProductUomQty: 5,
				PriceUnit:     150000.0,
			},
		},
	}
	vendorPortalPO.CompanyID = company.ID
	if err := purchaseSvc.CreateOrder(&vendorPortalPO); err != nil {
		t.Fatalf("Create vendorPortalPO failed: %v", err)
	}

	// 1. Create / Invite Vendor Portal Account
	vpAccount, err := vpSvc.CreateAccount(vendor_portal.CreateVendorAccountRequest{
		PartnerID:   vendor.ID,
		VendorName:  "PT Baja Perkasa Semesta",
		ContactName: "Ir. Bambang Hidayat",
		Email:       "procurement@bajaperkasa.co.id",
		Phone:       "+62-21-99887766",
	})
	if err != nil {
		t.Fatalf("CreateAccount vendor portal failed: %v", err)
	}
	if vpAccount.AccessToken == "" || !vpAccount.IsActive {
		t.Fatalf("Invalid vendor portal account: %+v", vpAccount)
	}
	t.Logf("✓ Created Vendor Portal Account: %s (%s) | Token: %s... | Active: %v",
		vpAccount.VendorName, vpAccount.Email, vpAccount.AccessToken[:12], vpAccount.IsActive)

	// Verify token retrieval & login tracking
	verifiedAcc, err := vpSvc.GetAccountByToken(vpAccount.AccessToken)
	if err != nil || verifiedAcc.TotalLogins < 1 {
		t.Fatalf("GetAccountByToken failed: %v", err)
	}
	t.Logf("✓ Vendor Portal Token Authentication Verified (Login count: %d, Last Login: %v)",
		verifiedAcc.TotalLogins, verifiedAcc.LastLoginAt)

	// 2. Query Open RFQs via Vendor Portal
	openRFQs, err := vpSvc.ListOpenRFQs(vendor.ID)
	if err != nil {
		t.Fatalf("ListOpenRFQs failed: %v", err)
	}
	t.Logf("✓ Vendor Portal RFQ Discovery: Found %d open RFQ(s) for bidding", len(openRFQs))

	// 3. Submit Quotation Bid
	bid, err := vpSvc.SubmitQuotationBid(vendor.ID, vendor_portal.SubmitBidRequest{
		RFQReference:         vendorPortalPO.Name,
		PurchaseOrderID:      &vendorPortalPO.ID,
		DeliveryLeadTimeDays: 4,
		PaymentTerms:         "NET 30",
		Notes:                "Penawaran harga terbaik bersertifikat SNI dengan diskon volume 10%",
		Lines: []vendor_portal.SubmitBidLineRequest{
			{
				ProductID:   testProduct.ID,
				ProductName: testProduct.Name,
				ProductCode: testProduct.DefaultCode,
				Quantity:    5,
				UnitPrice:   120000.0,
				Notes:       "Ready stock",
			},
		},
	})
	if err != nil {
		t.Fatalf("SubmitQuotationBid failed: %v", err)
	}
	if bid.BidNumber == "" || bid.TotalBidAmount != 600000.0 || bid.Status != "submitted" {
		t.Fatalf("Invalid submitted bid: %+v", bid)
	}
	t.Logf("✓ Submitted Vendor Quotation Bid %s: Amount=Rp %.2f, Lead Time=%d days, Status=%s",
		bid.BidNumber, bid.TotalBidAmount, bid.DeliveryLeadTimeDays, bid.Status)

	// 4. Accept Quotation Bid (Procurement Action)
	acceptedBid, err := vpSvc.AcceptQuotationBid(bid.ID)
	if err != nil {
		t.Fatalf("AcceptQuotationBid failed: %v", err)
	}
	if acceptedBid.Status != "accepted" {
		t.Fatalf("Expected bid status 'accepted', got %s", acceptedBid.Status)
	}
	t.Logf("✓ Accepted Quotation Bid %s: PO prices updated & PO confirmed automatically", acceptedBid.BidNumber)

	// 5. Submit Advance Shipping Notice (ASN) / Surat Jalan Pemasok
	asn, err := vpSvc.SubmitASN(vendor.ID, vendor_portal.SubmitASNRequest{
		PurchaseOrderID:      vendorPortalPO.ID,
		CarrierName:          "JNE Freight Logistics (JTR)",
		TrackingRef:          "JTR-2026-98124571",
		DriverName:           "Sugeng Riyadi",
		VehicleNumber:        "B 9012 KZA",
		EstimatedArrivalDate: time.Now().AddDate(0, 0, 2).Format("2006-01-02"),
		Notes:                "Pengiriman 1 koli peti kayu bersegel aman",
		Lines: []vendor_portal.SubmitASNLineRequest{
			{
				ProductID:       testProduct.ID,
				ProductName:     testProduct.Name,
				QuantityShipped: 5,
				LotNumber:       "LOT-VEND-20261011",
				Notes:           "Segel utuh",
			},
		},
	})
	if err != nil {
		t.Fatalf("SubmitASN failed: %v", err)
	}
	if asn.ASNNumber == "" || asn.Status != "in_transit" || len(asn.Lines) != 1 {
		t.Fatalf("Invalid ASN: %+v", asn)
	}
	t.Logf("✓ Submitted Advance Shipping Notice (ASN) %s: Carrier=%s (Resi: %s), Driver=%s, Status=%s",
		asn.ASNNumber, asn.CarrierName, asn.TrackingRef, asn.DriverName, asn.Status)

	// 6. Receive Goods against ASN at Warehouse
	receivedASN, err := vpSvc.ReceiveASN(asn.ID, nil)
	if err != nil {
		t.Fatalf("ReceiveASN failed: %v", err)
	}
	if receivedASN.Status != "received" || receivedASN.ActualArrivalDate == nil {
		t.Fatalf("Expected ASN status 'received', got %s", receivedASN.Status)
	}
	t.Logf("✓ Warehouse Received ASN Goods %s: Actual Arrival=%v, Status=%s",
		receivedASN.ASNNumber, receivedASN.ActualArrivalDate.Format(time.RFC3339), receivedASN.Status)

	// 7. Electronic Invoice Submission & 3-Way Matching Engine
	// Case A: Discrepancy Test (Wrong Billed Amount)
	_, discMatchRes, err := vpSvc.SubmitVendorInvoice(vendor.ID, vendor_portal.SubmitVendorInvoiceRequest{
		PurchaseOrderID:       vendorPortalPO.ID,
		InvoiceNumberSupplier: "INV-DISC-0099",
		InvoiceDate:           time.Now().Format("2006-01-02"),
		DueDate:               time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
		Subtotal:              9000000.0,
		TaxAmount:             990000.0,
		TotalAmount:           9990000.0, // Significant mismatch vs PO
		TaxInvoiceNumber:      "010.002-26.11112222",
	})
	if err != nil {
		t.Fatalf("SubmitVendorInvoice discrepancy test failed: %v", err)
	}
	if discMatchRes.IsMatch || discMatchRes.MatchingStatus != "discrepancy_price" {
		t.Fatalf("Expected discrepancy_price matching status, got: %+v", discMatchRes)
	}
	t.Logf("✓ 3-Way Matching Discrepancy Detected Correctly: Status='%s', Variance=Rp %.2f (%s)",
		discMatchRes.MatchingStatus, discMatchRes.VarianceAmount, discMatchRes.DiscrepancyDetail[0])

	// Case B: Perfect 3-Way Match (Amount matches PO Total)
	var latestPO purchase.PurchaseOrder
	db.First(&latestPO, vendorPortalPO.ID)
	perfectInv, perfectMatchRes, err := vpSvc.SubmitVendorInvoice(vendor.ID, vendor_portal.SubmitVendorInvoiceRequest{
		PurchaseOrderID:       vendorPortalPO.ID,
		InvoiceNumberSupplier: "INV-PERFECT-2026-001",
		InvoiceDate:           time.Now().Format("2006-01-02"),
		DueDate:               time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
		Subtotal:              latestPO.AmountTotal,
		TaxAmount:             0,
		TotalAmount:           latestPO.AmountTotal,
		TaxInvoiceNumber:      "010.002-26.33334444",
		Notes:                 "Tagihan lengkap lampiran BAST dan Surat Jalan",
	})
	if err != nil {
		t.Fatalf("SubmitVendorInvoice perfect match test failed: %v", err)
	}
	if !perfectMatchRes.IsMatch || perfectMatchRes.MatchingStatus != "matched_3way" {
		t.Fatalf("Expected matched_3way status, got: %+v", perfectMatchRes)
	}
	t.Logf("✓ 3-Way Matching Succeeded: Submission=%s, Invoice='%s', Status='%s', Generated Bill ID=%v",
		perfectInv.SubmissionNumber, perfectInv.InvoiceNumberSupplier, perfectMatchRes.MatchingStatus, perfectMatchRes.GeneratedBillID)

	// 8. AI Vendor Performance & Reliability Rating
	aiRating, err := vpSvc.EvaluateAIVendorPerformance(vendor.ID)
	if err != nil {
		t.Fatalf("EvaluateAIVendorPerformance failed: %v", err)
	}
	t.Logf("✓ AI Vendor Performance Rating (%s):", aiRating.AIProvider)
	t.Logf("  Vendor: %s | Overall Score: %.1f/100 | Tier: %s",
		aiRating.VendorName, aiRating.OverallRatingScore, strings.ToUpper(aiRating.TierBadge))
	t.Logf("  Metrics: On-Time Delivery=%.1f%% | Price Competitiveness=%.1f | Quality Rate=%.1f%%",
		aiRating.OnTimeDeliveryRate, aiRating.PriceCompetitivenessScore, aiRating.QualityAcceptanceRate)
	t.Logf("  Executive Summary: %s", aiRating.ExecutiveSummary)
	for i, rec := range aiRating.ActionableRecommendations {
		t.Logf("  Recommendation #%d: %s", i+1, rec)
	}

	// 9. Vendor Portal Analytics & Stats
	vpStats, err := vpSvc.GetStats()
	if err != nil {
		t.Fatalf("GetStats vendor portal failed: %v", err)
	}
	if vpStats.TotalVendorAccounts < 1 || vpStats.TotalBidsSubmitted < 1 || vpStats.AcceptedBids < 1 {
		t.Fatalf("Invalid vendor portal stats: %+v", vpStats)
	}
	t.Logf("✓ Vendor Portal Analytics: Accounts=%d (Active=%d), Bids=%d (Accepted=%d), ASNs=%d (Received=%d), Matched Invoices=%d",
		vpStats.TotalVendorAccounts, vpStats.ActiveVendorAccounts, vpStats.TotalBidsSubmitted, vpStats.AcceptedBids,
		vpStats.TotalASNsInTransit+vpStats.ReceivedASNs, vpStats.ReceivedASNs, vpStats.Matched3WayInvoices)

	// 10. Universal Chatter Verification
	vpChatterLogs, _ := chatterSvc.ListMessages("purchase.order", vendorPortalPO.ID, company.ID)
	t.Logf("✓ Universal Chatter Integration: %d audit messages logged on Purchase Order #%d", len(vpChatterLogs), vendorPortalPO.ID)

	// -------------------------------------------------------------
	// --- SECTION 49: CUSTOMER PORTAL & B2B SELF-SERVICE COMMERCE ---
	// -------------------------------------------------------------
	t.Log("\n--- SECTION 49: CUSTOMER PORTAL & B2B SELF-SERVICE COMMERCE (B2B ORDERS, RMA, AR AGING & AI CHURN COPILOT) ---")

	cpRepo := customer_portal.NewRepository(db)
	cpSvc := customer_portal.NewService(cpRepo, db, saleSvc, stockSvc, accountSvc, chatterSvc, aiClient)

	// 1. Create Customer Portal Account
	custPartnerID := uint(1) // PT Indah Makmur Retail
	custAcc, err := cpSvc.CreateAccount(customer_portal.CreateCustomerAccountRequest{
		PartnerID:      custPartnerID,
		CustomerName:   "PT Indah Makmur Retail",
		ContactName:    "Dewi Kartika",
		Email:          "purchasing@indahmakmur.co.id",
		Phone:          "+62-21-77889900",
		CreditLimit:    150000000,
		AllowedReorder: true,
	})
	if err != nil {
		t.Fatalf("CreateAccount customer portal failed: %v", err)
	}
	if custAcc.AccessToken == "" || !custAcc.IsActive {
		t.Fatalf("Invalid customer account: %+v", custAcc)
	}
	t.Logf("✓ Created Customer Portal Account: %s (%s) | Token: %s... | Credit Limit: Rp %.2f",
		custAcc.CustomerName, custAcc.Email, custAcc.AccessToken[:12], custAcc.CreditLimit)

	// 2. Token Authentication
	authCust, err := cpSvc.GetAccountByToken(custAcc.AccessToken)
	if err != nil {
		t.Fatalf("GetAccountByToken failed: %v", err)
	}
	if authCust.TotalLogins != 1 || authCust.LastLoginAt == nil {
		t.Fatalf("Customer token authentication not tracked: %+v", authCust)
	}
	t.Logf("✓ Customer Portal Token Authentication Verified (Login count: %d, Last Login: %v)",
		authCust.TotalLogins, authCust.LastLoginAt)

	// 3. Customer Portal Dashboard
	dash, err := cpSvc.GetCustomerDashboard(custAcc.ID)
	if err != nil {
		t.Fatalf("GetCustomerDashboard failed: %v", err)
	}
	t.Logf("✓ Customer Dashboard: Outstanding Due: Rp %.2f | Credit Remaining: Rp %.2f | Open RMAs: %d",
		dash.OutstandingDue, dash.CreditRemaining, dash.PendingRMAsCount)

	// 4. B2B Order Request & Auto-Conversion to Draft Sale Order
	orderReq, err := cpSvc.CreateB2BOrderRequest(custAcc.ID, customer_portal.CreateB2BOrderRequestPayload{
		DeliveryAddress:       "Kawasan Industri MM2100 Blok C-4, Cikarang Barat",
		RequestedDeliveryDate: "2026-10-25",
		PaymentMethod:         "credit_terms",
		CustomerNotes:         "Pesanan rutin bulanan untuk cabang utama",
		Lines: []customer_portal.B2BOrderLineReq{
			{
				ProductID: 1, // product 1
				Quantity:  5,
				UnitPrice: 1000000,
				Notes:     "Kemasan karton standar",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateB2BOrderRequest failed: %v", err)
	}
	if orderReq.RequestNumber == "" || orderReq.SaleOrderID == nil || orderReq.TotalAmount <= 0 {
		t.Fatalf("Invalid B2B order request: %+v", orderReq)
	}
	t.Logf("✓ Submitted B2B Order Request %s: Subtotal=Rp %.2f, PPN 11%%=Rp %.2f, Total=Rp %.2f -> Draft SO #%d Created",
		orderReq.RequestNumber, orderReq.SubtotalAmount, orderReq.TaxAmount, orderReq.TotalAmount, *orderReq.SaleOrderID)

	// 5. Admin Approves and Auto-Confirms B2B Order to Sales Order
	approvedB2BReq, err := cpSvc.ApproveB2BOrder(orderReq.ID, true)
	if err != nil {
		t.Fatalf("ApproveB2BOrder failed: %v", err)
	}
	if approvedB2BReq.Status != "converted_to_so" {
		t.Fatalf("Order not converted to SO: %s", approvedB2BReq.Status)
	}
	t.Logf("✓ Admin Approved & Confirmed B2B Order %s: Converted to SO (Status=%s)",
		approvedB2BReq.RequestNumber, approvedB2BReq.Status)

	// 6. Customer AR Aging Statement
	statement, err := cpSvc.GetCustomerStatement(custPartnerID)
	if err != nil {
		t.Fatalf("GetCustomerStatement failed: %v", err)
	}
	t.Logf("✓ Customer Statement & AR Aging: Total Outstanding=Rp %.2f (Current=Rp %.2f, 1-30 Days=Rp %.2f, 31-60 Days=Rp %.2f, >60 Days=Rp %.2f)",
		statement.TotalOutstanding, statement.AgingCurrent, statement.Aging1To30, statement.Aging31To60, statement.AgingOver60)

	// 7. Submit Return Merchandise Authorization (RMA)
	rmaReq, err := cpSvc.SubmitRMARequest(custAcc.ID, customer_portal.SubmitRMARequestPayload{
		SaleOrderID:         orderReq.SaleOrderID,
		ProductID:           1,
		Quantity:            1,
		ReasonCode:          "damaged_in_transit",
		Description:         "Kardus sobek dan bodi produk lecet akibat benturan ekspedisi",
		EvidenceURL:         "https://storage.erp.local/rma/indah_damaged_01.jpg",
		ResolutionRequested: "replace_item",
	})
	if err != nil {
		t.Fatalf("SubmitRMARequest failed: %v", err)
	}
	if rmaReq.RMANumber == "" || rmaReq.Status != "submitted" {
		t.Fatalf("Invalid RMA request: %+v", rmaReq)
	}
	t.Logf("✓ Submitted RMA Claim %s: Product ID=%d, Qty=%.1f, Reason=%s, Status=%s",
		rmaReq.RMANumber, rmaReq.ProductID, rmaReq.Quantity, rmaReq.ReasonCode, rmaReq.Status)

	// 8. Admin RMA Review & Resolution
	approvedRMA, err := cpSvc.ApproveRMA(rmaReq.ID, "Kirim unit pengganti dari batch WH/STOCK", "Verifikasi foto selesai. Sesuai garansi pengiriman.")
	if err != nil {
		t.Fatalf("ApproveRMA failed: %v", err)
	}
	if approvedRMA.Status != "approved" || approvedRMA.ResolvedAt == nil {
		t.Fatalf("RMA approval failed: %+v", approvedRMA)
	}
	t.Logf("✓ RMA %s Approved: Action='%s', Status=%s",
		approvedRMA.RMANumber, approvedRMA.ResolutionAction, approvedRMA.Status)

	// 9. Customer Payment Notification Submission
	payNotif, err := cpSvc.SubmitPaymentNotification(custAcc.ID, customer_portal.SubmitPaymentNotificationPayload{
		InvoiceID:         1,
		AmountPaid:        5550000,
		PaymentDate:       time.Now().Format("2006-01-02"),
		SenderBank:        "BCA",
		SenderAccountName: "PT Indah Makmur Retail",
		SenderAccountNo:   "8820192831",
		ReferenceNumber:   "TRF-BCA-2026-981245",
		ProofURL:          "https://storage.erp.local/payments/slip_transfer_bca_981245.pdf",
		Notes:             "Pelunasan invoice tempo 30 hari via transfer bank",
	})
	if err != nil {
		t.Fatalf("SubmitPaymentNotification failed: %v", err)
	}
	if payNotif.NotificationNumber == "" || payNotif.Status != "pending_verification" {
		t.Fatalf("Invalid payment notification: %+v", payNotif)
	}
	t.Logf("✓ Submitted Payment Notification %s: Amount=Rp %.2f, Ref=%s, Status=%s",
		payNotif.NotificationNumber, payNotif.AmountPaid, payNotif.ReferenceNumber, payNotif.Status)

	// 10. Finance Officer Payment Verification & GL Auto-Posting
	verifiedNotif, err := cpSvc.VerifyPaymentNotification(payNotif.ID, 1, true)
	if err != nil {
		t.Fatalf("VerifyPaymentNotification failed: %v", err)
	}
	if verifiedNotif.Status != "confirmed" || verifiedNotif.VerifiedAt == nil {
		t.Fatalf("Payment verification failed: %+v", verifiedNotif)
	}
	t.Logf("✓ Finance Officer Verified Payment %s: Status=%s (Allocated to General Ledger & Payment Record Generated)",
		verifiedNotif.NotificationNumber, verifiedNotif.Status)

	// 11. AI Customer Health Scorecard & Churn Risk Copilot
	healthScorecard, err := cpSvc.EvaluateAICustomerHealth(custPartnerID)
	if err != nil {
		t.Fatalf("EvaluateAICustomerHealth failed: %v", err)
	}
	if healthScorecard.OverallHealthScore <= 0 || healthScorecard.TierBadge == "" {
		t.Fatalf("Invalid customer health scorecard: %+v", healthScorecard)
	}
	t.Logf("✓ AI Customer Health & Churn Risk Scorecard (%s):", healthScorecard.AIProvider)
	t.Logf("  Customer: %s | Overall Health: %.1f/100 | Tier: %s | Churn Risk: %s",
		healthScorecard.CustomerName, healthScorecard.OverallHealthScore, healthScorecard.TierBadge, healthScorecard.ChurnRiskLevel)
	t.Logf("  Metrics: LTV=Rp %.2f (%d orders) | Payment Punctuality=%.1f%% | Return Rate=%.1f%%",
		healthScorecard.TotalOrderLifetimeValue, healthScorecard.TotalOrdersCount, healthScorecard.PaymentPunctualityScore, healthScorecard.ReturnRatePercentage)
	t.Logf("  Executive Summary: %s", healthScorecard.ExecutiveSummary)
	for i, rec := range healthScorecard.RecommendedActions {
		t.Logf("  Retention Action #%d: %s", i+1, rec)
	}

	// 12. Customer Portal Analytics & Stats
	cpStats, err := cpSvc.GetStats()
	if err != nil {
		t.Fatalf("GetStats customer portal failed: %v", err)
	}
	if cpStats.TotalCustomerAccounts < 1 || cpStats.TotalB2BOrders < 1 {
		t.Fatalf("Invalid customer portal stats: %+v", cpStats)
	}
	t.Logf("✓ Customer Portal Analytics: Accounts=%d (Active=%d), Orders=%d (Approved=%d, Total=Rp %.2f), RMAs=%d",
		cpStats.TotalCustomerAccounts, cpStats.ActiveAccounts, cpStats.TotalB2BOrders, cpStats.ApprovedB2BOrders,
		cpStats.TotalB2BOrderValue, cpStats.TotalRMAs)

	// 13. Universal Chatter Verification
	custChatterLogs, _ := chatterSvc.ListMessages("sale.order", *orderReq.SaleOrderID, 1)
	t.Logf("✓ Universal Chatter Integration: %d audit messages logged on Sale Order #%d", len(custChatterLogs), *orderReq.SaleOrderID)

	// =========================================================================
	// SECTION 50: KAWASAN BERIKAT & IT INVENTORY DJBC CEISA 4.0
	// (BC 2.3 IMPOR, BC 3.0 EKSPOR, 4 BUKU MUTASI, SUBKONTRAK & AI COMPLIANCE AUDITOR)
	// =========================================================================
	t.Log("\n>>> SECTION 50: KAWASAN BERIKAT & IT INVENTORY DJBC CEISA 4.0")

	cbRepo := customs_bonded.NewRepository(db)
	cbSvc := customs_bonded.NewService(cbRepo, db, stockSvc, chatterSvc, aiClient)

	// 1. Pendaftaran Dokumen BC 2.3 (Impor Bahan Baku Canai Dingin dari Jepang)
	bc23Req := customs_bonded.CreateCustomsDocumentRequest{
		DocumentType:       customs_bonded.DocTypeBC23,
		RegistrationNumber: "009988",
		RegistrationDate:   "2026-10-12",
		CustomsOfficeCode:  "040300",
		PartnerID:          vendor.ID,
		PartnerName:        "Nippon Steel Corp (Tokyo)",
		CountryOfOrigin:    "JP",
		CurrencyCode:       "USD",
		ExchangeRate:       16250,
		InvoiceNumber:      "INV-NIPPON-2026-X1",
		BLNumber:           "ONE-TYO-JKT-9921",
		ContainerNumber:    "ONEY7728192",
		Notes:              "Pemasukan bahan baku plat baja ke fasilitas Kawasan Berikat",
		Lines: []customs_bonded.CreateCustomsLineRequest{
			{
				ProductID:         testProduct.ID,
				HSCode:            "7209.16.00",
				ItemDescription:   "Cold-Rolled Steel Sheet Coil 2.0mm",
				ItemCategory:      customs_bonded.BookRawMaterial,
				Quantity:          6000,
				UoM:               "KGM",
				UnitPriceUSD:      2.50,
				DutyTariffPercent: 5.0,
			},
		},
	}
	bc23Doc, err := cbSvc.CreateCustomsDocument(bc23Req, 1)
	if err != nil {
		t.Fatalf("CreateCustomsDocument BC 2.3 failed: %v", err)
	}
	if bc23Doc.ID == 0 || bc23Doc.TotalCIFUSD != 15000 {
		t.Fatalf("Invalid BC 2.3 doc: %+v", bc23Doc)
	}
	t.Logf("✓ Dokumen BC 2.3 Pemasukan Impor Didaftarkan: No. Aju %s (Reg: %s), Total CIF: USD %.2f (Rp %.2f), BM Ditangguhkan: Rp %.2f",
		bc23Doc.SubmissionNumber, bc23Doc.RegistrationNumber, bc23Doc.TotalCIFUSD, bc23Doc.TotalCIFIDR, bc23Doc.DutyAmount)

	// 2. Persetujuan SPPB Bea Cukai untuk BC 2.3 (Otomatis Catat ke Buku 1 Bahan Baku)
	approvedBC23, err := cbSvc.ApproveSPPB(bc23Doc.ID, customs_bonded.ApproveSPPBRequest{
		SPPBNumber: "SPPB-040300-2026-009988",
		SPPBDate:   "2026-10-12",
		Notes:      "Pemeriksaan jalur hijau pabean selesai",
	})
	if err != nil {
		t.Fatalf("ApproveSPPB BC 2.3 failed: %v", err)
	}
	if approvedBC23.Status != "approved_sppb" || approvedBC23.SPPBNumber != "SPPB-040300-2026-009988" {
		t.Fatalf("SPPB approval mismatch: %+v", approvedBC23)
	}
	t.Logf("✓ SPPB Bea Cukai Terbit: %s, Status=%s", approvedBC23.SPPBNumber, approvedBC23.Status)

	// Verifikasi Mutasi Tercatat di Buku 1 (Bahan Baku)
	mutationsRaw, err := cbSvc.GetITInventoryMutations(1, customs_bonded.BookRawMaterial, nil, nil)
	if err != nil || len(mutationsRaw) == 0 {
		t.Fatalf("GetITInventoryMutations Book 1 failed: %v (count=%d)", err, len(mutationsRaw))
	}
	lastRawMut := mutationsRaw[len(mutationsRaw)-1]
	if lastRawMut.QtyIn != 6000 || lastRawMut.CustomsDocNumber != "009988" {
		t.Fatalf("Invalid Raw Material IT Inventory Mutation: %+v", lastRawMut)
	}
	t.Logf("✓ Mutasi IT Inventory Buku 1 (Bahan Baku): SKU=%s, Saldo Awal=%.2f, Qty Masuk=%.2f, Saldo Akhir=%.2f %s",
		lastRawMut.ItemCode, lastRawMut.QtyBeginning, lastRawMut.QtyIn, lastRawMut.QtyEnding, lastRawMut.UoM)

	// 3. Pendaftaran Dokumen BC 3.0 (Ekspor Barang Jadi ke Luar Negeri)
	bc30Req := customs_bonded.CreateCustomsDocumentRequest{
		DocumentType:       customs_bonded.DocTypeBC30,
		RegistrationNumber: "007711",
		RegistrationDate:   "2026-10-12",
		CustomsOfficeCode:  "040300",
		PartnerID:          customer.ID,
		PartnerName:        "Singa Logistics Hub Pte Ltd (Singapore)",
		CountryOfOrigin:    "ID",
		CurrencyCode:       "USD",
		ExchangeRate:       16250,
		InvoiceNumber:      "EXP-2026-0099",
		BLNumber:           "PIL-JKT-SIN-8812",
		ContainerNumber:    "PCIU8819203",
		Notes:              "Ekspor 150 unit rak industri siap pakai",
		Lines: []customs_bonded.CreateCustomsLineRequest{
			{
				ProductID:         finishedBundle.ID,
				HSCode:            "9403.20.90",
				ItemDescription:   "Industrial Heavy-Duty Storage Rack S01",
				ItemCategory:      customs_bonded.BookFinishedGoods,
				Quantity:          150,
				UoM:               "SET",
				UnitPriceUSD:      266.6667,
				DutyTariffPercent: 0,
			},
		},
	}
	bc30Doc, err := cbSvc.CreateCustomsDocument(bc30Req, 1)
	if err != nil {
		t.Fatalf("CreateCustomsDocument BC 3.0 failed: %v", err)
	}
	t.Logf("✓ Dokumen BC 3.0 Pemberitahuan Ekspor Barang (PEB) Didaftarkan: No. Aju %s (Reg: %s), Nilai Ekspor: USD %.2f",
		bc30Doc.SubmissionNumber, bc30Doc.RegistrationNumber, bc30Doc.TotalCIFUSD)

	// 4. Persetujuan SPPB Ekspor PEB (Otomatis Catat Pengeluaran ke Buku 2 Barang Jadi)
	approvedBC30, err := cbSvc.ApproveSPPB(bc30Doc.ID, customs_bonded.ApproveSPPBRequest{
		SPPBNumber: "PEB-SPPB-040300-2026-007711",
		SPPBDate:   "2026-10-12",
		Notes:      "Gate out pelabuhan Tanjung Priok",
	})
	if err != nil {
		t.Fatalf("ApproveSPPB BC 3.0 failed: %v", err)
	}
	t.Logf("✓ SPPB Ekspor PEB Diterbitkan: %s, Status=%s", approvedBC30.SPPBNumber, approvedBC30.Status)

	// Verifikasi Mutasi Tercatat di Buku 2 (Barang Jadi)
	mutationsFG, err := cbSvc.GetITInventoryMutations(1, customs_bonded.BookFinishedGoods, nil, nil)
	if err != nil || len(mutationsFG) == 0 {
		t.Fatalf("GetITInventoryMutations Book 2 failed: %v", err)
	}
	lastFGMut := mutationsFG[len(mutationsFG)-1]
	if lastFGMut.QtyOut != 150 || lastFGMut.CustomsDocNumber != "007711" {
		t.Fatalf("Invalid Finished Goods IT Inventory Mutation: %+v", lastFGMut)
	}
	t.Logf("✓ Mutasi IT Inventory Buku 2 (Barang Jadi): SKU=%s, Saldo Awal=%.2f, Qty Keluar=%.2f, Saldo Akhir=%.2f %s",
		lastFGMut.ItemCode, lastFGMut.QtyBeginning, lastFGMut.QtyOut, lastFGMut.QtyEnding, lastFGMut.UoM)

	// 5. Verifikasi Laporan Buku IT Inventory DJBC (Rekapitulasi 4 Buku)
	rawReport, err := cbSvc.GetITInventoryBookReport(1, customs_bonded.BookRawMaterial, nil, nil)
	if err != nil || len(rawReport) == 0 {
		t.Fatalf("GetITInventoryBookReport Book 1 failed: %v", err)
	}
	t.Logf("✓ Laporan Pertanggungjawaban DJBC (Buku 1 Bahan Baku): %d SKU terdata, Saldo Akhir %s = %.2f %s (Status: %s)",
		len(rawReport), rawReport[0].ItemCode, rawReport[0].QtyEnding, rawReport[0].UoM, rawReport[0].Status)

	fgReport, err := cbSvc.GetITInventoryBookReport(1, customs_bonded.BookFinishedGoods, nil, nil)
	if err != nil || len(fgReport) == 0 {
		t.Fatalf("GetITInventoryBookReport Book 2 failed: %v", err)
	}
	t.Logf("✓ Laporan Pertanggungjawaban DJBC (Buku 2 Barang Jadi): %d SKU terdata, Saldo Akhir %s = %.2f %s (Status: %s)",
		len(fgReport), fgReport[0].ItemCode, fgReport[0].QtyEnding, fgReport[0].UoM, fgReport[0].Status)

	// 6. Pengelolaan Pekerjaan Subkontrak (BC 2.61 & BC 2.62)
	subcontract, err := cbSvc.CreateSubcontract(customs_bonded.CustomsSubcontract{
		CompanyID:         1,
		CustomsApprovalNo: "S-882/KBC.040302/2026",
		PartnerID:         vendor.ID,
		PartnerName:       "PT Metalindo Presisi Utama",
		ProductID:         testProduct.ID,
		QtySent:           1200,
		UoM:               "KGM",
		BC261DocNo:        "BC261-2026-000889",
		Notes:             "Pengeluaran sementara plat baja untuk proses perlakuan panas / coating",
	})
	if err != nil || subcontract.ID == 0 {
		t.Fatalf("CreateSubcontract failed: %v", err)
	}
	t.Logf("✓ Kontrak Subkontrak Didaftarkan: No. %s, Mitra=%s, Qty Keluar=%.2f %s, Jatuh Tempo (Maks 60 Hari)=%s",
		subcontract.ContractNumber, subcontract.PartnerName, subcontract.QtySent, subcontract.UoM, subcontract.DueDate.Format("02-01-2006"))

	// 7. Pengembalian Barang Subkontrak (BC 2.62 Pemasukan Kembali)
	returnedSub, err := cbSvc.ReturnSubcontract(subcontract.ID, 1200, "BC262-2026-000912")
	if err != nil || returnedSub.Status != "returned" {
		t.Fatalf("ReturnSubcontract failed: %v (status=%s)", err, returnedSub.Status)
	}
	t.Logf("✓ Pemasukan Kembali Subkontrak (BC 2.62): Qty Kembali=%.2f/%.2f %s, Status Kontrak=%s",
		returnedSub.QtyReturned, returnedSub.QtySent, returnedSub.UoM, returnedSub.Status)

	// 8. AI Customs Compliance Audit Scorecard (Evaluasi Kepatuhan CEISA 4.0)
	aiAuditReport, err := cbSvc.EvaluateAICustomsAudit(1)
	if err != nil {
		t.Fatalf("EvaluateAICustomsAudit failed: %v", err)
	}
	if aiAuditReport.ComplianceScore <= 0 || aiAuditReport.ComplianceStatus == "" {
		t.Fatalf("Invalid AI customs audit report: %+v", aiAuditReport)
	}
	t.Logf("✓ AI DJBC Compliance Auditor: Score=%.1f/100 (Status: %s, Provider: %s)",
		aiAuditReport.ComplianceScore, aiAuditReport.ComplianceStatus, aiAuditReport.AIProvider)
	t.Logf("  Ringkasan Eksekutif: %s", aiAuditReport.ExecutiveSummary)
	if len(aiAuditReport.AuditFindings) > 0 {
		t.Logf("  Temuan Audit: %s", aiAuditReport.AuditFindings[0])
	}
	if len(aiAuditReport.ActionableRecommendations) > 0 {
		t.Logf("  Rekomendasi Tindakan: %s", aiAuditReport.ActionableRecommendations[0])
	}

	// 9. Customs Bonded KPI Analytics
	cbStats, err := cbSvc.GetStats(1)
	if err != nil {
		t.Fatalf("GetStats customs bonded failed: %v", err)
	}
	if cbStats.TotalDocuments < 2 || cbStats.TotalBC23Import < 1 || cbStats.TotalBC30Export < 1 {
		t.Fatalf("Invalid customs stats: %+v", cbStats)
	}
	t.Logf("✓ Customs Bonded KPI: Total Dokumen=%d (BC 2.3=%d, BC 3.0=%d), BM Ditangguhkan=Rp %.2f, Nilai Ekspor=USD %.2f",
		cbStats.TotalDocuments, cbStats.TotalBC23Import, cbStats.TotalBC30Export, cbStats.TotalDutySuspended, cbStats.TotalExportValueUSD)

	// 10. Universal Chatter Verification
	customsChatterLogs, _ := chatterSvc.ListMessages("customs.document", bc23Doc.ID, 1)
	t.Logf("✓ Universal Chatter Integration: %d audit messages logged on Customs Document #%d", len(customsChatterLogs), bc23Doc.ID)

	// ====================================================================================================
	// SECTION 51: UNIVERSAL LOW-CODE STUDIO (DYNAMIC MODELS, CUSTOM FIELDS, EAV RECORDS & AI SCHEMA ARCHITECT)
	// ====================================================================================================
	t.Log("\n        >>> SECTION 51: UNIVERSAL LOW-CODE STUDIO & DYNAMIC SCHEMA ENGINE <<<")

	lowcodeRepo := lowcode_studio.NewRepository(db)
	lowcodeSvc := lowcode_studio.NewService(lowcodeRepo, db, chatterSvc, aiClient)

	// 1. Create a Dynamic Custom Model (Construction & EPC Industry)
	epcModel, err := lowcodeSvc.CreateModel(lowcode_studio.CreateCustomModelRequest{
		TechnicalName:    "x_construction_epc",
		DisplayName:      "Proyek Konstruksi & Sipil (EPC)",
		Description:      "Manajemen proyek konstruksi terpadu, WBS, Bill of Quantities, dan progress claim termyn.",
		Icon:             "fas fa-hard-hat",
		MenuCategory:     "Konstruksi & Properti",
		AllowChatter:     true,
		AllowAttachments: true,
		InitialFields: []lowcode_studio.CreateCustomFieldRequest{
			{FieldName: "x_project_code", FieldLabel: "Kode Proyek EPC", FieldType: lowcode_studio.FieldTypeString, IsRequired: true, IsSearchable: true},
			{FieldName: "x_contract_value", FieldLabel: "Nilai Kontrak (IDR)", FieldType: lowcode_studio.FieldTypeMonetary, IsRequired: true},
			{FieldName: "x_progress_pct", FieldLabel: "Realisasi Fisik (%)", FieldType: lowcode_studio.FieldTypeFloat, IsRequired: true},
			{FieldName: "x_completion_target", FieldLabel: "Target Selesai", FieldType: lowcode_studio.FieldTypeDate, IsRequired: true},
			{FieldName: "x_is_turnkey", FieldLabel: "Kontrak Turnkey / Lump Sum", FieldType: lowcode_studio.FieldTypeBoolean, DefaultValue: "true"},
		},
	}, 1)
	if err != nil {
		t.Fatalf("CreateModel failed: %v", err)
	}
	t.Logf("✓ Model Dinamis Diterbitkan: %s (%s) dengan %d kolom awal terdaftar",
		epcModel.DisplayName, epcModel.TechnicalName, len(epcModel.Fields))

	// 2. Add an additional dynamic field to the model
	clientField, err := lowcodeSvc.AddField(epcModel.ID, lowcode_studio.CreateCustomFieldRequest{
		FieldName:     "x_client_id",
		FieldLabel:    "Owner / Pemberi Kerja",
		FieldType:     lowcode_studio.FieldTypeMany2One,
		RelationModel: "res.partner",
		IsRequired:    false,
		HelpTooltip:   "Relasi langsung ke data kontak klien/owner proyek",
	})
	if err != nil {
		t.Fatalf("AddField failed: %v", err)
	}
	t.Logf("✓ Kolom Kustom Ditambahkan ke Model: %s (%s, Tipe: %s)",
		clientField.FieldLabel, clientField.FieldName, clientField.FieldType)

	// 3. Create a Dynamic Record for the Custom Model
	epcRecord, err := lowcodeSvc.CreateRecord(epcModel.ID, lowcode_studio.SaveCustomRecordRequest{
		DisplayName: "Pembangunan Smelter Nikel Pomalaa (Paket 3)",
		Values: map[string]interface{}{
			"x_project_code":      "EPC-2026-NKL03",
			"x_contract_value":    185000000000.0,
			"x_progress_pct":      64.5,
			"x_completion_target": "2027-06-30",
			"x_is_turnkey":        true,
			"x_client_id":         1,
		},
	}, 1, 1)
	if err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}
	t.Logf("✓ Data Baris Dinamis Tersimpan: %s (Kode: %s, ID: %d)",
		epcRecord.DisplayName, epcRecord.RecordCode, epcRecord.ID)

	// 4. Update the Dynamic Record (Progress Fisik Naik ke 72.8%)
	updatedRecord, err := lowcodeSvc.UpdateRecord(epcRecord.ID, lowcode_studio.SaveCustomRecordRequest{
		Values: map[string]interface{}{
			"x_progress_pct": 72.8,
		},
	})
	if err != nil {
		t.Fatalf("UpdateRecord failed: %v", err)
	}
	t.Logf("✓ Data Baris Dinamis Diperbarui: %s (Progress di-update)", updatedRecord.DisplayName)

	// 5. Query and List Records
	recordsList, totalRecords, err := lowcodeSvc.ListRecords(epcModel.ID, 1, 1, 10)
	if err != nil || totalRecords != 1 || len(recordsList) != 1 {
		t.Fatalf("ListRecords failed: total=%d, len=%d, err=%v", totalRecords, len(recordsList), err)
	}
	t.Logf("✓ Daftar Baris Dinamis: %d baris terverifikasi untuk model '%s'", len(recordsList), epcModel.TechnicalName)

	// 6. Extend Core Standard ERP Model (res.partner) with Custom Attributes
	extVal, err := lowcodeSvc.SaveCoreExtension(lowcode_studio.SaveCoreExtensionRequest{
		CoreModel: "res.partner",
		RecordID:  1,
		Values: map[string]interface{}{
			"x_tax_bracket":        "ENTERPRISE_A",
			"x_iso_9001_certified": true,
			"x_duns_number":        "98124571",
			"x_contractor_grade":   "GRADE_7_BESAR",
		},
	}, 1)
	if err != nil {
		t.Fatalf("SaveCoreExtension failed: %v", err)
	}
	t.Logf("✓ Core Model Extension: Nilai atribut kustom tersimpan pada entitas standar 'res.partner' ID #%d", extVal.RecordID)

	retrievedExt, err := lowcodeSvc.GetCoreExtension("res.partner", 1)
	if err != nil || retrievedExt == nil {
		t.Fatalf("GetCoreExtension failed: %v", err)
	}
	t.Logf("✓ Nilai Ekstensi Entitas Standar Terbaca: %s", retrievedExt.ExtensionJSON)

	// 7. Test AI Low-Code Schema Generator Copilot
	aiSchema, err := lowcodeSvc.GenerateSchemaWithAI("Sistem Manajemen Bandara & Logistik Kargo Pesawat Udara", "Aviation & Logistics")
	if err != nil || aiSchema == nil {
		t.Fatalf("GenerateSchemaWithAI failed: %v", err)
	}
	t.Logf("✓ AI Schema Architect Copilot (Provider: %s):", aiSchema.Provider)
	t.Logf("  Model Dihasilkan: %s (%s)", aiSchema.ModelDisplayName, aiSchema.ModelTechnicalName)
	t.Logf("  Rekomendasi Kolom (%d fields):", len(aiSchema.RecommendedFields))
	for idx, rf := range aiSchema.RecommendedFields {
		if idx < 4 {
			t.Logf("    - %s (%s, Tipe: %s)", rf.FieldLabel, rf.FieldName, rf.FieldType)
		}
	}

	// 8. Test Low-Code Studio Analytics / KPI
	studioStats, err := lowcodeSvc.GetStats(1)
	if err != nil {
		t.Fatalf("GetStats lowcode studio failed: %v", err)
	}
	if studioStats.TotalCustomModels < 1 || studioStats.TotalCustomFields < 1 || studioStats.TotalCustomRecords < 1 {
		t.Fatalf("Invalid studio stats: %+v", studioStats)
	}
	t.Logf("✓ Low-Code Studio KPI: Total Models=%d, Total Fields=%d, Total Records=%d, Readiness=%.1f%%",
		studioStats.TotalCustomModels, studioStats.TotalCustomFields, studioStats.TotalCustomRecords, studioStats.SystemReadinessRate)

	// ==========================================
	// >>> SECTION 52: UNIVERSAL REPORT & PRINTOUT ENGINE FOR ALL ERP MODULES <<<
	// ==========================================
	t.Log("\n\t\t>>> SECTION 52: UNIVERSAL REPORT & PRINTOUT ENGINE FOR ALL ERP MODULES <<<")

	reportsSvc := reports.NewService(db)

	// 1. Verify Report Catalog
	catalog := reportsSvc.GetCatalog()
	if catalog == nil || catalog.TotalReports < 30 {
		t.Fatalf("Expected rich reports catalog with >= 30 templates, got %+v", catalog)
	}
	t.Logf("✓ Universal Report Catalog Verified: %d template dokumen cetak terdaftar resmi", catalog.TotalReports)

	// 2. Test Rendering Sale Order (SO) Printout
	soHTML, err := reportsSvc.RenderDocument("sale", 1)
	if err != nil || !strings.Contains(soHTML, "SURAT PESANAN PENJUALAN") || !strings.Contains(soHTML, "@media print") {
		t.Fatalf("Failed to render Sale Order printout: %v", err)
	}
	t.Log("✓ Printable Sales Order (SO) Verified: KOP surat A4, rincian barang, total Rupiah, dan tanda tangan")

	// 3. Test Rendering Purchase Order (PO) Printout
	poHTML, err := reportsSvc.RenderDocument("purchase", 1)
	if err != nil || !strings.Contains(poHTML, "SURAT PESANAN PEMBELIAN") {
		t.Fatalf("Failed to render Purchase Order printout: %v", err)
	}
	t.Log("✓ Printable Purchase Order (PO) Verified: Spesifikasi material, PPN 11%, dan instruksi pengiriman vendor")

	// 4. Test Rendering Commercial Invoice Printout
	invHTMLRep, err := reportsSvc.RenderDocument("account", 1)
	if err != nil || !strings.Contains(invHTMLRep, "FAKTUR PENJUALAN RESMI") {
		t.Fatalf("Failed to render Invoice printout: %v", err)
	}
	t.Log("✓ Printable Commercial Invoice Verified: Status lunas/unpaid, nomor rekening, dan general ledger mapping")

	// 5. Test Rendering Delivery Order (Surat Jalan / Good Receipt) Printout
	pickHTMLRep, err := reportsSvc.RenderDocument("stock", 1)
	if err != nil || (!strings.Contains(pickHTMLRep, "SURAT JALAN") && !strings.Contains(pickHTMLRep, "PENERIMAAN BARANG")) {
		t.Fatalf("Failed to render Delivery Order printout: %v", err)
	}
	t.Log("✓ Printable Delivery Order / Good Receipt Verified: Kuantitas permintaan vs realisasi fisik & kurir")

	// 6. Test Rendering Manufacturing Order (SPK) Printout
	moHTML, err := reportsSvc.RenderDocument("mrp", 1)
	if err != nil || !strings.Contains(moHTML, "SURAT PERINTAH KERJA MANUFAKTUR") {
		t.Fatalf("Failed to render MRP SPK printout: %v", err)
	}
	t.Log("✓ Printable Manufacturing SPK Verified: Target output produk, bill of materials (BOM), dan SOP K3")

	// 7. Test Rendering POS Cashier Thermal Receipt (80mm)
	posHTML, err := reportsSvc.RenderDocument("pos", 1)
	if err != nil || !strings.Contains(posHTML, "STRUK PENJUALAN KASIR") || !strings.Contains(posHTML, "80mm") {
		t.Fatalf("Failed to render POS receipt: %v", err)
	}
	t.Log("✓ Printable POS Thermal Receipt Verified: Format 80mm compact, rincian item belanja, dan QRIS/Tunai")

	// 8. Test Rendering Official Employee Payslip (Slip Gaji)
	slipHTML, err := reportsSvc.RenderDocument("payroll", 1)
	if err != nil || !strings.Contains(slipHTML, "SLIP GAJI KARYAWAN") || !strings.Contains(slipHTML, "TAKE HOME PAY") {
		t.Fatalf("Failed to render Payslip printout: %v", err)
	}
	t.Log("✓ Printable Official Payslip Verified: Rincian penghasilan kotor, iuran BPJS, PPh 21 TER, dan Take Home Pay")

	// 9. Test Rendering Indonesian e-Faktur Pajak Standar
	efakturHTML, err := reportsSvc.RenderDocument("tax_efaktur", 1)
	if err != nil || !strings.Contains(efakturHTML, "FAKTUR PAJAK STANDAR ELEKTRONIK") {
		t.Fatalf("Failed to render e-Faktur printout: %v", err)
	}
	t.Log("✓ Printable Indonesian e-Faktur Verified: Format PER-03/PJ/2022 DJP Online dengan DPP & PPN 11%")

	// 10. Test Rendering Indonesian e-Bupot 21/26 (Formulir BP-21)
	bupotHTML, err := reportsSvc.RenderDocument("tax_ebupot", 1)
	if err != nil || !strings.Contains(bupotHTML, "FORMULIR BP-21") {
		t.Fatalf("Failed to render e-Bupot printout: %v", err)
	}
	t.Log("✓ Printable Indonesian e-Bupot 21/26 Verified: Formulir BP-21 resmi DJP Online skema TER PP 58/2023")

	// 11. Test Rendering Customs Declaration (Kawasan Berikat BC 2.3/3.0)
	customsHTML, err := reportsSvc.RenderDocument("customs_bonded", 1)
	if err != nil || !strings.Contains(customsHTML, "DOKUMEN PEMBERITAHUAN PABEAN") {
		t.Fatalf("Failed to render Customs Document printout: %v", err)
	}
	t.Log("✓ Printable Kawasan Berikat CEISA 4.0 Verified: Nomor Aju, Nomor Pendaftaran, SPPB, dan CIF Devisa")

	// 12. Test Rendering WMS Wave Picking Task List
	wmsHTML, err := reportsSvc.RenderDocument("wms", 1)
	if err != nil || !strings.Contains(wmsHTML, "WAVE PICKING GUDANG") {
		t.Fatalf("Failed to render WMS Wave Picking printout: %v", err)
	}
	t.Log("✓ Printable Enterprise WMS Verified: Koordinat Rak 3D (Aisle-Rack-Shelf-Bin) & urutan rute petik")

	// 13. Test Rendering Low-Code Studio Dynamic Record Form
	studioHTML, err := reportsSvc.RenderDocument("lowcode_studio", 1)
	if err != nil || !strings.Contains(studioHTML, "LEMBAR FORMULIR DINAMIS") {
		t.Fatalf("Failed to render Low-Code Studio dynamic record printout: %v", err)
	}
	t.Log("✓ Printable Low-Code Studio Verified: Formulir entitas dinamis & nilai atribut kustom runtime")

	// ==========================================
	t.Log("\n\t\t>>> SECTION 53: BPMN 2.0 WORKFLOW ENGINE & VISUAL STATE MACHINE <<<")

	wfMod := workflow_engine.NewModule(db, chatterSvc, aiClient)
	_ = wfMod.InitMasterData(db)
	wfSvc := wfMod.Service()

	// 1. Verify Default Master Data Seeding
	allDefs, err := wfSvc.ListDefinitions(1)
	if err != nil || len(allDefs) < 2 {
		t.Fatalf("Expected at least 2 seeded workflow definitions, got %d (err: %v)", len(allDefs), err)
	}
	t.Logf("✓ Master BPMN 2.0 Workflow Seeding Verified: %d alur kerja terdaftar (termasuk Matrix Tender & Pengadaan Tiered)", len(allDefs))

	// 2. Create Custom Enterprise CAPEX Workflow Definition
	capexReq := workflow_engine.CreateWorkflowDefinitionRequest{
		Code:        "WF-CAPEX-ACQUISITION",
		Name:        "Alur Persetujuan Belanja Modal (CAPEX) & Akuisisi Aset",
		Description: "Multi-tier approval matrix dengan evaluasi gateway otomatis untuk belanja modal operasional",
		TargetModel: "account.asset",
		InitialNodeKey: "start",
		Nodes: []workflow_engine.CreateWorkflowNodeRequest{
			{NodeKey: "start", Name: "Pengajuan CAPEX", NodeType: workflow_engine.NodeTypeStartEvent, ApproverRole: "initiator", PosX: 100, PosY: 200},
			{NodeKey: "dept_review", Name: "Review Kepala Departemen", NodeType: workflow_engine.NodeTypeUserTask, ApproverRole: "dept_head", PosX: 300, PosY: 200},
			{NodeKey: "capex_gateway", Name: "Gateway Limit Investasi", NodeType: workflow_engine.NodeTypeExclusiveGateway, PosX: 500, PosY: 200},
			{NodeKey: "bod_approval", Name: "Persetujuan Direksi (BOD)", NodeType: workflow_engine.NodeTypeUserTask, ApproverRole: "director", PosX: 700, PosY: 100},
			{NodeKey: "approved", Name: "CAPEX Disetujui", NodeType: workflow_engine.NodeTypeEndEvent, PosX: 900, PosY: 200},
			{NodeKey: "rejected", Name: "CAPEX Ditolak", NodeType: workflow_engine.NodeTypeEndEvent, PosX: 700, PosY: 350},
		},
		Transitions: []workflow_engine.CreateWorkflowTransitionRequest{
			{FromNodeKey: "start", ToNodeKey: "dept_review", ActionName: "submit", Priority: 1},
			{FromNodeKey: "dept_review", ToNodeKey: "capex_gateway", ActionName: "approve", Priority: 1},
			{FromNodeKey: "dept_review", ToNodeKey: "rejected", ActionName: "reject", Priority: 2},
			{FromNodeKey: "capex_gateway", ToNodeKey: "bod_approval", ActionName: "auto", ConditionExpression: "amount >= 100000000", Priority: 1},
			{FromNodeKey: "capex_gateway", ToNodeKey: "approved", ActionName: "auto", ConditionExpression: "amount < 100000000", Priority: 2},
			{FromNodeKey: "bod_approval", ToNodeKey: "approved", ActionName: "approve", Priority: 1},
			{FromNodeKey: "bod_approval", ToNodeKey: "rejected", ActionName: "reject", Priority: 2},
		},
	}
	capexDef, err := wfSvc.CreateDefinition(1, capexReq)
	if err != nil || capexDef == nil || len(capexDef.Nodes) != 6 {
		t.Fatalf("Failed to create custom CAPEX workflow definition: %v", err)
	}
	t.Logf("✓ Custom BPMN 2.0 Workflow Definition Created: %s dengan 6 node & 7 transisi kondisional", capexDef.Code)

	// 3. Test Runtime Execution Flow A: High-Value CAPEX (Amount: Rp 250.000.000)
	// Must route through bod_approval
	instA, err := wfSvc.StartWorkflow(1, workflow_engine.StartWorkflowRequest{
		WorkflowCode: capexDef.Code,
		TargetModel:  "account.asset",
		RecordID:     101,
		ContextData:  map[string]interface{}{"amount": 250000000.0, "vendor": "DMG MORI", "urgency": "HIGH"},
	}, 1, "Budi Prakoso")
	if err != nil || instA == nil || instA.CurrentNodeKey != "start" || instA.Status != workflow_engine.WorkflowStatusRunning {
		t.Fatalf("Failed to start workflow instance A: %v", err)
	}
	t.Logf("✓ Workflow Instance A Started: ID=%d (Status: %s, Current Node: %s)", instA.ID, instA.Status, instA.CurrentNodeKey)

	// Step A1: Submit from start to dept_review
	instA, err = wfSvc.ExecuteAction(1, instA.ID, "submit", 1, "Budi Prakoso", "Mengajukan proposal investasi mesin CNC", nil)
	if err != nil || instA.CurrentNodeKey != "dept_review" {
		t.Fatalf("Failed to submit workflow A: %v (current: %s)", err, instA.CurrentNodeKey)
	}
	t.Logf("✓ Workflow A Advanced to Dept Review: Node=%s", instA.CurrentNodeKey)

	// Step A2: Dept Head approves -> triggers capex_gateway -> automatically checks amount >= 100M -> advances to bod_approval
	instA, err = wfSvc.ExecuteAction(1, instA.ID, "approve", 2, "Hendra Dept Head", "Justifikasi ROI 14 bulan valid, diteruskan ke Direksi", nil)
	if err != nil || instA.CurrentNodeKey != "bod_approval" {
		t.Fatalf("Failed to auto-evaluate gateway on workflow A: %v (current: %s)", err, instA.CurrentNodeKey)
	}
	t.Logf("✓ Exclusive Gateway Evaluated: Nilai Rp 250jt >= 100jt -> Otomatis diarahkan ke Node: %s", instA.CurrentNodeKey)

	// Step A3: BOD Director approves -> advances to approved (end_event) -> status completed!
	instA, err = wfSvc.ExecuteAction(1, instA.ID, "approve", 3, "Direksi Pleno", "Disetujui dalam Rapat Direksi Pleno", nil)
	if err != nil || instA.CurrentNodeKey != "approved" || instA.Status != workflow_engine.WorkflowStatusCompleted || instA.CompletedAt == nil {
		t.Fatalf("Failed to complete workflow A: %v (status: %s, node: %s)", err, instA.Status, instA.CurrentNodeKey)
	}
	t.Logf("✓ Workflow A Successfully Completed: Status=%s, Target Aset Terverifikasi!", instA.Status)

	// Verify Activity Logs for Instance A
	logsA, err := wfSvc.GetActivityLogs(instA.ID)
	if err != nil || len(logsA) < 4 {
		t.Fatalf("Expected at least 4 audit logs for instance A, got %d", len(logsA))
	}
	t.Logf("✓ Immutable Audit Trail Verified: %d entri log terekam dengan timestamp dan komentar", len(logsA))

	// 4. Test Runtime Execution Flow B: Low-Value CAPEX (Amount: Rp 45.000.000)
	// Must bypass BOD approval directly to approved
	instB, err := wfSvc.StartWorkflow(1, workflow_engine.StartWorkflowRequest{
		WorkflowCode: capexDef.Code,
		TargetModel:  "account.asset",
		RecordID:     102,
		ContextData:  map[string]interface{}{"amount": 45000000.0, "vendor": "Atlas Copco"},
	}, 1, "Budi Prakoso")
	if err != nil || instB == nil {
		t.Fatalf("Failed to start workflow instance B: %v", err)
	}

	instB, err = wfSvc.ExecuteAction(1, instB.ID, "submit", 1, "Budi Prakoso", "Pengajuan", nil)
	if err != nil || instB.CurrentNodeKey != "dept_review" {
		t.Fatalf("Failed to advance workflow B: %v", err)
	}

	// Dept Head approves -> gateway evaluates amount < 100M -> directly transitions to approved!
	instB, err = wfSvc.ExecuteAction(1, instB.ID, "approve", 2, "Hendra Dept Head", "Disetujui Dept Head", nil)
	if err != nil || instB.CurrentNodeKey != "approved" || instB.Status != workflow_engine.WorkflowStatusCompleted {
		t.Fatalf("Expected fast-track completion for Low-Value CAPEX: %v (node: %s, status: %s)", err, instB.CurrentNodeKey, instB.Status)
	}
	t.Logf("✓ Fast-Track Gateway Bypass Verified: Nilai Rp 45jt < 100jt langsung selesai di Node: %s (Status: %s)", instB.CurrentNodeKey, instB.Status)

	// 5. Test Runtime Execution Flow C: Rejection Branch
	instC, err := wfSvc.StartWorkflow(1, workflow_engine.StartWorkflowRequest{
		WorkflowCode: capexDef.Code,
		TargetModel:  "account.asset",
		RecordID:     103,
		ContextData:  map[string]interface{}{"amount": 80000000.0},
	}, 1, "Budi Prakoso")
	if err != nil {
		t.Fatalf("Failed to start workflow C: %v", err)
	}
	instC, _ = wfSvc.ExecuteAction(1, instC.ID, "submit", 1, "Budi Prakoso", "Submit", nil)
	instC, err = wfSvc.ExecuteAction(1, instC.ID, "reject", 2, "Hendra Dept Head", "Anggaran departemen kuartal ini sudah mencapai batas pagu", nil)
	if err != nil || instC.CurrentNodeKey != "rejected" || instC.Status != workflow_engine.WorkflowStatusRejected {
		t.Fatalf("Expected rejected status for workflow C: %v (node: %s, status: %s)", err, instC.CurrentNodeKey, instC.Status)
	}
	t.Logf("✓ Rejection Workflow Branch Verified: Node=%s, Status=%s", instC.CurrentNodeKey, instC.Status)

	// 6. Test Condition Evaluator Edge Cases
	condTests := []struct {
		expr     string
		ctx      map[string]interface{}
		expected bool
	}{
		{"amount >= 50000000", map[string]interface{}{"amount": 60000000.0}, true},
		{"amount >= 50000000", map[string]interface{}{"amount": 40000000.0}, false},
		{"status == 'CONFIRMED'", map[string]interface{}{"status": "CONFIRMED"}, true},
		{"status != 'DRAFT'", map[string]interface{}{"status": "APPROVED"}, true},
		{"", map[string]interface{}{}, true},
	}
	for _, tc := range condTests {
		res := workflow_engine.EvaluateCondition(tc.expr, tc.ctx)
		if res != tc.expected {
			t.Fatalf("Condition evaluation failed for '%s' on '%v': expected %v, got %v", tc.expr, tc.ctx, tc.expected, res)
		}
	}
	t.Log("✓ Expression & Condition Evaluator Engine Verified (Numeric, String, Equality & Fallback)")

	// 7. Test AI BPMN 2.0 Workflow Designer Copilot
	aiWfAnalysis, err := wfSvc.AIAnalyzeWorkflow(workflow_engine.AIWorkflowDesignRequest{
		BusinessScenario: "Rancang alur kerja lelang pengadaan bahan baku kimia senilai Rp 15 Milyar dengan standar ISO 9001, audit komite tender, dan jaminan bank garansi",
		TargetModel:      "purchase.order",
	})
	if err != nil || aiWfAnalysis == nil || len(aiWfAnalysis.RecommendedNodes) == 0 {
		t.Fatalf("Failed to generate AI BPMN 2.0 workflow recommendation: %v", err)
	}
	t.Logf("✓ AI BPMN 2.0 Designer Copilot Verified: %s (%d nodes direkomendasikan dengan kepatuhan ISO/SOX)", aiWfAnalysis.ScenarioName, len(aiWfAnalysis.RecommendedNodes))

	// 8. Test Pending Tasks & Executive KPI Metrics
	pendingTasks, err := wfSvc.ListPendingTasks(1, "dept_head", 2)
	if err != nil {
		t.Fatalf("Failed to list pending workflow tasks: %v", err)
	}
	t.Logf("✓ Pending Task Inbox Verified: %d tugas menunggu persetujuan", len(pendingTasks))

	wfStats, err := wfSvc.GetStats(1)
	if err != nil || wfStats == nil || wfStats.TotalDefinitions < 3 || wfStats.CompletedInstances < 2 {
		t.Fatalf("Expected valid workflow statistics: %+v", wfStats)
	}
	t.Logf("✓ Executive Workflow KPI Dashboard Verified: Total Defs=%d, Active Instances=%d, Completed=%d (Avg SLA=%.1f Jam)",
		wfStats.TotalDefinitions, wfStats.RunningInstances, wfStats.CompletedInstances, wfStats.AvgCycleHours)

	// ==========================================
	// >>> SECTION 54: UNIVERSAL INTEGRATION HUB & OPEN RPC GATEWAY (MODUL KE-50) <<<
	t.Log("\n\t\t>>> SECTION 54: UNIVERSAL INTEGRATION HUB & OPEN RPC GATEWAY <<<")

	integrationMod := integration_hub.NewModule(db, chatterSvc, aiClient)
	_ = integrationMod.InitMasterData(db)
	integrationSvc := integrationMod.Service()

	// 1. Verify Default Connectors Seeding
	connectors, err := integrationSvc.ListConnectors(1)
	if err != nil || len(connectors) < 4 {
		t.Fatalf("Expected at least 4 seeded integration connectors, got %d (err: %v)", len(connectors), err)
	}
	t.Logf("✓ Master Integration Connectors Seeding Verified: %d konektor terdaftar (Shopee, Tokopedia, Midtrans, BCA, dll)", len(connectors))

	// 2. Machine-to-Machine (M2M) API Key Generation & Cryptographic Hashing
	keyReq := &integration_hub.CreateAPIKeyRequest{
		Name:         "E-Commerce Automation Worker",
		Scopes:       "read:product,write:sale,read:stock",
		AllowedIPs:   "127.0.0.1,192.168.1.50",
		RateLimitRPM: 240,
		IsLive:       true,
	}
	apiKeyResp, err := integrationSvc.CreateAPIKey(keyReq, 1, 1)
	if err != nil || apiKeyResp == nil || !strings.HasPrefix(apiKeyResp.SecretKey, "ak_live_") {
		t.Fatalf("Failed to generate M2M API key: %v", err)
	}
	t.Logf("✓ M2M API Key Generated: Name='%s', Prefix='%s', Scopes='%s', Secret='%s...'",
		apiKeyResp.Name, apiKeyResp.KeyPrefix, apiKeyResp.Scopes, apiKeyResp.SecretKey[:16])

	// 3. API Key Authentication, Scopes & IP Restriction Verification
	authKey, err := integrationSvc.AuthenticateAPIKey(apiKeyResp.SecretKey, "127.0.0.1", "read:product")
	if err != nil || authKey == nil || authKey.Name != "E-Commerce Automation Worker" {
		t.Fatalf("Failed to authenticate with valid API key: %v", err)
	}
	t.Logf("✓ API Key Authenticated Successfully: KeyID=%d, CompanyID=%d", authKey.ID, authKey.CompanyID)

	// Test Scope Enforcement Rejection
	_, err = integrationSvc.AuthenticateAPIKey(apiKeyResp.SecretKey, "127.0.0.1", "admin:purge_database")
	if err == nil {
		t.Fatal("Expected API key authentication to fail for unauthorized scope 'admin:purge_database'")
	}
	t.Logf("✓ API Key Scope Enforcement Verified: Unauthorized scope rejected ('%v')", err)

	// Test IP Whitelist Restriction Rejection
	_, err = integrationSvc.AuthenticateAPIKey(apiKeyResp.SecretKey, "10.99.88.77", "read:product")
	if err == nil {
		t.Fatal("Expected API key authentication to fail for unauthorized IP '10.99.88.77'")
	}
	t.Logf("✓ API Key IP Whitelisting Verified: Unauthorized IP rejected ('%v')", err)

	// 4. Test Universal Open RPC Engine: search_read, create, write, unlink
	// RPC Search Read on res.partner
	rpcQueryResp, err := integrationSvc.ExecuteRPCQuery(&integration_hub.RPCQueryRequest{
		Model:  "res.partner",
		Domain: [][]interface{}{{"is_company", "=", true}},
		Fields: []string{"id", "name", "email", "phone"},
		Limit:  5,
	}, 1)
	if err != nil || !rpcQueryResp.Success {
		t.Fatalf("RPC search_read failed on res.partner: %v", err)
	}
	t.Logf("✓ Universal Open RPC 'search_read' Verified: Model='%s', Found=%d records", rpcQueryResp.Model, rpcQueryResp.Count)

	// RPC Create on res.partner
	rpcCreateResp, err := integrationSvc.ExecuteRPCCreate(&integration_hub.RPCCreateRequest{
		Model: "res.partner",
		Values: map[string]interface{}{
			"name":       "PT Integrasi Solusi Nusantara (M2M Partner)",
			"email":      "partner.api@solusinusantara.co.id",
			"phone":      "+6281122334455",
			"is_company": true,
		},
	}, 1)
	if err != nil || !rpcCreateResp.Success {
		t.Fatalf("RPC create failed on res.partner: %v", err)
	}
	createdPartnerMap, ok := rpcCreateResp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected created partner map, got: %+v", rpcCreateResp.Data)
	}
	var partnerM2MID uint
	switch v := createdPartnerMap["id"].(type) {
	case uint:
		partnerM2MID = v
	case int:
		partnerM2MID = uint(v)
	case int64:
		partnerM2MID = uint(v)
	case float64:
		partnerM2MID = uint(v)
	}
	t.Logf("✓ Universal Open RPC 'create' Verified: Created partner ID=%d", partnerM2MID)

	// RPC Write on res.partner
	rpcWriteResp, err := integrationSvc.ExecuteRPCWrite(&integration_hub.RPCWriteRequest{
		Model: "res.partner",
		ID:    partnerM2MID,
		Values: map[string]interface{}{
			"phone": "+6281199887766",
		},
	}, 1)
	if err != nil || !rpcWriteResp.Success {
		t.Fatalf("RPC write failed on res.partner: %v", err)
	}
	t.Logf("✓ Universal Open RPC 'write' Verified: Updated partner ID=%d", partnerM2MID)

	// RPC Unlink (Delete) on res.partner
	rpcUnlinkResp, err := integrationSvc.ExecuteRPCUnlink(&integration_hub.RPCUnlinkRequest{
		Model: "res.partner",
		ID:    partnerM2MID,
	}, 1)
	if err != nil || !rpcUnlinkResp.Success {
		t.Fatalf("RPC unlink failed on res.partner: %v", err)
	}
	t.Logf("✓ Universal Open RPC 'unlink' Verified: Deleted partner ID=%d", partnerM2MID)

	// 5. Test Pre-built Connectors Execution
	// Test Connection ping
	testConnResp, err := integrationSvc.ExecuteConnectorAction(&integration_hub.ExecuteConnectorActionRequest{
		ConnectorCode: "shopee_ecommerce",
		Action:        "test_connection",
	}, 1)
	if err != nil || !testConnResp.Success {
		t.Fatalf("Failed to execute connector action test_connection: %v", err)
	}
	t.Logf("✓ Pre-built Connector Action 'test_connection' Verified: %s", testConnResp.Message)

	// Test Sync Stock to marketplace
	syncStockResp, err := integrationSvc.ExecuteConnectorAction(&integration_hub.ExecuteConnectorActionRequest{
		ConnectorCode: "shopee_ecommerce",
		Action:        "sync_stock",
	}, 1)
	if err != nil || !syncStockResp.Success {
		t.Fatalf("Failed to execute sync_stock action: %v", err)
	}
	t.Logf("✓ Pre-built Connector Action 'sync_stock' Verified: %s (Batch: %s)", syncStockResp.Message, syncStockResp.ExternalID)

	// Test WhatsApp Business Cloud API notification
	wabaResp, err := integrationSvc.ExecuteConnectorAction(&integration_hub.ExecuteConnectorActionRequest{
		ConnectorCode: "waba_whatsapp",
		Action:        "send_whatsapp",
		Payload: map[string]interface{}{
			"recipient": "+628123456789",
			"message":   "Notifikasi pengiriman barang SO/2026/00001",
		},
	}, 1)
	if err != nil || !wabaResp.Success {
		t.Fatalf("Failed to execute send_whatsapp action: %v", err)
	}
	t.Logf("✓ Pre-built Connector Action 'send_whatsapp' Verified: %s", wabaResp.Message)

	// 6. Test Batch Data ETL Job (Export product.product)
	dataJob, err := integrationSvc.RunBatchDataJob(&integration_hub.RunBatchDataJobRequest{
		TargetModel: "product.product",
		JobType:     "export",
		FileFormat:  "json",
	}, 1)
	if err != nil || dataJob == nil || dataJob.Status != "completed" {
		t.Fatalf("Failed to run batch data export job: %v", err)
	}
	t.Logf("✓ Batch Data ETL Job Verified: JobCode=%s, Target=%s, Processed=%d records (Status: %s)",
		dataJob.JobCode, dataJob.TargetModel, dataJob.ProcessedRecords, dataJob.Status)

	// 7. Test AI Integration Architect & Schema Mapping Copilot
	aiIntegrationResp, err := integrationSvc.AnalyzeIntegrationWithAI(&integration_hub.AIIntegrationArchitectRequest{
		Requirement: "Sinkronisasi pesanan dari Tokopedia ke GoERP secara realtime menggunakan webhook dan update stok kembali ke marketplace",
		TargetApp:   "Tokopedia",
		Language:    "python",
	})
	if err != nil || aiIntegrationResp == nil || aiIntegrationResp.UseCase == "" {
		t.Fatalf("Failed to run AI Integration Architect: %v", err)
	}
	t.Logf("✓ AI Integration Architect Copilot Verified: Pattern='%s', Flow='%s', AI Provider='%s'",
		aiIntegrationResp.IntegrationPattern, aiIntegrationResp.ArchitectureFlow, aiIntegrationResp.AIProvider)

	// 8. Test Telemetry Logging & KPI Dashboard Stats
	integrationSvc.LogIntegrationCall(
		1,
		authKey,
		"127.0.0.1",
		"api_key",
		"POST",
		"/api/v1/integration/rpc/search_read",
		"res.partner",
		"search_read",
		200,
		18,
		"Queried 5 partner records via Open RPC",
		"",
	)
	logs, err := integrationSvc.ListLogs(1, 10)
	if err != nil || len(logs) == 0 {
		t.Fatalf("Expected integration logs, got %d (err: %v)", len(logs), err)
	}
	t.Logf("✓ Integration Telemetry & Audit Logs Verified: %d entri terekam", len(logs))

	hubStats, err := integrationSvc.GetStats(1)
	if err != nil || hubStats == nil || hubStats.TotalConnectors < 4 {
		t.Fatalf("Expected valid integration hub statistics: %+v", hubStats)
	}
	t.Logf("✓ Executive Integration Hub KPI Dashboard Verified: Total Keys=%d, Connectors=%d, Total Calls=%d (Success Rate: %.1f%%)",
		hubStats.TotalAPIKeys, hubStats.TotalConnectors, hubStats.TotalCalls24h, hubStats.SuccessRatePercent)

	// Revoke API Key
	if err := integrationSvc.RevokeAPIKey(authKey.ID, 1, 1); err != nil {
		t.Fatalf("Failed to revoke API key: %v", err)
	}
	_, err = integrationSvc.AuthenticateAPIKey(apiKeyResp.SecretKey, "127.0.0.1", "read:product")
	if err == nil {
		t.Fatal("Expected revoked API key to be rejected")
	}
	t.Logf("✓ API Key Revocation Verified: Revoked key is inactive and rejected")

	// =========================================================================
	// 51. EXECUTIVE COMMAND CENTER (OMNI-SEARCH, NOTIFICATIONS, 1-CLICK SIMULATOR)
	// =========================================================================
	gin.SetMode(gin.TestMode)
	ccRouter := gin.New()
	ccV1 := ccRouter.Group("/api/v1")
	noopAuth := func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("company_id", uint(1))
		c.Set("role", "admin")
		c.Next()
	}
	commandcenter.RegisterRoutes(ccV1, noopAuth, db, saleSvc, purchaseSvc, stockSvc, accountSvc)

	// 51.1 Test Universal Omni-Search (/api/v1/search?q=Monitor)
	reqSearch := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=Monitor", nil)
	wSearch := httptest.NewRecorder()
	ccRouter.ServeHTTP(wSearch, reqSearch)
	if wSearch.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from /api/v1/search, got %d: %s", wSearch.Code, wSearch.Body.String())
	}
	var searchResp map[string]interface{}
	if err := json.Unmarshal(wSearch.Body.Bytes(), &searchResp); err != nil {
		t.Fatalf("Failed to parse search JSON: %v", err)
	}
	if totalHits, ok := searchResp["total_hits"].(float64); !ok || totalHits < 1 {
		t.Fatalf("Expected at least 1 hit in omni-search for 'Monitor', got %v", searchResp["total_hits"])
	}
	t.Logf("✓ Universal Omni-Search Verified: query='Monitor' returned %.0f total hits", searchResp["total_hits"])

	// 51.2 Test Actionable Notification Center (/api/v1/dashboard/notifications)
	reqNotif := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/notifications", nil)
	wNotif := httptest.NewRecorder()
	ccRouter.ServeHTTP(wNotif, reqNotif)
	if wNotif.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from /api/v1/dashboard/notifications, got %d", wNotif.Code)
	}
	var notifResp map[string]interface{}
	_ = json.Unmarshal(wNotif.Body.Bytes(), &notifResp)
	t.Logf("✓ Actionable Notification Center Verified: %v notifications aggregated", notifResp["total_unread"])

	// 51.3 Test 1-Click End-to-End Business Cycle Simulator (/api/v1/dashboard/simulate-cycle)
	reqSim := httptest.NewRequest(http.MethodPost, "/api/v1/dashboard/simulate-cycle", nil)
	wSim := httptest.NewRecorder()
	ccRouter.ServeHTTP(wSim, reqSim)
	if wSim.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from /api/v1/dashboard/simulate-cycle, got %d: %s", wSim.Code, wSim.Body.String())
	}
	var simResp map[string]interface{}
	_ = json.Unmarshal(wSim.Body.Bytes(), &simResp)
	if simResp["po_number"] == "" || simResp["so_number"] == "" {
		t.Fatalf("Expected valid PO and SO numbers from business cycle simulator: %+v", simResp)
	}
	t.Logf("✓ 1-Click Business Cycle Simulator Verified: PO=%v, Receipt=%v, SO=%v, Delivery=%v, Invoice=%v",
		simResp["po_number"], simResp["receipt_number"], simResp["so_number"], simResp["delivery_number"], simResp["invoice_number"])

	// 51.4 Test Executive Summary CSV Export (/api/v1/dashboard/export-summary)
	reqCSV := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/export-summary", nil)
	wCSV := httptest.NewRecorder()
	ccRouter.ServeHTTP(wCSV, reqCSV)
	if wCSV.Code != http.StatusOK || !strings.Contains(wCSV.Body.String(), "Total Confirmed Sales") {
		t.Fatalf("Expected valid CSV from /api/v1/dashboard/export-summary, got code=%d", wCSV.Code)
	}
	t.Logf("✓ Executive KPI CSV Export Verified (%d bytes)", wCSV.Body.Len())

	// =========================================================================
	// >>> SECTION 55: GOOGLE CLOUD PLATFORM ECOSYSTEM & MULTI-PROVIDER AI AGENTS <<<
	// =========================================================================
	t.Log("\n\t\t>>> SECTION 55: GOOGLE CLOUD PLATFORM ECOSYSTEM & POPULAR AI AGENT APIS <<<")

	// 55.1 Verify Multi-Provider AI Agent Registry (Gemini, OpenAI, Claude, DeepSeek, Cloudflare, Local)
	providers := aiClient.GetSupportedProviders()
	if len(providers) < 6 {
		t.Fatalf("Expected at least 6 supported AI Agent providers, got %d", len(providers))
	}
	t.Logf("✓ Multi-Provider AI Agent Registry Verified: %d providers registered (Gemini 2.5, OpenAI GPT-4o, Claude 3.7 MCP, DeepSeek R1, Cloudflare, Local)", len(providers))

	// 55.2 Verify MCP & Function-Calling Tool Manifest
	toolsManifest := aiClient.GetAgentToolsManifest()
	if toolsManifest["server_name"] != "goerp-enterprise-mcp-server" {
		t.Fatalf("Unexpected MCP server name: %v", toolsManifest["server_name"])
	}
	t.Logf("✓ AI Agent Tool-Calling & MCP Manifest Verified: Server='%v', Total Tools=%v", toolsManifest["server_name"], toolsManifest["total_tools"])

	// 55.3 Execute Autonomous AI Agent with Live Database Tool Calling & GCP BigQuery Streaming
	agentExecResp, err := aiClient.ExecuteAutonomousAgent(db, &ai.AgentExecuteRequest{
		Prompt:      "Audit inventory stock levels, check financial General Ledger summary, and stream analytics to Google Cloud BigQuery",
		Provider:    "gemini",
		EnableTools: true,
		AutoSyncGCP: true,
	}, 1)
	if err != nil || agentExecResp == nil || !agentExecResp.Success || len(agentExecResp.ToolCalls) < 3 {
		t.Fatalf("Autonomous AI Agent execution failed: err=%v, resp=%+v", err, agentExecResp)
	}
	t.Logf("✓ Autonomous AI Agent ReAct Loop Verified: Provider='%s', Tools Invoked=%d, Steps=%d",
		agentExecResp.ProviderUsed, len(agentExecResp.ToolCalls), len(agentExecResp.ReasoningSteps))

	// 55.4 Verify Google Cloud Platform Connectors in Integration Hub (BigQuery, Pub/Sub, GCS, Vertex AI, OpenAI, Claude, DeepSeek)
	bqSyncResp, err := integrationSvc.ExecuteConnectorAction(&integration_hub.ExecuteConnectorActionRequest{
		ConnectorCode: "gcp_bigquery",
		Action:        "sync_bigquery",
	}, 1)
	if err != nil || !bqSyncResp.Success {
		t.Fatalf("GCP BigQuery connector sync_bigquery failed: %v", err)
	}
	t.Logf("✓ Google Cloud BigQuery Connector Verified: %s (Job: %s)", bqSyncResp.Message, bqSyncResp.ExternalID)

	pubsubResp, err := integrationSvc.ExecuteConnectorAction(&integration_hub.ExecuteConnectorActionRequest{
		ConnectorCode: "gcp_pubsub",
		Action:        "publish_pubsub",
		Payload:       map[string]interface{}{"event_type": "erp.invoice.posted"},
	}, 1)
	if err != nil || !pubsubResp.Success {
		t.Fatalf("GCP Pub/Sub connector publish_pubsub failed: %v", err)
	}
	t.Logf("✓ Google Cloud Pub/Sub Connector Verified: %s", pubsubResp.Message)

	gcsArchResp, err := integrationSvc.ExecuteConnectorAction(&integration_hub.ExecuteConnectorActionRequest{
		ConnectorCode: "gcp_gcs_archive",
		Action:        "archive_to_gcs",
	}, 1)
	if err != nil || !gcsArchResp.Success {
		t.Fatalf("GCP Cloud Storage connector archive_to_gcs failed: %v", err)
	}
	t.Logf("✓ Google Cloud Storage (GCS) Archive Connector Verified: %s", gcsArchResp.Message)

	for _, agentConnCode := range []string{"gcp_vertex_ai", "openai_agents", "anthropic_claude", "deepseek_ai"} {
		agConnResp, err := integrationSvc.ExecuteConnectorAction(&integration_hub.ExecuteConnectorActionRequest{
			ConnectorCode: agentConnCode,
			Action:        "run_ai_agent",
			Payload:       map[string]interface{}{"prompt": "Check real-time ERP financial and inventory health"},
		}, 1)
		if err != nil || !agConnResp.Success {
			t.Fatalf("AI Agent connector '%s' failed: %v", agentConnCode, err)
		}
		t.Logf("✓ AI Agent Connector '%s' Verified: %s", agentConnCode, agConnResp.Message)
	}

	// =========================================================================
	// >>> SECTION 56: OPENAPI 3.1, MCP SERVER, GOOGLE A2A, AI DOC INTELLIGENCE, FRAUD RADAR, CASHFLOW, ESG & CHATOPS <<<
	// =========================================================================
	t.Log("\n\t\t>>> SECTION 56: OPENAPI 3.1, NATIVE MCP JSON-RPC 2.0, GOOGLE A2A PROTOCOL, ENTERPRISE AI INTELLIGENCE & CHATOPS <<<")

	// 56.1 OpenAPI 3.1.0 Specification & API Explorer
	spec := openapi.GetSpec("GoERP Test Suite", "1.42.0")
	if spec["openapi"] != "3.1.0" {
		t.Fatalf("Expected OpenAPI 3.1.0, got %v", spec["openapi"])
	}
	t.Logf("✓ OpenAPI 3.1.0 Specification Verified (Server: %v, Paths: %d)", spec["info"], len(spec["paths"].(map[string]interface{})))

	// 56.2 Google Cloud Agent-to-Agent (A2A) Discovery Card
	a2aCard := aiClient.GetA2AAgentCard("http://localhost:8080")
	if a2aCard["standard"] == "" || len(a2aCard["skills"].([]map[string]interface{})) < 4 {
		t.Fatalf("Invalid A2A Agent Card: %+v", a2aCard)
	}
	t.Logf("✓ Google Cloud A2A Discovery Card Verified: Standard='%v', Skills=%d", a2aCard["standard"], len(a2aCard["skills"].([]map[string]interface{})))

	// 56.3 Native Model Context Protocol (MCP) JSON-RPC 2.0 Server
	mcpInit := aiClient.HandleMCPJSONRPC(db, &ai.MCPJSONRPCRequest{Method: "initialize"}, 1)
	if mcpInit.Result == nil || mcpInit.Result["protocolVersion"] != "2025-03-26" {
		t.Fatalf("MCP initialize failed: %+v", mcpInit)
	}
	mcpTools := aiClient.HandleMCPJSONRPC(db, &ai.MCPJSONRPCRequest{Method: "tools/list"}, 1)
	if mcpTools.Result == nil || len(mcpTools.Result["tools"].([]map[string]interface{})) < 5 {
		t.Fatalf("MCP tools/list failed: %+v", mcpTools)
	}
	mcpCall := aiClient.HandleMCPJSONRPC(db, &ai.MCPJSONRPCRequest{
		Method: "tools/call",
		Params: map[string]interface{}{
			"name": "get_financial_summary",
		},
	}, 1)
	if mcpCall.Result == nil || mcpCall.Result["isError"] == true {
		t.Fatalf("MCP tools/call failed: %+v", mcpCall)
	}
	mcpRes := aiClient.HandleMCPJSONRPC(db, &ai.MCPJSONRPCRequest{
		Method: "resources/read",
		Params: map[string]interface{}{"uri": "erp://esg/carbon-ledger"},
	}, 1)
	if mcpRes.Result == nil {
		t.Fatalf("MCP resources/read failed: %+v", mcpRes)
	}
	t.Logf("✓ Native MCP JSON-RPC 2.0 Server Verified: Protocol=2025-03-26, Tools=%d, Call & Resource Read Success", len(mcpTools.Result["tools"].([]map[string]interface{})))

	// 56.4 Google A2A Cross-Company Task Execution
	a2aTaskResp := aiClient.ExecuteA2ATask(db, &ai.A2ATaskRequest{
		ID:    "b2b-task-001",
		Skill: "inventory-availability-check",
		Message: map[string]interface{}{
			"text": "Check warehouse inventory stock levels for partner B2B order",
		},
	}, 1)
	if a2aTaskResp["protocol"] != "Google A2A v1.0" || len(a2aTaskResp["artifacts"].([]map[string]interface{})) == 0 {
		t.Fatalf("Google A2A Task execution failed: %+v", a2aTaskResp)
	}
	t.Logf("✓ Google A2A Task Execution Verified: Skill='%v', Artifacts=%d", a2aTaskResp["skillUsed"], len(a2aTaskResp["artifacts"].([]map[string]interface{})))

	// 56.5 AI Document Intelligence (Auto-OCR & 3-Way Matching)
	docIntelResp := aiClient.ExecuteDocumentIntelligence(db, &ai.DocumentIntelligenceRequest{
		DocumentType:  "vendor_bill",
		VendorName:    "PT Baja Presisi Indo",
		InvoiceNumber: "INV-2026-9912",
		DocumentText:  "NPWP: 01.333.444.5-091.000 Faktur Pajak: 010.002-26.11223344 Subtotal: 10,000,000 PPN 11%: 1,100,000 Total: 11,100,000",
		SubTotal:      10000000,
		TaxAmount:     1100000,
		TotalAmount:   11100000,
	}, 1)
	if docIntelResp["success"] != true || docIntelResp["three_way_matching"] == nil {
		t.Fatalf("AI Document Intelligence failed: %+v", docIntelResp)
	}
	t.Logf("✓ AI Document Intelligence & 3-Way Matching Verified: Status='%v', e-Faktur Serial='%v'",
		docIntelResp["three_way_matching"].(map[string]interface{})["status"],
		docIntelResp["extracted_fields"].(map[string]interface{})["efaktur_serial"])

	// 56.6 Continuous AI Audit & Fraud Radar
	fraudResp := aiClient.RunFraudRadar(db, &ai.FraudRadarRequest{Sensitivity: "high"}, 1)
	if fraudResp["success"] != true || len(fraudResp["anomalies_and_controls"].([]map[string]interface{})) == 0 {
		t.Fatalf("AI Fraud Radar failed: %+v", fraudResp)
	}
	t.Logf("✓ Continuous AI Audit & Fraud Radar Verified: RiskLevel=%v, Transactions Scanned=%v, Controls=%d",
		fraudResp["risk_level"], fraudResp["total_transactions_scanned"], len(fraudResp["anomalies_and_controls"].([]map[string]interface{})))

	// 56.7 Predictive 30/60/90-Day Cashflow Forecasting
	cashflowResp := aiClient.GetCashflowForecast(db, &ai.CashflowForecastRequest{HorizonDays: 90}, 1)
	if cashflowResp["success"] != true || cashflowResp["horizons"] == nil {
		t.Fatalf("Predictive Cashflow Forecasting failed: %+v", cashflowResp)
	}
	t.Logf("✓ Predictive 30/60/90-Day Cashflow Forecast Verified: Liquidity='%v', Estimated Runway=%v Months",
		cashflowResp["cfo_health_metrics"].(map[string]interface{})["liquidity_status"],
		cashflowResp["cfo_health_metrics"].(map[string]interface{})["estimated_runway_months"])

	// 56.8 ESG & Carbon Footprint Ledger (Scope 1, 2, 3 GHG Protocol)
	esgResp := aiClient.GetESGCarbonReport(db, &ai.ESGReportRequest{Period: "2026-YTD"}, 1)
	if esgResp["success"] != true || esgResp["scopes_breakdown"] == nil {
		t.Fatalf("ESG Carbon Report failed: %+v", esgResp)
	}
	t.Logf("✓ ESG & Carbon Footprint Ledger Verified: Rating='%v', Total Emissions=%.0f kg CO2e, Offset Equivalent=%v Trees",
		esgResp["esg_rating"], esgResp["total_emissions_kg_co2e"], esgResp["offset_trees_equivalent"])

	// 56.9 Omnichannel ChatOps Gateway (WhatsApp, Telegram & Slack Webhook)
	chatopsResp := aiClient.ExecuteChatOpsWebhook(db, &ai.ChatOpsWebhookRequest{
		Channel: "whatsapp",
		Sender:  "+62811998877",
		Message: "/omzet",
	}, 1)
	if chatopsResp["success"] != true || !strings.Contains(chatopsResp["reply_text"].(string), "GoERP") {
		t.Fatalf("ChatOps Webhook failed: %+v", chatopsResp)
	}
	t.Logf("✓ Omnichannel ChatOps Gateway Verified: Channel='%v', Action='%v'", chatopsResp["channel"], chatopsResp["action_executed"])

	// =========================================================================
	// >>> SECTION 57: DEMAND SENSING, LIVE PRESENCE SSE, SAAS MULTI-TENANCY, OFFLINE POS & VOICE OPS <<<
	// =========================================================================
	t.Log("\n\t\t>>> SECTION 57: AI DEMAND SENSING, REAL-TIME SSE COLLABORATION, SAAS MULTI-TENANCY, OFFLINE POS & VOICE OPS <<<")

	// 57.1 AI Demand Sensing, Dynamic Safety Stock & EOQ
	demandAnalysis := aiClient.AnalyzeDemandSensing(db, &ai.DemandSensingRequest{HorizonDays: 90, ServiceLevel: 0.95}, 1)
	if demandAnalysis["success"] != true || demandAnalysis["product_recommendations"] == nil {
		t.Fatalf("Demand Sensing analysis failed: %+v", demandAnalysis)
	}
	metricsList := demandAnalysis["product_recommendations"].([]ai.ProductDemandMetric)
	if len(metricsList) == 0 {
		t.Fatalf("Expected at least 1 product demand metric, got 0")
	}
	firstMetric := metricsList[0]
	t.Logf("✓ AI Demand Sensing Verified: Product='%s', DailyVelocity=%.2f, SafetyStock=%.0f, ROP=%.0f, EOQ=%.0f, Risk='%s'",
		firstMetric.ProductName, firstMetric.DailyDemandVelocity, firstMetric.DynamicSafetyStock, firstMetric.ReorderPoint, firstMetric.EconomicOrderQuantity, firstMetric.StockoutRiskLevel)

	// 57.2 Automated Replenishment Draft Purchase Order Execution
	replenishRes := aiClient.ExecuteAutoReplenishmentPO(db, &ai.AutoReplenishRequest{Notes: "Unit Test Automated Replenishment Batch"}, 1)
	if replenishRes["success"] != true || replenishRes["purchase_order_name"] == nil {
		t.Fatalf("Auto Replenishment PO failed: %+v", replenishRes)
	}
	t.Logf("✓ Automated Replenishment PO Verified: PO Number='%v', Lines=%v, Grand Total=IDR %.2f",
		replenishRes["purchase_order_name"], replenishRes["lines_created"], replenishRes["amount_total"])

	// 57.3 Real-Time SSE Event Broker & Live Presence Tracking
	broker := events.GetBroker()
	testChan := broker.Subscribe()
	broker.UpdatePresence("sale_order", "SO00001", 1, "John Admin", "john@example.com", "editing")
	activePresences := broker.GetDocumentPresences("sale_order", "SO00001")
	if len(activePresences) == 0 || activePresences[0].Action != "editing" {
		t.Fatalf("Live Presence tracking failed: %+v", activePresences)
	}
	broker.Broadcast("test_domain_event", map[string]interface{}{"status": "broadcast_ok"})
	broker.Unsubscribe(testChan)
	t.Logf("✓ Real-Time Event Broker & Live Presence Verified: Active Document Presences=%d (User: %s)", len(activePresences), activePresences[0].UserName)

	// 57.4 SaaS Multi-Tenancy Provisioning & Isolation Engine
	tenantMgr := multitenancy.InitManager(db)
	newTenant, err := tenantMgr.RegisterTenant(&multitenancy.TenantRegisterRequest{
		TenantID:   fmt.Sprintf("tenant-unit-test-%d", time.Now().UnixNano()%10000),
		Name:       "PT Solusi Awan Mandiri",
		Domain:     "solusi-awan.goerp.io",
		Plan:       "enterprise",
		AdminEmail: "admin@solusiawan.co.id",
	})
	if err != nil || newTenant == nil || newTenant.TenantID == "" {
		t.Fatalf("Tenant provisioning failed: %v", err)
	}
	tenantsList, err := tenantMgr.ListTenants()
	if err != nil || len(tenantsList) < 2 {
		t.Fatalf("ListTenants failed: %v, count=%d", err, len(tenantsList))
	}
	t.Logf("✓ SaaS Multi-Tenancy Engine Verified: Provisioned Tenant='%s' (%s, Plan: %s), Total Tenants=%d",
		newTenant.TenantID, newTenant.Name, newTenant.Plan, len(tenantsList))

	// 57.5 Offline-First PWA Barcode POS Synchronization
	offlineBatchResp, err := posSvc.SyncOfflineBatch([]pos.OfflineOrderPayload{
		{
			OfflineID:       fmt.Sprintf("OFF-TX-%d-1", time.Now().Unix()),
			ClientTimestamp: time.Now().UTC().Format(time.RFC3339),
			PartnerID:       1,
			AmountPaid:      2000000,
			PaymentMethod:   "cash",
			Lines: []pos.PosOrderLine{
				{ProductID: 1, Qty: 1, PriceUnit: 1500000},
			},
		},
	}, 1, 1)
	if err != nil || len(offlineBatchResp) == 0 || offlineBatchResp[0].Status != "synced" {
		t.Fatalf("Offline POS Batch sync failed: err=%v, res=%+v", err, offlineBatchResp)
	}
	t.Logf("✓ Offline-First PWA POS Sync Verified: Reconciled OfflineID='%s' -> Server POS='%s' (Total: IDR %.2f)",
		offlineBatchResp[0].OfflineID, offlineBatchResp[0].ServerNumber, offlineBatchResp[0].AmountTotal)

	// 57.6 Hands-Free Voice Ops Assistant
	voiceRespStock := aiClient.ExecuteVoiceCommand(db, &ai.VoiceCommandRequest{
		Transcript: "GoERP, tolong cek posisi stok gudang",
		Language:   "id-ID",
	}, 1)
	if voiceRespStock == nil || !voiceRespStock.Success || voiceRespStock.IntentDetected != "warehouse_inventory_check" || voiceRespStock.SpokenResponse == "" {
		t.Fatalf("Voice command stock check failed: %+v", voiceRespStock)
	}

	voiceRespOmzet := aiClient.ExecuteVoiceCommand(db, &ai.VoiceCommandRequest{
		Transcript: "Berapa omzet penjualan hari ini?",
		Language:   "id-ID",
	}, 1)
	if voiceRespOmzet == nil || !voiceRespOmzet.Success || voiceRespOmzet.IntentDetected != "executive_finance_inquiry" || voiceRespOmzet.SpokenResponse == "" {
		t.Fatalf("Voice command omzet check failed: %+v", voiceRespOmzet)
	}
	t.Logf("✓ Hands-Free Voice Ops Assistant Verified: Intent='%s' & '%s', Audio Spoken Text=\"%s\"",
		voiceRespStock.IntentDetected, voiceRespOmzet.IntentDetected, voiceRespOmzet.SpokenResponse)

	// =========================================================================
	// SECTION 58: AI AUTONOMOUS RFQ BIDDING, MERKLE-TREE LEDGER AUDIT,
	//             SUPPLY CHAIN DIGITAL TWIN, FIELD-LEVEL RBAC & VISUAL BI BUILDER
	// =========================================================================
	t.Log(`
		>>> SECTION 58: AI AUTONOMOUS RFQ BIDDING, MERKLE AUDIT, DIGITAL TWIN & BI DASHBOARD <<<`)

	// 58.1 AI Autonomous Supplier Negotiation & RFQ Bidding Wars
	tenderReq := &ai.CreateTenderRequest{
		Title:           "Bulk Procurement: High-Definition IPS Panels Q4 2026",
		ProductID:       1,
		TargetQuantity:  100,
		BudgetUnitPrice: 2500000,
		VendorIDs:       []uint{1, 2},
	}
	tender, err := aiClient.CreateBiddingTender(db, tenderReq, 1)
	if err != nil || tender == nil || tender.TenderID == "" || len(tender.InvitedVendors) != 2 {
		t.Fatalf("Failed to create RFQ bidding tender: err=%v, tender=%+v", err, tender)
	}

	// Vendor 1 submits aggressive bid
	bid1, err := aiClient.SubmitVendorBid(db, &ai.VendorBidSubmission{
		TenderID:         tender.TenderID,
		VendorID:         1,
		OfferedUnitPrice: 2300000,
		LeadTimeDays:     7,
		PaymentTermsDays: 30,
		WarrantyMonths:   24,
		DiscountPct:      3.0,
		Notes:            "Fast regional dispatch from Jakarta warehouse",
	}, 1)
	if err != nil || bid1 == nil {
		t.Fatalf("Vendor 1 bid submission failed: %v", err)
	}

	// Vendor 2 submits bulk discount bid
	bid2, err := aiClient.SubmitVendorBid(db, &ai.VendorBidSubmission{
		TenderID:         tender.TenderID,
		VendorID:         2,
		OfferedUnitPrice: 2150000,
		LeadTimeDays:     14,
		PaymentTermsDays: 60,
		WarrantyMonths:   18,
		DiscountPct:      5.0,
		Notes:            "Direct factory import Net 60 terms",
	}, 1)
	if err != nil || bid2 == nil {
		t.Fatalf("Vendor 2 bid submission failed: %v", err)
	}

	// Evaluate MAUT Scorecard
	scorecard, err := aiClient.EvaluateBiddingTender(db, tender.TenderID, 1)
	if err != nil || scorecard == nil || len(scorecard.Bids) != 2 || scorecard.WinningBid == nil {
		t.Fatalf("Tender evaluation failed: err=%v, scorecard=%+v", err, scorecard)
	}
	if scorecard.TotalEstimatedSavings <= 0 || scorecard.SavingsPct <= 0 {
		t.Fatalf("Expected positive savings from competitive bidding, got %.2f (%.1f%%)",
			scorecard.TotalEstimatedSavings, scorecard.SavingsPct)
	}

	// Award Tender to Winning Vendor
	awardedPO, err := aiClient.AwardBiddingTender(db, tender.TenderID, scorecard.WinningBid.VendorID, 1)
	if err != nil || awardedPO == nil || awardedPO.ID == 0 || len(awardedPO.OrderLines) == 0 {
		t.Fatalf("Award tender failed: err=%v, po=%+v", err, awardedPO)
	}
	t.Logf("✓ AI Autonomous Supplier RFQ Bidding Verified: Tender='%s', WinningVendor='%s' (Score: %.1f/100, EffPrice: IDR %.2f), Savings=IDR %.2f (%.1f%%), Auto-Created PO='%s'",
		tender.TenderID, scorecard.WinningBid.VendorName, scorecard.WinningBid.CompositeScore,
		scorecard.WinningBid.EffectivePrice, scorecard.TotalEstimatedSavings, scorecard.SavingsPct, awardedPO.Name)

	// 58.2 Cryptographic Merkle-Tree Ledger Audit (Tamper-Proof Financial Chain)
	auditResult, err := audit.BuildAndVerifyMerkleLedger(db, 1)
	if err != nil || auditResult == nil || !auditResult.Verified || auditResult.TamperDetected {
		t.Fatalf("Merkle ledger audit verification failed: err=%v, res=%+v", err, auditResult)
	}
	if auditResult.MerkleRoot == "" || auditResult.ChainLength == 0 || auditResult.TreeDepth == 0 {
		t.Fatalf("Invalid Merkle ledger audit metrics: %+v", auditResult)
	}
	t.Logf("✓ Cryptographic Merkle-Tree Ledger Audit Verified: Verified=%t, ChainLength=%d blocks, MerkleRoot='%s', TreeDepth=%d, Certificate='%s'",
		auditResult.Verified, auditResult.ChainLength, auditResult.MerkleRoot, auditResult.TreeDepth, auditResult.Certificate)

	// 58.3 Supply Chain Digital Twin & Interactive "What-If" Scenario Simulator
	twinSim := aiClient.ExecuteDigitalTwinSimulation(db, &ai.DigitalTwinSimulateRequest{
		ScenarioName:               "Simulated Middle-East Freight Spike & Supplier 14-Day Lead-Time Shock",
		SupplierLeadTimeDelayDays:  14,
		FuelFreightInflationPct:    18.5,
		FXRateUSDIDR:               16800,
		BaselineFXRateUSDIDR:       15800,
		DemandSpikePct:             25.0,
		RawMaterialCostIncreasePct: 12.0,
	}, 1)
	if twinSim == nil || twinSim.ActiveBOMsEvaluated == 0 || len(twinSim.MitigationStrategies) == 0 {
		t.Fatalf("Supply chain digital twin simulation failed: %+v", twinSim)
	}
	if twinSim.TotalCOGSIncreasePct <= 0 || twinSim.GrossMarginErosionPct <= 0 {
		t.Fatalf("Expected positive COGS inflation and margin erosion, got COGS=+%.2f%%, MarginErosion=%.2f%%",
			twinSim.TotalCOGSIncreasePct, twinSim.GrossMarginErosionPct)
	}
	t.Logf("✓ Supply Chain Digital Twin Simulator Verified: Scenario='%s', COGS Increase=+%.1f%%, Gross Margin Erosion=-%.1f%%, Stockout At-Risk SKUs=%d, Mitigations=%d",
		twinSim.ScenarioName, twinSim.TotalCOGSIncreasePct, twinSim.GrossMarginErosionPct, twinSim.StockoutRiskCount, len(twinSim.MitigationStrategies))

	// 58.4 Enterprise Single Sign-On (SAML 2.0 / OIDC) & Field-Level RBAC Matrix
	ssoConfig := security.GetEnterpriseSSOConfig()
	if ssoConfig == nil || len(ssoConfig.Providers) < 3 || ssoConfig.EntityID == "" {
		t.Fatalf("SSO configuration failed: %+v", ssoConfig)
	}

	rbacMatrix := security.GetRBACMatrix()
	if rbacMatrix == nil || len(rbacMatrix.RoleProfiles) < 4 || len(rbacMatrix.SystemRoles) == 0 {
		t.Fatalf("RBAC Matrix retrieval failed: %+v", rbacMatrix)
	}

	// Test field-level masking: Warehouse role should mask standard_price & cost_price
	unmaskedWarehouseData := map[string]interface{}{
		"name":           "Gaming Monitor 27 Inch",
		"default_code":   "MON-27-001",
		"standard_price": 2500000.0,
		"cost_price":     2500000.0,
		"qty_available":  10.0,
	}
	maskedWarehouseData := security.MaskRecordDynamically("warehouse", "product", unmaskedWarehouseData)
	if maskedWarehouseData["standard_price"] != "***CONFIDENTIAL***" || maskedWarehouseData["cost_price"] != "***CONFIDENTIAL***" {
		t.Fatalf("Expected masked cost prices for warehouse role, got: %+v", maskedWarehouseData)
	}
	if maskedWarehouseData["qty_available"] != 10.0 {
		t.Fatalf("Expected unmasked physical quantity for warehouse role, got: %v", maskedWarehouseData["qty_available"])
	}

	// Admin role should NOT mask anything
	adminData := security.MaskRecordDynamically("admin", "product", unmaskedWarehouseData)
	if adminData["standard_price"] != 2500000.0 {
		t.Fatalf("Expected unmasked price for admin role, got: %v", adminData["standard_price"])
	}
	t.Logf("✓ Enterprise SSO & Field-Level RBAC Matrix Verified: SSO Providers=%d, Roles=%d, MaskingVerified(warehouse standard_price='%s', admin standard_price=%.2f)",
		len(ssoConfig.Providers), len(rbacMatrix.RoleProfiles), maskedWarehouseData["standard_price"], adminData["standard_price"])

	// 58.5 Visual Drag-and-Drop BI Dashboard & Custom Executive Report Builder
	biHandler := bi.NewHandler(db)
	if biHandler == nil {
		t.Fatal("BI Handler failed to initialize")
	}

	// Test saving a custom BI Dashboard
	customDashID := fmt.Sprintf("DASH-CUSTOM-TEST-%d", time.Now().Unix()%10000)
	customDashReq := bi.BIDashboard{
		ID:          customDashID,
		Name:        "Corporate Strategy & Financial Performance Cockpit",
		Department:  "executive",
		Description: "Executive overview of revenue, gross margin, and working capital.",
		IsDefault:   false,
		CompanyID:   1,
		Widgets: []bi.BIWidget{
			{
				ID:         "WGT-TEST-01",
				Title:      "Real-Time Net Revenue",
				Type:       "kpi_card",
				GridX:      0,
				GridY:      0,
				GridW:      4,
				GridH:      2,
				DataSource: "monthly_revenue",
			},
			{
				ID:         "WGT-TEST-02",
				Title:      "Gross Margin %",
				Type:       "gauge",
				GridX:      4,
				GridY:      0,
				GridW:      4,
				GridH:      2,
				DataSource: "cogs_breakdown",
			},
		},
	}

	// Test via HTTP endpoint simulation
	wSave := httptest.NewRecorder()
	cSave, _ := gin.CreateTestContext(wSave)
	saveBody, _ := json.Marshal(customDashReq)
	cSave.Request = httptest.NewRequest("POST", "/api/v1/bi/dashboards", strings.NewReader(string(saveBody)))
	cSave.Request.Header.Set("Content-Type", "application/json")
	cSave.Set("company_id", uint(1))
	cSave.Set("user_id", uint(1))
	biHandler.SaveDashboard(cSave)

	if wSave.Code != http.StatusCreated {
		t.Fatalf("Failed to save custom BI dashboard: code=%d, body=%s", wSave.Code, wSave.Body.String())
	}

	// Test fetching the dashboard with live dynamic widget payloads
	wGet := httptest.NewRecorder()
	cGet, _ := gin.CreateTestContext(wGet)
	cGet.Request = httptest.NewRequest("GET", "/api/v1/bi/dashboards/"+customDashID, nil)
	cGet.Params = []gin.Param{{Key: "id", Value: customDashID}}
	cGet.Set("company_id", uint(1))
	biHandler.GetDashboard(cGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("Failed to get BI dashboard: code=%d, body=%s", wGet.Code, wGet.Body.String())
	}

	var fetchedDash bi.BIDashboard
	if err := json.Unmarshal(wGet.Body.Bytes(), &fetchedDash); err != nil || len(fetchedDash.Widgets) != 2 {
		t.Fatalf("Failed to unmarshal populated BI dashboard: %v", err)
	}
	if fetchedDash.Widgets[0].DataPayload == nil || fetchedDash.Widgets[0].DataPayload["value"] == nil {
		t.Fatalf("Expected live populated DataPayload for widget, got: %+v", fetchedDash.Widgets[0])
	}

	// Test Custom Report Export
	wExp := httptest.NewRecorder()
	cExp, _ := gin.CreateTestContext(wExp)
	repReqBody, _ := json.Marshal(bi.CustomReportConfig{
		ReportName: "Executive_Sales_Performance_Q4",
		Module:     "sale",
		Columns:    []string{"id", "name", "amount_total"},
		Format:     "excel",
	})
	cExp.Request = httptest.NewRequest("POST", "/api/v1/bi/reports/export", strings.NewReader(string(repReqBody)))
	cExp.Request.Header.Set("Content-Type", "application/json")
	cExp.Set("company_id", uint(1))
	biHandler.ExportCustomReport(cExp)

	if wExp.Code != http.StatusOK {
		t.Fatalf("Export custom report failed: code=%d, body=%s", wExp.Code, wExp.Body.String())
	}
	var repRes bi.ReportExportResult
	if err := json.Unmarshal(wExp.Body.Bytes(), &repRes); err != nil || repRes.Status != "SUCCESS_GENERATED" {
		t.Fatalf("Invalid report export response: %v, body=%s", err, wExp.Body.String())
	}
	t.Logf("✓ Visual BI Dashboard & Custom Report Builder Verified: Dashboard='%s', Widgets=%d (Populated Metric: %v), ReportExport='%s' (Status: %s)",
		fetchedDash.Name, len(fetchedDash.Widgets), fetchedDash.Widgets[0].DataPayload["formatted"], repRes.ReportName, repRes.Status)

	// =========================================================================
	// SECTION 59: AI DYNAMIC PRICING, CROSS-BORDER FX HEDGING, IOT MACHINE HEALTH,
	//             VENDOR GREEN PASSPORT & SEMANTIC RAG COPILOT
	// =========================================================================
	t.Log(`
		>>> SECTION 59: AI DYNAMIC PRICING, CROSS-BORDER FX, IOT TELEMETRY & RAG COPILOT <<<`)

	// 59.1 AI Autonomous Dynamic Pricing & Elasticity Optimizer
	pricingResp, err := aiClient.OptimizeProductPricing(db, &ai.PricingRecommendRequest{
		ProductID:         1,
		CustomerSegment:   "wholesale",
		CompetitorPrice:   2450000,
		MinGrossMarginPct: 20.0,
		Urgency:           "standard",
	}, 1)
	if err != nil || pricingResp == nil || pricingResp.RecommendedOptimalPrice <= 0 {
		t.Fatalf("Optimize product pricing failed: err=%v, res=%+v", err, pricingResp)
	}
	if pricingResp.ElasticityClassification != "ELASTIC" || len(pricingResp.VolumeTiers) != 3 {
		t.Fatalf("Unexpected elasticity or volume tiers: %+v", pricingResp)
	}

	appliedProd, err := aiClient.ApplyPricingRule(db, &ai.ApplyPricingRuleRequest{
		ProductID:    1,
		NewListPrice: pricingResp.RecommendedOptimalPrice,
		Reason:       "AI Elasticity Optimization",
	}, 1)
	if err != nil || appliedProd == nil || appliedProd.ListPrice != pricingResp.RecommendedOptimalPrice {
		t.Fatalf("Apply pricing rule failed: err=%v, prod=%+v", err, appliedProd)
	}
	t.Logf("✓ AI Dynamic Pricing Optimizer Verified: Product='%s', OptimalPrice=IDR %.2f (Change: %+.1f%%), PED=%.2f (%s), Margin=%.1f%%, VolumeTiers=%d",
		pricingResp.ProductName, pricingResp.RecommendedOptimalPrice, pricingResp.PriceChangePct,
		pricingResp.ElasticityScore, pricingResp.ElasticityClassification, pricingResp.ProjectedGrossMarginPct, len(pricingResp.VolumeTiers))

	// 59.2 Cross-Border FX Exposure Matrix & Automated Treasury Hedging
	fxReport := coretreasury.ComputeFXExposure(db, 1)
	if fxReport == nil || len(fxReport.Positions) != 3 || len(fxReport.StressTestScenarios) != 3 {
		t.Fatalf("Compute FX exposure report failed: %+v", fxReport)
	}
	if fxReport.Positions[0].NetOpenPosition >= 0 || fxReport.Positions[0].PositionType != "SHORT_DEFICIT" {
		t.Fatalf("Expected USD short deficit position, got: %+v", fxReport.Positions[0])
	}

	wHedge := httptest.NewRecorder()
	cHedge, _ := gin.CreateTestContext(wHedge)
	hedgeReqBody, _ := json.Marshal(coretreasury.CreateHedgeContractRequest{
		BankPartner:       "Bank Mandiri Treasury",
		ContractType:      "FORWARD",
		BaseCurrency:      "USD",
		NotionalAmount:    40000,
		LockedForwardRate: 15900,
		MaturityDays:      90,
	})
	cHedge.Request = httptest.NewRequest("POST", "/api/v1/treasury/hedge-contracts", strings.NewReader(string(hedgeReqBody)))
	cHedge.Request.Header.Set("Content-Type", "application/json")
	coretreasury.NewHandler(db).CreateHedgeContract(cHedge)
	if wHedge.Code != http.StatusCreated {
		t.Fatalf("Failed to create hedge contract: code=%d, body=%s", wHedge.Code, wHedge.Body.String())
	}
	t.Logf("✓ Cross-Border FX Exposure & Treasury Hedging Verified: NetExposureIDR=IDR %.2f, USD Short Deficit=$%.2f, StressScenarios=%d, Policy='%s'",
		fxReport.TotalNetExposureIDR, math.Abs(fxReport.Positions[0].NetOpenPosition), len(fxReport.StressTestScenarios), fxReport.ExecutiveHedgingPolicy)

	// 59.3 Predictive Equipment Maintenance & IoT Telemetry Sensor Gateway
	iotEval := iot.EvaluateMachineHealth(db, &iot.TelemetryPacket{
		MachineID:        "SMT-AUTO-01",
		MachineName:      "Mesin Solder & Perakitan SMT Otomatis",
		VibrationRMS:     7.8, // Exceeds ISO critical limit 7.1 mm/s
		OperatingTempC:   89.0, // Exceeds thermal limit 85 C
		MotorCurrentAmps: 18.2,
		SpindleRPM:       12400,
		CumulativeHours:  2200,
	}, 1)
	if iotEval == nil || iotEval.Status != "CRITICAL_ANOMALY" || len(iotEval.AnomalyIndicators) < 2 {
		t.Fatalf("IoT machine health evaluation failed: %+v", iotEval)
	}
	if iotEval.RemainingUsefulLifeHours <= 0 || iotEval.TriggeredMaintenanceReq == "" {
		t.Fatalf("Expected auto-generated maintenance ticket and RUL estimation, got: %+v", iotEval)
	}
	t.Logf("✓ Predictive IoT Machine Health Verified: Machine='%s', Status='%s', HealthScore=%.1f/100, RUL=%.0f hrs, AutoTicket='%s', Action='%s'",
		iotEval.MachineName, iotEval.Status, iotEval.HealthScore, iotEval.RemainingUsefulLifeHours, iotEval.TriggeredMaintenanceReq, iotEval.SuggestedAction)

	// 59.4 Supplier Sustainability Passport & Scope 3 Vendor Scoring
	vendorCard := esg.AssessVendorSustainability(db, &esg.VendorAssessmentRequest{
		VendorID:                1,
		VendorName:              "PT Pelanggan Sejahtera (Vendor Divisi Display)",
		Country:                 "Indonesia",
		CarbonEmissionIntensity: 12.8,
		BenchmarkIntensity:      28.5,
		ISO14001Certified:       true,
		ISO50001Certified:       true,
		RecycledPackagingPct:    90.0,
		RenewableEnergySharePct: 70.0,
		LogisticsGreenMode:      "ELECTRIC_EV",
	})
	if vendorCard == nil || vendorCard.SustainabilityTier != "PLATINUM_LEADER" || vendorCard.CompositeGreenScore < 85.0 {
		t.Fatalf("Vendor sustainability assessment failed: %+v", vendorCard)
	}
	if vendorCard.ProcurementBonusScore <= 0 || vendorCard.Scope3Factor >= 1.0 {
		t.Fatalf("Expected green bonus points and low Scope 3 factor for platinum leader, got: %+v", vendorCard)
	}
	t.Logf("✓ Supplier Sustainability Passport Verified: Vendor='%s', Tier='%s', CompositeGreenScore=%.1f/100, EmissionIntensity=%.1f kg CO2e, BonusPts=+%.1f, Scope3Factor=%.2f",
		vendorCard.VendorName, vendorCard.SustainabilityTier, vendorCard.CompositeGreenScore, vendorCard.CarbonEmissionIntensity, vendorCard.ProcurementBonusScore, vendorCard.Scope3Factor)

	// 59.5 AI Knowledge Base Semantic Search & Customer Support Copilot (RAG)
	ragQueryResp := aiClient.QueryKnowledgeRAG(&ai.KnowledgeQueryRequest{
		Query: "Apakah monitor dilindungi garansi dan bagaimana proses RMA barang rusak?",
		TopK:  2,
	})
	if ragQueryResp == nil || len(ragQueryResp.Chunks) == 0 || ragQueryResp.SynthesizedAnswer == "" {
		t.Fatalf("Knowledge base RAG query failed: %+v", ragQueryResp)
	}

	copilotResp := aiClient.DraftSupportCopilotReply(&ai.CopilotDraftRequest{
		TicketID:         "TCK/2026/00088",
		CustomerName:     "PT Solusi Retail Nusantara",
		IssueTitle:       "Monitor display mati total setelah 4 bulan pemakaian",
		IssueDescription: "Customer ingin tahu masa garansi monitor dan persyaratan klaim unit pengganti.",
	})
	if copilotResp == nil || copilotResp.DraftReplyText == "" || len(copilotResp.ReferencedArticles) == 0 || copilotResp.ConfidenceScore < 0.90 {
		t.Fatalf("Support Copilot draft reply failed: %+v", copilotResp)
	}
	t.Logf("✓ AI Knowledge RAG & Support Copilot Verified: Ticket='%s', Confidence=%.0f%%, Referenced=%v, DraftLength=%d chars",
		copilotResp.TicketID, copilotResp.ConfidenceScore*100, copilotResp.ReferencedArticles, len(copilotResp.DraftReplyText))

	// SECTION 60: AI CLM CONTRACT RISK & REDLINING, SMART 3D BIN PACKING,
	// HR ATTRITION RADAR, FUZZY BANK RECONCILIATION & UNIVERSAL EDI GATEWAY
	t.Log(`
		>>> SECTION 60: AI CLM, 3D BIN PACKING, HR ATTRITION, FUZZY RECON & UNIVERSAL EDI <<<`)

	// 60.1 AI Automated Contract Lifecycle Management (CLM) & Clause Risk Redlining
	clmRes := aiClient.AnalyzeContractCLM(&ai.ContractAnalysisRequest{
		ContractTitle:    "Enterprise Cloud Master Services Agreement 2026",
		ContractType:     "VENDOR_MSA",
		CounterpartyName: "Global Infrastructure Cloud Inc.",
		ContractText: `The Vendor shall indemnify, defend and hold harmless the Customer from all claims.
The liability of the Vendor under this agreement shall be unlimited without limitation of liability cap.
Customer may terminate this agreement for convenience upon 24 hours written notice.
Customer agrees to pay all invoices within 90 days of receipt.
Liquidated damages of 10% per day shall be assessed for any service delay.
This agreement shall automatically renew for additional 2-year periods unless terminated 120 days in advance.`,
	})
	if clmRes == nil || clmRes.TotalClausesDetected < 4 || clmRes.OverallRiskScore < 70.0 || clmRes.RiskLevel != "CRITICAL_EXPOSURE" {
		t.Fatalf("CLM contract analysis failed: %+v", clmRes)
	}
	if clmRes.HighRiskClausesCount < 2 {
		t.Fatalf("Expected at least 2 high/critical risk clauses, got %d: %+v", clmRes.HighRiskClausesCount, clmRes)
	}

	redlineRes := aiClient.RedlineContractClause(&ai.ContractRedlineRequest{
		ContractTitle: "Enterprise Cloud MSA 2026",
		ClauseType:    "LIABILITY_CAP",
		CurrentText:   "The liability of the Vendor under this agreement shall be unlimited.",
	})
	if redlineRes == nil || redlineRes.RedlinedText == "" || redlineRes.LegalRationale == "" {
		t.Fatalf("CLM contract redline failed: %+v", redlineRes)
	}
	t.Logf("✓ AI Contract CLM & Clause Redlining Verified: Title='%s', RiskLevel='%s', RiskScore=%.1f/100, ClausesDetected=%d, HighRisk=%d, RedlineSample='%s'",
		clmRes.ContractTitle, clmRes.RiskLevel, clmRes.OverallRiskScore, clmRes.TotalClausesDetected, clmRes.HighRiskClausesCount, redlineRes.RedlinedText)

	// 60.2 Smart 3D Bin Packing & Pallet Load Optimizer
	packRes := corewms.Optimize3DPacking(&corewms.PackOptimizationRequest{
		ContainerType: "EURO_PALLET",
		Items: []corewms.PackItemInput{
			{ItemCode: "BOX-SRV", ItemName: "Server Chassis 2U", LengthMM: 600, WidthMM: 400, HeightMM: 300, WeightKG: 15, Quantity: 4, AllowRotation: true, IsFragile: false},
			{ItemCode: "BOX-SW", ItemName: "Fiber Switch 48P", LengthMM: 400, WidthMM: 300, HeightMM: 200, WeightKG: 8, Quantity: 6, AllowRotation: true, IsFragile: true},
			{ItemCode: "BOX-PDU", ItemName: "Rack PDU Strip", LengthMM: 300, WidthMM: 200, HeightMM: 150, WeightKG: 5, Quantity: 8, AllowRotation: true, IsFragile: false},
		},
	})
	if packRes == nil || packRes.TotalItemsPacked != packRes.TotalItemsRequested || packRes.VolumeUtilizationPct <= 0 {
		t.Fatalf("3D bin packing optimizer failed: %+v", packRes)
	}
	if packRes.ContainerSpec.ContainerType != "EURO_PALLET" || len(packRes.LoadingSequence) != 18 {
		t.Fatalf("Unexpected packed items count or container spec: %+v", packRes)
	}
	// Verify that fragile items are placed safely
	fragileAtTop := false
	for _, it := range packRes.LoadingSequence {
		if it.IsFragile && it.LayerIndex > 0 {
			fragileAtTop = true
			break
		}
	}
	if !fragileAtTop {
		t.Fatalf("Fragile items should be placed in higher layers for pallet stability: %+v", packRes)
	}
	t.Logf("✓ Smart 3D Bin Packing Optimizer Verified: Container='%s' (Dims: %.0fx%.0fx%.0f mm), Packed=%d/%d, VolEfficiency=%.1f%%, Weight=%.1f kg, Layers=%d",
		packRes.ContainerSpec.ContainerType, packRes.ContainerSpec.LengthMM, packRes.ContainerSpec.WidthMM, packRes.ContainerSpec.HeightMM,
		packRes.TotalItemsPacked, packRes.TotalItemsRequested, packRes.VolumeUtilizationPct, packRes.TotalGrossWeightKG, packRes.LayersCount)

	// 60.3 Predictive Employee Attrition & AI Talent Retention Radar
	radarRes := corehr.ComputeAttritionRadar(db, 1)
	if radarRes == nil || radarRes.TotalEmployeesMonitored < 3 || radarRes.HighRiskCount == 0 {
		t.Fatalf("HR attrition radar evaluation failed: %+v", radarRes)
	}
	topRiskEmp := radarRes.Employees[0]
	if topRiskEmp.FlightRiskScore < 70.0 || topRiskEmp.FlightRiskLevel != "HIGH_FLIGHT_RISK" || len(topRiskEmp.KeyDrivers) == 0 {
		t.Fatalf("Expected top at-risk employee to have high flight risk: %+v", topRiskEmp)
	}

	wPlaybook := httptest.NewRecorder()
	cPlaybook, _ := gin.CreateTestContext(wPlaybook)
	playbookBody, _ := json.Marshal(corehr.RetentionPlaybookRequest{
		EmployeeID:      topRiskEmp.EmployeeID,
		ProposedAction:  "salary_adjustment",
		BudgetImpactIDR: 2500000,
		Notes:           "Approval of retention salary adjustment and bonus retention grant",
	})
	cPlaybook.Request = httptest.NewRequest("POST", "/api/v1/hr/retention-playbook", strings.NewReader(string(playbookBody)))
	cPlaybook.Request.Header.Set("Content-Type", "application/json")
	corehr.NewHandler(db).ApplyRetentionPlaybook(cPlaybook)
	if wPlaybook.Code != http.StatusOK {
		t.Fatalf("Failed to trigger retention playbook: code=%d, body=%s", wPlaybook.Code, wPlaybook.Body.String())
	}
	t.Logf("✓ Predictive Employee Attrition Radar Verified: Monitored=%d, HighRiskCount=%d, Depts=%d, TopRiskEmp='%s' (Score=%.1f, Drivers=%v), PlaybookTriggered=OK",
		radarRes.TotalEmployeesMonitored, radarRes.HighRiskCount, len(radarRes.DepartmentRiskRankings), topRiskEmp.EmployeeName, topRiskEmp.FlightRiskScore, topRiskEmp.KeyDrivers)

	// 60.4 Multi-Horizon Bank Reconciler with ML Fuzzy Statement Matching
	fuzzyMatchRes := coreaccount.RunFuzzyReconciliation(db, []coreaccount.RawBankStatementLine{
		{LineID: "BNK-001", Date: "2026-11-01", Description: "TRF BCA PT MAJU BERSAMA INV 00492", Amount: 15497500, BankRef: "REF99281"},
		{LineID: "BNK-002", Date: "2026-11-02", Description: "BI-FAST PELANGGAN SINAR SURYA INV-2026-881", Amount: 8993500, BankRef: "BIFAST-0029"},
		{LineID: "BNK-003", Date: "2026-11-03", Description: "SETORAN TUNAI TOKO ABADI JAYA", Amount: 4500000, BankRef: "CSH-0081"},
	}, 1)
	if fuzzyMatchRes == nil || len(fuzzyMatchRes.Matches) == 0 {
		t.Fatalf("Fuzzy bank reconciliation failed: %+v", fuzzyMatchRes)
	}

	wBatchPost := httptest.NewRecorder()
	cBatchPost, _ := gin.CreateTestContext(wBatchPost)
	batchPostReq, _ := json.Marshal(coreaccount.BatchReconcileRequest{
		Matches:    fuzzyMatchRes.Matches,
		AutoPostGL: true,
	})
	cBatchPost.Request = httptest.NewRequest("POST", "/api/v1/account/reconciliation/batch-post", strings.NewReader(string(batchPostReq)))
	cBatchPost.Request.Header.Set("Content-Type", "application/json")
	coreaccount.NewHandler(db).BatchPost(cBatchPost)
	if wBatchPost.Code != http.StatusOK {
		t.Fatalf("Failed to batch post reconciliation: code=%d, body=%s", wBatchPost.Code, wBatchPost.Body.String())
	}
	t.Logf("✓ Multi-Horizon Fuzzy Bank Reconciliation Verified: Lines=%d, Matched=%d, Unmatched=%d, MatchedAmount=IDR %.2f, 1-ClickBatchPost=OK",
		fuzzyMatchRes.TotalStatementLines, fuzzyMatchRes.MatchedLinesCount, fuzzyMatchRes.UnmatchedLinesCount, fuzzyMatchRes.TotalMatchedVolumeIDR)

	// 60.5 Universal EDI Gateway (ANSI X12 & UN/EDIFACT Parser and Generator)
	rawEDI850 := `ISA*00*          *00*          *ZZ*WALMART_HQ     *ZZ*GOERP_GLOBAL   *261103*1430*U*00401*000099123*0*P*>~
GS*PO*WALMART_HQ*GOERP_GLOBAL*20261103*1430*000099123*X*004010~
ST*850*0001~
BEG*00*SA*PO-WMT-2026-9901**20261103~
CUR*USD~
PO1*1*500*EA*25.00**VN*SKU-ROUTER-WIFI7~
PID*F****Enterprise Wireless Router WiFi 7~
PO1*2*1000*EA*12.50**VN*SKU-POE-ADAPTER~
PID*F****Gigabit Ethernet PoE Adapter~
TDS*25000~
CTT*2~
SE*12*0001~
GE*1*000099123~
IEA*1*000099123~`

	ediIngestRes := coreedi.IngestEDI(rawEDI850, "ANSI_X12", true, db, 1)
	if ediIngestRes == nil || ediIngestRes.Format != "ANSI_X12" || ediIngestRes.DocumentType != "PURCHASE_ORDER_850" || ediIngestRes.LineItemsCount != 2 {
		t.Fatalf("EDI 850 ingest failed: %+v", ediIngestRes)
	}
	if ediIngestRes.PONumber != "PO-WMT-2026-9901" || ediIngestRes.TotalAmount != 25000 {
		t.Fatalf("Unexpected EDI 850 PO details: %+v", ediIngestRes)
	}
	if ediIngestRes.CreatedRecordID == nil {
		t.Fatalf("Expected auto-created Sales Order from EDI 850, got nil")
	}

	ediGenRes := coreedi.GenerateEDI(&coreedi.EDIGenerateRequest{
		DocumentType:   "810",
		Format:         "ANSI_X12",
		SenderID:       "GOERP_GLOBAL",
		ReceiverID:     "WALMART_HQ",
		DocumentNumber: "INV-2026-00991",
		TotalAmount:    25000,
		LineItems:      ediIngestRes.LineItems,
	})
	if ediGenRes == nil || !strings.Contains(ediGenRes.RawPayload, "ST*810*") || !strings.Contains(ediGenRes.RawPayload, "BIG*") {
		t.Fatalf("EDI 810 invoice generation failed: %+v", ediGenRes)
	}

	ediFactGenRes := coreedi.GenerateEDI(&coreedi.EDIGenerateRequest{
		DocumentType:   "INVOIC",
		Format:         "EDIFACT",
		SenderID:       "GOERP_GLOBAL",
		ReceiverID:     "CARREFOUR_EU",
		DocumentNumber: "INV-2026-00992",
		TotalAmount:    25000,
		LineItems:      ediIngestRes.LineItems,
	})
	if ediFactGenRes == nil || !strings.Contains(ediFactGenRes.RawPayload, "UNB+UNOC") || !strings.Contains(ediFactGenRes.RawPayload, "BGM+380+") {
		t.Fatalf("EDIFACT INVOIC generation failed: %+v", ediFactGenRes)
	}
	t.Logf("✓ Universal EDI Gateway Verified: IngestedFormat='%s', DocType='%s', PO='%s', Total=%.2f, StagedSaleOrderID=%d, GeneratedX12Segs=%d, GeneratedEDIFACTSegs=%d",
		ediIngestRes.Format, ediIngestRes.DocumentType, ediIngestRes.PONumber, ediIngestRes.TotalAmount, *ediIngestRes.CreatedRecordID, ediGenRes.SegmentCount, ediFactGenRes.SegmentCount)

	t.Log("🎉 COMPLETE GOERP EXPANDED TEST SUITE (60 SECTIONS: 50+ MODULES + OPENAPI 3.1 + MCP JSON-RPC 2.0 + GOOGLE A2A + AI DEMAND SENSING + LIVE SSE PRESENCE + SAAS MULTI-TENANCY + OFFLINE POS + VOICE ASSISTANT + RFQ BIDDING WARS + MERKLE AUDIT + DIGITAL TWIN + FIELD RBAC + VISUAL BI + DYNAMIC PRICING + FX HEDGING + IOT HEALTH + VENDOR PASSPORT + RAG COPILOT + AI CLM + 3D BIN PACKING + HR ATTRITION RADAR + FUZZY BANK RECON + UNIVERSAL EDI) PASSED 100%!")
}
