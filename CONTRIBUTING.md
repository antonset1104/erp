# Internal Engineering Guide — GoERP Enterprise

> **PROPRIETARY & CONFIDENTIAL**  
> This repository contains proprietary intellectual property of GoERP Systems. Access is restricted to authorized employees, internal engineers, and verified development contractors. Public forks, external pull requests, and unauthorized copying or distribution are strictly prohibited.

---

## 1. Development Principles & Code Standards

Internal GoERP engineers are expected to uphold the highest engineering rigor:
- **Clean Architecture**: Follow the standard layered structure (*Handler → Service → Repository → GORM Model*).
- **Double-Entry Invariants**: Maintain mathematical balance across financial general ledgers ($\sum \text{Debit} = \sum \text{Credit}$) and inventory quant movements.
- **Zero-CGO Compatibility**: Ensure code compiles cleanly with `CGO_ENABLED=0` for pure-Go SQLite deployments.
- **No Third-Party Copyleft Code**: Never import or copy code licensed under GPL, AGPL, or similar copyleft licenses into any GoERP module.

---

## 2. Local Development Environment

### Prerequisites
- **Go 1.22 or higher**
- **Git**
- *(Optional)* **Docker & Docker Compose** for local PostgreSQL and test clusters.

### Local Setup Steps
1. Clone the internal repository:
   ```bash
   git clone https://github.com/antonset1104/erp.git
   cd erp
   ```
2. Download and verify Go modules:
   ```bash
   go mod download
   go mod verify
   ```
3. Initialize the local configuration file:
   ```bash
   cp .env.example .env
   ```
4. Start the development server (runs auto-migrations and seeds default master data):
   ```bash
   go run ./cmd/server
   ```
   The local application interface will be available at `http://localhost:8080`.

---

## 3. Automated Testing & Quality Assurance

Before opening an internal pull request, verify that all test suites pass with zero warnings:

```bash
# 1. Run static analysis and formatting checks
go fmt ./...
go vet ./...

# 2. Run the comprehensive integration test suite
go test -v ./tests/...

# 3. Run race condition detection
go test -v -race ./tests/...
```

---

## 4. Architecture Guidelines for Modules

All business capabilities reside in `internal/modules/<domain>/` with strict separation of concerns:

```text
internal/modules/<module_name>/
├── models.go       # GORM entity definitions with BaseModel embedding
├── repository.go   # Data access layer and isolated SQL queries
├── service.go      # Business logic and transaction coordinator
├── handler.go      # HTTP request binding and JSON response serialization
└── module.go       # Route definitions and module registry registration
```

### Engineering Requirements
1. **Base Model Inheritance**: Master and transactional entities must embed `models.BaseModel` (`ID`, `CreatedAt`, `UpdatedAt`, `DeletedAt`, `CompanyID`).
2. **Transaction Atomicity**: Multi-entity mutations must execute inside an active database transaction (`tx.Transaction(func(tx *gorm.DB) error { ... })`).
3. **Multi-Company Data Isolation**: Queries filtering transactional data must include `company_id` to guarantee tenant isolation.
4. **Collision-Resistant Document Sequences**: Use the atomic sequence generator for document codes (e.g., `SO/2026/00001`, `INV/2026/00001`).

---

## 5. Internal Git Workflow & Branching Strategy

GoERP uses a trunk-based branching model anchored on `main`:

```text
main (production-ready)
└── feature/ERP-1024-wms-wave-picking
└── fix/ERP-1088-reconciliation-tolerance
└── release/v1.83.0
```

### Branch Naming Conventions
- `feature/ERP-<ticket-id>-<short-description>`
- `fix/ERP-<ticket-id>-<short-description>`
- `refactor/ERP-<ticket-id>-<short-description>`
- `release/v<major>.<minor>.<patch>`

### Commit Message Standard (Conventional Commits)
All commit messages must reference the internal Jira/ticket identifier and follow the Conventional Commits format:
- `feat(stock): implement wave picking bin allocation (ERP-1024)`
- `fix(account): adjust foreign currency exchange rounding tolerance (ERP-1088)`
- `docs(tech): update customs bonded zone API specification (ERP-1102)`
- `refactor(core): optimize database connection pooling (ERP-1115)`

---

## 6. Internal Pull Request & Review Procedures

1. **Internal Pull Request Submission**:
   - Open a PR targeting `main` from your feature/fix branch.
   - Use the internal [Pull Request Template](.github/pull_request_template.md).
   - Link the relevant internal issue or Jira ticket number.
2. **Mandatory Review Approvals**:
   - Every PR requires at least **two approvals** from Senior Engineers or Subsystem Leads.
   - Core changes (`internal/core/`) require approval from a Principal Systems Architect.
3. **Automated CI Validation**:
   - The Enterprise CI pipeline must pass 100% on all matrices (Ubuntu and Windows runners).
4. **Merge Policy**:
   - Authorized leads merge PRs using **Squash and Merge** to maintain a clean linear Git history.
