# User Operational Guide — GoERP v1.0

## 1. Getting Started

### 1.1 Accessing the ERP System
1. Open any modern web browser (Google Chrome, Microsoft Edge, Mozilla Firefox, or Safari) and navigate to:
   **`http://localhost:8080`**
2. The application includes a built-in **Enterprise Demo Auto-Login** as **Administrator (`admin`)** for **PT Golang ERP Nusantara**, allowing you to explore all 50 enterprise modules immediately.
3. If you log out using the **🚪** icon in the bottom-left corner, the system refreshes your session seamlessly.

### 1.2 User Interface Layout
The GoERP interface is structured into three primary areas:
- **Sidebar Navigation (Left)**: Displays the full, scrollable directory of **50 Enterprise Modules**. Click on any module name to switch screens instantly without page reloads (SPA architecture).
- **Top Header Bar**:
  - Displays the active module title and operational subtitle.
  - **🔍 Global Search (`Ctrl+K`)**: Opens the universal Omni-Search command palette.
  - **🔔 Notifications**: Actionable notification center badge for pending approvals and task queues.
  - **☀️ / 🌙 Theme Toggle**: Switch seamlessly between Light Enterprise and Dark mode.
  - **📷 Scan Barcode**: Opens the universal camera barcode/QR scanner.
  - **✨ Ask ERP AI**: Executive AI Copilot to ask natural language questions about your business data.
- **Main Content Area**: Hosts KPI metric cards, operational sub-tabs, data tables with live filtering and sorting, CSV export buttons, and interactive modals.

---

## 2. Core Operational Modules

### 2.1 Executive Dashboard (`Dashboard`)
- Features **6 Core Metric Cards**: *Total Sales (Confirmed)*, *Sales Orders*, *Purchase Orders*, *Stock Pickings*, *Invoices & Bills*, and *Active Products*.
- **Quick Workflow Actions**: 1-click creation cards to launch a new *Sales Order*, *Purchase Order*, *Product*, or *Partner*.
- **Executive Pulse Strip**: High-level financial indicators for liquid bank cash, Accounts Receivable (AR), Accounts Payable (AP), CRM pipeline revenue, and MRR.

### 2.2 Sales & Invoicing (`Sales Orders` & `Invoicing`)
1. Open **Sales Orders** in the sidebar, then click **+ Create Sales Order**.
2. Select a **Customer**, add line items with quantities and unit prices, then click **Save Quotation**.
3. On a `DRAFT` order, click **Confirm** to convert it to a confirmed `SALE`. The system automatically generates a Delivery Order (`WH/OUT`) in **Inventory**.
4. To print official business documents, click the **🖨️ Print** icon to generate an A4 SHA-256 sealed document ready for printing or saving as PDF.

### 2.3 Purchasing & Warehouse Operations (`Purchase Orders` & `Inventory`)
1. Open **Purchase Orders**, click **+ Create RFQ**, choose a **Vendor**, add items, and save.
2. Click **Confirm PO**. The system automatically generates a warehouse incoming receipt document (`WH/IN`).
3. Open **Inventory**, find the incoming `WH/IN` or outgoing `WH/OUT` record with status `ASSIGNED` or `DRAFT`, and click **Validate**. Stock on hand in **Stock On-Hand (Quants)** updates atomically.

### 2.4 Manufacturing & MRP II (`Manufacturing` & `MRP II: Work Center & OEE`)
1. In **Manufacturing**, manage **Bills of Materials (BoM)** or issue new production orders via **+ Manufacturing Order**.
2. Click **Produce** on a Manufacturing Order to atomically consume raw components and receive finished goods into stock.
3. In **MRP II: Work Center, Routing & OEE**, manage production machinery stations (*Work Centers*), configure *Routing*, start/finish shop-floor *Work Orders*, and run the **🤖 AI OEE & Bottleneck Analyzer**.

### 2.5 Human Resources, Attendance, Leaves & Payroll (`HR`, `Attendance`, `Leave` & `Enterprise Payroll`)
1. **HR & Employees**: Manage departments and employee profiles with base salary information.
2. **Attendance**: Use **📍 Clock In** and **🏁 Clock Out** with GPS location capture to log employee work hours.
3. **Enterprise Payroll**:
   - Open **Enterprise Payroll**, click **+ Generate Payslip** or **⚡ Batch Payroll Run**.
   - The engine automatically calculates base salary, allowances, health insurance, social security, and tax withholdings.
   - Click **Confirm & Post to GL** to post balanced payroll journal entries to the General Ledger.

### 2.6 Fiscal & Tax Localization (`Tax & e-Faktur` & `Unified e-Bupot`)
1. **VAT 11% (e-Faktur)**:
   - Register official tax invoice numbers (**NSFP**) via **+ Register NSFP**.
   - Create Output/Input tax invoices, approve them, export official CSV formats, or run the **🤖 AI Tax Compliance Auditor**.
2. **Withholding Tax (e-Bupot)**:
   - Issue official withholding tax slips and manage unified periodic tax returns.

### 2.7 Customs & Bonded Zones (`Customs & Bonded Zones (CEISA 4.0)`)
1. Open **Customs & Bonded Zones**.
2. Click **+ Create Customs Document** to register customs declarations (**BC 2.3**, **BC 2.5**, **BC 2.7**, **BC 4.0**, **BC 4.1**, **BC 2.6.1**, **BC 2.6.2**, or **BC 3.0**).
3. Click **🛃 Issue SPPB** on submitted declarations to authorize goods movement and automatically update the **4 Official IT Inventory Ledgers** (Raw Materials, Finished Goods, Machinery, and Scrap/Waste).
4. Run the **🤖 AI Customs Compliance Audit** to reconcile physical warehouse stock with customs balances.

---

## 3. Advanced Platform Tools

### 3.1 Universal Low-Code Studio (`Low-Code Studio`)
Create custom business models and data tables without writing Go code:
1. Open **Low-Code Studio** from the sidebar.
2. **Manual Model Creation**:
   - Click **➕ Create Custom Model**, provide a Model Name (e.g., *Mine Safety Inspection*), Technical Name (`x_mine_safety`), menu category, and save.
   - Click **🧩 Fields** to add custom columns (Text, Integer, Float, Date, Dropdown Selection, Boolean, or JSON).
3. **AI Schema Copilot**:
   - Click **🤖 AI Schema Copilot**, enter a plain-language prompt of your business requirements, click **🚀 Design Schema with AI**, then click **✅ Apply Schema to ERP Database**.
4. **Data Management**:
   - Click **📊 Browse Data** on any model (or open **Dynamic Data Explorer**) to create, edit, and delete records dynamically.
5. **Core Entity Extensions**:
   - Open the **🧩 Core Entity Extensions** sub-tab to append custom dynamic JSON attributes (`x_...`) onto standard models like `res.partner`, `product.product`, `sale.order`, or `purchase.order`.

### 3.2 BPMN 2.0 Dynamic Workflow Engine (`Workflow Engine`)
Visually configure multi-tiered document approval workflows:
1. Open **Workflow Engine**.
2. Under **📋 Workflow Definitions & Visual Canvas**, click **👁️ View Diagram** to render the BPMN 2.0 visual state machine with *Exclusive Gateway* conditional branching (e.g., `IF amount >= 50000000`).
3. Click **🤖 AI Workflow Designer** to generate complete approval workflows from a text prompt, or click **+ Create Workflow** to define custom nodes and transitions.
4. Click **🚀 Launch Instance** to initiate an approval process against any ERP document.
5. Review and process approval tasks (**📤 Submit**, **✅ Approve**, **❌ Reject**) from the **📥 Approval Task Inbox** or inspect historical audit trails in **⚡ Active Instances & Audit Trail**.

### 3.3 Universal Integration Hub & Open RPC Gateway (`Integration Hub & API`)
Manage third-party integrations and external API connectivity:
1. **Pre-Built Connector Catalog**: Test external connectivity (**🔌 Test Connection**), trigger marketplace stock synchronization (**🔄 Sync Stock**), or send test transactional WhatsApp messages (**💬 Test WA**).
2. **M2M API Key Management**: Click **🔑 Generate M2M API Key** to provision secure API keys (`ak_live_...`) with granular scope constraints and rate limiting.
3. **Universal Open RPC Playground**: Under **🧪 Universal Open RPC Playground**, select an operation (`search_read`, `create`, `write`, `unlink`, `execute_action`), pick a target ERP model, configure your JSON payload, and click **🚀 Send RPC Request** to test live API responses.
4. **Batch Data ETL Jobs**: Execute batch export or import jobs for thousands of records across JSON and CSV formats.

---

## 4. Continuous Improvement Features

### 4.1 Universal Command Palette & Omni-Search (`Ctrl+K`)
1. Press the **`Ctrl+K`** shortcut anywhere in the application (or click **🔍 Global Search** in the header).
2. Type any module name (e.g., `accounting`, `wms`, `payroll`), document number (`SO00001`, `PO00001`, `INV/`), product name, or customer name.
3. Click any result to navigate directly to the target record or screen.

### 4.2 Sidebar Instant Module Filter (`/`)
1. Press **`/`** on your keyboard (or click into the **🔍 Search 50 modules...** input in the left sidebar).
2. Type a keyword to filter the 50 modules in real time. Click **`✕`** to reset the filter.

### 4.3 1-Click Business Cycle Simulator
1. Open the **📊 Dashboard**.
2. On the **Enterprise Command Center** banner, click **🚀 Simulate Business Cycle (1-Click)**.
3. The engine automatically creates & confirms a Purchase Order, receives goods into stock (`WH/IN`), creates & confirms a Sales Order, executes warehouse delivery (`WH/OUT`), and posts a balanced Customer Invoice (`INV`) with 11% VAT to the General Ledger.

### 4.4 Notification Center (`🔔`) & Enterprise Theme Switcher (`☀️ / 🌙`)
- Click **🔔 Notifications** to view pending approval tasks, draft documents, and operational alerts.
- Click **☀️ Light / 🌙 Dark** to toggle themes with settings preserved in your browser's local storage.

---

## 5. Frequently Asked Questions (FAQ)

- **How do I export data tables to Microsoft Excel?**
  Almost all data tables include an **📥 Export CSV** button at the top right. Downloaded CSV files can be opened directly in Microsoft Excel, Google Sheets, or LibreOffice Calc.
- **Does the system function without internet connectivity or Cloudflare credentials?**
  **Yes.** All 20+ AI Copilot features in GoERP include a built-in *Smart Local Heuristic Fallback* that provides instant financial insights, risk audits, and schema generation even when external AI API keys are not configured.
- **How do I export Invoices or Reports as PDF?**
  Click the **🖨️ Print** icon on any transaction record, review the print preview window, and select **Save as PDF** in your browser's print dialog.
