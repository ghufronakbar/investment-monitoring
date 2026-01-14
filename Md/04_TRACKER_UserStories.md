# Module: TRACKER (Ticketing System)

## Epics

Penyelesaian hambatan lapangan (Issue Resolution) lintas instansi. Modul ini menggunakan pendekatan **Event-Sourced Status**, di mana status tiket saat ini ditentukan oleh log terakhir (_Latest Log Entry_).

---

### Story 1: Report Field Issue

**As a** CONTRACTOR (Site Engineer),
**I want to** report an obstruction (e.g., land not cleared) with evidence,
**So that** the project owner knows why the physical work is halted.

#### Acceptance Criteria

1.  **Input Data:**
    - User menginput Judul, Deskripsi, dan Lokasi.
    - **Evidence:** Wajib upload minimal 1 foto bukti (`evidence_image_urls`).
    - **Link Task (Optional):** User bisa memilih `task_item_id` dari WBS untuk menandakan pekerjaan mana yang terhambat.
2.  **Atomic Creation:**
    - Sistem membuat record di tabel `tickets`.
    - **CRITICAL:** Sistem secara otomatis membuat log pertama di tabel `ticket_logs` dengan:
      - `action`: 'CREATE'
      - `new_status`: 'OPEN'
      - `description`: "Ticket created"

#### Technical Notes & Schema Mapping

- **Endpoint:** `POST /tracker/project/{id}/ticket`
- **Table `tickets`:** Simpan data statis (Title, Location, Creator, Images).
- **Table `ticket_logs`:** Insert log pertama.

---

### Story 2: Assign Issue to Gov Partner

**As an** EXECUTING_AGENCY (PT MRT Admin),
**I want to** review the ticket and assign it to the relevant agency (e.g., BPN),
**So that** the right party handles the issue.

#### Acceptance Criteria

1.  **Assignment:**
    - Admin memilih Organisasi Partner (`GOV_PARTNER`) dari dropdown.
2.  **Update Responsibility:**
    - Sistem mengupdate kolom `ticket_assign_to_id` pada tabel `tickets`.
3.  **Log Status Change:**
    - Sistem membuat record baru di `ticket_logs` dengan:
      - `action`: 'ASSIGNED'
      - `new_status`: 'ASSIGNED'
      - `previous_status`: 'OPEN'
      - `description`: "Assigned to [Nama Organisasi Partner]"

#### Technical Notes & Schema Mapping

- **Endpoint:** `PUT /tracker/ticket/{id}/assign`
- **Table `tickets`:** Update `ticket_assign_to_id`.
- **Table `ticket_logs`:** Insert log baru (Status Change).

---

### Story 3: Resolve Issue (Update Progress)

**As a** GOV_PARTNER (e.g., BPN Staff),
**I want to** update the ticket with my actions or attach proof of resolution,
**So that** stakeholders know the issue is being handled.

#### Acceptance Criteria

1.  **Activity Logging:**
    - Partner bisa memposting komentar teks.
    - Partner bisa mengupload dokumen lampiran (`attachment_url`), misal: PDF Berita Acara / Surat Izin.
2.  **Status Update:**
    - Partner bisa mengubah status menjadi `IN_PROGRESS` (sedang dikerjakan) atau `RESOLVED` (selesai menurut Partner).
3.  **Record History:**
    - Setiap update disimpan sebagai baris baru di `ticket_logs`.

#### Technical Notes & Schema Mapping

- **Endpoint:** `POST /tracker/ticket/{id}/log`
- **Table `ticket_logs`:**
  - `action`: 'COMMENTED' atau 'STATUS_CHANGED'
  - `new_status`: Input dari user.
  - `organization_id`: ID BPN (dari Token).

---

### Story 4: Close Ticket

**As an** EXECUTING_AGENCY (PT MRT Admin),
**I want to** close the ticket definitively,
**So that** it confirms the obstruction is cleared and work can resume.

#### Acceptance Criteria

1.  **Final Verification:**
    - Hanya MRT yang memiliki tombol "Close Ticket".
2.  **Closing:**
    - Sistem membuat record terakhir di `ticket_logs` dengan:
      - `action`: 'RESOLVED' (atau 'STATUS_CHANGED')
      - `new_status`: 'CLOSED'
      - `description`: "Ticket closed by Project Owner."
3.  **Effect:** Tiket dengan status terakhir 'CLOSED' tidak lagi muncul di daftar "Outstanding Issues".

#### Technical Notes & Schema Mapping

- **Endpoint:** `PUT /tracker/ticket/{id}/close`
- **Logic:** Cukup Insert ke `ticket_logs`. Tidak perlu ubah tabel `tickets`.
- **Querying Status:** Untuk mengetahui status tiket saat ini, Backend harus mengambil **Log Terakhir** (`ORDER BY created_at DESC LIMIT 1`) dari setiap tiket.
