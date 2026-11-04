package openapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetSpec generates the OpenAPI 3.1.0 specification for GoERP
func GetSpec(appName, version string) map[string]interface{} {
	if appName == "" {
		appName = "GoERP Enterprise Cloud"
	}
	if version == "" {
		version = "1.43.0"
	}

	return map[string]interface{}{
		"openapi": "3.1.0",
		"info": map[string]interface{}{
			"title":       appName + " — OpenAPI 3.1 & AI Agent Specification",
			"version":     version,
			"description": "Production-grade Open-Source Enterprise Resource Planning (ERP) REST API, Native Model Context Protocol (MCP) JSON-RPC 2.0 Server, Google Cloud A2A (Agent-to-Agent) Protocol, and Multi-Provider AI Intelligence Engine.",
			"contact": map[string]interface{}{
				"name": "GoERP Open Source Community",
				"url":  "https://github.com/antonset1104/erp",
			},
			"license": map[string]interface{}{
				"name": "MIT License",
				"url":  "https://opensource.org/licenses/MIT",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url":         "/",
				"description": "Active GoERP Server Instance",
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"BearerAuth": map[string]interface{}{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
					"description":  "Obtain a JWT token via POST /api/v1/auth/login (Default: admin@erp.local / admin123)",
				},
			},
		},
		"security": []map[string]interface{}{
			{"BearerAuth": []string{}},
		},
		"tags": []map[string]interface{}{
			{"name": "System & Protocols", "description": "Health check, OpenAPI 3.1, Native MCP JSON-RPC 2.0, and Google A2A Agent Discovery"},
			{"name": "AI Agent & GCP Ecosystem", "description": "Multi-Provider AI Agents (Gemini, OpenAI, Claude, DeepSeek), ReAct Tool-Calling, and Google Cloud BigQuery/PubSub/GCS"},
			{"name": "AI Enterprise Intelligence", "description": "Document OCR & 3-Way Matching, Fraud Radar, 30/60/90-Day Cashflow Forecast, ESG Carbon Ledger, and Omnichannel ChatOps"},
			{"name": "Core ERP Modules", "description": "35+ Enterprise Modules: Sales, Purchase, Inventory, Accounting, MRP, POS, HR, CRM, Customs Bonded (CEISA 4.0), Low-Code Studio, Workflow Engine & Integration Hub"},
		},
		"paths": map[string]interface{}{
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"System & Protocols"},
					"summary":     "System Health & Active Modules Check",
					"security":    []map[string]interface{}{},
					"description": "Returns server health, database engine, storage backend, and list of 35+ registered ERP modules.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Server is healthy"},
					},
				},
			},
			"/.well-known/agent.json": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"System & Protocols"},
					"summary":     "Google Cloud A2A (Agent-to-Agent) Discovery Card",
					"security":    []map[string]interface{}{},
					"description": "Publishes GoERP's autonomous agent capabilities, skills, authentication schemes, and endpoints following the Google A2A open standard.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "A2A Agent Card JSON"},
					},
				},
			},
			"/mcp": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"System & Protocols"},
					"summary":     "Native Model Context Protocol (MCP) JSON-RPC 2.0 Server",
					"description": "Standard MCP JSON-RPC 2.0 endpoint supporting initialize, tools/list, tools/call, resources/list, resources/read, and prompts/list for Claude Desktop, Cursor, Gemini CLI, and Antigravity.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "JSON-RPC 2.0 Response"},
					},
				},
			},
			"/api/v1/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"System & Protocols"},
					"summary":     "Authenticate User & Issue JWT",
					"security":    []map[string]interface{}{},
					"description": "Authenticates user credentials and returns a signed JWT Bearer token.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "JWT Token & User Profile"},
					},
				},
			},
			"/api/v1/ai/providers": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"AI Agent & GCP Ecosystem"},
					"summary":     "List Supported AI Agent Providers & Status",
					"description": "Returns configuration status and capabilities for Google Gemini 2.5, Vertex AI, OpenAI GPT-4o, Anthropic Claude 3.7, DeepSeek R1, Cloudflare Workers AI, and Local Heuristic Engine.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Provider list"},
					},
				},
			},
			"/api/v1/ai/tools": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"AI Agent & GCP Ecosystem"},
					"summary":     "Export Standardized AI Agent Tools Manifest",
					"description": "Returns tool schemas compatible with MCP (2025-03-26), OpenAI function calling, Gemini function_declarations, and Anthropic tool_use.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Tool definitions"},
					},
				},
			},
			"/api/v1/ai/agent/execute": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"AI Agent & GCP Ecosystem"},
					"summary":     "Execute Autonomous ReAct AI Agent with Live Database Tool-Calling",
					"description": "Runs a multi-step Reason + Act (ReAct) loop against live ERP database tables and optional Google Cloud BigQuery telemetry sync.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "ReAct trace, tool call results, and synthesized answer"},
					},
				},
			},
			"/api/v1/ai/gcp/status": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"AI Agent & GCP Ecosystem"},
					"summary":     "Google Cloud Platform (GCP) Ecosystem Readiness",
					"description": "Checks Vertex AI, BigQuery OLAP Data Warehouse, Cloud Pub/Sub Event Bus, Cloud Storage (GCS), and Cloud Run status.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "GCP services status"},
					},
				},
			},
			"/api/v1/ai/gcp/sync": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"AI Agent & GCP Ecosystem"},
					"summary":     "Synchronize ERP Data with Google Cloud BigQuery, Pub/Sub, or GCS",
					"description": "Streams analytical snapshots to BigQuery, publishes domain events to Pub/Sub, or archives audit snapshots to GCS.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Sync job confirmation"},
					},
				},
			},
			"/api/v1/ai/a2a/tasks/send": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"System & Protocols"},
					"summary":     "Google A2A Protocol Task Execution",
					"description": "Executes cross-company B2B agent tasks (inventory check, quote negotiation, 3-way match, ESG Scope 3 exchange).",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "A2A Task Artifact Response"},
					},
				},
			},
			"/api/v1/ai/doc-intelligence": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"AI Enterprise Intelligence"},
					"summary":     "AI Document Intelligence (OCR Invoice/e-Faktur & Automated 3-Way Matching)",
					"description": "Extracts structured vendor bill / tax invoice fields and performs automated 3-Way Matching against Purchase Orders (PO) and Goods Receipts (GRN).",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Extracted fields and 3-Way Match verification report"},
					},
				},
			},
			"/api/v1/ai/fraud-radar": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"AI Enterprise Intelligence"},
					"summary":     "Continuous AI Audit & Fraud Anomaly Radar",
					"description": "Scans live Purchase Orders, Vendor Bills, Journal Entries, and Expenses for Split POs, duplicate invoices, weekend postings, and price anomalies.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Fraud radar risk score and anomaly findings"},
					},
				},
				"post": map[string]interface{}{
					"tags":        []string{"AI Enterprise Intelligence"},
					"summary":     "Run Targeted AI Fraud & Compliance Scan",
					"description": "Executes deep forensic scan with custom sensitivity threshold.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Forensic audit report"},
					},
				},
			},
			"/api/v1/ai/cashflow-forecast": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"AI Enterprise Intelligence"},
					"summary":     "Predictive 30/60/90-Day Cashflow Forecasting",
					"description": "Projects cash inflows (AR, Sales, Subscriptions) and outflows (AP, POs, Payroll, Expenses) across 30/60/90-day horizons with Optimistic, Baseline, and Conservative scenarios.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Cashflow forecast & runway metrics"},
					},
				},
			},
			"/api/v1/ai/esg-report": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"AI Enterprise Intelligence"},
					"summary":     "ESG & Carbon Footprint Ledger (Scope 1, 2 & 3 GHG Accounting)",
					"description": "Calculates Scope 1 (Fleet Fuel), Scope 2 (MRP Manufacturing Energy), and Scope 3 (Supply Chain & Logistics) CO2e emissions aligned with GHG Protocol, GRI, EU CSRD, and OJK POJK 51.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "ESG Carbon Footprint & Sustainability Report"},
					},
				},
			},
			"/api/v1/ai/chatops/webhook": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"AI Enterprise Intelligence"},
					"summary":     "Omnichannel ChatOps Gateway (WhatsApp Business, Telegram & Slack Bot)",
					"description": "Processes natural language and slash commands (/omzet, /stok, /approve, /fraud, /esg) from WhatsApp, Telegram, or Slack and returns interactive executive cards.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Formatted ChatOps response and executed ERP action"},
					},
				},
			},
			"/api/v1/ai/demand-sensing": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"AI Enterprise Intelligence"},
					"summary":     "AI Demand Sensing, Dynamic Safety Stock & EOQ Replenishment",
					"description": "Calculates daily demand velocity, dynamic Gaussian safety stock (95% service level), Reorder Point (ROP), and Economic Order Quantity (EOQ).",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Demand metrics and replenishment recommendations"},
					},
				},
			},
			"/api/v1/ai/replenish/auto-po": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"AI Enterprise Intelligence"},
					"summary":     "Execute 1-Click Automated Draft PO Replenishment",
					"description": "Generates draft Purchase Orders with EOQ batch sizes for critical or reorder-needed products.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Created Purchase Order details"},
					},
				},
			},
			"/api/v1/events/stream": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"System & Protocols"},
					"summary":     "Real-Time Server-Sent Events (SSE) Stream",
					"description": "Opens persistent SSE connection receiving live domain events, document changes, and presence updates.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "SSE text/event-stream"},
					},
				},
			},
			"/api/v1/events/presence": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"System & Protocols"},
					"summary":     "Broadcast Document Viewing/Editing Presence",
					"description": "Records user active presence on a specific document to prevent collaborative overwrites.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Active users list"},
					},
				},
			},
			"/api/v1/saas/tenants/register": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"System & Protocols"},
					"summary":     "Self-Service SaaS Multi-Tenant Provisioning",
					"description": "Provisions a new isolated corporate tenant with custom domain, storage quota, and dedicated database schema.",
					"responses": map[string]interface{}{
						"201": map[string]interface{}{"description": "Provisioned tenant profile"},
					},
				},
			},
			"/api/v1/pos/sync-offline-batch": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Reconcile Offline PWA POS Transactions",
					"description": "Atomically processes batched offline transactions, deduces warehouse inventory, and posts GL moves.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Reconciled server POS orders"},
					},
				},
			},
			"/api/v1/ai/voice/command": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"AI Enterprise Intelligence"},
					"summary":     "Hands-Free Voice Ops Assistant",
					"description": "Processes voice commands from shop floor or warehouse, executes ERP tools, and returns spoken synthesized audio text.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Voice intent and spoken audio response"},
					},
				},
			},
			"/api/v1/ai/rfq/tenders": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"AI Enterprise Intelligence"},
					"summary":     "AI Autonomous RFQ Bidding Tender Creation",
					"description": "Initializes multi-vendor RFQ bidding tender with target quantity, budget, and tokenized vendor intake links.",
					"responses": map[string]interface{}{
						"201": map[string]interface{}{"description": "Created RFQ tender profile with vendor portal tokens"},
					},
				},
			},
			"/api/v1/ai/rfq/evaluate/{tender_id}": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"AI Enterprise Intelligence"},
					"summary":     "AI Multi-Criteria Vendor Scorecard & Award Recommendation",
					"description": "Runs Multi-Attribute Utility Theory (MAUT) scoring on price (40%), lead time (25%), payment terms (20%), and warranty (15%).",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Ranked scorecard, savings analysis, and recommended award"},
					},
				},
			},
			"/api/v1/audit/verify-tamper": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"System & Protocols"},
					"summary":     "Cryptographic Merkle-Tree Ledger Audit (Zero-Tamper Verification)",
					"description": "Recomputes sequential SHA-256 block hash chain and binary Merkle tree across GL journal entries, invoices, and payments.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Merkle root hash, tree depth, and cryptographic verification certificate"},
					},
				},
			},
			"/api/v1/ai/digital-twin/simulate": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"AI Enterprise Intelligence"},
					"summary":     "Supply Chain Digital Twin & What-If Scenario Simulator",
					"description": "Runs stress simulations on BOM costs, gross margins, stockout risks, and delivery delays given lead-time, freight, and FX shocks.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Simulated COGS inflation, margin erosion, at-risk SKUs, and strategic mitigation playbooks"},
					},
				},
			},
			"/api/v1/security/rbac-matrix": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"System & Protocols"},
					"summary":     "Field-Level RBAC Permission Matrix & Masking Rules",
					"description": "Returns module permissions and granular field-level masking rules (e.g. standard_price hidden for warehouse operators).",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Enterprise role profiles and field masking configuration"},
					},
				},
			},
			"/api/v1/bi/dashboards": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Visual Drag-and-Drop BI Dashboards & Live Widgets",
					"description": "Lists configured departmental and C-suite dashboards with real-time aggregated metrics (KPI cards, bar/line/pie charts).",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Configured dashboards with live populated widget data"},
					},
				},
			},
			"/api/v1/ai/pricing/recommend": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"AI Enterprise Intelligence"},
					"summary":     "AI Autonomous Dynamic Pricing & Elasticity Optimizer",
					"description": "Calculates profit-maximizing price using Price Elasticity of Demand (PED), margin floors, inventory aging, and customer segmentation.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Optimal price, elasticity classification, and volume discount tiers"},
					},
				},
			},
			"/api/v1/treasury/fx-exposure": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Cross-Border FX Exposure Matrix & Stress-Test Simulator",
					"description": "Calculates multi-currency Net Open Positions (USD/EUR/SGD vs IDR), unhedged VaR, and automated forward contract recommendations.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Currency positions, sensitivity shocks, and active hedge contracts"},
					},
				},
			},
			"/api/v1/iot/telemetry/ingest": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Predictive Equipment Maintenance & IoT Telemetry Ingestion",
					"description": "Ingests vibration (ISO 10816), operating temperature, and acoustic metrics to compute Remaining Useful Life (RUL) and auto-trigger maintenance requests.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Machine health score, RUL hours, anomaly indicators, and auto-created maintenance orders"},
					},
				},
			},
			"/api/v1/esg/vendor-scorecards": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"AI Enterprise Intelligence"},
					"summary":     "Supplier Sustainability Passport & Scope 3 Vendor Scoring",
					"description": "Audits vendor carbon intensity (kg CO2e/unit), ISO 14001/50001 certifications, recycled packaging, and green logistics scores.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "List of vendor green scorecards and procurement bonus factors"},
					},
				},
			},
			"/api/v1/ai/knowledge/query": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"AI Enterprise Intelligence"},
					"summary":     "AI Knowledge Base Semantic Search & Customer Support Copilot (RAG)",
					"description": "Retrieves semantic knowledge chunks across internal SOPs, warranties, and product manuals to synthesize context-aware answers.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Semantic chunks, confidence score, and synthesized answer"},
					},
				},
			},
			"/api/v1/ai/contracts/analyze": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"AI Enterprise Intelligence"},
					"summary":     "AI Contract Lifecycle Management (CLM) & Clause Risk Redlining",
					"description": "Extracts liability caps, indemnification, liquidated damages, and auto-renewal terms with 0-100 risk scoring and redlines.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Risk score, detected high-risk clauses, and recommended counter-terms"},
					},
				},
			},
			"/api/v1/wms/pack-optimizer": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Smart 3D Bin Packing & Pallet Load Optimizer",
					"description": "Computes 3D cuboid packing coordinates (x, y, z), rotation, layer stability, and container volume utilization for pallets and containers.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Container volume efficiency, placed item coordinates, and 3D visual preview"},
					},
				},
			},
			"/api/v1/hr/attrition-radar": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Predictive Employee Attrition & AI Talent Retention Radar",
					"description": "Calculates employee flight risk (0-100) based on overtime burnout, leave deficit, promotion stagnation, and compensation benchmarks.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Ranked at-risk employees and automated retention playbooks"},
					},
				},
			},
			"/api/v1/account/reconciliation/auto-match": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Multi-Horizon Bank Reconciler with ML Fuzzy Statement Matching",
					"description": "Fuzzy string matching with BI-FAST fee tolerance (+/- Rp 2,500/Rp 6,500) and 1-click batch payment reconciliation.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Matched bank lines, invoice candidates, and confidence score"},
					},
				},
			},
			"/api/v1/edi/ingest": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"System & Protocols"},
					"summary":     "Universal EDI Gateway (ANSI X12 & UN/EDIFACT)",
					"description": "Ingests EDI 850 / ORDERS (Purchase Orders), EDI 810 (Invoices), EDI 856 (ASNs) with automated draft Sales Order creation.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Parsed segments, document type, line items, and staged ERP record ID"},
					},
				},
			},
			"/api/v1/wms/cross-dock/match": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "AI Automated Cross-Docking & Wave Dispatch Director",
					"description": "Matches incoming warehouse stock receipts directly to waiting customer backorders, cutting warehouse dwell time by 75%.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Cross-dock allocations, time saved, and staging dock routing"},
					},
				},
			},
			"/api/v1/accounting/consolidation/report": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Multi-Entity Intercompany Consolidated Financials (IFRS 10 / PSAK 65)",
					"description": "Generates real-time consolidated multi-subsidiary Balance Sheet and P&L with automated intra-group elimination entries.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Consolidated financials, elimination entries, and balanced status"},
					},
				},
			},
			"/api/v1/ai/supply-chain/bullwhip-index": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"AI Enterprise Intelligence"},
					"summary":     "Autonomous Multi-Tier Bullwhip Effect Guard & Lead-Time Anomaly Detector",
					"description": "Computes demand variance amplification across supply chain echelons (Tier 0 to Tier 3) and predicts component stockouts.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Bullwhip ratio, echelon amplifications, and buffer dampening playbooks"},
					},
				},
			},
			"/api/v1/expense/audit/evaluate": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Corporate Expense Fraud & Policy Governance AI",
					"description": "Detects smurfing/split claims, duplicate receipt hashes, and non-working day policy violations with automated audit holds.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Audit risk scores, detected violations, and policy actions"},
					},
				},
			},
			"/api/v1/wms/rma/diagnose": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "AI Reverse Logistics & RMA Warranty Diagnostic Engine",
					"description": "Validates warranty entitlement, classifies defects, and computes optimal economic disposition (Refurbish vs Scrap vs Restock).",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Warranty status, repair cost vs salvage value, and recommended disposition"},
					},
				},
			},
			"/api/v1/mrp/line-balancing/analyze": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "AI Production Line Balancing & OEE Bottleneck Optimizer",
					"description": "Analyzes takt time vs station cycle times, line balance efficiency, and OEE to identify bottlenecks and WIP pile-ups.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Line balance efficiency, bottleneck identification, and rebalancing recommendations"},
					},
				},
			},
			"/api/v1/fleet/route-optimizer": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Multi-Carrier Last-Mile Dispatch & Route Geometry Optimizer (VRPTW)",
					"description": "Clusters delivery waypoints, assigns internal vs 3PL fleet, sequences stops, and projects fuel/cost savings.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Optimized route clusters, waypoints sequence, and cost breakdown"},
					},
				},
			},
			"/api/v1/account/credit-risk/customer/{id}": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Dynamic Customer Credit Limit & Bad Debt Risk Scorer",
					"description": "Evaluates Altman Z-Score, default probability, DSO, and invoice aging matrix with automated Credit Hold rules.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Customer credit profile, risk rating, aging buckets, and recommendations"},
					},
				},
			},
			"/api/v1/asset/carbon-depreciation/matrix": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Fixed Asset Depreciation & Carbon Tax Cap-and-Trade Ledger",
					"description": "Calculates tax depreciation schedules, lifetime energy emissions, regulatory allowance caps, and carbon tax liability.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Asset net book values, annual emissions, cap variances, and tax liabilities"},
					},
				},
			},
			"/api/v1/graphql": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"System & Protocols"},
					"summary":     "Universal Pure-Go Headless GraphQL Query Engine",
					"description": "Executes nested relational GraphQL queries (sales orders, inventory, partners, financials) with schema introspection.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Standard GraphQL data response payload"},
					},
				},
			},
			"/api/v1/hr/roster-optimizer/generate": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "AI Automated Shift Scheduling & Labor Rostering Optimizer",
					"description": "Constraint satisfaction scheduler adhering to PP 35/2021 overtime ceilings, 11-hour rest intervals, and skill requirements.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Generated shift schedule, labor law compliance score, and overtime breakdown"},
					},
				},
			},
			"/api/v1/wms/inventory-balancing/evaluate": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Autonomous Multi-Warehouse Stock Equalizer & Inter-Branch Balancer",
					"description": "Calculates Days-of-Supply, deficit/surplus nodes, and recommends consolidated inter-warehouse transfers (WH/INT).",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Stock evaluation report, transfer recommendations, and working capital savings"},
					},
				},
			},
			"/api/v1/crm/clv-matrix": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Predictive Customer Lifetime Value (CLV) & Dynamic Churn Preventer",
					"description": "BG/NBD and Gamma-Gamma predictive CLV, RFM decay metrics, and automated VIP win-back retention playbooks.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Customer equity profiles, churn risk tiers, and retention triggers"},
					},
				},
			},
			"/api/v1/security/vault/shard-keys": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"System & Protocols"},
					"summary":     "Zero-Trust Cryptographic Secret Sharing & Quorum Vault",
					"description": "Shamir's Secret Sharing (k-of-n threshold) for critical ERP operations requiring C-suite quorum consensus.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Generated key shards, master fingerprint, and quorum policy"},
					},
				},
			},
			"/api/v1/asset/tracking/radar-map": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Real-Time RFID / BLE Asset Tracking Radar & Geofence Demurrage Engine",
					"description": "Tracks physical asset zones (dock, production, customer site), misplaced asset alerts, and geofence demurrage penalties.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Active asset radar map, zone distributions, and demurrage accruals"},
					},
				},
			},
			"/api/v1/accounting/continuous-close/run": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Continuous Financial Virtual Close & Smart Accruals Engine",
					"description": "24/7 continuous subledger reconciliation, automated GRNI accruals, and audit-ready fast close readiness.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Virtual close report, smart accruals, and days saved"},
					},
				},
			},
			"/api/v1/vmi/replenishment-forecast": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Vendor-Managed Inventory (VMI) Consignment & Replenishment Gateway",
					"description": "Consignment min-max stock bands, automated supplier self-dispatch advice, and pay-as-consumed auto-billing.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "VMI forecast report, replenishment urgency, and consignment values"},
					},
				},
			},
			"/api/v1/customs/fta-origin/evaluate": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Automated FTA Rules-of-Origin & Preferential Tariff Optimizer",
					"description": "Evaluates multi-level BOMs for Regional Value Content (RVC >= 40%) across ATIGA Form D, Form E, and RCEP.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "FTA qualification results, origin certificates, and projected duty savings"},
					},
				},
			},
			"/api/v1/system/cqrs/health": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"System & Protocols"},
					"summary":     "Pure-Go CQRS Database Read-Replica Router & Cluster Health",
					"description": "Monitors multi-region database master-replica topology, replication lag, and OLTP write isolation.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "CQRS cluster status, node health, and read traffic offload percentage"},
					},
				},
			},
			"/api/v1/ehs/safety-radar": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "AI Workplace Safety, Incident Radar & ISO 45001 Compliance",
					"description": "Lost Time Injury Frequency Rate (LTIFR), Severity Rate (SR), zero-harm streaks, and automated 5-Why root cause CAPA.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "EHS safety report, incident statistics, and ISO 45001 compliance audit trail"},
					},
				},
			},
			"/api/v1/treasury/cash-pool/sweep": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Global Treasury Cash Pooling, In-House Banking & Multilateral Netting",
					"description": "End-of-day zero-balance account (ZBA) sweeps, intercompany arm's-length interest, and multilateral cross-border netting.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Sweep execution report, liquidity concentration, and interest accrual metrics"},
					},
				},
			},
			"/api/v1/costing/landed-cost/allocate": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Automated Landed Cost Allocation & Multi-Modal Freight Accrual",
					"description": "Capitalizes freight, customs duties, and terminal fees into inventory valuation layers with retroactive COGS adjustments.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Allocated product line costs, new unit valuation, and GL journal entry reference"},
					},
				},
			},
			"/api/v1/esg/scope3/calculate-product-passport": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Supply Chain Scope 3 Carbon Accounting & EU CBAM Passport",
					"description": "Calculates cradle-to-gate embedded emissions (Scope 1, 2, 3) per ton and generates EU CBAM export declarations.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Product carbon passport, supplier Scope 3 breakdown, and CBAM obligation deduction"},
					},
				},
			},
			"/api/v1/field-service/dispatch/recommend": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Field Service Lightning & Geospatial Dispatching Radar",
					"description": "Haversine distance routing, certified technician skill matching, and on-board van spare parts availability scoring.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Ranked dispatch recommendations, ETA, and SLA window advice"},
					},
				},
			},
			"/api/v1/pricing/dynamic/evaluate": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Algorithmic Dynamic Pricing & Margin Guard Intelligence",
					"description": "Calculates demand-elastic repricing with competitor intelligence while strictly enforcing gross margin floor protection.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Suggested dynamic price, margin achieved, and omnichannel sync status"},
					},
				},
			},
			"/api/v1/mrp/andon/trigger-event": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Smart Factory Floor Andon Visual Control & Shopfloor Anomaly Gateway",
					"description": "Real-time workstation alert triggering (machine breakdown, material starvation), automated forklift dispatch, and OEE line pacing.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Active Andon event, runner task assignment, and line severity status"},
					},
				},
			},
			"/api/v1/trade-finance/lc/audit-documents": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Letters of Credit (L/C) & Trade Finance Zero-Discrepancy Engine",
					"description": "Audits export shipping documents against UCP 600 Letter of Credit clauses, verifying dates, amounts, and zero discrepancy risk.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Zero discrepancy certification, discrepancy risk score, and bank acceptance probability"},
					},
				},
			},
			"/api/v1/treasury/dynamic-discounting/offer": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Supplier Early-Payment Dynamic Discounting & Supply Chain Finance",
					"description": "Sliding-scale early settlement discounts for vendor invoices, converting AP cash outflow into high-yield treasury returns.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Dynamic discount offer, net payable amount, and annualized yield APR"},
					},
				},
			},
			"/api/v1/iot/vibration/analyze-fft": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Predictive Machine Vibration FFT & Remaining Useful Life (RUL) Radar",
					"description": "Fast Fourier Transform (FFT) harmonic spectral analysis, ISO 10816 severity zoning, and Weibull Remaining Useful Life forecasting.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Vibration analysis result, spectral peaks, remaining hours, and auto-reserved spare part"},
					},
				},
			},
			"/api/v1/hr/travel/evaluate-policy": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Corporate Travel Policy Engine & Multi-Tier Per-Diem Matrix",
					"description": "Audits corporate travel booking compliance (advance window, cabin class, hotel ceiling) and calculates city-tier per-diem budgets.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Compliance evaluation, audit checks, per-diem allowance, and total trip budget"},
					},
				},
			},
			"/api/v1/tax/transfer-pricing/audit-margins": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Cross-Border Transfer Pricing & OECD BEPS Action 13 Hub",
					"description": "Conducts arm's-length interquartile benchmark analysis (Q1-Q3), TNMM operating margin audits, and transfer pricing tax adjustment calculations.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Interquartile benchmarks, arm's length compliance verdict, and required median adjustments"},
					},
				},
			},
			"/api/v1/accounting/leases/schedule": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "High-Value Asset Lease Accounting Engine (IFRS 16 / PSAK 73)",
					"description": "Calculates present value discounting at IBR, monthly liability amortization, straight-line ROU asset depreciation, and dual-column schedules.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Initial present value recognition, total interest, and full monthly amortization schedule"},
					},
				},
			},
			"/api/v1/wms/yard/dock-radar": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Yard Management System (YMS) & Carrier Dock Appointment Scheduler",
					"description": "Real-time dock door occupancy radar, carrier gate check-in, detention fee risk detection, and turn-time analytics.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Dock radar status, total doors, occupied doors, active trailers, and detention alerts"},
					},
				},
			},
			"/api/v1/cpq/configure-product": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Dynamic Rule-Based Product Configurator & CPQ Engine",
					"description": "Propositional logic constraint checks (Requires, Excludes), margin floor enforcement, and automated multi-level production BOM synthesis.",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "Validation result, rule violations, cost and pricing totals, and achieved gross margin"},
					},
				},
			},
			"/api/v1/grc/ethics/report-incident": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Core ERP Modules"},
					"summary":     "Corporate Whistleblower & Anonymous Ethics Hotline Vault (ISO 37002)",
					"description": "Zero-knowledge anonymous incident disclosure, cryptographic SHA-256 evidence hashing, and independent audit committee investigation workflow.",
					"responses": map[string]interface{}{
						"201": map[string]interface{}{"description": "Case ID, anonymous tracking token, evidence digests, and triage risk score"},
					},
				},
			},
		},
	}
}

// RegisterRoutes registers OpenAPI 3.1 JSON spec and Interactive API Explorer HTML UI
func RegisterRoutes(r *gin.Engine, appName, version string) {
	specHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, GetSpec(appName, version))
	}

	r.GET("/openapi.json", specHandler)
	r.GET("/api/v1/openapi.json", specHandler)

	r.GET("/api/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(interactiveDocsHTML))
	})
}

const interactiveDocsHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>GoERP — OpenAPI 3.1 & AI Protocol Explorer</title>
  <style>
    :root { --bg: #0f172a; --card: #1e293b; --border: #334155; --text: #f8fafc; --muted: #94a3b8; --accent: #38bdf8; --green: #22c55e; --purple: #a855f7; }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body { font-family: system-ui, -apple-system, sans-serif; background: var(--bg); color: var(--text); padding: 24px; line-height: 1.5; }
    .container { max-width: 1180px; margin: 0 auto; }
    header { display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid var(--border); padding-bottom: 16px; margin-bottom: 24px; flex-wrap: wrap; gap: 12px; }
    h1 { font-size: 1.5rem; color: var(--accent); }
    .badges { display: flex; gap: 8px; flex-wrap: wrap; }
    .badge { padding: 4px 10px; border-radius: 999px; font-size: 0.75rem; font-weight: 600; background: #1e3a8a; color: #93c5fd; }
    .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }
    @media (max-width: 900px) { .grid { grid-template-columns: 1fr; } }
    .card { background: var(--card); border: 1px solid var(--border); border-radius: 10px; padding: 16px; margin-bottom: 16px; }
    .endpoint { display: flex; align-items: center; justify-content: space-between; padding: 10px 12px; border: 1px solid var(--border); border-radius: 8px; margin-bottom: 8px; cursor: pointer; transition: border-color 0.15s; }
    .endpoint:hover { border-color: var(--accent); }
    .method { font-weight: 700; font-size: 0.75rem; padding: 3px 8px; border-radius: 4px; margin-right: 10px; }
    .GET { background: rgba(34,197,94,0.2); color: #4ade80; }
    .POST { background: rgba(56,189,248,0.2); color: #38bdf8; }
    input, textarea, select { width: 100%; background: #090d16; color: var(--text); border: 1px solid var(--border); border-radius: 6px; padding: 8px 10px; font-family: monospace; font-size: 0.85rem; margin-top: 6px; margin-bottom: 12px; }
    button { background: var(--accent); color: #0f172a; border: none; padding: 9px 16px; border-radius: 6px; font-weight: 700; cursor: pointer; }
    pre { background: #090d16; padding: 12px; border-radius: 8px; overflow-x: auto; font-size: 0.8rem; color: #a7f3d0; max-height: 460px; border: 1px solid var(--border); }
  </style>
</head>
<body>
  <div class="container">
    <header>
      <div>
        <h1>GoERP — OpenAPI 3.1, Native MCP & Google A2A Explorer</h1>
        <p style="color:var(--muted);font-size:0.9rem;">Interactive REST, Model Context Protocol (JSON-RPC 2.0), and Agent-to-Agent Console</p>
      </div>
      <div class="badges">
        <span class="badge">OpenAPI 3.1.0</span>
        <span class="badge">MCP 2025-03-26</span>
        <span class="badge">Google A2A</span>
        <a href="/openapi.json" target="_blank" class="badge" style="text-decoration:none;background:#065f46;color:#6ee7b7;">Raw /openapi.json</a>
        <a href="/.well-known/agent.json" target="_blank" class="badge" style="text-decoration:none;background:#581c87;color:#e9d5ff;">/.well-known/agent.json</a>
        <a href="/" class="badge" style="text-decoration:none;background:#334155;color:#f8fafc;">← Back to GoERP Dashboard</a>
      </div>
    </header>
    <div class="grid">
      <div>
        <div class="card">
          <h3 style="margin-bottom:10px;">1. Select Endpoint to Test</h3>
          <div id="endpointsList">Loading OpenAPI 3.1 specification...</div>
        </div>
      </div>
      <div>
        <div class="card">
          <h3 style="margin-bottom:10px;">2. Live API & Protocol Console</h3>
          <label style="font-size:0.8rem;color:var(--muted);">JWT Bearer Token (auto-loaded from localStorage or login)</label>
          <input id="jwtToken" placeholder="Optional JWT Token (click Authenticate Default Admin below)" />
          <div style="display:flex;gap:8px;margin-bottom:12px;">
            <button onclick="loginDefaultAdmin()" style="background:#334155;color:#fff;font-size:0.78rem;">Authenticate Default Admin</button>
            <button onclick="sendRequest()">Send Live Request</button>
          </div>
          <label style="font-size:0.8rem;color:var(--muted);">Method & Path</label>
          <div style="display:flex;gap:8px;">
            <select id="reqMethod" style="width:110px;"><option>GET</option><option>POST</option></select>
            <input id="reqPath" value="/.well-known/agent.json" />
          </div>
          <label style="font-size:0.8rem;color:var(--muted);">JSON Request Body (for POST)</label>
          <textarea id="reqBody" rows="5">{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}</textarea>
          <label style="font-size:0.8rem;color:var(--muted);">Live Server Response</label>
          <pre id="respBox">// Click any endpoint on the left or "Send Live Request" to inspect response</pre>
        </div>
      </div>
    </div>
  </div>
  <script>
    const savedToken = localStorage.getItem('erp_token') || '';
    if (savedToken) document.getElementById('jwtToken').value = savedToken;

    const samplePayloads = {
      '/mcp': JSON.stringify({jsonrpc: "2.0", id: 1, method: "tools/list", params: {}}, null, 2),
      '/api/v1/ai/agent/execute': JSON.stringify({prompt: "Check warehouse stock and financial revenue summary", provider: "auto", enable_tools: true}, null, 2),
      '/api/v1/ai/doc-intelligence': JSON.stringify({document_type: "vendor_bill", vendor_name: "PT Baja Nusantara", invoice_number: "INV-2026-0881", document_text: "NPWP: 01.234.567.8-012.000 Faktur: 010.002-26.12345678 Subtotal: 10000000 PPN 11%: 1100000 Total: 11100000", subtotal: 10000000, tax_amount: 1100000, total_amount: 11100000}, null, 2),
      '/api/v1/ai/chatops/webhook': JSON.stringify({channel: "whatsapp", sender: "+62811998877", message: "/omzet"}, null, 2),
      '/api/v1/ai/a2a/tasks/send': JSON.stringify({id: "task-b2b-001", skill: "inventory-availability-check", message: {role: "user", parts: [{type: "text", text: "Check stock availability and pricing for B2B partner order"}] }}, null, 2)
    };

    fetch('/openapi.json').then(r => r.json()).then(spec => {
      const container = document.getElementById('endpointsList');
      container.innerHTML = '';
      Object.entries(spec.paths || {}).forEach(([path, methods]) => {
        Object.entries(methods).forEach(([method, info]) => {
          const m = method.toUpperCase();
          const div = document.createElement('div');
          div.className = 'endpoint';
          div.innerHTML = '<div><span class="method ' + m + '">' + m + '</span><code>' + path + '</code><div style="font-size:0.78rem;color:#94a3b8;margin-top:4px;">' + (info.summary || '') + '</div></div>';
          div.onclick = () => {
            document.getElementById('reqMethod').value = m;
            document.getElementById('reqPath').value = path;
            if (samplePayloads[path]) document.getElementById('reqBody').value = samplePayloads[path];
            sendRequest();
          };
          container.appendChild(div);
        });
      });
    });

    async function loginDefaultAdmin() {
      const res = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({email: 'admin@erp.local', password: 'admin123'})
      });
      const data = await res.json();
      if (data.token) {
        document.getElementById('jwtToken').value = data.token;
        localStorage.setItem('erp_token', data.token);
        document.getElementById('respBox').textContent = 'Authenticated as admin@erp.local! Token populated.';
      }
    }

    async function sendRequest() {
      const method = document.getElementById('reqMethod').value;
      const path = document.getElementById('reqPath').value;
      const token = document.getElementById('jwtToken').value.trim();
      const headers = {'Content-Type': 'application/json'};
      if (token) headers['Authorization'] = 'Bearer ' + token;
      const opts = { method, headers };
      if (method === 'POST') opts.body = document.getElementById('reqBody').value;
      document.getElementById('respBox').textContent = 'Executing ' + method + ' ' + path + '...';
      try {
        const res = await fetch(path, opts);
        const text = await res.text();
        try {
          document.getElementById('respBox').textContent = JSON.stringify(JSON.parse(text), null, 2);
        } catch (_) {
          document.getElementById('respBox').textContent = text;
        }
      } catch (err) {
        document.getElementById('respBox').textContent = 'Error: ' + err.message;
      }
    }
  </script>
</body>
</html>`
