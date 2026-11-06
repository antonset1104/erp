# GoERP: High-Performance Enterprise Resource Planning in Go

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![CI Status](https://img.shields.io/badge/CI-Passing-success?style=flat&logo=githubactions)](https://github.com/antonset1104/erp/actions)
[![Release](https://img.shields.io/badge/Release-v1.82.0-emerald)](https://github.com/antonset1104/erp/releases)

GoERP is a lightweight, high-performance Enterprise Resource Planning (ERP) platform built from the ground up in Go. It compiles into a single self-contained executable binary with an embedded responsive web frontend, requiring zero external runtime dependencies like Python or Node.js.

Designed as an enterprise-grade modular monolith, GoERP enforces strict double-entry accounting balances, inventory mass conservation across all warehouse movements, and multi-tenant security while consuming under 30 MB of idle memory.

---

## Key Highlights

- **Single Binary Deployment**: The entire application—backend API, business modules, database migration engine, and single-page web UI—compiles to a single static binary.
- **Minimal Resource Footprint**: Cold starts in under 50 milliseconds with ~25 MB idle memory consumption, easily running on edge devices, branch servers, or cloud containers.
- **Double-Entry Everything**: Both general ledger accounts ($\sum \text{Debit} = \sum \text{Credit}$) and physical inventory tracking (double-entry stock locations) strictly conserve transactional integrity.
- **Dual Storage Engine**: Uses pure-Go SQLite for zero-configuration local and branch deployments, or PostgreSQL for high-throughput multi-node enterprise environments.
- **API & Protocol First**: Full RESTful API with interactive OpenAPI 3.1 documentation (`/api/docs`), native Model Context Protocol (MCP) server support (`/mcp`), and configurable webhooks.
- **Extensible Architecture**: Includes a runtime Low-Code Studio for custom models and fields without code changes, alongside a visual BPMN 2.0 workflow approval engine.

---

## Documentation Index

Detailed architectural, operational, and development guides are maintained in the `docs/` directory:

- [Technical Documentation](docs/TECHNICAL_DOCUMENTATION.md): System architecture, database schema, security model, and API specifications.
- [Functional Documentation](docs/FUNCTIONAL_DOCUMENTATION.md): End-to-end business workflows (Order-to-Cash, Procure-to-Pay, MRP II, HR/Payroll, and Tax).
- [User Operational Guide](docs/USER_DOCUMENTATION.md): Step-by-step handbook for daily operations across administrative and operational roles.
- [Open Source Guide](docs/OPEN_SOURCE_GUIDE.md): Governance structure, RFC process, and repository standards.
- [Contributing Guidelines](CONTRIBUTING.md): Environment setup, coding conventions, and pull request workflows.
- [Security Policy](SECURITY.md): Vulnerability reporting procedures and response timelines.

---

## Core Functional Domains

GoERP organizes enterprise business capabilities into modular domains:

### 1. Financial Management & Accounting
- **General Ledger**: Chart of Accounts, multi-currency support (IAS 21 / PSAK 10), journal vouchers, and automated posting validation.
- **Financial Statements**: Real-time generation of Balance Sheet, Profit & Loss (P&L), and Trial Balance reports.
- **Accounts Receivable & Payable**: Customer invoicing, vendor bill processing, aging analysis, and payment allocation.
- **Bank Reconciliation**: Statement parser with multi-attribute rule matching, fee write-offs, and batch reconciliation.
- **Fixed Assets**: Asset registry, automatic straight-line or declining depreciation schedules, and book value tracking.
- **Corporate Treasury**: Multi-bank account liquidity monitoring, rolling cash flow projections, and zero-balance account sweeping.

### 2. Supply Chain & Logistics
- **Double-Entry Inventory**: Internal, vendor, and customer locations; serial and lot tracking; expiration date governance.
- **Warehouse Management (WMS)**: 3D bin coordinates (Aisle/Rack/Shelf/Bin), wave picking, and FIFO/FEFO putaway strategies.
- **Landed Cost Capitalization**: Allocation of freight, customs tariffs, and handling charges into inventory valuation layers.
- **Procurement & Tenders**: Requisitions, Requests for Quotation (RFQs), multi-vendor bidding, and blanket purchase agreements.
- **Physical Inventory**: Periodic stocktaking cycles, system-to-physical count variance tracking, and automatic adjustment journals.

### 3. Manufacturing & Maintenance
- **Bill of Materials (BoM)**: Multi-level component hierarchies, scrap factor modeling, and variant formulas.
- **Shop Floor Control (MRP II)**: Work centers, machine capacities, sequential routings, and Overall Equipment Effectiveness (OEE) metrics.
- **Plant Maintenance**: Equipment registry, scheduled preventive maintenance intervals, breakdown work orders, and spare parts allocation.
- **Quality Assurance**: Inbound and in-process quality control points, quantitative tolerance gates, and corrective action (CAPA) tracking.

### 4. Sales, CRM & Retail
- **Sales Pipeline**: Opportunity qualification, quotation generation, automated delivery order reservations, and customer invoicing.
- **Point of Sale (POS)**: Retail terminal with cashier shift reconciliation, cash drawer controls, split payments, and 80mm thermal receipt printing.
- **B2B Customer & Vendor Portals**: Self-service interfaces for supplier tender bids, purchase order acknowledgments, and customer invoice tracking.
- **Subscriptions & Billing**: Contracted recurring billing schedules, prorated charges, and MRR/ARR analytics.

### 5. Human Resources & Payroll
- **Personnel Directory**: Organizational hierarchy, job titles, department assignments, and employment contracts.
- **Time & Attendance**: Shift scheduling, attendance logging with optional GPS geofencing, overtime calculation, and leave balances.
- **Payroll Processing**: Configurable salary structures, statutory deductions (BPJS, income tax withholdings), batch payslip runs, and automatic General Ledger posting.
- **Expense Reimbursement**: Multi-item expense report submissions, managerial approvals, and disbursement accounting.

### 6. Regulatory & Fiscal Compliance
- **Tax Localization**: Value Added Tax (VAT 11%), electronic tax invoicing (e-Faktur) with official serial numbers (NSFP), and withholding tax slips (e-Bupot).
- **Customs & Bonded Zones**: Regulatory IT inventory reporting (CEISA 4.0 declarations BC 2.3, BC 2.7, BC 4.0, BC 4.1) with real-time mutation ledgers for raw materials, machinery, finished goods, and scrap.

### 7. Extensibility & Integration
- **Low-Code Studio**: Dynamic entity creation (EAV), custom fields, and runtime schema extensions without code recompilation.
- **BPMN 2.0 Workflow Engine**: Graphical workflow definitions, conditional decision gateways, approval task inboxes, and SLA monitoring.
- **Integration Hub**: Standardized M2M API keys, Open RPC playground, inbound/outbound webhooks with HMAC signatures, and scheduled batch ETL pipelines.

---

## Quick Start Guide

### Prerequisites
- Go 1.22 or later
- C compiler (GCC or Clang) for SQLite CGO driver (optional if using external PostgreSQL)
- Git

### 1. Clone the Repository
```bash
git clone https://github.com/antonset1104/erp.git
cd erp
```

### 2. Configure Environment
```bash
cp .env.example .env
```
Default settings use SQLite in the local directory, so no database installation is required for development.

### 3. Run the Application
```bash
go run ./cmd/server
```

Open your browser and navigate to:
```
http://localhost:8080
```
The application initializes with demo data for immediate testing. Log in with the default administrator credentials:
- **Username**: `admin`
- **Password**: `admin123`

---

## Configuration Reference

Key variables available in `.env`:

| Variable | Default Value | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | HTTP service listening port |
| `DB_DRIVER` | `sqlite` | Database driver (`sqlite` or `postgres`) |
| `DB_NAME` | `erp.db` | SQLite database file path or PostgreSQL database name |
| `DB_HOST` | `localhost` | PostgreSQL host address (ignored for SQLite) |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | PostgreSQL username |
| `DB_PASSWORD` | `postgres` | PostgreSQL password |
| `JWT_SECRET` | *(auto-generated)* | Minimum 32-character secret key for JWT signing |
| `ENVIRONMENT` | `development` | Deployment environment (`development` or `production`) |

---

## Testing & Quality Assurance

Run the automated test suite:

```bash
# Run all tests
go test -v ./tests/...

# Run tests with race detection
go test -v -race ./tests/...

# Run static analysis
go vet ./...
```

The integration test suite verifies transactional integrity, balancing invariants, and multi-tenant isolation across all functional areas.

---

## Directory Structure

```text
erp/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point, migrations, and HTTP router
├── internal/
│   ├── config/                     # Configuration loader and environment bindings
│   ├── core/                       # Shared core infrastructure
│   │   ├── account/                # Financial and accounting utilities
│   │   ├── ai/                     # Intelligence, document processing, and optimization
│   │   ├── commandcenter/          # Omni-search and executive dashboard aggregation
│   │   ├── database/               # GORM database connection and migrations
│   │   ├── middleware/             # Authentication, RBAC, and request tracing
│   │   ├── openapi/                # OpenAPI 3.1 specification builder
│   │   └── storage/                # Object storage abstractions
│   └── modules/                    # Isolated business domain modules
│       ├── account/                # General Ledger and financial accounting
│       ├── base/                   # Partner, company, and user master data
│       ├── hr/                     # Human resources and employee directory
│       ├── mrp/                    # Manufacturing and production orders
│       ├── purchase/               # Purchase orders and procurement
│       ├── sale/                   # Quotations and sales orders
│       ├── stock/                  # Warehouses, locations, and inventory moves
│       └── ...                     # Additional enterprise domain modules
├── web/                            # Embedded SPA frontend (HTML5, CSS, vanilla JavaScript)
├── docs/                           # Architecture, user, and functional documentation
└── tests/                          # Integration and end-to-end test suites
```

---

## License

GoERP is released under the [Apache License 2.0](LICENSE). You are free to use, modify, and distribute this software for commercial or private purposes in accordance with the license terms.
