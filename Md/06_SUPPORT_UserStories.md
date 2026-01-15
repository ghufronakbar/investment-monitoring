# Module: SUPPORT (Utility & Dashboard)

## Epics

Fungsi pendukung visualisasi data (Dashboard) dan manajemen file (Utility). Modul ini tidak menyimpan data transaksional sendiri, melainkan mengagregasi data dari modul lain (Project, Monitoring, Finance) untuk kebutuhan reporting.

---

## File Upload Routes

### 1. Upload Single File

**Endpoint:** `POST /support/upload/single`

**Description:** Upload single file (image atau PDF) ke Cloudinary.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:** Any authenticated user

**Content-Type:** `multipart/form-data`

**Request Body:**

```
Form Field: file
```

**Allowed MIME Types:**

- `image/jpeg` (JPG/JPEG)
- `image/png` (PNG)
- `application/pdf` (PDF)

**Constraints:**

- Max file size: **5MB**

**Response:**

```json
{
  "publicId": "jica-investment_abc123",
  "url": "https://res.cloudinary.com/xxx/image/upload/v1234/jica-investment_abc123.jpg",
  "format": "jpg"
}
```

**Flow:**

1. Parse multipart form
2. Validasi file exists
3. Validasi file size <= 5MB
4. Validasi MIME type (JPG, PNG, PDF only)
5. Determine resource type (image atau pdf)
6. Upload ke Cloudinary
7. Return public URL

**Error Codes:**

- `FILE_REQUIRED` - Tidak ada file yang diupload
- `FILE_TOO_LARGE` - File > 5MB
- `FILE_TOO_LARGE_5MB_MAX` - File > 5MB (after size check)
- `INVALID_FILE_TYPE` - MIME type tidak diperbolehkan

**Success Code:** `SUCCESS_UPLOAD`

---

### 2. Upload Multiple Files

**Endpoint:** `POST /support/upload/multiple`

**Description:** Upload multiple files (images atau PDFs) ke Cloudinary dalam satu request.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:** Any authenticated user

**Content-Type:** `multipart/form-data`

**Request Body:**

```
Form Field: files (multiple)
```

**Constraints:**

- Max **10 files** per request
- Max **5MB per file**
- Total max: 50MB

**Response:**

```json
[
  { "publicId": "abc123", "url": "https://...", "format": "jpg" },
  { "publicId": "def456", "url": "https://...", "format": "pdf" },
  { "publicId": "ghi789", "url": "https://...", "format": "png" }
]
```

**Error Codes:**

- `FILES_REQUIRED` - Tidak ada file yang diupload
- `MAX_10_FILES_ALLOWED` - Lebih dari 10 files
- `TOTAL_SIZE_TOO_LARGE` - Total size > 50MB
- `FILE_TOO_LARGE_5MB_MAX` - Salah satu file > 5MB
- `INVALID_FILE_TYPE` - Salah satu file MIME type tidak valid

**Success Code:** `SUCCESS_UPLOAD_MULTIPLE`

---

## Dashboard Routes

### 3. Get S-Curve Data

**Endpoint:** `GET /support/dashboard/s-curve/{projectId}`

**Description:** Mendapatkan data S-Curve (Plan vs Actual progress) per minggu untuk visualisasi grafik.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:** Any project participant

**URL Parameters:**

- `projectId`: UUID of the project

**Request Body:** None

**Response:**

```json
{
  "projectName": "MRT Jakarta Phase 2",
  "startDate": "2024-01-01T00:00:00Z",
  "baselineDeadline": "2024-12-31T00:00:00Z",
  "dataPoints": [
    { "date": "2024-01-01", "plan": 0.0, "actual": 0.0 },
    { "date": "2024-01-08", "plan": 5.0, "actual": 2.5 },
    { "date": "2024-01-15", "plan": 10.0, "actual": 8.0 },
    { "date": "2024-01-22", "plan": 15.0, "actual": 12.0 },
    { "date": "2024-12-31", "plan": 100.0, "actual": 85.0 }
  ]
}
```

**Data Point Fields:**

- `date`: Tanggal dalam format YYYY-MM-DD
- `plan`: Cumulative planned progress (%) pada tanggal tersebut
- `actual`: Cumulative actual progress (%) pada tanggal tersebut

**Calculation Logic:**

**Plan Line:**

```
For each date:
  Plan = Sum(weight of leaf tasks where deadline <= date) / TotalWeight * 100
```

**Actual Line:**

```
For each date:
  Actual = Sum(snapshot_weight × real_progress_percentage / 100
           for accepted validations where validation_date <= date) / TotalWeight * 100
```

**Data Source:**

- Date range: `project_settings.start_date` → `project_settings.baseline_deadline`
- Interval: Weekly (every 7 days)
- Tasks: Only **leaf tasks** (no children)
- Progress: Only **accepted validations** (`progress_validations.is_accepted = TRUE`)

**Error Codes:**

- `PROJECT_SETTINGS_NOT_FOUND` - Project settings belum di-configure
- `NOT_PROJECT_PARTICIPANT` - User tidak memiliki akses

**Success Code:** `SUCCESS_GET_SCURVE`

---

### 4. Get Dashboard Statistics

**Endpoint:** `GET /support/dashboard/stats/{projectId}`

**Description:** Mendapatkan statistik high-level project untuk dashboard overview.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:** Any project participant

**URL Parameters:**

- `projectId`: UUID of the project

**Request Body:** None

**Response:**

```json
{
  "projectName": "MRT Jakarta Phase 2",
  "totalBudget": "1000000000000",
  "disbursedAmount": "250000000000",
  "currency": "IDR",
  "physicalProgress": 45.5,
  "ticketStats": {
    "open": 5,
    "inProgress": 3,
    "resolved": 10,
    "closed": 20,
    "total": 38
  }
}
```

**Field Explanations:**

| Field                    | Description                         | Source                                                   |
| ------------------------ | ----------------------------------- | -------------------------------------------------------- |
| `projectName`            | Nama project                        | `projects.name`                                          |
| `totalBudget`            | Total budget (String for BigInt)    | `project_settings.total_budget`                          |
| `disbursedAmount`        | Total sudah dibayarkan              | Sum of `payment_requests.amount` where `status = 'PAID'` |
| `currency`               | Currency code                       | `project_settings.currency`                              |
| `physicalProgress`       | Progress fisik saat ini (%)         | Calculated from S-Curve actual progress                  |
| `ticketStats.open`       | Tickets dengan status OPEN/ASSIGNED | Latest `ticket_logs.new_status`                          |
| `ticketStats.inProgress` | Tickets dengan status IN_PROGRESS   | Latest `ticket_logs.new_status`                          |
| `ticketStats.resolved`   | Tickets dengan status RESOLVED      | Latest `ticket_logs.new_status`                          |
| `ticketStats.closed`     | Tickets dengan status CLOSED        | Latest `ticket_logs.new_status`                          |
| `ticketStats.total`      | Total semua tickets                 | Count of all tickets                                     |

**Error Codes:**

- `PROJECT_SETTINGS_NOT_FOUND` - Project settings belum di-configure
- `NOT_PROJECT_PARTICIPANT` - User tidak memiliki akses

**Success Code:** `SUCCESS_GET_STATS`

---

## Summary Table

| #   | Endpoint                                 | Method | Auth | Authorization       | Description           |
| --- | ---------------------------------------- | ------ | ---- | ------------------- | --------------------- |
| 1   | `/support/upload/single`                 | POST   | ✅   | Any user            | Upload single file    |
| 2   | `/support/upload/multiple`               | POST   | ✅   | Any user            | Upload multiple files |
| 3   | `/support/dashboard/s-curve/{projectId}` | GET    | ✅   | Project participant | Get S-Curve data      |
| 4   | `/support/dashboard/stats/{projectId}`   | GET    | ✅   | Project participant | Get dashboard stats   |

---

## Technical Notes

### Cloudinary Integration

File uploads menggunakan Cloudinary SDK dengan konfigurasi:

- Folder: `{APP_NAME}` dari environment
- Tags: `source:api`, `type:image` atau `type:document`
- Resource type: `image` untuk gambar, `raw` untuk PDF

### No Database Tables

Support module **tidak memiliki table database sendiri**:

- **Upload:** Stateless, hanya berinteraksi dengan Cloudinary
- **Dashboard:** Mengagregasi data dari:
  - `project_settings`
  - `task_items`
  - `progress_logs`
  - `progress_validations`
  - `payment_requests`
  - `tickets`
  - `ticket_logs`

### Performance Considerations

S-Curve calculation dapat menjadi berat untuk project dengan banyak tasks dan logs. Pertimbangkan caching jika diperlukan.
