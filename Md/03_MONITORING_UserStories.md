# Module: MONITORING (WBS & Progress)

## Epics

Pemantauan progres fisik lapangan secara real-time dan transparan.

---

### Story 1: Define WBS (Work Breakdown Structure)

**As an** EXECUTING_AGENCY (PMO),
**I want to** create a hierarchy of tasks and weights,
**So that** we have a Master Schedule to track against.

#### Acceptance Criteria

1.  Bisa membuat Task (Parent) dan Sub-Task (Child).
2.  Setiap Task memiliki `Weight` (Bobot). Total bobot proyek harus 100% (Validasi di FE/BE).
3.  Setiap Task memiliki `Deadline`.
4.  Hanya Task level terbawah (Leaf Node) yang bisa dilaporkan progresnya nanti.

#### Technical Notes

- Endpoint: `POST /monitoring/project/{id}/tasks`.
- Table: `task_items`.

---

### Story 2: Submit Progress Report

**As a** CONTRACTOR (Site Engineer),
**I want to** report the weekly progress of a specific task,
**So that** I can claim the work done.

#### Acceptance Criteria

1.  User memilih Task spesifik.
2.  Input `% Progress` (Kumulatif).
3.  **Mandatory Evidence:** Wajib upload minimal 1 foto lapangan.
4.  Sistem menyimpan "Snapshot" (Judul Task & Bobot saat itu) untuk integritas data historis.
5.  Status laporan menjadi `PENDING VALIDATION`.

#### Technical Notes

- Endpoint: `POST /monitoring/task/{id}/log`.
- Table: `progress_logs`.

---

### Story 3: Validate Progress

**As an** INDEPENDENT_AUDITOR,
**I want to** approve or reject the contractor's report,
**So that** only valid data enters the S-Curve calculation.

#### Acceptance Criteria

1.  Auditor melihat foto bukti dan angka klaim.
2.  **Action Approve:** Data masuk ke `real_progress_percentage`.
3.  **Action Reject:** Wajib memberikan komentar alasan penolakan. Data tidak dihitung di Kurva S.
4.  Audit trail mencatat siapa auditor yang melakukan klik.

#### Technical Notes

- Endpoint: `POST /monitoring/log/{id}/validate`.
- Table: `progress_validations`.
