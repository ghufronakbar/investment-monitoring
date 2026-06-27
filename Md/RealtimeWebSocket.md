# Realtime WebSocket

Dokumentasi ini mencatat kontrak WebSocket yang sudah tersedia pada implementasi saat ini.

## Endpoint

```text
GET /api/realtime/ws?accessToken={accessToken}
```

Alternatif untuk tool seperti Postman:

```http
Authorization: Bearer {accessToken}
```

Catatan:

- Query token yang didukung adalah `accessToken`.
- `postmaticAccessToken` tidak dipakai untuk WebSocket project ini.
- Origin sementara allow all melalui config `WEBSOCKET_ALLOWED_ORIGINS=*`.

## Initial Event

Setelah connect berhasil, server mengirim:

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

Tipe data `data`:

```go
type ConnectedData struct {
    ClientID string
    UserID   uuid.UUID
    Topics   []string
}
```

## Ping/Pong

Client message:

```json
{
  "type": "ping"
}
```

Server response:

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

Tipe data `data`:

```go
type PongData struct {
    ReceivedAt time.Time
}
```

## Subscribe

Subscribe melewati authorization di backend. Topic yang valid tetapi tidak boleh diakses user akan masuk ke `data.rejected`.

Topic yang didukung:

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

Rule authorization:

- `user.{userId}` hanya untuk user itu sendiri.
- `project.{projectId}` memakai akses project umum.
- `project.{projectId}.monitoring` memakai akses monitoring project.
- `project.{projectId}.ticket` memakai akses tracker project.
- `project.{projectId}.finance` memakai akses finance project.
- `ticket.{ticketId}` memakai akses view ticket.
- `payment.{paymentRequestId}` memakai akses view payment request.
- `organization.{organizationId}` hanya untuk member organization tersebut.

Client message single topic:

```json
{
  "type": "subscribe",
  "topic": "project.{projectId}.monitoring"
}
```

Client message multi topic:

```json
{
  "type": "subscribe",
  "topics": [
    "project.{projectId}.monitoring",
    "project.{projectId}.ticket"
  ]
}
```

Server acknowledgement:

```json
{
  "type": "realtime.subscription_updated",
  "topic": "realtime.system",
  "data": {
    "subscribed": [
      "project.{projectId}.monitoring"
    ],
    "unsubscribed": [],
    "rejected": []
  },
  "sentAt": "2026-06-27T00:00:00Z"
}
```

## Unsubscribe

Client message:

```json
{
  "type": "unsubscribe",
  "topic": "project.{projectId}.monitoring"
}
```

Server acknowledgement memakai event yang sama:

```json
{
  "type": "realtime.subscription_updated",
  "topic": "realtime.system",
  "data": {
    "subscribed": [],
    "unsubscribed": [
      "project.{projectId}.monitoring"
    ],
    "rejected": []
  },
  "sentAt": "2026-06-27T00:00:00Z"
}
```

Tipe data `data`:

```go
type SubscriptionUpdatedData struct {
    Subscribed   []string
    Unsubscribed []string
    Rejected     []string
}
```

## Error Event

Jika client mengirim JSON invalid, message type tidak dikenal, atau message type WebSocket bukan text, server mengirim:

```json
{
  "type": "realtime.error",
  "topic": "realtime.system",
  "data": {
    "message": "INVALID_JSON"
  },
  "sentAt": "2026-06-27T00:00:00Z"
}
```

Tipe data `data`:

```go
type ErrorData struct {
    Message string
}
```

## Batasan Saat Ini

- Tidak ada replay untuk client yang offline.
- Presence belum dibuat.
- Manual test tetap disarankan untuk validasi browser/proxy karena automated test memakai `httptest`.

## Redis Fanout

Jika Redis tersedia, server memakai Pub/Sub untuk fanout event realtime lintas API instance.

Channel:

```text
{WEBSOCKET_REDIS_CHANNEL_PREFIX}:events
```

Payload yang dipublish ke Redis memakai envelope yang sama dengan payload WebSocket:

```json
{
  "type": "ticket.created",
  "topic": "project.{projectId}.ticket",
  "data": {},
  "sentAt": "2026-06-27T00:00:00Z"
}
```

Jika Redis tidak tersedia, server tetap berjalan dan `Publish` fallback ke local hub. Fallback ini cukup untuk single API instance, tetapi tidak mengirim event lintas instance.

## Outbox Domain Events

Outbox worker sekarang mengubah event domain menjadi envelope WebSocket dan publish ke realtime publisher.

Tipe `data` untuk event domain:

```json
{
  "outboxEventId": "uuid",
  "eventType": "TicketCreated",
  "aggregateType": "ticket",
  "aggregateId": "uuid",
  "payload": {
    "ticketId": "uuid",
    "projectId": "uuid"
  }
}
```

Mapping event:

| Outbox Event | WebSocket Type | Topics |
|---|---|---|
| `ProgressSubmitted` | `progress.submitted` | `project.{projectId}.monitoring` |
| `ProgressValidated` | `progress.validated` | `project.{projectId}.monitoring` |
| `TicketCreated` | `ticket.created` | `project.{projectId}.ticket`, `ticket.{ticketId}` |
| `TicketAssigned` | `ticket.assigned` | `project.{projectId}.ticket`, `ticket.{ticketId}` |
| `TicketResolved` | `ticket.resolved` | `project.{projectId}.ticket`, `ticket.{ticketId}` |
| `TicketClosed` | `ticket.closed` | `project.{projectId}.ticket`, `ticket.{ticketId}` |
| `PaymentApproved` | `payment.approved` | `project.{projectId}.finance`, `payment.{paymentRequestId}` |
| `PaymentPaid` | `payment.paid` | `project.{projectId}.finance`, `payment.{paymentRequestId}` |
| `MemberInvited` | `organization.member.invited` | `organization.{organizationId}`, `user.{userId}` |

Jika outbox payload belum memiliki routing metadata seperti `projectId`, mapper akan melakukan query fallback berdasarkan aggregate id.

## Frontend Example

```js
const ws = new WebSocket(
  `ws://localhost:2001/api/realtime/ws?accessToken=${encodeURIComponent(accessToken)}`
)

let heartbeat

ws.onopen = () => {
  ws.send(JSON.stringify({
    type: 'subscribe',
    topics: [
      `user.${userId}`,
      `project.${projectId}.monitoring`,
      `project.${projectId}.ticket`
    ]
  }))

  heartbeat = setInterval(() => {
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'ping' }))
    }
  }, 30000)
}

ws.onmessage = (event) => {
  const message = JSON.parse(event.data)

  switch (message.type) {
    case 'realtime.connected':
      break
    case 'realtime.subscription_updated':
      break
    case 'progress.submitted':
    case 'progress.validated':
    case 'ticket.created':
    case 'ticket.assigned':
    case 'ticket.resolved':
    case 'ticket.closed':
    case 'payment.approved':
    case 'payment.paid':
      // refetch dashboard/list/detail sesuai topic
      break
  }
}

ws.onclose = () => {
  clearInterval(heartbeat)
}
```

## Postman Manual Test

1. Login via REST API dan ambil access token.
2. Buat request baru dengan type `WebSocket`.
3. Connect ke URL:

```text
ws://localhost:{PORT}/api/realtime/ws?accessToken={accessToken}
```

4. Setelah connect, pastikan menerima:

```json
{
  "type": "realtime.connected"
}
```

5. Kirim ping:

```json
{
  "type": "ping"
}
```

6. Pastikan server membalas:

```json
{
  "type": "realtime.pong"
}
```

7. Subscribe topic yang valid:

```json
{
  "type": "subscribe",
  "topic": "user.{yourUserId}"
}
```

8. Subscribe topic yang tidak boleh diakses untuk memastikan topic masuk `rejected`:

```json
{
  "type": "subscribe",
  "topic": "user.{otherUserId}"
}
```

9. Jalankan aksi domain, misalnya create ticket atau submit progress, lalu pastikan client yang subscribe topic terkait menerima event domain.

## Operational Notes

- Access token boleh dikirim lewat query `accessToken`; request logger saat ini hanya mencatat path tanpa query string.
- Jangan log full WebSocket URL di reverse proxy production.
- `WEBSOCKET_ALLOWED_ORIGINS=*` dipakai untuk development. Untuk production, ganti menjadi whitelist domain frontend.
- Jika Redis mati, API tetap menerima WebSocket connection. Event dari worker tidak bisa fanout lintas instance sampai Redis pulih.
- Graceful shutdown menghentikan Redis subscriber, menutup hub, dan menutup client connection yang tercatat di hub.
