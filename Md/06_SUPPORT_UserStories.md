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
    - **[IMPLEMENTED]** Max 10 files untuk multi-upload.
2.  **Cloud Storage Integration:**
    - Backend tidak menyimpan file di disk lokal / database (BLOB).
    - Backend mengupload file ke **Cloudinary**.
3.  **Response:**
    - Return JSON berisi URL publik.
    - URL inilah yang nanti dikirim Frontend ke endpoint lain (misal: `create-ticket`).

#### Technical Notes & Schema Mapping

- **Single Upload Endpoint:** `POST /support/upload/single`
  - Form field: `file`
  - Response: `{ "publicId": "...", "url": "https://...", "format": "jpg" }`
- **Multi Upload Endpoint:** `POST /support/upload/multiple`
  - Form field: `files` (multiple)
  - Response: `{ "files": [...], "count": 3 }`
- **No DB Table:** Modul ini _stateless_ terhadap database. Hanya berinteraksi dengan Cloud Storage Service.

---

### Story 2: View S-Curve (Dashboard)

**As a** Stakeholder (JICA/Bappenas/MRT),
**I want to** see the S-Curve graph overlaying Plan vs Actual,
**So that** I can instantly assess the project schedule performance (Delayed/Ahead).

#### Acceptance Criteria

1.  **Data Source - Time Axis (Sumbu X):**
    - Rentang waktu diambil dari table `project_settings`: `start_date` sampai `baseline_deadline`.
    - **[IMPLEMENTED]** Data di-generate per Minggu (weekly).
2.  **Logic - Plan Line (Garis Rencana):**
    - Rumus: Akumulasi `task_items.weight` di mana `task_items.deadline` <= Tanggal Sumbu X.
    - **[IMPLEMENTED]** Hanya leaf tasks (tanpa children) yang dihitung.
3.  **Logic - Actual Line (Garis Realisasi):**
    - Rumus: Akumulasi (`progress_logs.snapshot_weight` \* `progress_validations.real_progress_percentage` / 100).
    - Filter: Hanya ambil data yang `progress_validations.is_accepted = TRUE` dan `updated_at` <= Tanggal Sumbu X.
4.  **Deviation (Deviasi):**
    - Hitung `Actual - Plan`. (Negatif = Terlambat).
5.  **Structure:**
    - Response JSON berupa Array of Objects.

#### Technical Notes & Schema Mapping

- **Endpoint:** `GET /support/dashboard/s-curve/{projectId}`
- **Response:**

```json
{
  "projectName": "Project X",
  "startDate": "2024-01-01T00:00:00Z",
  "baselineDeadline": "2024-12-31T00:00:00Z",
  "dataPoints": [
    { "date": "2024-01-01", "plan": 0, "actual": 0 },
    { "date": "2024-01-08", "plan": 5.0, "actual": 2.1 },
    ...
  ]
}
```

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
    - **[IMPLEMENTED]** Ambil nilai _Actual Progress_ terkini dari kalkulasi S-Curve.
3.  **Issues/Tickets:**
    - **[IMPLEMENTED]** Hitung jumlah tiket berdasarkan status (Open, In Progress, Resolved, Closed).

#### Technical Notes & Schema Mapping

- **Endpoint:** `GET /support/dashboard/stats/{projectId}`
- **Response:**

```json
{
  "projectName": "Project X",
  "totalBudget": "1000000000",
  "disbursedAmount": "200000000",
  "currency": "IDR",
  "physicalProgress": 45.5,
  "ticketStats": {
    "open": 5,
    "inProgress": 3,
    "resolved": 10,
    "closed": 2,
    "total": 20
  }
}
```
