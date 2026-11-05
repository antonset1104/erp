package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"erp-golang/internal/config"
	"erp-golang/internal/core/ai"
	"erp-golang/internal/core/audit"
	"erp-golang/internal/core/bi"
	"erp-golang/internal/core/commandcenter"
	coreaccount "erp-golang/internal/core/account"
	coreaerospace "erp-golang/internal/core/aerospace"
	coreagriculture "erp-golang/internal/core/agriculture"
	coreammonia "erp-golang/internal/core/ammonia"
	coreasset "erp-golang/internal/core/asset"
	coreautomotive "erp-golang/internal/core/automotive"
	coreaviation "erp-golang/internal/core/aviation"
	corebess "erp-golang/internal/core/bess"
	corebiopharma "erp-golang/internal/core/biopharma"
	corebop "erp-golang/internal/core/bop"
	corecart "erp-golang/internal/core/cart"
	coreccus "erp-golang/internal/core/ccus"
	corecosting "erp-golang/internal/core/costing"
	corecpo "erp-golang/internal/core/cpo"
	corecpq "erp-golang/internal/core/cpq"
	corecrm "erp-golang/internal/core/crm"
	corecryo "erp-golang/internal/core/cryo"
	corecustoms "erp-golang/internal/core/customs"
	coredac "erp-golang/internal/core/dac"
	coreedi "erp-golang/internal/core/edi"
	coreehs "erp-golang/internal/core/ehs"
	coreenergy "erp-golang/internal/core/energy"
	coreevtol "erp-golang/internal/core/evtol"
	coreexpense "erp-golang/internal/core/expense"
	corefermentation "erp-golang/internal/core/fermentation"
	corefieldservice "erp-golang/internal/core/fieldservice"
	corefinance "erp-golang/internal/core/finance"
	corefleet "erp-golang/internal/core/fleet"
	corefusion "erp-golang/internal/core/fusion"
	coregaafet "erp-golang/internal/core/gaafet"
	coregeotech "erp-golang/internal/core/geotech"
	coregeothermal "erp-golang/internal/core/geothermal"
	coregeothermalshr "erp-golang/internal/core/geothermal_shr"
	coregraphql "erp-golang/internal/core/graphql"
	coregrc "erp-golang/internal/core/grc"
	corehallthruster "erp-golang/internal/core/hallthruster"
	corehr "erp-golang/internal/core/hr"
	corehyperloop "erp-golang/internal/core/hyperloop"
	corehypersonic "erp-golang/internal/core/hypersonic"
	coreitops "erp-golang/internal/core/itops"
	corelegal "erp-golang/internal/core/legal"
	corelnp "erp-golang/internal/core/lnp"
	corelogistics "erp-golang/internal/core/logistics"
	corelyo "erp-golang/internal/core/lyophilization"
	coremaintenance "erp-golang/internal/core/maintenance"
	coremarine "erp-golang/internal/core/marine"
	coremining "erp-golang/internal/core/mining"
	coremoltensalt "erp-golang/internal/core/moltensalt"
	coremrp "erp-golang/internal/core/mrp"
	coremsr "erp-golang/internal/core/msr"
	corenuclear "erp-golang/internal/core/nuclear"
	corepackaging "erp-golang/internal/core/packaging"
	corepellicle "erp-golang/internal/core/pellicle"
	corepharma "erp-golang/internal/core/pharma"
	corepogo "erp-golang/internal/core/pogo"
	corepowergrid "erp-golang/internal/core/powergrid"
	corepricing "erp-golang/internal/core/pricing"
	coreprocurement "erp-golang/internal/core/procurement"
	coreproject "erp-golang/internal/core/project"
	corepyrolysis "erp-golang/internal/core/pyrolysis"
	corequality "erp-golang/internal/core/quality"
	coreqled "erp-golang/internal/core/qled"
	corequantum "erp-golang/internal/core/quantum"
	corerailway "erp-golang/internal/core/railway"
	corerenewables "erp-golang/internal/core/renewables"
	corescm "erp-golang/internal/core/scm"
	coresaltcavern "erp-golang/internal/core/saltcavern"
	corerocketgridfin "erp-golang/internal/core/rocket_gridfin"
	corembmw "erp-golang/internal/core/mbmw"
	coreaavgenetherapy "erp-golang/internal/core/aav_genetherapy"
	coredryelectrode "erp-golang/internal/core/solidstate_dryelectrode"
	coreisrumethalox "erp-golang/internal/core/isru_methalox"
	corelh2aviation "erp-golang/internal/core/lh2_aviation"
	corehbm4bonding "erp-golang/internal/core/hbm4_bonding"
	coremicroreactor "erp-golang/internal/core/microreactor_heatpipe"
	coremrnacapping "erp-golang/internal/core/mrna_capping"
	coresubseacompressor "erp-golang/internal/core/subsea_compressor"
	coreairbornehts "erp-golang/internal/core/airborne_hts"
	corehdristeel "erp-golang/internal/core/hdri_steel"
	corecryoquantum "erp-golang/internal/core/cryo_quantum"
	coreammoniamarine "erp-golang/internal/core/ammonia_marine"
	corelunarsintering "erp-golang/internal/core/lunar_sintering"
	corescramjet "erp-golang/internal/core/scramjet"
	coreseabed "erp-golang/internal/core/seabed"
	coresemiconductor "erp-golang/internal/core/semiconductor"
	coresmr "erp-golang/internal/core/smr"
	coresoec "erp-golang/internal/core/soec"
	coresolidstate "erp-golang/internal/core/solidstate"
	coreservice "erp-golang/internal/core/service"
	coresubsea "erp-golang/internal/core/subsea"
	coresubseaccs "erp-golang/internal/core/subsea_ccs"
	coresubseaauv "erp-golang/internal/core/subsea_auv"
	coresubscription "erp-golang/internal/core/subscription"
	coresystem "erp-golang/internal/core/system"
	coretax "erp-golang/internal/core/tax"
	coretradefinance "erp-golang/internal/core/tradefinance"
	coretreasury "erp-golang/internal/core/treasury"
	coreuhvdc "erp-golang/internal/core/uhvdc"
	coreutilities "erp-golang/internal/core/utilities"
	corevmi "erp-golang/internal/core/vmi"
	corewms "erp-golang/internal/core/wms"
	corewetchem "erp-golang/internal/core/wetchem"
	"erp-golang/internal/core/esg"
	"erp-golang/internal/core/iot"
	"erp-golang/internal/core/database"
	"erp-golang/internal/core/events"
	"erp-golang/internal/core/middleware"
	"erp-golang/internal/core/multitenancy"
	"erp-golang/internal/core/openapi"
	"erp-golang/internal/core/registry"
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
	"gorm.io/gorm"
)

func main() {
	log.Println("=====================================================")
	log.Println("     Starting GoERP Backend & Dashboard        ")
	log.Println("=====================================================")

	// 1. Load Configurations
	cfg := config.LoadConfig()

	// 2. Initialize Database (SQLite by default, or PostgreSQL)
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("Fatal: Database initialization failed: %v", err)
	}

	// 3. Initialize AI Client & Storage Provider (Cloudflare R2 / Local Disk)
	aiClient := ai.NewClient(cfg)
	storageProv, err := storage.InitStorageProvider(cfg)
	if err != nil {
		log.Fatalf("Fatal: Storage provider initialization failed: %v", err)
	}

	// 4. Initialize Module Manager
	manager := registry.NewModuleManager(db)

	// Create and register core & extended modules
	baseMod := base.NewModule(db, cfg.JWTSecret)
	productMod := product.NewModule(db)
	stockMod := stock.NewModule(db)
	saleMod := sale.NewModule(db, stockMod.Service())
	purchaseMod := purchase.NewModule(db, stockMod.Service())
	accountMod := account.NewModule(db)
	mrpMod := mrp.NewModule(db, stockMod.Service())
	posMod := pos.NewModule(db)
	hrMod := hr.NewModule(db)
	crmMod := crm.NewModule(db, saleMod.Service())
	chatterMod := chatter.NewModule(db)
	projectMod := project.NewModule(db)
	assetMod := asset.NewModule(db, accountMod.Service())
	helpdeskMod := helpdesk.NewModule(db)
	maintenanceMod := maintenance.NewModule(db)
	fleetMod := fleet.NewModule(db)
	leaveMod := leave.NewModule(db)
	expenseMod := expense.NewModule(db, accountMod.Service(), chatterMod.Service())
	qualityMod := quality.NewModule(db, stockMod.Service(), mrpMod.Service(), chatterMod.Service())
	approvalMod := approval.NewModule(db, chatterMod.Service())
	attachmentMod := attachment.NewModule(db, storageProv, aiClient, chatterMod.Service(), expenseMod.Service())
	attendanceMod := attendance.NewModule(db, chatterMod.Service(), aiClient)
	recruitmentMod := recruitment.NewModule(db, hrMod.Service(), chatterMod.Service(), aiClient)
	stockInventoryMod := stock_inventory.NewModule(db, chatterMod.Service(), aiClient)
	payrollMod := payroll.NewModule(db, chatterMod.Service(), aiClient, accountMod.Service())
	budgetMod := budget.NewModule(db, chatterMod.Service(), aiClient, accountMod.Service())
	barcodeMod := barcode.NewModule(db)
	purchaseRequisitionMod := purchase_requisition.NewModule(db, purchaseMod.Service(), chatterMod.Service())
	currencyMod := currency.NewModule(db, accountMod.Service(), chatterMod.Service(), aiClient)
	taxEFakturMod := tax_efaktur.NewModule(db, accountMod.Service(), chatterMod.Service(), aiClient)
	bankReconcileMod := bank_reconciliation.NewModule(db, accountMod.Service(), chatterMod.Service(), aiClient)
	landedCostMod := landed_cost.NewModule(db, accountMod.Service(), stockMod.Service(), chatterMod.Service(), aiClient)
	traceabilityMod := traceability.NewModule(db, chatterMod.Service(), aiClient)
	subscriptionMod := subscription.NewModule(db, accountMod.Service(), chatterMod.Service(), aiClient)
	auditTrailMod := audit_trail.NewModule(db, chatterMod.Service(), aiClient)
	webhookMod := webhook.NewModule(db, chatterMod.Service(), aiClient)
	inboundWebhookMod := inbound_webhook.NewModule(db, accountMod.Service(), chatterMod.Service(), aiClient)
	intercompanyMod := intercompany.NewModule(db, accountMod.Service(), chatterMod.Service(), aiClient)
	treasuryMod := treasury.NewModule(db, accountMod.Service(), chatterMod.Service(), aiClient)
	mrpAdvancedMod := mrp_advanced.NewModule(db, mrpMod.Service(), chatterMod.Service(), aiClient)
	taxEbupotMod := tax_ebupot.NewModule(db, payrollMod.Service(), accountMod.Service(), chatterMod.Service(), aiClient)
	esignatureMod := esignature.NewModule(db, chatterMod.Service(), aiClient)
	wmsMod := wms.NewModule(db, stockMod.Service(), chatterMod.Service(), aiClient)
	vendorPortalMod := vendor_portal.NewModule(db, purchaseMod.Service(), stockMod.Service(), accountMod.Service(), chatterMod.Service(), aiClient)
	customerPortalMod := customer_portal.NewModule(db, saleMod.Service(), stockMod.Service(), accountMod.Service(), chatterMod.Service(), aiClient)
	customsBondedMod := customs_bonded.NewModule(db, stockMod.Service(), chatterMod.Service(), aiClient)
	lowCodeStudioMod := lowcode_studio.NewModule(db, chatterMod.Service(), aiClient)
	reportsMod := reports.NewModule(db)
	workflowEngineMod := workflow_engine.NewModule(db, chatterMod.Service(), aiClient)
	integrationHubMod := integration_hub.NewModule(db, chatterMod.Service(), aiClient)

	manager.Register(baseMod)
	manager.Register(productMod)
	manager.Register(stockMod)
	manager.Register(saleMod)
	manager.Register(purchaseMod)
	manager.Register(accountMod)
	manager.Register(mrpMod)
	manager.Register(posMod)
	manager.Register(hrMod)
	manager.Register(crmMod)
	manager.Register(chatterMod)
	manager.Register(projectMod)
	manager.Register(assetMod)
	manager.Register(helpdeskMod)
	manager.Register(maintenanceMod)
	manager.Register(fleetMod)
	manager.Register(leaveMod)
	manager.Register(expenseMod)
	manager.Register(qualityMod)
	manager.Register(approvalMod)
	manager.Register(attachmentMod)
	manager.Register(attendanceMod)
	manager.Register(recruitmentMod)
	manager.Register(stockInventoryMod)
	manager.Register(payrollMod)
	manager.Register(budgetMod)
	manager.Register(barcodeMod)
	manager.Register(purchaseRequisitionMod)
	manager.Register(currencyMod)
	manager.Register(taxEFakturMod)
	manager.Register(bankReconcileMod)
	manager.Register(landedCostMod)
	manager.Register(traceabilityMod)
	manager.Register(subscriptionMod)
	manager.Register(auditTrailMod)
	manager.Register(webhookMod)
	manager.Register(inboundWebhookMod)
	manager.Register(intercompanyMod)
	manager.Register(treasuryMod)
	manager.Register(mrpAdvancedMod)
	manager.Register(taxEbupotMod)
	manager.Register(esignatureMod)
	manager.Register(wmsMod)
	manager.Register(vendorPortalMod)
	manager.Register(customerPortalMod)
	manager.Register(customsBondedMod)
	manager.Register(lowCodeStudioMod)
	manager.Register(reportsMod)
	manager.Register(workflowEngineMod)
	manager.Register(integrationHubMod)

	// 4. Auto-migrate all models
	if err := manager.MigrateAll(); err != nil {
		log.Fatalf("Fatal: Database migration failed: %v", err)
	}

	// 5. Seed default master data
	if err := manager.SeedAll(); err != nil {
		log.Fatalf("Fatal: Database seeding failed: %v", err)
	}

	// 5.1 Start automated background worker for Reordering Rules / Procurement
	go func() {
		log.Println("[Cron Worker] Automated Reordering & Procurement engine started (interval: 3m)")
		ticker := time.NewTicker(3 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			orders, err := purchaseMod.Service().RunReorderingRules(1)
			if err == nil && len(orders) > 0 {
				log.Printf("[Cron Worker] Generated %d automatic draft purchase orders from stock reordering rules", len(orders))
			}
		}
	}()

	// 5.2 Start automated background worker for Recurring Subscription Invoicing
	go func() {
		log.Println("[Cron Worker] Automated Subscription Recurring Invoicing engine started (interval: 5m)")
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			todayStr := time.Now().Format("2006-01-02")
			res, err := subscriptionMod.Service().RunBillingBatch(subscription.BillingBatchRequest{TargetDate: todayStr}, 1, 1)
			if err == nil && res.InvoicesGenerated > 0 {
				log.Printf("[Cron Worker] Subscription Billing Batch: Generated %d recurring invoices (Revenue: Rp %.2f)", res.InvoicesGenerated, res.TotalBilledRevenue)
			}
		}
	}()

	// 5.3 Start automated background worker for Webhook Outbound Retry Queue
	go func() {
		log.Println("[Cron Worker] Automated Webhook Outbound Retry engine started (interval: 1m)")
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			retried, err := webhookMod.Service().RunRetryQueue(1)
			if err == nil && retried > 0 {
				log.Printf("[Cron Worker] Webhook Engine: Processed and retried %d outbound deliveries", retried)
			}
		}
	}()

	// 6. Setup Gin HTTP Server
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Enable CORS & Multi-Tenancy Resolution
	r.Use(middleware.CORSMiddleware())
	tenantMgr := multitenancy.InitManager(db)
	r.Use(multitenancy.TenantMiddleware(tenantMgr))

	// Health Check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":     "ok",
			"app":        cfg.AppName,
			"version":    "1.79.0",
			"db_type":    cfg.DBType,
			"storage":    storageProv.Type(),
			"upload_dir": cfg.UploadDir,
			"modules":    []string{"base", "product", "stock", "sale", "purchase", "account", "mrp", "pos", "hr", "crm", "chatter", "project", "asset", "helpdesk", "maintenance", "fleet", "leave", "expense", "quality", "approval", "attachment", "attendance", "recruitment", "stock_inventory", "payroll", "budget", "barcode", "purchase_requisition", "currency", "tax_efaktur", "bank_reconciliation", "landed_cost", "traceability", "subscription", "audit_trail", "webhook", "inbound_webhook", "intercompany", "treasury", "mrp_advanced", "tax_ebupot", "esignature", "wms", "vendor_portal", "customer_portal", "customs_bonded", "lowcode_studio", "reports", "workflow_engine", "integration_hub"},
		})
	})

	// Auth Middleware
	authMiddleware := middleware.AuthMiddleware(cfg.JWTSecret)

	// Cloudflare Workers AI Service Handler
	aiHandler := ai.NewHandler(aiClient, db)

	// Register Root Protocol Routes (MCP JSON-RPC 2.0 & Google A2A Agent Card)
	ai.RegisterRootProtocolRoutes(r, aiHandler)

	// Register OpenAPI 3.1 & Interactive API Explorer
	openapi.RegisterRoutes(r, cfg.AppName, "1.58.0")

	// API v1 Router Group
	apiV1 := r.Group("/api/v1")
	{
		// Dashboard Stats
		apiV1.GET("/dashboard/stats", authMiddleware, func(c *gin.Context) {
			getDashboardStats(c, db)
		})

		// Register routes for all modules
		manager.RegisterAllRoutes(apiV1, authMiddleware)

		// Register Cloudflare Workers AI endpoints
		ai.RegisterAIRoutes(apiV1, authMiddleware, aiHandler)
		commandcenter.RegisterRoutes(apiV1, authMiddleware, db, saleMod.Service(), purchaseMod.Service(), stockMod.Service(), accountMod.Service())
		events.RegisterEventRoutes(apiV1, authMiddleware)
		multitenancy.RegisterSaaSRoutes(apiV1, authMiddleware, tenantMgr)
		audit.RegisterAuditRoutes(apiV1, authMiddleware, db)
		security.RegisterSecurityRoutes(apiV1, authMiddleware)
		bi.RegisterBIRoutes(apiV1, authMiddleware, db)
		coretreasury.RegisterTreasuryFXRoutes(apiV1, authMiddleware, db)
		iot.RegisterIoTRoutes(apiV1, authMiddleware, db)
		esg.RegisterESGVendorRoutes(apiV1, authMiddleware, db)
		corewms.RegisterWMSPackRoutes(apiV1, authMiddleware)
		corehr.RegisterHRRadarRoutes(apiV1, authMiddleware, db)
		coreaccount.RegisterFuzzyReconcileRoutes(apiV1, authMiddleware, db)
		coreedi.RegisterEDIRoutes(apiV1, authMiddleware, db)
		corewms.RegisterCrossDockRoutes(apiV1, authMiddleware, db)
		corewms.RegisterRMARoutes(apiV1, authMiddleware, db)
		coreaccount.RegisterConsolidationRoutes(apiV1, authMiddleware, db)
		coreexpense.RegisterExpenseAuditRoutes(apiV1, authMiddleware, db)
		coremrp.RegisterLineBalancingRoutes(apiV1, authMiddleware, db)
		corefleet.RegisterFleetRouteRoutes(apiV1, authMiddleware, db)
		coreaccount.RegisterCreditRiskRoutes(apiV1, authMiddleware, db)
		coreasset.RegisterCarbonDepreciationRoutes(apiV1, authMiddleware, db)
		coregraphql.RegisterGraphQLRoutes(apiV1, authMiddleware, db)
		corehr.RegisterRosterRoutes(apiV1, authMiddleware, db)
		corewms.RegisterInventoryBalancerRoutes(apiV1, authMiddleware, db)
		corecrm.RegisterCLVRoutes(apiV1, authMiddleware, db)
		security.RegisterQuorumVaultRoutes(apiV1, authMiddleware, db)
		coreasset.RegisterRFIDTrackerRoutes(apiV1, authMiddleware, db)
		coreaccount.RegisterContinuousCloseRoutes(apiV1, authMiddleware, db)
		corevmi.RegisterVMIRoutes(apiV1, authMiddleware, db)
		corecustoms.RegisterFTARoutes(apiV1, authMiddleware, db)
		coresystem.RegisterCQRSRoutes(apiV1, authMiddleware, db)
		coreehs.RegisterEHSRoutes(apiV1, authMiddleware, db)
		coretreasury.RegisterCashPoolingRoutes(apiV1, authMiddleware, db)
		corecosting.RegisterLandedCostRoutes(apiV1, authMiddleware, db)
		esg.RegisterScope3CBAMRoutes(apiV1, authMiddleware, db)
		corefieldservice.RegisterFieldServiceRoutes(apiV1, authMiddleware, db)
		corepricing.RegisterDynamicPricingGuardRoutes(apiV1, authMiddleware, db)
		coremrp.RegisterAndonRoutes(apiV1, authMiddleware, db)
		coretradefinance.RegisterTradeFinanceRoutes(apiV1, authMiddleware, db)
		coretreasury.RegisterDynamicDiscountingRoutes(apiV1, authMiddleware, db)
		iot.RegisterVibrationFFTRoutes(apiV1, authMiddleware, db)
		corehr.RegisterTravelPolicyRoutes(apiV1, authMiddleware, db)
		coretax.RegisterTransferPricingRoutes(apiV1, authMiddleware, db)
		coreaccount.RegisterLeaseAccountingRoutes(apiV1, authMiddleware, db)
		corewms.RegisterYardManagementRoutes(apiV1, authMiddleware, db)
		corecpq.RegisterRoutes(apiV1)
		coregrc.RegisterRoutes(apiV1)
		corescm.RegisterRoutes(apiV1)
		coretreasury.RegisterHedgeAccountingRoutes(apiV1)
		coremrp.RegisterEBRRoutes(apiV1)
		coretax.RegisterWHTRoutes(apiV1)
		corecrm.RegisterLoyaltyRoutes(apiV1)
		esg.RegisterCarbonPricingRoutes(apiV1)
		corewms.RegisterAMROrchestratorRoutes(apiV1)
		coreprocurement.RegisterEscrowRoutes(apiV1)
		coreaccount.RegisterRevRecRoutes(apiV1)
		corehr.RegisterSuccessionRoutes(apiV1)
		corescm.RegisterDPPRoutes(apiV1)
		corefleet.RegisterFreightAuctionRoutes(apiV1)
		coretreasury.RegisterLiquiditySweeperRoutes(apiV1)
		corecpq.RegisterDealDeskRoutes(apiV1)
		coreehs.RegisterEHSSentinelRoutes(apiV1)
		coreprocurement.RegisterCRMResilienceRoutes(apiV1)
		coreasset.RegisterEnergyOptimizerRoutes(apiV1)
		coretreasury.RegisterSLLRoutes(apiV1)
		corewms.RegisterDynamicSlottingRoutes(apiV1)
		corehr.RegisterExpertLocatorRoutes(apiV1)
		coremrp.RegisterECORoutes(apiV1)
		corefleet.RegisterColdChainInsuranceRoutes(apiV1)
		coretreasury.RegisterLiquidityStressRoutes(apiV1)
		coregrc.RegisterCSDDDRoutes(apiV1)
		corehr.RegisterGigMarketplaceRoutes(apiV1)
		coremrp.RegisterTOCRoutes(apiV1)
		coreenergy.RegisterVPPRoutes(apiV1)
		coretax.RegisterDTAARoutes(apiV1)
		security.RegisterZKPRoutes(apiV1)
		corecustoms.RegisterGATTRoutes(apiV1)
		corequality.RegisterRCARoutes(apiV1)
		coretreasury.RegisterNDFRoutes(apiV1)
		corelogistics.RegisterCarrierRoutes(apiV1)
		coreitops.RegisterFinOpsRoutes(apiV1)
		esg.RegisterTravelCarbonRoutes(apiV1)
		coretreasury.RegisterSCFRoutes(apiV1)
		corelogistics.RegisterHazmatRoutes(apiV1)
		coreservice.RegisterWarrantyRoutes(apiV1)
		coretax.RegisterPEPPOLRoutes(apiV1)
		coreehs.RegisterHeatStressRoutes(apiV1)
		esg.RegisterCBAMRoutes(apiV1)
		corewms.RegisterAGFSwarmRoutes(apiV1)
		coretreasury.RegisterZBARoutes(apiV1)
		coremaintenance.RegisterCannibalizationRoutes(apiV1)
		corefinance.RegisterCreditRoutes(apiV1)
		coregrc.RegisterContractLeakageRoutes(apiV1)
		coreenergy.RegisterMicrogridRoutes(apiV1)
		corefleet.RegisterGeofenceRoutes(apiV1)
		corehr.RegisterShiftBiddingRoutes(apiV1)
		corescm.RegisterPackagingRoutes(apiV1)
		coreehs.RegisterWaterRoutes(apiV1)
		coregrc.RegisterSanctionsRoutes(apiV1)
		coremaintenance.RegisterBoilerRoutes(apiV1)
		coresubscription.RegisterDunningRoutes(apiV1)
		coremrp.RegisterTollManufacturingRoutes(apiV1)
		coreagriculture.RegisterAgricultureRoutes(apiV1)
		coreproject.RegisterEPCRoutes(apiV1)
		corelogistics.RegisterBerthRoutes(apiV1)
		coremaintenance.RegisterTransformerRoutes(apiV1)
		coretreasury.RegisterRepatriationRoutes(apiV1)
		coremining.RegisterOreRoutes(apiV1)
		corepharma.RegisterCryogenicRoutes(apiV1)
		coremaintenance.RegisterKilnRoutes(apiV1)
		corelegal.RegisterPatentRoutes(apiV1)
		coreenergy.RegisterTurbineRoutes(apiV1)
		coreaviation.RegisterAviationRoutes(apiV1)
		coremarine.RegisterMarineRoutes(apiV1)
		coresemiconductor.RegisterSemiconductorRoutes(apiV1)
		coreutilities.RegisterUtilitiesRoutes(apiV1)
		corerailway.RegisterRailwayRoutes(apiV1)
		corenuclear.RegisterNuclearRoutes(apiV1)
		corebess.RegisterBESSRoutes(apiV1)
		coreaerospace.RegisterAerospaceRoutes(apiV1)
		coresubsea.RegisterSubseaRoutes(apiV1)
		coreautomotive.RegisterAutomotiveRoutes(apiV1)
		corebiopharma.RegisterBiopharmaRoutes(apiV1)
		corerenewables.RegisterRenewablesRoutes(apiV1)
		corepowergrid.RegisterPowerGridRoutes(apiV1)
		coreccus.RegisterCCUSRoutes(apiV1)
		coregeotech.RegisterGeotechRoutes(apiV1)
		corefusion.RegisterFusionRoutes(apiV1)
		corecryo.RegisterCryoRoutes(apiV1)
		corehypersonic.RegisterHypersonicRoutes(apiV1)
		corepackaging.RegisterPackagingRoutes(apiV1)
		coregeothermal.RegisterGeothermalRoutes(apiV1)
		corequantum.RegisterQuantumRoutes(apiV1)
		coreseabed.RegisterSeabedRoutes(apiV1)
		coremoltensalt.RegisterMoltenSaltRoutes(apiV1)
		coresolidstate.RegisterSolidStateRoutes(apiV1)
		coredac.RegisterDACRoutes(apiV1)
		coreevtol.RegisterEVTOLRoutes(apiV1)
		corehyperloop.RegisterHyperloopRoutes(apiV1)
		corebop.RegisterBOPRoutes(apiV1)
		corecart.RegisterCARTRoutes(apiV1)
		corepyrolysis.RegisterPyrolysisRoutes(apiV1)
		coreammonia.RegisterAmmoniaRoutes(apiV1)
		corepogo.RegisterPogoRoutes(apiV1)
		coresmr.RegisterSMRRoutes(apiV1)
		corepellicle.RegisterPellicleRoutes(apiV1)
		coreuhvdc.RegisterUHVDCRoutes(apiV1)
		coresubseaccs.RegisterSubseaCCSRoutes(apiV1)
		corescramjet.RegisterScramjetRoutes(apiV1)
		corelnp.RegisterMRNALNPRoutes(apiV1)
		corecpo.RegisterCPORoutes(apiV1)
		coresoec.RegisterSOECRoutes(apiV1)
		coregeothermalshr.RegisterGeothermalSHRRoutes(apiV1)
		coresubseaauv.RegisterSubseaAUVRoutes(apiV1)
		corewetchem.RegisterH2O2RunawayRoutes(apiV1)
		corelyo.RegisterLyophilizationRoutes(apiV1)
		coreqled.RegisterQLEDRoutes(apiV1)
		corehallthruster.RegisterHallThrusterRoutes(apiV1)
		coregaafet.RegisterGAAFETRoutes(apiV1)
		corefermentation.RegisterFermentationRoutes(apiV1)
		coresaltcavern.RegisterSaltCavernRoutes(apiV1)
		coremsr.RegisterMSRRoutes(apiV1)
		corerocketgridfin.RegisterGridFinRoutes(apiV1)
		corembmw.RegisterMBMWRoutes(apiV1)
		coreaavgenetherapy.RegisterAAVRoutes(apiV1)
		coredryelectrode.RegisterDryElectrodeRoutes(apiV1)
		coreisrumethalox.RegisterISRURoutes(apiV1)
		corelh2aviation.RegisterLH2Routes(apiV1)
		corehbm4bonding.RegisterHBM4Routes(apiV1)
		coremicroreactor.RegisterMicroreactorRoutes(apiV1)
		coremrnacapping.RegisterMRNARoutes(apiV1)
		coresubseacompressor.RegisterSubseaCompressorRoutes(apiV1)
		coreairbornehts.RegisterAirborneHTSRoutes(apiV1)
		corehdristeel.RegisterHDRIShaftRoutes(apiV1)
		corecryoquantum.RegisterCryoQuantumRoutes(apiV1)
		coreammoniamarine.RegisterAmmoniaMarineRoutes(apiV1)
		corelunarsintering.RegisterLunarSinteringRoutes(apiV1)
	}

	// Serve Static Web Frontend
	r.Static("/web", "./web")
	r.StaticFile("/", "./web/index.html")

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("[Server] GoERP Server is listening on http://localhost%s", addr)
	log.Printf("[Server] Web Dashboard available at http://localhost%s/", addr)
	log.Printf("[Server] Default Admin Login: admin@erp.local / admin123")
	log.Printf("[Storage] Storage Backend: %s (Upload directory: %s)", storageProv.Type(), cfg.UploadDir)
	if storageProv.Type() == "r2" {
		log.Printf("[Cloudflare R2] Connected to Cloudflare R2 Bucket: %s", cfg.CloudflareR2Bucket)
	}
	if aiClient.IsCloudflareConfigured() {
		log.Printf("[Cloudflare AI] Connected to Cloudflare Workers AI edge network (Model: %s)", cfg.CloudflareAIModel)
	} else {
		log.Printf("[Cloudflare AI] Running with Local Smart Heuristic Engine (Cloudflare credentials optional)")
	}

	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed to run: %v", err)
	}
}

func getDashboardStats(c *gin.Context, db *gorm.DB) {
	_, companyID, _ := middleware.GetUserContext(c)

	var productCount int64
	db.Table("products").Where("company_id = ? AND active = ?", companyID, true).Count(&productCount)

	var customerCount int64
	db.Table("partners").Where("company_id = ? AND customer_rank > 0", companyID).Count(&customerCount)

	var vendorCount int64
	db.Table("partners").Where("company_id = ? AND supplier_rank > 0", companyID).Count(&vendorCount)

	var saleCount int64
	db.Table("sale_orders").Where("company_id = ?", companyID).Count(&saleCount)

	var purchaseCount int64
	db.Table("purchase_orders").Where("company_id = ?", companyID).Count(&purchaseCount)

	var pickingCount int64
	db.Table("stock_pickings").Where("company_id = ?", companyID).Count(&pickingCount)

	var invoiceCount int64
	db.Table("account_moves").Where("company_id = ?", companyID).Count(&invoiceCount)

	var totalSales float64
	db.Table("sale_orders").Where("company_id = ? AND state IN ('sale', 'done')", companyID).
		Select("COALESCE(SUM(amount_total), 0)").Scan(&totalSales)

	var mrpCount int64
	db.Table("mrp_productions").Where("company_id = ?", companyID).Count(&mrpCount)

	var posOrderCount int64
	db.Table("pos_orders").Where("company_id = ?", companyID).Count(&posOrderCount)

	var empCount int64
	db.Table("employees").Where("company_id = ? AND status = 'active'", companyID).Count(&empCount)

	var leadCount int64
	db.Table("crm_leads").Where("company_id = ?", companyID).Count(&leadCount)

	var projectCount int64
	db.Table("projects").Where("company_id = ?", companyID).Count(&projectCount)

	var taskCount int64
	db.Table("tasks").Where("company_id = ?", companyID).Count(&taskCount)

	var assetCount int64
	db.Table("assets").Where("company_id = ?", companyID).Count(&assetCount)

	var ticketCount int64
	db.Table("helpdesk_tickets").Where("company_id = ?", companyID).Count(&ticketCount)

	var equipmentCount int64
	db.Table("maintenance_equipments").Where("company_id = ?", companyID).Count(&equipmentCount)

	var maintRequestCount int64
	db.Table("maintenance_requests").Where("company_id = ?", companyID).Count(&maintRequestCount)

	var vehicleCount int64
	db.Table("fleet_vehicles").Where("company_id = ?", companyID).Count(&vehicleCount)

	var leaveCount int64
	db.Table("leave_requests").Where("company_id = ?", companyID).Count(&leaveCount)

	var expenseCount int64
	db.Table("expenses").Where("company_id = ?", companyID).Count(&expenseCount)

	var expenseSheetCount int64
	db.Table("expense_sheets").Where("company_id = ?", companyID).Count(&expenseSheetCount)

	var qcCount int64
	db.Table("quality_checks").Where("company_id = ?", companyID).Count(&qcCount)

	var qaCount int64
	db.Table("quality_alerts").Where("company_id = ?", companyID).Count(&qaCount)

	var approvalCount int64
	db.Table("approval_requests").Where("company_id = ?", companyID).Count(&approvalCount)

	var docCount int64
	db.Table("ir_attachments").Where("company_id = ?", companyID).Count(&docCount)

	var attCount int64
	db.Table("hr_attendances").Where("company_id = ?", companyID).Count(&attCount)

	var openingCount int64
	db.Table("hr_job_openings").Where("company_id = ? AND state = 'open'", companyID).Count(&openingCount)

	var applicantCount int64
	db.Table("hr_applicants").Where("company_id = ?", companyID).Count(&applicantCount)

	var stockInventoryCount int64
	db.Table("stock_inventories").Where("company_id = ?", companyID).Count(&stockInventoryCount)

	var payrollSlipCount int64
	db.Table("payroll_payslips").Where("company_id = ?", companyID).Count(&payrollSlipCount)

	var budgetCount int64
	db.Table("account_budgets").Where("company_id = ?", companyID).Count(&budgetCount)

	var lotCount int64
	db.Table("stock_production_lots").Where("company_id = ?", companyID).Count(&lotCount)

	var reqCount int64
	db.Table("purchase_requisitions").Where("company_id = ?", companyID).Count(&reqCount)

	var currencyCount int64
	db.Table("res_currencies").Where("active = ?", true).Count(&currencyCount)

	var taxInvoiceCount int64
	db.Table("account_tax_invoices").Where("company_id = ?", companyID).Count(&taxInvoiceCount)

	var bankStatementCount int64
	db.Table("account_bank_statements").Where("company_id = ?", companyID).Count(&bankStatementCount)

	var landedCostCount int64
	db.Table("stock_landed_costs").Where("company_id = ?", companyID).Count(&landedCostCount)

	var traceCount int64
	db.Table("stock_lot_traces").Where("company_id = ?", companyID).Count(&traceCount)

	var recallCount int64
	db.Table("stock_lot_recalls").Where("company_id = ?", companyID).Count(&recallCount)

	var subCount int64
	db.Table("subscriptions").Where("company_id = ?", companyID).Count(&subCount)

	var subMRR float64
	db.Table("subscriptions").Where("company_id = ? AND state = 'in_progress'", companyID).
		Select("COALESCE(SUM(total_mrr), 0)").Scan(&subMRR)

	var auditLogCount int64
	db.Table("audit_log_entries").Where("company_id = ?", companyID).Count(&auditLogCount)

	var suspiciousAuditCount int64
	db.Table("audit_log_entries").Where("company_id = ? AND is_suspicious = ?", companyID, true).Count(&suspiciousAuditCount)

	var webhookSubCount int64
	db.Table("webhook_subscriptions").Where("company_id = ? AND is_active = ?", companyID, true).Count(&webhookSubCount)

	var webhookDeliveryCount int64
	db.Table("webhook_deliveries").Where("company_id = ?", companyID).Count(&webhookDeliveryCount)

	var inboundEndpointCount int64
	db.Table("inbound_webhook_endpoints").Where("company_id = ? AND is_active = ?", companyID, true).Count(&inboundEndpointCount)

	var inboundLogCount int64
	db.Table("inbound_webhook_logs").Where("company_id = ?", companyID).Count(&inboundLogCount)

	var icGroupCount, icTxCount, icRunCount int64
	db.Table("intercompany_groups").Count(&icGroupCount)
	db.Table("intercompany_transactions").Count(&icTxCount)
	db.Table("consolidation_runs").Count(&icRunCount)

	var bankAccCount int64
	db.Table("treasury_bank_accounts").Where("company_id = ? AND active = ?", companyID, true).Count(&bankAccCount)

	var forecastCount int64
	db.Table("cash_flow_forecasts").Where("company_id = ?", companyID).Count(&forecastCount)

	var transferCount int64
	db.Table("inter_bank_transfers").Where("company_id = ?", companyID).Count(&transferCount)

	var totalLiquidCash float64
	db.Table("treasury_bank_accounts").Where("company_id = ? AND active = ? AND status = 'active'", companyID, true).
		Select("COALESCE(SUM(current_balance), 0)").Scan(&totalLiquidCash)

	var workCenterCount int64
	var routingCount int64
	var workOrderCount int64
	db.Table("work_centers").Where("company_id = ? OR company_id = 0", companyID).Count(&workCenterCount)
	db.Table("routings").Where("company_id = ? OR company_id = 0", companyID).Count(&routingCount)
	db.Table("work_orders").Where("company_id = ? OR company_id = 0", companyID).Count(&workOrderCount)

	var ebupotPeriodCount int64
	var ebupotSlipCount int64
	var totalEbupotTax float64
	db.Table("tax_ebupot_periods").Where("company_id = ? OR company_id = 0", companyID).Count(&ebupotPeriodCount)
	db.Table("tax_ebupot_withholding_slips").Where("company_id = ? OR company_id = 0", companyID).Count(&ebupotSlipCount)
	db.Table("tax_ebupot_withholding_slips").Where("company_id = ? OR company_id = 0", companyID).Select("COALESCE(SUM(tax_withheld), 0)").Scan(&totalEbupotTax)

	var esignRequestCount, esignSignedCount int64
	db.Table("document_sign_requests").Where("company_id = ? OR company_id = 0", companyID).Count(&esignRequestCount)
	db.Table("document_sign_requests").Where("(company_id = ? OR company_id = 0) AND status = 'signed'", companyID).Count(&esignSignedCount)

	var wmsZoneCount, wmsBinCount, wmsWaveCount int64
	db.Table("wms_zones").Count(&wmsZoneCount)
	db.Table("wms_bin_locations").Count(&wmsBinCount)
	db.Table("wms_wave_pickings").Count(&wmsWaveCount)

	var vendorAccCount, vendorBidCount, vendorASNCount, vendorInvCount int64
	db.Table("vendor_portal_accounts").Count(&vendorAccCount)
	db.Table("vendor_quotation_bids").Count(&vendorBidCount)
	db.Table("vendor_asns").Count(&vendorASNCount)
	db.Table("vendor_invoice_submissions").Count(&vendorInvCount)

	var custAccCount, custOrderCount, custRMACount, custPayCount int64
	db.Table("customer_portal_accounts").Count(&custAccCount)
	db.Table("customer_portal_order_requests").Count(&custOrderCount)
	db.Table("customer_portal_rma_requests").Count(&custRMACount)
	db.Table("customer_portal_payment_notifications").Count(&custPayCount)

	var customsDocCount, customsMutationCount, customsSubcontractCount int64
	db.Table("customs_documents").Count(&customsDocCount)
	db.Table("customs_it_inventory_mutations").Count(&customsMutationCount)
	db.Table("customs_subcontracts").Count(&customsSubcontractCount)

	var workflowDefCount, workflowInstCount int64
	db.Table("workflow_definitions").Count(&workflowDefCount)
	db.Table("workflow_instances").Count(&workflowInstCount)

	var integrationKeyCount, integrationConnectorCount, integrationLogCount int64
	db.Table("integration_api_keys").Count(&integrationKeyCount)
	db.Table("integration_connectors").Count(&integrationConnectorCount)
	db.Table("integration_logs").Count(&integrationLogCount)

	payload := gin.H{
		"products":                  productCount,
		"customers":                 customerCount,
		"vendors":                   vendorCount,
		"sales_orders":              saleCount,
		"purchase_orders":           purchaseCount,
		"pickings":                  pickingCount,
		"invoices":                  invoiceCount,
		"total_sales":               totalSales,
		"mrp_orders":                mrpCount,
		"pos_orders":                posOrderCount,
		"employees":                 empCount,
		"crm_leads":                 leadCount,
		"projects":                  projectCount,
		"tasks":                     taskCount,
		"assets":                    assetCount,
		"tickets":                   ticketCount,
		"equipments":                equipmentCount,
		"maintenance_requests":      maintRequestCount,
		"vehicles":                  vehicleCount,
		"leaves":                    leaveCount,
		"expenses":                  expenseCount,
		"expense_sheets":            expenseSheetCount,
		"quality_checks":            qcCount,
		"quality_alerts":            qaCount,
		"approvals":                 approvalCount,
		"attachments":               docCount,
		"attendances":               attCount,
		"job_openings":              openingCount,
		"applicants":                applicantCount,
		"stock_inventories":         stockInventoryCount,
		"payroll_payslips":          payrollSlipCount,
		"budgets":                   budgetCount,
		"lots":                      lotCount,
		"purchase_requisitions":     reqCount,
		"currencies":                currencyCount,
		"tax_invoices":              taxInvoiceCount,
		"bank_statements":           bankStatementCount,
		"landed_costs":              landedCostCount,
		"traceability_records":      traceCount,
		"recalls":                   recallCount,
		"subscriptions":             subCount,
		"subscription_mrr":          subMRR,
		"audit_logs":                auditLogCount,
		"suspicious_audit_logs":     suspiciousAuditCount,
		"webhook_subscriptions":     webhookSubCount,
		"webhook_deliveries":        webhookDeliveryCount,
		"inbound_webhook_endpoints": inboundEndpointCount,
		"inbound_webhook_logs":      inboundLogCount,
		"intercompany_groups":       icGroupCount,
		"intercompany_transactions": icTxCount,
		"consolidation_runs":        icRunCount,
		"treasury_bank_accounts":    bankAccCount,
		"cash_flow_forecasts":       forecastCount,
		"interbank_transfers":       transferCount,
		"total_liquid_cash":         totalLiquidCash,
		"work_centers":              workCenterCount,
		"routings":                  routingCount,
		"work_orders":               workOrderCount,
		"ebupot_periods":            ebupotPeriodCount,
		"ebupot_slips":              ebupotSlipCount,
		"total_ebupot_tax":          totalEbupotTax,
		"esign_requests":            esignRequestCount,
		"esign_signed":              esignSignedCount,
		"wms_zones":                 wmsZoneCount,
		"wms_bins":                  wmsBinCount,
		"wms_waves":                 wmsWaveCount,
		"vendor_portal_accounts":    vendorAccCount,
		"vendor_quotation_bids":     vendorBidCount,
		"vendor_asns":               vendorASNCount,
		"vendor_invoice_submissions": vendorInvCount,
		"customer_portal_accounts":  custAccCount,
		"customer_b2b_orders":       custOrderCount,
		"customer_rmas":             custRMACount,
		"customer_payment_notifs":   custPayCount,
		"customs_documents":         customsDocCount,
		"customs_mutations":         customsMutationCount,
		"customs_subcontracts":      customsSubcontractCount,
		"workflow_definitions":      workflowDefCount,
		"workflow_instances":        workflowInstCount,
		"integration_api_keys":      integrationKeyCount,
		"integration_connectors":    integrationConnectorCount,
		"integration_logs":          integrationLogCount,
	}
	commandcenter.EnrichDashboardStats(db, companyID, payload)
	c.JSON(http.StatusOK, payload)
}

