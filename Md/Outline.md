# Outline Jurnal Informatika

## Judul

**Perancangan Backend Workflow Governance Proyek Infrastruktur Transnasional Berbasis ODA Menggunakan Finite State Machine, SLA Tracking, dan Realtime Event Delivery: Studi Kasus MRT Jakarta Fase 2A**

Bagian judul menegaskan bahwa jurnal ini berfokus pada kontribusi Informatika, yaitu rancangan dan implementasi backend untuk mengelola workflow proyek infrastruktur multi-stakeholder. MRT Jakarta Fase 2A diposisikan sebagai studi kasus atau konteks kebutuhan, bukan klaim bahwa sistem sudah dipakai secara resmi oleh MRT/JICA.

---

## Abstrak

Abstrak menjelaskan ringkas latar belakang kompleksitas proyek infrastruktur transnasional berbasis ODA, masalah red tape dan asimetri informasi, metode pengembangan prototipe backend, teknologi yang digunakan, serta hasil utama sistem. Tekankan bahwa sistem menggunakan FSM, SLA Tracking, Organization-Centric Access Control, audit trail, outbox pattern, Redis, dan WebSocket realtime delivery. Akhiri dengan kontribusi utama: peningkatan transparansi, traceability, data integrity, dan readiness untuk monitoring digital lintas lembaga.

Kata kunci yang disarankan:

- workflow governance;
- finite state machine;
- SLA tracking;
- RBAC;
- audit trail;
- outbox pattern;
- WebSocket;
- project monitoring.

---

## I. Pendahuluan

### 1.1 Latar Belakang

Subbab ini menjelaskan konteks proyek infrastruktur transnasional seperti MRT Jakarta Fase 2A yang melibatkan banyak aktor: pemerintah, executing agency, pemberi dana asing, kontraktor, auditor, dan instansi pendukung. Fokus penulisan bukan pada diplomasi, tetapi pada masalah teknis koordinasi data, validasi progress, pelaporan hambatan, approval pembayaran, dan kebutuhan transparansi status.

### 1.2 Identifikasi Masalah

Subbab ini merumuskan masalah dari sudut Informatika. Masalah yang dapat ditulis meliputi sulitnya memodelkan workflow lintas lembaga, risiko status proses yang tidak konsisten, keterlambatan yang tidak terukur, akses data lintas organisasi yang rawan salah izin, serta kebutuhan pembaruan dashboard yang lebih responsif tanpa mengorbankan konsistensi data.

### 1.3 Rumusan Masalah

Subbab ini berisi pertanyaan penelitian. Contoh rumusan masalah:

1. Bagaimana memodelkan proses monitoring proyek lintas lembaga menjadi workflow formal berbasis state dan transition?
2. Bagaimana menerapkan access control berbasis organisasi dan role untuk menjaga batas kewenangan antar stakeholder?
3. Bagaimana SLA Tracking dapat digunakan untuk mendeteksi bottleneck proses?
4. Bagaimana outbox pattern dan WebSocket dapat mendukung realtime delivery tanpa menggantikan REST sebagai source of truth?
5. Bagaimana pengujian backend membuktikan correctness workflow dan rule sistem?

### 1.4 Tujuan Penelitian

Subbab ini menjelaskan tujuan yang ingin dicapai. Tujuannya adalah merancang dan mengimplementasikan prototipe backend workflow governance, menerapkan FSM pada monitoring, ticket, dan finance, menerapkan policy layer, menyediakan SLA Tracking, mencatat audit trail, menyediakan event realtime, dan mengevaluasi sistem melalui pengujian.

### 1.5 Batasan Masalah

Subbab ini membatasi klaim agar aman secara akademik. Jelaskan bahwa sistem adalah prototipe berbasis studi kasus, bukan sistem resmi MRT/JICA. Integrasi eksternal antar instansi belum menjadi fokus. WebSocket hanya digunakan sebagai sinyal realtime untuk refetch data, sedangkan REST tetap menjadi sumber data utama. Redis bersifat optional dan sistem tetap berjalan dengan graceful degradation.

### 1.6 Kontribusi Penelitian

Subbab ini menuliskan kontribusi teknis jurnal. Kontribusi utama mencakup model workflow governance berbasis FSM, policy layer terpusat, SLA Tracking, WBS integrity validation, finance concurrency control, audit log, secure upload metadata, dashboard cache, outbox pattern, WebSocket realtime delivery, serta unit test dan contract test.

---

## II. Tinjauan Pustaka

### 2.1 Sistem Informasi Monitoring Proyek

Subbab ini membahas konsep sistem monitoring proyek: progress tracking, baseline vs actual progress, dashboard status, issue tracking, dan approval flow. Hubungkan teori ini dengan kebutuhan proyek infrastruktur yang membutuhkan visibilitas lintas organisasi.

### 2.2 Finite State Machine dalam Workflow

Subbab ini menjelaskan FSM sebagai model formal untuk proses bisnis stateful. Tuliskan komponen FSM seperti state, input, transition function, initial state, dan final state. Jelaskan mengapa FSM cocok untuk mencegah status loncat, invalid transition, dan proses yang tidak deterministik.

### 2.3 SLA Tracking

Subbab ini membahas SLA sebagai mekanisme pengukuran durasi proses. Jelaskan konsep `enteredAt`, `dueAt`, `resolvedAt`, remaining time, status warning, dan violation. Kaitkan SLA dengan kebutuhan mendeteksi bottleneck pada ticket atau hambatan proyek.

### 2.4 RBAC, ABAC, dan Organization-Centric Access Control

Subbab ini menjelaskan teori access control. Bahas perbedaan RBAC dan ABAC, lalu jelaskan mengapa sistem multi-stakeholder membutuhkan model hybrid yang mempertimbangkan user, organisasi, role organisasi, member role, dan relasi organisasi terhadap project/contract.

### 2.5 Audit Trail dan Accountability

Subbab ini membahas pentingnya audit trail pada sistem lintas lembaga. Jelaskan bahwa sistem perlu mencatat actor, organization, action, entity, request id, IP, user agent, timestamp, serta before/after data jika diperlukan. Gunakan istilah audit trail terpusat atau append-oriented audit log, bukan immutable audit log kecuali sudah ada proteksi database khusus.

### 2.6 Event-Driven Architecture dan Outbox Pattern

Subbab ini menjelaskan masalah dual-write antara transaksi database dan pengiriman event. Jelaskan bagaimana outbox pattern menyimpan event di database terlebih dahulu, lalu worker memproses event tersebut untuk dikirim ke channel realtime. Tekankan bahwa pola ini meningkatkan reliability event delivery.

### 2.7 WebSocket Realtime Delivery

Subbab ini menjelaskan WebSocket sebagai channel komunikasi dua arah yang cocok untuk dashboard monitoring. Jelaskan konsep connect, subscribe topic, ping/pong, dan push event. Tegaskan bahwa WebSocket hanya mengirim sinyal event, sedangkan data detail tetap diambil ulang melalui REST API.

### 2.8 Secure File Upload

Subbab ini membahas risiko upload dokumen: MIME spoofing, file kosong, ukuran terlalu besar, dan dokumen tanpa metadata. Jelaskan solusi berupa content sniffing, checksum, metadata file, uploader identity, dan entity binding.

---

## III. Metode Penelitian

### 3.1 Desain Penelitian

Subbab ini menjelaskan metode penelitian yang digunakan. Metode yang cocok adalah Research and Development dengan pendekatan prototyping. Tahapannya mencakup analisis kebutuhan, perancangan arsitektur, implementasi backend, integrasi frontend, dan evaluasi melalui pengujian.

### 3.2 Analisis Aktor Lintas Institusi

Subbab ini memetakan aktor sistem. Aktor yang dapat dibahas adalah GOV_SUPERVISOR, LOAN_PROVIDER, EXECUTING_AGENCY, CONTRACTOR, INDEPENDENT_AUDITOR, dan GOV_PARTNER. Jelaskan peran masing-masing dalam konteks proyek ODA/MRT, lalu jelaskan bagaimana role tersebut dinormalisasi menjadi role sistem.

### 3.3 Analisis Kebutuhan Sistem

Subbab ini menuliskan kebutuhan fungsional dan non-fungsional. Kebutuhan fungsional mencakup login, project assignment, WBS, progress submission, validation, ticket tracking, payment approval, upload, dashboard, dan realtime event. Kebutuhan non-fungsional mencakup security, consistency, auditability, performance, maintainability, dan testability.

### 3.4 Perancangan Arsitektur Sistem

Subbab ini menjelaskan arsitektur modular monolith. Bahas layer HTTP router, middleware, controller, service, policy/workflow/audit/outbox, repository SQLC, PostgreSQL, Redis, dan WebSocket. Jelaskan alasan pemilihan modular monolith: lebih sederhana untuk prototipe, transaksi lebih mudah dijaga, tetapi boundary domain tetap jelas.

### 3.5 Perancangan Data dan Module

Subbab ini menjelaskan pembagian module sistem. Module utama adalah IAM, Project, Monitoring, Tracker, Finance, Support, Policy, Workflow, Audit, Outbox, dan Realtime. Jelaskan fungsi setiap module dan relasinya terhadap workflow governance.

### 3.6 Perancangan FSM Monitoring Progress

Subbab ini menjelaskan workflow progress. State yang dapat dibahas adalah `SUBMITTED`, `VALIDATED_ACCEPTED`, dan `VALIDATED_REJECTED`. Jelaskan bahwa contractor submit progress, auditor melakukan validasi, dan progress baru memengaruhi realisasi setelah diterima.

### 3.7 Perancangan FSM Ticket

Subbab ini menjelaskan workflow ticket sebagai representasi hambatan lapangan. State yang dapat dibahas adalah `OPEN`, `ASSIGNED`, `IN_PROGRESS`, `RESOLVED`, dan `CLOSED`. Jelaskan actor yang terlibat serta bagaimana ticket log menjadi riwayat perubahan status.

### 3.8 Perancangan FSM Finance

Subbab ini menjelaskan workflow approval pembayaran. State yang dapat dibahas adalah `SUBMITTED`, `AUDITOR_APPROVED`, `AUDITOR_REJECTED`, `AGENCY_APPROVED`, `AGENCY_REJECTED`, dan `PAID`. Jelaskan approval bertingkat dari contractor, auditor, executing agency, hingga loan provider.

### 3.9 Perancangan SLA Tracking

Subbab ini menjelaskan rancangan SLA record. Bahas field seperti `entityType`, `entityId`, `state`, `ownerOrgId`, `enteredAt`, `dueAt`, dan `resolvedAt`. Jelaskan algoritma sederhana untuk menentukan status `ON_TRACK`, `SLA_WARNING`, `SLA_VIOLATED`, dan `RESOLVED`.

### 3.10 Perancangan Policy Layer

Subbab ini menjelaskan bagaimana authorization dipusatkan. Bahas rule seperti user harus member organisasi, organisasi harus participant project, actor harus sesuai role contract, dan member role harus admin untuk aksi tertentu. Jelaskan manfaatnya untuk mengurangi duplikasi rule di controller.

### 3.11 Perancangan WBS Integrity Validation

Subbab ini menjelaskan WBS sebagai struktur tree. Bahas validasi root weight, leaf weight, sibling weight, self-parent prevention, parent beda contract, cycle detection, orphan task, dan deadline child terhadap parent. Jelaskan mengapa validasi ini penting untuk menjaga akurasi progress.

### 3.12 Perancangan Realtime Delivery

Subbab ini menjelaskan alur event: business transaction, insert outbox event, worker memproses event, publish ke topic, WebSocket client menerima event, lalu frontend refetch REST query. Jelaskan juga topic seperti `project.{projectId}.monitoring`, `project.{projectId}.ticket`, dan `project.{projectId}.finance`.

### 3.13 Strategi Pengujian

Subbab ini menjelaskan jenis pengujian yang digunakan. Bahas unit test service, workflow test, policy test, WBS validation test, auth test, rate limiter test, uploader test, WebSocket test, outbox processor test, dan controller contract test. Jelaskan bahwa pengujian dipakai untuk membuktikan correctness business rules.

---

## IV. Hasil dan Pembahasan

### 4.1 Implementasi Modular Monolith

Subbab ini membahas hasil implementasi arsitektur. Jelaskan bahwa sistem dipisah menjadi module IAM, Project, Monitoring, Tracker, Finance, Support, Policy, Workflow, Audit, Outbox, dan Realtime. Bahas Go, PostgreSQL, SQLC, Redis, JWT, dan WebSocket sebagai stack utama.

### 4.2 Implementasi Organization-Centric Access Control

Subbab ini membahas policy layer. Jelaskan bahwa akses tidak hanya ditentukan oleh role user, tetapi juga organisasi user, role organisasi, member role, dan keterlibatan organisasi pada project/contract. Berikan contoh contractor tidak dapat melakukan approval auditor, dan loan provider hanya dapat disburse pada tahap yang valid.

### 4.3 Implementasi FSM Transition Engine

Subbab ini membahas implementasi transition validation. Jelaskan bagaimana backend menolak invalid transition pada monitoring, ticket, dan finance. Tuliskan contoh bahwa payment request tidak dapat langsung berpindah dari `SUBMITTED` ke `PAID` tanpa approval bertingkat.

### 4.4 Implementasi SLA Tracking untuk Ticket

Subbab ini membahas hasil implementasi SLA pada ticket. Jelaskan bahwa ticket memiliki SLA record, remaining seconds, due date, status SLA, dan flag violation. Hubungkan fitur ini dengan deteksi bottleneck pada organisasi yang sedang memegang tanggung jawab.

### 4.5 Implementasi WBS Integrity Validation

Subbab ini membahas hasil validasi WBS. Jelaskan endpoint validasi WBS, error yang dapat muncul, dan bagaimana frontend dapat menampilkan status valid/invalid. Tekankan bahwa dashboard progress tidak boleh bergantung pada WBS yang struktur dan bobotnya tidak valid.

### 4.6 Implementasi Finance Concurrency Control

Subbab ini membahas proteksi race condition pada payment request. Jelaskan transaction boundary, row lock `FOR UPDATE`, validasi pending request, dan validasi amount terhadap available amount. Hubungkan dengan kebutuhan menjaga konsistensi nilai kontrak pada proyek ODA.

### 4.7 Implementasi Refresh Token Revocation dan Rate Limiting

Subbab ini membahas keamanan auth. Jelaskan refresh token dengan `jti`, logout, revoke-all, Redis blacklist/cutoff, dan rate limit endpoint auth. Hubungkan fitur ini dengan kebutuhan session control pada sistem lintas organisasi.

### 4.8 Implementasi Audit Log Global

Subbab ini membahas pencatatan aktivitas penting. Jelaskan bahwa audit log menyimpan actor, organization, action, entity, request id, IP, user agent, dan timestamp. Tekankan bahwa audit log meningkatkan traceability dan accountability, tetapi hindari klaim immutable jika belum ada proteksi append-only di database.

### 4.9 Implementasi Secure Upload dan File Metadata

Subbab ini membahas upload dokumen sebagai bukti progress, ticket, atau finance. Jelaskan metadata file, content sniffing, checksum, uploader user/org, dan entity binding. Tekankan bahwa file tidak hanya disimpan sebagai URL, tetapi memiliki metadata yang bisa dilacak.

### 4.10 Implementasi Dashboard Cache

Subbab ini membahas optimasi dashboard. Jelaskan Redis read-through cache, TTL singkat, invalidasi saat data domain berubah, dan fallback database ketika Redis tidak tersedia. Tekankan bahwa cache meningkatkan performa tanpa mengubah PostgreSQL sebagai source of truth.

### 4.11 Implementasi Domain Event, Outbox, dan WebSocket

Subbab ini membahas realtime delivery. Jelaskan event domain seperti progress submitted, progress validated, ticket created, ticket resolved, payment approved, dan payment paid. Jelaskan outbox worker, Redis Pub/Sub fanout, topic authorization, dan WebSocket client. Tegaskan bahwa event realtime adalah sinyal untuk refetch data REST.

### 4.12 Implementasi Frontend Pendukung

Subbab ini membahas frontend secara singkat sebagai validasi contract backend. Jelaskan fitur seperti refresh token rotation, logout/revoke-all, rate limit countdown, upload metadata, WBS validation panel, SLA ticket UI, finance FSM action UI, WebSocket realtime client, dan UI error mapping.

### 4.13 Keterkaitan dengan Studi Kasus MRT Jakarta Fase 2A

Subbab ini menjadi jembatan dengan jurnal HI. Jelaskan bahwa MRT Jakarta Fase 2A, ODA Loan Jepang, CP205, koordinasi utilitas, dan stakeholder lintas institusi digunakan sebagai konteks kebutuhan. Tegaskan bahwa sistem adalah prototipe yang merepresentasikan kebutuhan teknis proyek sejenis, bukan sistem resmi yang sudah diintegrasikan.

---

## V. Evaluasi Sistem

### 5.1 Evaluasi Correctness Workflow

Subbab ini menjelaskan hasil pengujian workflow. Bahas bahwa valid transition diterima, invalid transition ditolak, terminal state tidak bisa dilanjutkan sembarangan, dan actor yang salah tidak dapat menjalankan action.

### 5.2 Evaluasi Authorization

Subbab ini menjelaskan pengujian policy layer. Bahas skenario user tanpa project access, organization role salah, member role tidak cukup, dan participant project yang valid. Tujuannya membuktikan boundary akses lintas organisasi berjalan konsisten.

### 5.3 Evaluasi Data Integrity

Subbab ini membahas pengujian integritas data. Bahas WBS cycle, self-parent, sibling weight lebih dari 100, child deadline melewati parent, duplicate validation, dan payment amount yang melebihi available amount.

### 5.4 Evaluasi Security

Subbab ini membahas pengujian keamanan. Bahas refresh token revoked, token tanpa claim wajib, rate limit `429 RATE_LIMIT_EXCEEDED`, logout server-side, dan upload file invalid MIME.

### 5.5 Evaluasi Realtime Delivery

Subbab ini membahas pengujian WebSocket dan outbox. Bahas connect menggunakan `accessToken`, subscribe topic authorized, rejected topic tanpa memutus koneksi, ping/pong, mapping outbox event ke realtime envelope, dan refetch data frontend setelah event diterima.

### 5.6 Evaluasi Build dan Contract

Subbab ini mencatat verifikasi teknis. Backend diverifikasi dengan `go test ./...`, sedangkan frontend diverifikasi dengan `npm run build`. Jelaskan juga contract baru yang telah diadaptasi: upload metadata, ticket SLA, auth logout/revoke-all, rate limit, WBS validation, WebSocket, dan UI error mapping.

---

## VI. Kesimpulan dan Saran

### 6.1 Kesimpulan

Subbab ini merangkum hasil penelitian. Tulis bahwa workflow proyek lintas lembaga dapat dimodelkan dengan FSM, SLA Tracking membantu mengukur bottleneck, Organization-Centric Access Control menjaga boundary antar stakeholder, audit trail dan outbox meningkatkan traceability, WebSocket meningkatkan responsivitas dashboard, dan pengujian backend memperkuat reliability.

### 6.2 Keterbatasan

Subbab ini menuliskan batasan penelitian. Jelaskan bahwa sistem masih berupa prototipe, belum terintegrasi dengan sistem resmi MRT/JICA, belum memiliki API Gateway lintas instansi, WebSocket belum menjadi sumber data utama, audit log belum immutable secara database-level, dan digital signature legal belum diterapkan.

### 6.3 Saran Pengembangan

Subbab ini menuliskan pengembangan lanjutan. Saran yang dapat ditulis: audit log viewer, konfigurasi SLA per project/state/organization, escalation notification, OpenAPI spec, metrics dan tracing, materialized view dashboard, virus scanning upload, session/device management, dan signed approval record untuk finance.

---

## Catatan Klaim Aman

Gunakan frasa berikut agar klaim jurnal tetap aman:

> Sistem yang dikembangkan merupakan prototipe backend workflow governance berbasis studi kasus proyek infrastruktur transnasional seperti MRT Jakarta Fase 2A. Implementasi ini tidak menggantikan sistem resmi instansi, tetapi menunjukkan bagaimana FSM, SLA Tracking, access control, audit trail, dan realtime event delivery dapat digunakan untuk meningkatkan transparansi serta traceability proses.

Hindari klaim berikut:

- sudah terintegrasi resmi dengan JICA, Pemprov, MRT Jakarta, atau kontraktor;
- event terkirim tanpa delay;
- audit log immutable secara absolut;
- digital signature legal;
- sistem menyelesaikan red tape sepenuhnya;
- WebSocket menggantikan REST API sebagai source of truth.
