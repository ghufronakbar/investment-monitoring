# Next Plan: Improvisasi Backend Sistem

Dokumen ini merangkum code/fitur yang belum ada dan disarankan untuk dibuat agar sistem naik kelas secara teknis. Fokusnya adalah peningkatan backend yang relevan dengan jurnal Informatika: FSM, SLA Tracking, RBAC, auditability, data integrity, testing, dan observability.

---

## 1. Centralized Policy Layer

### Status Implementasi

Sudah dibuat package `internal/policy` sebagai pusat pengecekan akses untuk resource project-scoped. Controller Project, Monitoring, Tracker, Finance, dan Support Dashboard sekarang memakai policy layer, bukan lagi melakukan `GetUserProjectAccess` dan perbandingan role secara manual di masing-masing handler.

### Yang Sudah Dibuat

Package baru:

```text
internal/policy/
  policy.go
```

Fungsi utama yang tersedia:

```go
CanCreateProject(ctx, userID) (orgID, error)
CanViewProject(ctx, userID, projectID) error
CanConfigureProject(ctx, userID, projectID) error
CanAssignExecutingAgency(ctx, userID, projectID) error
CanManageProjectContract(ctx, userID, projectID) error
CanSubmitProgress(ctx, userID, taskID) (*ProjectAccess, error)
CanValidateProgress(ctx, userID, logID) (*ProjectAccess, error)
CanAssignTicket(ctx, userID, ticketID) (*ProjectAccess, error)
CanAddTicketLog(ctx, userID, ticketID) (*ProjectAccess, error)
CanApprovePaymentAsAuditor(ctx, userID, requestID) (*ProjectAccess, error)
CanApprovePaymentAsAgency(ctx, userID, requestID) (*ProjectAccess, error)
CanDisbursePayment(ctx, userID, requestID) (*ProjectAccess, error)
```

Controller yang sudah direfactor:

- `internal/module/project/controller`
- `internal/module/monitoring/controller`
- `internal/module/tracker/controller`
- `internal/module/finance/controller`
- `internal/module/support/dashboard/controller`

### Manfaat

- Authorization lebih konsisten.
- Controller menjadi lebih tipis.
- Lebih mudah dites.
- Cocok untuk pembahasan `Organization-Centric RBAC` di jurnal.

### Sisa Peningkatan

- Tambahkan unit test untuk `internal/policy`.
- Pecah `policy.go` menjadi beberapa file jika logic bertambah besar.
- Tambahkan policy untuk IAM organization jika ingin seluruh access control benar-benar berada di satu layer.

---

## 2. FSM Transition Engine

### Status Implementasi

Sudah dibuat package `internal/workflow` sebagai FSM transition engine untuk workflow utama. Service Tracker, Finance, dan Monitoring sekarang memakai FSM untuk menentukan atau memvalidasi next state.

### Yang Sudah Dibuat

Package:

```text
internal/workflow/
  fsm.go
  errors.go
  finance.go
  tracker.go
  monitoring.go
  finance_test.go
  tracker_test.go
  monitoring_test.go
```

Core FSM:

```go
type Transition struct {
    From  State
    Input Input
    To    State
}

func (m Machine) Next(current State, input Input) (State, bool)
```

FSM yang sudah diformalisasi:

- Tracker: `OPEN -> ASSIGNED -> IN_PROGRESS -> RESOLVED -> CLOSED`
- Finance: `SUBMITTED -> AUDITOR_APPROVED/REJECTED -> AGENCY_APPROVED/REJECTED -> PAID`
- Monitoring: `PENDING_VALIDATION -> ACCEPTED / REJECTED`

Integrasi service:

- `internal/module/tracker/service`: assign, status change/comment, close ticket.
- `internal/module/finance/service`: auditor approval, agency approval, disbursement.
- `internal/module/monitoring/service`: progress validation accepted/rejected.

Test yang sudah ada:

- Finance tidak bisa skip approval stage.
- Ticket `CLOSED` menjadi terminal state.
- Monitoring menolak real progress yang lebih tinggi dari claimed progress.

### Manfaat

- Workflow menjadi eksplisit.
- Invalid transition lebih mudah dicegah.
- Bisa dibuat transition matrix untuk jurnal.
- Testing lebih jelas karena setiap state dan input bisa diuji.

### Sisa Peningkatan

- Tambahkan role/actor guard langsung ke FSM jika logic makin kompleks.
- Tambahkan audit trail setiap transition.
- Tambahkan diagram transition ke dokumen jurnal.

---

## 3. SLA Tracking untuk Ticket

### Status Implementasi

Sudah dibuat baseline SLA tracking untuk ticket. SLA disimpan terpisah dari `tickets` agar modelnya bisa dipakai ulang untuk entity lain seperti progress, payment, permit, atau approval workflow.

### Yang Sudah Dibuat

Migration baru:

```text
migrations/20260627090000_add_sla_records.sql
```

Table baru:

```text
sla_records
- id
- entity_type
- entity_id
- state
- owner_org_id
- entered_at
- due_at
- resolved_at
- status
- created_at
- updated_at
- deleted_at
```

Query sqlc:

```text
internal/repository/queries/sla.sql
- UpsertSLARecord
- ResolveSLARecord
- GetSLARecordByEntity
- TouchSLAStatus
```

Service baru:

```text
internal/module/support/sla/
  service.go
  return.go
```

Integrasi tracker:

- `CreateTicket` membuat SLA awal untuk state `OPEN`.
- `AssignTicket` memindahkan SLA ke state `ASSIGNED` dan owner instansi tujuan.
- `AddLog` memperbarui SLA saat status berubah dan tetap atomik dengan `ticket_logs`.
- `CloseTicket` menandai SLA selesai melalui `resolved_at`.
- Response ticket sekarang menyertakan field `sla`.

Status SLA dihitung dari `due_at`, `resolved_at`, dan warning threshold:

```text
RESOLVED      jika ticket selesai sebelum due_at
SLA_WARNING   jika sisa waktu <= warning threshold
SLA_VIOLATED  jika now/resolved_at melewati due_at
ON_TRACK      jika masih dalam batas normal
```

### Manfaat

- Red tape bisa diukur dari durasi status.
- Dashboard dapat menampilkan bottleneck per instansi.
- Cocok dengan niche jurnal `SLA Tracking`.

### Sisa Peningkatan

- Tambahkan endpoint/dashboard khusus SLA agar bottleneck per organisasi terlihat eksplisit.
- Tambahkan konfigurasi SLA per state atau per jenis ticket, bukan fixed 7 hari.
- Tambahkan scheduler/worker untuk update status dan mengirim notifikasi warning/escalation.
- Tambahkan test service SLA dengan clock injection agar kasus warning, violated, dan resolved bisa diuji deterministik.

---

## 4. Audit Log Global

### Status Implementasi

Sudah dibuat baseline audit log global untuk mencatat perubahan penting lintas module. Audit global ini melengkapi riwayat spesifik yang sudah ada:

- `ticket_logs` untuk ticket.
- `progress_logs` untuk laporan progress.
- approval timestamp di `payment_requests`.

Audit global sekarang menyimpan actor, organisasi, action, entity, snapshot data sebelum/sesudah, dan metadata request.

### Yang Sudah Dibuat

Migration baru:

```text
migrations/20260627093000_add_audit_logs.sql
```

Table:

```text
audit_logs
- id
- request_id
- actor_user_id
- actor_org_id
- action
- entity_type
- entity_id
- before_data JSONB
- after_data JSONB
- ip_address
- user_agent
- created_at
```

Package:

```text
internal/audit/
  middleware.go
  service.go
  types.go
```

Query sqlc:

```text
internal/repository/queries/audit.sql
- CreateAuditLog
- ListAuditLogsByEntity
- ListAuditLogsByActor
```

Middleware:

- `audit.Middleware` dipasang di root router setelah `chi` request ID.
- Metadata yang disimpan: `request_id`, `ip_address`, dan `user_agent`.

Integrasi service:

- create project;
- configure settings;
- assign executing agency;
- assign vendor;
- assign auditor;
- assign gov partners;
- submit progress;
- validate progress;
- create/assign/close ticket;
- update ticket status;
- create payment request;
- approve payment auditor;
- approve payment executing agency;
- disburse payment;
- create organization;
- invite organization member.

Audit ditulis melalui:

```go
Record(ctx, params)
RecordTx(ctx, q, params)
```

Untuk flow yang sudah memakai transaksi, audit dicatat dalam transaksi yang sama agar perubahan bisnis dan audit tidak berbeda state.

### Manfaat

- Sistem lebih accountable.
- Cocok untuk proyek lintas lembaga dan pendanaan internasional.
- Memperkuat pembahasan auditability di jurnal.

### Sisa Peningkatan

- Tambahkan endpoint admin untuk membaca audit log per entity, actor, project, atau request id.
- Tambahkan audit policy: siapa yang boleh melihat audit log.
- Tambahkan correlation id antar event jika nanti memakai outbox/event-driven architecture.
- Tambahkan redaction untuk field sensitif sebelum masuk `before_data` dan `after_data`.

---

## 5. Finance Concurrency Control

### Status Implementasi

`CreatePaymentRequest` sudah diperkuat dengan transaction boundary dan row-level lock pada contract. Risiko dua request paralel membaca available amount yang sama sudah dikurangi karena perhitungan paid/pending dan insert payment request sekarang berjalan setelah `contracts` terkunci.

### Yang Sudah Dibuat

Query sqlc baru:

```sql
-- name: GetContractByIDForUpdate :one
SELECT * FROM contracts WHERE id = $1 AND deleted_at IS NULL FOR UPDATE;
```

Integrasi service:

1. Lookup contract dari project.
2. Mulai transaction.
3. Lock contract row dengan `GetContractByIDForUpdate`.
4. Validasi contractor participant dan contract value saat lock aktif.
5. Hitung `hasPending`, paid amount, pending amount, dan available amount dalam transaction yang sama.
6. Validasi amount.
7. Insert payment request.
8. Tulis audit log `CREATE_PAYMENT_REQUEST` dalam transaction yang sama.
9. Commit.

### Manfaat

- Mencegah double spending contract value.
- Membuat workflow finance lebih aman.
- Ini improvement senior-level karena menyentuh consistency dan race condition.

### Sisa Peningkatan

- Tambahkan integration/concurrency test dengan dua goroutine yang submit payment request ke contract yang sama.
- Pertimbangkan lock/read consistency tambahan untuk flow yang mengubah status finance jika nanti ada reversal atau cancel payment.
- Tambahkan metric/log untuk lock wait time jika volume transaksi meningkat.

---

## 6. Refresh Token Revocation

### Status Implementasi

Refresh token revocation sudah dibuat dengan Redis sebagai storage blacklist. Access token tetap stateless JWT, sedangkan refresh token sekarang memiliki `jti`, `iat`, dan `exp` sehingga bisa direvoke secara server-side.

### Yang Sudah Dibuat

Refresh token claims:

```text
jti        -> jwt.RegisteredClaims.ID
iat        -> jwt.RegisteredClaims.IssuedAt
exp        -> jwt.RegisteredClaims.ExpiresAt
user id    -> custom claim id
email      -> custom claim email
```

Redis keys:

```text
{REDIS_PREFIX}:blacklist:refresh:{jti} -> TTL sampai refresh token expire
{REDIS_PREFIX}:revoke-all:refresh:{user_id} -> cutoff unix nano dengan TTL refresh token
```

Endpoint:

```text
POST /iam/auth/logout
POST /iam/auth/revoke-all
```

Behavior:

```text
login:
  generate refresh token dengan jti

refresh-token:
  validate signature/expiry
  cek jti blacklist
  cek revoke-all cutoff user
  generate access token baru
  generate refresh token baru
  blacklist refresh token lama

logout:
  validate refresh token
  blacklist jti sampai token expire

revoke-all:
  validate refresh token
  set revoke-all cutoff untuk user
  blacklist refresh token yang dipakai
```

Redis connection dibuat optional di `cmd/api/main.go`: jika Redis tidak tersedia, server tetap berjalan dan refresh token biasa tetap bisa dipakai, tetapi endpoint logout/revoke-all membutuhkan Redis.

### Manfaat

- Logout benar-benar memutus sesi.
- Token bocor bisa direvoke.
- Meningkatkan security posture sistem.

### Sisa Peningkatan

- Tambahkan device/session list jika ingin user bisa revoke session tertentu dari UI.
- Tambahkan hashing token jika suatu saat refresh session disimpan di PostgreSQL.
- Tambahkan test untuk token rotation, logout, revoke-all, dan fallback saat Redis tidak tersedia.

---

## 7. Rate Limiting Login dan Auth Endpoint

### Status Implementasi

Sudah dibuat rate limiting untuk endpoint auth yang sensitif. Middleware memakai Redis saat Redis tersedia, dan otomatis fallback ke in-memory counter saat Redis tidak tersedia atau error, sehingga server tetap bisa berjalan di local development.

### Yang Sudah Dibuat

Middleware:

```text
internal/http/internal_middleware/rate_limit.go
```

Endpoint yang perlu dilindungi:

- `/iam/auth/check-email`
- `/iam/auth/login`
- `/iam/auth/refresh-token`

Storage:

- Redis key jika Redis tersedia.
- In-memory map sebagai fallback lokal.

Rule default:

```text
max 5 request per action + IP/email per 5 minutes
```

Key Redis:

```text
{REDIS_PREFIX}:rate-limit:auth:{action}:{sha256(identity)}
```

Identity:

- Untuk request yang punya `email`, limiter memakai kombinasi IP + email.
- Untuk `refresh-token`, limiter memakai IP karena payload tidak punya email.

Response saat limit terlampaui:

```text
HTTP 429
RATE_LIMIT_EXCEEDED
Retry-After: seconds
```

Test:

```text
internal/http/internal_middleware/rate_limit_test.go
```

### Manfaat

- Mengurangi risiko brute force.
- Redis yang sebelumnya belum digunakan bisa punya fungsi jelas.
- Cocok untuk pembahasan security engineering.

### Sisa Peningkatan

- Tambahkan konfigurasi limit/window via env.
- Tambahkan limit berbeda untuk `login`, `check-email`, dan `refresh-token`.
- Tambahkan observability untuk jumlah request yang diblokir.

---

## 8. Dashboard Cache

### Status Implementasi

Dashboard S-Curve dan stats sekarang memakai Redis read-through cache. Jika Redis tidak tersedia, service otomatis menghitung dari database seperti sebelumnya.

### Yang Sudah Dibuat

Package cache:

```text
internal/module/support/dashboard/cache
```

Cache key:

```text
{REDIS_PREFIX}:dashboard:scurve:{projectId}
{REDIS_PREFIX}:dashboard:stats:{projectId}
```

TTL:

```text
3 menit
```

Integrasi:

- `DashboardService.GetSCurve` membaca cache dulu, lalu fallback ke query database jika cache miss.
- `DashboardService.GetStats` membaca cache dulu, lalu fallback ke query database jika cache miss.
- Cache write dilakukan best-effort. Error Redis tidak menggagalkan response dashboard.

Invalidasi cache sudah dipasang pada:

- progress divalidasi;
- ticket status berubah;
- payment disburse;
- WBS diubah.
- project settings diubah.

### Manfaat

- Dashboard lebih cepat.
- Redis punya fungsi yang nyata.
- Mendukung klaim scalability.

### Sisa Peningkatan

- Tambahkan cache hit/miss metric.
- Tambahkan konfigurasi TTL via env.
- Tambahkan invalidasi berbasis domain event/outbox agar invalidasi tidak tersebar di banyak service.

---

## 9. Domain Event dan Outbox Pattern

### Status Saat Ini

Beberapa action penting terjadi, tetapi belum menghasilkan event formal. Email invitation saat ini dikirim via goroutine langsung setelah transaksi, belum memakai queue/outbox.

### Yang Perlu Dibuat

Migration:

```text
outbox_events
- id
- event_type
- aggregate_type
- aggregate_id
- payload JSONB
- status
- created_at
- processed_at
```

Event awal:

- `ProgressSubmitted`
- `ProgressValidated`
- `TicketCreated`
- `TicketAssigned`
- `TicketResolved`
- `PaymentApproved`
- `PaymentPaid`
- `MemberInvited`

Worker:

```text
cmd/worker/main.go
internal/outbox/
```

### Manfaat

- Fondasi notification system.
- Lebih reliable dibanding goroutine langsung.
- Membuka jalan untuk real-time update, email, dan alert SLA.

---

## 10. Notification System

### Status Saat Ini

Email invitation sudah ada, tetapi belum ada sistem notifikasi workflow untuk progress, ticket, finance, atau SLA.

### Yang Perlu Dibuat

Table:

```text
notifications
- id
- user_id
- organization_id
- title
- message
- type
- read_at
- created_at
```

Service:

```text
internal/module/support/notification/
```

Trigger dari event:

- ticket assigned;
- ticket SLA warning;
- progress waiting validation;
- payment waiting approval;
- payment paid.

### Manfaat

- Workflow tidak bergantung pada user membuka dashboard.
- SLA escalation bisa berjalan.
- Meningkatkan operational usefulness sistem.

---

## 11. OpenAPI Specification

### Status Saat Ini

Dokumentasi route ada di Markdown, tetapi beberapa route tidak sinkron dengan implementasi aktual.

### Yang Perlu Dibuat

File:

```text
docs/openapi.yaml
```

Isi:

- semua endpoint aktual;
- request body;
- response schema;
- error schema;
- security scheme JWT bearer;
- enum role/status.

Opsional:

- generate TypeScript client untuk frontend.
- contract test untuk memastikan route tidak berubah tanpa update docs.

### Manfaat

- API menjadi jelas dan konsisten.
- Frontend lebih mudah integrasi.
- Jurnal bisa menyebut formal API contract.

---

## 12. Automated Tests

### Status Saat Ini

Belum ada file `_test.go`.

### Yang Perlu Dibuat

Test prioritas:

```text
internal/workflow/*_test.go
internal/policy/*_test.go
internal/module/monitoring/service/*_test.go
internal/module/tracker/service/*_test.go
internal/module/finance/service/*_test.go
```

Test penting:

- FSM valid transition.
- FSM invalid transition.
- wrong role rejected.
- contractor cannot validate progress.
- auditor cannot submit payment.
- payment cannot skip approval stage.
- rejected progress requires comment.
- ticket cannot close by contractor.
- payment request cannot exceed available amount.

### Manfaat

- Validasi teknis untuk jurnal Informatika.
- Mengurangi risiko regression.
- Membuat klaim sistem lebih kuat.

---

## 13. WBS Integrity Validation

### Status Saat Ini

Task WBS sudah punya `weight`, `parent_id`, dan deadline. Namun belum ada validasi kuat bahwa total bobot tepat 100% atau tree bebas cycle.

### Yang Perlu Dibuat

Rules:

- total leaf weight atau root weight harus 100%;
- child deadline tidak boleh melebihi parent deadline;
- parent tidak boleh descendant dari task itu sendiri;
- progress hanya boleh pada leaf task.

Endpoint opsional:

```text
GET /monitoring/project/{projectId}/wbs/validation
```

Return:

```json
{
  "isValid": false,
  "totalWeight": 87,
  "errors": ["TOTAL_WEIGHT_NOT_100"]
}
```

### Manfaat

- S-Curve lebih valid.
- Mencegah data WBS rusak.
- Memperkuat data integrity.

---

## 14. File Metadata dan Secure Upload

### Status Saat Ini

Upload file langsung ke Cloudinary dan module bisnis menyimpan URL. Belum ada metadata file di database.

### Yang Perlu Dibuat

Table:

```text
files
- id
- public_id
- url
- file_name
- mime_type
- size
- checksum
- uploaded_by_user_id
- uploaded_by_org_id
- entity_type
- entity_id
- created_at
```

Tambahkan validasi:

- MIME sniffing dari content, bukan hanya header.
- checksum untuk integritas.
- optional virus scanning.

### Manfaat

- Dokumen kontrak/invoice lebih traceable.
- Upload bisa diaudit.
- Mengurangi risiko file spoofing.

---

## 15. Observability dan Metrics

### Status Saat Ini

Sudah ada request logger dengan request ID, status, bytes, dan latency. Belum ada metrics, tracing, atau alerting.

### Yang Perlu Dibuat

Tambahkan:

- structured error log konsisten;
- metric endpoint `/metrics`;
- request duration histogram;
- DB query duration;
- upload duration;
- count failed login;
- count SLA violation.

Package:

```text
internal/observability/
```

Opsional:

- Prometheus metrics.
- OpenTelemetry tracing.

### Manfaat

- Sistem lebih siap production.
- Memudahkan debugging.
- Cocok untuk pembahasan reliability engineering.

---

## 16. Route dan Docs Synchronization

### Status Saat Ini

Ada perbedaan antara docs dan route aktual. Beberapa route di docs belum ada atau berbeda nama.

### Yang Perlu Dibuat

Langkah:

1. Audit semua route aktual dari `router.go` dan handler module.
2. Update `Md/Route.md`.
3. Buat OpenAPI.
4. Tambahkan test sederhana untuk route registration jika memungkinkan.

### Manfaat

- Dokumentasi tidak menyesatkan.
- Memudahkan penulisan jurnal dan demo.
- Mengurangi miskomunikasi frontend-backend.

---

## Urutan Implementasi yang Disarankan

Jika ingin bertahap, urutan paling rasional:

1. Automated Tests untuk policy layer yang sudah dibuat.
2. SLA Tracking untuk Ticket.
3. Audit Log Global.
4. Finance Concurrency Control.
5. Dashboard Cache dengan Redis.
6. Refresh Token Revocation dan Rate Limiting.
7. OpenAPI Specification.
8. Notification + Outbox Pattern.
9. WBS Integrity Validation.
10. File Metadata.
11. Observability Metrics.
12. Route/docs synchronization.

Prioritas paling tinggi untuk jurnal:

1. Test role access untuk policy layer.
2. SLA Tracking.
3. Audit trail untuk workflow transition.
4. Diagram transition dan hasil test FSM.

Empat poin tersebut paling langsung memperkuat niche Informatika yang ingin diangkat.
