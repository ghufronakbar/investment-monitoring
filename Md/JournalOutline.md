# Journal Outline: Fokus Informatika

Dokumen ini adalah versi fokus dari outline jurnal. Tujuannya adalah menjaga pembahasan tetap kuat secara Informatika tanpa melebar ke semua improvement teknis yang sudah dibuat di project.

Fokus utama jurnal:

> Backend workflow governance untuk proyek infrastruktur transnasional berbasis ODA dengan empat pilar utama: Finite State Machine, Organization-Centric Access Control, SLA Tracking + Auditability, dan Realtime Event Delivery.

Fitur seperti dashboard cache, secure upload, refresh token revocation, rate limiting, WBS integrity validation, finance concurrency control, dan UI error mapping tetap bernilai secara engineering, tetapi tidak perlu menjadi subbab utama. Fitur tersebut cukup disebut singkat sebagai supporting mechanism jika relevan.

---

## 1. Arah Utama Jurnal

### 1.1 Niche Informatika

Niche yang disarankan:

> Perancangan dan implementasi backend workflow governance proyek infrastruktur transnasional berbasis ODA menggunakan Finite State Machine, Organization-Centric Access Control, SLA Tracking, Audit Trail, dan Realtime Event Delivery.

Nilai Informatika utamanya:

- proses lintas lembaga dimodelkan sebagai workflow formal;
- akses tidak hanya berdasarkan user role, tetapi juga organisasi dan relasi ke project;
- bottleneck proses diukur melalui SLA;
- accountability didukung audit trail;
- asimetri informasi dikurangi melalui event realtime berbasis outbox dan WebSocket;
- correctness sistem dapat diuji melalui unit test dan contract test.

### 1.2 Hubungan dengan Jurnal Hubungan Internasional

Jurnal HI menjelaskan mengapa sistem ini penting dari sisi diplomasi, ODA Loan, red tape, dan international trust. Jurnal Informatika menjelaskan bagaimana kebutuhan tersebut diwujudkan secara teknis.

Korelasi utama:

- red tape dan saling tunggu dijawab dengan FSM workflow;
- bottleneck birokrasi dijawab dengan SLA Tracking;
- trust dan accountability dijawab dengan policy layer dan audit trail;
- asimetri informasi dijawab dengan outbox pattern dan WebSocket realtime delivery.

Catatan klaim:

> Sistem ini adalah prototipe berbasis studi kasus MRT Jakarta Fase 2A, bukan sistem resmi yang telah terintegrasi dengan MRT Jakarta, Pemprov DKI, JICA, atau kontraktor terkait.

### 1.3 Judul yang Direkomendasikan

> Perancangan Backend Workflow Governance Proyek Infrastruktur Transnasional Berbasis ODA Menggunakan Finite State Machine, SLA Tracking, dan Realtime Event Delivery: Studi Kasus MRT Jakarta Fase 2A

Alternatif:

1. Model Backend Workflow Governance Proyek Infrastruktur Lintas Lembaga Berbasis FSM dan Organization-Centric Access Control
2. Implementasi Workflow Governance Anti-Red Tape pada Monitoring Proyek Infrastruktur Menggunakan FSM, SLA Tracking, dan Audit Trail
3. Arsitektur Backend Monitoring Proyek Transnasional Berbasis Policy Layer, Outbox Pattern, dan WebSocket Realtime Delivery

---

## 2. Abstrak yang Disarankan

Abstrak sebaiknya ringkas dan tidak menyebut semua fitur teknis.

Isi abstrak:

- latar belakang proyek infrastruktur transnasional berbasis ODA;
- masalah koordinasi multi-stakeholder, red tape, asimetri informasi, dan kurangnya traceability;
- metode pengembangan prototipe backend;
- empat komponen inti: FSM, policy layer, SLA + audit trail, outbox + WebSocket;
- evaluasi melalui workflow test, policy test, SLA/tracker test, WebSocket/outbox test, dan build/contract verification;
- hasil berupa backend yang dapat memvalidasi transisi, membatasi akses, mencatat aktivitas penting, dan mengirim event realtime sebagai sinyal refetch.

Kata kunci:

- workflow governance;
- finite state machine;
- SLA tracking;
- access control;
- audit trail;
- outbox pattern;
- WebSocket;
- project monitoring.

---

## 3. Outline Jurnal Final

## I. Pendahuluan

### 1.1 Latar Belakang

Tulis konteks MRT Jakarta Fase 2A sebagai contoh proyek infrastruktur transnasional berbasis ODA. Fokuskan pada persoalan Informatika: koordinasi data, status pekerjaan, validasi progress, pelaporan hambatan, approval bertingkat, dan kebutuhan visibility bagi banyak organisasi.

### 1.2 Identifikasi Masalah

Tulis masalah inti:

- workflow manual sulit dikontrol;
- status proses bisa tidak konsisten;
- keterlambatan tidak selalu terukur;
- akses data lintas organisasi rawan salah izin;
- dashboard membutuhkan pembaruan cepat;
- sistem harus tetap menjaga REST/database sebagai source of truth.

### 1.3 Rumusan Masalah

Rumusan masalah:

1. Bagaimana memodelkan workflow monitoring proyek lintas lembaga menggunakan FSM?
2. Bagaimana menerapkan access control berbasis organisasi dan role?
3. Bagaimana SLA tracking dan audit trail meningkatkan traceability proses?
4. Bagaimana outbox pattern dan WebSocket mendukung realtime delivery tanpa menggantikan REST?
5. Bagaimana pengujian backend membuktikan correctness workflow dan authorization?

### 1.4 Tujuan Penelitian

Tujuan penelitian:

- merancang backend workflow governance;
- menerapkan FSM pada workflow monitoring, ticket, dan finance;
- menerapkan organization-centric policy layer;
- menerapkan SLA tracking dan audit trail;
- menerapkan realtime event delivery berbasis outbox dan WebSocket;
- mengevaluasi sistem melalui pengujian teknis.

### 1.5 Batasan Masalah

Batasan:

- sistem adalah prototipe akademik;
- studi kasus MRT/JICA digunakan sebagai konteks kebutuhan;
- tidak ada klaim integrasi resmi dengan instansi;
- WebSocket hanya menjadi sinyal refetch;
- REST dan PostgreSQL tetap menjadi source of truth;
- fitur supporting tidak menjadi fokus pembahasan.

### 1.6 Kontribusi Penelitian

Kontribusi cukup ditulis empat:

- FSM workflow governance;
- organization-centric access control;
- SLA tracking dan auditability;
- outbox-based realtime event delivery.

---

## II. Tinjauan Pustaka

### 2.1 Sistem Informasi Monitoring Proyek

Bahas konsep monitoring proyek, progress tracking, issue tracking, approval flow, dan dashboard status.

### 2.2 Finite State Machine untuk Workflow

Bahas komponen FSM dan alasan FSM cocok untuk workflow proyek yang memiliki state, actor, dan transition rule.

### 2.3 Access Control pada Sistem Multi-Stakeholder

Bahas RBAC, ABAC, dan kebutuhan model organization-centric pada sistem lintas organisasi.

### 2.4 SLA Tracking dan Audit Trail

Bahas SLA sebagai pengukuran durasi proses dan audit trail sebagai mekanisme accountability.

### 2.5 Event-Driven Architecture dan WebSocket

Bahas outbox pattern, event delivery, topic subscription, dan WebSocket sebagai channel realtime notification.

---

## III. Metode Penelitian

### 3.1 Desain Penelitian

Gunakan Research and Development dengan pendekatan prototyping. Jelaskan tahapan dari analisis kebutuhan sampai evaluasi.

### 3.2 Analisis Aktor Lintas Institusi

Bahas aktor: government supervisor, loan provider, executing agency, contractor, independent auditor, dan government partner.

### 3.3 Perancangan Arsitektur Backend

Bahas modular monolith, Go, PostgreSQL, SQLC, Redis, REST API, outbox worker, dan WebSocket secara ringkas sebagai fondasi teknis.

### 3.4 Perancangan FSM Workflow Governance

Bahas tiga workflow:

- monitoring progress;
- ticket hambatan;
- finance approval.

Untuk masing-masing, jelaskan state, input transition, actor, dan invalid transition.

### 3.5 Perancangan Policy Layer

Bahas aturan access control: user membership, organization role, project participation, contract role, dan actor guard.

### 3.6 Perancangan SLA Tracking dan Auditability

Bahas SLA record, owner organization, due time, violation status, dan audit log untuk mencatat aksi penting.

### 3.7 Perancangan Realtime Event Delivery

Bahas alur: domain action -> outbox event -> worker -> topic -> WebSocket -> frontend refetch REST query.

### 3.8 Strategi Pengujian

Bahas pengujian yang langsung relevan: workflow test, policy test, SLA/tracker test, audit/outbox test, WebSocket test, dan controller contract test.

---

## IV. Hasil dan Pembahasan

### 4.1 Implementasi Arsitektur Backend Workflow Governance

Bahas modular monolith dan module utama yang mendukung workflow governance. Sebut stack teknis secara ringkas, tetapi jangan masuk terlalu dalam ke fitur supporting.

### 4.2 Implementasi FSM Transition Engine

Bahas implementasi FSM pada monitoring, ticket, dan finance. Tunjukkan contoh transisi valid/invalid dan manfaatnya untuk mencegah status loncat.

### 4.3 Implementasi Organization-Centric Policy Layer

Bahas implementasi authorization berbasis organisasi. Tunjukkan bagaimana actor berbeda memiliki kewenangan berbeda pada project/contract yang sama.

### 4.4 Implementasi SLA Tracking dan Auditability

Gabungkan SLA dan audit log dalam satu subbab. SLA menunjukkan durasi dan bottleneck, sedangkan audit log menunjukkan siapa melakukan apa, kapan, dan terhadap entity apa.

### 4.5 Implementasi Realtime Event Delivery

Bahas outbox event, worker, topic authorization, Redis Pub/Sub fanout, dan WebSocket. Tegaskan bahwa event realtime dipakai untuk refetch REST data.

### 4.6 Keterkaitan dengan Studi Kasus MRT Jakarta Fase 2A

Bahas bagaimana empat pilar tersebut relevan dengan proyek seperti MRT Fase 2A: lintas organisasi, pemberi dana asing, validasi progress, hambatan lapangan, dan approval bertingkat.

---

## V. Evaluasi Sistem

### 5.1 Evaluasi Workflow Correctness

Bahas hasil pengujian valid transition, invalid transition, terminal state, dan actor yang salah.

### 5.2 Evaluasi Authorization

Bahas hasil pengujian policy layer untuk memastikan boundary akses lintas organisasi.

### 5.3 Evaluasi SLA dan Auditability

Bahas hasil pengujian SLA/tracker dan audit log.

### 5.4 Evaluasi Realtime Delivery

Bahas WebSocket connect, subscribe topic, rejected topic, ping/pong, outbox mapping, dan frontend refetch.

### 5.5 Evaluasi Build dan Contract

Bahas `go test ./...` dan `npm run build`. Contract frontend yang dibahas cukup yang relevan dengan fokus utama: SLA ticket, FSM UX, WebSocket client, dan workflow error handling.

---

## VI. Kesimpulan dan Saran

### 6.1 Kesimpulan

Simpulkan bahwa FSM, organization-centric access control, SLA + audit trail, dan realtime event delivery dapat menjadi model backend workflow governance untuk proyek infrastruktur lintas lembaga.

### 6.2 Keterbatasan

Tuliskan bahwa sistem masih prototipe, belum terintegrasi resmi, belum memakai API Gateway lintas instansi, belum benchmark produksi, dan WebSocket belum menjadi source of truth.

### 6.3 Saran Pengembangan

Saran yang tetap relevan:

- konfigurasi SLA per project/state;
- audit log viewer;
- notification center;
- OpenAPI contract;
- observability metrics;
- hardening deployment.

---

## 4. Fitur yang Tidak Perlu Menjadi Subbab Utama

Fitur berikut cukup disebut singkat sebagai pendukung:

| Fitur | Posisi yang Disarankan |
| --- | --- |
| Dashboard Cache | Disebut singkat di arsitektur/performance, bukan subbab utama |
| Secure Upload Metadata | Disebut sebagai supporting evidence/document traceability |
| Refresh Token Revocation | Disebut singkat di security, bukan fokus jurnal |
| Rate Limiting Auth | Disebut singkat di security/evaluasi |
| WBS Integrity Validation | Disebut singkat sebagai bagian data integrity pendukung FSM monitoring |
| Finance Concurrency Control | Disebut singkat dalam FSM finance/evaluasi consistency |
| UI Error Mapping | Tidak perlu dibahas di jurnal, cukup frontend implementation detail |
| Frontend Contract Update | Cukup disebut di evaluasi build/contract |

Alasan:

> Jika semua improvement dijadikan subbab utama, jurnal akan melebar menjadi laporan engineering umum. Agar kontribusi akademiknya tajam, pembahasan utama harus tetap berada pada workflow governance, access control, SLA/auditability, dan realtime delivery.

---

## 5. Catatan Klaim Aman

Gunakan klaim:

- prototipe backend workflow governance;
- studi kasus MRT Jakarta Fase 2A sebagai konteks kebutuhan;
- REST tetap source of truth;
- WebSocket sebagai realtime delivery signal;
- audit log sebagai fondasi accountability;
- Redis optional dan sistem bisa graceful degradation.

Hindari klaim:

- sistem resmi MRT/JICA/Pemprov;
- realtime tanpa delay;
- audit log immutable absolut;
- digital signature legal;
- sistem menyelesaikan red tape sepenuhnya;
- WebSocket menggantikan REST API.

Kalimat aman:

> Sistem yang dikembangkan merupakan prototipe backend workflow governance berbasis studi kasus proyek infrastruktur transnasional seperti MRT Jakarta Fase 2A. Implementasi ini tidak menggantikan sistem resmi instansi, tetapi menunjukkan bagaimana FSM, organization-centric access control, SLA tracking, audit trail, dan realtime event delivery dapat digunakan untuk meningkatkan transparansi serta traceability proses.
