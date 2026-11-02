# Security Policy

The security of financial records, business operations, and user privacy is our highest priority in **GoERP**. We deeply appreciate the security research community and developers who help maintain system integrity through responsible vulnerability disclosure.

---

## 🛡️ Supported Versions

Security updates and vulnerability patches are actively provided for the following releases:

| Version | Supported | Maintenance Status |
| :--- | :---: | :--- |
| `1.41.x` | ✅ Yes | Current Production Release |
| `1.40.x` | ✅ Yes | Critical Security Patch Support |
| `< 1.40.0`| ❌ No | End-of-Life (Please upgrade immediately) |

---

## 🚨 Reporting a Vulnerability

**DO NOT** file public GitHub Issues or discuss vulnerabilities on public channels.

If you discover a potential security vulnerability (such as authentication bypass, privilege escalation, SQL injection, Cross-Site Scripting, or sensitive data exposure), please report it responsibly using one of the following methods:

### Option 1: GitHub Private Vulnerability Reporting (Recommended)
1. Go to the project repository: [https://github.com/antonset1104/erp](https://github.com/antonset1104/erp)
2. Click on the **Security** tab $\rightarrow$ **Advisories** $\rightarrow$ **Report a vulnerability**.
3. Submit technical details, reproduction steps (*Proof-of-Concept*), and potential impact securely.

### Option 2: Direct Confidential Email
Send an encrypted or confidential report directly to:
📧 **antonset1104@users.noreply.github.com**

Please include the following information in your advisory:
- Description of the vulnerability and affected module paths (e.g., `internal/core/database`, `internal/modules/account`).
- Step-by-step reproduction guide (curl command, PoC payload, or screenshots).
- Estimated severity rating (Low, Medium, High, Critical).
- Suggested remediation or patch (if available).

---

## ⏱️ Response Timelines & SLAs

1. **Acknowledgment**: The maintainers will acknowledge receipt of your report within **48 hours**.
2. **Triage & Validation**: Investigation and severity assessment will be completed within **3–5 business days**.
3. **Patch Development & Testing**: Fixes will be prepared in a private fork and thoroughly verified against automated regression suites.
4. **Public Patch Release**: Security releases (e.g., `v1.41.1`) will be published within **14 calendar days** of vulnerability verification.
5. **Credit & Recognition**: You will be publicly credited in the release notes and GitHub Security Advisory (unless you request anonymity).

---

## 🔒 Built-in Security Architecture

GoERP incorporates enterprise-grade defense-in-depth principles:
- **Secure Authentication**: Short-lived JWTs (JSON Web Tokens) with a mandatory $\ge 32$-character secret key.
- **Password Hashing**: Bcrypt with adaptive work factor $\ge 10$.
- **Database Safety**: Parameterized queries via GORM to eliminate SQL Injection risks.
- **XSS Protection**: Strict HTML escaping on frontend rendering and text-based DOM bindings.
- **Multi-Tenant Isolation**: Automated `company_id` tenancy scoping across database operations.
- **ISO 27001 Audit Trail**: Immutable transaction change logging (CDC & user attribution).
