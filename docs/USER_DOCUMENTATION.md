# User Operational Guide — GoERP

This guide provides step-by-step operational instructions for day-to-day users, department managers, and system administrators working with GoERP.

---

## 1. Getting Started

### 1.1 Accessing the Application
1. Open a modern web browser (Google Chrome, Mozilla Firefox, Microsoft Edge, or Apple Safari) and navigate to your GoERP host address (for local instances: `http://localhost:8080`).
2. Enter your credentials on the login screen. Default local installations provide pre-configured demonstration accounts:
   - **Administrator**: Username `admin` / Password `admin123`
3. Click **Sign In** to access the main workspace.

### 1.2 Interface Navigation
The interface is divided into three primary functional zones:
- **Left Sidebar**: Contains the complete categorized navigation tree for all enterprise business areas. Click any module name to navigate directly without a full page reload.
- **Top Navigation Bar**:
  - Displays the active screen title and contextual operational tools.
  - **Global Search (`Ctrl+K`)**: Opens the universal search palette for finding records, documents, or navigation targets.
  - **Module Filter (`/`)**: Focuses the module filter input in the sidebar for quick keyword filtering.
  - **Notifications**: Displays pending approval requests, workflow tasks, and operational alerts.
  - **Theme Toggle**: Switches between Light and Dark visual themes.
  - **Barcode Scanner**: Activates the camera or hardware barcode reader for scanning product codes or document numbers.
- **Main Workspace**: Displays key metrics, searchable data tables, filter toolbars, document creation buttons, and detail views.

---

## 2. Sales & Customer Invoicing (Order-to-Cash)

### 2.1 Creating a Sales Quotation
1. Navigate to **Sales Orders** in the sidebar.
2. Click **+ Create Sales Order**.
3. Select an existing **Customer** from the dropdown or type to filter.
4. Set the quotation date and payment terms (e.g., Immediate Payment, Net 30, Net 60).
5. Add line items in the **Order Lines** section:
   - Select the **Product**.
   - Enter the requested **Quantity** and verify the **Unit Price**.
   - Tax rates (e.g., standard 11% VAT) are calculated automatically.
6. Click **Save Quotation**. The record is saved in `DRAFT` status.

### 2.2 Confirming an Order and Fulfilling Delivery
1. Open the draft quotation and review line item totals.
2. Click **Confirm Order**. The document status transitions to `SALE`.
3. The system automatically reserves stock and generates a corresponding Delivery Order (`WH/OUT/xxxxx`) in the Inventory module.
4. When warehouse personnel pick, pack, and ship the items:
   - Open **Inventory** -> **Delivery Orders**.
   - Locate the picking document (`WH/OUT/xxxxx`).
   - Click **Validate** to register the physical stock movement. Stock on hand is reduced immediately.

### 2.3 Issuing Customer Invoices
1. From the confirmed sales order (or via **Invoicing** -> **Customer Invoices**), click **Create Invoice**.
2. Select whether to invoice the full delivered quantity or specific line items.
3. Review line items, payment terms, and applicable tax codes.
4. Click **Post to General Ledger**. The invoice moves to `POSTED` status, generating double-entry journal lines:
   - Debit: Accounts Receivable (Asset)
   - Credit: Sales Revenue (Income)
   - Credit: Output VAT Payable (Liability)
5. To print or download a formal PDF invoice with an SHA-256 seal, click the **Print** icon on the document view.

### 2.4 Registering Customer Payments
1. Open the posted customer invoice.
2. Click **Register Payment**.
3. Select the payment method (Bank Transfer, Cash, or Credit Card), enter the received amount, and select the destination bank account.
4. Click **Confirm Payment**. The invoice status transitions to `PAID`, and the open balance updates to zero.

---

## 3. Procurement & Inbound Receiving (Procure-to-Pay)

### 3.1 Creating a Request for Quotation (RFQ)
1. Navigate to **Purchase Orders** in the sidebar.
2. Click **+ Create RFQ**.
3. Select the target **Vendor** and set expected delivery dates.
4. Add components or products in the line items table with required quantities.
5. Click **Save RFQ**. You can print or export the request to send to your supplier.

### 3.2 Confirming a Purchase Order
1. Once vendor pricing is agreed upon, open the RFQ and update unit prices if needed.
2. If the total order exceeds authorization limits, the order routes through the approval matrix automatically.
3. Click **Confirm Order**. The document moves to `PURCHASE` status, and an incoming warehouse receipt (`WH/IN/xxxxx`) is created.

### 3.3 Receiving Goods into Warehouse
1. Navigate to **Inventory** -> **Receipts**.
2. Locate the incoming receipt (`WH/IN/xxxxx`).
3. Click **Check Availability** to verify dock staging space.
4. Review received quantities against the purchase order. Enter serial or lot numbers if tracking is enabled on the product.
5. Click **Validate**. Inventory valuation layers and stock quantities update automatically.

### 3.4 Processing Vendor Bills
1. Navigate to **Invoicing** -> **Vendor Bills**.
2. Click **+ Create Vendor Bill** and link it to the confirmed Purchase Order.
3. Verify line items against the warehouse Goods Receipt (3-way matching).
4. Enter the vendor's invoice number and billing date.
5. Click **Post**. The system records:
   - Debit: Inventory Received Not Invoiced / Expense
   - Debit: Input VAT Receivable
   - Credit: Accounts Payable

---

## 4. Inventory & Warehouse Management

### 4.1 Checking Real-Time Stock
1. Navigate to **Inventory** -> **Stock On-Hand (Quants)**.
2. Filter by product name, SKU, warehouse, or storage location.
3. Columns display:
   - **On Hand**: Physical inventory present in warehouse locations.
   - **Reserved**: Stock allocated to pending delivery orders or manufacturing orders.
   - **Available**: On hand minus reserved quantities available for immediate sale.

### 4.2 Internal Stock Transfers
1. Navigate to **Inventory** -> **Internal Transfers**.
2. Click **+ Create Transfer**.
3. Select the **Source Location** (e.g., `WH/Stock` or a specific bin) and **Destination Location** (e.g., a branch warehouse or transit location).
4. Add products and quantities to transfer.
5. Click **Validate**. Both source and destination balances update atomically.

### 4.3 Physical Inventory & Stocktaking
1. Navigate to **Stock Opname**.
2. Click **+ Start Stocktake Cycle**.
3. Select the target warehouse or storage zone. The system takes a snapshot of current theoretical book quantities.
4. Print physical count sheets or enter counted quantities directly via mobile tablet or barcode scanner.
5. Click **Calculate Discrepancies**. The system highlights variances between book stock and physical count.
6. Click **Post Inventory Adjustments**. Balanced inventory adjustment journals post to the General Ledger.

---

## 5. Manufacturing & Work Orders (Shop Floor)

### 5.1 Setting Up a Bill of Materials (BoM)
1. Navigate to **Manufacturing** -> **Bills of Materials**.
2. Click **+ Create BoM**.
3. Select the parent finished product.
4. In the components table, specify required raw materials, quantities, and expected scrap percentages.
5. Optionally assign production operations to specific **Work Centers** with estimated standard run times.
6. Save the BoM.

### 5.2 Executing a Manufacturing Order
1. Navigate to **Manufacturing** -> **Manufacturing Orders**.
2. Click **+ Create Production Order**.
3. Select the finished product, specify the planned quantity, and choose the BoM.
4. Click **Check Material Availability**. The system confirms whether raw components exist in inventory.
5. Click **Produce**. The system:
   - Consumes component materials from raw stock.
   - Credits component inventory accounts.
   - Debits finished goods inventory.
   - Adds completed units to finished stock.

---

## 6. Financial Accounting & Reporting

### 6.1 Reviewing Financial Statements
Navigate to **Accounting** -> **Financial Reports**:
- **Balance Sheet**: Displays assets, liabilities, and equity at any selected date.
- **Profit & Loss**: Displays operating revenue, cost of goods sold (COGS), gross profit, operating expenses, and net income over a custom date range.
- **Trial Balance**: Confirms that total debits equal total credits across all ledger accounts.
- **General Ledger**: Filter transactions by account code to view full transaction histories with audit links.

### 6.2 Manual Journal Entries
1. Navigate to **Accounting** -> **Journal Entries**.
2. Click **+ Create Journal Entry**.
3. Select the target journal (e.g., Miscellaneous, Bank, General Operations).
4. Enter line items, specifying Account, Debit, and Credit amounts.
5. The entry can only be saved when total debits match total credits ($\sum \text{Debit} = \sum \text{Credit}$).
6. Click **Post**.

---

## 7. Human Resources & Payroll

### 7.1 Employee Directory
1. Navigate to **HR** -> **Employees**.
2. Click **+ Add Employee**.
3. Enter personal details, department, job position, contract start date, and base salary.
4. Assign bank account details for direct deposit processing.

### 7.2 Time & Attendance
- **Clock In / Out**: Employees access the **Attendance** screen and click **Clock In** or **Clock Out**. If geofencing is enabled, coordinates must fall within company-defined boundaries.
- **Leave Requests**: Navigate to **Leave** -> **Request Leave**. Select leave type (Annual, Sick, Maternity), choose date range, and submit for managerial approval.

### 7.3 Running Monthly Payroll
1. Navigate to **Enterprise Payroll**.
2. Click **+ Batch Payroll Run**.
3. Select the pay period (Month and Year) and target department or all employees.
4. Click **Compute Payslips**. The system calculates:
   - Base salary and position allowances.
   - Attendance deductions for unpaid absences.
   - Statutory health and employment insurance withholdings.
   - Income tax withholdings.
5. Review the summary registers.
6. Click **Confirm & Post to GL**. Salary expense and tax liability entries post to the General Ledger automatically.

---

## 8. Fiscal & Tax Compliance

### 8.1 Electronic Tax Invoices (e-Faktur)
1. Navigate to **Tax & e-Faktur**.
2. Manage official tax invoice serial numbers (**NSFP**) under the **NSFP Quota** tab.
3. Outbound customer invoices automatically assign the next sequential NSFP number.
4. Export official CSV tax returns for periodic filing with tax authorities.

### 8.2 Customs Bonded Zones (CEISA)
1. Navigate to **Customs & Bonded Zones**.
2. Register customs declarations (BC 2.3 for inbound materials, BC 2.7 for inter-bonded transfers, BC 4.0 for domestic supply, BC 3.0 for export).
3. Record clearance permits (SPPB) to authorize goods movement.
4. The system automatically maintains the four mandatory regulatory mutation ledgers:
   - Raw Materials
   - Finished Goods
   - Capital Machinery & Equipment
   - Scrap & Rejected Output

---

## 9. Platform Administration & Utilities

### 9.1 Low-Code Studio
Create custom business models and data tables without editing source code:
1. Open **Low-Code Studio** from the sidebar.
2. Click **Create Custom Model**, specify a display name (e.g., *Safety Inspection Log*), and technical identifier (`x_safety_log`).
3. Click **Fields** to add columns: Text, Number, Date, Selection, Boolean, or JSON.
4. Click **Browse Data** to immediately create, view, and edit records in the custom table.

### 9.2 BPMN 2.0 Workflow Engine
Design multi-tier approval rules visually:
1. Open **Workflow Engine**.
2. Select an existing definition or create a new workflow.
3. Configure conditional gateways (e.g., `IF amount >= 100000000`) and assign approval roles.
4. Monitor active running instances and review pending approval queues from your inbox.

### 9.3 Global Command Palette (`Ctrl+K`)
Press **`Ctrl+K`** anywhere in the application to:
- Jump directly to any module or menu.
- Search for specific document numbers (`SO00001`, `PO00002`, `INV/2026/00001`).
- Search for customers, suppliers, or product SKUs.
- Run quick administrative tasks.

---

## 10. Frequently Asked Questions

**How do I export tables to Excel or CSV?**
Click the **Export CSV** button located at the top-right of almost all data tables. The generated file uses UTF-8 encoding with a byte order mark (BOM) for native compatibility with Microsoft Excel, Apple Numbers, and Google Sheets.

**How do I save a document as a PDF?**
Click the **Print** icon on any document (Sales Order, Purchase Order, Invoice, or Payslip). In your browser's print preview window, select **Save as PDF** as the destination printer.

**Can GoERP run offline?**
Yes. The Point of Sale (POS) and warehouse scanning screens operate offline using local browser storage. Transactions are queued locally and synchronized automatically once network connectivity is restored.

**How is data access controlled between departments?**
GoERP implements role-based access control (RBAC). Departmental roles (e.g., Warehouse Operator, Sales Representative, Accountant, HR Manager) restrict visibility and actions so staff only access functions relevant to their responsibilities.
