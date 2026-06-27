# WebSocket Implementation Plan

Dokumen ini menjadi acuan sebelum eksekusi implementasi WebSocket di backend JICA Investment System. Referensi utama berasal dari `Md/ExampleWs.md`, tetapi plan ini disesuaikan dengan struktur project saat ini: Go + chi, JWT auth, Redis opsional, outbox pattern, dan route root `/api`.

## 1. Tujuan Implementasi

WebSocket dipakai sebagai realtime delivery layer untuk mengirim event backend ke client tanpa polling manual.

Target awal:

- client bisa connect ke endpoint WebSocket dengan access token;
- client bisa subscribe/unsubscribe topic;
- server bisa kirim `realtime.connected`, `realtime.pong`, dan domain event;
- outbox worker bisa meneruskan event domain ke realtime channel;
- API instance yang memiliki connection aktif bisa deliver event ke client;
- Redis Pub/Sub dipakai sebagai fanout lintas instance jika Redis tersedia.

Yang bukan target tahap awal:

- chat dua arah antar user;
- presence detail yang kompleks;
- guaranteed delivery ke client offline;
- message replay;
- private file streaming;
- multi-region realtime gateway.

## 2. Status Saat Ini

Project saat ini sudah punya:

- REST API berbasis `chi`;
- JWT access token dan refresh token;
- Redis opsional untuk token revocation, rate limit, dan dashboard cache;
- outbox table dan worker polling;
- domain event seperti `ProgressSubmitted`, `TicketCreated`, `PaymentApproved`, dan `MemberInvited`.

Project belum punya:

- WebSocket dependency;
- realtime route;
- connection hub;
- topic subscription;
- Redis Pub/Sub realtime;
- delivery handler dari outbox ke client.

## 3. Dependency

Gunakan package:

```text
nhooyr.io/websocket
```

Catatan:

- Dokumentasi contoh menyebut `nhooyr.io/websocket`, tetapi repository aktif sekarang dikenal sebagai `github.com/coder/websocket`.
- Untuk implementasi, pilih satu import path yang stabil sesuai `go get` yang berhasil di environment project.
- Hindari `gorilla/websocket` kecuali ada alasan kompatibilitas khusus.

Checklist:

- [ ] Tambah dependency WebSocket ke `go.mod`.
- [ ] Pastikan `go mod tidy` tidak menghapus dependency yang masih dipakai.
- [ ] Pastikan `go test ./...` tetap pass.

## 4. Struktur Direktori

Tambahkan module realtime sebagai transport lintas domain:

```text
internal/module/realtime/
  websocket/
    controller/
      handler.go
    service/
      client.go
      envelope.go
      hub.go
      message.go
      publisher.go
      service.go
      subscription.go
```

Jika Redis Pub/Sub dibuat eksplisit:

```text
internal/repository/redis/realtime/
  pubsub.go
```

Alasan:

- WebSocket bukan bagian dari module monitoring, tracker, finance, atau IAM.
- WebSocket adalah transport realtime yang bisa dipakai banyak module.
- Domain module tidak perlu import controller WebSocket.

Checklist:

- [ ] Buat package `internal/module/realtime/websocket/service`.
- [ ] Buat package `internal/module/realtime/websocket/controller`.
- [ ] Hindari dependency dari domain service ke controller WebSocket.
- [ ] Sediakan interface publisher agar domain/outbox tidak tahu detail connection.

## 5. Config

Tambahkan config berikut di `config.Config`:

```text
WEBSOCKET_ENABLED=true
WEBSOCKET_ALLOWED_ORIGINS=*
WEBSOCKET_READ_LIMIT_BYTES=65536
WEBSOCKET_WRITE_TIMEOUT_SECONDS=10
WEBSOCKET_IDLE_TIMEOUT_SECONDS=60
WEBSOCKET_PING_INTERVAL_SECONDS=25
WEBSOCKET_CLIENT_SEND_BUFFER=64
WEBSOCKET_REDIS_CHANNEL_PREFIX=jica:ws
WEBSOCKET_PRESENCE_TTL_SECONDS=90
```

Mapping Go:

```go
WEBSOCKET_ENABLED bool
WEBSOCKET_ALLOWED_ORIGINS []string
WEBSOCKET_READ_LIMIT_BYTES int64
WEBSOCKET_WRITE_TIMEOUT time.Duration
WEBSOCKET_IDLE_TIMEOUT time.Duration
WEBSOCKET_PING_INTERVAL time.Duration
WEBSOCKET_CLIENT_SEND_BUFFER int
WEBSOCKET_REDIS_CHANNEL_PREFIX string
WEBSOCKET_PRESENCE_TTL time.Duration
```

Default aman:

- enabled: `true`;
- allowed origins: `*` untuk tahap awal agar development tidak terblokir CORS/origin policy;
- read limit: `65536`;
- write timeout: `10s`;
- idle timeout: `60s`;
- ping interval recommendation: `25s`;
- send buffer: `64`;
- Redis channel prefix: pakai `cfg.REDIS_PREFIX + ":ws"` jika env khusus kosong.

Checklist:

- [ ] Tambah field config.
- [ ] Tambah parser bool, int, duration, dan list string bila belum ada helper.
- [ ] Pastikan config punya default agar local development tetap jalan.
- [ ] Jangan log full query string yang berisi access token.

## 6. Routing

Route final mengikuti root `/api` yang sudah ada di `cmd/api/main.go`.

Endpoint:

```text
GET /api/realtime/ws
```

Mounting di `internal/http/router.go`:

```go
r.Route("/realtime", func(r chi.Router) {
    r.Mount("/ws", websocketHandler.Routes())
})
```

Auth handshake:

- support `?accessToken=...`;
- support `Authorization: Bearer ...` untuk tool seperti Postman;
- validasi token memakai `token_service.TokenMaker`.

Checklist:

- [ ] Buat controller `GET /ws`.
- [ ] Inject token service ke WebSocket handler/service.
- [ ] Validasi access token saat handshake.
- [ ] Return unauthorized jika token invalid.
- [ ] Mount route di `/api/realtime/ws`.

## 7. Message Contract

Envelope server ke client:

```json
{
  "type": "ticket.created",
  "topic": "project.{projectId}.ticket",
  "data": {},
  "sentAt": "2026-06-27T00:00:00Z"
}
```

Client message:

```json
{
  "type": "subscribe",
  "topic": "project.{projectId}"
}
```

Client message banyak topic:

```json
{
  "type": "subscribe",
  "topics": [
    "user.{userId}",
    "project.{projectId}",
    "ticket.{ticketId}"
  ]
}
```

Unsubscribe:

```json
{
  "type": "unsubscribe",
  "topic": "project.{projectId}"
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

Checklist:

- [ ] Buat `Envelope`.
- [ ] Buat `ClientMessage`.
- [ ] Validasi JSON message.
- [ ] Unknown message type dibalas error system, bukan panic.
- [ ] Batasi read limit.

## 8. Hub dan Client Lifecycle

Komponen:

- `Hub`: menyimpan client aktif dan index topic.
- `Client`: menyimpan connection, user id, subscribed topics, send channel, dan last activity.
- `Service`: koordinasi handler, hub, publisher, subscriber.

Perilaku:

- satu user bisa punya banyak connection;
- subscribe menambahkan client ke topic;
- unsubscribe menghapus client dari topic;
- disconnect membersihkan semua topic;
- write dilakukan lewat buffered channel;
- client lambat bisa diputus jika send buffer penuh.

Checklist:

- [ ] Implement `Hub.Register`.
- [ ] Implement `Hub.Unregister`.
- [ ] Implement `Hub.Subscribe`.
- [ ] Implement `Hub.Unsubscribe`.
- [ ] Implement `Hub.Publish`.
- [ ] Implement cleanup connection saat context done.
- [ ] Tambah close reason yang jelas saat server shutdown.

## 9. Topic Design untuk Project Ini

Topic awal:

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

Mapping event awal:

```text
ProgressSubmitted   -> project.{projectId}.monitoring
ProgressValidated   -> project.{projectId}.monitoring
TicketCreated       -> project.{projectId}.ticket
TicketAssigned      -> project.{projectId}.ticket, ticket.{ticketId}
TicketResolved      -> project.{projectId}.ticket, ticket.{ticketId}
TicketClosed        -> project.{projectId}.ticket, ticket.{ticketId}
PaymentApproved     -> project.{projectId}.finance, payment.{paymentRequestId}
PaymentPaid         -> project.{projectId}.finance, payment.{paymentRequestId}
MemberInvited       -> organization.{organizationId}, user.{invitedUserId}
```

Catatan penting:

- Outbox event saat ini belum selalu menyimpan `projectId` secara eksplisit.
- Untuk delivery yang akurat, payload outbox perlu berisi routing metadata seperti `projectId`, `ticketId`, `paymentRequestId`, `organizationId`, atau `targetUserId`.
- Jika payload belum lengkap, processor bisa melakukan query tambahan berdasarkan aggregate ID.

Checklist:

- [ ] Tentukan topic final tahap awal.
- [ ] Tambahkan helper event-to-topic mapping.
- [ ] Tambahkan routing metadata ke payload outbox bila perlu.
- [ ] Dokumentasikan event contract per module.

## 10. Topic Authorization

Subscribe topic harus divalidasi server-side.

Rules awal:

- `user.{userId}` hanya boleh untuk user itu sendiri.
- `project.{projectId}` dan turunannya hanya boleh jika user punya akses project via policy layer.
- `ticket.{ticketId}` hanya boleh jika user punya akses ticket lewat project/contract.
- `payment.{paymentRequestId}` hanya boleh jika user punya akses finance payment tersebut.
- `organization.{organizationId}` hanya boleh jika user member organization tersebut.

Reuse yang sudah ada:

- `internal/policy` untuk akses project/monitoring/tracker/finance jika memungkinkan.
- query IAM `CheckMembershipExists` untuk organization topic.

Checklist:

- [ ] Buat `AuthorizeTopic(ctx, userID, topic)`.
- [ ] Parse topic secara ketat.
- [ ] Reject topic format yang tidak dikenal.
- [ ] Return ack berisi `subscribed` dan `rejected`.
- [ ] Tambah test untuk authorization topic.

## 11. Redis Pub/Sub Fanout

In-memory hub hanya berlaku untuk satu API instance. Redis Pub/Sub dipakai agar event dari worker atau API instance lain tetap sampai ke client di instance yang berbeda.

Channel:

```text
{REDIS_PREFIX}:ws:events
```

Payload Redis:

```json
{
  "type": "ticket.created",
  "topic": "project.{projectId}.ticket",
  "data": {},
  "sentAt": "2026-06-27T00:00:00Z"
}
```

Mode fallback:

- Jika Redis tidak tersedia, WebSocket tetap bisa berjalan untuk event lokal satu process.
- Worker tidak bisa fanout lintas process tanpa Redis.
- Log warning, jangan crash server.

Checklist:

- [ ] Implement Redis publisher.
- [ ] Implement Redis subscriber.
- [ ] Subscriber deliver ke local hub.
- [ ] Fallback local publish jika Redis nil.
- [ ] Stop subscriber saat shutdown.

## 12. Outbox Integration

Saat ini `outbox.Processor` hanya log event lalu mark processed. Untuk realtime, processor perlu handler nyata.

Opsi implementasi:

1. Processor menerima `RealtimePublisher`.
2. Saat event diklaim, processor build envelope/topic.
3. Publish ke Redis Pub/Sub atau local hub.
4. Jika publish sukses, mark processed.
5. Jika publish gagal, mark failed atau retry sesuai strategi.

Interface:

```go
type RealtimePublisher interface {
    Publish(ctx context.Context, event Envelope) error
}
```

Checklist:

- [ ] Ubah `outbox.Processor` agar menerima optional realtime publisher.
- [ ] Buat mapper `OutboxEvent -> []Envelope`.
- [ ] Integrasikan di `cmd/worker/main.go`.
- [ ] Pertahankan behavior lama jika publisher nil.
- [ ] Tambah log event type dan topic, bukan payload sensitif.

## 13. API Lifecycle

`cmd/api/main.go` perlu start dan shutdown WebSocket service.

Flow:

```text
load config
connect DB
connect Redis optional
create websocket service
start websocket Redis subscriber
mount router
on shutdown:
  shutdown HTTP server
  shutdown websocket service
```

Checklist:

- [ ] Buat `websocketService.Start(ctx)`.
- [ ] Buat `websocketService.Shutdown(ctx)`.
- [ ] Start subscriber sebelum server listen.
- [ ] Shutdown hub setelah HTTP shutdown dimulai.
- [ ] Pastikan tidak ada goroutine leak.

## 14. Security Notes

Risiko utama:

- access token di query string bisa masuk log;
- subscribe topic tanpa authorization bisa bocorkan data project;
- client lambat bisa menumpuk memory;
- origin bebas bisa membuka koneksi dari domain tidak diinginkan;
- payload event bisa berisi data sensitif.

Mitigasi:

- allowed origins sementara `*` untuk development; saat production sebaiknya diganti whitelist domain frontend;
- jangan log full URL query untuk endpoint WebSocket;
- enforce topic authorization;
- gunakan read limit;
- gunakan send buffer kecil dan disconnect client lambat;
- payload event hanya berisi data yang dibutuhkan UI;
- access token tetap pendek masa berlakunya.

Checklist:

- [ ] Origin check.
- [ ] Token auth saat handshake.
- [ ] Topic authorization.
- [ ] Read limit.
- [ ] Send buffer limit.
- [ ] Sanitasi log.

## 15. Testing Strategy

Unit test:

- topic parser;
- topic authorization;
- envelope mapper dari outbox event;
- hub subscribe/unsubscribe/publish.

Integration/manual test:

- connect tanpa token harus gagal;
- connect token valid harus menerima `realtime.connected`;
- subscribe topic valid harus masuk `subscribed`;
- subscribe topic tidak valid harus masuk `rejected`;
- publish event ke topic harus diterima client subscribed;
- ping harus dibalas `realtime.pong`;
- disconnect harus cleanup topic;
- Redis off tidak membuat server crash.

Checklist:

- [ ] Tambah unit test parser topic.
- [ ] Tambah unit test hub.
- [ ] Tambah unit test outbox mapper.
- [ ] Jalankan `go test ./...`.
- [ ] Manual test dengan Postman WebSocket atau frontend kecil.

## 16. Dokumentasi Setelah Implementasi

Setelah kode dibuat, tambahkan dokumentasi final:

```text
Md/RealtimeWebSocket.md
```

Isi minimal:

- endpoint connect;
- auth token;
- message subscribe/unsubscribe/ping;
- event response;
- topic list;
- error behavior;
- contoh Postman;
- contoh frontend JavaScript.

Checklist:

- [ ] Buat docs API WebSocket.
- [ ] Update `Md/NextPlan.md` jika WebSocket menjadi item improvisasi baru.
- [ ] Update `Md/ImproveProject.md` bagian realtime monitoring.
- [ ] Cantumkan batasan tahap awal.

## 17. Urutan Eksekusi yang Disarankan

Tahap 1: Transport minimal

- [x] Tambah dependency.
- [x] Tambah config.
- [x] Buat envelope/message.
- [x] Buat hub/client.
- [x] Buat handler `/api/realtime/ws`.
- [x] Auth handshake.
- [x] Ping/pong.
- [x] Subscribe/unsubscribe in-memory.
- [x] Test connect otomatis dengan `httptest` dan WebSocket dialer.

Catatan implementasi Tahap 1:

- Origin WebSocket sementara allow all melalui `WEBSOCKET_ALLOWED_ORIGINS=*`.
- Query token yang didukung untuk WebSocket hanya `accessToken`.
- Header `Authorization: Bearer {token}` tetap didukung untuk Postman/tooling.
- Topic authorization belum dibuat dan masuk Tahap 2.
- Redis Pub/Sub dan outbox delivery belum dibuat dan masuk Tahap 3-4.
- Kontrak JSON Tahap 1 dicatat di `Md/RealtimeWebSocket.md`.

Tahap 2: Authorization dan topic project

- [x] Implement topic parser.
- [x] Implement topic authorization.
- [x] Tambah topic `user`, `project`, `ticket`, `payment`, `organization`.
- [x] Tambah unit test topic.

Catatan implementasi Tahap 2:

- Topic parser tersedia untuk `user.{userId}`, `project.{projectId}`, `project.{projectId}.monitoring`, `project.{projectId}.ticket`, `project.{projectId}.finance`, `ticket.{ticketId}`, `payment.{paymentRequestId}`, dan `organization.{organizationId}`.
- Subscribe sekarang melewati server-side authorization sebelum topic didaftarkan ke hub.
- Topic unauthorized masuk ke `data.rejected`, bukan menyebabkan koneksi langsung putus.
- Rule authorization memakai policy layer yang sama dengan REST API:
  - `user.{userId}` hanya untuk user itu sendiri;
  - `project.*` memakai policy project/monitoring/tracker/finance;
  - `ticket.{ticketId}` memakai `CanViewTicket`;
  - `payment.{paymentRequestId}` memakai `CanViewPaymentRequest`;
  - `organization.{organizationId}` memakai membership check.

Tahap 3: Redis fanout

- [x] Implement Redis publisher/subscriber.
- [x] Start subscriber di API.
- [x] Fallback saat Redis nil.
- [x] Test publish local fallback.

Catatan implementasi Tahap 3:

- Redis channel yang dipakai adalah `{WEBSOCKET_REDIS_CHANNEL_PREFIX}:events`.
- Default prefix mengikuti config `WEBSOCKET_REDIS_CHANNEL_PREFIX`, yang fallback ke `{REDIS_PREFIX}:ws`.
- `Service.Publish(ctx, envelope)` akan publish ke Redis jika Redis tersedia.
- API instance menjalankan Redis subscriber melalui `websocketSvc.Start(ctx)`.
- Subscriber Redis menerima envelope JSON lalu deliver ke local hub berdasarkan `topic`.
- Jika Redis tidak tersedia, `Publish` fallback ke local hub sehingga single instance tetap dapat mengirim event ke client yang terkoneksi.
- Integration test Redis live belum dibuat karena test suite saat ini tidak menyediakan Redis test container.

Tahap 4: Outbox delivery

- [ ] Buat outbox event mapper.
- [ ] Inject realtime publisher ke worker.
- [ ] Publish event dari outbox processor.
- [ ] Mark processed setelah publish sukses.
- [ ] Tambah test mapper.

Tahap 5: Docs dan hardening

- [ ] Update docs final.
- [ ] Tambah logging ringkas.
- [ ] Pastikan graceful shutdown.
- [ ] Jalankan `go test ./...`.
- [ ] Manual test Postman.

## 18. Definition of Done

Implementasi dianggap selesai jika:

- [ ] server expose `GET /api/realtime/ws`;
- [ ] client token valid menerima `realtime.connected`;
- [ ] client bisa subscribe dan unsubscribe topic;
- [ ] client bisa ping dan menerima pong;
- [ ] subscribe topic unauthorized ditolak;
- [ ] Redis Pub/Sub fanout berjalan saat Redis tersedia;
- [ ] outbox worker bisa publish minimal satu event domain ke WebSocket;
- [ ] server tetap berjalan saat Redis tidak tersedia;
- [ ] graceful shutdown menutup koneksi client;
- [ ] `go test ./...` pass;
- [ ] dokumentasi final tersedia.
