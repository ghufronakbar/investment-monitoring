# Database Structure

Dokumen ini merangkum struktur database berdasarkan file migration di `server/migrations`. Fokusnya adalah tabel, kolom penting, relasi, dan fungsi masing-masing tabel dalam sistem JICA Investment Monitoring System.

Migration yang dianalisis:

- `20260114083618_init_database.sql`
- `20260114110651_init_seeder.sql`
- `20260627090000_add_sla_records.sql`
- `20260627093000_add_audit_logs.sql`
- `20260627094500_add_outbox_events.sql`
- `20260627100000_add_file_metadata.sql`

---

## 1. Pola Umum Database

### 1.1 UUID Primary Key

Sebagian besar tabel memakai:

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid()
```

Extension yang digunakan:

```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
```

`pgcrypto` dipakai agar PostgreSQL dapat membuat UUID melalui `gen_random_uuid()`.

### 1.2 Timestamp dan Soft Delete

Mayoritas tabel domain memiliki kolom:

```sql
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
deleted_at TIMESTAMPTZ
```

`deleted_at` digunakan untuk soft delete. Query aplikasi umumnya memfilter `deleted_at IS NULL`.

### 1.3 Trigger `updated_at`

Migration membuat function:

```sql
update_updated_at_column()
```

Function ini dipasang sebagai trigger `BEFORE UPDATE` pada banyak tabel agar `updated_at` otomatis berubah ketika row di-update.

Tabel yang memakai trigger `updated_at`:

- `organizations`
- `users`
- `projects`
- `organization_members`
- `project_settings`
- `contracts`
- `gov_participants`
- `task_items`
- `payment_requests`
- `tickets`
- `progress_logs`
- `ticket_logs`
- `progress_validations`
- `examples`
- `sla_records`
- `files`

Tabel yang tidak memakai `updated_at` trigger:

- `audit_logs`
- `outbox_events`

---

## 2. ENUM

### 2.1 `organization_role_enum`

Nilai:

```text
GOV_SUPERVISOR
LOAN_PROVIDER
EXECUTING_AGENCY
CONTRACTOR
INDEPENDENT_AUDITOR
GOV_PARTNER
```

Fungsi:

Mendefinisikan jenis organisasi dalam sistem. Enum ini menjadi dasar role organisasi untuk authorization lintas module, seperti project, monitoring, tracker, dan finance.

### 2.2 `member_role_enum`

Nilai:

```text
ADMIN
MEMBER
```

Fungsi:

Mendefinisikan role user di dalam organisasi. `ADMIN` biasanya memiliki hak konfigurasi atau assignment, sedangkan `MEMBER` lebih terbatas.

### 2.3 `ticket_status_enum`

Nilai:

```text
OPEN
ASSIGNED
IN_PROGRESS
RESOLVED
CLOSED
```

Fungsi:

Mendefinisikan status ticket/hambatan. Status aktual ticket diturunkan dari log terbaru pada `ticket_logs`.

### 2.4 `ticket_log_action_enum`

Nilai:

```text
CREATE
ASSIGNED
COMMENTED
STATUS_CHANGED
RESOLVED
```

Fungsi:

Mendefinisikan jenis aksi yang tercatat pada riwayat ticket.

### 2.5 `payment_request_status_enum`

Nilai:

```text
SUBMITTED
AUDITOR_APPROVED
AUDITOR_REJECTED
AGENCY_APPROVED
AGENCY_REJECTED
PAID
```

Fungsi:

Mendefinisikan status workflow payment request. Status ini dipakai pada proses approval bertingkat finance.

---

## 3. Ringkasan Tabel

| Tabel | Domain | Fungsi Utama |
| --- | --- | --- |
| `organizations` | IAM / Project | Menyimpan organisasi dan role institusinya |
| `users` | IAM | Menyimpan akun pengguna |
| `organization_members` | IAM | Relasi user dengan organisasi |
| `projects` | Project | Data utama proyek |
| `project_settings` | Project | Baseline waktu, budget, dan currency proyek |
| `contracts` | Project | Kontrak utama dan mapping participant inti |
| `gov_participants` | Project | Pivot instansi pemerintah pendukung pada contract |
| `task_items` | Monitoring | WBS/task tree proyek |
| `progress_logs` | Monitoring | Laporan progress contractor |
| `progress_validations` | Monitoring | Validasi progress oleh auditor |
| `tickets` | Tracker | Ticket hambatan lapangan |
| `ticket_logs` | Tracker | Riwayat status/komentar ticket |
| `payment_requests` | Finance | Pengajuan dan approval pembayaran |
| `sla_records` | Support / Tracker | Tracking SLA untuk entity seperti ticket |
| `audit_logs` | Audit | Audit trail global |
| `outbox_events` | Event / Realtime | Event domain untuk outbox pattern |
| `files` | Support / Upload | Metadata file upload |
| `examples` | Example | Tabel contoh/dev |

---

## 4. IAM Tables

## 4.1 `organizations`

Struktur:

| Kolom | Tipe | Constraint / Catatan |
| --- | --- | --- |
| `id` | `UUID` | Primary key, default `gen_random_uuid()` |
| `name` | `TEXT` | Wajib |
| `logo_url` | `TEXT` | Opsional |
| `role` | `organization_role_enum` | Wajib |
| `created_at` | `TIMESTAMPTZ` | Default `NOW()` |
| `updated_at` | `TIMESTAMPTZ` | Auto update via trigger |
| `deleted_at` | `TIMESTAMPTZ` | Soft delete |

Fungsi:

Menyimpan data organisasi/institusi dalam sistem. Role organisasi menentukan posisi organisasi dalam workflow, misalnya `GOV_SUPERVISOR`, `LOAN_PROVIDER`, `EXECUTING_AGENCY`, `CONTRACTOR`, `INDEPENDENT_AUDITOR`, atau `GOV_PARTNER`.

Relasi utama:

- Direferensikan oleh `organization_members.organization_id`.
- Direferensikan oleh `contracts` untuk participant inti.
- Direferensikan oleh `gov_participants`.
- Direferensikan oleh `payment_requests` sebagai approval organization.
- Direferensikan oleh `tickets` dan `ticket_logs`.
- Direferensikan oleh `progress_logs.vendor_id` dan `progress_validations.auditor_id`.
- Direferensikan oleh `sla_records.owner_org_id`, `audit_logs.actor_org_id`, dan `files.uploaded_by_org_id`.

## 4.2 `users`

Struktur:

| Kolom | Tipe | Constraint / Catatan |
| --- | --- | --- |
| `id` | `UUID` | Primary key |
| `image_url` | `TEXT` | Opsional |
| `name` | `TEXT` | Wajib |
| `email` | `TEXT` | Wajib, unique |
| `password` | `TEXT` | Opsional |
| `created_at` | `TIMESTAMPTZ` | Default `NOW()` |
| `updated_at` | `TIMESTAMPTZ` | Auto update via trigger |
| `deleted_at` | `TIMESTAMPTZ` | Soft delete |

Fungsi:

Menyimpan akun pengguna. User dapat menjadi anggota satu atau lebih organisasi melalui `organization_members`.

Relasi utama:

- Direferensikan oleh `organization_members.user_id`.
- Direferensikan oleh `payment_requests` sebagai approval user.
- Direferensikan oleh `tickets.ticket_assigner_user_id`.
- Direferensikan oleh `ticket_logs.user_id`.
- Direferensikan oleh `audit_logs.actor_user_id`.
- Direferensikan oleh `files.uploaded_by_user_id`.

## 4.3 `organization_members`

Struktur:

| Kolom | Tipe | Constraint / Catatan |
| --- | --- | --- |
| `id` | `UUID` | Primary key |
| `role` | `member_role_enum` | `ADMIN` atau `MEMBER` |
| `user_id` | `UUID` | FK ke `users(id)`, `ON DELETE RESTRICT` |
| `organization_id` | `UUID` | FK ke `organizations(id)`, `ON DELETE RESTRICT` |
| `created_at` | `TIMESTAMPTZ` | Default `NOW()` |
| `updated_at` | `TIMESTAMPTZ` | Auto update via trigger |
| `deleted_at` | `TIMESTAMPTZ` | Soft delete |

Fungsi:

Menjadi pivot antara user dan organisasi. Tabel ini penting untuk authorization karena hak akses user bergantung pada organisasi tempat ia menjadi member dan role membership-nya.

Relasi utama:

- `user_id` mengarah ke `users`.
- `organization_id` mengarah ke `organizations`.

---

## 5. Project Tables

## 5.1 `projects`

Struktur:

| Kolom | Tipe | Constraint / Catatan |
| --- | --- | --- |
| `id` | `UUID` | Primary key |
| `image_urls` | `TEXT[]` | Opsional |
| `name` | `TEXT` | Wajib |
| `description` | `TEXT` | Wajib |
| `contract_file_url` | `TEXT` | Wajib |
| `created_at` | `TIMESTAMPTZ` | Default `NOW()` |
| `updated_at` | `TIMESTAMPTZ` | Auto update via trigger |
| `deleted_at` | `TIMESTAMPTZ` | Soft delete |

Fungsi:

Menyimpan data utama proyek. Project menjadi root domain untuk kontrak, setting, monitoring, tracker, finance, dan dashboard.

Relasi utama:

- Satu `projects` memiliki satu `project_settings`.
- Satu `projects` memiliki satu `contracts` karena `contracts.project_id` unique.

## 5.2 `project_settings`

Struktur:

| Kolom | Tipe | Constraint / Catatan |
| --- | --- | --- |
| `id` | `UUID` | Primary key |
| `start_date` | `TIMESTAMPTZ` | Wajib |
| `baseline_deadline` | `TIMESTAMPTZ` | Wajib |
| `total_budget` | `BIGINT` | Wajib |
| `currency` | `TEXT` | Wajib, default `IDR` |
| `project_id` | `UUID` | Wajib, unique, FK ke `projects(id)`, `ON DELETE CASCADE` |
| `created_at` | `TIMESTAMPTZ` | Default `NOW()` |
| `updated_at` | `TIMESTAMPTZ` | Auto update via trigger |
| `deleted_at` | `TIMESTAMPTZ` | Soft delete |

Fungsi:

Menyimpan konfigurasi baseline proyek, seperti tanggal mulai, deadline baseline, total budget, dan currency. Data ini dipakai untuk dashboard dan perhitungan progress/budget.

## 5.3 `contracts`

Struktur:

| Kolom | Tipe | Constraint / Catatan |
| --- | --- | --- |
| `id` | `UUID` | Primary key |
| `description` | `TEXT` | Wajib |
| `project_id` | `UUID` | Wajib, unique, FK ke `projects(id)`, `ON DELETE RESTRICT` |
| `contract_value` | `BIGINT` | Opsional |
| `contract_doc_url` | `TEXT` | Opsional |
| `gov_supervisor_participant_id` | `UUID` | Wajib, FK ke `organizations(id)` |
| `loan_provider_participant_id` | `UUID` | Wajib, FK ke `organizations(id)` |
| `executing_agency_participant_id` | `UUID` | Opsional, FK ke `organizations(id)` |
| `vendor_participant_id` | `UUID` | Opsional, FK ke `organizations(id)` |
| `vendor_participant_selected_at` | `TIMESTAMPTZ` | Opsional |
| `auditor_participant_id` | `UUID` | Opsional, FK ke `organizations(id)` |
| `auditor_participant_selected_at` | `TIMESTAMPTZ` | Opsional |
| `created_at` | `TIMESTAMPTZ` | Default `NOW()` |
| `updated_at` | `TIMESTAMPTZ` | Auto update via trigger |
| `deleted_at` | `TIMESTAMPTZ` | Soft delete |

Fungsi:

Menyimpan kontrak utama proyek dan mapping participant inti. Tabel ini menjadi pusat authorization project karena menentukan organisasi mana yang berperan sebagai government supervisor, loan provider, executing agency, vendor/contractor, dan auditor.

Relasi utama:

- `project_id` mengarah ke `projects`.
- Participant id mengarah ke `organizations`.
- Direferensikan oleh `task_items`, `tickets`, `payment_requests`, dan `gov_participants`.

Catatan:

- `project_id` unique, sehingga satu project hanya memiliki satu contract utama dalam skema ini.

## 5.4 `gov_participants`

Struktur:

| Kolom | Tipe | Constraint / Catatan |
| --- | --- | --- |
| `id` | `UUID` | Primary key |
| `contract_id` | `UUID` | Wajib, FK ke `contracts(id)`, `ON DELETE CASCADE` |
| `gov_participant_id` | `UUID` | Wajib, FK ke `organizations(id)`, `ON DELETE RESTRICT` |
| `created_at` | `TIMESTAMPTZ` | Default `NOW()` |
| `updated_at` | `TIMESTAMPTZ` | Auto update via trigger |
| `deleted_at` | `TIMESTAMPTZ` | Soft delete |

Fungsi:

Pivot untuk organisasi `GOV_PARTNER` yang terlibat dalam contract. Contohnya instansi pendukung seperti pertanahan, utilitas, atau transportasi.

---

## 6. Monitoring Tables

## 6.1 `task_items`

Struktur:

| Kolom | Tipe | Constraint / Catatan |
| --- | --- | --- |
| `id` | `UUID` | Primary key |
| `title` | `TEXT` | Wajib |
| `description` | `TEXT` | Wajib |
| `deadline` | `TIMESTAMPTZ` | Wajib |
| `weight` | `INT` | Wajib |
| `parent_id` | `UUID` | FK ke `task_items(id)`, `ON DELETE SET NULL` |
| `contract_id` | `UUID` | Wajib, FK ke `contracts(id)`, `ON DELETE CASCADE` |
| `created_at` | `TIMESTAMPTZ` | Default `NOW()` |
| `updated_at` | `TIMESTAMPTZ` | Auto update via trigger |
| `deleted_at` | `TIMESTAMPTZ` | Soft delete |

Fungsi:

Menyimpan Work Breakdown Structure (WBS) dalam bentuk tree. `parent_id` membuat task dapat memiliki parent-child. `weight` digunakan sebagai bobot progress.

Relasi utama:

- `contract_id` mengarah ke `contracts`.
- `parent_id` mengarah ke `task_items` itu sendiri.
- Direferensikan oleh `progress_logs.task_item_id`.
- Direferensikan oleh `tickets.task_item_id`.

## 6.2 `progress_logs`

Struktur:

| Kolom | Tipe | Constraint / Catatan |
| --- | --- | --- |
| `id` | `UUID` | Primary key |
| `report` | `TEXT` | Wajib |
| `progress_percentage` | `INT` | Wajib |
| `evidence_image_urls` | `TEXT[]` | Opsional |
| `snapshot_title` | `TEXT` | Wajib |
| `snapshot_description` | `TEXT` | Wajib |
| `snapshot_weight` | `INT` | Wajib |
| `task_item_id` | `UUID` | Wajib, FK ke `task_items(id)`, `ON DELETE RESTRICT` |
| `vendor_id` | `UUID` | Wajib, FK ke `organizations(id)` |
| `created_at` | `TIMESTAMPTZ` | Default `NOW()` |
| `updated_at` | `TIMESTAMPTZ` | Auto update via trigger |
| `deleted_at` | `TIMESTAMPTZ` | Soft delete |

Fungsi:

Menyimpan laporan progress dari contractor/vendor. Kolom snapshot menyimpan salinan title, description, dan weight task saat progress dilaporkan, sehingga riwayat tetap stabil walaupun task berubah.

Relasi utama:

- `task_item_id` mengarah ke `task_items`.
- `vendor_id` mengarah ke `organizations`.
- Direferensikan oleh `progress_validations.progress_log_id`.

## 6.3 `progress_validations`

Struktur:

| Kolom | Tipe | Constraint / Catatan |
| --- | --- | --- |
| `id` | `UUID` | Primary key |
| `is_accepted` | `BOOLEAN` | Wajib |
| `comment` | `TEXT` | Opsional |
| `real_progress_percentage` | `INT` | Wajib |
| `progress_log_id` | `UUID` | Wajib, unique, FK ke `progress_logs(id)`, `ON DELETE CASCADE` |
| `auditor_id` | `UUID` | Wajib, FK ke `organizations(id)` |
| `created_at` | `TIMESTAMPTZ` | Default `NOW()` |
| `updated_at` | `TIMESTAMPTZ` | Auto update via trigger |
| `deleted_at` | `TIMESTAMPTZ` | Soft delete |

Fungsi:

Menyimpan hasil validasi auditor terhadap progress log. Satu progress log hanya dapat memiliki satu validation karena `progress_log_id` unique.

Relasi utama:

- `progress_log_id` mengarah ke `progress_logs`.
- `auditor_id` mengarah ke `organizations`.

---

## 7. Tracker Tables

## 7.1 `tickets`

Struktur:

| Kolom | Tipe | Constraint / Catatan |
| --- | --- | --- |
| `id` | `UUID` | Primary key |
| `title` | `TEXT` | Wajib |
| `description` | `TEXT` | Wajib |
| `location` | `TEXT` | Wajib |
| `evidence_image_urls` | `TEXT[]` | Opsional |
| `task_item_id` | `UUID` | Opsional, FK ke `task_items(id)`, `ON DELETE SET NULL` |
| `contract_id` | `UUID` | Wajib, FK ke `contracts(id)`, `ON DELETE RESTRICT` |
| `ticket_assigner_contractor_id` | `UUID` | Wajib, FK ke `organizations(id)` |
| `ticket_assigner_user_id` | `UUID` | Wajib, FK ke `users(id)` |
| `ticket_assign_to_id` | `UUID` | Opsional, FK ke `organizations(id)` |
| `created_at` | `TIMESTAMPTZ` | Default `NOW()` |
| `updated_at` | `TIMESTAMPTZ` | Auto update via trigger |
| `deleted_at` | `TIMESTAMPTZ` | Soft delete |

Fungsi:

Menyimpan ticket hambatan lapangan. Ticket dapat dikaitkan dengan task tertentu dan contract tertentu. Status ticket tidak disimpan sebagai kolom utama, tetapi diturunkan dari log terbaru pada `ticket_logs`.

Relasi utama:

- `contract_id` mengarah ke `contracts`.
- `task_item_id` mengarah ke `task_items`.
- `ticket_assigner_contractor_id` dan `ticket_assign_to_id` mengarah ke `organizations`.
- `ticket_assigner_user_id` mengarah ke `users`.
- Direferensikan oleh `ticket_logs.ticket_id`.

## 7.2 `ticket_logs`

Struktur:

| Kolom | Tipe | Constraint / Catatan |
| --- | --- | --- |
| `id` | `UUID` | Primary key |
| `action` | `ticket_log_action_enum` | Wajib |
| `description` | `TEXT` | Wajib |
| `attachment_url` | `TEXT` | Opsional |
| `previous_status` | `ticket_status_enum` | Opsional |
| `new_status` | `ticket_status_enum` | Wajib |
| `ticket_id` | `UUID` | Wajib, FK ke `tickets(id)`, `ON DELETE CASCADE` |
| `organization_id` | `UUID` | Wajib, FK ke `organizations(id)` |
| `user_id` | `UUID` | Wajib, FK ke `users(id)` |
| `created_at` | `TIMESTAMPTZ` | Default `NOW()` |
| `updated_at` | `TIMESTAMPTZ` | Auto update via trigger |
| `deleted_at` | `TIMESTAMPTZ` | Soft delete |

Fungsi:

Menyimpan riwayat ticket. Current status ticket dihitung dari `new_status` pada log terbaru. Model ini membuat histori perubahan status tetap tersedia.

Relasi utama:

- `ticket_id` mengarah ke `tickets`.
- `organization_id` mengarah ke `organizations`.
- `user_id` mengarah ke `users`.

---

## 8. Finance Tables

## 8.1 `payment_requests`

Struktur:

| Kolom | Tipe | Constraint / Catatan |
| --- | --- | --- |
| `id` | `UUID` | Primary key |
| `title` | `TEXT` | Wajib |
| `amount` | `BIGINT` | Wajib |
| `status` | `payment_request_status_enum` | Wajib |
| `invoice_file_url` | `TEXT` | Wajib |
| `contract_id` | `UUID` | Wajib, FK ke `contracts(id)`, `ON DELETE RESTRICT` |
| `auditor_org_id` | `UUID` | Opsional, FK ke `organizations(id)` |
| `auditor_user_id` | `UUID` | Opsional, FK ke `users(id)` |
| `auditor_at` | `TIMESTAMPTZ` | Opsional |
| `executing_agency_org_id` | `UUID` | Opsional, FK ke `organizations(id)` |
| `executing_agency_user_id` | `UUID` | Opsional, FK ke `users(id)` |
| `executing_agency_at` | `TIMESTAMPTZ` | Opsional |
| `loan_provider_org_id` | `UUID` | Opsional, FK ke `organizations(id)` |
| `loan_provider_user_id` | `UUID` | Opsional, FK ke `users(id)` |
| `loan_provider_at` | `TIMESTAMPTZ` | Opsional |
| `payment_proof_image_urls` | `TEXT[]` | Wajib, default `{}` |
| `created_at` | `TIMESTAMPTZ` | Default `NOW()` |
| `updated_at` | `TIMESTAMPTZ` | Auto update via trigger |
| `deleted_at` | `TIMESTAMPTZ` | Soft delete |

Fungsi:

Menyimpan pengajuan pembayaran dan approval chain. Status mengikuti workflow finance: submitted, auditor approved/rejected, agency approved/rejected, dan paid.

Relasi utama:

- `contract_id` mengarah ke `contracts`.
- Approval org id mengarah ke `organizations`.
- Approval user id mengarah ke `users`.

---

## 9. SLA, Audit, Outbox, dan File Tables

## 9.1 `sla_records`

Struktur:

| Kolom | Tipe | Constraint / Catatan |
| --- | --- | --- |
| `id` | `UUID` | Primary key |
| `entity_type` | `TEXT` | Wajib |
| `entity_id` | `UUID` | Wajib |
| `state` | `TEXT` | Wajib |
| `owner_org_id` | `UUID` | FK ke `organizations(id)`, `ON DELETE RESTRICT` |
| `entered_at` | `TIMESTAMPTZ` | Wajib, default `NOW()` |
| `due_at` | `TIMESTAMPTZ` | Wajib |
| `resolved_at` | `TIMESTAMPTZ` | Opsional |
| `status` | `TEXT` | Wajib, default `ON_TRACK` |
| `created_at` | `TIMESTAMPTZ` | Default `NOW()` |
| `updated_at` | `TIMESTAMPTZ` | Auto update via trigger |
| `deleted_at` | `TIMESTAMPTZ` | Soft delete |

Constraint:

- `status IN ('ON_TRACK', 'SLA_WARNING', 'SLA_VIOLATED', 'RESOLVED')`
- Unique `(entity_type, entity_id)`

Index:

- `idx_sla_records_entity` pada `(entity_type, entity_id)` untuk row aktif.
- `idx_sla_records_due_at` pada `due_at` untuk SLA aktif yang belum resolved.
- `idx_sla_records_owner` pada `owner_org_id`.

Fungsi:

Menyimpan SLA tracking untuk entity tertentu, terutama ticket. Tabel ini menyimpan state aktif, owner organization, waktu masuk state, deadline, status SLA, dan resolved time.

## 9.2 `audit_logs`

Struktur:

| Kolom | Tipe | Constraint / Catatan |
| --- | --- | --- |
| `id` | `UUID` | Primary key |
| `request_id` | `TEXT` | Opsional |
| `actor_user_id` | `UUID` | FK ke `users(id)`, `ON DELETE SET NULL` |
| `actor_org_id` | `UUID` | FK ke `organizations(id)`, `ON DELETE SET NULL` |
| `action` | `TEXT` | Wajib |
| `entity_type` | `TEXT` | Wajib |
| `entity_id` | `UUID` | Opsional |
| `before_data` | `JSONB` | Wajib, default `{}` |
| `after_data` | `JSONB` | Wajib, default `{}` |
| `ip_address` | `TEXT` | Opsional |
| `user_agent` | `TEXT` | Opsional |
| `created_at` | `TIMESTAMPTZ` | Default `NOW()` |

Index:

- `idx_audit_logs_actor_user`
- `idx_audit_logs_actor_org`
- `idx_audit_logs_entity`
- `idx_audit_logs_action`
- `idx_audit_logs_request_id`

Fungsi:

Menyimpan audit trail global untuk aksi penting. Tabel ini mendukung accountability karena mencatat actor, organisasi, action, entity, before/after data, request id, IP, user agent, dan waktu kejadian.

Catatan:

- Tabel ini tidak memiliki `updated_at` atau `deleted_at`, sehingga secara desain lebih append-oriented.

## 9.3 `outbox_events`

Struktur:

| Kolom | Tipe | Constraint / Catatan |
| --- | --- | --- |
| `id` | `UUID` | Primary key |
| `event_type` | `TEXT` | Wajib |
| `aggregate_type` | `TEXT` | Wajib |
| `aggregate_id` | `UUID` | Wajib |
| `payload` | `JSONB` | Wajib, default `{}` |
| `status` | `TEXT` | Wajib, default `PENDING` |
| `created_at` | `TIMESTAMPTZ` | Default `NOW()` |
| `processed_at` | `TIMESTAMPTZ` | Opsional |

Constraint:

- `status IN ('PENDING', 'PROCESSING', 'PROCESSED', 'FAILED')`

Index:

- `idx_outbox_events_pending` untuk event pending.
- `idx_outbox_events_aggregate` untuk riwayat event per aggregate.
- `idx_outbox_events_status` untuk pemrosesan berdasarkan status.

Fungsi:

Menyimpan domain event dengan outbox pattern. Service domain menulis event ke tabel ini dalam transaksi yang sama dengan perubahan data utama, lalu worker memprosesnya untuk realtime delivery.

## 9.4 `files`

Struktur:

| Kolom | Tipe | Constraint / Catatan |
| --- | --- | --- |
| `id` | `UUID` | Primary key |
| `public_id` | `TEXT` | Wajib, unique |
| `url` | `TEXT` | Wajib |
| `file_name` | `TEXT` | Wajib |
| `mime_type` | `TEXT` | Wajib |
| `size` | `BIGINT` | Wajib |
| `checksum` | `TEXT` | Wajib |
| `uploaded_by_user_id` | `UUID` | Wajib, FK ke `users(id)`, `ON DELETE RESTRICT` |
| `uploaded_by_org_id` | `UUID` | FK ke `organizations(id)`, `ON DELETE SET NULL` |
| `entity_type` | `TEXT` | Opsional |
| `entity_id` | `UUID` | Opsional |
| `created_at` | `TIMESTAMPTZ` | Default `NOW()` |
| `updated_at` | `TIMESTAMPTZ` | Auto update via trigger |
| `deleted_at` | `TIMESTAMPTZ` | Soft delete |

Constraint:

- `size > 0`
- `entity_type` dan `entity_id` harus sama-sama null atau sama-sama terisi.

Index:

- `idx_files_uploaded_by_user`
- `idx_files_uploaded_by_org`
- `idx_files_entity`
- `idx_files_checksum`

Fungsi:

Menyimpan metadata file upload. Tabel ini membuat file tidak hanya disimpan sebagai URL, tetapi juga memiliki metadata seperti nama file, MIME type, size, checksum, uploader, organization, dan optional binding ke entity bisnis.

---

## 10. Example Table

## 10.1 `examples`

Struktur:

| Kolom | Tipe | Constraint / Catatan |
| --- | --- | --- |
| `id` | `UUID` | Primary key |
| `name` | `TEXT` | Wajib |
| `price` | `BIGINT` | Wajib |
| `created_at` | `TIMESTAMPTZ` | Default `NOW()` |
| `updated_at` | `TIMESTAMPTZ` | Auto update via trigger |
| `deleted_at` | `TIMESTAMPTZ` | Soft delete |

Fungsi:

Tabel contoh/development. Tidak menjadi bagian inti domain JICA Investment Monitoring System.

---

## 11. Relasi Domain Utama

### 11.1 IAM

```text
users
  -> organization_members
      -> organizations
```

User mendapatkan konteks akses melalui membership pada organisasi.

### 11.2 Project dan Contract

```text
projects
  -> project_settings
  -> contracts
      -> organizations sebagai participant inti
      -> gov_participants
          -> organizations sebagai gov partner
```

Project memiliki setting dan contract. Contract menentukan participant utama dan government partner.

### 11.3 Monitoring

```text
contracts
  -> task_items
      -> progress_logs
          -> progress_validations
```

WBS berada pada `task_items`. Contractor membuat `progress_logs`, auditor membuat `progress_validations`.

### 11.4 Tracker

```text
contracts
  -> tickets
      -> ticket_logs
      -> sla_records (entity_type/entity_id)
```

Ticket menyimpan hambatan. Status ticket diturunkan dari `ticket_logs`. SLA ticket disimpan secara generic di `sla_records`.

### 11.5 Finance

```text
contracts
  -> payment_requests
      -> approval org/user chain
```

Payment request mengikuti workflow approval bertingkat dari auditor, executing agency, hingga loan provider.

### 11.6 Audit dan Realtime

```text
domain action
  -> audit_logs
  -> outbox_events
      -> realtime worker / WebSocket
```

Action penting dapat dicatat pada audit log dan menghasilkan outbox event untuk delivery realtime.

### 11.7 File Metadata

```text
users / organizations
  -> files
      -> optional entity_type + entity_id
```

File metadata dapat dikaitkan ke entity bisnis melalui pasangan `entity_type` dan `entity_id`.

---

## 12. Seeder

Migration `20260114110651_init_seeder.sql` menambahkan data awal untuk development/demo:

### 12.1 Organisasi inti

- Bappenas sebagai `GOV_SUPERVISOR`
- JICA sebagai `LOAN_PROVIDER`
- PT MRT Jakarta sebagai `EXECUTING_AGENCY`

### 12.2 Contractor

- Shimizu - Adhi Karya JO
- Sumitomo Mitsui Construction Co.
- PT Waskita Karya (Persero) Tbk

### 12.3 Independent Auditor

- Nippon Koei Co., Ltd.
- Oriental Consultants Global
- PT Virama Karya (Persero)

### 12.4 Government Partner

- Kantor Pertanahan (BPN) Jakarta Pusat
- PLN UID Jakarta Raya
- Dinas Perhubungan DKI Jakarta

Setiap organisasi seed memiliki user admin dan row membership `ADMIN`.

---

## 13. Catatan Desain Penting

### 13.1 Status Ticket Berdasarkan Log

Tabel `tickets` tidak menyimpan `status` langsung. Status aktual dihitung dari `ticket_logs.new_status` terbaru. Ini membuat histori status tetap tersedia.

### 13.2 Progress Menggunakan Snapshot

`progress_logs` menyimpan snapshot task (`snapshot_title`, `snapshot_description`, `snapshot_weight`) agar laporan historis tetap stabil walaupun task berubah setelah laporan dibuat.

### 13.3 Contract sebagai Pusat Authorization Project

Tabel `contracts` menyimpan participant inti project. Banyak policy akses project bergantung pada mapping organisasi di tabel ini.

### 13.4 SLA Generic

`sla_records` memakai `entity_type` dan `entity_id`, sehingga secara desain bisa dipakai untuk entity selain ticket. Saat ini fungsi utamanya adalah SLA ticket.

### 13.5 Outbox untuk Reliability Event

`outbox_events` memisahkan transaksi domain dari pengiriman event realtime. Ini mengurangi risiko event hilang ketika publish realtime gagal setelah data domain tersimpan.

### 13.6 File Binding Generic

`files` juga memakai pola generic binding melalui `entity_type` dan `entity_id`, sehingga metadata file bisa dikaitkan ke progress, ticket, payment, atau entity lain.
