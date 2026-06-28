# Outline Jurnal Informatika

## Judul

**Perancangan Backend Workflow Governance Proyek Infrastruktur Transnasional Berbasis ODA Menggunakan Finite State Machine, SLA Tracking, dan Realtime Event Delivery: Studi Kasus MRT Jakarta Fase 2A**

Judul ini menegaskan bahwa fokus jurnal adalah rekayasa backend untuk workflow governance. MRT Jakarta Fase 2A digunakan sebagai studi kasus/konteks kebutuhan, bukan klaim bahwa sistem sudah menjadi sistem resmi MRT/JICA.

---

## Abstrak

Abstrak menjelaskan masalah koordinasi proyek infrastruktur lintas lembaga, khususnya red tape, asimetri informasi, dan kebutuhan transparansi status. Tuliskan bahwa penelitian merancang prototipe backend dengan empat fokus utama: FSM workflow governance, organization-centric access control, SLA tracking + auditability, dan realtime event delivery. Akhiri dengan hasil evaluasi berupa pengujian workflow, authorization, SLA, event delivery, dan contract build.

Kata kunci:

- workflow governance;
- finite state machine;
- SLA tracking;
- organization-centric access control;
- audit trail;
- outbox pattern;
- WebSocket;
- project monitoring.

---

## I. Pendahuluan

### 1.1 Latar Belakang

Bahas proyek infrastruktur transnasional seperti MRT Jakarta Fase 2A sebagai domain yang melibatkan banyak aktor: pemerintah, executing agency, pemberi dana asing, kontraktor, auditor, dan instansi pendukung. Tekankan masalah Informatika: koordinasi data, validasi status, tracking hambatan, approval bertingkat, dan kebutuhan visibility lintas organisasi.

### 1.2 Identifikasi Masalah

Jelaskan masalah utama yang ingin diselesaikan sistem: workflow manual sulit dilacak, status proses dapat tidak konsisten, keterlambatan tidak terukur, akses lintas organisasi rawan salah izin, dan dashboard membutuhkan pembaruan cepat tanpa menjadikan data realtime sebagai sumber kebenaran utama.

### 1.3 Rumusan Masalah

Isi subbab ini dengan pertanyaan penelitian:

1. Bagaimana memodelkan workflow monitoring proyek lintas lembaga menggunakan Finite State Machine?
2. Bagaimana menerapkan access control berbasis organisasi agar setiap aktor hanya dapat menjalankan aksi sesuai kewenangannya?
3. Bagaimana SLA tracking dan audit trail dapat meningkatkan traceability dan accountability?
4. Bagaimana outbox pattern dan WebSocket dapat mendukung realtime delivery tanpa menggantikan REST sebagai source of truth?
5. Bagaimana pengujian backend membuktikan correctness dari workflow, authorization, dan delivery event?

### 1.4 Tujuan Penelitian

Tuliskan tujuan penelitian: merancang backend workflow governance, menerapkan FSM pada proses monitoring/ticket/finance, menerapkan policy layer, menyediakan SLA tracking dan audit trail, menyediakan realtime event delivery, serta mengevaluasi sistem melalui pengujian teknis.

### 1.5 Batasan Masalah

Jelaskan bahwa sistem adalah prototipe akademik berbasis studi kasus. Sistem tidak diklaim terintegrasi resmi dengan MRT/JICA/Pemprov. WebSocket hanya menjadi sinyal refetch, sedangkan REST tetap menjadi sumber data utama. Fitur pendukung seperti upload, cache, rate limit, dan refresh token revocation tidak menjadi fokus pembahasan utama.

### 1.6 Kontribusi Penelitian

Kontribusi utama yang ditulis cukup empat:

- model workflow governance berbasis FSM;
- organization-centric policy layer;
- SLA tracking dan auditability;
- event-driven realtime delivery berbasis outbox dan WebSocket.

---

## II. Tinjauan Pustaka

### 2.1 Sistem Informasi Monitoring Proyek

Bahas konsep monitoring proyek: progress tracking, issue tracking, approval flow, dashboard status, dan kebutuhan transparansi lintas organisasi.

### 2.2 Finite State Machine untuk Workflow

Jelaskan FSM sebagai model formal untuk state, event/input, transition, initial state, dan final state. Kaitkan dengan kebutuhan mencegah invalid transition dan status loncat pada proses proyek.

### 2.3 Access Control pada Sistem Multi-Stakeholder

Bahas RBAC, ABAC, dan kebutuhan model organization-centric. Jelaskan bahwa akses tidak cukup berdasarkan user role, tetapi juga organisasi user, role organisasi, membership, dan relasi organisasi terhadap project/contract.

### 2.4 SLA Tracking dan Audit Trail

Bahas SLA sebagai mekanisme pengukuran durasi proses pada state tertentu. Hubungkan dengan audit trail sebagai catatan siapa melakukan apa, kapan, dan terhadap entity apa.

### 2.5 Event-Driven Architecture, Outbox Pattern, dan WebSocket

Jelaskan masalah dual-write dan bagaimana outbox pattern membantu menjaga event tidak hilang setelah transaksi database. Bahas WebSocket sebagai channel delivery event, bukan pengganti REST API.

---

## III. Metode Penelitian

### 3.1 Desain Penelitian

Gunakan metode Research and Development dengan pendekatan prototyping. Alur penelitian: analisis kebutuhan, desain workflow, desain arsitektur, implementasi backend, integrasi frontend seperlunya, dan evaluasi teknis.

### 3.2 Analisis Aktor Lintas Institusi

Jelaskan aktor: GOV_SUPERVISOR, LOAN_PROVIDER, EXECUTING_AGENCY, CONTRACTOR, INDEPENDENT_AUDITOR, dan GOV_PARTNER. Tulis peran masing-masing serta bagaimana peran tersebut dinormalisasi menjadi role sistem.

### 3.3 Perancangan Arsitektur Backend

Bahas arsitektur modular monolith. Jelaskan layer utama: HTTP router, middleware, controller, service, policy/workflow/audit/outbox, repository SQLC, PostgreSQL, Redis, dan WebSocket. Supporting features cukup disebut sebagai bagian infrastruktur, bukan bahasan utama.

### 3.4 Perancangan FSM Workflow Governance

Bahas tiga workflow sebagai contoh implementasi FSM:

- monitoring progress: submit dan validasi progress;
- ticket: hambatan dari open sampai closed;
- finance: approval pembayaran bertingkat sampai paid.

Subbab ini harus memuat state, input transition, actor, dan invalid transition.

### 3.5 Perancangan Organization-Centric Policy Layer

Bahas rule authorization: user harus member organisasi, organisasi harus participant project, actor harus sesuai role contract, dan action tertentu membutuhkan admin/member role tertentu. Jelaskan alasan rule dipusatkan pada policy layer.

### 3.6 Perancangan SLA Tracking dan Auditability

Bahas rancangan SLA record, field penting seperti `entityType`, `entityId`, `state`, `ownerOrgId`, `enteredAt`, `dueAt`, `resolvedAt`, dan status SLA. Bahas juga audit log sebagai mekanisme traceability proses.

### 3.7 Perancangan Realtime Event Delivery

Bahas alur event: transaksi bisnis, insert outbox event, worker memproses event, publish ke topic, WebSocket client menerima event, lalu frontend melakukan refetch REST query. Jelaskan topic project seperti monitoring, ticket, dan finance.

### 3.8 Strategi Pengujian

Jelaskan pengujian yang relevan dengan fokus utama: workflow FSM test, policy test, SLA/tracker test, audit/outbox test, WebSocket test, dan controller contract test. Supporting test seperti upload/cache/auth cukup disebut singkat jika diperlukan.

---

## IV. Hasil dan Pembahasan

### 4.1 Implementasi Arsitektur Backend Workflow Governance

Bahas hasil implementasi modular monolith dan pembagian module utama. Jelaskan stack Go, PostgreSQL, SQLC, Redis, JWT, outbox worker, dan WebSocket secara ringkas. Jangan jadikan cache/upload/auth security sebagai subbab utama.

### 4.2 Implementasi FSM Transition Engine

Bahas implementasi transition validation pada monitoring, ticket, dan finance. Tunjukkan contoh transisi valid dan invalid. Tekankan bahwa backend mencegah workflow lompat tahap, misalnya payment tidak bisa langsung dari `SUBMITTED` ke `PAID`.

### 4.3 Implementasi Organization-Centric Policy Layer

Bahas hasil pemusatan authorization. Jelaskan contoh contractor hanya dapat submit progress/payment, auditor hanya dapat validasi/approve tahap auditor, executing agency mengelola assignment/approval administratif, dan loan provider melakukan disbursement.

### 4.4 Implementasi SLA Tracking dan Auditability

Bahas SLA ticket sebagai alat mendeteksi bottleneck. Jelaskan status SLA, remaining time, owner organization, dan due date. Gabungkan audit log di subbab ini sebagai mekanisme accountability: siapa melakukan aksi, terhadap entity apa, kapan, dan dalam request apa.

### 4.5 Implementasi Event-Driven Realtime Delivery

Bahas outbox pattern, domain event, topic authorization, Redis Pub/Sub fanout, dan WebSocket. Tegaskan bahwa event realtime hanya memberi sinyal agar frontend refetch REST API, sehingga konsistensi data tetap bergantung pada PostgreSQL/REST.

### 4.6 Keterkaitan dengan Studi Kasus MRT Jakarta Fase 2A

Jelaskan bahwa studi kasus MRT Fase 2A dipakai untuk memvalidasi urgensi sistem: koordinasi lintas institusi, visibility pemberi dana asing, hambatan lapangan, validasi progress, dan approval bertingkat. Tegaskan lagi bahwa ini prototipe berbasis konteks, bukan sistem resmi.

---

## V. Evaluasi Sistem

### 5.1 Evaluasi Workflow Correctness

Bahas hasil pengujian FSM: valid transition diterima, invalid transition ditolak, terminal state tidak bisa dilanjutkan sembarangan, dan actor yang salah tidak dapat menjalankan action.

### 5.2 Evaluasi Authorization

Bahas hasil pengujian policy layer: user tanpa project access ditolak, organization role salah ditolak, member role tidak cukup ditolak, dan participant yang valid dapat mengakses resource.

### 5.3 Evaluasi SLA dan Auditability

Bahas pengujian SLA/tracker serta audit logging. Jelaskan bahwa sistem dapat menghitung status SLA dan mencatat aksi penting untuk kebutuhan traceability.

### 5.4 Evaluasi Realtime Delivery

Bahas pengujian WebSocket dan outbox: connect dengan access token, subscribe topic authorized, rejected topic tidak memutus koneksi, ping/pong, event outbox dipetakan ke realtime envelope, dan frontend refetch setelah event diterima.

### 5.5 Evaluasi Build dan Contract

Bahas verifikasi teknis dengan `go test ./...` dan `npm run build`. Contract frontend cukup disebut yang terkait fokus utama: ticket SLA, finance/ticket FSM UX, WebSocket client, dan error handling workflow.

---

## VI. Kesimpulan dan Saran

### 6.1 Kesimpulan

Rangkum bahwa workflow proyek lintas lembaga dapat dimodelkan dengan FSM, access control berbasis organisasi menjaga boundary aktor, SLA tracking dan audit trail meningkatkan traceability, dan outbox + WebSocket mendukung realtime delivery tanpa menggantikan REST sebagai source of truth.

### 6.2 Keterbatasan

Tuliskan keterbatasan: sistem masih prototipe, belum terintegrasi resmi dengan instansi, belum ada API Gateway lintas lembaga, WebSocket belum menjadi sumber data utama, audit log belum immutable secara database-level, dan evaluasi belum mencakup benchmark produksi.

### 6.3 Saran Pengembangan

Tulis saran yang tetap relevan dengan fokus utama: konfigurasi SLA per project/state, audit log viewer, notification center, OpenAPI contract, observability metrics, dan hardening deployment.

---

## Catatan Fitur Supporting

Fitur berikut tidak perlu menjadi subbab utama jurnal:

- dashboard cache;
- secure upload metadata;
- refresh token revocation;
- rate limiting auth;
- WBS integrity validation;
- finance concurrency control;
- UI error mapping;
- frontend contract update.

Fitur tersebut boleh disebut singkat pada arsitektur, evaluasi, atau saran pengembangan jika diperlukan, tetapi jangan dijadikan inti pembahasan agar jurnal tetap fokus.
