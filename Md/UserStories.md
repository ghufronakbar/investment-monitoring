# MRT Jakarta Phase 2 Monitoring System - Master User Stories

**Document Version:** 1.0 (Final Revision)
**Stack:** Go (Golang), PostgreSQL, Prisma, Goose, Cloudinary.

---

## 🎭 Actor & Role Definitions

Sebelum masuk ke story, berikut adalah definisi Aktor dalam sistem sesuai Schema Database:

1.  **GOV_SUPERVISOR** (e.g., Bappenas/Kemenkeu)
    - _Peran:_ Pemilik Strategi & Inisiator Proyek.
    - _Akses:_ Full View (Monitoring & Finance), Create Project.
2.  **LOAN_PROVIDER** (e.g., JICA)
    - _Peran:_ Pemberi Dana.
    - _Akses:_ Read-Only Monitoring, Finance Disbursement (Mark as Paid).
3.  **EXECUTING_AGENCY** (e.g., PT MRT Jakarta)
    - _Peran:_ Pemilik Proyek (Project Owner) & Admin Operasional.
    - _Akses:_ Manage Contract, Assign Tickets, Approve Finance (Admin).
4.  **CONTRACTOR** (e.g., Shimizu-Adhi JO)
    - _Peran:_ Pelaksana Konstruksi.
    - _Akses:_ Submit Progress, Create Ticket, Request Payment.
5.  **INDEPENDENT_AUDITOR** (e.g., Nippon Koei)
    - _Peran:_ Pengawas/Wasit.
    - _Akses:_ Validate Progress, Approve Finance (Technical).
6.  **GOV_PARTNER** (e.g., BPN, PLN, Dishub)
    - _Peran:_ Instansi Pendukung.
    - _Akses:_ Resolve Assigned Tickets.

---

## 🔐 MODULE 1: IAM (Identity & Access Management)

### Story 1.1: Smart Login & Invitation Check

**As a** User,
**I want to** log in using my email,
**So that** the system can verify if I am invited or already active.

**Acceptance Criteria:**

1.  **Step 1 (Check Email):** Frontend mengirim email ke `/iam/auth/check-email`.
    - Jika email tidak ada: Return Error "Invitation Required".
    - Jika email ada tapi `password` null: Return "Setup Required" -> Arahkan user cek inbox email.
    - Jika email ada & `password` set: Return "Login Ready" -> Tampilkan input password.
2.  **Step 2 (Login):** User input password -> Return JWT (Access & Refresh Token).
3.  **Step 3 (Invitation Verify):** Jika user klik link di email (`/verify?token=...`), validasi token tersebut.

**Technical Notes:**

- **Table:** `users`
- **Endpoints:** `POST /iam/auth/check-email`, `POST /iam/auth/login`, `GET /iam/auth/invitation/verify`

### Story 1.2: Organization Setup & Admin Invite

**As a** GOV_SUPERVISOR or EXECUTING_AGENCY,
**I want to** create a partner organization (e.g., Create Org "PT Shimizu") and invite their Admin,
**So that** they can access the system.

**Acceptance Criteria:**

1.  User mengisi: Nama Organisasi, Tipe Role (misal `CONTRACTOR`), Email Admin, Nama Admin.
2.  System Action (Atomic Transaction):
    - Create `Organization`.
    - Create `User` (Admin) dengan status pending.
    - Create `OrganizationMember` (Role: ADMIN).
    - Send Invitation Email ke Admin tersebut.

**Technical Notes:**

- **Table:** `organizations`, `users`, `organization_members`
- **Endpoints:** `POST /iam/organization`

### Story 1.3: Internal Member Management

**As an** Organization Admin,
**I want to** invite my staff members,
**So that** I can delegate tasks.

**Acceptance Criteria:**

1.  Admin bisa invite user baru via email.
2.  User baru otomatis masuk ke organisasi pengundang dengan role `MEMBER`.
3.  User `MEMBER` memiliki akses terbatas (hanya bisa Draft/Upload, tidak bisa Final Submit di beberapa modul).

**Technical Notes:**

- **Endpoints:** `POST /iam/organization/{id}/member/invite`

---

## 🏗️ MODULE 2: PROJECT (Core Data & Contract)

### Story 2.1: Project Initialization

**As a** GOV_SUPERVISOR,
**I want to** create a new project and upload the Loan Agreement,
**So that** the project exists in the system.

**Acceptance Criteria:**

1.  Input: Nama Proyek, Deskripsi.
2.  **Upload:** PDF Loan Agreement (Wajib). Simpan URL-nya.
3.  Otomatis assign pembuat (Bappenas) ke dalam kontrak sebagai `govSupervisor`.

**Technical Notes:**

- **Table:** `projects`, `contracts`
- **Endpoints:** `POST /project`

### Story 2.2: Configure Parameters

**As an** EXECUTING_AGENCY,
**I want to** set the budget, timeline, and currency,
**So that** the S-Curve calculation works correctly.

**Acceptance Criteria:**

1.  Input: `Total Budget` (BigInt), `Start Date`, `Baseline Deadline`, `Currency`.
2.  Validasi: `Baseline Deadline` > `Start Date`.
3.  Data ini menjadi acuan 0% (Start) dan 100% (Deadline) pada grafik.

**Technical Notes:**

- **Table:** `project_settings`
- **Endpoints:** `PUT /project/{id}/settings`

### Story 2.3: Direct Contract Assignment (No Bidding)

**As an** EXECUTING_AGENCY,
**I want to** assign the Contractor, Auditor, and Gov Partners directly,
**So that** the project team is formed without a complex bidding process in the system.

**Acceptance Criteria:**

1.  **Assign Vendor:** Pilih Organisasi `CONTRACTOR`, Input Nilai Kontrak, Upload PDF Kontrak.
2.  **Assign Auditor:** Pilih Organisasi `INDEPENDENT_AUDITOR`.
3.  **Assign Partners:** Pilih multiple Organisasi `GOV_PARTNER` (BPN, PLN, dll).
4.  Setelah di-assign, organisasi-organisasi tersebut otomatis bisa melihat Project ini di dashboard mereka.

**Technical Notes:**

- **Table:** `contracts`, `gov_participants`
- **Endpoints:** `PUT /project/{id}/contract/assign-vendor`, `PUT .../assign-auditor`, `PUT .../assign-gov-partners`

---

## 📈 MODULE 3: MONITORING (WBS & Progress)

### Story 3.1: Define WBS (Master Schedule)

**As an** EXECUTING_AGENCY (PMO),
**I want to** create the Work Breakdown Structure (Tasks & Weights),
**So that** we have a baseline to measure progress.

**Acceptance Criteria:**

1.  Bisa membuat Task bertingkat (Parent -> Child).
2.  Input: Judul, Bobot (Weight), Deadline.
3.  **Validasi:** Total Weight seluruh Task level root harus 100% (atau total weight children = weight parent).

**Technical Notes:**

- **Table:** `task_items`
- **Endpoints:** `POST /monitoring/project/{id}/tasks`

### Story 3.2: Submit Progress Report

**As a** CONTRACTOR,
**I want to** report progress on a specific task with evidence,
**So that** I can claim the work done.

**Acceptance Criteria:**

1.  Pilih Task. Input `% Progress` (Kumulatif).
2.  Input Deskripsi/Laporan Harian.
3.  **Evidence:** Wajib upload minimal 1 foto (via Cloudinary URL).
4.  **Snapshot:** Sistem menyimpan judul & bobot task saat itu ke dalam log (untuk history jika WBS berubah).
5.  Status: Menunggu Validasi Auditor.

**Technical Notes:**

- **Table:** `progress_logs`
- **Endpoints:** `POST /monitoring/task/{id}/log`

### Story 3.3: Progress Validation

**As an** INDEPENDENT_AUDITOR,
**I want to** validate the contractor's claim,
**So that** the system reflects the _real_ progress.

**Acceptance Criteria:**

1.  Auditor melihat klaim kontraktor + foto bukti.
2.  **Approve:** Input `real_progress_percentage` (bisa sama dengan klaim atau dikoreksi). -> Data masuk kalkulasi S-Curve.
3.  **Reject:** Wajib isi komentar. -> Data diabaikan dari S-Curve.

**Technical Notes:**

- **Table:** `progress_validations`
- **Endpoints:** `POST /monitoring/log/{id}/validate`

---

## 🚧 MODULE 4: TRACKER (Ticketing System)

### Story 4.1: Report Issue

**As a** CONTRACTOR,
**I want to** report an obstruction (Ticket),
**So that** the project owner acts on it.

**Acceptance Criteria:**

1.  Input: Judul, Deskripsi, Lokasi, Foto Bukti.
2.  Optional: Link ke `taskItemId` (Task mana yang terhambat?).
3.  Status Awal: `OPEN`.

**Technical Notes:**

- **Table:** `tickets`
- **Endpoints:** `POST /tracker/project/{id}/ticket`

### Story 4.2: Assign Ticket

**As an** EXECUTING_AGENCY,
**I want to** assign the ticket to a specific Gov Partner,
**So that** the issue is handled by the right authority.

**Acceptance Criteria:**

1.  Review tiket -> Pilih Org `GOV_PARTNER` (misal BPN).
2.  Status berubah jadi `ASSIGNED`.
3.  Partner terkait mendapat notifikasi.

**Technical Notes:**

- **Endpoints:** `PUT /tracker/ticket/{id}/assign`

### Story 4.3: Resolve Issue

**As a** GOV_PARTNER,
**I want to** update the ticket status and attach proof of resolution,
**So that** the issue can be closed.

**Acceptance Criteria:**

1.  Partner update log: "Sedang dikerjakan" atau "Selesai".
2.  Upload lampiran (misal PDF Berita Acara).
3.  Status berubah jadi `IN_PROGRESS` atau `RESOLVED`.

**Technical Notes:**

- **Table:** `ticket_logs`
- **Endpoints:** `POST /tracker/ticket/{id}/log`

---

## 💸 MODULE 5: FINANCE (Disbursement)

### Story 5.1: Request Payment

**As a** CONTRACTOR,
**I want to** submit a payment request (Invoice),
**So that** I can receive funds.

**Acceptance Criteria:**

1.  Input: Nominal, Judul (Termin ke-X).
2.  **Upload:** File Invoice/Tagihan (PDF).
3.  Status: `SUBMITTED`.

**Technical Notes:**

- **Table:** `payment_requests`
- **Endpoints:** `POST /finance/project/{id}/request`

### Story 5.2: Tiered Approval (Auditor & Agency)

**As an** AUDITOR and EXECUTING_AGENCY,
**I want to** approve the request in sequence,
**So that** payment is verified technically and administratively.

**Acceptance Criteria:**

1.  **Step 1 (Auditor):** Cek % progress fisik. Approve/Reject. Status -> `AUDITOR_APPROVED`.
2.  **Step 2 (Agency):** Cek kelengkapan dokumen. Approve/Reject. Status -> `AGENCY_APPROVED`.
3.  Jika Rejected di tahap manapun, status kembali ke Kontraktor/Stop.

**Technical Notes:**

- **Endpoints:** `PUT .../approve-auditor`, `PUT .../approve-agency`

### Story 5.3: Disbursement (JICA)

**As a** LOAN_PROVIDER,
**I want to** confirm that funds have been transferred,
**So that** the payment cycle is complete.

**Acceptance Criteria:**

1.  JICA Admin klik "Mark as Paid".
2.  Status berubah jadi `PAID`.
3.  Tercatat waktu pembayaran (`loanProviderAt`).

**Technical Notes:**

- **Endpoints:** `PUT .../disburse`

---

## 🛠️ MODULE 6: SUPPORT (Utility & Dashboard)

### Story 6.1: File Upload

**As a** User,
**I want to** upload files to cloud storage,
**So that** I get a public URL to attach to my reports.

**Acceptance Criteria:**

1.  Upload File (Image/PDF) -> Save to Cloudinary.
2.  Return: URL String.

**Technical Notes:**

- **Endpoints:** `POST /support/upload`

### Story 6.2: S-Curve Calculation

**As a** System/Dashboard,
**I want to** calculate the Plan vs Actual progress,
**So that** users see the project health.

**Acceptance Criteria:**

1.  **Plan:** Akumulasi `TaskItem.weight` berdasarkan timeline hari ini vs `project_settings.baselineDeadline`.
2.  **Actual:** Akumulasi `ProgressValidation.realProgressPercentage` \* `TaskItem.weight`.
3.  Return data series JSON.

**Technical Notes:**

- **Endpoints:** `GET /support/dashboard/s-curve/{id}`

---
