# Contributing to GoERP

Thank you for your interest in contributing to **GoERP**! We welcome all forms of contributions from the global open-source community—including bug reports, feature proposals, architectural discussions, documentation improvements, and code contributions via Pull Requests.

---

## 📜 Code of Conduct

As contributors and maintainers of this project, we are committed to fostering an open, welcoming, and harassment-free community for everyone. Please read and adhere to our [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) before participating.

---

## 🛠️ Development Environment Setup

### Prerequisites
1. **Go 1.22 or higher** (Go 1.22+ is recommended for native HTTP routing enhancements and modern concurrency features).
2. **Git**.
3. **C Compiler** (GCC / MinGW on Windows, GCC/Clang on Linux/macOS) for the CGO-based `go-sqlite3` driver.
4. *(Optional)* **Docker & Docker Compose** for running and testing against PostgreSQL.

### Local Installation Steps
1. **Fork** this repository to your GitHub account: [https://github.com/antonset1104/erp](https://github.com/antonset1104/erp)
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/<your-username>/erp.git
   cd erp
   ```
3. **Add the upstream remote**:
   ```bash
   git remote add upstream https://github.com/antonset1104/erp.git
   ```
4. **Download and verify Go dependencies**:
   ```bash
   go mod download
   go mod verify
   ```
5. **Set up the local environment configuration**:
   ```bash
   cp .env.example .env
   ```
6. **Start the application server**:
   ```bash
   go run ./cmd/server
   ```
   The application dashboard will be live at `http://localhost:8080`.

---

## 🧪 Running the Test Suite

Before submitting any code changes, ensure that all unit and integration tests pass 100%:

```bash
# Run the complete test suite
go test -v ./tests/...

# Run tests with the race condition detector enabled
go test -v -race ./tests/...

# Verify code formatting and static analysis
go vet ./...
go fmt ./...
```

---

## 🏗️ System Architecture & Adding New Modules

GoERP is engineered using a high-performance **Modular Monolith** architecture:

```
erp/
├── cmd/server/               # HTTP server entrypoint & main route registration
├── internal/
│   ├── config/               # Environment configuration & flags
│   ├── core/
│   │   ├── database/         # Database abstraction (SQLite/PostgreSQL) & GORM migrations
│   │   ├── commandcenter/    # Command Palette (Ctrl+K), Omni-Search, & Business Simulator
│   │   └── models/           # ModelBase (ID, CreatedAt, UpdatedAt, Active Promoted)
│   └── modules/              # 50 Isolated Enterprise Modules (base, stock, sale, account, etc.)
│       └── <module_name>/
│           ├── model.go      # Data models & GORM hooks
│           ├── repository.go # Data queries & database access
│           ├── service.go    # Business logic & transactional invariants
│           └── handler.go    # REST API request handlers
├── web/                      # Modern SPA Frontend (HTML5, Vanilla ES6, CSS Variables)
└── tests/                    # End-to-End & Integration Test Suites
```

### Standards for New Modules:
1. **Inherit from `models.ModelBase`**: Every master and transactional model must embed `models.ModelBase` (`ID`, `CreatedAt`, `UpdatedAt`, `Active`).
2. **Financial & Inventory Invariants**:
   - Inventory movements must use atomic double-entry bookkeeping (*Source Location* $\rightarrow$ *Destination Location*).
   - General Ledger journal entries must strictly enforce balanced books: $\sum \text{Debit} == \sum \text{Credit}$.
3. **Collision-Resistant Number Sequences**:
   - Always implement sequence uniqueness verification loops (e.g., `PO%05d`, `SO%05d`, `INV%05d`) to guarantee safety under high-concurrency race conditions.
4. **Global Web Rule Compliance**:
   - Every user-facing HTML template must preserve the mandatory Google AdSense (`ca-pub-6180580801533680`) and Google Analytics (`G-79X7K7H4MJ`) scripts.

---

## 🌿 Git Workflow & Branching Conventions

1. Always create your feature branch from the latest `main` branch:
   ```bash
   git checkout main
   git pull upstream main
   git checkout -b feature/your-feature-name
   # or for bug fixes:
   git checkout -b fix/your-fix-description
   ```
2. Adhere to the **Conventional Commits** standard:
   - `feat:` Introduces a new feature or enterprise module
   - `fix:` Patches a bug or resolves an edge case
   - `docs:` Documentation improvements or additions
   - `perf:` Code improvements that optimize performance/concurrency
   - `refactor:` Code restructuring without changing external behavior
   - `test:` Adding or updating unit/integration test coverage
   - `chore:` Dependency upgrades, build scripts, or CI maintenance

Example commit messages:
```bash
git commit -m "feat(stock): add multi-bin wave picking validation algorithm"
git commit -m "fix(account): prevent negative debit entries during bank reconciliation"
```

---

## 📤 Submitting a Pull Request (PR)

1. Ensure your code is properly formatted with `go fmt ./...`.
2. Confirm that `go test -v ./tests/...` passes without any failures.
3. Push your feature branch to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```
4. Navigate to the main repository at [https://github.com/antonset1104/erp](https://github.com/antonset1104/erp) and click **Compare & pull request**.
5. Fill out the Pull Request template thoroughly:
   - Summarize the background and the problem solved.
   - List the affected modules.
   - Attach verification evidence (test outputs or UI screenshots).
6. The maintainers will review your PR and provide feedback.

---

## 🐛 Reporting Bugs & Proposing Features

- **Bug Reports**: Before creating a new issue, search the *Issues* tab to verify it hasn't been reported previously. If it's a new issue, use the **Bug Report** form and provide reproduction steps, Go version, and error logs.
- **Feature Requests**: Discuss your ideas for new business workflows or modules using the **Feature Request** form or in the **Discussions** forum.

Thank you for helping build the future of high-performance, open-source ERP systems in Golang!
