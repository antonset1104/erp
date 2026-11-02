## 📌 Pull Request Overview

Please provide a concise summary of the changes made, the problem addressed, and the affected modules.

- **Related Issue:** Closes #...
- **Type of Change:**
  - [ ] 🐛 Bug fix
  - [ ] ✨ New feature or enterprise module
  - [ ] ⚡ Performance improvement
  - [ ] 📝 Documentation update
  - [ ] 🎨 Code refactoring
  - [ ] 🧪 Testing & coverage

---

## 🏢 Affected Modules

- [ ] Core Engine / Database / Routing
- [ ] Sales & Quotations / CRM
- [ ] Purchases & Supplier RFQ
- [ ] Inventory / Stock / WMS
- [ ] Invoicing / Accounting / General Ledger
- [ ] MRP II / Manufacturing / BOM
- [ ] HRIS / Payroll / Attendance
- [ ] Tax & Fiscal Localization
- [ ] Customs & Bonded Zones
- [ ] Low-Code Studio / Workflow Engine / Integration Hub
- [ ] Web UI (Frontend SPA)

---

## 🧪 Verification & Evidence

Describe the testing performed to verify this change:

```bash
# Local test execution
go test -v ./tests/...
go test -v -race ./tests/...
```

- [ ] All unit and integration tests pass 100% (`PASS`).
- [ ] No regression introduced to intersecting modules.
- [ ] *(If UI-related)* Verified in live browser with 0 JavaScript console errors.

---

## ✅ Contributor Checklist

- [ ] My code follows the [CONTRIBUTING.md](CONTRIBUTING.md) guidelines.
- [ ] Code is formatted with `go fmt ./...`.
- [ ] Unit or integration tests covering the changes have been added/updated.
- [ ] Commit messages follow the Conventional Commits specification.
- [ ] All user-facing web templates preserve required AdSense and Analytics tags.
- [ ] No credentials, tokens, or confidential secrets are committed.
