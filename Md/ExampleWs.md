## Config & Directories

Tech: nhooyr.io/websocket
Github: https://github.com/coder/websocket
Docs: https://pkg.go.dev/github.com/coder/websocket

### Prinsip Penempatan

WebSocket sebaiknya diperlakukan sebagai transport realtime lintas module, bukan bagian dari salah satu domain seperti `chat`, `ticket`, atau `generative_content`.

Rekomendasi struktur:

```text
internal/module/realtime/
  websocket/
    handler/
      handler.go
    service/
      service.go
      hub.go
      client.go
      envelope.go
      publisher.go
      subscription.go
```

Alasan:

- `internal/module/realtime/websocket` tetap mengikuti pola project saat ini: setiap feature/module punya `handler` dan `service`.
- WebSocket tidak ditempatkan di `internal/module/headless` karena ini bukan client/integrasi eksternal seperti WAHA, Cloudinary, OpenAI, atau queue.
- WebSocket tidak ditempatkan di `pkg` karena membutuhkan dependency internal seperti auth token, profile, Redis, logger, dan domain authorization.
- Module domain seperti `chat/whatsapp`, `ticket`, atau `generative_content/chat` cukup inject publisher interface, lalu publish event setelah transaksi DB berhasil.

### Shared Core

`internal/module/realtime/websocket/service` berisi komponen transport umum:

- `Hub`: registry connection aktif per instance.
- `Client`: wrapper connection `nhooyr.io/websocket`, metadata profile, role, business access, subscribed topics.
- `Envelope`: format standar message realtime.
- `Publisher`: interface yang dipakai module lain untuk publish event tanpa tahu detail websocket.
- `Subscription`: validasi dan mapping topic yang boleh di-subscribe oleh user.

Contoh bentuk interface yang module lain consume:

```go
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}
```

Module domain tidak boleh import `handler` websocket. Jika perlu realtime, module cukup menerima `websocket_service.Publisher` lewat DI di `internal.NewRouter` atau worker bootstrap.

### Redis Placement

Karena API bisa berjalan lebih dari satu instance, local in-memory hub saja tidak cukup. Gunakan Redis untuk fanout lintas instance.

Rekomendasi struktur Redis:

```text
internal/repository/redis/websocket_repository/
  repository.go
  pubsub.go
  presence.go
```

Tanggung jawab:

- Pub/Sub channel untuk broadcast event antar API instance.
- Optional presence key untuk tracking user online.
- Optional connection/session metadata dengan TTL.

Rekomendasi channel:

```text
postmatic:ws:events
postmatic:ws:presence:{profileId}
```

Catatan penting:

- `Hub` tetap menyimpan connection aktif secara in-memory karena object websocket tidak bisa disimpan di Redis.
- Redis hanya untuk publish event dan metadata ringan.
- Jika satu event dibuat dari worker/asynq, worker publish ke Redis; API instance yang punya connection aktif akan deliver ke client.

### Config

Tambahkan config WebSocket di `config.Config`, mengikuti pola config yang sudah ada.

Rekomendasi env:

```env
WEBSOCKET_ENABLED=true
WEBSOCKET_ALLOWED_ORIGINS=https://workspace.postmatic.id,https://auth.postmatic.id
WEBSOCKET_READ_LIMIT_BYTES=65536
WEBSOCKET_WRITE_TIMEOUT_SECONDS=10
WEBSOCKET_IDLE_TIMEOUT_SECONDS=60
WEBSOCKET_PING_INTERVAL_SECONDS=25
WEBSOCKET_CLIENT_SEND_BUFFER=64
WEBSOCKET_REDIS_CHANNEL_PREFIX=postmatic:ws
WEBSOCKET_PRESENCE_TTL_SECONDS=90
```

Mapping ke `config.Config`:

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

Default yang aman:

- `WEBSOCKET_ENABLED`: `true`
- `WEBSOCKET_READ_LIMIT_BYTES`: `65536`
- `WEBSOCKET_WRITE_TIMEOUT_SECONDS`: `10`
- `WEBSOCKET_IDLE_TIMEOUT_SECONDS`: `60`
- `WEBSOCKET_PING_INTERVAL_SECONDS`: `25`
- `WEBSOCKET_CLIENT_SEND_BUFFER`: `64`
- `WEBSOCKET_PRESENCE_TTL_SECONDS`: `90`

### Routing

Rekomendasi route:

```text
GET /api/realtime/ws
```

Atau jika ingin pendek:

```text
GET /api/ws
```

Lebih disarankan `GET /api/realtime/ws` karena tidak bentrok dengan module bisnis existing dan lebih jelas secara domain.

Mounting dilakukan di `internal.NewRouter`, sejajar dengan module lain:

```go
r.Route("/realtime", func(r chi.Router) {
	r.Mount("/ws", websocketHandler.Routes())
})
```

Untuk auth:

- Jangan pakai `AuthMiddleware` biasa secara langsung jika handshake perlu handling khusus.
- Reuse `token.TokenMaker` untuk validasi access token di websocket handler/service.
- Browser native `WebSocket` tidak bisa set custom `Authorization` header dengan mudah, jadi support minimal salah satu:
  - `?accessToken=...`
  - cookie auth jika nanti auth web sudah final
  - `Sec-WebSocket-Protocol` untuk token jika ingin menghindari query token

Untuk tahap awal, paling praktis:

```text
GET /api/realtime/ws?accessToken={accessToken}
```

Jika digunakan di production, pastikan token tidak pernah ditulis ke log request mentah.

### Consumer Usage: Frontend/Postman

WebSocket memiliki dua tahap:

1. Connect ke endpoint websocket dengan access token.
2. Kirim message `subscribe` untuk topic yang ingin diterima.

Tanpa message `subscribe`, client hanya akan menerima event system seperti `realtime.connected` dan `realtime.pong`.

#### Connect URL

Development/local:

```text
ws://localhost:2001/api/realtime/ws?accessToken={accessToken}
```

Production/staging:

```text
wss://api.postmatic.id/api/realtime/ws?accessToken={accessToken}
wss://api-staging.postmatic.id/api/realtime/ws?accessToken={accessToken}
```

Token yang digunakan adalah access token login biasa.

Alternative query yang juga didukung:

```text
wss://api.postmatic.id/api/realtime/ws?postmaticAccessToken={accessToken}
```

Header fallback yang juga didukung jika client/tool memungkinkan:

```http
Authorization: Bearer {accessToken}
```

#### Initial Server Event

Jika connect berhasil, server akan mengirim event system:

```json
{
  "type": "realtime.connected",
  "topic": "realtime.system",
  "data": {
    "clientId": "generated-client-id",
    "topics": []
  },
  "sentAt": "2026-06-25T10:00:00Z"
}
```

#### Subscribe Topic

Untuk mulai menerima event feature tertentu, client harus mengirim message berikut melalui websocket:

```json
{
  "type": "subscribe",
  "topic": "profile.{profileId}"
}
```

Atau subscribe banyak topic sekaligus:

```json
{
  "type": "subscribe",
  "topics": [
    "profile.{profileId}",
    "business.{businessRootId}.member"
  ]
}
```

Server akan membalas acknowledgement:

```json
{
  "type": "realtime.subscription_updated",
  "topic": "realtime.system",
  "data": {
    "subscribed": [
      "profile.{profileId}"
    ],
    "unsubscribed": [],
    "rejected": []
  },
  "sentAt": "2026-06-25T10:00:00Z"
}
```

Jika topic tidak boleh diakses, topic masuk ke `rejected`:

```json
{
  "type": "realtime.subscription_updated",
  "topic": "realtime.system",
  "data": {
    "subscribed": [],
    "unsubscribed": [],
    "rejected": [
      "business.999.member"
    ]
  },
  "sentAt": "2026-06-25T10:00:00Z"
}
```

#### Unsubscribe Topic

```json
{
  "type": "unsubscribe",
  "topic": "profile.{profileId}"
}
```

Atau banyak topic:

```json
{
  "type": "unsubscribe",
  "topics": [
    "profile.{profileId}",
    "business.{businessRootId}.member"
  ]
}
```

#### Ping/Pong

Client harus mengirim app-level ping secara berkala agar koneksi tidak dianggap idle oleh server.
Dengan default `WEBSOCKET_IDLE_TIMEOUT_SECONDS=60`, gunakan interval yang lebih kecil dari 60 detik, misalnya 25-30 detik.

Penting:

- Event yang dikirim server ke client tidak mereset idle timeout.
- WebSocket protocol ping frame dari tool/browser belum tentu terbaca sebagai app message.
- Kirim text JSON berikut dari client:

```json
{
  "type": "ping"
}
```

Server akan membalas:

```json
{
  "type": "realtime.pong",
  "topic": "realtime.system",
  "data": {
    "receivedAt": "2026-06-25T10:00:00Z"
  },
  "sentAt": "2026-06-25T10:00:00Z"
}
```

#### Frontend Example

```js
const ws = new WebSocket(
  `wss://api.postmatic.id/api/realtime/ws?accessToken=${accessToken}`
)
let wsHeartbeat

ws.onopen = () => {
  ws.send(JSON.stringify({
    type: 'subscribe',
    topic: `profile.${profileId}`
  }))

  wsHeartbeat = setInterval(() => {
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'ping' }))
    }
  }, 30000)
}

ws.onclose = () => {
  clearInterval(wsHeartbeat)
}

ws.onmessage = (event) => {
  const message = JSON.parse(event.data)

  if (message.type === 'account.session.force_logout') {
    const data = message.data
    if (data.allSession || data.sessionId === currentSessionId) {
      // clear auth state and redirect to login
    }
  }
}
```

#### Postman Example

1. Buat request baru dengan type `WebSocket`.
2. Isi URL:

```text
wss://api-staging.postmatic.id/api/realtime/ws?accessToken={accessToken}
```

3. Klik `Connect`.
4. Setelah menerima `realtime.connected`, kirim message subscribe:

```json
{
  "type": "subscribe",
  "topic": "profile.{profileId}"
}
```

5. Pastikan server membalas `realtime.subscription_updated` dan topic masuk ke `data.subscribed`.
6. Kirim message ping berikut setiap 25-30 detik agar koneksi tidak disconnect karena idle timeout:

```json
{
  "type": "ping"
}
```

7. Untuk test force logout, lakukan revoke/logout session dari client/API lain. Client websocket yang subscribe `profile.{profileId}` akan menerima event `account.session.force_logout`.

### Event Contract Placement

Core websocket hanya tahu format envelope umum. Payload detail tetap milik module asal event.

Rekomendasi envelope umum:

```json
{
  "type": "chat.whatsapp.message.created",
  "topic": "chat.whatsapp.room.123",
  "data": {},
  "sentAt": "2026-06-25T10:00:00Z"
}
```

Penempatan payload/event builder:

```text
internal/module/chat/whatsapp/service/realtime.go
internal/module/ticket/whatsapp/service/realtime.go
internal/module/generative_content/chat/service/realtime.go
```

Aturan:

- `type` event menggunakan format dotted: `{module}.{resource}.{action}`.
- `topic` harus cukup spesifik untuk authorization.
- Module asal bertanggung jawab membentuk payload.
- WebSocket core bertanggung jawab mengirim, subscribe, unsubscribe, ping/pong, dan cleanup connection.

### Topic Authorization

Topic authorization jangan ditaruh di frontend. Backend harus validasi subscribe topic berdasarkan profile dan akses bisnis.

Contoh topic:

```text
profile.{profileId}
business.{businessRootId}
chat.whatsapp.room.{whatsappRoomChatId}
ticket.{ticketId}
generative_content.chat.session.{sessionId}
```

Rekomendasi rules:

- `profile.{profileId}` hanya untuk profile itu sendiri atau admin.
- `business.{businessRootId}` hanya untuk member accepted business tersebut atau admin.
- `chat.whatsapp.room.{id}` hanya admin untuk saat ini, mengikuti route `chat/whatsapp` existing.
- `ticket.{id}` hanya admin atau profile pemilik ticket jika ticket website user-facing.

### Dependency Injection

Tambahkan websocket service di dua tempat jika sudah implementasi:

- `internal.NewRouter`: untuk HTTP websocket endpoint dan publisher dari request lifecycle.
- `cmd/api/main.go`: untuk worker/asynq yang perlu publish event realtime dari background job.

Pola ideal:

```text
Redis Pub/Sub Repository
        ↓
WebSocket Service / Publisher
        ↓
Injected into module services that emit realtime event
```

Jika worker perlu publish event tetapi tidak menerima connection langsung, worker cukup publish ke Redis channel. API instance yang menjalankan websocket hub akan menerima event dan deliver ke client.

### Shutdown Lifecycle

Karena `cmd/api/main.go` sudah punya graceful shutdown, websocket service perlu expose:

```go
Start(ctx context.Context) error
Shutdown(ctx context.Context) error
```

Rekomendasi lifecycle:

- Start Redis subscriber saat app boot.
- Close semua websocket client saat `srv.Shutdown`.
- Stop goroutine hub/subscriber sebelum process exit.
- Jangan biarkan connection write goroutine menggantung.

### Dokumentasi Module

Jika implementasi sudah dibuat, dokumentasi user-facing lebih baik dipisah:

```text
Md/Module/Realtime/Realtime.WebSocket.md
```

Sedangkan event spesifik module tetap dicatat di docs module masing-masing, misalnya:

```text
Md/Module/Chat/Chat.Whatsapp.md
Md/Module/Ticket/Ticket.Whatsapp.md
Md/Module/GenerativeContent/GenerativeContent.Chat.md
```

`WebSocketPlan.md` digunakan sebagai planning teknis lintas module, bukan dokumentasi API final.

## Feature yang ingin diimplementasikan

### Session Logout

Ketika Client A melakukan logout/revoke untuk session tertentu milik profile yang sama, server harus menghapus refresh token/session terlebih dahulu.
Setelah berhasil, server publish event realtime agar Client B yang sedang subscribe dapat melakukan force logout di UI.

Digunakan pada:
`internal/module/account/session/service/service.go`

Type:
`account.session.force_logout`

Topic:
`profile.{profileId}`

Data:
```json
{
  "sessionId": "target-session-id",
  "allSession": false,
  "forceLogout": true,
  "reason": "SESSION_REVOKED_BY_USER"
}
```

Untuk logout all session:

```json
{
  "sessionId": null,
  "allSession": true,
  "forceLogout": true,
  "reason": "ALL_SESSIONS_REVOKED_BY_USER"
}
```

Client behavior:
Jika client menerima event `account.session.force_logout` dan `data.allSession` bernilai `true`, client clear auth state dan redirect ke login. Jika `data.allSession` bernilai `false`, client melakukan clear auth state hanya jika `data.sessionId` sama dengan session aktif client tersebut.

### Ping/Pong

Client dapat mengirim ping message untuk memastikan koneksi websocket masih hidup.

Client message:

```json
{
  "type": "ping"
}
```

Server event:

```json
{
  "type": "realtime.pong",
  "topic": "realtime.system",
  "data": {
    "receivedAt": "2026-06-25T10:00:00Z"
  },
  "sentAt": "2026-06-25T10:00:00Z"
}
```

### Business Member Invitation

Ketika admin mengundang member atau pengguna menjawab invitation member, server publish event realtime ke business terkait agar client yang sedang membuka workspace business tersebut dapat memperbarui member list.

Digunakan pada:
`internal/module/business/business_member/service/service.go`

Types:
`business.member.invited`
`business.member.invitation_answered`

Topic:
`business.{businessRootId}.member`

Data untuk `business.member.invited`:

```json
{
  "businessRootId": 0,
  "businessName": "Business Name",
  "memberId": 0,
  "profileId": "profile-id",
  "profileName": "Profile Name",
  "profileEmail": "user@example.com",
  "role": "member",
  "status": "pending",
  "invitedAt": "2026-06-25T10:00:00Z",
  "reason": "MEMBER_INVITED"
}
```

Catatan:
Payload invite tidak mengirim `invitationLink` karena link tersebut berisi token invitation.

Data untuk `business.member.invitation_answered`:

```json
{
  "businessRootId": 0,
  "businessName": "Business Name",
  "memberId": 0,
  "profileId": "profile-id",
  "profileName": "Profile Name",
  "profileEmail": "user@example.com",
  "role": "member",
  "status": "accepted|rejected",
  "answeredAt": "2026-06-25T10:00:00Z",
  "reason": "MEMBER_INVITATION_ACCEPTED|MEMBER_INVITATION_REJECTED"
}
```

Client behavior:
Client yang subscribe topic `business.{businessRootId}.member` dapat melakukan fetch ulang member list atau update member item berdasarkan payload. Untuk invite, UI bisa menambahkan item pending. Untuk accept/reject, UI bisa mengubah status item member terkait.

### Notifikasi Pembayaran dari Midtrans

Untuk client yang melakukan pembelian produk/token, client dapat subscribe ke payment tertentu agar UI mengetahui perubahan status pembayaran secara realtime.
Server publish event setelah status payment berhasil diperbarui, baik dari webhook Midtrans maupun flow pengecekan status payment yang ada di get by id.

Digunakan pada:
`internal/module/payment/common/service/service.go`

Type:
`payment.status_changed`

Topic:
`payment.{paymentHistoryId}`

Optional related topic:
`business.{businessRootId}.payment`

Data:
```json
{
  "paymentHistoryId": "payment-history-id",
  "businessRootId": 0,
  "profileId": "profile-id",
  "previousStatus": "pending",
  "status": "success|failed|expired|canceled|denied|refunded",
  "productName": "Product Name",
  "productAmount": 0,
  "totalAmount": 0,
  "currency": "IDR",
  "source": "midtrans_webhook|midtrans_status_check",
  "changedAt": "2026-06-25T10:00:00Z",
  "reason": "PAYMENT_STATUS_CHANGED"
}
```

Topic subscription:
- `payment.{paymentHistoryId}` untuk client checkout spesifik payment. Non-admin hanya bisa subscribe payment miliknya.
- `business.{businessRootId}.payment` untuk workspace business yang ingin menerima update pembayaran business terkait. Non-admin harus accepted member di business tersebut.

Client behavior:
Client checkout yang subscribe topic `payment.{paymentHistoryId}` dapat memperbarui status payment. Jika payment sukses dan berdampak ke business token/access, workspace dapat subscribe topic business terkait atau fetch ulang data token/access.

### Chat AI Generate Image

Untuk pengguna yang sedang melakukan generate image dari chat AI, client dapat subscribe ke session/proses generate untuk mendapatkan progress dan hasil secara realtime.
Progress percentage disimpan di `generative_content_chat_bubbles` pada user bubble, dicache sebagai last-progress di Redis, dan dikirim realtime melalui WebSocket.

Digunakan pada:
`internal/module/generative_content/chat/service/service.go`

Type:
`generative_content.chat.progress`

Topic:
`generative_content.chat.{chatSessionId}`

Authorization:
- Admin dapat subscribe semua chat topic.
- Non-admin hanya bisa subscribe jika accepted member pada business pemilik chat session.

Data:
```json
{
  "businessRootId": 0,
  "chatSessionId": 0,
  "userBubbleId": 0,
  "systemBubbleId": null,
  "generatedImagePostId": null,
  "processState": "queued|processing|success|failed",
  "progressStep": "queued|validating|loading_context|building_prompt|generating_image|uploading_result|saving_result|deducting_token|completed|failed",
  "progressPercentage": 0,
  "message": "Image generation queued",
  "imageUrls": [],
  "errorMessage": null,
  "updatedAt": "2026-06-25T10:00:00Z",
  "reason": "GENERATIVE_CONTENT_CHAT_PROGRESS"
}
```

Persisted state:
- `generative_content_chat_bubbles.process_state`
- `generative_content_chat_bubbles.progress_step`
- `generative_content_chat_bubbles.progress_percentage`

Redis last progress:
- Key: `generative_chat_progress:{chatSessionId}:{userBubbleId}`
- TTL: 30 menit

Progress step mapping:
- `queued`: task berhasil masuk queue.
- `validating`: worker memvalidasi bubble, model, dan akses premium.
- `loading_context`: worker membaca chat history, business knowledge, product, RSS, dan reference image.
- `building_prompt`: worker membangun final prompt.
- `generating_image`: request ke provider AI berjalan.
- `uploading_result`: hasil image sedang diupload ke Cloudinary.
- `saving_result`: system bubble, generated post, dan post items sedang disimpan.
- `deducting_token`: token usage sedang dicatat.
- `completed`: hasil final tersedia.
- `failed`: proses gagal, `errorMessage` berisi penyebab.

Client behavior:
Client yang sedang membuka chat session dapat menampilkan progress loading pada user bubble. Saat `processState=success`, client dapat memakai `systemBubbleId`, `generatedImagePostId`, dan `imageUrls`, atau fetch ulang detail chat. Saat `processState=failed`, client menampilkan `errorMessage`.

### Website Scrapper

Untuk profile yang sedang melakukan website scrapping, client dapat subscribe ke proses scrapper tertentu agar UI mengetahui progress, success, atau error secara realtime.
Karena scrapper berjalan melalui queue, event harus bisa dipublish dari worker ke Redis Pub/Sub agar API instance yang memegang websocket dapat deliver ke client.

Digunakan pada:
`internal/module/app/website_scrapper/service/service.go`

Type:
`website_scrapper.progress`

Topic:
`website_scrapper.{scrapperBusinessInformationId}`

Data:
```json
{
  "scrapperBusinessInformationId": 69,
  "profileId": "228a6e2f-136c-4d64-a649-34342cc0a5b04",
  "url": "https://www.example.com",
  "status": "pending|success|error",
  "processState": "queued|scrapping|cleaning|ai_generating_openai|ai_generating_gemini|saving|completed|failed",
  "progressStep": "queued|normalizing_url|rendering_website|cleaning_content|extracting_with_openai|extracting_with_gemini|saving_result|completed|failed",
  "progressPercentage": 0,
  "message": "Rendering website page",
  "errorReason": null,
  "reason": "WEBSITE_SCRAPPER_PROGRESS",
  "updatedAt": "2026-06-25T12:00:00Z"
}
```

Authorization:
- Admin dapat subscribe semua topic website scrapper.
- User biasa hanya dapat subscribe topic scrapper miliknya, berdasarkan `scrapper_business_informations.profile_id`.

State persistence:
- DB: `scrapper_business_informations.progress_step` dan `scrapper_business_informations.progress_percentage`.
- Redis snapshot: `website_scrapper_progress:{scrapperBusinessInformationId}` dengan TTL 30 menit.

Step mapping:
- `queued` (`0%`): job dibuat dan masuk queue.
- `normalizing_url` (`10%`): URL divalidasi dan dinormalisasi.
- `rendering_website` (`25%`): headless browser sedang render halaman.
- `cleaning_content` (`55%`): HTML dibersihkan, metadata/images/markdown diekstrak.
- `extracting_with_openai` (`75%`): AI extraction menggunakan OpenAI.
- `extracting_with_gemini` (`82%`): fallback extraction menggunakan Gemini.
- `saving_result` (`92%`): hasil disimpan ke tabel scrapper.
- `completed` (`100%`): proses sukses dan data tersedia.
- `failed` (`100%`): proses gagal, `errorReason` berisi penyebab ringkas.

Client behavior:
Client yang subscribe topic scrapper tersebut dapat memperbarui loading state, menampilkan error detail, atau fetch detail scrapper saat proses selesai.

### Chat Whatsapp

Untuk client admin, setiap ada pesan WhatsApp baru dari webhook WAHA, server publish event realtime agar UI dapat menampilkan notification dan append message jika sedang membuka room terkait.
Event ini hanya boleh diterima admin untuk saat ini, mengikuti akses route `chat/whatsapp` existing.

Digunakan pada:
`internal/module/ticket/common/service/service.go` pada saat handle webhook WAHA, atau service WhatsApp chat yang sesuai saat implementasi.

Type:
`chat.whatsapp.room.upserted`
`chat.whatsapp.message.created`
`chat.whatsapp.message.updated`

Global admin topic:
`chat.whatsapp.admin`

Room detail topic:
`chat.whatsapp.room.{whatsappRoomChatId}`

Data:
```json
{
  "whatsappRoomChatId": 1,
  "whatsappMessageChatId": 12,
  "room": {
    "id": 1,
    "chatId": "244048631738500@lid",
    "roomName": "Customer Name",
    "countryCode": "62",
    "phone": "85156031385",
    "createdAt": "2026-06-19T04:00:00Z",
    "updatedAt": "2026-06-19T04:00:00Z"
  },
  "message": {
    "id": 12,
    "whatsappRoomChatId": 1,
    "ticketId": null,
    "profileId": null,
    "profile": null,
    "messageId": "false_244048631738500@lid_AC9B59...",
    "quotedWhatsappMessageId": 11,
    "quotedWhatsappMessage": {
      "id": 11,
      "whatsappRoomChatId": 1,
      "ticketId": null,
      "profileId": null,
      "messageId": "false_244048631738500@lid_AC8A86...",
      "senderType": "customer",
      "body": "message yang di-quote",
      "sentStatus": "delivered",
      "mediaUrl": null,
      "mediaMimeType": null,
      "mediaFilename": null,
      "timestamp": "2026-06-19T04:00:00Z",
      "createdAt": "2026-06-19T04:00:00Z"
    },
    "senderType": "customer",
    "body": "pesan masuk",
    "sentStatus": "delivered",
    "mediaUrl": null,
    "mediaMimeType": null,
    "mediaFilename": null,
    "pendingAt": null,
    "sentAt": "2026-06-19T04:00:00Z",
    "deliveredAt": "2026-06-19T04:00:00Z",
    "readAt": null,
    "timestamp": "2026-06-19T04:00:00Z",
    "timestampDate": "2026-06-19T00:00:00Z",
    "createdAt": "2026-06-19T04:00:00Z",
    "updatedAt": "2026-06-19T04:00:00Z"
  },
  "reason": "WHATSAPP_MESSAGE_CREATED"
}
```

Event source:
- `chat.whatsapp.room.upserted`: admin membuat/mengambil room via `POST /api/chat/whatsapp`.
- `chat.whatsapp.message.created`: incoming webhook WAHA berhasil insert message, atau admin membuat outgoing message/reply/blast yang masih `pending`.
- `chat.whatsapp.message.updated`: worker WAHA berhasil mengirim outgoing message, atau webhook ack/status memperbarui `sentStatus`.

Payload notes:
- Untuk `chat.whatsapp.room.upserted`, field `message` dan `whatsappMessageChatId` tidak dikirim.
- Incoming media dari WAHA sudah berupa URL Cloudinary jika upload berhasil.
- `quotedWhatsappMessage` dikirim jika source quote berhasil ditemukan.

Client behavior:
Client admin yang subscribe `chat.whatsapp.admin` dapat menampilkan notification/badge. Client yang sedang membuka room dan subscribe `chat.whatsapp.room.{whatsappRoomChatId}` dapat append/update bubble message.

### Chat Website

Untuk client admin, setiap ada ticket website message/reply baru, server publish event realtime agar UI dapat menampilkan notification dan append message jika sedang membuka ticket terkait.

Digunakan pada:
`internal/module/ticket/website/service/service.go` atau `internal/module/ticket/common/service/service.go` sesuai titik pembuatan message saat implementasi.

Type:
`chat.website.message.created`

Global admin topic:
`chat.website.admin`

Ticket detail topic:
`ticket.{ticketId}`

Data:
```json
{
  "ticketId": 123,
  "ticketWebsiteMessageId": 456,
  "ticket": {
    "id": 123,
    "appTicketCategoryId": 1,
    "profileId": "profile-id",
    "channel": "website",
    "priority": "medium",
    "slaStatus": "open",
    "whatsappRoomChatId": null,
    "subject": "Tidak bisa top up",
    "body": "Payment saya belum masuk.",
    "countryCode": "ID",
    "phone": "81234567890",
    "email": "user@example.com",
    "attachments": [],
    "createdAt": "2026-06-25T10:00:00Z",
    "updatedAt": "2026-06-25T10:00:00Z"
  },
  "message": {
    "id": 456,
    "ticketId": 123,
    "profileId": "profile-id",
    "profile": null,
    "senderType": "customer|customer_service|agent",
    "body": "Baik, kami cek terlebih dahulu.",
    "attachments": [],
    "quotedWebMessageId": null,
    "createdAt": "2026-06-25T10:00:00Z",
    "updatedAt": "2026-06-25T10:00:00Z"
  },
  "reason": "WEBSITE_TICKET_MESSAGE_CREATED"
}
```

Authorization:
- `chat.website.admin` hanya admin.
- `ticket.{ticketId}` dapat disubscribe admin atau profile pemilik ticket website tersebut.

Event source:
- `chat.website.message.created`: reply website ticket berhasil disimpan via `POST /api/ticket/website/{id}/reply`.

Payload notes:
- `message.profile` mengikuti mapper existing. Pada event dari insert langsung, nilainya bisa `null`; client dapat fetch detail ticket jika perlu profile lengkap.
- `quotedWebMessageId` dikirim sebagai ID saja sesuai response existing, belum ada quoted message snapshot.

Client behavior:
Client admin yang subscribe `chat.website.admin` dapat menampilkan notification/badge. Client yang sedang membuka ticket dan subscribe `ticket.{ticketId}` dapat append/update message.

### Ticket

Untuk client admin, setiap ada ticket baru atau perubahan ticket, server publish event realtime agar admin lain dapat memperbarui list/detail ticket tanpa refresh manual.

Digunakan pada:
`internal/module/ticket/common/service/service.go`, `internal/module/ticket/website/service/service.go`, atau service ticket terkait sesuai titik create/update saat implementasi.

Type:
`ticket.created`
`ticket.updated`
`ticket.status_changed`

Global admin topic:
`ticket.admin`

Ticket detail topic:
`ticket.{ticketId}`

Data:
```json
{
  "ticketId": 123,
  "ticket": {
    "id": 123,
    "appTicketCategoryId": 1,
    "profileId": "profile-id",
    "channel": "website|whatsapp|email",
    "priority": "low|medium|high",
    "slaStatus": "open|pending|in_progress|resolved",
    "whatsappRoomChatId": null,
    "subject": "Tidak bisa top up",
    "body": "Payment saya belum masuk.",
    "countryCode": "ID",
    "phone": "81234567890",
    "email": "user@example.com",
    "attachments": [],
    "createdAt": "2026-06-25T10:00:00Z",
    "updatedAt": "2026-06-25T10:00:00Z"
  },
  "beforeStatus": "open",
  "afterStatus": "in_progress",
  "changedByProfileId": "admin-profile-id",
  "reason": "TICKET_STATUS_CHANGED"
}
```

Payload notes:
- `ticket.created` mengirim `ticket` dan `reason=TICKET_CREATED`.
- `ticket.status_changed` mengirim `ticket`, `beforeStatus`, `afterStatus`, `changedByProfileId`, dan `reason=TICKET_STATUS_CHANGED`.
- `ticket.updated` reserved untuk endpoint update metadata/category/priority ticket yang belum tersedia.
- Jika request change status mengirim status yang sama dengan status saat ini, server tidak publish event baru.

Event source:
- `ticket.created`: setelah `TicketCommonService.CreateTicket` berhasil insert ticket. Saat ini dipakai website ticket dan create ticket dari WhatsApp message.
- `ticket.status_changed`: setelah `TicketCommonService.ChangeStatus` berhasil commit update status dan membuat row `ticket_changes`.

Authorization:
- `ticket.admin` hanya admin.
- `ticket.{ticketId}` dapat disubscribe admin atau profile pemilik ticket website tersebut.

Client behavior:
Client admin yang subscribe `ticket.admin` dapat update list/badge. Client yang sedang membuka detail ticket dan subscribe `ticket.{ticketId}` dapat update state detail ticket.
