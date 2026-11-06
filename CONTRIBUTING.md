# Contributing to GoERP

Thank you for your interest in contributing to **GoERP**. We welcome contributions from the community, including bug reports, documentation updates, performance improvements, and feature pull requests.

---

## Code of Conduct

All contributors and maintainers are expected to adhere to our [Code of Conduct](CODE_OF_CONDUCT.md) to ensure an inclusive, respectful environment for everyone.

---

## Development Environment Setup

### Prerequisites
- **Go 1.22 or higher**
- **Git**
- **C Compiler** (GCC or Clang on Linux/macOS, MinGW or TDM-GCC on Windows) for SQLite CGO driver support.
- *(Optional)* **Docker & Docker Compose** for running against local PostgreSQL instances.

### Local Setup
1. **Fork** the repository on GitHub: [https://github.com/antonset1104/erp](https://github.com/antonset1104/erp)
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/<your-username>/erp.git
   cd erp
   ```
3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/antonset1104/erp.git
   ```
4. **Download Go dependencies**:
   ```bash
   go mod download
   go mod verify
   ```
5. **Create local environment file**:
   ```bash
   cp .env.example .env
   ```
6. **Start the local development server**:
   ```bash
   go run ./cmd/server
   ```
   The application dashboard will be accessible at `http://localhost:8080`.

---

## Running Tests

Before submitting a pull request, ensure that all unit and integration tests pass:

```bash
# Run all tests
go test -v ./tests/...

# Run tests with race detection
go test -v -race ./tests/...

# Run static analysis and formatting checks
go vet ./...
go fmt ./...
```

---

## Architecture Guidelines for New Modules

GoERP follows a modular monolith architecture. When introducing new business features:

```text
internal/modules/<module_name>/
├── models.go       # GORM entity definitions and table bindings
├── repository.go   # Database queries and data access logic
├── service.go      # Business rules and transaction management
├── handler.go      # HTTP request handling and JSON response encoding
└── module.go       # Module registration and route mounting
```

### Key Engineering Invariants
1. **Model Inheritance**: Master and transactional entities must embed `models.ModelBase` (`ID`, `CreatedAt`, `UpdatedAt`, `DeletedAt`, `CompanyID`).
2. **Double-Entry Balance**: All financial transactions touching ledger accounts must enforce zero-sum balance: $\sum \text{Debit} == \sum \text{Credit}$.
3. **Inventory Conservation**: Physical stock adjustments must always be modeled between two locations (`SourceLocationID` $\rightarrow$ `DestinationLocationID`).
4. **Collision-Resistant Numbering**: Document numbers (e.g., `SO%05d`, `INV%05d`) must use collision-resistant sequence logic with database transaction safety.

---

## Git Workflow & Pull Requests

1. Create a feature branch from `main`:
   ```bash
   git checkout -b feat/your-feature-name
   ```
2. Commit your changes following Conventional Commits guidelines:
   - `feat(scope): add new capability`
   - `fix(scope): resolve issue description`
   - `docs(scope): update documentation`
   - `refactor(scope): internal code restructuring`
3. Sign off on your commits (enforces DCO):
   ```bash
   git commit -s -m "feat(wms): implement wave picking algorithm"
   ```
4. Rebase against the latest upstream `main` before opening your PR:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```
5. Push to your fork and submit a Pull Request describing:
   - The business problem or bug being addressed.
   - The technical approach taken.
   - Evidence of test verification.
