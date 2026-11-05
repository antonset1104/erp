# GoERP: Enterprise Resource Planning in Golang

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![CI Status](https://img.shields.io/badge/CI-Passing-success?style=flat&logo=githubactions)](https://github.com/antonset1104/erp/actions)
[![Modules](https://img.shields.io/badge/Modules-50_Enterprise-indigo)](docs/FUNCTIONAL_DOCUMENTATION.md)
[![Release](https://img.shields.io/badge/Release-v1.80.0-emerald)](https://github.com/antonset1104/erp/releases)
[![Open Source](https://img.shields.io/badge/Open_Source-Community_Driven-purple)](docs/OPEN_SOURCE_GUIDE.md)

**GoERP** is a modern, high-performance, enterprise-grade Enterprise Resource Planning (ERP) platform built natively with **Golang**. Engineered with a clean **Modular Monolith** architecture comprising **50 integrated business modules**, GoERP delivers single-binary static compilation, native Go concurrency, ultra-low memory overhead (~25 MB idle RAM), and strict transactional integrity for mission-critical enterprise workloads.

---

## 📚 Official Documentation Suite

Comprehensive technical, functional, and user manuals are maintained in the `docs/` directory:

- 🌐 **[Open Source Project Guide](docs/OPEN_SOURCE_GUIDE.md)** — Open source governance model, GitHub repository hygiene, SemVer release workflows, and Apache 2.0 licensing policies.
- 📘 **[Technical Documentation](docs/TECHNICAL_DOCUMENTATION.md)** — Architectural design, clean 50-module layout, ERD schemas, REST and Open RPC API specifications, JWT & M2M authentication, and deployment blueprints.
- 📗 **[Functional Documentation](docs/FUNCTIONAL_DOCUMENTATION.md)** — End-to-end business workflows (O2C, P2P, MRP II, General Ledger, HRIS & Payroll, e-Faktur/e-Bupot tax compliance, Customs Bonded Zones, Low-Code Studio, BPMN 2.0 Engine, and Integration Hub).
- 📙 **[User Documentation](docs/USER_DOCUMENTATION.md)** — Step-by-step operational handbook for Administrators, Sales, Procurement, Warehouse/WMS, Manufacturing, Finance/Accounting, and HR.
- 🤝 **[Contributing Guidelines](CONTRIBUTING.md)** — Pull request lifecycle, code standards, test execution, and Conventional Commits conventions.
- 📜 **[Code of Conduct](CODE_OF_CONDUCT.md)** — Inclusive community standards (Contributor Covenant v2.1).
- 🛡️ **[Security Policy](SECURITY.md)** — Responsible vulnerability disclosure procedures, supported versions, and patch release SLAs.

---

## ⚡ Executive Command Center & Continuous Improvements

GoERP includes built-in executive productivity tools to accelerate day-to-day operations across 50 enterprise modules:

- **🔍 Universal Command Palette & Omni-Search (`Ctrl+K`)**: Instant unified search across all **50 ERP modules**, quick operational actions, and live database lookups (*Products*, *Partners*, *Sales Orders*, *Purchase Orders*, *Invoices*) via `GET /api/v1/search?q=...`.
- **🧭 Instant 50-Module Sidebar Filter (`/` Shortcut)**: Real-time search filter at the top of the sidebar navigation to instantly filter modules with a live counter.
- **🚀 1-Click End-to-End Business Cycle Simulator (`POST /api/v1/dashboard/simulate-cycle`)**: Automated one-click execution of the entire enterprise cycle (*Procure-to-Pay* `PO` $\rightarrow$ `WH/IN`, *Order-to-Cash* `SO` $\rightarrow$ `WH/OUT`, and *Balanced General Ledger Posting* `INV` with 11% VAT) with an interactive 5-step audit log.
- **🔔 Actionable Notification & Approval Center (`GET /api/v1/dashboard/notifications`)**: Intelligent notification bell aggregating pending approvals, active BPMN workflows, draft documents, and security alerts.
- **📊 Executive Financial Pulse Strip & 50-Module Health Matrix**: Live liquidity tracking across Treasury bank accounts, Accounts Receivable (AR), Accounts Payable (AP), CRM Pipeline revenue, Subscription MRR, live transaction feeds, and 1-click executive KPI CSV export (`GET /api/v1/dashboard/export-summary`).
- **📑 OpenAPI 3.1 & Interactive API Explorer (`/api/docs`)**: Native browser console to test all REST endpoints, Model Context Protocol (MCP), and Google A2A tasks.
- **🔌 Native Model Context Protocol (MCP) JSON-RPC 2.0 Server (`/mcp`)**: Zero-code integration for Claude Desktop, Cursor, Gemini CLI, and Antigravity.
- **🤖 Google Cloud Agent-to-Agent (A2A) Discovery Card (`/.well-known/agent.json`)**: Autonomous B2B agent negotiation for inventory availability, 3-way matching, and Scope 3 ESG carbon exchange.
- **🧾 AI Document Intelligence & Automated 3-Way Matching**: Multimodal OCR invoice/e-Faktur extraction with automated PO and Goods Receipt (GRN) verification.
- **🛡️ AI Fraud Radar & Predictive Cashflow (30/60/90 Days)**: Real-time anomaly detection (Split POs, high-value postings) and CFO liquidity runway projections.
- **🌱 ESG & Carbon Footprint Ledger**: Automated GHG Scope 1 (Fleet), Scope 2 (MRP Energy), and Scope 3 (Logistics) accounting with tree offset equivalents.
- **💬 Omnichannel ChatOps Gateway**: WhatsApp Business, Telegram, and Slack bot processing executive commands (`/omzet`, `/stok`, `/approve`, `/fraud`, `/esg`).
- **📊 AI Demand Sensing & Smart Replenishment**: Daily velocity forecasting, Dynamic Gaussian Safety Stock (95% service level), Reorder Point (ROP), and Economic Order Quantity (EOQ) with 1-click auto-PO creation.
- **⚡ Real-Time SSE Collaboration & Live Presence**: Server-Sent Events stream (`/api/v1/events/stream`) and document editing lock indicators to prevent concurrent overwrite collisions.
- **🏢 Enterprise SaaS Multi-Tenancy Engine**: Database-per-tenant and schema isolation with automatic routing by subdomain or `X-Tenant-ID` header, and self-service registration.
- **📴 Offline-First PWA & Barcode POS Sync**: Service worker app shell caching (`/web/sw.js`), web app manifest (`/web/manifest.json`), and batch background synchronization for offline transactions.
- **🎙️ Hands-Free Voice Ops Assistant**: Speech recognition & synthesis voice assistant executing warehouse inventory checks, financial summaries, and approvals via voice commands.
- **⚔️ AI Autonomous Supplier Negotiation & RFQ Bidding Wars (`POST /api/v1/ai/rfq/tenders`, `GET /api/v1/ai/rfq/evaluate/:id`)**: Tokenized vendor bid portal, automated bid intake, and Multi-Attribute Utility Theory (MAUT) scorecard matrix (Price 40%, Lead Time 25%, Payment Terms 20%, Warranty 15%) with 1-click auto-PO tender awarding.
- **🔒 Cryptographic Merkle-Tree Ledger Audit (Tamper-Proof Financial Chain) (`GET /api/v1/audit/verify-tamper`)**: Sequential SHA-256 linear hash chaining and binary Merkle tree recomputation over General Ledger entries (`account_moves`), payments, and purchase orders, generating mathematical zero-tamper certificates.
- **🌐 Supply Chain Digital Twin & "What-If" Scenario Simulator (`POST /api/v1/ai/digital-twin/simulate`)**: Monte Carlo stress test engine simulating supplier lead-time shocks, freight rate spikes (+18.5%), and USD/IDR FX volatility on BOM unit costs, gross margins, and customer delivery orders with actionable AI playbooks.
- **🛡️ Enterprise Single Sign-On (SAML 2.0 / OIDC) & Field-Level RBAC Matrix (`GET /api/v1/security/rbac-matrix`, `GET /api/v1/security/sso/config`)**: Federated corporate identity (Okta, Azure AD, Google Workspace) and dynamic field-level masking (e.g. `standard_price`/`cost_price` masked for warehouse operators).
- **📊 Visual Drag-and-Drop BI Dashboard & Custom Report Builder (`GET /api/v1/bi/dashboards`, `POST /api/v1/bi/reports/export`)**: Dynamic KPI cards, gauges, charts (Bar/Line/Pie) backed by live DB metrics, and user-configurable custom PDF/Excel executive reports.
- **🏷️ AI Autonomous Dynamic Pricing & Elasticity Optimizer (`POST /api/v1/ai/pricing/recommend`)**: Profit-maximizing price recommendations using Price Elasticity of Demand (PED), margin floors (20%+), inventory velocity, and multi-tier volume discounts.
- **💱 Cross-Border FX Exposure & Automated Treasury Hedging (`GET /api/v1/treasury/fx-exposure`, `POST /api/v1/treasury/hedge-contracts`)**: Multi-currency Net Open Position (NOP) tracking (USD/EUR/SGD vs IDR), unhedged Value-at-Risk (VaR), sensitivity stress-testing (+/- 10%), and automated forward contracts.
- **🏭 Predictive Equipment Maintenance & IoT Telemetry Gateway (`POST /api/v1/iot/telemetry/ingest`, `GET /api/v1/iot/predictive-health`)**: Industrial sensor ingestion (vibration ISO 10816, temperature, motor current), Remaining Useful Life (RUL) estimation, and auto-creation of maintenance work orders.
- **🌍 Supplier Sustainability Passport & Scope 3 Vendor Scoring (`GET /api/v1/esg/vendor-scorecards`, `POST /api/v1/esg/vendor-assessments`)**: Green credentials passport tracking emission intensity (kg CO2e/unit), ISO 14001/50001 certifications, recycled packaging, and procurement bonus scoring.
- **🧠 AI Knowledge Base Semantic Search & Customer Support Copilot (RAG) (`POST /api/v1/ai/knowledge/query`, `POST /api/v1/ai/helpdesk/copilot-draft`)**: Semantic vector RAG retrieval across corporate SOPs, warranties, and product specs, generating automated, context-aware Helpdesk responses.
- **📜 AI Automated Contract Lifecycle Management (CLM) & Clause Risk Redlining (`POST /api/v1/ai/contracts/analyze`, `POST /api/v1/ai/contracts/redline`)**: Legal clause extraction (Liability caps, indemnification, auto-renewal, liquidated damages), 0-100 risk scoring, and automated AI legal redline counter-proposals.
- **📦 Smart 3D Bin Packing & Pallet Load Optimizer (`POST /api/v1/wms/pack-optimizer`)**: 3D knapsack spatial geometry engine computing item coordinates (x, y, z), rotation, layer stability, and container volume utilization for Euro pallets, standard pallets, and shipping containers.
- **👥 Predictive Employee Attrition & AI Talent Retention Radar (`GET /api/v1/hr/attrition-radar`, `POST /api/v1/hr/retention-playbook`)**: Flight risk radar analyzing overtime burnout, unused leaves, promotion stagnation, and compensation benchmarks, with automated 1-click HR retention playbooks.
- **🏦 Multi-Horizon Bank Reconciler with ML Fuzzy Statement Matching (`POST /api/v1/account/reconciliation/auto-match`, `POST /api/v1/account/reconciliation/batch-post`)**: Fuzzy Levenshtein + BI-FAST fee tolerance band matching (+/- Rp 2,500/Rp 6,500) between bank statement lines and open invoices, with 1-click batch auto-post reconciliation.
- **🌐 Universal EDI (Electronic Data Interchange) Gateway (`POST /api/v1/edi/ingest`, `POST /api/v1/edi/generate`, `GET /api/v1/edi/supported`)**: ANSI X12 & UN/EDIFACT gateway translating EDI 850 (PO intake to Sales Order), EDI 855 (PO Ack), EDI 856 (ASN shipping notice), and EDI 810 (Commercial Invoice).
- **⚡ AI Automated Cross-Docking & Wave Dispatch Director (`POST /api/v1/wms/cross-dock/match`, `POST /api/v1/wms/cross-dock/dispatch`)**: Intercepts incoming warehouse receipts, matches waiting customer backorders by SLA priority, and routes directly to dock staging (`STAGING/CROSSDOCK`), slashing warehouse dwell time by 75%.
- **🏢 Multi-Entity Intercompany Consolidated Financials & Elimination Engine (`GET /api/v1/accounting/consolidation/report`, `POST /api/v1/accounting/consolidation/eliminate`)**: Real-time IFRS 10 / PSAK 65 group consolidation, eliminating intra-group revenue, COGS, and intercompany AR/AP with balanced elimination entries.
- **📉 Autonomous Multi-Tier Bullwhip Effect Guard & Lead-Time Anomaly Detector (`GET /api/v1/ai/supply-chain/bullwhip-index`, `POST /api/v1/ai/supply-chain/lead-time-anomalies`)**: Computes demand variance amplification across supply chain echelons (Tier 0 to Tier 3), predicts component stockouts 14–30 days in advance, and provides dampening buffer recommendations.
- **🔍 Corporate Expense Fraud & Policy Governance AI (`POST /api/v1/expense/audit/evaluate`, `GET /api/v1/expense/audit/flagged`)**: Deep claim audit engine detecting smurfing/split claims below approval ceilings, duplicate receipt hashes, weekend/holiday policy violations, and unauthorized entertainment venues.
- **🔄 AI Reverse Logistics & RMA Warranty Diagnostic Engine (`POST /api/v1/wms/rma/diagnose`, `POST /api/v1/wms/rma/process-resolution`)**: Serial number warranty entitlement validation, economic disposition analysis (Refurbish vs Scrap vs Restock), and 1-click execution of replacement pickings or credit notes.
- **⚙️ AI Production Line Balancing & OEE Bottleneck Optimizer (`POST /api/v1/mrp/line-balancing/analyze`, `POST /api/v1/mrp/line-balancing/rebalance`)**: Dynamic cycle time vs takt time analysis, workstation bottleneck detection, WIP queue monitoring, and AI task reallocation raising line balance efficiency to 95.5% (+30.6% gain).
- **🚚 Multi-Carrier Last-Mile Dispatch & Route Geometry Optimizer (VRPTW) (`POST /api/v1/fleet/route-optimizer`, `POST /api/v1/fleet/dispatch-manifest`)**: Capacitated Vehicle Routing Problem with Time Windows using Haversine geometry, spatial cluster dispatching across internal fleet and 3PL carriers, and dynamic driver manifest generation with tracking URLs.
- **💳 Dynamic Customer Credit Risk & Bad Debt Scorer with DSO Aging Guard (`GET /api/v1/account/credit-risk/customer/:id`, `POST /api/v1/account/credit-risk/evaluate-order`)**: Multi-factor creditworthiness profiling, invoice aging matrix (Current, 1-30, 31-60, 61-90, 90+ days), Days Sales Outstanding (DSO), Altman Z-score default risk probability, and automated order credit hold blocking.
- **🌱 Fixed Asset Depreciation & Carbon Tax Cap-and-Trade Ledger (`GET /api/v1/asset/carbon-depreciation/matrix`, `POST /api/v1/asset/carbon-depreciation/settle-allowance`)**: Dual financial depreciation and Scope 1/2 operational carbon emissions tracking, statutory carbon allowance cap comparison, emission liability tax calculations, and 1-click green certificate settlement.
- **🔮 Universal Headless GraphQL & Webhook Gateway (`POST /api/v1/graphql`, `POST /api/v1/webhooks/subscriptions`)**: Pure-Go zero-dependency GraphQL query engine with recursive nested entity resolution (`salesOrder -> partner -> orderLines -> delivery -> invoice -> financialMetrics`), full schema introspection, and HMAC-SHA256 signature event subscriptions.
- **👥 AI Automated Shift Scheduling & Labor Rostering Optimizer (`POST /api/v1/hr/roster-optimizer/generate`, `POST /api/v1/hr/roster-optimizer/shift-swap`)**: Constraint-satisfaction heuristic scheduler balancing workload demand, skills mix, and Indonesian labor regulations (PP 35/2021 overtime ceilings, mandatory 11-hour rest intervals, and shift swap verification).
- **🔄 Autonomous Multi-Warehouse Stock Equalizer & Inter-Branch Balancer (`POST /api/v1/wms/inventory-balancing/evaluate`, `POST /api/v1/wms/inventory-balancing/execute-transfers`)**: Multi-echelon stock health auditor analyzing Days-of-Supply (DoS), sales velocity, and inter-branch freight transit times to generate consolidated internal transfers (`WH/INT/BAL`), slashing working capital spend.
- **🎯 Predictive Customer Lifetime Value (CLV) & Dynamic Churn Preventer (`GET /api/v1/crm/clv-matrix`, `POST /api/v1/crm/churn-preventer/trigger-campaign`)**: BG/NBD and Gamma-Gamma predictive CLV engine calculating 90-day pipeline value, customer equity, and RFM churn probability, triggering 1-click VIP retention playbooks.
- **🔐 Zero-Trust Cryptographic Secret Sharing & Quorum Vault (`POST /api/v1/security/vault/shard-keys`, `POST /api/v1/security/vault/execute-quorum-action`)**: Shamir's Secret Sharing ($k$-of-$n$ threshold scheme over GF(P)) guarding critical administrative actions (fiscal year GL unsealing, master payroll export) with cryptographic multi-executive quorum consensus.
- **📡 Real-Time RFID / BLE Asset Tracking Radar & Geofence Demurrage Engine (`POST /api/v1/asset/tracking/rfid-ping`, `GET /api/v1/asset/tracking/radar-map`)**: Continuous EPC Gen2 RFID and BLE beacon telemetry tracking asset zones, misplaced asset radar alerts, and automated geofence detention/demurrage fee calculation.
- **⚡ Continuous Financial Virtual Close & Smart Accruals Engine (`POST /api/v1/accounting/continuous-close/run`, `GET /api/v1/accounting/continuous-close/dashboard`)**: 24/7 continuous subledger reconciliation across AR, AP, Inventory, and Intercompany ledgers, auto-posting Goods Received Not Invoiced (GRNI) adjustments and unbilled service accruals with automated next-period reversal dates, slashing close cycles from 10 days to < 1 day.
- **🤝 Vendor-Managed Inventory (VMI) Consignment & Replenishment Gateway (`POST /api/v1/vmi/replenishment-forecast`, `POST /api/v1/vmi/consignment-consumption/log`)**: Supplier-facing telemetric consignment visibility with dynamic min-max safety stock bands, authorized self-dispatch restocking, and automated Pay-as-Consumed 11% VAT billing upon shopfloor consumption.
- **🌐 Automated FTA Rules-of-Origin & Preferential Tariff Optimizer (`POST /api/v1/customs/fta-origin/evaluate`, `GET /api/v1/customs/fta-origin/certificate-matrix`)**: Multi-level BOM analyzer auditing Regional Value Content (RVC $\ge 40\%$) and Change in Tariff Classification (CTC) across ATIGA Form D (ASEAN), Form E (China), and RCEP, capturing billions in preferential duty exemptions.
- **🔀 Pure-Go CQRS Database Read-Replica Router & Distributed Latency Guard (`GET /api/v1/system/cqrs/health`, `POST /api/v1/system/cqrs/route-query`)**: Command Query Responsibility Segregation (CQRS) router isolating mission-critical ACID writes on Master DB while offloading heavy analytic scans and reports across multi-region read replicas with replication lag monitoring.
- **🦺 AI Workplace Safety, Incident Radar & ISO 45001 Compliance (`POST /api/v1/ehs/incident/report`, `GET /api/v1/ehs/safety-radar`)**: Lost Time Injury Frequency Rate (LTIFR) and Severity Rate (SR) tracking, continuous zero-harm streak monitoring, automated 5-Why root-cause reasoning, and corrective action (CAPA) compliance audit trails.
- **💰 Global Treasury Cash Pooling, In-House Banking & Multilateral Netting (`POST /api/v1/treasury/cash-pool/sweep`, `POST /api/v1/treasury/netting/run-cycle`, `GET /api/v1/treasury/in-house-bank/balances`)**: Automated EOD Zero-Balance Account (ZBA) physical sweeps, arm's-length intercompany interest calculation, and multilateral debt netting compressing cross-border FX transfer volume by 80%+.
- **🚢 Automated Landed Cost Allocation & Multi-Modal Freight Accrual Engine (`POST /api/v1/costing/landed-cost/allocate`, `GET /api/v1/costing/landed-cost/audit-layer/:receipt_id`)**: Capitalizes ocean/air freight, customs tariffs, THC, and insurance into inventory valuation layers (`stock.valuation.layer`) by weight, volume, or CIF value with retroactive COGS adjustments.
- **🌿 Supply Chain Scope 3 Carbon Accounting & EU CBAM Passport (`POST /api/v1/esg/scope3/calculate-product-passport`, `GET /api/v1/esg/cbam/export-declaration/:export_order_id`)**: Upstream cradle-to-gate carbon intensity decomposition per ton of product and automated EU Carbon Border Adjustment Mechanism (CBAM) export declaration with origin carbon tax deductions.
- **🚐 Field Service Lightning, Geospatial Dispatching & Van Stock Radar (`POST /api/v1/field-service/dispatch/recommend`, `POST /api/v1/field-service/work-order/complete`, `GET /api/v1/field-service/van-stock/:technician_id`)**: Multi-criteria geospatial dispatching based on Haversine distance, certified skills, and on-board van spare parts availability, with mobile stock relief and digital e-signature completion.
- **🏷️ Algorithmic Dynamic Pricing & Margin Guard Intelligence (`POST /api/v1/pricing/dynamic/evaluate`, `POST /api/v1/pricing/margin-guard/verify-order`)**: Demand elasticity & competitor-aware automated repricing with strict hard-floor gross margin protection and omnichannel sync to POS and eCommerce storefronts.
- **🏭 Smart Factory Floor Andon Visual Control & Shopfloor Anomaly Gateway (`POST /api/v1/mrp/andon/trigger-event`, `GET /api/v1/mrp/andon/board/:line_id`, `POST /api/v1/mrp/andon/resolve-event`)**: Real-time workstation status visualization (Green/Yellow/Red) for shopfloor screens, automated forklift runner replenishment dispatch, and Mean Time To Respond (MTTR) tracking.
- **📜 Letters of Credit (L/C) & Trade Finance Zero-Discrepancy Engine (`POST /api/v1/trade-finance/lc/audit-documents`, `GET /api/v1/trade-finance/lc/collateral-utilization`)**: Comprehensive pre-presentation auditing of export documents against UCP 600 Letter of Credit rules, eliminating discrepancy fees and monitoring trade credit limits.
- **⚡ Supplier Early-Payment Dynamic Discounting & Supply Chain Finance (`POST /api/v1/treasury/dynamic-discounting/offer`, `POST /api/v1/treasury/dynamic-discounting/accept`, `GET /api/v1/treasury/dynamic-discounting/pipeline`)**: Sliding-scale APR early settlement discounts for vendor bills, turning Accounts Payable into a high-yield treasury returns generator while easing supplier working capital crunches.
- **🔬 Predictive Machine Vibration FFT & Remaining Useful Life (RUL) Radar (`POST /api/v1/iot/vibration/analyze-fft`, `GET /api/v1/maintenance/predictive-rul/:equipment_id`)**: Fast Fourier Transform (FFT) harmonic peak decomposition, ISO 10816 vibration severity grading, Weibull Remaining Useful Life (RUL) prediction, and auto-reservation of spare parts.
- **✈️ Corporate Travel & Entertainment (T&E) Policy Engine & Per-Diem Matrix (`POST /api/v1/hr/travel/evaluate-policy`, `POST /api/v1/hr/travel/settle-per-diem`)**: Multi-rule corporate travel policy compliance auditor (advance window, cabin class, hotel ceiling) with city-tier per-diem allowances and automated cash advance reconciliation.
- **🌍 Cross-Border Transfer Pricing & OECD BEPS Action 13 Hub (`POST /api/v1/tax/transfer-pricing/audit-margins`, `GET /api/v1/tax/transfer-pricing/cbcr-matrix`)**: Interquartile benchmark arm's-length testing ($Q1 \le \text{Margin} \le Q3$), TNMM operating margin audits, transfer pricing adjustment calculations, and automated Country-by-Country Reporting (CbCR).
- **🏢 High-Value Asset Lease Accounting Engine (IFRS 16 / PSAK 73) (`POST /api/v1/accounting/leases/schedule`, `POST /api/v1/accounting/leases/remeasure`)**: Present value capitalization at Incremental Borrowing Rate (IBR), dual-amortization liability & straight-line ROU depreciation schedules, and retroactive indexation remeasurement.
- **🚛 Yard Management System (YMS) & Carrier Dock Appointment Scheduler (`POST /api/v1/wms/yard/book-slot`, `POST /api/v1/wms/yard/gate-checkin`, `GET /api/v1/wms/yard/dock-radar`)**: Self-service carrier dock slot appointments, mobile geofenced gate check-in with gate passes, real-time dock bay radar, and dwell-time demurrage alerts.
- **⚙️ Dynamic Rule-Based Product Configurator & CPQ Engine (`POST /api/v1/cpq/configure-product`, `POST /api/v1/cpq/generate-bom`)**: Propositional logic constraint checks (*Requires*, *Excludes*), hard margin floor protection, dynamic multi-level production BOM synthesis, and automated labor routing rollup.
- **🛡️ Corporate Whistleblower & Anonymous Ethics Hotline Vault (ISO 37002) (`POST /api/v1/grc/ethics/report-incident`, `POST /api/v1/grc/ethics/investigate-case`, `GET /api/v1/grc/ethics/dashboard`)**: Zero-knowledge tokenized reporting, cryptographic SHA-256 evidence digests for non-repudiation, automated risk triage scoring, and independent audit committee investigation lifecycle.
- **📦 Supply Chain Multi-Echelon Inventory Optimization (MEIO) & Risk Pooling (`POST /api/v1/scm/meio/optimize-network`, `GET /api/v1/scm/meio/tradeoff-curve`)**: Multi-tier network inventory optimization across Central DCs, Regional Hubs, and Retail Spokes, unlocking 15%–30% working capital through echelon variance pooling.
- **📈 Multi-Currency Hedging & Derivative Accounting (IFRS 9 / PSAK 71) (`POST /api/v1/treasury/hedging/create-contract`, `POST /api/v1/treasury/hedging/revalue-mtm`, `GET /api/v1/treasury/hedging/portfolio`)**: Formal designation of FX forwards/collars, discounted Mark-to-Market fair values, Dollar-Value effectiveness testing (80%–125%), and Cash Flow Hedge Reserve (OCI) vs P&L accounting segregation.
- **💊 Electronic Batch Record (EBR) & Regulated Pharma Hub (FDA 21 CFR Part 11) (`POST /api/v1/mrp/ebr/execute-step`, `POST /api/v1/mrp/ebr/sign-deviation`, `GET /api/v1/mrp/ebr/batch-record/:batch_number`)**: Step-by-step master manufacturing formula execution, In-Process Control (IPC) parameters, dual operator/QA cryptographic digital signatures, deviation containment, and automated Certificate of Analysis (CoA) release.
- **📑 Corporate Withholding Tax Equalization & Cross-Border DGT-1 Treaty Engine (`POST /api/v1/tax/wht/audit-equalization`, `POST /api/v1/tax/wht/validate-dgt1`)**: Cross-reconciliation between General Ledger expense accounts and reported WHT tax bases (DPP), detecting unwithheld tax exposure and validating foreign vendor Form DGT-1 treaty certificates (P3B) for rate reduction.
- **👑 Omnichannel Customer Loyalty Points & IFRS 15 Contract Liability Ledger (`POST /api/v1/crm/loyalty/earn-points`, `POST /api/v1/crm/loyalty/redeem-reward`, `GET /api/v1/crm/loyalty/member/:member_id`)**: Multi-tier rewards accrual, automatic promotion thresholds, and IFRS 15 / PSAK 72 contract liability deferral and release upon redemption.
- **🌱 Internal Carbon Pricing (ICP) & Shadow Carbon Fee Ledger (`POST /api/v1/esg/carbon-pricing/evaluate-capex`, `GET /api/v1/esg/carbon-pricing/internal-tax-pool`)**: Evaluates capital investments with internal shadow carbon pricing ($75/ton CO2e), carbon-adjusted TCO, and corporate Green Innovation Fund grants.
- **🤖 Autonomous Warehouse Mobile Robot (AMR & AGV) Fleet Orchestrator (`POST /api/v1/wms/amr/dispatch-mission`, `POST /api/v1/wms/amr/telemetry-update`, `GET /api/v1/wms/amr/fleet-radar`)**: Real-time collision-free robot mission dispatching, State-of-Charge battery monitoring, and opportunistic fast-charging management.
- **🔒 Smart Milestone Escrow & Condition-Based Payment Engine (`POST /api/v1/procurement/escrow/create-milestone`, `POST /api/v1/procurement/escrow/verify-release`, `GET /api/v1/procurement/escrow/vault-summary`)**: Condition-governed escrow fund locking and automated milestone releases based on IoT proof, WMS receipts, and dual e-signature BASTs.
- **📊 Dynamic Multi-Element Revenue Recognition (IFRS 15 / PSAK 72) (`POST /api/v1/accounting/revrec/allocate-ssp`, `POST /api/v1/accounting/revrec/modify-contract`)**: Relative Standalone Selling Price (SSP) allocation across bundled obligations and prospective/cumulative contract modifications.
- **🎯 9-Box Talent Matrix & Strategic Succession Bench Strength Analyzer (`POST /api/v1/hr/succession/evaluate-talent`, `GET /api/v1/hr/succession/bench-strength/:role_id`)**: Maps performance vs potential on the 9-box grid, automates executive development action plans, and tracks Key Person Risk across mission-critical roles.
- **🌓 Dark / Light Enterprise Theme Modes**: Seamless visual theme switcher (`☀️ Light` / `🌙 Dark`) with state persisted across browser sessions.

---

## 🚀 Enterprise Modules Overview (50 Core Modules)

### Group 1: Core Operations & Supply Chain
1. **Base & Multi-Company (`internal/modules/base`)**: Multi-company hierarchy, currency handling, customer/vendor partners (`res.partner`), user management (`res.users`), and document auto-sequencing (`ir.sequence`).
2. **Product Catalog (`internal/modules/product`)**: Product templates and variants (`product.product`), categories, units of measure (UoM), standard cost, and sales pricing.
3. **Double-Entry Inventory & Stock (`internal/modules/stock`)**: Multi-warehouse locations (`stock.warehouse`), stock quants, internal moves, delivery orders (`WH/OUT`), receipts (`WH/IN`), internal transfers (`WH/INT`), and automated reordering rules.
4. **Sales Management (`internal/modules/sale`)**: Quotation-to-order pipeline (`sale.order`), line items, automated delivery order reservation, and invoicing hooks.
5. **Purchase Management (`internal/modules/purchase`)**: RFQ generation (`purchase.order`), purchase approval, and automated inbound receipt (`WH/IN`) creation.
6. **Point of Sale (`internal/modules/pos`)**: Fast retail checkout terminal (`pos.session`, `pos.order`), cash register open/close reconciliation, and 80mm thermal receipt printing.
7. **Barcode & Lot Engine (`internal/modules/barcode`)**: Pure-Go SVG Code 128 barcode generator, lot and serial tracking (`stock.production.lot`), expiration dates, and context-aware mobile scanner.
8. **Purchase Requisitions & Tenders (`internal/modules/purchase_requisition`)**: Internal purchase requests, competitive multi-vendor tender auctions (*Call for Bids*), and long-term blanket purchase agreements.
9. **Landed Costs (`internal/modules/landed_cost`)**: Capitalization of ocean freight, customs tariffs, terminal fees, and cargo insurance into inventory valuation across 5 allocation methods (`by_value`, `by_quantity`, `by_weight`, `by_volume`, `equal`).
10. **3D Warehouse WMS (`internal/modules/wms`)**: 3D bin coordinate hierarchy (Aisle/Rack/Shelf/Bin), FEFO/FIFO putaway rules, pick routing, and consolidated wave picking (`wms.wave.picking`).
11. **Lot Traceability & Recall (`internal/modules/traceability`)**: Complete upstream and downstream supply chain lot genealogy, ingredient tracking, and rapid product recall management.

### Group 2: Financial Management & Accounting
12. **Accounting & General Ledger (`internal/modules/account`)**: Double-entry bookkeeping with strict $\sum \text{Debit} == \sum \text{Credit}$ validation, Chart of Accounts, multi-journal entry, customer invoices, vendor bills, and financial statements (P&L, Balance Sheet, Trial Balance).
13. **Multi-Currency & Forex Engine (`internal/modules/currency`)**: Multi-currency exchange rate tables, daily spot rates (JISDOR/BI), realized exchange gain/loss, and month-end unrealized FX revaluation (PSAK 10 / IAS 21).
14. **Fixed Assets & Depreciation (`internal/modules/asset`)**: Asset registration (`account.asset`), straight-line depreciation schedules, net book value calculations, and automated GL posting.
15. **Enterprise Budgeting & Cost Centers (`internal/modules/budget`)**: Corporate operational budget plans (RKAB), analytic accounts (`account.analytic.account`), theoretical vs practical spend variance, and AI burn-rate early warning.
16. **Bank Reconciliation (`internal/modules/bank_reconciliation`)**: Multi-bank statement CSV parser (BCA, Mandiri, BRI, generic), algorithmic auto-matching engine, bank fee write-offs, and 1-click ledger reconciliation.
17. **Corporate Treasury (`internal/modules/treasury`)**: Multi-bank account liquidity tracking, 13-week rolling cash flow forecasts, and automated inter-account cash sweep transfers.
18. **Intercompany Holding (`internal/modules/intercompany`)**: Multi-company organizational structures, automated sales/purchase order mirroring, and intercompany elimination consolidation.
19. **Recurring Subscriptions & SaaS (`internal/modules/subscription`)**: Contracted subscription billing cycles (Monthly/Annual), automated recurring invoicing, MRR/ARR metrics, and AI churn prediction.

### Group 3: Tax & Customs Compliance
20. **Tax e-Faktur Engine (`internal/modules/tax_efaktur`)**: Tax invoice serial quota management (NSFP), 11% VAT e-Faktur generation, QR verification codes, and periodic VAT return exports.
21. **Tax e-Bupot Unifikasi (`internal/modules/tax_ebupot`)**: Unified withholding tax calculation (PPh 21/23/26/4(2)), official tax withholding slips, and periodic tax return filing summaries.
22. **Customs Bonded Zone CEISA 4.0 (`internal/modules/customs_bonded`)**: Bonded zone declarations (BC 2.3, BC 4.0, BC 2.5, BC 2.7, BC 4.1, BC 2.6.1/2, BC 3.0), SPPB clearance, subcontracting, and 4 real-time IT Inventory mutation ledgers.
23. **Digital Signatures & Stamp Duty (`internal/modules/esignature`)**: Cryptographic SHA-256 document signing workflows, signer verification audit trails, and digital stamp duty integration.

### Group 4: Manufacturing & Industrial Operations
24. **Manufacturing MRP I (`internal/modules/mrp`)**: Multi-level Bill of Materials (BoM), manufacturing orders (`mrp.production`), component consumption, and finished goods inventory receipt.
25. **Advanced MRP II (`internal/modules/mrp_advanced`)**: Industrial work centers (`mrp.workcenter`), machine hourly capacities, sequential routing operations, shop floor execution, and Overall Equipment Effectiveness (OEE) metrics.
26. **Maintenance & Equipment (`internal/modules/maintenance`)**: Plant machinery catalog (`maintenance.equipment`), automated preventive maintenance scheduling, and corrective breakdown work orders.
27. **Quality Control & CAPA (`internal/modules/quality`)**: Quality Control Points (QCP) across inbound receiving and manufacturing, quantitative tolerance checks, and CAPA non-conformance alerts.

### Group 5: Human Resources & Payroll
28. **Human Resources (`internal/modules/hr`)**: Department hierarchy, job positions, employee master directory, and employment contracts.
29. **GPS Attendance & Shift Telemetry (`internal/modules/attendance`)**: GPS geofenced check-in/out, shift definitions, Cloudflare edge anti-fraud telemetry, and automated payroll synchronization.
30. **Leave & Time Off (`internal/modules/leave`)**: Annual leave quota allocations, multi-tiered leave approval workflows, and automated unpaid absence deductions.
31. **Enterprise Payroll (`internal/modules/payroll`)**: Multi-tier salary structures, dynamic compensation rules, statutory social security (BPJS), batch payroll runs, and balanced GL posting.
32. **Employee Expense Claims (`internal/modules/expense`)**: Travel and operational expense submissions, multi-item expense sheets, manager approval, and automatic GL journal reimbursement.
33. **AI Talent Recruitment (`internal/modules/recruitment`)**: Job openings, applicant pipeline, automated AI resume/CV screening and scoring, and 1-click hire to active employee directory.

### Group 6: CRM, Services & Collaboration
34. **CRM Pipeline (`internal/modules/crm`)**: Sales pipeline stages, lead qualification, win-rate analytics, expected revenue forecasting, and 1-click sales order conversion.
35. **Helpdesk & Support Ticketing (`internal/modules/helpdesk`)**: Customer support ticket intake, tiered priority queues, support team routing, SLA resolution tracking, and AI triage.
36. **Fleet Management (`internal/modules/fleet`)**: Fleet vehicle registry (`fleet.vehicle`), driver assignments, odometer tracking, and scheduled maintenance records.
37. **Chatter & Collaboration (`internal/modules/chatter`)**: Document-level audit trail logging (`mail.message`), internal team discussions, and status transition notes.
38. **Self-Service Vendor Portal (`internal/modules/vendor_portal`)**: Dedicated portal for suppliers: tender bid submissions, advance shipping notices (ASN), and invoice status tracking.
39. **Self-Service Customer Portal (`internal/modules/customer_portal`)**: Dedicated portal for B2B wholesale buyers: online ordering, order status tracking, return merchandise authorization (RMA), and payment confirmations.

### Group 7: Low-Code, BPMN & Extensibility
40. **Low-Code Studio (`internal/modules/lowcode_studio`)**: Dynamic model and field creation (EAV), core model extensions, and conversational AI Schema Copilot.
41. **BPMN 2.0 Workflow Engine (`internal/modules/workflow_engine`)**: Visual BPMN 2.0 state machine engine, conditional decision gateways, user task approval inboxes, and SLA timers.
42. **Universal Integration Hub (`internal/modules/integration_hub`)**: M2M API key management, Open RPC gateway, pre-built marketplace/payment connectors, and scheduled batch ETL pipelines.
43. **Inbound Webhooks (`internal/modules/inbound_webhook`)**: Omnichannel webhook receiver for payment gateways (Midtrans, Xendit, Stripe), e-commerce marketplaces, and logistics providers.
44. **Outbound Webhooks (`internal/modules/webhook`)**: Real-time event notifications delivered to external endpoints with HMAC-SHA256 signatures and automated retry policies.
45. **Immutable Audit Trail (`internal/modules/audit_trail`)**: Forensic audit log capturing deep before-and-after JSON diffs, user attribution, and suspicious activity detection.
46. **Universal Attachments & R2 OCR (`internal/modules/attachment`)**: File attachments backed by Cloudflare R2 / local disk with integrated AI Vision receipt OCR.
47. **Universal Report Engine (`internal/modules/reports`)**: Corporate A4 print layouts (SO, PO, Invoices, Delivery Slips, Payslips, Financial Statements) and 80mm POS receipts with cryptographic SHA-256 audit hashes.
48. **Physical Stock Opname (`internal/modules/stock_inventory`)**: Stocktaking cycles, automated book quant snapshots, physical count entry, AI shrinkage audit, and GL variance adjustments.
49. **Corporate Approval Matrix (`internal/modules/approval`)**: Multi-tier corporate authorization matrix based on financial thresholds, departments, and executive roles.
50. **CSV & Excel Export Engine (`internal/core/export`)**: Universal data export engine delivering UTF-8 BOM CSV files compatible with Microsoft Excel across all ERP entities.

---

## 📂 Project Directory Structure

```text
c:\erp\
├── cmd/
│   └── server/
│       └── main.go                 # Primary server entrypoint, AutoMigrate, Seeding, Routing & Dashboard Stats
├── internal/
│   ├── config/
│   │   └── config.go               # Environment variables configuration (.env, DB, JWT, Cloudflare AI & R2)
│   ├── core/
│   │   ├── ai/                     # Cloudflare Workers AI client, ERP Copilot & Vision OCR
│   │   ├── commandcenter/          # Omni-search, notifications, cycle simulator & executive KPI summary
│   │   ├── database/               # GORM database connection initialization (Pure-Go SQLite & PostgreSQL)
│   │   ├── export/                 # Universal CSV export utility across all ERP data tables
│   │   ├── middleware/             # JWT AuthMiddleware & Role-Based Access Control (RBAC)
│   │   ├── models/                 # Standard BaseModel (ID, CreatedAt, UpdatedAt, DeletedAt, CompanyID)
│   │   ├── registry/               # Dynamic module registry (Enterprise-style module loader)
│   │   └── storage/                # Cloudflare R2 & Local File System object storage abstraction
│   └── modules/                    # 50 Enterprise Business Modules (Each: models, repository, service, handler)
│       ├── base/                   # 1. Base, Multi-Company, Partners & Users
│       ├── product/                # 2. Product Categories & Product Master
│       ├── stock/                  # 3. Warehouses, Locations, Double-Entry Stock Moves & Quants
│       ├── sale/                   # 4. Quotations & Sales Orders
│       ├── purchase/               # 5. RFQs & Purchase Orders
│       ├── account/                # 6. Chart of Accounts, Invoices, Payments & GL
│       ├── mrp/                    # 7. Bill of Materials & Manufacturing Orders
│       ├── pos/                    # 8. Point of Sale Sessions & Instant Retail Checkout
│       ├── hr/                     # 9. Departments, Employees & Organization
│       ├── crm/                    # 10. Lead Pipeline & Opportunity Conversion
│       ├── project/                # 11. Projects, Tasks, Kanban Stages & Timesheets
│       ├── chatter/                # 12. Universal Document Audit Trail & Notes
│       ├── asset/                  # 13. Fixed Assets & Depreciation Boards
│       ├── helpdesk/               # 14. Support Tickets, SLA Stages & Escalation
│       ├── maintenance/            # 15. Equipment, Preventive Scheduling & Work Orders
│       ├── fleet/                  # 16. Fleet Vehicles, Odometer Logs & Servicing
│       ├── leave/                  # 17. Leave Allocations, Requests & Deductions
│       ├── expense/                # 18. Expense Claims, Multi-Item Sheets & GL Posting
│       ├── quality/                # 19. Quality Control Points, Checks & CAPA Alerts
│       ├── approval/               # 20. Multi-Tier Approval Matrix
│       ├── attachment/             # 21. Document Attachments & AI Vision Receipt OCR
│       ├── attendance/             # 22. GPS Geofenced Attendance & Shift Telemetry
│       ├── recruitment/            # 23. Job Openings, AI CV Screening & 1-Click Hire
│       ├── stock_inventory/        # 24. Physical Stock Opname & Discrepancy Adjustments
│       ├── payroll/                # 25. Salary Structures, BPJS & Batch Payslips
│       ├── budget/                 # 26. Budgets (RKAB), Cost Centers & Variance Auditing
│       ├── barcode/                # 27. Pure-Go SVG Code 128 Generator & Mobile Scanner
│       ├── purchase_requisition/   # 28. Purchase Requisitions, Tenders & Blanket Orders
│       ├── currency/               # 29. Multi-Currency, Spot Rates & FX Revaluation
│       ├── tax_efaktur/            # 30. VAT e-Faktur Engine, NSFP Quotas & SPT Export
│       ├── bank_reconciliation/    # 31. Bank Statement Import, Auto-Match & Reconciliation
│       ├── landed_cost/            # 32. Landed Cost Capitalization to Inventory Valuation
│       ├── traceability/           # 33. Lot Genealogy & Product Recall Management
│       ├── subscription/           # 34. Recurring SaaS Subscriptions & Invoicing
│       ├── audit_trail/            # 35. Cryptographic Before/After JSON Audit Log
│       ├── webhook/                # 36. Outbound Webhook Event Dispatcher
│       ├── inbound_webhook/        # 37. Omnichannel Inbound Webhook Receiver
│       ├── intercompany/           # 38. Multi-Company Holding & Intercompany Consolidation
│       ├── treasury/               # 39. Multi-Bank Liquidity & 13-Week Cash Forecast
│       ├── mrp_advanced/           # 40. Advanced MRP II: Work Centers, Routings & OEE
│       ├── tax_ebupot/             # 41. Unified Tax Withholding & Return Filing
│       ├── esignature/             # 42. Cryptographic Digital Signatures & Stamp Duty
│       ├── wms/                    # 43. 3D Warehouse Bin Locations & Wave Picking
│       ├── vendor_portal/          # 44. Self-Service Vendor Portal
│       ├── customer_portal/        # 45. Self-Service B2B Customer Portal
│       ├── customs_bonded/         # 46. Customs Bonded Zone IT Inventory Compliance
│       ├── lowcode_studio/         # 47. Low-Code Studio: Dynamic Models (EAV) & Fields
│       ├── reports/                # 48. Universal Report Engine: Corporate A4 & POS Receipts
│       ├── workflow_engine/        # 49. Visual BPMN 2.0 State Machine Engine
│       └── integration_hub/        # 50. Universal Integration Hub: Open RPC & ETL
├── tests/
│   └── erp_test.go                 # 54-Section Comprehensive End-to-End Test Suite
├── web/
│   ├── index.html                  # Single Page Application Dashboard (50 Module Tabs)
│   ├── app.js                      # Application State, REST Client & Interactive Canvases
│   └── style.css                   # Enterprise Dark Glassmorphism CSS Design System
├── Dockerfile                      # Production multi-stage Docker build
├── docker-compose.yml              # Service orchestration for GoERP + PostgreSQL 15
└── .env.example                    # Environment variable configuration template
```

---

## 🛠️ Quick Start & Execution

### 1. Run the Pre-compiled Binary
```powershell
./server.exe
```

### 2. Run from Source Code (Go Run)
```powershell
go run ./cmd/server
```

The system will initialize and listen on:
- **Web Dashboard**: [http://localhost:8080/](http://localhost:8080/)
- **API Health**: [http://localhost:8080/health](http://localhost:8080/health)
- **Default Admin Login**: `admin@erp.local` / `admin123`

### 3. Run with Docker Compose
```bash
docker compose up -d --build
```
This orchestrates:
- **GoERP Server**: Port 8080
- **PostgreSQL 15**: Port 5432

### 4. Run the Automated Test Suite
```powershell
go test -v ./tests/...
```

---

## 📡 Core REST API Endpoint Reference

All endpoints are hosted under the `/api/v1` namespace:

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/auth/login` | Authenticate user and issue JWT token |
| `GET` | `/api/v1/auth/me` | Fetch active user profile and company context |
| `GET` | `/api/v1/dashboard/stats` | Executive KPI metrics across all 50 enterprise modules |
| `GET` | `/api/v1/search?q=:query` | 🔍 Universal omni-search across modules, records, and SKU codes |
| `GET` | `/api/v1/dashboard/notifications` | 🔔 Actionable notification and approval center |
| `POST` | `/api/v1/dashboard/simulate-cycle` | 🚀 1-Click end-to-end business cycle simulator (PO $\rightarrow$ WH/IN $\rightarrow$ SO $\rightarrow$ WH/OUT $\rightarrow$ INV) |
| `GET` | `/api/v1/dashboard/export-summary` | 📥 Export executive financial and operational KPI summary to CSV |
| `GET` | `/api/v1/partners` | List business partners (filter: `customer`, `supplier`) |
| `POST` | `/api/v1/partners` | Create a new partner record |
| `GET` | `/api/v1/products` | Retrieve product catalog with real-time stock balances |
| `POST` | `/api/v1/products` | Create a new product entry |
| `GET` | `/api/v1/sales/orders` | List sales quotations and confirmed orders |
| `POST` | `/api/v1/sales/orders` | Create a new sales quotation |
| `POST` | `/api/v1/sales/orders/:id/confirm` | Confirm Sales Order $\rightarrow$ auto-generates Delivery Order (`WH/OUT`) |
| `GET` | `/api/v1/purchases/orders` | List RFQs and confirmed purchase orders |
| `POST` | `/api/v1/purchases/orders` | Create a new purchase RFQ |
| `POST` | `/api/v1/purchases/orders/:id/confirm` | Confirm Purchase Order $\rightarrow$ auto-generates Goods Receipt (`WH/IN`) |
| `GET` | `/api/v1/stock/pickings` | List warehouse delivery, receipt, and transfer documents |
| `POST` | `/api/v1/stock/pickings/:id/validate` | Validate double-entry stock movement and update quants |
| `GET` | `/api/v1/stock/warehouses` | List enterprise multi-warehouses and branches |
| `POST` | `/api/v1/stock/transfers` | Execute atomic inter-warehouse inventory transfers (`WH/INT`) |
| `GET` | `/api/v1/stock/quants` | Real-time on-hand stock quantities per product, warehouse, and location |
| `GET` | `/api/v1/accounting/moves` | List customer invoices, vendor bills, and general journal entries |
| `POST` | `/api/v1/accounting/moves/:id/post` | Validate $\sum \text{Debit} == \sum \text{Credit}$ and post entry to General Ledger |
| `POST` | `/api/v1/accounting/moves/:id/pay` | Register invoice payment and reconcile balances |
| `GET` | `/api/v1/accounting/reports/profit-loss` | Generate official Profit & Loss financial statement |
| `GET` | `/api/v1/accounting/reports/balance-sheet` | Generate official Balance Sheet financial statement |
| `GET` | `/api/v1/accounting/reports/trial-balance` | Generate official Trial Balance financial statement |
| `GET` | `/api/v1/mrp/boms` | List assembly Bill of Materials (BoM) |
| `POST` | `/api/v1/mrp/productions/:id/produce` | Execute manufacturing production $\rightarrow$ consumes raw materials, produces finished goods |
| `POST` | `/api/v1/pos/orders` | Process instant retail point-of-sale checkout $\rightarrow$ deducts store stock |
| `GET` | `/api/v1/hr/employees` | Employee master directory, departments, and compensation |
| `POST` | `/api/v1/payroll/runs/:id/generate` | ⚡ 1-Click mass generation of monthly employee payslips |
| `POST` | `/api/v1/payroll/payslips/:id/confirm` | Approve payslip and post balanced salary journal entry to GL |
| `GET` | `/api/v1/crm/leads` | List CRM sales leads and opportunities |
| `POST` | `/api/v1/crm/leads/:id/convert` | 🛒 Convert won CRM opportunity into official Sales Quotation |
| `GET` | `/api/v1/chatter/:model/:res_id` | Document-level audit trail, notes, and activity history |
| `POST` | `/api/v1/chatter/:model/:res_id` | Post a collaboration message or internal note to a record |
| `POST` | `/api/v1/barcode/scan` | Context-aware barcode scanner (products, lots, locations, pickings) |
| `GET` | `/api/v1/barcode/label/:type/:id` | 🖨️ Pure-Go SVG Code 128 barcode label generator (58mm/80mm) |
| `POST` | `/api/v1/purchase-requisitions/:id/select-bid` | Select winning vendor tender bid $\rightarrow$ confirms PO and auto-cancels competing RFQs |
| `POST` | `/api/v1/currencies/revaluations` | Calculate month-end unrealized foreign exchange revaluation (PSAK 10) |
| `POST` | `/api/v1/tax-efaktur/invoices/out` | Issue official VAT 11% e-Faktur from confirmed invoice (auto-decrements NSFP) |
| `GET` | `/api/v1/tax-efaktur/export-csv` | Download official tax CSV format for electronic tax return filing |
| `POST` | `/api/v1/bank-reconcile/statements/import` | Import multi-bank CSV statements (BCA, Mandiri, BRI, generic) |
| `POST` | `/api/v1/bank-reconcile/statements/:id/reconcile-all` | 1-Click algorithmic reconciliation of high-confidence bank transactions |
| `POST` | `/api/v1/landed-costs/:id/validate` | Allocate landed costs and post inventory valuation adjustment to GL |
| `GET` | `/api/v1/customs-bonded/reports/:type` | Generate official Customs Bonded Zone IT Inventory mutation ledgers |
| `GET` | `/api/v1/lowcode/models` | List dynamic EAV models created at runtime |
| `POST` | `/api/v1/lowcode/models` | Create a new dynamic business model and fields without SQL DDL migration |
| `POST` | `/api/v1/workflows/start` | Instantiate a new BPMN 2.0 workflow state machine instance |
| `POST` | `/api/v1/workflows/instances/:id/action` | Execute user task approval/rejection transition within a workflow |
| `POST` | `/api/v1/integration/rpc/:method` | Universal Open RPC gateway for M2M API integration |
| `GET` | `/api/v1/reports/:template/:id` | Render PDF-ready corporate A4 print layout with SHA-256 audit hash |
| `POST` | `/api/v1/ai/receipt-ocr` | ✨ AI Vision Receipt OCR data extraction |
| `POST` | `/api/v1/ai/ask` | 🤖 Natural Language Executive Copilot querying live database context |
| `GET` | `/api/v1/ai/providers` | 🌐 List supported AI Agent & LLM providers (Gemini 2.5, OpenAI GPT-4o, Claude 3.7 MCP, DeepSeek R1, Cloudflare) |
| `GET` | `/api/v1/ai/tools` | 🔧 Export MCP (Model Context Protocol), OpenAI, Gemini, and Anthropic tool-calling schemas |
| `POST` | `/api/v1/ai/agent/execute` | 🧠 Execute Autonomous AI Agent ReAct loop with live database tool calling & BigQuery streaming |
| `GET` | `/api/v1/ai/gcp/status` | ☁️ Inspect Google Cloud Platform (BigQuery, Pub/Sub, GCS, Vertex AI, Cloud Run) integration status |
| `POST` | `/api/v1/ai/gcp/sync` | ☁️ Trigger real-time synchronization to Google Cloud BigQuery, Pub/Sub, or Cloud Storage (GCS) |
| `POST` | `/mcp` | 🔌 Native Model Context Protocol (MCP) JSON-RPC 2.0 Server |
| `GET` | `/.well-known/agent.json` | 🤖 Google Cloud Agent-to-Agent (A2A) Discovery Card |
| `GET` | `/api/docs` | 📑 OpenAPI 3.1 Interactive API Explorer & Protocol Console |
| `POST` | `/api/v1/ai/doc-intelligence` | 🧾 AI Document Intelligence (Auto-OCR & Automated 3-Way PO/GRN Matching) |
| `GET` | `/api/v1/ai/fraud-radar` | 🛡️ Continuous Forensic AI Audit & Fraud Detection Radar |
| `GET` | `/api/v1/ai/cashflow-forecast` | 📈 Predictive 30/60/90-Day Cashflow Forecasting & CFO Runway |
| `GET` | `/api/v1/ai/esg-report` | 🌱 ESG & Carbon Footprint Ledger (Scope 1, 2, 3 GHG Accounting) |
| `POST` | `/api/v1/ai/chatops/webhook` | 💬 Omnichannel ChatOps Gateway (WhatsApp, Telegram & Slack Webhook) |
| `GET` | `/api/v1/ai/demand-sensing` | 📊 AI Demand Sensing, Dynamic Safety Stock & EOQ Replenishment |
| `POST` | `/api/v1/ai/replenish/auto-po` | 🚀 1-Click Automated Draft Purchase Order Replenishment |
| `GET` | `/api/v1/events/stream` | ⚡ Real-Time Server-Sent Events (SSE) Stream |
| `POST` | `/api/v1/events/presence` | 👥 Document Viewing & Editing Live Presence |
| `POST` | `/api/v1/saas/tenants/register` | 🏢 Self-Service SaaS Multi-Tenant Provisioning |
| `GET` | `/api/v1/saas/tenants` | 🏢 List Registered SaaS Tenants & Isolation Metadata |
| `POST` | `/api/v1/pos/sync-offline-batch` | 📴 Reconcile & Synchronize Offline PWA POS Transactions |
| `POST` | `/api/v1/ai/voice/command` | 🎙️ Hands-Free Voice Ops Assistant & Audio Synthesis |

---

## ☁️ Google Cloud Platform (GCP), Popular AI Agents & Cloudflare Integration

GoERP features a unified **Multi-Provider AI Agent Router** and native **Google Cloud Platform (GCP)** & **Cloudflare** cloud adapters:

### 1. Popular AI Agent APIs & Autonomous Tool-Calling Gateway (`internal/core/ai`)
GoERP supports automatic priority routing (`AI_PROVIDER=auto`) or explicit selection across the industry's leading AI Agent APIs, complete with **Model Context Protocol (MCP)** and function-calling access to live ERP data (`query_inventory_stock`, `get_financial_summary`, `search_erp_records`, `create_sales_quotation`, `create_purchase_rfq`, `sync_gcp_bigquery`):
- **Google Gemini 2.5 & Vertex AI Agent Builder**: `GEMINI_API_KEY`, `GEMINI_MODEL=gemini-2.5-flash` (or `gemini-2.5-pro`).
- **OpenAI GPT-4o & Agents SDK**: `OPENAI_API_KEY`, `OPENAI_MODEL=gpt-4o` (or `o3-mini`).
- **Anthropic Claude 3.7 Sonnet & MCP**: `ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL=claude-3-7-sonnet-latest`.
- **DeepSeek R1 / V3 Reasoning Agent**: `DEEPSEEK_API_KEY`, `DEEPSEEK_MODEL=deepseek-chat` (or `deepseek-reasoner`).
- **Cloudflare Workers AI**: `CLOUDFLARE_ACCOUNT_ID`, `CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_AI_MODEL=@cf/meta/llama-3.1-8b-instruct`.
- **GoERP Built-In Deterministic Agent**: Zero-dependency offline fallback ensuring 100% deterministic reliability in air-gapped or local environments.

### 2. Google Cloud Platform (GCP) Ecosystem
- **Google Cloud BigQuery Data Warehouse (`gcp_bigquery`)**: Stream real-time General Ledger entries, Sales Orders, Purchase Orders, and Inventory Quants to BigQuery (`GCP_BIGQUERY_DATASET=goerp_analytics`) for Looker Studio and enterprise BI.
- **Google Cloud Pub/Sub Event Bus (`gcp_pubsub`)**: Publish asynchronous domain events (`erp.invoice.posted`, `erp.order.confirmed`) to `projects/$GCP_PROJECT_ID/topics/$GCP_PUBSUB_TOPIC`.
- **Google Cloud Storage Archive Vault (`internal/core/storage` & `gcp_gcs_archive`)**: S3-interoperable object storage (`GCS_BUCKET`, `GCS_ACCESS_KEY_ID`, `GCS_SECRET_ACCESS_KEY`) for attachments, e-Faktur CSVs, and immutable audit snapshots.
- **Google Cloud Build & Cloud Run (`cloudbuild.yaml`)**: Declarative CI/CD pipeline building multi-stage Docker images to Google Artifact Registry and deploying serverless containers to Google Cloud Run with Cloud SQL for PostgreSQL:
  ```bash
  gcloud builds submit --config cloudbuild.yaml
  ```

### 3. Cloudflare Edge & R2 Object Storage
```bash
export CLOUDFLARE_R2_BUCKET="your-r2-bucket"
export CLOUDFLARE_R2_ACCESS_KEY_ID="your-r2-access-key-id"
export CLOUDFLARE_R2_SECRET_ACCESS_KEY="your-r2-secret-access-key"
export CLOUDFLARE_R2_ENDPOINT="https://<account-id>.r2.cloudflarestorage.com"
```
*When cloud storage credentials are omitted, attachments seamlessly fall back to local disk storage (`./uploads/attachments`).*

### 4. Cloudflare Tunnel (Zero-Trust Edge Access)
Expose an on-premise or private cloud GoERP server to the public Internet without opening any inbound firewall ports:
```bash
cloudflared tunnel --config cloudflare/cloudflared-config.yml run go-erp-tunnel
```

---

## 🌐 Community & Open Source Governance

GoERP is distributed freely under the permissive **[Apache License 2.0](LICENSE)**. We believe enterprise software should be transparent, free from artificial per-user licensing fees, and readily extensible by developers worldwide.

### Ways to Get Involved:
- 💬 **Community Discussions & Q&A**: Join our [GitHub Discussions](https://github.com/antonset1104/erp/discussions) to share implementation experiences, request architectural feedback, and connect with other contributors.
- 🐛 **Issue Reporting**: Encountered a bug or transactional anomaly? Submit a report via [GitHub Issues](https://github.com/antonset1104/erp/issues/new/choose) using our structured issue forms.
- 🚀 **Submitting Code**: Review our [Contributing Guidelines](CONTRIBUTING.md) to understand branch conventions, test expectations, and the Pull Request review workflow.
- 🛡️ **Vulnerability Disclosures**: Submit security vulnerabilities privately by following our [Security Policy](SECURITY.md).

---

## ⚙️ Repository Configuration Guide for Maintainers

For repository maintainers and forks, verify the following configuration settings:
1. **Branch Protection on `main`**: Enable *Require a pull request before merging*, *Require status checks to pass* (`Test (Go 1.22.x on ubuntu-latest)` and `windows-latest`), and *Require linear history*.
2. **Issue Forms & PR Templates**: The repository includes declarative YAML templates in `.github/ISSUE_TEMPLATE/` (Bug Report, Feature Request, and Discussions config) and `.github/pull_request_template.md`.
3. **Automated CI/CD**: Matrix build, vet, and automated testing are defined in `.github/workflows/ci.yml`.
4. **Dependabot Automated Updates**: Security and version updates for Go modules and GitHub Actions are monitored in `.github/dependabot.yml`.
5. **Multi-Platform Automated Releases**: Automated multi-architecture binaries (`linux/amd64`, `linux/arm64`, `windows/amd64`) with SHA-256 checksums are built upon tagging releases `v*` (`.github/workflows/release.yml`).
