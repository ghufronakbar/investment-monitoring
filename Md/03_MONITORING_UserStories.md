# Module: MONITORING (WBS & Progress)

## Epics

Pemantauan progres fisik lapangan secara real-time, transparan, dan terstruktur. Modul ini mengubah klaim subyektif kontraktor menjadi data obyektif yang tervalidasi untuk perhitungan Kurva S.

---

## Task Routes (WBS Management)

### 1. Create Task

**Endpoint:** `POST /monitoring/project/{projectId}/task`

**Description:** Membuat task baru dalam WBS (Work Breakdown Structure).

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus ADMIN dari EXECUTING_AGENCY yang terdaftar di project

**URL Parameters:**

- `projectId`: UUID of the project

**Request Body:**

```json
{
  "title": "string (required)",
  "description": "string (required)",
  "deadline": "timestamp (required, ISO 8601)",
  "weight": 10,
  "parentId": "uuid | null (optional, for sub-tasks)"
}
```

**Response:**

```json
{
  "id": "uuid",
  "title": "string",
  "description": "string",
  "deadline": "timestamp",
  "weight": 10,
  "parentId": "uuid | null",
  "contractId": "uuid",
  "children": null,
  "isLeaf": true,
  "claimedPercentage": 0,
  "actualPercentage": 0,
  "isCompleted": false,
  "createdAt": "timestamp"
}
```

**Flow:**

1. Validasi user adalah ADMIN dari EXECUTING_AGENCY
2. Validasi weight 0-100
3. Jika parentId diberikan, validasi parent exists dan deadline child <= parent
4. Buat task_item dengan contract_id dari project

**Error Codes:**

- `EXECUTING_AGENCY_ADMIN_REQUIRED` - Bukan ADMIN dari EXECUTING_AGENCY
- `PARENT_NOT_FOUND` - Parent task tidak ditemukan
- `INVALID_DEADLINE` - Deadline melebihi parent

**Success Code:** `SUCCESS_CREATE_TASK`

---

### 2. Get Tasks (Tree View)

**Endpoint:** `GET /monitoring/project/{projectId}/task`

**Description:** Mendapatkan semua task dalam bentuk hierarki (tree) beserta status progress.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:** Any project participant

**URL Parameters:**

- `projectId`: UUID of the project

**Request Body:** None

**Response:**

```json
{
  "tasks": [
    {
      "id": "uuid",
      "title": "Phase 1 - Foundation",
      "description": "Foundation work",
      "deadline": "timestamp",
      "weight": 30,
      "parentId": null,
      "isLeaf": false,
      "claimedPercentage": 50,
      "actualPercentage": 45,
      "isCompleted": false,
      "children": [
        {
          "id": "uuid",
          "title": "Excavation",
          "weight": 15,
          "isLeaf": true,
          "claimedPercentage": 100,
          "actualPercentage": 100,
          "isCompleted": true,
          "children": null
        }
      ]
    }
  ],
  "totalWeight": 100
}
```

**Success Code:** `SUCCESS_GET_TASKS`

---

### 3. Get Leaf Tasks

**Endpoint:** `GET /monitoring/project/{projectId}/task/leaf`

**Description:** Mendapatkan hanya leaf tasks (task tanpa children) untuk dropdown submit progress.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:** Any project participant

**URL Parameters:**

- `projectId`: UUID of the project

**Response:**

```json
{
  "tasks": [
    {
      "id": "uuid",
      "title": "Excavation",
      "weight": 15,
      "isLeaf": true,
      "claimedPercentage": 50,
      "actualPercentage": 45
    }
  ],
  "totalWeight": 100
}
```

**Use Case:** Digunakan oleh CONTRACTOR untuk memilih task saat submit progress.

**Success Code:** `SUCCESS_GET_LEAF_TASKS`

---

### 4. Update Task

**Endpoint:** `PUT /monitoring/task/{taskId}`

**Description:** Mengupdate task item.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus ADMIN dari EXECUTING_AGENCY

**URL Parameters:**

- `taskId`: UUID of the task

**Request Body:**

```json
{
  "title": "string (required)",
  "description": "string (required)",
  "deadline": "timestamp (required)",
  "weight": 15,
  "parentId": "uuid | null"
}
```

**Response:** Updated TaskReturn object

**Error Codes:**

- `TASK_NOT_FOUND` - Task tidak ditemukan
- `EXECUTING_AGENCY_ADMIN_REQUIRED` - Bukan ADMIN dari EXECUTING_AGENCY

**Success Code:** `SUCCESS_UPDATE_TASK`

---

### 5. Delete Task

**Endpoint:** `DELETE /monitoring/task/{taskId}`

**Description:** Menghapus task item (soft delete).

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus ADMIN dari EXECUTING_AGENCY

**URL Parameters:**

- `taskId`: UUID of the task

**Request Body:** None

**Response:** `null`

**Error Codes:**

- `TASK_NOT_FOUND` - Task tidak ditemukan
- `TASK_HAS_CHILDREN` - Task memiliki sub-task
- `TASK_HAS_LOGS` - Task sudah memiliki progress logs

**Success Code:** `SUCCESS_DELETE_TASK`

---

## Progress Log Routes

### 6. Submit Progress

**Endpoint:** `POST /monitoring/task/{taskId}/log`

**Description:** Kontraktor melaporkan progress pekerjaan dengan bukti foto.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus dari CONTRACTOR organization yang terdaftar di project

**URL Parameters:**

- `taskId`: UUID of the task (must be leaf task)

**Request Body:**

```json
{
  "progressPercentage": 75,
  "report": "string (required, description of work done)",
  "evidenceImageUrls": ["url1", "url2", "..."]
}
```

> Note: Minimal 1 evidence image URL required

**Response:**

```json
{
  "id": "uuid",
  "report": "Completed excavation work...",
  "progressPercentage": 75,
  "evidenceImageUrls": ["https://cloudinary.com/..."],
  "snapshotTitle": "Excavation",
  "snapshotDescription": "Excavation work for foundation",
  "snapshotWeight": 15,
  "taskItemId": "uuid",
  "vendorId": "uuid",
  "validation": null,
  "createdAt": "timestamp"
}
```

**Flow:**

1. Validasi user adalah CONTRACTOR
2. Validasi task adalah leaf node
3. **Snapshot:** Copy title, description, weight dari task_items
4. Simpan progress log dengan status pending (validation = null)

**Error Codes:**

- `CONTRACTOR_REQUIRED` - User bukan dari CONTRACTOR org
- `TASK_NOT_FOUND` - Task tidak ditemukan
- `NOT_LEAF_TASK` - Task bukan leaf node

**Success Code:** `SUCCESS_SUBMIT_PROGRESS`

---

### 7. Get Progress Logs

**Endpoint:** `GET /monitoring/project/{projectId}/log`

**Description:** Mendapatkan semua progress logs untuk project.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:** Any project participant

**URL Parameters:**

- `projectId`: UUID of the project

**Response:**

```json
[
  {
    "id": "uuid",
    "report": "Completed excavation...",
    "progressPercentage": 75,
    "evidenceImageUrls": ["url1", "url2"],
    "snapshotTitle": "Excavation",
    "snapshotWeight": 15,
    "taskItemId": "uuid",
    "taskTitle": "Excavation",
    "vendorId": "uuid",
    "vendorName": "PT Konstruksi ABC",
    "validation": {
      "id": "uuid",
      "isAccepted": true,
      "claimedPercentage": 75,
      "realProgressPercentage": 70,
      "comment": null,
      "auditorId": "uuid",
      "createdAt": "timestamp"
    },
    "createdAt": "timestamp"
  }
]
```

**Success Code:** `SUCCESS_GET_PROGRESS_LOGS`

---

### 8. Get Pending Validations

**Endpoint:** `GET /monitoring/project/{projectId}/log/pending`

**Description:** Mendapatkan progress logs yang belum divalidasi (untuk auditor queue).

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus dari INDEPENDENT_AUDITOR organization

**URL Parameters:**

- `projectId`: UUID of the project

**Response:** Array of ProgressLogReturn dengan `validation: null`

**Use Case:** Digunakan oleh AUDITOR untuk melihat antrian validasi.

**Error Codes:**

- `AUDITOR_REQUIRED` - User bukan dari INDEPENDENT_AUDITOR org

**Success Code:** `SUCCESS_GET_PENDING_VALIDATIONS`

---

## Validation Routes

### 9. Validate Progress

**Endpoint:** `POST /monitoring/log/{logId}/validate`

**Description:** Auditor melakukan validasi terhadap klaim progress kontraktor.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus dari INDEPENDENT_AUDITOR organization yang terdaftar di project

**URL Parameters:**

- `logId`: UUID of the progress log

**Request Body:**

```json
{
  "realProgressPercentage": 70,
  "comment": "string | null (required if rejecting)"
}
```

**Response:**

```json
{
  "id": "uuid",
  "isAccepted": true,
  "claimedPercentage": 75,
  "realProgressPercentage": 70,
  "comment": null,
  "auditorId": "uuid",
  "createdAt": "timestamp"
}
```

**Flow:**

1. Validasi user adalah INDEPENDENT_AUDITOR
2. Validasi log belum divalidasi (unique constraint)
3. Jika realProgressPercentage > 0: is_accepted = true
4. Jika realProgressPercentage = 0: is_accepted = false, comment required
5. Simpan validation record
6. Data ini akan dihitung dalam S-Curve jika accepted

**Error Codes:**

- `AUDITOR_REQUIRED` - User bukan dari INDEPENDENT_AUDITOR org
- `PROGRESS_LOG_NOT_FOUND` - Log tidak ditemukan
- `ALREADY_VALIDATED` - Log sudah divalidasi
- `COMMENT_REQUIRED` - Comment wajib untuk rejection

**Success Code:** `SUCCESS_VALIDATE_PROGRESS`

---

## S-Curve Routes

### 10. Get S-Curve Data

**Endpoint:** `GET /monitoring/project/{projectId}/scurve`

**Description:** Mendapatkan data untuk kalkulasi S-Curve.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:** Any project participant

**URL Parameters:**

- `projectId`: UUID of the project

**Response:**

```json
{
  "actualProgress": 45.5,
  "plannedProgress": 60.0,
  "totalSnapshotWeight": 100,
  "totalLogs": 15,
  "approvedLogs": 12,
  "rejectedLogs": 3,
  "taskDetails": [
    {
      "taskId": "uuid",
      "taskTitle": "Excavation",
      "plannedWeight": 15,
      "deadline": "timestamp",
      "actualPercentage": 100,
      "claimedPercentage": 100
    }
  ]
}
```

**Calculation:**

- **Planned Progress:** Sum of weights of tasks whose deadline has passed
- **Actual Progress:** Sum of (snapshot_weight × real_progress_percentage / 100) for accepted validations

**Success Code:** `SUCCESS_GET_SCURVE`

---

## Summary Table

| #   | Endpoint                                      | Method | Auth | Authorization          | Description             |
| --- | --------------------------------------------- | ------ | ---- | ---------------------- | ----------------------- |
| 1   | `/monitoring/project/{projectId}/task`        | POST   | ✅   | EXECUTING_AGENCY ADMIN | Create task             |
| 2   | `/monitoring/project/{projectId}/task`        | GET    | ✅   | Project participant    | Get tasks (tree)        |
| 3   | `/monitoring/project/{projectId}/task/leaf`   | GET    | ✅   | Project participant    | Get leaf tasks          |
| 4   | `/monitoring/task/{taskId}`                   | PUT    | ✅   | EXECUTING_AGENCY ADMIN | Update task             |
| 5   | `/monitoring/task/{taskId}`                   | DELETE | ✅   | EXECUTING_AGENCY ADMIN | Delete task             |
| 6   | `/monitoring/task/{taskId}/log`               | POST   | ✅   | CONTRACTOR             | Submit progress         |
| 7   | `/monitoring/project/{projectId}/log`         | GET    | ✅   | Project participant    | Get logs                |
| 8   | `/monitoring/project/{projectId}/log/pending` | GET    | ✅   | INDEPENDENT_AUDITOR    | Get pending validations |
| 9   | `/monitoring/log/{logId}/validate`            | POST   | ✅   | INDEPENDENT_AUDITOR    | Validate progress       |
| 10  | `/monitoring/project/{projectId}/scurve`      | GET    | ✅   | Project participant    | Get S-Curve data        |

---

## Progress Flow

```
┌──────────────────────────────────────────────────────────────────────┐
│                    PROGRESS REPORTING FLOW                            │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  1. EXECUTING_AGENCY creates WBS (task hierarchy)                    │
│     ↓                                                                │
│  2. CONTRACTOR submits progress on leaf tasks with photos            │
│     ↓                                                                │
│  3. AUDITOR reviews pending validations                              │
│     ↓                                                                │
│  4a. APPROVE: Set real_progress_percentage → Data masuk S-Curve      │
│  4b. REJECT: Set comment → Data tidak dihitung                       │
│     ↓                                                                │
│  5. S-Curve calculates actual vs planned progress                    │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```
