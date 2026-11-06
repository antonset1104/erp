# Security Policy

The security and integrity of financial records, business operations, and customer data is a foundational priority in GoERP. We value responsible security research and appreciate disclosures that help protect our users.

---

## Supported Versions

Security updates and vulnerability patches are actively maintained for the following release branches:

| Version | Supported | Maintenance Status |
| :--- | :---: | :--- |
| `1.82.x` | Yes | Current Production Release |
| `1.81.x` | Yes | Active Security Maintenance |
| `< 1.81.0`| No | End-of-Life (Please upgrade to the latest release) |

---

## Reporting a Vulnerability

Please do not file public GitHub issues for security vulnerabilities.

If you discover a vulnerability (such as authentication bypass, privilege escalation, SQL injection, remote code execution, or sensitive data exposure), report it responsibly using one of the following methods:

### Option 1: GitHub Private Vulnerability Reporting (Preferred)
1. Navigate to the repository: [https://github.com/antonset1104/erp](https://github.com/antonset1104/erp)
2. Go to the **Security** tab -> **Advisories** -> **Report a vulnerability**.
3. Submit a detailed description, reproduction steps or proof-of-concept, and expected impact.

### Option 2: Direct Email
Send a confidential report to:
`antonset1104@users.noreply.github.com`

Please include:
- Affected components or module paths.
- Step-by-step reproduction instructions (curl commands, script payloads, or screenshots).
- Estimated severity (Low, Medium, High, or Critical).
- Any proposed mitigations or patch suggestions.

---

## Response Process & SLAs

1. **Initial Acknowledgment**: The maintainers will acknowledge receipt of your report within 48 hours.
2. **Triage & Validation**: Severity and reproducibility assessment will be completed within 3 to 5 business days.
3. **Patch Preparation**: Fixes will be developed and validated against our regression test suites in a private fork.
4. **Public Disclosure**: Security releases will be published in coordination with the reporter within 14 calendar days of patch validation.
5. **Credit**: Reporters will be credited in the release notes and advisory (unless anonymity is requested).

---

## Built-In Security Architecture

GoERP applies defense-in-depth principles across its core services:
- **Authentication**: Stateless, time-limited JSON Web Tokens (JWT) signed using HMAC-SHA256 with mandatory minimum key lengths.
- **Credential Storage**: Passwords hashed using Bcrypt with an adaptive work factor.
- **SQL Injection Prevention**: All database queries are executed via parameterized statements in GORM.
- **Multi-Tenant Isolation**: Database operations enforce company-scoping (`company_id`) at the service and repository layers.
- **Audit Logging**: Sensitive master data mutations and financial document postings generate immutable audit records with user attribution and timestamping.
