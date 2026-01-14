# Module: MONITORING (WBS & Progress)

## Epics

Pemantauan progres fisik lapangan secara real-time, transparan, dan terstruktur. Modul ini mengubah klaim subyektif kontraktor menjadi data obyektif yang tervalidasi untuk perhitungan Kurva S.

---

### Story 1: Define WBS (Work Breakdown Structure)

**As an** EXECUTING_AGENCY (PMO Team),
**I want to** create a hierarchical structure of tasks with weights and deadlines,
**So that** we have a "Master Schedule" (Baseline) to measure the project's performance.

#### Acceptance Criteria

1.  **Hierarchy Support (Parent-Child):**
    - User bisa membuat Task yang memiliki Sub-Task (Parent -> Child).
    - Sistem mendukung kedalaman level tak terbatas (via `parent_id` self-reference).
2.  **Weight Validation:**
    - Setiap task memiliki `weight` (Bobot).
    - Validasi Backend: Total bobot seluruh _Task_ di level Root harus bernilai 100% (atau representasi integer 100).
3.  **Deadline Constraint:**
    - Setiap task wajib memiliki `deadline`.
    - Deadline Child tidak boleh melebihi Deadline Parent (Logical Validation).
4.  **Leaf Node Logic:**
    - Hanya Task yang **tidak memiliki anak** (Leaf Node) yang nantinya akan muncul di dropdown pelaporan progres Kontraktor. Task yang memiliki anak dianggap sebagai "Header/Grouping" saja.

#### Technical Notes & Schema Mapping

- **Endpoint:** `POST /monitoring/project/{id}/tasks`
- **Table:** `task_items`
  - `id` (UUID)
  - `title` (TEXT)
  - `weight` (INT) -> _Bobot perencanaan_
  - `deadline` (TIMESTAMPTZ)
  - `parent_id` (UUID, Nullable) -> _Untuk hierarki_
  - `contract_id` (UUID)

---

### Story 2: Submit Progress Report

**As a** CONTRACTOR (Site Engineer),
**I want to** report the cumulative progress of a specific task with visual evidence,
**So that** I can claim the work done and trigger a validation request.

#### Acceptance Criteria

1.  **Cumulative Reporting:**
    - User memilih Task (Leaf Node) dari WBS.
    - Input `progress_percentage` (Nilai 0-100). Ini adalah angka klaim sepihak dari Kontraktor.
2.  **Mandatory Evidence (No Pic = Hoax):**
    - User **WAJIB** mengupload minimal 1 foto bukti lapangan.
    - Sistem menyimpan URL foto tersebut dalam bentuk Array (`TEXT[]`) karena satu task bisa memiliki banyak angle foto.
3.  **Snapshotting (Data Integrity):**
    - **CRITICAL:** Saat data disimpan, sistem harus **meng-copy** nilai `title`, `description`, dan `weight` dari tabel `task_items` ke dalam kolom `snapshot_...` di tabel `progress_logs`.
    - _Alasan:_ Jika di masa depan PMO mengubah bobot di WBS (`task_items`), data history laporan ini tidak boleh ikut berubah.
4.  **Status:** Laporan yang masuk otomatis berstatus "Pending Validation" (karena belum ada entry di tabel `progress_validations`).

#### Technical Notes & Schema Mapping

- **Endpoint:** `POST /monitoring/task/{id}/log`
- **Table:** `progress_logs`
  - `progress_percentage` (INT) -> _Klaim Kontraktor_
  - `evidence_image_urls` (TEXT[]) -> _Array URL Cloudinary_
  - `snapshot_weight` (INT) -> _Copy dari task_items.weight saat itu_
  - `snapshot_title` (TEXT)
  - `task_item_id` (UUID)
  - `vendor_id` (UUID) -> _Diambil dari token login user_

---

### Story 3: Validate Progress

**As an** INDEPENDENT_AUDITOR,
**I want to** review the contractor's claim and decide the "Real Percentage",
**So that** only valid data enters the official S-Curve calculation.

#### Acceptance Criteria

1.  **Review Interface:**
    - Auditor melihat detail klaim: Foto Bukti, Deskripsi, dan Angka % Klaim Kontraktor.
2.  **Action: APPROVE:**
    - Auditor setuju dengan klaim.
    - Input `is_accepted = true`.
    - Input `real_progress_percentage`. (Biasanya sama dengan klaim kontraktor, tapi Auditor berhak mengoreksi, misal Kontraktor klaim 50%, Auditor hanya mengakui 45%).
3.  **Action: REJECT:**
    - Auditor menolak klaim (misal foto tidak sesuai/buram).
    - Input `is_accepted = false`.
    - Wajib mengisi `comment` (Alasan penolakan).
    - Data ini **tidak akan dihitung** dalam rumus Kurva S "Actual Progress".
4.  **Single Source of Truth:**
    - Satu `progress_log` hanya boleh memiliki satu `progress_validation` (Constraint `UNIQUE`).

#### Technical Notes & Schema Mapping

- **Endpoint:** `POST /monitoring/log/{id}/validate`
- **Table:** `progress_validations`
  - `progress_log_id` (UUID) -> _Foreign Key Unique_
  - `is_accepted` (BOOLEAN)
  - `real_progress_percentage` (INT) -> _Angka Final untuk S-Curve_
  - `comment` (TEXT)
  - `auditor_id` (UUID) -> _Siapa yang memvalidasi_
