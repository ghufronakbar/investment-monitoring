# Module: SUPPORT (Utility & Dashboard)

## Epics

Fungsi pendukung visualisasi data (Dashboard) dan manajemen file (Utility). Modul ini tidak menyimpan data transaksional sendiri, melainkan mengagregasi data dari modul lain (Project, Monitoring, Finance) untuk kebutuhan reporting.

---

### Story 1: Upload File (Utility)

**As a** User (All Roles),
**I want to** upload images or PDFs to the server,
**So that** I can get a public URL to attach to my reports, contracts, or tickets.

#### Acceptance Criteria

1.  **Validation:**
    - **Format:** Hanya perbolehkan `JPG`, `JPEG`, `PNG`, `PDF`. Tolak `.exe` atau script lain.
    - **Size:** Maksimal 5MB per file.
2.  **Cloud Storage Integration:**
    - Backend tidak menyimpan file di disk lokal / database (BLOB).
    - Backend mengupload file ke **Cloudinary**.
3.  **Response:**
    - Return JSON berisi URL publik: `{ "url": "https://res.cloudinary.com/..." }`.
    - URL inilah yang nanti dikirim Frontend ke endpoint lain (misal: `create-ticket`).

#### Technical Notes & Schema Mapping

- **Endpoint:** `POST /support/upload`
- **No DB Table:** Modul ini _stateless_ terhadap database. Hanya berinteraksi dengan Cloud Storage Service.

---

### Story 2: View S-Curve (Dashboard)

**As a** Stakeholder (JICA/Bappenas/MRT),
**I want to** see the S-Curve graph overlaying Plan vs Actual,
**So that** I can instantly assess the project schedule performance (Delayed/Ahead).

#### Acceptance Criteria

1.  **Data Source - Time Axis (Sumbu X):**
    - Rentang waktu diambil dari table `project_settings`: `start_date` sampai `baseline_deadline`.
    - Data di-generate per satuan waktu (misal: Per Minggu atau Per Bulan).
2.  **Logic - Plan Line (Garis Rencana):**
    - Rumus: Akumulasi `task_items.weight` di mana `task_items.deadline` <= Tanggal Sumbu X.
    - _Contoh:_ Jika Minggu ke-10 deadline task A (Bobot 5%) dan B (Bobot 10%) sudah lewat, maka Plan = 15%.
3.  **Logic - Actual Line (Garis Realisasi):**
    - Rumus: Akumulasi (`progress_logs.snapshot_weight` \* `progress_validations.real_progress_percentage` / 100).
    - Filter: Hanya ambil data yang `progress_validations.is_accepted = TRUE` dan `updated_at` <= Tanggal Sumbu X.
4.  **Deviation (Deviasi):**
    - Hitung `Actual - Plan`. (Negatif = Terlambat).
5.  **Structure:**
    - Response JSON berupa Array of Objects: `[{ date: "2024-01-01", plan: 5.0, actual: 2.1 }, ...]`.

#### Technical Notes & Schema Mapping

- **Endpoint:** `GET /support/dashboard/s-curve/{projectId}`
- **Required Joins:**
  1.  **Get Project Settings:** `projects` -> `project_settings` (untuk Start/End Date).
  2.  **Get Contract:** `projects` -> `contracts` (untuk dapat `contract_id`).
  3.  **Get Plan Data:** `contracts` -> `task_items`.
  4.  **Get Actual Data:** `task_items` -> `progress_logs` -> `progress_validations`.

---

### Story 3: Dashboard Statistics (Summary)

**As a** Stakeholder,
**I want to** see high-level summary numbers,
**So that** I can capture the project status in a glance.

#### Acceptance Criteria

1.  **Total Budget & Disbursement:**
    - `Total Budget`: Ambil dari `project_settings.total_budget`.
    - `Disbursed`: Sum `amount` dari table `payment_requests` di mana `status = 'PAID'`.
2.  **Physical Progress:**
    - Ambil nilai _Actual Progress_ terakhir dari kalkulasi S-Curve.
3.  **Issues/Tickets:**
    - Hitung jumlah tiket berdasarkan status dari `ticket_logs` terakhir (Open, In Progress, Resolved).

#### Technical Notes & Schema Mapping

- **Endpoint:** `GET /support/dashboard/stats/{projectId}`
- **Tables:** `project_settings`, `payment_requests`, `tickets`, `ticket_logs`.
