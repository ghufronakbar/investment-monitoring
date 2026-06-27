# New Version Summary

Dokumen ini merangkum perbedaan penting antara `origin/main` dan `main`, terutama dampaknya ke frontend. Fokus utama versi ini adalah peningkatan backend dari REST API biasa menjadi sistem yang lebih kuat secara policy, workflow, auditability, file integrity, cache, outbox event, dan realtime delivery.

## 1. Kesimpulan Umum

Versi baru menambahkan fondasi backend yang jauh lebih "production-like":

- centralized policy layer untuk authorization lintas module;
- FSM transition engine untuk monitoring, tracker, dan finance;
- SLA tracking untuk ticket;
- audit log global;
- finance concurrency control;
- refresh token revocation;
- rate limiting endpoint auth;
- dashboard Redis cache;
- domain event + outbox pattern;
- WBS integrity validation;
- file metadata + secure upload;
- WebSocket realtime transport, topic authorization, Redis Pub/Sub fanout, dan outbox delivery ke WebSocket.

Secara frontend, mayoritas endpoint lama tetap ada. Namun ada beberapa contract/response baru yang perlu diperhatikan, terutama:

- response upload file sekarang bertambah metadata;
- response ticket sekarang bisa berisi field `sla`;
- endpoint auth bisa mengembalikan `429 RATE_LIMIT_EXCEEDED`;
- refresh token lama sebelum versi ini kemungkinan perlu login ulang karena format token sekarang wajib punya `jti`, `iat`, dan `exp`;
- ada endpoint auth baru `logout` dan `revoke-all`;
- ada endpoint monitoring baru untuk validasi WBS;
- ada endpoint WebSocket baru untuk realtime updates.

## 2. Migration dan Infrastructure

Migration baru:

```text
migrations/20260627090000_add_sla_records.sql
migrations/20260627093000_add_audit_logs.sql
migrations/20260627094500_add_outbox_events.sql
migrations/20260627100000_add_file_metadata.sql
```

Tabel baru:

```text
sla_records
audit_logs
outbox_events
files
```

Command migration tetap:

```sh
./up.sh
```

`up.sh` sekarang membaca `DATABASE_URL` dari environment atau `.env`, tidak hardcoded.

Dependency baru:

```text
nhooyr.io/websocket
```

Config/env baru untuk WebSocket:

```env
WEBSOCKET_ENABLED=true
WEBSOCKET_ALLOWED_ORIGINS=*
WEBSOCKET_READ_LIMIT_BYTES=65536
WEBSOCKET_WRITE_TIMEOUT_SECONDS=10
WEBSOCKET_IDLE_TIMEOUT_SECONDS=60
WEBSOCKET_PING_INTERVAL_SECONDS=25
WEBSOCKET_CLIENT_SEND_BUFFER=64
WEBSOCKET_REDIS_CHANNEL_PREFIX=invmonitor:ws
WEBSOCKET_PRESENCE_TTL_SECONDS=90
```

Catatan:

- `WEBSOCKET_ALLOWED_ORIGINS=*` dipakai sementara untuk development.
- Untuk production sebaiknya diganti whitelist domain frontend.
- Redis tetap optional. Jika Redis mati, server tetap jalan, tetapi beberapa fitur degrade: token revocation/rate limit/cache/realtime fanout tidak lintas instance.

## 3. Perubahan Contract REST untuk Frontend

### 3.1 Auth: Endpoint Baru Logout dan Revoke All

Endpoint baru:

```text
POST /api/iam/auth/logout
POST /api/iam/auth/revoke-all
```

Request body:

```json
{
  "refreshToken": "string"
}
```

Response logout:

```json
{
  "message": "SUCCESS_LOGOUT",
  "data": {
    "message": "LOGOUT_SUCCESSFULLY"
  }
}
```

Response revoke-all:

```json
{
  "message": "SUCCESS_REVOKE_ALL_REFRESH_TOKENS",
  "data": {
    "message": "REFRESH_TOKENS_REVOKED_SUCCESSFULLY"
  }
}
```

Dampak frontend:

- Saat logout, frontend sebaiknya panggil `/logout` dengan refresh token aktif sebelum clear local auth state.
- Untuk logout semua device/session, panggil `/revoke-all`.
- Jika Redis aktif, refresh token yang sudah logout/revoke tidak bisa dipakai lagi.

### 3.2 Auth: Refresh Token Rotation dan Revocation

Refresh token sekarang punya JWT claims wajib:

```text
jti
iat
exp
```

Dampak frontend:

- Setelah `POST /api/iam/auth/refresh-token`, frontend wajib menyimpan `refreshToken` baru dari response.
- Jangan terus memakai refresh token lama setelah refresh sukses.
- Token refresh lama yang dibuat sebelum versi ini bisa gagal dengan `INVALID_REFRESH_TOKEN` atau `INVALID_REFRESH_TOKEN_CLAIMS`; user mungkin perlu login ulang sekali setelah deploy.

Response refresh token tetap:

```json
{
  "accessToken": "string",
  "refreshToken": "string"
}
```

### 3.3 Auth: Rate Limiting

Endpoint yang sekarang dilindungi rate limit:

```text
POST /api/iam/auth/check-email
POST /api/iam/auth/login
POST /api/iam/auth/refresh-token
```

Default limit:

```text
5 request / 5 menit
```

Jika limit terlampaui:

```http
HTTP 429
Retry-After: {seconds}
```

Response error:

```json
{
  "message": "RATE_LIMIT_EXCEEDED"
}
```

Dampak frontend:

- UI login/check-email perlu handle `429`.
- Gunakan header `Retry-After` untuk countdown atau disable submit sementara.
- Jangan auto-retry refresh token secara agresif.

### 3.4 Monitoring: Endpoint Baru WBS Validation

Endpoint baru:

```text
GET /api/monitoring/project/{projectId}/wbs/validation
```

Response:

```json
{
  "isValid": false,
  "totalRootWeight": 80,
  "totalLeafWeight": 95,
  "errors": [
    "TOTAL_ROOT_WEIGHT_NOT_100",
    "TOTAL_LEAF_WEIGHT_NOT_100"
  ]
}
```

Dampak frontend:

- Frontend bisa menampilkan status validitas WBS sebelum user memakai data untuk baseline/progress.
- Draft WBS dengan total kurang dari 100 masih boleh disimpan, tetapi endpoint ini akan menandai `isValid=false`.
- Create/update task sekarang bisa menolak WBS yang membuat total melebihi 100, cycle, parent beda contract, self-parent, orphan, atau deadline child melewati parent.

Error code penting:

```text
TASK_CANNOT_PARENT_ITSELF
PARENT_TASK_NOT_FOUND
PARENT_TASK_NOT_IN_CONTRACT
CHILD_DEADLINE_EXCEEDS_PARENT
WBS_CYCLE_DETECTED
ORPHAN_TASK
TOTAL_ROOT_WEIGHT_EXCEEDS_100
TOTAL_LEAF_WEIGHT_EXCEEDS_100
SIBLING_WEIGHT_EXCEEDS_100
```

### 3.5 Tracker: Ticket Response Bertambah Field SLA

`TicketReturn` sekarang memiliki field opsional:

```json
{
  "sla": {
    "id": "uuid",
    "entityType": "TICKET",
    "entityId": "uuid",
    "state": "OPEN",
    "ownerOrgId": "uuid",
    "enteredAt": "2026-06-27T00:00:00Z",
    "dueAt": "2026-06-28T00:00:00Z",
    "resolvedAt": null,
    "status": "ACTIVE",
    "remainingSeconds": 3600,
    "isViolated": false
  }
}
```

Dampak frontend:

- Existing UI yang mengabaikan unknown field tetap aman.
- UI ticket bisa menampilkan SLA countdown, status violation, due date, dan resolved time.
- Karena `sla` memakai `omitempty`, field bisa tidak ada pada data lama atau kondisi tertentu. Frontend harus null-safe.

### 3.6 Upload: Response Bertambah Metadata File

Endpoint lama tetap:

```text
POST /api/support/upload/single
POST /api/support/upload/multiple
```

Form field wajib tetap:

```text
file   // single
files  // multiple
```

Form field opsional baru:

```text
organizationId
entityType
entityId
```

Response lama:

```json
{
  "publicId": "string",
  "url": "string",
  "format": "jpg"
}
```

Response baru:

```json
{
  "id": "uuid",
  "publicId": "string",
  "url": "string",
  "format": "jpg",
  "fileName": "evidence.jpg",
  "mimeType": "image/jpeg",
  "size": 123456,
  "checksum": "sha256-hex",
  "uploadedByUserId": "uuid",
  "uploadedByOrgId": "uuid",
  "entityType": "PROGRESS_LOG",
  "entityId": "uuid",
  "createdAt": "2026-06-27T00:00:00Z"
}
```

Dampak frontend:

- Field lama `publicId`, `url`, dan `format` tetap ada.
- Frontend lama yang hanya membaca `url` tetap aman.
- Frontend baru bisa menyimpan/menampilkan metadata file.
- Validasi MIME sekarang berdasarkan content sniffing, bukan header multipart. File dengan ekstensi/header palsu akan ditolak.
- Jika mengirim `organizationId`, user harus member organization tersebut.
- `entityType` dan `entityId` harus dikirim bersama; jika hanya salah satu akan error:

```text
ENTITY_TYPE_AND_ENTITY_ID_REQUIRED_TOGETHER
```

Error baru yang perlu di-handle:

```text
FILE_EMPTY
INVALID_FILE_TYPE
USER_NOT_MEMBER_OF_ORGANIZATION
ENTITY_TYPE_AND_ENTITY_ID_REQUIRED_TOGETHER
```

### 3.7 Realtime WebSocket: Endpoint Baru

Endpoint baru:

```text
GET /api/realtime/ws?accessToken={accessToken}
```

Alternatif untuk tool yang bisa set header:

```http
Authorization: Bearer {accessToken}
```

Catatan penting:

- Query token yang didukung hanya `accessToken`.
- `postmaticAccessToken` tidak dipakai untuk WebSocket project ini.
- Ini WebSocket endpoint, bukan REST JSON endpoint.

Initial event:

```json
{
  "type": "realtime.connected",
  "topic": "realtime.system",
  "data": {
    "clientId": "generated-client-id",
    "userId": "authenticated-user-id",
    "topics": []
  },
  "sentAt": "2026-06-27T00:00:00Z"
}
```

Ping:

```json
{
  "type": "ping"
}
```

Pong:

```json
{
  "type": "realtime.pong",
  "topic": "realtime.system",
  "data": {
    "receivedAt": "2026-06-27T00:00:00Z"
  },
  "sentAt": "2026-06-27T00:00:00Z"
}
```

Subscribe:

```json
{
  "type": "subscribe",
  "topics": [
    "user.{userId}",
    "project.{projectId}.monitoring",
    "project.{projectId}.ticket",
    "project.{projectId}.finance"
  ]
}
```

Subscribe acknowledgement:

```json
{
  "type": "realtime.subscription_updated",
  "topic": "realtime.system",
  "data": {
    "subscribed": ["project.{projectId}.monitoring"],
    "unsubscribed": [],
    "rejected": []
  },
  "sentAt": "2026-06-27T00:00:00Z"
}
```

Supported topics:

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

Domain event data shape:

```json
{
  "type": "ticket.created",
  "topic": "project.{projectId}.ticket",
  "data": {
    "outboxEventId": "uuid",
    "eventType": "TicketCreated",
    "aggregateType": "ticket",
    "aggregateId": "uuid",
    "payload": {}
  },
  "sentAt": "2026-06-27T00:00:00Z"
}
```

Dampak frontend:

- Frontend bisa mulai subscribe realtime updates, tetapi REST API tetap sumber data utama.
- Saat menerima event domain, strategi paling aman adalah refetch list/detail terkait berdasarkan topic/event.
- Topic unauthorized tidak menutup koneksi; topic akan masuk `data.rejected`.
- Client perlu kirim app-level ping berkala, misalnya tiap 25-30 detik.

## 4. Perubahan Behavior yang Tidak Mengubah Response Besar

### 4.1 Centralized Policy Layer

Authorization banyak route sekarang memakai `internal/policy`.

Dampak frontend:

- Endpoint sama, tetapi beberapa request yang sebelumnya mungkin lolos bisa sekarang ditolak lebih konsisten dengan `403`.
- Error message bisa lebih spesifik, misalnya:

```text
NO_PROJECT_ACCESS
NOT_PROJECT_PARTICIPANT
EXECUTING_AGENCY_ADMIN_REQUIRED
CONTRACTOR_REQUIRED
AUDITOR_REQUIRED
LOAN_PROVIDER_REQUIRED
ADMIN_REQUIRED
```

### 4.2 FSM Transition Engine

Monitoring progress, tracker ticket, dan finance payment memakai transition validation yang lebih eksplisit.

Dampak frontend:

- Jika user melakukan transisi status tidak valid, backend bisa menolak dengan error workflow.
- UI sebaiknya hanya menampilkan action sesuai status sekarang dan role user.

### 4.3 Finance Concurrency Control

Create payment request sekarang memakai transaction dan row lock `FOR UPDATE`.

Dampak frontend:

- Response sukses tetap sama.
- Jika ada request paralel yang menyebabkan amount melewati available amount, salah satu request akan ditolak.
- UI harus handle error domain seperti payment amount melebihi sisa kontrak/pending amount.

### 4.4 Dashboard Cache

Dashboard S-Curve dan stats sekarang memakai Redis read-through cache.

Dampak frontend:

- Response dashboard tetap sama.
- Data dashboard di-cache singkat dan invalidasi dilakukan saat WBS/progress/ticket/payment/project berubah.
- Jika Redis tidak tersedia, backend fallback hitung dari database.

### 4.5 Audit Log Global

Request write penting sekarang dicatat ke `audit_logs`.

Dampak frontend:

- Tidak ada response contract baru untuk frontend.
- Berguna untuk admin/audit feature di masa depan.

### 4.6 Domain Event dan Outbox

Event domain sekarang disimpan di `outbox_events`, diproses worker, dan bisa dikirim ke WebSocket.

Dampak frontend:

- REST contract tidak berubah.
- WebSocket client bisa menerima event realtime dari worker.

## 5. Breaking/Compatibility Notes untuk Frontend

Hal yang perlu frontend adaptasi:

1. **Refresh token lama mungkin invalid setelah deploy.**

   Karena refresh token sekarang wajib punya `jti`, `iat`, dan `exp`, user yang masih memegang token lama mungkin perlu login ulang.

2. **Frontend wajib menyimpan refresh token baru setiap refresh.**

   Refresh token lama akan di-blacklist ketika refresh sukses jika Redis aktif.

3. **Auth endpoint bisa return `429 RATE_LIMIT_EXCEEDED`.**

   UI login/check-email/refresh-token perlu handle countdown/retry.

4. **Upload response bertambah banyak field.**

   Tidak breaking jika frontend hanya membaca `url`, tetapi TypeScript type/interface perlu di-update.

5. **Ticket response bertambah `sla`.**

   Tidak breaking jika parser fleksibel, tetapi TypeScript type perlu di-update dan harus null-safe.

6. **WBS create/update sekarang lebih ketat.**

   UI WBS perlu menampilkan error backend jika user membuat bobot >100, cycle, parent invalid, atau deadline invalid.

7. **WebSocket adalah channel baru, bukan pengganti REST.**

   Frontend sebaiknya tetap refetch REST saat menerima event, terutama untuk dashboard/list yang kompleks.

8. **Authorization lebih konsisten.**

   Beberapa action yang sebelumnya bisa lolos karena validasi tersebar sekarang bisa ditolak oleh policy layer.

## 6. Checklist Update Frontend

Prioritas tinggi:

- [ ] Update auth client agar menyimpan refresh token baru dari response refresh.
- [ ] Tambah call `POST /api/iam/auth/logout` saat logout.
- [ ] Handle `429 RATE_LIMIT_EXCEEDED` dan header `Retry-After`.
- [ ] Update TypeScript/interface upload response.
- [ ] Update TypeScript/interface ticket response untuk field `sla`.
- [ ] Tambah UI WBS validation menggunakan `GET /api/monitoring/project/{projectId}/wbs/validation`.
- [ ] Pastikan form WBS menampilkan error domain dari backend.

Prioritas menengah:

- [ ] Tambah WebSocket client `GET /api/realtime/ws?accessToken=...`.
- [ ] Subscribe `user.{userId}` setelah connect.
- [ ] Subscribe topic project sesuai halaman aktif:
  - `project.{projectId}.monitoring`
  - `project.{projectId}.ticket`
  - `project.{projectId}.finance`
- [ ] Kirim ping berkala setiap 25-30 detik.
- [ ] Saat menerima event domain, refetch data terkait.

Prioritas lanjutan:

- [ ] Tampilkan SLA countdown/status pada ticket.
- [ ] Tampilkan metadata file upload di UI admin/detail dokumen.
- [ ] Tambah notification center dari event WebSocket.
- [ ] Buat handling reconnect WebSocket dengan backoff.

## 7. Backend Test dan Status Build

Test yang sudah ditambahkan mencakup:

- rate limiter;
- refresh token claims/revocation;
- workflow FSM;
- WBS validation;
- WebSocket connect/ping/subscribe;
- WebSocket topic parser/authorization;
- WebSocket local publish fallback;
- outbox realtime mapper.

Status verifikasi terakhir:

```text
go test ./... PASS
```

Catatan:

- Test WebSocket memakai `httptest`, sehingga di environment sandbox perlu izin bind ke loopback port.
- Tidak ada migration tambahan setelah `files`, `sla_records`, `audit_logs`, dan `outbox_events`.

## 8. Ringkasan Akhir

Versi baru ini bukan sekadar penambahan endpoint, tetapi perubahan arsitektural:

- backend lebih aman karena authorization terpusat, refresh token bisa dicabut, auth endpoint rate-limited, dan upload lebih sulit dipalsukan;
- backend lebih auditable karena audit log, progress snapshot, SLA, file metadata, dan outbox event;
- backend lebih robust karena workflow FSM, finance concurrency control, WBS integrity validation, dan Redis cache/fallback;
- backend mulai realtime karena WebSocket, topic authorization, Redis Pub/Sub, dan outbox delivery.

Untuk frontend, perubahan yang paling perlu segera ditangani adalah refresh token rotation, 429 rate limit, upload response metadata, ticket `sla`, WBS validation endpoint, dan optional WebSocket client.
