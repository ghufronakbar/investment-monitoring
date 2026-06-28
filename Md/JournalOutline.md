# Journal Outline: Fokus Informatika

Dokumen ini menyusun outline jurnal dari sudut pandang Informatika berdasarkan dokumen awal `AdviceFromGemini.md`, tanggapan `AnswerFromCodex.md`, dan improvement backend/frontend yang sudah dikerjakan pada project JICA Investment Monitoring System.

Fokus tulisan disarankan tidak hanya menjadi "dashboard monitoring proyek", tetapi diarahkan menjadi pembahasan sistem informasi workflow governance lintas lembaga dengan studi kasus proyek infrastruktur transnasional berbasis ODA, seperti MRT Jakarta Fase 2A. Dengan begitu, bagian IT memiliki novelty yang jelas: pemodelan proses bisnis stateful, kontrol akses multi-organisasi, integrity validation, auditability, event-driven realtime delivery, dan evaluasi teknis.

---

## 1. Arah Utama Jurnal

### 1.1 Niche Informatika

Niche yang paling kuat:

> Perancangan dan implementasi backend workflow governance proyek infrastruktur transnasional berbasis ODA menggunakan Finite State Machine, SLA Tracking, Organization-Centric Access Control, Audit Trail, dan Realtime Event Delivery.

Alasan niche ini kuat:

- project bukan CRUD sederhana, tetapi sistem workflow lintas aktor;
- domain proyek infrastruktur memiliki banyak state, approval, validasi, dan bottleneck;
- sistem sudah memiliki implementasi nyata pada backend, bukan hanya desain konseptual;
- improvement yang sudah dibuat dapat dibahas sebagai kontribusi rekayasa perangkat lunak;
- konteks HI tetap bisa menjadi latar belakang dan studi kasus, tetapi novelty tulisan berada pada model sistem informasinya.

### 1.2 Pembeda dengan Jurnal Hubungan Internasional

Jurnal teman yang fokus HI dapat membahas:

- diplomasi ekonomi RI-Jepang;
- ODA Loan Jepang;
- dampak red tape terhadap international trust;
- dinamika MRT Jakarta Fase 2A dan CP205.

Jurnal Informatika sebaiknya membahas:

- bagaimana proses lintas lembaga dimodelkan menjadi state machine;
- bagaimana sistem memastikan hanya aktor yang tepat bisa melakukan transisi;
- bagaimana durasi tiap state dipantau melalui SLA;
- bagaimana audit log menjaga accountability;
- bagaimana domain event dan WebSocket membuat dashboard lebih responsif;
- bagaimana pengujian backend membuktikan aturan workflow berjalan benar.

### 1.2.1 Titik Temu dengan Jurnal Hubungan Internasional

Jurnal HI dapat diposisikan sebagai penjelasan kebutuhan bisnis dan kebijakan: mengapa proyek transnasional seperti MRT Jakarta Fase 2A membutuhkan transparansi, trust-building, dan mitigasi red tape. Jurnal Informatika kemudian menjawab sisi teknis: bagaimana kebutuhan tersebut dapat diwujudkan melalui rancangan backend yang formal, aman, dan dapat diuji.

Korelasi yang bisa ditulis:

- red tape dan saling tunggu dijawab dengan FSM Transition Engine dan SLA Tracking;
- asimetri informasi dijawab dengan outbox pattern dan WebSocket realtime delivery;
- international trust dijawab dengan audit trail, policy layer, dan akses berbasis organisasi;
- risiko miskomunikasi lintas institusi dijawab dengan topic authorization, role boundary, dan workflow contract yang eksplisit.

Catatan klaim:

> Sistem ini diposisikan sebagai prototipe berbasis studi kasus, bukan sistem resmi yang telah terintegrasi dengan MRT Jakarta, Pemprov DKI, JICA, atau kontraktor terkait.

### 1.3 Usulan Judul

Judul utama yang direkomendasikan:

> Perancangan Backend Workflow Governance Proyek Infrastruktur Transnasional Berbasis ODA Menggunakan Finite State Machine, SLA Tracking, dan Realtime Event Delivery: Studi Kasus MRT Jakarta Fase 2A

Alternatif:

1. Model Sistem Informasi Monitoring Proyek Infrastruktur Transnasional Berbasis FSM, SLA Tracking, dan Organization-Centric RBAC
2. Implementasi Workflow Governance Anti-Red Tape pada Monitoring Proyek Infrastruktur ODA Menggunakan Finite State Machine dan Audit Trail
3. Arsitektur Backend untuk Transparansi Proyek Infrastruktur Multi-Stakeholder Berbasis Policy Layer, Outbox Pattern, dan WebSocket Realtime Delivery
4. Perancangan Sistem Monitoring Digital Proyek MRT Berbasis Workflow Engine, SLA Tracking, dan Event-Driven Architecture

---

## 2. Abstrak yang Disarankan

Proyek infrastruktur transnasional berbasis ODA seperti MRT Jakarta Fase 2A melibatkan banyak pemangku kepentingan, seperti pemerintah lokal, executing agency, penyedia pendanaan asing, kontraktor, auditor independen, dan instansi pendukung. Kompleksitas koordinasi tersebut berisiko menimbulkan red tape, asimetri informasi, keterlambatan validasi, dan lemahnya akuntabilitas proses. Penelitian ini merancang dan mengimplementasikan prototipe backend sistem monitoring proyek infrastruktur multi-stakeholder dengan pendekatan Finite State Machine, SLA Tracking, Organization-Centric Access Control, audit trail, dan event-driven realtime delivery. Sistem dikembangkan sebagai modular monolith menggunakan Go, PostgreSQL, SQLC, Redis, JWT, WebSocket, dan outbox pattern. Workflow utama yang dimodelkan mencakup validasi progress, ticket penyelesaian hambatan, approval pembayaran, validasi integritas WBS, serta pengiriman event realtime ke dashboard. Hasil pengujian menunjukkan bahwa sistem mampu menolak transisi status tidak valid, membatasi akses berdasarkan organisasi dan role, mencatat aktivitas penting ke audit log, memvalidasi struktur WBS, mengendalikan concurrency pada pembayaran, serta mengirim sinyal pembaruan realtime tanpa menjadikan WebSocket sebagai sumber data utama. Penelitian ini menunjukkan bahwa model backend workflow governance dapat meningkatkan transparansi, traceability, dan kesiapan integrasi digital pada monitoring proyek infrastruktur lintas lembaga.

Kata kunci:

- workflow governance;
- finite state machine;
- SLA tracking;
- RBAC;
- audit trail;
- outbox pattern;
- WebSocket;
- project monitoring.

---

## 3. Struktur Artikel Jurnal

## I. Pendahuluan

### 1.1 Latar Belakang

Bahas konteks proyek infrastruktur besar seperti MRT Jakarta Fase 2A sebagai contoh domain yang melibatkan koordinasi lintas lembaga dan lintas negara. Dalam konteks Informatika, masalah utama bukan diplomasi itu sendiri, tetapi kompleksitas aliran data, validasi, approval, dan transparansi status antar organisasi.

Poin yang dapat ditulis:

- proyek infrastruktur memiliki banyak aktor dengan kewenangan berbeda;
- proyek berbasis ODA memiliki kebutuhan pelaporan yang tidak hanya lokal, tetapi juga berkaitan dengan pemberi dana asing;
- proses manual berbasis rapat, dokumen, atau komunikasi informal rentan menimbulkan keterlambatan;
- kasus seperti koordinasi pekerjaan sipil, integrasi sistem, pemindahan utilitas, dan paket kontrak seperti CP205 dapat dipakai sebagai latar domain;
- status pekerjaan sering tidak cukup hanya "sedang berjalan", karena perlu diketahui siapa pemegang tanggung jawab, sejak kapan status aktif, dan apakah sudah melewati SLA;
- sistem informasi dibutuhkan untuk mengubah proses birokrasi menjadi workflow digital yang terukur dan dapat diaudit.

### 1.2 Identifikasi Masalah

Rumusan masalah teknis:

1. Bagaimana memodelkan proses monitoring proyek lintas lembaga menjadi workflow formal berbasis state dan transition?
2. Bagaimana membatasi aksi setiap pengguna berdasarkan organisasi, role, dan keterlibatan pada proyek?
3. Bagaimana mendeteksi keterlambatan proses melalui SLA Tracking?
4. Bagaimana menjaga integritas data WBS, progress, ticket, dan payment request?
5. Bagaimana menyediakan pembaruan dashboard secara realtime tanpa mengorbankan konsistensi REST API?
6. Bagaimana membuktikan reliability backend melalui unit test dan contract test?

### 1.3 Tujuan Penelitian

Tujuan penelitian:

- merancang model backend workflow governance untuk proyek multi-stakeholder dengan studi kasus proyek infrastruktur transnasional;
- menerapkan FSM pada workflow monitoring, tracker, dan finance;
- menerapkan Organization-Centric Access Control untuk menjaga boundary antar lembaga;
- menerapkan SLA Tracking untuk mendeteksi bottleneck proses;
- menerapkan audit log dan outbox pattern untuk accountability dan realtime delivery;
- mengevaluasi sistem melalui pengujian service, policy, workflow, controller contract, dan helper teknis.

### 1.4 Batasan Masalah

Batasan yang disarankan:

- sistem berupa prototipe backend dan frontend pendukung, bukan sistem produksi resmi MRT/JICA;
- data proyek digunakan sebagai konteks studi kasus, bukan klaim integrasi langsung dengan sistem pemerintah/JICA;
- MRT Jakarta Fase 2A dan ODA Loan Jepang dipakai sebagai konteks kebutuhan, sedangkan implementasi sistem tetap bersifat prototipe akademik;
- integrasi eksternal seperti API Gateway antar instansi belum menjadi fokus implementasi;
- WebSocket digunakan sebagai sinyal invalidasi/refetch, sedangkan REST tetap menjadi source of truth;
- Redis bersifat optional dan sistem tetap berjalan dengan graceful degradation ketika Redis tidak tersedia.

### 1.5 Kontribusi Penelitian

Kontribusi teknis:

- model workflow governance berbasis FSM untuk proses multi-aktor;
- policy layer terpusat untuk authorization berbasis organisasi;
- SLA Tracking untuk entity ticket;
- validasi integritas WBS berbentuk tree;
- finance concurrency control dengan transaction dan row lock;
- audit log global untuk write operation penting;
- outbox pattern dan WebSocket realtime delivery;
- secure upload metadata dengan content sniffing dan checksum;
- unit test dan contract test untuk menjaga correctness.

---

## II. Tinjauan Pustaka

### 2.1 Sistem Informasi Monitoring Proyek

Bahas konsep sistem monitoring proyek:

- progress tracking;
- baseline vs actual progress;
- dashboard status;
- issue/ticket tracking;
- approval dan validation flow.

Hubungkan dengan module project:

- `Project`;
- `Monitoring`;
- `Tracker`;
- `Finance`;
- `Support Dashboard`.

### 2.2 Finite State Machine dalam Workflow

Bahas FSM sebagai model formal:

```text
FSM = (S, I, f, s0, F)
```

Keterangan:

- `S`: himpunan state;
- `I`: input/event yang memicu transisi;
- `f`: fungsi transisi;
- `s0`: state awal;
- `F`: final/terminal state.

Contoh penerapan di project:

- progress validation;
- ticket lifecycle;
- payment approval lifecycle.

Nilai akademik:

- FSM membuat workflow lebih eksplisit;
- invalid transition dapat ditolak secara deterministik;
- status terminal dapat didefinisikan jelas;
- workflow dapat diuji dengan unit test.

### 2.3 SLA Tracking

Bahas SLA sebagai mekanisme pengukuran waktu pada state aktif:

- setiap entity masuk ke state tertentu pada `enteredAt`;
- sistem menentukan `dueAt`;
- status SLA dihitung dari waktu sekarang terhadap due time;
- state bisa dikategorikan `ON_TRACK`, `SLA_WARNING`, `SLA_VIOLATED`, atau `RESOLVED`.

Konteks domain:

- ticket yang terlalu lama berada di satu instansi menjadi indikator bottleneck;
- dashboard SLA membantu supervisor melihat hambatan administratif secara objektif.

### 2.4 RBAC, ABAC, dan Organization-Centric Access Control

Bahas perbedaan:

- RBAC: akses berdasarkan role;
- ABAC: akses berdasarkan atribut;
- Organization-Centric Access Control: akses berdasarkan kombinasi user, organisasi, role organisasi, member role, dan relasi organisasi terhadap project/contract.

Contoh aturan:

- hanya `GOV_SUPERVISOR` admin yang dapat membuat project;
- hanya contractor yang dapat submit progress;
- hanya auditor yang dapat validasi progress;
- hanya executing agency yang dapat assign ticket;
- hanya loan provider yang dapat melakukan disbursement.

### 2.5 Audit Trail dan Accountability

Audit trail diperlukan karena sistem melibatkan keputusan lintas lembaga.

Bahas field penting:

- actor user;
- actor organization;
- action;
- entity type;
- entity id;
- before/after JSON;
- request id;
- IP;
- user agent;
- timestamp.

Nilai IT:

- traceability;
- non-repudiation secara aplikasi;
- forensic debugging;
- accountability antar organisasi.

### 2.6 Event-Driven Architecture dan Outbox Pattern

Bahas masalah dual-write:

- data utama masuk database;
- event realtime harus terkirim;
- jika publish event langsung gagal setelah commit, event bisa hilang.

Outbox pattern menyelesaikan masalah dengan:

- menyimpan event ke tabel `outbox_events` dalam transaksi yang sama;
- worker membaca event pending;
- worker mengirim event ke WebSocket/Redis fanout;
- status event ditandai processed atau retry.

### 2.7 WebSocket Realtime Delivery

Bahas WebSocket sebagai channel delivery:

- koneksi persistent;
- client subscribe topic;
- server push event;
- cocok untuk dashboard yang membutuhkan update cepat.

Catatan penting untuk jurnal:

> WebSocket tidak menggantikan REST API. WebSocket hanya mengirim sinyal event, sedangkan data detail tetap diambil ulang melalui REST agar konsistensi tetap terjaga.

### 2.8 Secure File Upload

Bahas risiko upload:

- MIME spoofing;
- file kosong;
- file terlalu besar;
- dokumen tanpa metadata;
- file tidak terikat entity bisnis.

Solusi yang sudah dibuat:

- content sniffing;
- batas ukuran file;
- metadata file;
- checksum SHA-256;
- uploaded user/org;
- optional entity binding.

---

## III. Metode Penelitian

### 3.1 Desain Penelitian

Metode yang cocok:

> Research and Development dengan pendekatan prototyping.

Tahapan:

1. analisis kebutuhan domain multi-stakeholder pada studi kasus proyek infrastruktur transnasional;
2. perancangan model workflow;
3. implementasi backend modular monolith;
4. integrasi frontend untuk validasi contract;
5. pengujian unit, service, policy, workflow, dan controller contract;
6. analisis hasil berdasarkan correctness, access control, auditability, dan realtime readiness.

### 3.2 Analisis Aktor Lintas Institusi

Aktor domain:

- `GOV_SUPERVISOR`: pengawas pemerintah;
- `LOAN_PROVIDER`: penyedia pendanaan;
- `EXECUTING_AGENCY`: pelaksana proyek;
- `CONTRACTOR`: pelaksana pekerjaan;
- `INDEPENDENT_AUDITOR`: validator progress;
- `GOV_PARTNER`: instansi pendukung.

Dalam konteks studi kasus MRT/ODA, aktor tersebut dapat dipetakan secara konseptual sebagai:

- pemerintah pusat atau lokal sebagai pengawas dan pembuat kebijakan;
- executing agency seperti operator/pelaksana proyek;
- pemberi dana asing sebagai loan provider;
- kontraktor lokal atau asing sebagai pelaksana pekerjaan;
- auditor/konsultan independen sebagai validator progress;
- instansi pendukung seperti pemilik utilitas, transportasi, pertanahan, dan perizinan.

Pembahasan IT:

- setiap aktor memiliki boundary akses berbeda;
- satu user dapat berada dalam organisasi tertentu;
- organisasi berpartisipasi pada project melalui contract/assignment;
- policy layer menentukan apakah action diizinkan.
- terminologi lintas budaya dan institusi perlu dinormalisasi ke role sistem agar hierarki pelaporan lokal dan pihak asing dapat diproses tanpa bentrok.

### 3.3 Perancangan Arsitektur Sistem

Arsitektur yang dapat dibahas:

```text
Client Web
   |
REST API + WebSocket
   |
HTTP Router + Middleware
   |
Controller Layer
   |
Service Layer
   |
Policy / Workflow / Audit / Outbox
   |
Repository sqlc
   |
PostgreSQL

Redis:
- refresh token revocation
- rate limit
- dashboard cache
- WebSocket fanout
```

Karakter arsitektur:

- modular monolith;
- Go sebagai bahasa backend terkompilasi yang cocok untuk service API konkuren;
- type-safe persistence dengan SQLC agar query SQL tetap eksplisit namun hasilnya tervalidasi pada compile time;
- PostgreSQL sebagai source of truth;
- Redis sebagai acceleration layer;
- WebSocket sebagai realtime delivery;
- outbox worker sebagai event processor.

Catatan pembahasan:

> Pemilihan Go, PostgreSQL, dan SQLC dapat dijelaskan sebagai strategi untuk menjaga predictable performance, type-safety, dan kontrol eksplisit terhadap query. Namun, jika tidak ada benchmark formal, hindari klaim performa absolut seperti "paling cepat" atau "tanpa delay".

### 3.4 Perancangan Data dan Module

Module utama:

- `IAM`: auth, organization, membership;
- `Project`: project, contract, participant assignment;
- `Monitoring`: WBS, progress log, validation;
- `Tracker`: ticket dan ticket log;
- `Finance`: payment request dan approval;
- `Support`: dashboard, upload, SLA;
- `Policy`: authorization;
- `Workflow`: FSM transition;
- `Audit`: audit log;
- `Outbox`: domain event;
- `Realtime`: WebSocket.

Tabel baru dari improvement:

- `sla_records`;
- `audit_logs`;
- `outbox_events`;
- `files`.

### 3.5 Perancangan Workflow FSM

#### 3.5.1 FSM Monitoring Progress

State:

```text
SUBMITTED
VALIDATED_ACCEPTED
VALIDATED_REJECTED
```

Input:

```text
submit_progress
accept_progress
reject_progress
```

Actor:

- contractor submit progress;
- auditor melakukan validasi.

Nilai yang dibahas:

- progress tidak langsung masuk realisasi sebelum accepted;
- validation mencegah klaim sepihak;
- snapshot task pada progress log menjaga histori saat WBS berubah.

#### 3.5.2 FSM Ticket

State:

```text
OPEN
ASSIGNED
IN_PROGRESS
RESOLVED
CLOSED
```

Input:

```text
create_ticket
assign_ticket
start_resolution
resolve_ticket
close_ticket
```

Actor:

- contractor membuat ticket;
- executing agency assign dan close;
- gov partner update penyelesaian;
- supervisor/JICA melihat dashboard.

Nilai yang dibahas:

- ticket menjadi representasi hambatan lapangan;
- latest ticket log menentukan current status;
- SLA record mengukur lamanya ticket berada pada state tertentu.

#### 3.5.3 FSM Finance

State:

```text
SUBMITTED
AUDITOR_APPROVED
AUDITOR_REJECTED
AGENCY_APPROVED
AGENCY_REJECTED
PAID
```

Input:

```text
submit_payment_request
auditor_approve
auditor_reject
agency_approve
agency_reject
loan_provider_disburse
```

Actor:

- contractor submit request;
- auditor approve/reject tahap teknis;
- executing agency approve/reject administratif;
- loan provider melakukan disbursement.

Nilai yang dibahas:

- approval bertingkat dimodelkan eksplisit;
- invalid transition ditolak;
- concurrency control mencegah request paralel melampaui available amount.

### 3.6 Perancangan SLA Tracking

Entity utama:

```text
sla_records
```

Field penting:

- `entity_type`;
- `entity_id`;
- `state`;
- `owner_org_id`;
- `entered_at`;
- `due_at`;
- `resolved_at`.

Algoritma ringkas:

```text
Jika resolvedAt != null:
  status = RESOLVED
Jika now > dueAt:
  status = SLA_VIOLATED
Jika remaining time mendekati dueAt:
  status = SLA_WARNING
Selain itu:
  status = ON_TRACK
```

Bahasan jurnal:

- SLA bukan sekadar deadline global, tetapi deadline per state;
- owner organization membuat bottleneck bisa dilacak;
- dashboard dapat menampilkan ticket mana yang melanggar SLA.

### 3.7 Perancangan Policy Layer

Policy layer dibuat untuk memusatkan authorization.

Contoh policy:

- user harus member organisasi;
- organisasi harus participant project;
- organisasi harus memiliki role tertentu pada contract;
- member role harus admin untuk aksi konfigurasi;
- actor harus sesuai tahap workflow.

Manfaat:

- controller lebih bersih;
- business rule akses tidak tersebar;
- unit test policy lebih mudah;
- response `403` lebih konsisten.

### 3.8 Perancangan WBS Integrity Validation

WBS dimodelkan sebagai tree.

Validasi yang dibahas:

- root weight;
- leaf weight;
- sibling weight;
- self-parent prevention;
- parent beda contract;
- cycle detection;
- orphan task;
- child deadline tidak boleh melewati parent deadline.

Endpoint:

```text
GET /api/monitoring/project/{projectId}/wbs/validation
```

Nilai akademik:

- sistem tidak hanya menyimpan task, tetapi memvalidasi struktur tree;
- validasi integrity mencegah dashboard progress memakai baseline yang keliru.

### 3.9 Perancangan Realtime Delivery

Komponen:

- tabel `outbox_events`;
- outbox worker;
- WebSocket hub;
- topic authorization;
- Redis Pub/Sub fanout;
- frontend realtime client.

Topic:

```text
user.{userId}
project.{projectId}
project.{projectId}.monitoring
project.{projectId}.ticket
project.{projectId}.finance
ticket.{ticketId}
payment.{paymentRequestId}
organization.{organizationId}
```

Alur:

```text
Business transaction
  -> commit data utama
  -> simpan outbox event
  -> worker proses event
  -> publish ke topic
  -> WebSocket client menerima event
  -> frontend invalidate/refetch REST query
```

### 3.10 Metode Pengujian

Pengujian yang sudah relevan dibahas:

- workflow FSM test;
- policy layer test;
- WBS validation test;
- auth middleware/service/controller test;
- refresh token revocation test;
- rate limiter test;
- audit service/middleware test;
- finance service test;
- tracker service test;
- monitoring service test;
- dashboard cache test;
- uploader helper test;
- WebSocket connect/ping/subscribe test;
- topic authorization test;
- outbox processor test;
- controller contract test.

Perintah verifikasi:

```sh
go test ./...
npm run build
```

---

## IV. Hasil dan Pembahasan

### 4.1 Implementasi Modular Monolith

Bahas kenapa modular monolith cocok:

- satu deployment lebih sederhana;
- transaksi database lebih mudah dijaga;
- boundary domain tetap jelas;
- cocok untuk prototipe akademik dan sistem internal.
- setiap domain seperti IAM, Project, Monitoring, Tracker, Finance, Support, Audit, Outbox, dan Realtime tetap dipisahkan pada package/module berbeda.

Bahas juga stack implementasi:

- Go dipakai untuk backend API yang konkuren dan type-safe secara bahasa;
- PostgreSQL dipakai sebagai source of truth untuk data transaksional;
- SQLC dipakai agar query SQL tetap eksplisit, tetapi parameter dan return type dapat dicek oleh compiler;
- Redis dipakai secara optional untuk revocation, rate limit, cache dashboard, dan WebSocket fanout;
- WebSocket dipakai sebagai delivery channel, sedangkan REST tetap menjadi kontrak data utama.

Mapping code:

```text
internal/module/iam
internal/module/project
internal/module/monitoring
internal/module/tracker
internal/module/finance
internal/module/support
internal/policy
internal/workflow
internal/audit
internal/outbox
internal/module/realtime
```

### 4.2 Implementasi Organization-Centric Access Control

Bahas bahwa akses tidak hanya berdasarkan user role, tetapi berdasarkan relasi user dengan organisasi dan project.

Contoh kasus:

- user dengan role benar tetapi bukan participant project tetap ditolak;
- organization admin dapat melakukan konfigurasi tertentu;
- contractor tidak dapat menjalankan approval auditor;
- loan provider hanya dapat melakukan disbursement pada tahap yang valid.

Hasil yang bisa ditulis:

- policy layer membuat authorization konsisten;
- controller tidak perlu menduplikasi pengecekan yang sama;
- test policy membuktikan rule dapat diverifikasi otomatis.
- role sistem menormalisasi perbedaan terminologi antar institusi, misalnya pemberi dana asing, executing agency lokal, kontraktor, auditor, dan instansi pendukung.

### 4.3 Implementasi FSM Transition Engine

Bahas bahwa sistem memakai transition validation untuk:

- monitoring progress;
- ticket tracker;
- finance payment.

Tabel contoh finance:

| Current State | Input | Next State | Actor |
| --- | --- | --- | --- |
| `SUBMITTED` | `auditor_approve` | `AUDITOR_APPROVED` | Auditor |
| `SUBMITTED` | `auditor_reject` | `AUDITOR_REJECTED` | Auditor |
| `AUDITOR_APPROVED` | `agency_approve` | `AGENCY_APPROVED` | Executing Agency |
| `AGENCY_APPROVED` | `disburse` | `PAID` | Loan Provider |

Titik pembahasan:

- invalid transition menghasilkan error;
- frontend kemudian menyesuaikan action yang ditampilkan;
- workflow engine mengurangi risiko status loncat.

### 4.4 Implementasi SLA Tracking untuk Ticket

Bahas ticket sebagai representasi hambatan lapangan.

Poin teknis:

- setiap ticket punya SLA record;
- sistem menghitung remaining seconds;
- status SLA dikembalikan ke frontend;
- ticket list/detail dapat menampilkan badge SLA;
- SLA membantu mendeteksi bottleneck organisasi.

Contoh response:

```json
{
  "sla": {
    "state": "OPEN",
    "dueAt": "2026-06-28T00:00:00Z",
    "status": "SLA_WARNING",
    "remainingSeconds": 3600,
    "isViolated": false
  }
}
```

### 4.5 Implementasi WBS Integrity Validation

Bahas bahwa WBS menjadi dasar perhitungan progress.

Masalah jika tidak divalidasi:

- total bobot lebih dari 100%;
- task cycle;
- child task deadline melampaui parent;
- task orphan;
- progress dashboard tidak akurat.

Hasil implementasi:

- create/update task menolak struktur yang merusak integrity;
- endpoint validation memberi status keseluruhan WBS;
- frontend menampilkan panel WBS integrity.

### 4.6 Implementasi Finance Concurrency Control

Bahas risiko race condition:

- dua user mengajukan payment request bersamaan;
- available amount dihitung dari data yang sama;
- tanpa lock, total request bisa melebihi kontrak.

Solusi:

- transaction boundary;
- row lock `FOR UPDATE` pada contract/payment summary;
- validasi pending request dan amount dalam transaksi.
- operasi perhitungan available amount dilakukan dekat dengan database agar keputusan tidak hanya bergantung pada validasi frontend.

Nilai jurnal:

- menunjukkan pembahasan data consistency;
- tidak hanya UI validation, tetapi proteksi backend.
- relevan pada proyek ODA karena approval dan disbursement melibatkan aktor berbeda serta harus menjaga konsistensi nilai kontrak.

### 4.7 Implementasi Refresh Token Revocation dan Rate Limiting

Bahas sisi security:

- refresh token diberi `jti`;
- token lama dapat direvoke;
- logout dan revoke-all tersedia;
- Redis blacklist/cutoff dipakai ketika tersedia;
- rate limit auth endpoint mencegah brute force.

Endpoint penting:

```text
POST /api/iam/auth/logout
POST /api/iam/auth/revoke-all
POST /api/iam/auth/login
POST /api/iam/auth/check-email
POST /api/iam/auth/refresh-token
```

Nilai jurnal:

- sistem lintas lembaga membutuhkan session control;
- keamanan auth dibahas sebagai bagian dari governance.

### 4.8 Implementasi Audit Log Global

Bahas bahwa write operation penting dicatat.

Audit log mendukung:

- investigasi perubahan data;
- pertanggungjawaban aktor;
- tracing request berdasarkan request id;
- dasar future feature audit dashboard.

Kaitkan dengan governance:

> Pada sistem multi-stakeholder, transparansi tidak cukup hanya melihat status terakhir. Sistem juga harus menyimpan siapa melakukan apa, kapan, terhadap entity apa, dan melalui request apa.

Catatan istilah:

> Gunakan istilah audit trail terpusat atau append-oriented audit log. Hindari klaim "immutable audit log" kecuali nanti database sudah diberi proteksi khusus seperti append-only permission, trigger pencegah update/delete, atau penyimpanan WORM.

### 4.9 Implementasi Secure Upload dan File Metadata

Bahas upload sebagai bagian penting karena progress/ticket/payment sering memiliki bukti dokumen.

Improvement:

- metadata file disimpan di tabel `files`;
- MIME dicek dari content;
- checksum dihitung;
- uploader user/org dicatat;
- entity binding tersedia.

Nilai jurnal:

- file bukan hanya URL;
- dokumen dapat dilacak;
- integritas file lebih baik.

### 4.10 Implementasi Dashboard Cache

Bahas dashboard sebagai beban query agregasi.

Solusi:

- Redis read-through cache;
- TTL singkat;
- invalidasi saat data domain berubah;
- fallback database jika Redis mati.

Nilai jurnal:

- menunjukkan performance engineering;
- cache tidak mengubah source of truth.

### 4.11 Implementasi Domain Event, Outbox, dan WebSocket

Bahas event-driven flow:

- business service membuat event;
- event masuk outbox;
- worker publish event;
- WebSocket hub mengirim ke client yang subscribe;
- Redis Pub/Sub mendukung fanout lintas instance;
- frontend invalidate/refetch query berdasarkan event.

Event yang dapat dibahas:

```text
progress.submitted
progress.validated
ticket.created
ticket.assigned
ticket.resolved
ticket.closed
payment.approved
payment.paid
organization.member.invited
```

Nilai jurnal:

- realtime delivery meningkatkan visibility dan mengurangi kebutuhan polling manual;
- outbox menjaga reliability event;
- topic authorization menjaga keamanan channel realtime;
- event realtime diperlakukan sebagai sinyal untuk refetch data REST, bukan sebagai sumber kebenaran utama.

### 4.12 Implementasi Frontend Pendukung

Walaupun fokus jurnal di backend, frontend dapat dibahas singkat sebagai client validasi contract.

Frontend improvement:

- API client mengikuti refresh token rotation;
- logout/revoke-all;
- rate limit countdown;
- upload metadata;
- WBS validation panel;
- SLA ticket UI;
- finance FSM action UI;
- WebSocket realtime client;
- UI error mapping.

Posisi dalam jurnal:

> Frontend digunakan untuk membuktikan bahwa contract backend dapat dikonsumsi secara praktis, namun novelty utama tetap berada pada desain dan implementasi backend workflow governance.

### 4.13 Keterkaitan dengan Studi Kasus MRT Jakarta Fase 2A

Bagian ini dapat menjadi jembatan eksplisit dengan jurnal HI.

Cara menulis:

- MRT Jakarta Fase 2A dipakai sebagai studi kasus proyek infrastruktur transnasional berbasis ODA;
- CP205, pekerjaan sipil, integrasi sistem, utilitas, dan koordinasi lintas instansi dipakai sebagai konteks kebutuhan workflow;
- sistem tidak diklaim sebagai sistem resmi MRT/JICA, tetapi sebagai prototipe yang merepresentasikan kebutuhan teknis proyek sejenis;
- aktor sistem dinormalisasi menjadi role generik agar model dapat diterapkan pada proyek infrastruktur lain;
- nilai IT yang diuji adalah correctness workflow, access control, traceability, data integrity, dan realtime readiness.

Contoh narasi:

> Pada studi kasus MRT Jakarta Fase 2A, tantangan koordinasi tidak hanya muncul dari aspek konstruksi, tetapi juga dari aliran persetujuan, validasi progress, pelaporan hambatan, dan kebutuhan visibilitas pemberi dana asing. Prototipe backend yang dikembangkan memodelkan kebutuhan tersebut menjadi workflow digital berbasis FSM, SLA Tracking, policy layer, audit trail, dan realtime event delivery.

---

## V. Evaluasi Sistem

### 5.1 Evaluasi Correctness Workflow

Uji:

- valid transition diterima;
- invalid transition ditolak;
- terminal state tidak bisa dilanjutkan sembarangan;
- actor yang salah ditolak.

Contoh klaim:

> Pengujian workflow memastikan bahwa payment request tidak dapat berpindah langsung dari `SUBMITTED` ke `PAID` tanpa approval auditor dan executing agency.

### 5.2 Evaluasi Authorization

Uji:

- user tanpa project access ditolak;
- organization role salah ditolak;
- member role tidak cukup ditolak;
- participant project yang valid diterima.

### 5.3 Evaluasi Data Integrity

Uji:

- WBS cycle ditolak;
- self-parent ditolak;
- sibling weight >100 ditolak;
- child deadline melewati parent ditolak;
- duplicate validation ditolak;
- payment amount melebihi available amount ditolak.

### 5.4 Evaluasi Security

Uji:

- refresh token revoked tidak bisa dipakai;
- token tanpa claim wajib ditolak;
- rate limit menghasilkan `429 RATE_LIMIT_EXCEEDED`;
- logout membersihkan session server-side jika Redis tersedia;
- upload file invalid MIME ditolak.

### 5.5 Evaluasi Realtime Delivery

Uji:

- WebSocket connect memakai `accessToken`;
- subscribe topic authorized berhasil;
- subscribe topic unauthorized ditolak tanpa memutus koneksi;
- ping/pong berjalan;
- outbox event dipetakan ke realtime envelope;
- frontend melakukan refetch setelah menerima event.

### 5.6 Evaluasi Build dan Contract

Backend:

```sh
go test ./...
```

Frontend:

```sh
npm run build
```

Contract yang sudah diadaptasi frontend:

- upload response metadata;
- ticket `sla`;
- auth logout/revoke-all;
- rate limit `Retry-After`;
- WBS validation endpoint;
- WebSocket endpoint;
- UI error mapping.

---

## VI. Kesimpulan dan Saran

### 6.1 Kesimpulan

Kesimpulan yang dapat ditulis:

- proses monitoring proyek lintas lembaga dapat dimodelkan sebagai workflow digital berbasis FSM;
- SLA Tracking membantu mengubah keterlambatan birokrasi menjadi indikator teknis yang terukur;
- Organization-Centric Access Control lebih sesuai daripada RBAC sederhana untuk sistem multi-stakeholder;
- audit log, secure upload, dan outbox pattern meningkatkan traceability serta reliability;
- WebSocket realtime delivery dapat meningkatkan responsivitas dashboard tanpa menggantikan REST sebagai source of truth;
- pengujian backend membuat klaim reliability lebih kuat untuk jurnal Informatika.

### 6.2 Saran Pengembangan

Saran lanjutan:

- audit log viewer untuk admin/supervisor;
- konfigurasi SLA per project, state, dan organization;
- escalation notification ke email atau notification center;
- OpenAPI spec sebagai kontrak resmi backend/frontend;
- metrics dan tracing untuk observability;
- materialized view untuk dashboard analytics besar;
- virus scanning untuk upload dokumen sensitif;
- device/session management untuk refresh token;
- signed approval record untuk finance.

---

## 7. Mapping Improvement Project ke Pembahasan Jurnal

| Improvement | Bahasan Jurnal | Nilai Informatika |
| --- | --- | --- |
| Centralized Policy Layer | Access Control | Organization-Centric RBAC/ABAC |
| FSM Transition Engine | Workflow Modeling | Formalisasi state dan transition |
| SLA Tracking | Bottleneck Detection | Pengukuran durasi state |
| Audit Log Global | Accountability | Traceability actor dan entity |
| Finance Concurrency Control | Data Consistency | Transaction dan row lock |
| Refresh Token Revocation | Security | Server-side session invalidation |
| Rate Limiting Auth | Security | Mitigasi brute force |
| Dashboard Cache | Performance | Redis read-through cache |
| Domain Event + Outbox | Reliability | Event delivery tanpa dual-write loss |
| WebSocket + Redis Fanout | Realtime Delivery | Topic-based push update |
| WBS Integrity Validation | Data Integrity | Tree validation dan weight consistency |
| File Metadata + Secure Upload | File Governance | MIME sniffing, checksum, metadata |
| Unit Test + Contract Test | Quality Assurance | Verifikasi business rules |
| Frontend Contract Update | System Integration | Konsumsi contract dan UX error mapping |

---

## 8. Outline Singkat Siap Pakai

Jika jurnal perlu dibuat lebih ringkas, gunakan struktur berikut:

```text
Judul:
Perancangan Backend Workflow Governance Proyek Infrastruktur Transnasional Berbasis ODA Menggunakan Finite State Machine, SLA Tracking, dan Realtime Event Delivery: Studi Kasus MRT Jakarta Fase 2A

Abstrak

I. Pendahuluan
1.1 Latar belakang proyek infrastruktur transnasional dan studi kasus MRT Jakarta Fase 2A
1.2 Masalah asimetri informasi, red tape, dan workflow manual
1.3 Rumusan masalah teknis
1.4 Tujuan penelitian
1.5 Kontribusi penelitian

II. Tinjauan Pustaka
2.1 Sistem informasi monitoring proyek
2.2 Finite State Machine
2.3 SLA Tracking
2.4 RBAC/ABAC multi-organisasi
2.5 Audit trail
2.6 Outbox pattern dan realtime delivery

III. Metode
3.1 Research and development
3.2 Analisis aktor lintas institusi dan kebutuhan sistem
3.3 Perancangan arsitektur modular monolith
3.4 Perancangan FSM monitoring, ticket, dan finance
3.5 Perancangan policy layer dan audit log
3.6 Perancangan outbox dan WebSocket
3.7 Strategi pengujian

IV. Hasil dan Pembahasan
4.1 Implementasi modular monolith
4.2 Implementasi policy layer
4.3 Implementasi FSM transition engine
4.4 Implementasi SLA tracking
4.5 Implementasi WBS integrity validation
4.6 Implementasi finance concurrency control
4.7 Implementasi audit log dan secure upload
4.8 Implementasi dashboard cache
4.9 Implementasi outbox dan WebSocket realtime
4.10 Keterkaitan dengan studi kasus MRT Jakarta Fase 2A
4.11 Evaluasi unit test, contract test, dan build

V. Kesimpulan dan Saran
5.1 Kesimpulan
5.2 Keterbatasan
5.3 Saran pengembangan
```

---

## 9. Catatan Penting agar Klaim Jurnal Aman

Gunakan klaim yang sesuai implementasi:

- boleh klaim sistem memiliki prototipe backend workflow governance;
- boleh klaim WebSocket sudah menjadi channel realtime delivery;
- boleh klaim REST tetap source of truth;
- boleh klaim Redis optional dan sistem punya graceful degradation;
- boleh klaim audit log dan outbox sudah menjadi fondasi accountability dan event reliability;
- boleh klaim pengujian mencakup rule penting backend;
- boleh klaim MRT Jakarta Fase 2A sebagai studi kasus atau konteks kebutuhan.

Hindari klaim berlebihan:

- jangan klaim sudah terintegrasi resmi dengan JICA, Pemprov, atau PT MRT Jakarta;
- jangan klaim menggantikan seluruh proses birokrasi nyata;
- jangan klaim realtime analytics penuh jika dashboard masih refetch dari REST;
- jangan klaim event terkirim "tanpa delay"; gunakan near-real-time atau realtime delivery signal;
- jangan klaim digital signature legal jika approval masih berupa user/org/timestamp;
- jangan klaim audit log immutable jika belum ada proteksi append-only di level database/storage;
- jangan klaim anti-red tape menyelesaikan masalah organisasi sepenuhnya, cukup tulis sebagai mekanisme mitigasi teknis.

Kalimat aman:

> Sistem yang dikembangkan merupakan prototipe backend workflow governance berbasis studi kasus proyek infrastruktur transnasional seperti MRT Jakarta Fase 2A. Implementasi ini tidak menggantikan sistem resmi instansi, tetapi menunjukkan bagaimana FSM, SLA Tracking, access control, audit trail, dan realtime event delivery dapat digunakan untuk meningkatkan transparansi serta traceability proses.
