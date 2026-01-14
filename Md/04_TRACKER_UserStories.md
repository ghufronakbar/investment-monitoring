# Module: TRACKER (Ticketing System)

## Epics

Penyelesaian hambatan lapangan (Issue Resolution) lintas instansi.

---

### Story 1: Report Field Issue

**As a** CONTRACTOR,
**I want to** report an obstruction (e.g., land not cleared),
**So that** the project owner knows why work is halted.

#### Acceptance Criteria

1.  Input: Judul, Deskripsi, Lokasi.
2.  **Optional Link:** Bisa menautkan tiket ke `Task Item` tertentu (misal: "Pengecoran" terhambat).
3.  **Evidence:** Wajib upload foto kondisi masalah.
4.  Status awal tiket: `OPEN`.

#### Technical Notes

- Endpoint: `POST /tracker/project/{id}/ticket`.
- Table: `tickets`.

---

### Story 2: Assign Issue to Gov Partner

**As an** EXECUTING_AGENCY (PT MRT),
**I want to** review the ticket and assign it to the relevant agency (e.g., BPN),
**So that** the right party handles it.

#### Acceptance Criteria

1.  MRT memverifikasi laporan Kontraktor.
2.  MRT memilih organisasi `GOV_PARTNER` (misal: BPN Jakarta Pusat).
3.  Status tiket berubah menjadi `ASSIGNED`.
4.  BPN menerima notifikasi (via email/dashboard).

#### Technical Notes

- Endpoint: `PUT /tracker/ticket/{id}/assign`.
- Table: `tickets`, `ticket_logs`.

---

### Story 3: Resolve Issue

**As a** GOV_PARTNER (e.g., BPN Staff),
**I want to** update the ticket with my actions,
**So that** stakeholders know the issue is being handled.

#### Acceptance Criteria

1.  Partner bisa posting komentar atau upload dokumen bukti penyelesaian (misal: Surat Ukur).
2.  Partner mengubah status menjadi `RESOLVED` jika sudah selesai.
3.  JICA bisa melihat log aktivitas ini secara real-time.

#### Technical Notes

- Endpoint: `POST /tracker/ticket/{id}/log`.
- Table: `ticket_logs`.

---

### Story 4: Close Ticket

**As an** EXECUTING_AGENCY,
**I want to** close the ticket definitively,
**So that** it confirms the obstruction is cleared.

#### Acceptance Criteria

1.  Hanya MRT yang bisa mengubah status ke `CLOSED`.
2.  Memastikan masalah di lapangan benar-benar sudah tidak ada sebelum ditutup.

#### Technical Notes

- Endpoint: `PUT /tracker/ticket/{id}/close`.
