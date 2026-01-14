# Module: PROJECT (Core Data & Contracting)

## Epics

Inisialisasi proyek dan penunjukan langsung (Direct Assignment) para pemangku kepentingan.

---

### Story 1: Project Initialization

**As a** GOV_SUPERVISOR (Bappenas),
**I want to** input the initial project data based on the G-to-G agreement,
**So that** the project has a digital baseline.

#### Acceptance Criteria

1.  Input wajib: Nama Proyek, Deskripsi.
2.  **Mandatory Upload:** User wajib mengupload file PDF "Loan Agreement" (via Cloudinary) sebagai referensi legal.
3.  Otomatis assign pembuat (Bappenas) sebagai `govSupervisor` di kontrak.

#### Technical Notes

- Endpoint: `POST /project`.
- Table: `projects`, `contracts`.

---

### Story 2: Configure Project Settings

**As an** EXECUTING_AGENCY (PT MRT Admin),
**I want to** define the budget, timeline, and currency,
**So that** the S-Curve calculation has accurate parameters.

#### Acceptance Criteria

1.  Input: `Total Budget` (BigInt), `Start Date`, `Baseline Deadline`, `Currency`.
2.  Validasi: `Baseline Deadline` tidak boleh sebelum `Start Date`.
3.  Data ini akan mengunci perhitungan deviasi progres nanti.

#### Technical Notes

- Endpoint: `PUT /project/{id}/settings`.
- Table: `project_settings`.

---

### Story 3: Assign Contractor (Direct Assignment)

**As an** EXECUTING_AGENCY (PT MRT Admin),
**I want to** input the winner of the tender into the system,
**So that** the contractor can start reporting progress.

#### Acceptance Criteria

1.  Admin memilih Organisasi Kontraktor dari list yang sudah didaftarkan di IAM.
2.  Input nilai kontrak final (`contractValue`) dan upload PDF Kontrak (`contractDocUrl`).
3.  Sistem mengupdate tabel `contracts` kolom `vendor_participant_id`.
4.  Kontraktor yang dipilih sekarang bisa melihat proyek ini di dashboard mereka.

#### Technical Notes

- Endpoint: `PUT /project/{id}/contract/assign-vendor`.
- Table: `contracts`.

---

### Story 4: Assign Support Partners (Gov & Auditor)

**As an** EXECUTING_AGENCY (PT MRT Admin),
**I want to** assign the Independent Auditor and Gov Partners (BPN/PLN),
**So that** they can participate in validation and issue resolution.

#### Acceptance Criteria

1.  Multi-select input untuk Gov Partners (bisa lebih dari satu, misal BPN + PLN).
2.  Single-select input untuk Independent Auditor.
3.  User dari organisasi yang dipilih otomatis mendapatkan akses ke proyek tersebut.

#### Technical Notes

- Endpoint: `PUT .../assign-auditor`, `PUT .../assign-gov-partners`.
- Table: `gov_participants`.
