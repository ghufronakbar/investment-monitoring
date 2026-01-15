# Module: PROJECT (Core Data & Contracting)

## Epics

Inisialisasi proyek dan penunjukan langsung (Direct Assignment) para pemangku kepentingan.

---

## Project Routes

### 1. Create Project

**Endpoint:** `POST /project`

**Description:** Membuat proyek baru berdasarkan G-to-G agreement.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus ADMIN dari organisasi dengan role `GOV_SUPERVISOR`

**Request Body:**

```json
{
  "name": "string (required, project name)",
  "description": "string (required, project description)",
  "contractFileUrl": "string (required, URL to Loan Agreement PDF)",
  "loanProviderId": "uuid (required, organization ID of loan provider e.g. JICA)",
  "imageUrls": ["string (required, min 1, array of image URLs)"]
}
```

**Response:**

```json
{
  "project": {
    "id": "uuid",
    "name": "string",
    "description": "string",
    "contractFileUrl": "string",
    "imageUrls": ["string"],
    "createdAt": "timestamp"
  },
  "contract": {
    "id": "uuid",
    "description": "string",
    "contractValue": null,
    "contractDocUrl": null,
    "govSupervisor": {
      "orgId": "uuid",
      "orgName": "string",
      "orgRole": "GOV_SUPERVISOR",
      "logoUrl": "string | null"
    },
    "loanProvider": {
      "orgId": "uuid",
      "orgName": "string",
      "orgRole": "LOAN_PROVIDER",
      "logoUrl": "string | null"
    },
    "executingAgency": null,
    "vendor": null,
    "auditor": null,
    "govPartners": []
  }
}
```

**Flow:**

1. Validasi user adalah ADMIN dari GOV_SUPERVISOR organization
2. Buat record di table `projects`
3. Buat record di table `contracts` dengan `gov_supervisor_participant_id` dan `loan_provider_participant_id`
4. Return project dan contract data

**Error Codes:**

- `GOV_SUPERVISOR_ADMIN_REQUIRED` - User bukan ADMIN dari GOV_SUPERVISOR
- `LOAN_PROVIDER_NOT_FOUND` - Loan provider organization tidak ditemukan

**Success Code:** `SUCCESS_CREATE_PROJECT`

---

### 2. List Projects

**Endpoint:** `GET /project`

**Description:** Mendapatkan daftar semua proyek yang accessible oleh user.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:** Any authenticated user (returns only accessible projects)

**Request Body:** None

**Response:**

```json
[
  {
    "id": "uuid",
    "name": "string",
    "description": "string",
    "contractFileUrl": "string",
    "imageUrls": ["string"],
    "createdAt": "timestamp"
  }
]
```

**Flow:**

1. Get user dari JWT token
2. Query semua project yang accessible berdasarkan:
   - User adalah member dari organization yang terlibat dalam contract
   - GOV_SUPERVISOR dapat melihat semua proyek
3. Return list projects

**Success Code:** `SUCCESS_LIST_PROJECTS`

---

### 3. Get Project Detail

**Endpoint:** `GET /project/{projectId}`

**Description:** Mendapatkan detail lengkap proyek beserta contract dan settings.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- User harus menjadi participant di proyek ATAU
- User adalah member dari GOV_SUPERVISOR organization

**URL Parameters:**

- `projectId`: UUID of the project

**Request Body:** None

**Response:**

```json
{
  "project": {
    "id": "uuid",
    "name": "string",
    "description": "string",
    "contractFileUrl": "string",
    "imageUrls": ["string"],
    "createdAt": "timestamp"
  },
  "contract": {
    "id": "uuid",
    "description": "string",
    "contractValue": 1000000000,
    "contractDocUrl": "string | null",
    "govSupervisor": {
      "orgId": "uuid",
      "orgName": "Bappenas",
      "orgRole": "GOV_SUPERVISOR",
      "logoUrl": "string | null"
    },
    "loanProvider": {
      "orgId": "uuid",
      "orgName": "JICA",
      "orgRole": "LOAN_PROVIDER",
      "logoUrl": "string | null"
    },
    "executingAgency": {
      "orgId": "uuid",
      "orgName": "PT MRT Jakarta",
      "orgRole": "EXECUTING_AGENCY",
      "logoUrl": "string | null"
    },
    "vendor": {
      "orgId": "uuid",
      "orgName": "PT Konstruksi ABC",
      "orgRole": "CONTRACTOR",
      "logoUrl": "string | null"
    },
    "vendorSelectedAt": "timestamp | null",
    "auditor": {
      "orgId": "uuid",
      "orgName": "KAP XYZ",
      "orgRole": "INDEPENDENT_AUDITOR",
      "logoUrl": "string | null"
    },
    "auditorSelectedAt": "timestamp | null",
    "govPartners": [
      {
        "orgId": "uuid",
        "orgName": "BPN",
        "orgRole": "GOV_PARTNER",
        "logoUrl": "string | null"
      }
    ]
  },
  "settings": {
    "id": "uuid",
    "startDate": "timestamp",
    "baselineDeadline": "timestamp",
    "totalBudget": 1000000000,
    "currency": "IDR"
  }
}
```

**Error Codes:**

- `NO_PROJECT_ACCESS` - User tidak memiliki akses ke proyek
- `PROJECT_NOT_FOUND` - Proyek tidak ditemukan

**Success Code:** `SUCCESS_GET_PROJECT`

---

### 4. Configure Project Settings

**Endpoint:** `PUT /project/{projectId}/settings`

**Description:** Mengkonfigurasi settings proyek (budget, timeline, currency).

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus ADMIN dari EXECUTING_AGENCY yang terdaftar di contract proyek

**URL Parameters:**

- `projectId`: UUID of the project

**Request Body:**

```json
{
  "startDate": "timestamp (required, ISO 8601 format)",
  "baselineDeadline": "timestamp (required, ISO 8601 format)",
  "totalBudget": 1000000000,
  "currency": "IDR (required)"
}
```

**Response:**

```json
{
  "success": true,
  "message": "Settings configured successfully"
}
```

**Flow:**

1. Validasi user adalah ADMIN dari EXECUTING_AGENCY
2. Validasi baselineDeadline > startDate
3. Upsert ke table `project_settings`
4. Data ini akan digunakan untuk kalkulasi S-Curve

**Error Codes:**

- `NO_PROJECT_ACCESS` - User tidak memiliki akses
- `EXECUTING_AGENCY_ADMIN_REQUIRED` - User bukan ADMIN dari EXECUTING_AGENCY
- `INVALID_DEADLINE` - Baseline deadline sebelum start date

**Success Code:** `SUCCESS_CONFIGURE_SETTINGS`

---

### 5. Assign Executing Agency

**Endpoint:** `PUT /project/{projectId}/contract/assign-executing-agency`

**Description:** Menunjuk Executing Agency (misal PT MRT) untuk mengelola proyek.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus ADMIN dari GOV_SUPERVISOR yang terdaftar di contract proyek

**URL Parameters:**

- `projectId`: UUID of the project

**Request Body:**

```json
{
  "executingAgencyId": "uuid (required, organization ID with role EXECUTING_AGENCY)"
}
```

**Response:**

```json
{
  "success": true,
  "message": "Executing agency assigned successfully"
}
```

**Flow:**

1. Validasi user adalah ADMIN dari GOV_SUPERVISOR
2. Validasi executingAgencyId adalah organization dengan role EXECUTING_AGENCY
3. Update `contracts.executing_agency_participant_id`
4. Executing Agency sekarang dapat manage project

**Error Codes:**

- `GOV_SUPERVISOR_ADMIN_REQUIRED` - User bukan ADMIN dari GOV_SUPERVISOR
- `ORGANIZATION_NOT_FOUND` - Organization tidak ditemukan
- `INVALID_ORGANIZATION_ROLE` - Organization bukan EXECUTING_AGENCY

**Success Code:** `SUCCESS_ASSIGN_EXECUTING_AGENCY`

---

### 6. Assign Vendor (Contractor)

**Endpoint:** `PUT /project/{projectId}/contract/assign-vendor`

**Description:** Menunjuk Kontraktor pemenang tender dan nilai kontrak.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus ADMIN dari EXECUTING_AGENCY yang terdaftar di contract proyek

**URL Parameters:**

- `projectId`: UUID of the project

**Request Body:**

```json
{
  "vendorOrgId": "uuid (required, organization ID with role CONTRACTOR)",
  "contractValue": 1000000000,
  "contractDocUrl": "string (required, URL to contract document PDF)"
}
```

**Response:**

```json
{
  "success": true,
  "message": "Vendor assigned successfully"
}
```

**Flow:**

1. Validasi user adalah ADMIN dari EXECUTING_AGENCY
2. Validasi vendorOrgId adalah organization dengan role CONTRACTOR
3. Update `contracts.vendor_participant_id`, `contract_value`, `contract_doc_url`, `vendor_selected_at`
4. Kontraktor sekarang dapat melihat dan submit progress

**Error Codes:**

- `EXECUTING_AGENCY_ADMIN_REQUIRED` - User bukan ADMIN dari EXECUTING_AGENCY
- `ORGANIZATION_NOT_FOUND` - Organization tidak ditemukan
- `INVALID_ORGANIZATION_ROLE` - Organization bukan CONTRACTOR

**Success Code:** `SUCCESS_ASSIGN_VENDOR`

---

### 7. Assign Auditor

**Endpoint:** `PUT /project/{projectId}/contract/assign-auditor`

**Description:** Menunjuk Independent Auditor untuk validasi progress.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus ADMIN dari EXECUTING_AGENCY yang terdaftar di contract proyek

**URL Parameters:**

- `projectId`: UUID of the project

**Request Body:**

```json
{
  "auditorOrgId": "uuid (required, organization ID with role INDEPENDENT_AUDITOR)"
}
```

**Response:**

```json
{
  "success": true,
  "message": "Auditor assigned successfully"
}
```

**Flow:**

1. Validasi user adalah ADMIN dari EXECUTING_AGENCY
2. Validasi auditorOrgId adalah organization dengan role INDEPENDENT_AUDITOR
3. Update `contracts.auditor_participant_id`, `auditor_selected_at`
4. Auditor sekarang dapat melihat dan validasi progress

**Error Codes:**

- `EXECUTING_AGENCY_ADMIN_REQUIRED` - User bukan ADMIN dari EXECUTING_AGENCY
- `ORGANIZATION_NOT_FOUND` - Organization tidak ditemukan
- `INVALID_ORGANIZATION_ROLE` - Organization bukan INDEPENDENT_AUDITOR

**Success Code:** `SUCCESS_ASSIGN_AUDITOR`

---

### 8. Assign Gov Partners

**Endpoint:** `PUT /project/{projectId}/contract/assign-gov-partners`

**Description:** Menunjuk Government Partners (BPN, PLN, dll) untuk issue resolution.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus ADMIN dari EXECUTING_AGENCY yang terdaftar di contract proyek

**URL Parameters:**

- `projectId`: UUID of the project

**Request Body:**

```json
{
  "govPartnerIds": ["uuid", "uuid", "..."]
}
```

> Note: Array of organization IDs with role GOV_PARTNER, minimum 1 item

**Response:**

```json
{
  "success": true,
  "message": "Gov partners assigned successfully"
}
```

**Flow:**

1. Validasi user adalah ADMIN dari EXECUTING_AGENCY
2. Validasi semua govPartnerIds adalah organization dengan role GOV_PARTNER
3. Delete existing gov_participants untuk contract ini
4. Insert new records ke table `gov_participants`
5. Gov Partners sekarang dapat melihat proyek dan handle tickets

**Error Codes:**

- `EXECUTING_AGENCY_ADMIN_REQUIRED` - User bukan ADMIN dari EXECUTING_AGENCY
- `ORGANIZATION_NOT_FOUND` - Salah satu organization tidak ditemukan
- `INVALID_ORGANIZATION_ROLE` - Salah satu organization bukan GOV_PARTNER

**Success Code:** `SUCCESS_ASSIGN_GOV_PARTNERS`

---

## Summary Table

| #   | Endpoint                                                | Method | Auth | Authorization                        | Description              |
| --- | ------------------------------------------------------- | ------ | ---- | ------------------------------------ | ------------------------ |
| 1   | `/project`                                              | POST   | ✅   | GOV_SUPERVISOR ADMIN                 | Create new project       |
| 2   | `/project`                                              | GET    | ✅   | Any user                             | List accessible projects |
| 3   | `/project/{projectId}`                                  | GET    | ✅   | Project participant / GOV_SUPERVISOR | Get project detail       |
| 4   | `/project/{projectId}/settings`                         | PUT    | ✅   | EXECUTING_AGENCY ADMIN               | Configure settings       |
| 5   | `/project/{projectId}/contract/assign-executing-agency` | PUT    | ✅   | GOV_SUPERVISOR ADMIN                 | Assign executing agency  |
| 6   | `/project/{projectId}/contract/assign-vendor`           | PUT    | ✅   | EXECUTING_AGENCY ADMIN               | Assign contractor        |
| 7   | `/project/{projectId}/contract/assign-auditor`          | PUT    | ✅   | EXECUTING_AGENCY ADMIN               | Assign auditor           |
| 8   | `/project/{projectId}/contract/assign-gov-partners`     | PUT    | ✅   | EXECUTING_AGENCY ADMIN               | Assign gov partners      |

---

## Contract Assignment Flow

```
┌──────────────────────────────────────────────────────────────────────┐
│                    PROJECT SETUP FLOW                                 │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  1. GOV_SUPERVISOR creates project                                   │
│     ↓                                                                │
│  2. GOV_SUPERVISOR assigns EXECUTING_AGENCY                          │
│     ↓                                                                │
│  3. EXECUTING_AGENCY configures settings (budget, timeline)          │
│     ↓                                                                │
│  4. EXECUTING_AGENCY assigns VENDOR (Contractor)                     │
│     ↓                                                                │
│  5. EXECUTING_AGENCY assigns AUDITOR                                 │
│     ↓                                                                │
│  6. EXECUTING_AGENCY assigns GOV_PARTNERS                            │
│     ↓                                                                │
│  ✅ Project ready for monitoring                                     │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```
