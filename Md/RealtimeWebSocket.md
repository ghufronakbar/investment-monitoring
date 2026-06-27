# Realtime WebSocket

Dokumentasi ini mencatat kontrak WebSocket yang sudah tersedia pada Tahap 1 implementasi.

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

## Batasan Tahap 1

- Topic authorization belum dibuat.
- Redis Pub/Sub fanout belum dibuat.
- Outbox event belum dikirim ke WebSocket.
- Tidak ada replay untuk client yang offline.
- Presence belum dibuat.

