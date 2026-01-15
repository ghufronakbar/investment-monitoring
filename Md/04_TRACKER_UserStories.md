# Module: TRACKER (Ticketing System)

## Epics

Penyelesaian hambatan lapangan (Issue Resolution) lintas instansi. Modul ini menggunakan pendekatan **Event-Sourced Status**, di mana status tiket saat ini ditentukan oleh log terakhir (_Latest Log Entry_).

---

## Ticket Routes

### 1. Create Ticket

**Endpoint:** `POST /tracker/project/{projectId}/ticket`

**Description:** Kontraktor melaporkan hambatan lapangan dengan bukti foto.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus dari CONTRACTOR organization yang terdaftar di project

**URL Parameters:**

- `projectId`: UUID of the project

**Request Body:**

```json
{
  "title": "string (required, ticket title)",
  "description": "string (required, detailed description)",
  "location": "string (required, location of the issue)",
  "evidenceImageUrls": ["url1", "url2"],
  "taskItemId": "uuid | null (optional, link to WBS task)"
}
```

> Note: Minimal 1 evidence image URL required

**Response:**

```json
{
  "id": "uuid",
  "title": "Land obstruction at KM 5",
  "description": "Land not cleared by owner...",
  "location": "KM 5, Section A",
  "evidenceImageUrls": ["https://cloudinary.com/..."],
  "taskItemId": "uuid | null",
  "taskTitle": "Foundation Work | null",
  "contractId": "uuid",
  "currentStatus": "OPEN",
  "creator": {
    "orgId": "uuid",
    "orgName": "PT Konstruksi ABC"
  },
  "assignedTo": null,
  "logs": [
    {
      "id": "uuid",
      "action": "CREATE",
      "description": "Ticket created",
      "newStatus": "OPEN",
      "orgId": "uuid",
      "orgName": "PT Konstruksi ABC",
      "userId": "uuid",
      "userName": "John Doe",
      "createdAt": "timestamp"
    }
  ],
  "createdAt": "timestamp"
}
```

**Flow:**

1. Validasi user adalah CONTRACTOR
2. Buat record di table `tickets`
3. **Atomic:** Buat log pertama di `ticket_logs` dengan action=CREATE, status=OPEN
4. Return ticket dengan log

**Error Codes:**

- `CONTRACTOR_REQUIRED` - User bukan dari CONTRACTOR org
- `NOT_PROJECT_PARTICIPANT` - User tidak memiliki akses ke project

**Success Code:** `SUCCESS_CREATE_TICKET`

---

### 2. Get Tickets

**Endpoint:** `GET /tracker/project/{projectId}/ticket`

**Description:** Mendapatkan daftar semua tickets untuk project.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:** Any project participant

**URL Parameters:**

- `projectId`: UUID of the project

**Query Parameters:**

- `status`: `open` | `closed` | (empty for all)

**Request Body:** None

**Response:**

```json
[
  {
    "id": "uuid",
    "title": "Land obstruction at KM 5",
    "description": "Land not cleared...",
    "location": "KM 5, Section A",
    "evidenceImageUrls": ["url1"],
    "taskItemId": "uuid | null",
    "taskTitle": "Foundation Work | null",
    "contractId": "uuid",
    "currentStatus": "ASSIGNED",
    "creator": { "orgId": "uuid", "orgName": "PT Konstruksi ABC" },
    "assignedTo": { "orgId": "uuid", "orgName": "BPN Jakarta" },
    "createdAt": "timestamp",
    "lastLogAt": "timestamp"
  }
]
```

**Use Case:**

- `?status=open` - Tickets yang masih aktif (OPEN, ASSIGNED, IN_PROGRESS)
- `?status=closed` - Tickets yang sudah ditutup (CLOSED, RESOLVED)

**Success Code:** `SUCCESS_GET_TICKETS`

---

### 3. Get Ticket Detail

**Endpoint:** `GET /tracker/ticket/{ticketId}`

**Description:** Mendapatkan detail ticket beserta semua logs (history).

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:** Any project participant

**URL Parameters:**

- `ticketId`: UUID of the ticket

**Response:**

```json
{
  "id": "uuid",
  "title": "Land obstruction at KM 5",
  "description": "Land not cleared by owner...",
  "location": "KM 5, Section A",
  "evidenceImageUrls": ["url1", "url2"],
  "taskItemId": "uuid | null",
  "taskTitle": "Foundation Work | null",
  "contractId": "uuid",
  "currentStatus": "IN_PROGRESS",
  "creator": { "orgId": "uuid", "orgName": "PT Konstruksi ABC" },
  "assignedTo": { "orgId": "uuid", "orgName": "BPN Jakarta" },
  "logs": [
    {
      "id": "uuid",
      "action": "CREATE",
      "description": "Ticket created",
      "attachmentUrl": null,
      "previousStatus": null,
      "newStatus": "OPEN",
      "orgId": "uuid",
      "orgName": "PT Konstruksi ABC",
      "userId": "uuid",
      "userName": "John Doe",
      "createdAt": "timestamp"
    },
    {
      "id": "uuid",
      "action": "ASSIGNED",
      "description": "Assigned to BPN Jakarta",
      "previousStatus": "OPEN",
      "newStatus": "ASSIGNED",
      "orgId": "uuid",
      "orgName": "PT MRT Jakarta",
      "userId": "uuid",
      "userName": "Admin MRT",
      "createdAt": "timestamp"
    },
    {
      "id": "uuid",
      "action": "STATUS_CHANGED",
      "description": "Surveyor dispatched to location",
      "attachmentUrl": "https://cloudinary.com/survey-report.pdf",
      "previousStatus": "ASSIGNED",
      "newStatus": "IN_PROGRESS",
      "orgId": "uuid",
      "orgName": "BPN Jakarta",
      "userId": "uuid",
      "userName": "BPN Staff",
      "createdAt": "timestamp"
    }
  ],
  "createdAt": "timestamp",
  "lastLogAt": "timestamp"
}
```

**Error Codes:**

- `TICKET_NOT_FOUND` - Ticket tidak ditemukan
- `NOT_PROJECT_PARTICIPANT` - User tidak memiliki akses

**Success Code:** `SUCCESS_GET_TICKET`

---

### 4. Assign Ticket

**Endpoint:** `PUT /tracker/ticket/{ticketId}/assign`

**Description:** Menugaskan ticket ke GOV_PARTNER organization untuk ditangani.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus ADMIN dari EXECUTING_AGENCY yang terdaftar di project

**URL Parameters:**

- `ticketId`: UUID of the ticket

**Request Body:**

```json
{
  "assignToOrgId": "uuid (required, GOV_PARTNER organization ID)"
}
```

**Response:** Updated TicketReturn with new log entry

**Flow:**

1. Validasi user adalah ADMIN dari EXECUTING_AGENCY
2. Validasi assignToOrgId adalah GOV_PARTNER yang terdaftar di project
3. Update `tickets.ticket_assign_to_id`
4. Buat log dengan action=ASSIGNED, status=ASSIGNED
5. Return updated ticket

**Error Codes:**

- `EXECUTING_AGENCY_REQUIRED` - User bukan dari EXECUTING_AGENCY
- `ADMIN_REQUIRED` - User bukan ADMIN
- `INVALID_ASSIGNEE` - Target organization bukan GOV_PARTNER

**Success Code:** `SUCCESS_ASSIGN_TICKET`

---

### 5. Add Log (Update Progress)

**Endpoint:** `POST /tracker/ticket/{ticketId}/log`

**Description:** Menambahkan log/comment pada ticket dengan optional status change.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus dari GOV_PARTNER (yang di-assign) ATAU EXECUTING_AGENCY

**URL Parameters:**

- `ticketId`: UUID of the ticket

**Request Body:**

```json
{
  "description": "string (required, log description)",
  "newStatus": "IN_PROGRESS | RESOLVED | null (optional)",
  "attachmentUrl": "string | null (optional, URL to attachment)"
}
```

**Valid Status Values:**

- `IN_PROGRESS` - Sedang dikerjakan
- `RESOLVED` - Selesai menurut GOV_PARTNER (menunggu verifikasi)

**Response:** TicketLogReturn

**Flow:**

1. Validasi user adalah GOV_PARTNER (yang di-assign) atau EXECUTING_AGENCY
2. Jika GOV_PARTNER: validasi adalah org yang di-assign ke ticket
3. Determine action type:
   - Jika newStatus diberikan: action = STATUS_CHANGED
   - Jika tidak: action = COMMENTED
4. Buat log entry dengan previousStatus dari log terakhir
5. Return log entry

**Status Transition Rules:**

- `ASSIGNED` → `IN_PROGRESS` ✅
- `IN_PROGRESS` → `RESOLVED` ✅
- `OPEN` → `IN_PROGRESS` ✅

**Error Codes:**

- `GOV_PARTNER_OR_EXECUTING_AGENCY_REQUIRED` - User bukan dari org yang diperbolehkan
- `NOT_ASSIGNED_TO_YOUR_ORG` - GOV_PARTNER bukan yang di-assign
- `INVALID_STATUS_TRANSITION` - Status transition tidak valid

**Success Code:** `SUCCESS_ADD_LOG`

---

### 6. Close Ticket

**Endpoint:** `PUT /tracker/ticket/{ticketId}/close`

**Description:** Menutup ticket secara definitif setelah issue resolved.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus ADMIN dari EXECUTING_AGENCY yang terdaftar di project

**URL Parameters:**

- `ticketId`: UUID of the ticket

**Request Body:** None

**Response:** Updated TicketReturn with CLOSED status

**Flow:**

1. Validasi user adalah ADMIN dari EXECUTING_AGENCY
2. Buat log dengan action=STATUS_CHANGED, newStatus=CLOSED
3. Ticket tidak akan muncul lagi di "Outstanding Issues"
4. Return updated ticket

**Error Codes:**

- `EXECUTING_AGENCY_REQUIRED` - User bukan dari EXECUTING_AGENCY
- `ADMIN_REQUIRED` - User bukan ADMIN
- `TICKET_ALREADY_CLOSED` - Ticket sudah ditutup

**Success Code:** `SUCCESS_CLOSE_TICKET`

---

## Summary Table

| #   | Endpoint                              | Method | Auth | Authorization                  | Description           |
| --- | ------------------------------------- | ------ | ---- | ------------------------------ | --------------------- |
| 1   | `/tracker/project/{projectId}/ticket` | POST   | ✅   | CONTRACTOR                     | Create ticket         |
| 2   | `/tracker/project/{projectId}/ticket` | GET    | ✅   | Project participant            | List tickets          |
| 3   | `/tracker/ticket/{ticketId}`          | GET    | ✅   | Project participant            | Get ticket detail     |
| 4   | `/tracker/ticket/{ticketId}/assign`   | PUT    | ✅   | EXECUTING_AGENCY ADMIN         | Assign to GOV_PARTNER |
| 5   | `/tracker/ticket/{ticketId}/log`      | POST   | ✅   | GOV_PARTNER / EXECUTING_AGENCY | Add log/comment       |
| 6   | `/tracker/ticket/{ticketId}/close`    | PUT    | ✅   | EXECUTING_AGENCY ADMIN         | Close ticket          |

---

## Ticket Status Flow

```
┌──────────────────────────────────────────────────────────────────────┐
│                    TICKET LIFECYCLE                                   │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  CONTRACTOR creates ticket                                           │
│     ↓                                                                │
│  [OPEN] ─────────────────────────────────────────────────────────┐   │
│     ↓                                                            │   │
│  EXECUTING_AGENCY assigns to GOV_PARTNER                         │   │
│     ↓                                                            │   │
│  [ASSIGNED] ─────────────────────────────────────────────────────┤   │
│     ↓                                                            │   │
│  GOV_PARTNER starts working on issue                             │   │
│     ↓                                                            │   │
│  [IN_PROGRESS] ──────────────────────────────────────────────────┤   │
│     ↓                                                            │   │
│  GOV_PARTNER marks as resolved                                   │   │
│     ↓                                                            │   │
│  [RESOLVED] ─────────────────────────────────────────────────────┤   │
│     ↓                                                            │   │
│  EXECUTING_AGENCY verifies and closes                            │   │
│     ↓                                                            │   │
│  [CLOSED] ✅                                                      │   │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

## Event-Sourcing Note

> **Important:** Status tiket TIDAK disimpan di table `tickets`.
> Status ditentukan dari `ticket_logs` terakhir (`ORDER BY created_at DESC LIMIT 1`).
> Ini memastikan full audit trail dan data integrity.
