# Enterprise Security & Vulnerability Policy — GoERP Enterprise

The security and integrity of financial transactions, customer data, and supply chain operations is paramount. This policy outlines security standards, supported versions, and the enterprise incident reporting process for **GoERP Enterprise**.

---

## 1. Supported Enterprise Releases

Security maintenance updates and emergency patches are actively provided for licensed enterprise installations on the following versions:

| Version Stream | Security Support | Lifecycle Status |
| :--- | :---: | :--- |
| `1.82.x Enterprise` | Active | Current Production Enterprise LTS |
| `1.81.x Enterprise` | Active | Maintenance Enterprise LTS |
| `< 1.81.0` | Inactive | End-of-Life (Upgrade required for security coverage) |

---

## 2. Reporting a Vulnerability (Confidential Disclosure)

Under the terms of the GoERP Enterprise Commercial Agreement and applicable non-disclosure agreements (NDAs), all potential vulnerabilities must be submitted exclusively through private, encrypted channels. **Do not create public tickets or issues.**

### Channel 1: Enterprise Support Portal (Commercial Customers)
Licensed customers should submit a high-priority ticket via the **Enterprise Support Portal** under the category `Security Incident`:
- URL: `https://support.goerp-enterprise.internal/security`

### Channel 2: Enterprise Security Desk (Email)
Send an encrypted report to the dedicated enterprise security desk:
- Email: `security-desk@goerp.internal` (or `antonset1104@users.noreply.github.com` for verified security disclosures).

### Required Information
To accelerate triage and remediation, reports should include:
- Affected subsystem or module (e.g., `account/moves`, `stock/pickings`, `auth/jwt`).
- Reproduction steps, including sample API requests or sanitized logs.
- Assessed business impact and estimated CVSS severity score.
- Any observed indicators of compromise (IoC).

---

## 3. Enterprise Incident Response SLA

GoERP maintains strict Service Level Agreements for enterprise security incidents:

| Severity Level | Initial Acknowledgment | Containment / Mitigation | Hotfix & Patch Delivery |
| :--- | :--- | :--- | :--- |
| **Critical (P1)** | **< 2 hours** | **< 12 hours** | **< 24 hours** |
| **High (P2)** | **< 8 hours** | **< 48 hours** | **< 72 hours** |
| **Medium (P3)** | **< 24 hours** | **< 5 business days** | Next scheduled maintenance release |
| **Low (P4)** | **< 48 hours** | **< 10 business days** | Next minor release cycle |

---

## 4. Confidentiality & Non-Disclosure

All security disclosures, vulnerability analyses, proof-of-concept materials, and remediation patches constitute confidential, proprietary information of GoERP Systems. 

Public disclosure or dissemination of unresolved or resolved vulnerabilities without explicit written authorization from the GoERP Technical Review Board is strictly prohibited.

---

## 5. Defense-in-Depth Security Architecture

GoERP Enterprise incorporates defense-in-depth principles across all architectural layers:
- **Stateless Authentication**: Cryptographically signed JSON Web Tokens (JWT HS256) with strict token expiry and revocation safeguards.
- **Machine-to-Machine Security**: API key authentication with SHA-256 storage hashing, per-key route scoping, and IP address allowlists.
- **Parametric Query Enforcement**: Zero dynamic SQL concatenation; all database operations use GORM parameterized statements to eliminate SQL injection vectors.
- **Multi-Tenant Isolation**: Rigid company-level tenancy constraints (`company_id`) enforced across both service and repository boundaries.
- **Tamper-Evident Audit Trail**: High-consequence financial mutations and master data changes record immutable change logs with user attribution and cryptographic hashing.
