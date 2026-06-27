# Improve Project: Sudut Pandang Teknologi/IT

Dokumen ini merangkum konteks teknis backend JICA Investment Monitoring System dan memberi saran improvisasi agar pembahasan jurnal teknologi/IT terlihat lebih komprehensif, matang, dan "senior level".

Fokus utama dokumen ini adalah backend: arsitektur, domain model, workflow, data integrity, security, observability, performance, dan roadmap peningkatan.

---

## 1. Ringkasan Sistem Saat Ini

Project ini adalah sistem monitoring investasi/proyek infrastruktur yang dibiayai JICA, dengan studi kasus MRT Jakarta Phase 2. Dari sisi domain, sistem mencoba menjembatani banyak aktor:

- GOV_SUPERVISOR: Bappenas atau pengawas pemerintah pusat.
- LOAN_PROVIDER: JICA sebagai penyedia pinjaman.
- EXECUTING_AGENCY: PT MRT Jakarta sebagai pelaksana proyek.
- CONTRACTOR: kontraktor pelaksana pekerjaan.
- INDEPENDENT_AUDITOR: auditor/pengawas independen.
- GOV_PARTNER: instansi pendukung seperti BPN, PLN, Dishub.

Secara teknis, backend menggunakan pola modular monolith. Setiap domain dipisah menjadi module controller dan service:

- IAM: autentikasi, organisasi, membership, RBAC.
- Project: data proyek, setting proyek, contract assignment.
- Monitoring: WBS, progress log, validasi progress, S-Curve.
- Tracker: ticketing hambatan lapangan dan riwayat penyelesaian.
- Finance: payment request, approval bertingkat, disbursement.
- Support: upload file Cloudinary dan dashboard aggregation.

Stack backend utama:

- Go sebagai bahasa backend.
- chi sebagai HTTP router.
- PostgreSQL sebagai database utama.
- sqlc untuk type-safe SQL generation.
- goose untuk migration.
- JWT untuk access token dan refresh token.
- Cloudinary untuk object/file storage.
- Redis dependency ada di project, tetapi saat ini belum digunakan oleh service aktif.

Nilai teknis utama sistem ini adalah transformasi proses lintas lembaga menjadi workflow digital yang dapat diaudit: siapa membuat data, siapa memvalidasi, status apa yang berubah, kapan perubahan terjadi, dan bagaimana data tersebut memengaruhi dashboard progres.

---

## 2. Temuan Arsitektur Backend

### 2.1 Modular Monolith

Backend saat ini cocok disebut modular monolith, bukan microservices. Semua module berjalan dalam satu binary/server, namun dibagi berdasarkan boundary domain.

Kelebihan untuk project akademik:

- Lebih sederhana untuk dikembangkan dan dijalankan.
- Tetap menunjukkan pemahaman domain-driven decomposition.
- Menghindari kompleksitas premature seperti service discovery, distributed tracing, dan message broker.
- Cocok untuk sistem internal pemerintah/lintas lembaga yang masih membutuhkan konsistensi data kuat.

Hal yang bisa ditekankan di jurnal:

> Modular monolith dipilih untuk menjaga konsistensi transaksi dan mempercepat pengembangan, sambil tetap memisahkan domain IAM, Project, Monitoring, Tracker, Finance, dan Support.

### 2.2 Layering

Struktur backend secara umum:

- `cmd/api/main.go`: entrypoint server, koneksi database, router root, middleware global.
- `config`: load env, database connection, Cloudinary, Redis.
- `internal/http`: router komposisi module.
- `internal/http/internal_middleware`: auth, request logger, role middleware.
- `internal/module/*/controller`: HTTP handler dan parsing request.
- `internal/module/*/service`: business rules.
- `internal/repository/queries`: SQL mentah untuk sqlc.
- `internal/repository/entity`: generated query code dan transaction store.
- `pkg`: shared utility seperti response, error, validator, logger.

Ini sudah cukup baik sebagai basis pembahasan backend engineering karena ada pemisahan antara HTTP concern, business logic, dan data access.

### 2.3 SQLC dan Raw SQL

Penggunaan sqlc adalah nilai plus. Dibanding ORM penuh, sqlc memberi:

- Query SQL eksplisit dan mudah diaudit.
- Type-safety di Go.
- Performa lebih predictable.
- Cocok untuk query kompleks seperti project access, dashboard aggregation, ticket latest status, dan S-Curve.

Untuk jurnal IT, ini bisa dibahas sebagai pendekatan "type-safe persistence layer" yang menjaga transparansi query namun tetap mengurangi error runtime.

### 2.4 Transaction Boundary

Repository memiliki `ExecTx` untuk transaksi database. Sudah digunakan pada beberapa flow penting:

- Create project + create contract.
- Assign government partners dengan delete old + insert new.
- Create ticket + create initial ticket log.
- Create organization + create user + create organization member.
- Create payment request + audit log dengan contract row lock.

Ini bagus karena workflow penting tidak meninggalkan data setengah jadi.

Improvisasi:

- Flow create payment request sudah memakai row lock pada contract untuk mengurangi race condition.
- Tambahkan concurrency/integration test agar proteksi tersebut bisa dibuktikan secara otomatis.
- Progress validation perlu idempotency atau unique constraint sudah ada, tetapi error handling bisa dibuat lebih eksplisit.

---

## 3. Ringkasan Domain dan Workflow Penting

### 3.1 IAM

IAM menggunakan invitation-only model. User tidak register bebas, tetapi dibuat atau diundang oleh organisasi.

Fitur utama:

- Check email untuk menentukan flow login.
- Login dengan JWT access token dan refresh token.
- Set password untuk user baru.
- Profile dan change password.
- Organization creation dan invite member.
- Membership berbasis organization role dan member role.

Nilai IT:

- Cocok untuk sistem antar lembaga karena akses harus terkendali.
- Model access control tidak hanya user-based, tetapi organization-centric.
- Ada pemisahan antara role organisasi dan role individu.

Catatan improvement:

- Refresh token sudah memiliki `jti` dan bisa direvoke melalui Redis blacklist.
- Logout dan revoke-all refresh token sudah tersedia sebagai baseline server-side revocation.
- Sisa improvement IAM adalah session/device list, audit login, dan invitation token formal.

### 3.2 Project dan Contract Assignment

Project module merepresentasikan proyek dan kontrak utama:

- GOV_SUPERVISOR membuat proyek.
- Loan provider ditentukan saat project creation.
- Executing agency, vendor/contractor, auditor, dan gov partners di-assign kemudian.
- Project settings menyimpan baseline timeline, budget, dan currency.

Nilai IT:

- Data project tidak hanya CRUD, tetapi membangun "participant graph" lintas organisasi.
- Access control project ditentukan dari kontrak dan daftar participant.
- Query `GetUserProjectAccess` menjadi pusat logika akses project.

Catatan improvement:

- Participant assignment sebaiknya memiliki audit trail khusus.
- Ubah assignment penting perlu alasan, aktor, dan timestamp historis.
- Contract value dan settings perlu validasi domain lebih ketat, termasuk revisi kontrak atau addendum.

### 3.3 Monitoring: WBS, Progress Log, dan Validation

Monitoring adalah module paling kuat untuk pembahasan teknologi.

Fitur utama:

- WBS berbentuk tree melalui `task_items.parent_id`.
- Task memiliki deadline dan weight.
- Kontraktor hanya bisa submit progress pada leaf task.
- Progress log menyimpan snapshot title, description, dan weight saat laporan dibuat.
- Auditor memvalidasi progress.
- Actual progress dihitung dari data yang sudah accepted.

Nilai IT yang penting:

- Snapshot pada progress log adalah desain auditability yang kuat.
- Sistem menjaga integritas historis saat WBS berubah setelah progress dilaporkan.
- Validasi auditor mencegah klaim kontraktor langsung masuk ke realisasi.

Catatan improvement:

- Validasi total WBS weight perlu diperkuat. Dokumen menyebut total 100%, tetapi kode saat ini baru menghitung total root weight dan belum tampak memblokir total lebih/kurang dari 100.
- Update task perlu mencegah cycle pada parent-child tree.
- Planned S-Curve di endpoint monitoring masih sederhana, sedangkan dashboard support sudah lebih matang dengan weekly data points.
- Perlu aturan apakah progress bersifat cumulative atau incremental. Saat ini query mengambil `MAX` percentage per task, namun jurnal perlu menjelaskan model yang dipilih.

### 3.4 Tracker: Ticketing dan Event-Sourced Status

Tracker menangani hambatan lapangan:

- Kontraktor membuat ticket.
- Executing agency assign ticket ke gov partner.
- Gov partner atau executing agency menambah log/status.
- Executing agency menutup ticket.

Nilai IT:

- Current status tidak disimpan sebagai kolom utama di `tickets`, tetapi dihitung dari latest `ticket_logs`.
- Ini mendekati event-sourced status model: riwayat perubahan menjadi sumber kebenaran.
- Cocok untuk transparansi birokrasi karena setiap perubahan status punya aktor, organisasi, dan timestamp.
- SLA ticket sudah mulai dipisahkan ke `sla_records`, sehingga durasi penyelesaian bisa dianalisis tanpa mengubah model utama ticket.

Catatan improvement:

- State machine status sudah diformalisasi di `internal/workflow`; langkah berikutnya adalah menambahkan actor guard jika proses bisnis ingin dibuat lebih strict.
- SLA tracking sudah tersedia sebagai baseline, tetapi dashboard bottleneck, konfigurasi SLA per state, escalation, dan notifikasi masih perlu dibuat.
- Tambahkan attachment metadata, bukan hanya URL string.

### 3.5 Finance: Tiered Approval dan Disbursement

Finance module menangani payment request:

- Contractor submit payment request.
- Auditor approve/reject teknis.
- Executing agency approve/reject administratif.
- Loan provider melakukan disbursement.
- Status disimpan sebagai enum.

Nilai IT:

- Ini adalah workflow persetujuan bertingkat.
- Ada status transition yang eksplisit.
- Ada signature-like fields: org id, user id, timestamp pada tiap approval layer.
- Payment summary menghitung paid, pending, available amount.

Catatan improvement:

- Disbursement service perlu validasi eksplisit bahwa org yang membayar adalah loan provider participant dalam contract, bukan hanya dilakukan di controller.
- Payment request creation sudah memakai transaction + row lock pada contract. Sisa peningkatan adalah concurrency test dan observability lock wait time.
- Rejection reason/comment belum tampak sebagai field utama. Untuk audit, rejection harus memiliki alasan.
- Digital signature bisa ditingkatkan dari sekadar user/org/timestamp menjadi signed approval record.

### 3.6 Support: Upload dan Dashboard Aggregator

Support module berisi:

- Upload single/multiple file ke Cloudinary.
- Dashboard S-Curve dan statistics.

Nilai IT:

- Upload dipisahkan dari module bisnis. Module bisnis menyimpan URL hasil upload.
- Dashboard aggregation dipisahkan agar query/kalkulasi analytics tidak mencemari transactional module.

Catatan improvement:

- File upload perlu virus scanning, checksum, content sniffing, dan metadata table.
- Cloudinary public URL sebaiknya disertai access policy jika dokumen kontrak/invoice sensitif.
- Dashboard bisa memakai cache dan pre-computation jika data membesar.

---

## 4. Gap Antara Dokumentasi dan Implementasi

Bagian ini penting untuk jurnal karena menunjukkan evaluasi kritis terhadap sistem.

### 4.1 Real-time Monitoring Belum Real-time

Dokumen menyebut real-time monitoring, tetapi implementasi backend saat ini masih HTTP request/response biasa. Belum ada WebSocket, Server-Sent Events, message broker, atau push notification.

Cara menulis di jurnal:

> Sistem menyediakan near-real-time visibility melalui API dashboard dan data agregat. Untuk mencapai real-time monitoring penuh, sistem dapat ditingkatkan dengan event-driven notification dan streaming update.

### 4.2 Redis Sudah Mulai Digunakan

Redis dependency dan config sekarang digunakan untuk refresh token revocation, rate limiting auth endpoint, dan dashboard cache. Koneksi Redis dibuat optional agar server tetap bisa run tanpa Redis lokal; jika Redis tersedia, logout, revoke-all refresh token, rate limit counter, dan cache dashboard memakai Redis.

Pilihan improvement:

- notification queue;
- distributed lock untuk finance request jika nanti perlu lock lintas proses.

Untuk jurnal IT, Redis sekarang bisa dibahas sebagai komponen security untuk token revocation/brute-force mitigation dan performance optimization untuk dashboard.

### 4.3 Beberapa Endpoint Belum Lengkap

Contoh:

- Monitoring detail task masih `NOT_IMPLEMENTED`.
- Monitoring detail progress log masih `NOT_IMPLEMENTED`.
- Route docs perlu disinkronkan ulang dengan endpoint auth terbaru seperti logout dan revoke-all.
- Ada perbedaan nama route di docs dan kode, misalnya beberapa docs memakai plural `tasks`, sementara kode memakai `task`.

Improvement:

- Jadikan route documentation sebagai OpenAPI spec.
- Generate API client untuk frontend dari spec.
- Tambahkan contract test agar docs dan backend tidak divergen.

### 4.4 Test Coverage Belum Ada

Tidak ditemukan file `_test.go` di backend. Untuk sistem yang membawa workflow finance dan approval, ini gap besar.

Minimum test yang perlu ada:

- Unit test service business rules.
- Integration test repository dengan test database.
- Authorization test untuk role dan participant access.
- Workflow test untuk Monitoring, Tracker, Finance.
- Race/concurrency test untuk payment request.

### 4.5 Observability Masih Dasar

Sudah ada request ID, request logger, status, bytes, latency. Ini bagus sebagai fondasi.

Yang belum tampak:

- Structured error logging yang konsisten.
- Metrics seperti request latency histogram, DB query duration, upload duration.
- Tracing lintas handler-service-repository.
- Alerting untuk failure rate, slow query, dan failed upload.

---

## 5. Rekomendasi Improvisasi Prioritas

### Priority 1: Perkuat Security dan Access Control

Alasan:

Sistem ini membawa data proyek, kontrak, invoice, progress, dan approval lintas lembaga. Security harus menjadi bagian utama narasi IT.

Rekomendasi:

1. Centralized Policy Layer

   Saat ini pengecekan role banyak tersebar di controller. Buat package policy, misalnya:

   - `CanCreateProject(user)`
   - `CanConfigureProject(user, projectID)`
   - `CanSubmitProgress(user, taskID)`
   - `CanApprovePayment(user, requestID, level)`

   Manfaat:

   - Mengurangi duplikasi role check.
   - Lebih mudah dites.
   - Lebih mudah dijelaskan sebagai RBAC/ABAC hybrid.

2. Refresh Token Revocation

   Baseline sudah dibuat dengan token id (`jti`) dan Redis blacklist dengan TTL. `revoke-all` memakai cutoff timestamp per user.

   Sisa peningkatan:

   - PostgreSQL untuk audit permanen jika ingin session history.
   - UI/session list untuk revoke device tertentu.

3. Rate Limiting Login

   Baseline rate limiting sudah dibuat pada:

   - `/iam/auth/check-email`
   - `/iam/auth/login`
   - `/iam/auth/refresh-token`

   Redis dipakai jika tersedia, dengan fallback in-memory untuk local development. Sisa peningkatan adalah env-based limit/window dan observability request yang diblokir.

4. Audit Trail Global

   Baseline `audit_logs` sudah dibuat untuk mencatat:

   - actor user id;
   - actor organization id;
   - action;
   - entity type;
   - entity id;
   - before/after JSON;
   - request id;
   - IP/user agent;
   - timestamp.

   Ini sangat kuat untuk jurnal karena sistem lintas negara/lembaga memerlukan accountability. Tahap berikutnya adalah menambahkan endpoint pembacaan audit log, policy akses audit, dan redaction untuk field sensitif.

### Priority 2: Formalisasi Workflow Engine Ringan

Alasan:

Project ini bukan CRUD biasa. Kekuatan IT-nya ada pada workflow lintas aktor.

Rekomendasi:

1. State Machine untuk Finance

   Transition table dasar sudah dibuat di `internal/workflow`. Tahap berikutnya adalah memperluas state machine dengan alasan reject, audit event, dan guard actor.

   - SUBMITTED -> AUDITOR_APPROVED
   - SUBMITTED -> AUDITOR_REJECTED
   - AUDITOR_APPROVED -> AGENCY_APPROVED
   - AUDITOR_APPROVED -> AGENCY_REJECTED
   - AGENCY_APPROVED -> PAID

   Reject harus menyimpan reason.

2. State Machine untuk Tracker

   Definisikan transisi:

   - OPEN -> ASSIGNED
   - ASSIGNED -> IN_PROGRESS
   - IN_PROGRESS -> RESOLVED
   - RESOLVED -> CLOSED

   Jika perlu reopen:

   - RESOLVED -> IN_PROGRESS
   - CLOSED -> REOPENED, dengan status enum baru.

3. Workflow Timeline API

   Buat endpoint timeline untuk project:

   - contract assignment;
   - WBS creation;
   - progress submitted;
   - progress validated;
   - ticket created/resolved;
   - payment approved/paid.

   Ini bisa menjadi fitur "digital accountability timeline".

### Priority 3: Perkuat Data Integrity dan Concurrency

Alasan:

Sistem finance dan progress rawan data race jika dipakai banyak user.

Rekomendasi:

1. Payment Request Atomicity

   Saat create payment request, baseline sudah dibuat:

   - buka transaction;
   - lock contract row `FOR UPDATE`;
   - hitung paid/pending;
   - insert payment request;
   - commit.

   Tujuannya mencegah double spending pada available amount. Tahap berikutnya adalah membuat concurrency test untuk membuktikan hanya satu request paralel yang bisa lolos saat rule pending request aktif.

2. WBS Weight Constraint

   Buat rule:

   - total root task weight = 100;
   - atau total leaf task weight = 100;
   - tentukan satu model secara eksplisit.

   Jika ingin lebih ketat, buat validation endpoint:

   - `GET /monitoring/project/{projectId}/wbs/validation`

3. Prevent Tree Cycle

   Saat update parent task, cegah task menjadi child dari descendant-nya sendiri.

4. Database Constraints dan Indexes

   Tambahkan index untuk query yang sering:

   - `organization_members(user_id, deleted_at)`
   - `contracts(project_id, deleted_at)`
   - `task_items(contract_id, deleted_at)`
   - `progress_logs(task_item_id, deleted_at)`
   - `progress_validations(progress_log_id, deleted_at)`
   - `ticket_logs(ticket_id, deleted_at, created_at DESC)`
   - `payment_requests(contract_id, status, deleted_at)`

   Ini akan memperkuat pembahasan performance engineering.

### Priority 4: Jadikan Dashboard Analytics Lebih Matang

Alasan:

Dashboard adalah pusat nilai bagi JICA dan pemerintah. Di sinilah transparansi menjadi terlihat.

Rekomendasi:

1. S-Curve Cache

   Dashboard S-Curve dan stats sudah memakai Redis read-through cache:

   - key: `{REDIS_PREFIX}:dashboard:scurve:{projectId}`
   - key: `{REDIS_PREFIX}:dashboard:stats:{projectId}`
   - TTL: 3 menit
   - invalidasi saat progress validation, ticket status, payment disburse, WBS, atau project settings berubah.

2. Materialized View

   Untuk PostgreSQL, buat materialized view untuk:

   - progress per project;
   - ticket stats;
   - finance disbursement summary.

3. Baseline vs Revised Baseline

   Proyek infrastruktur sering mengalami perubahan jadwal. Tambahkan:

   - original baseline;
   - revised baseline;
   - reason for revision;
   - approver.

   Ini membuat jurnal lebih realistis terhadap project governance.

4. Risk Indicator

   Buat indikator:

   - schedule variance;
   - cost absorption variance;
   - open ticket aging;
   - payment delay;
   - unresolved gov partner dependency.

### Priority 5: Testing dan Quality Gate

Alasan:

Tanpa test, klaim reliability sulit dipertahankan di jurnal teknologi.

Rekomendasi test:

1. IAM

   - login invalid;
   - check email;
   - set password;
   - role membership.

2. Project

   - only GOV_SUPERVISOR ADMIN can create project;
   - loan provider must have LOAN_PROVIDER role;
   - contract assignment role validation.

3. Monitoring

   - child deadline cannot exceed parent;
   - cannot submit progress on non-leaf task;
   - cannot validate same progress twice;
   - rejected validation requires comment.

4. Tracker

   - create ticket creates initial OPEN log;
   - assign ticket creates ASSIGNED log;
   - latest log determines current status.

5. Finance

   - invalid state transition rejected;
   - contractor cannot exceed contract value;
   - only auditor can approve auditor stage;
   - concurrent payment requests cannot exceed available amount.

Quality gate:

- `go test ./...`
- `go vet ./...`
- staticcheck
- sqlc generate check
- migration up/down test

### Priority 6: API Contract dan Documentation

Alasan:

Frontend dan backend harus sepakat pada route, payload, dan response.

Rekomendasi:

1. OpenAPI Spec

   Buat `openapi.yaml` sebagai sumber kebenaran.

2. Generate Client

   Frontend bisa generate TypeScript client dari OpenAPI.

3. Response Schema

   Standard response sudah ada. Jadikan formal di OpenAPI:

   - `metaData`
   - `responseMessage`
   - `data`
   - `validationErrors`
   - `filterQuery`
   - `pagination`

4. Contract Test

   Tambahkan test agar route docs tidak berbeda dengan route aktual.

### Priority 7: Event-Driven Notification

Alasan:

Project mengklaim monitoring real-time. Untuk membuatnya benar-benar kuat, perlu event-driven architecture meskipun tetap modular monolith.

Rekomendasi:

1. Domain Events

   Buat event internal:

   - `ProgressSubmitted`
   - `ProgressValidated`
   - `TicketAssigned`
   - `TicketResolved`
   - `PaymentApproved`
   - `PaymentPaid`

2. Outbox Pattern

   Simpan event di table `outbox_events` dalam transaksi yang sama dengan perubahan data.

3. Worker

   Worker membaca outbox dan mengirim:

   - email;
   - notification;
   - WebSocket/SSE update;
   - audit pipeline.

4. Redis atau PostgreSQL Queue

   Untuk tahap awal, PostgreSQL outbox cukup. Redis bisa dipakai untuk queue/cache jika ingin menunjukkan peningkatan teknologi.

---

## 6. Saran Topik Jurnal IT

Berikut beberapa angle jurnal yang cocok dengan project ini.

### Opsi A: Workflow-Based Monitoring System

Judul potensial:

> Perancangan Backend Modular Monolith untuk Sistem Monitoring Investasi Infrastruktur Berbasis Workflow Multi-Stakeholder

Fokus:

- Modular monolith.
- RBAC berbasis organisasi.
- Workflow progress, ticket, dan finance.
- Auditability.

Kontribusi:

- Menunjukkan bagaimana proses lintas lembaga dimodelkan sebagai workflow digital.
- Menjelaskan boundary IAM, Project, Monitoring, Tracker, Finance.

### Opsi B: Auditability dan Data Integrity

Judul potensial:

> Penerapan Snapshot Data dan Event Log untuk Meningkatkan Auditability pada Sistem Monitoring Proyek Infrastruktur

Fokus:

- Progress log snapshot.
- Ticket log sebagai sumber status.
- Approval signature finance.
- Audit trail global sebagai improvement.

Kontribusi:

- Membahas cara menjaga riwayat data tetap valid walaupun master data berubah.
- Cocok untuk konteks proyek pemerintah dan pendanaan internasional.

### Opsi C: Secure Multi-Organization Access Control

Judul potensial:

> Desain Access Control Berbasis Organisasi untuk Platform Monitoring Proyek Multi-Stakeholder

Fokus:

- Organization role.
- Member role.
- Project participant access.
- Policy layer.
- Token security.

Kontribusi:

- Menunjukkan bahwa akses tidak cukup berbasis user role tunggal.
- Perlu kombinasi role organisasi, role member, dan relasi project.

### Opsi D: Real-Time Readiness dan Event-Driven Roadmap

Judul potensial:

> Evolusi Sistem Monitoring Proyek dari Request-Response API menuju Event-Driven Near Real-Time Architecture

Fokus:

- Kondisi sekarang: REST API + dashboard aggregation.
- Gap real-time.
- Domain event, outbox, Redis, WebSocket/SSE.

Kontribusi:

- Menjelaskan roadmap realistis tanpa langsung memecah sistem menjadi microservices.

---

## 7. Rekomendasi Narasi "Senior Level"

Agar tulisan tidak terdengar seperti laporan CRUD, gunakan istilah dan sudut pandang berikut.

### 7.1 Dari CRUD ke Workflow

Jangan hanya menulis "sistem memiliki fitur tambah data progress". Tulis:

> Sistem memodelkan pelaporan progress sebagai workflow klaim dan validasi. Klaim kontraktor tidak langsung menjadi realisasi proyek, melainkan harus melewati validasi auditor independen sebelum masuk ke perhitungan S-Curve.

### 7.2 Dari Role Biasa ke Organization-Centric Access Control

Tulis:

> Hak akses tidak hanya ditentukan oleh identitas user, tetapi oleh relasi user terhadap organisasi dan relasi organisasi terhadap proyek. Model ini lebih sesuai untuk sistem lintas lembaga.

### 7.3 Dari Log Biasa ke Auditability

Tulis:

> Ticket status diturunkan dari log terakhir, sehingga histori perubahan tetap tersimpan dan dapat ditelusuri. Pendekatan ini meningkatkan transparansi karena setiap perubahan status memiliki aktor, organisasi, dan waktu kejadian.

### 7.4 Dari Dashboard ke Decision Support

Tulis:

> Dashboard tidak hanya menampilkan data, tetapi mengagregasi progress fisik, serapan anggaran, dan hambatan lapangan sebagai basis decision support bagi pemerintah dan loan provider.

### 7.5 Dari Monolith ke Modular Monolith

Tulis:

> Sistem tidak menggunakan microservices karena kebutuhan utama adalah konsistensi data dan simplicity deployment. Modular monolith dipilih agar boundary domain tetap jelas tanpa menambah kompleksitas distribusi.

---

## 8. Roadmap Implementasi yang Paling Direkomendasikan

Jika waktu pengembangan terbatas, urutan paling efektif:

1. Sinkronisasi docs dan route aktual.
2. Tambahkan test untuk workflow Monitoring, Tracker, dan Finance.
3. Lengkapi automated test untuk centralized policy layer yang sudah dibuat.
4. Tambahkan endpoint pembacaan audit_logs global.
5. Tambahkan concurrency test untuk finance transaction lock.
6. Lengkapi audit policy dan redaction untuk state machine finance dan tracker.
7. Tambahkan dashboard cache atau queue berbasis Redis.
8. Tambahkan OpenAPI spec.
9. Tambahkan dashboard risk indicator.
10. Tambahkan outbox pattern dan notification worker.

Jika ingin cepat menaikkan kualitas jurnal, tiga improvement paling bernilai adalah:

1. Audit trail global.
2. Audit trail untuk workflow state machine.
3. Test coverage untuk authorization dan financial workflow.

Alasannya: ketiganya langsung menunjukkan maturity backend, bukan hanya penambahan fitur visual.

---

## 9. Risiko Teknis Saat Ini

### Risiko 1: Authorization belum punya test khusus

Policy layer project-scoped sudah dipusatkan di `internal/policy`, tetapi belum ada automated test khusus untuk membuktikan setiap role dan resource access bekerja konsisten.

Mitigasi:

- Authorization unit tests untuk `internal/policy`.
- Perluas policy ke IAM organization jika ingin seluruh access control berada di satu layer.

### Risiko 2: Financial race condition

Payment request sebelumnya menghitung available amount lalu insert request tanpa lock. Jika request paralel, potensi race condition muncul.

Mitigasi:

- Transaction dengan row lock sudah diterapkan pada create payment request.
- Unique/partial constraint untuk pending request jika rule hanya satu pending request per contract.

### Risiko 3: Klaim real-time terlalu kuat

Belum ada push update.

Mitigasi:

- Ubah istilah menjadi near-real-time untuk kondisi sekarang.
- Tambahkan roadmap event-driven.

### Risiko 4: Test belum ada

Workflow penting belum punya automated verification.

Mitigasi:

- Prioritaskan service tests.
- Gunakan test database untuk repository integration tests.

### Risiko 5: Dokumen dan route tidak sinkron

Docs menyebut beberapa endpoint yang belum ada atau punya nama berbeda.

Mitigasi:

- OpenAPI sebagai single source of truth.
- Contract testing.

---

## 10. Rekomendasi Struktur Pembahasan Jurnal IT

Struktur tulisan yang disarankan:

1. Pendahuluan

   Jelaskan masalah monitoring proyek infrastruktur lintas lembaga: transparansi, validasi progress, bottleneck, dan pencairan dana.

2. Analisis Kebutuhan Sistem

   Bahas aktor, role organisasi, workflow, dan kebutuhan auditability.

3. Desain Arsitektur

   Jelaskan modular monolith, module boundary, dan data flow.

4. Desain Backend

   Bahas Go, chi, PostgreSQL, sqlc, JWT, Cloudinary, migration, transaction.

5. Desain Access Control

   Bahas organization role, member role, project participant access.

6. Desain Workflow

   Bahas Monitoring, Tracker, Finance.

7. Data Integrity dan Auditability

   Bahas snapshot progress, ticket logs, approval timestamp, audit log improvement.

8. Evaluasi dan Gap

   Bahas real-time gap, Redis unused, test coverage, docs mismatch, performance.

9. Roadmap Improvisasi

   Bahas policy layer, audit trail, state machine, Redis, OpenAPI, outbox.

10. Kesimpulan

   Tegaskan bahwa nilai IT utama sistem adalah digitalisasi governance proyek lintas lembaga berbasis workflow, access control, dan auditable data.

---

## 11. Kalimat Kunci yang Bisa Dipakai di Jurnal

- Sistem dirancang sebagai modular monolith untuk menjaga keseimbangan antara maintainability dan konsistensi transaksi.
- Hak akses berbasis organisasi lebih sesuai untuk konteks multi-stakeholder dibanding role user tunggal.
- Progress proyek tidak dihitung dari klaim kontraktor, tetapi dari klaim yang telah divalidasi auditor independen.
- Snapshot data pada progress log menjaga integritas historis ketika struktur WBS berubah.
- Ticket log berperan sebagai event history yang memungkinkan status hambatan ditelusuri dari waktu ke waktu.
- Approval bertingkat pada finance module merepresentasikan separation of duties antara auditor, executing agency, dan loan provider.
- Dashboard berfungsi sebagai decision support layer yang menggabungkan progress fisik, serapan finansial, dan status hambatan.
- Untuk mencapai real-time monitoring, sistem dapat berevolusi dari REST API menuju event-driven notification menggunakan outbox pattern dan Redis/WebSocket/SSE.

---

## 12. Kesimpulan

Backend project ini sudah memiliki fondasi domain yang kuat untuk dijadikan jurnal teknologi/IT. Kekuatan utamanya bukan pada kompleksitas framework, tetapi pada pemodelan proses lintas lembaga menjadi workflow digital yang dapat diaudit.

Bagian paling layak diangkat:

- modular monolith berbasis domain;
- organization-centric RBAC;
- WBS dan validated progress;
- ticket log sebagai event history;
- tiered finance approval;
- dashboard aggregation untuk monitoring.

Bagian yang perlu ditingkatkan agar lebih matang:

- automated test untuk centralized policy layer;
- audit log global;
- audit trail dan diagram untuk workflow state machine;
- concurrency control pada finance;
- OpenAPI dan contract test;
- automated test coverage;
- observability;
- Redis/event-driven usage yang jelas.

Dengan peningkatan tersebut, project ini bisa diposisikan bukan hanya sebagai aplikasi monitoring, tetapi sebagai backend governance platform untuk transparansi, akuntabilitas, dan pengambilan keputusan pada proyek investasi infrastruktur internasional.
