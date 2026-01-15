# JICA Investment Monitoring System - API Documentation

## Project Overview

Sistem Pemantauan Investasi JICA adalah platform digital untuk monitoring proyek infrastruktur yang dibiayai oleh Japan International Cooperation Agency (JICA). Sistem ini memungkinkan transparansi dan akuntabilitas penuh dalam pelaksanaan proyek strategis negara.

### Key Features

- **Invite-Only Access**: Keamanan berbasis undangan untuk menjaga integritas data
- **Multi-Stakeholder**: Melibatkan berbagai pihak (Bappenas, JICA, PT MRT, Kontraktor, Auditor, BPN/PLN)
- **Real-time Monitoring**: Pemantauan progress fisik dan finansial secara real-time
- **Tiered Approval**: Persetujuan berjenjang untuk pencairan dana
- **S-Curve Analytics**: Visualisasi perbandingan rencana vs realisasi

---

## Base Response Format

Semua API endpoint mengembalikan response dalam format yang **konsisten** untuk memudahkan handling di frontend.

### Success Response (HTTP 200)

```json
{
  "metaData": {
    "code": 200,
    "message": "OK",
    "requestId": "abc123-def456",
    "path": "/project/123",
    "method": "GET"
  },
  "responseMessage": "SUCCESS_GET_PROJECT",
  "data": { ... },
  "validationErrors": null,
  "filterQuery": null,
  "pagination": null
}
```

### Validation Error Response (HTTP 400)

```json
{
  "metaData": {
    "code": 400,
    "message": "Bad Request",
    "requestId": "abc123-def456",
    "path": "/project",
    "method": "POST"
  },
  "responseMessage": "VALIDATION_FAILED",
  "data": null,
  "validationErrors": {
    "name": "Name is required",
    "amount": "Amount must be greater than 0"
  },
  "filterQuery": null,
  "pagination": null
}
```

### Error Response (HTTP 4xx/5xx)

```json
{
  "metaData": {
    "code": 403,
    "message": "Forbidden",
    "requestId": "abc123-def456",
    "path": "/project/123/settings",
    "method": "PUT"
  },
  "responseMessage": "EXECUTING_AGENCY_ADMIN_REQUIRED",
  "data": null,
  "validationErrors": null,
  "filterQuery": null,
  "pagination": null
}
```

### Response Fields

| Field                | Type         | Description                                  |
| -------------------- | ------------ | -------------------------------------------- |
| `metaData.code`      | int          | HTTP status code                             |
| `metaData.message`   | string       | HTTP status text (OK, Bad Request, etc)      |
| `metaData.requestId` | string       | Unique request ID for tracing/debugging      |
| `metaData.path`      | string       | Request path                                 |
| `metaData.method`    | string       | HTTP method (GET, POST, PUT, DELETE)         |
| `responseMessage`    | string       | Application-specific message code            |
| `data`               | object/array | Response data (null on error)                |
| `validationErrors`   | object       | Field-level validation errors (null if none) |
| `filterQuery`        | object       | Filter query params for list endpoints       |
| `pagination`         | object       | Pagination info for list endpoints           |

---

## HTTP Status Codes

| Code  | Factory Function         | Usage                                          |
| ----- | ------------------------ | ---------------------------------------------- |
| `200` | OK()                     | Success response                               |
| `400` | NewBadRequest()          | Invalid input, business rule violation         |
| `400` | NewValidationFailed()    | Struct validation failed (with field errors)   |
| `401` | NewUnauthorized()        | Missing or invalid JWT token                   |
| `403` | NewForbidden()           | Valid token but insufficient permissions       |
| `404` | NewNotFound()            | Resource not found                             |
| `500` | NewInternalServerError() | Server-side error (logged, hidden from client) |

---

## Authentication

### JWT Bearer Token

Semua protected endpoint memerlukan JWT token di header:

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Token Types

| Type          | Lifetime          | Usage                |
| ------------- | ----------------- | -------------------- |
| Access Token  | Short (15-30 min) | API requests         |
| Refresh Token | Long (7-30 days)  | Get new access token |

### Token Refresh Flow

```
1. Access token expired → API returns 401
2. Frontend calls POST /iam/auth/refresh-token with refresh token
3. API returns new access token + rotated refresh token
4. Frontend updates stored tokens and retries original request
```

---

## Organization Roles

Setiap organisasi memiliki **satu role** yang menentukan permission-nya di sistem:

| Role                  | Description           | Can Create Org | Example           |
| --------------------- | --------------------- | -------------- | ----------------- |
| `GOV_SUPERVISOR`      | Government supervisor | ✅             | Bappenas          |
| `EXECUTING_AGENCY`    | Project executor      | ✅             | PT MRT Jakarta    |
| `CONTRACTOR`          | Construction company  | -              | PT Konstruksi ABC |
| `INDEPENDENT_AUDITOR` | Third-party auditor   | -              | KAP XYZ           |
| `LOAN_PROVIDER`       | Funding institution   | -              | JICA              |
| `GOV_PARTNER`         | Government partner    | -              | BPN, PLN          |

## Member Roles

Setiap user dalam organisasi memiliki **member role**:

| Role     | Description                       |
| -------- | --------------------------------- |
| `ADMIN`  | Full access, can invite members   |
| `MEMBER` | Read/write access sesuai org role |

---

## Complete User Flow

### Phase 1: System Initialization

```
┌─────────────────────────────────────────────────────────────────┐
│  1. SYSTEM SEEDING                                               │
│     - GOV_SUPERVISOR org (Bappenas) created via migration       │
│     - LOAN_PROVIDER org (JICA) created via migration            │
│     - Initial admin users created                               │
└─────────────────────────────────────────────────────────────────┘
```

### Phase 2: Organization Setup

```
┌─────────────────────────────────────────────────────────────────┐
│  2. ORGANIZATION REGISTRATION (by GOV_SUPERVISOR/EXECUTING_AGENCY)│
│                                                                 │
│  a. GOV_SUPERVISOR creates EXECUTING_AGENCY                     │
│     POST /iam/organization/{orgId}                              │
│     ↓                                                           │
│  b. EXECUTING_AGENCY creates: CONTRACTOR, AUDITOR, GOV_PARTNER  │
│     POST /iam/organization/{orgId}                              │
│     ↓                                                           │
│  c. Organization admins receive email invitation                │
│     ↓                                                           │
│  d. Admins set password via POST /iam/auth/set-password         │
│     ↓                                                           │
│  e. Admins invite team members                                  │
│     POST /iam/organization/{orgId}/member/invite                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Phase 3: Project Creation

```
┌─────────────────────────────────────────────────────────────────┐
│  3. PROJECT INITIALIZATION (by GOV_SUPERVISOR)                   │
│                                                                 │
│  a. Create project with loan agreement                          │
│     POST /project                                               │
│     - Upload loan agreement PDF                                 │
│     - Select LOAN_PROVIDER (JICA)                               │
│     ↓                                                           │
│  b. Assign EXECUTING_AGENCY to project                          │
│     PUT /project/{id}/contract/assign-executing-agency          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Phase 4: Project Configuration

```
┌─────────────────────────────────────────────────────────────────┐
│  4. PROJECT SETUP (by EXECUTING_AGENCY)                          │
│                                                                 │
│  a. Configure project settings                                  │
│     PUT /project/{id}/settings                                  │
│     - Set start date, baseline deadline                         │
│     - Set total budget, currency                                │
│     ↓                                                           │
│  b. Assign CONTRACTOR (tender winner)                           │
│     PUT /project/{id}/contract/assign-vendor                    │
│     - Upload contract document PDF                              │
│     - Set contract value                                        │
│     ↓                                                           │
│  c. Assign INDEPENDENT_AUDITOR                                  │
│     PUT /project/{id}/contract/assign-auditor                   │
│     ↓                                                           │
│  d. Assign GOV_PARTNERS (BPN, PLN, etc)                         │
│     PUT /project/{id}/contract/assign-gov-partners              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Phase 5: Work Breakdown Structure

```
┌─────────────────────────────────────────────────────────────────┐
│  5. WBS CREATION (by EXECUTING_AGENCY)                           │
│                                                                 │
│  a. Create hierarchical task structure                          │
│     POST /monitoring/project/{id}/task                          │
│     - Tasks with weights (total = 100%)                         │
│     - Deadlines (child <= parent)                               │
│     - Parent-child relationships                                │
│     ↓                                                           │
│  Example WBS:                                                   │
│  ├── Phase 1 - Foundation (30%)                                 │
│  │   ├── Excavation (15%)           ← Leaf task               │
│  │   └── Concrete Pouring (15%)     ← Leaf task               │
│  ├── Phase 2 - Structure (40%)                                  │
│  │   ├── Steel Installation (20%)   ← Leaf task               │
│  │   └── Floor Construction (20%)   ← Leaf task               │
│  └── Phase 3 - Finishing (30%)                                  │
│      ├── Electrical (15%)           ← Leaf task               │
│      └── Plumbing (15%)             ← Leaf task               │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Phase 6: Progress Reporting Cycle

```
┌─────────────────────────────────────────────────────────────────┐
│  6. PROGRESS MONITORING (Recurring Cycle)                        │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ CONTRACTOR submits progress                              │   │
│  │ POST /monitoring/task/{id}/log                           │   │
│  │ - Select leaf task                                       │   │
│  │ - Input progress percentage                              │   │
│  │ - Upload evidence photos (mandatory)                     │   │
│  │ - System snapshots task weight for audit trail           │   │
│  └────────────────────────┬─────────────────────────────────┘   │
│                           ↓                                     │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ AUDITOR validates progress                               │   │
│  │ GET /monitoring/project/{id}/log/pending                 │   │
│  │ POST /monitoring/log/{id}/validate                       │   │
│  │ - Review photos and description                          │   │
│  │ - APPROVE: Set real_progress_percentage → S-Curve        │   │
│  │ - REJECT: Add comment → Back to contractor               │   │
│  └────────────────────────┬─────────────────────────────────┘   │
│                           ↓                                     │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ S-CURVE UPDATES                                          │   │
│  │ GET /support/dashboard/s-curve/{projectId}               │   │
│  │ - Plan vs Actual visualization                           │   │
│  │ - Deviation calculation                                  │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Phase 7: Issue Resolution

```
┌─────────────────────────────────────────────────────────────────┐
│  7. TICKET MANAGEMENT (When Issues Arise)                        │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ CONTRACTOR reports obstacle                              │   │
│  │ POST /tracker/project/{id}/ticket                        │   │
│  │ - Title, description, location                           │   │
│  │ - Evidence photos (mandatory)                            │   │
│  │ - Optional: Link to blocked WBS task                     │   │
│  │ Status: OPEN                                             │   │
│  └────────────────────────┬─────────────────────────────────┘   │
│                           ↓                                     │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ EXECUTING_AGENCY assigns to GOV_PARTNER                  │   │
│  │ PUT /tracker/ticket/{id}/assign                          │   │
│  │ - Select responsible agency (BPN/PLN/etc)                │   │
│  │ Status: ASSIGNED                                         │   │
│  └────────────────────────┬─────────────────────────────────┘   │
│                           ↓                                     │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ GOV_PARTNER resolves issue                               │   │
│  │ POST /tracker/ticket/{id}/log                            │   │
│  │ - Add progress updates                                   │   │
│  │ - Upload supporting documents                            │   │
│  │ - Change status: IN_PROGRESS → RESOLVED                  │   │
│  └────────────────────────┬─────────────────────────────────┘   │
│                           ↓                                     │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ EXECUTING_AGENCY closes ticket                           │   │
│  │ PUT /tracker/ticket/{id}/close                           │   │
│  │ - Verify resolution                                      │   │
│  │ Status: CLOSED                                           │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Phase 8: Payment Processing

```
┌─────────────────────────────────────────────────────────────────┐
│  8. PAYMENT DISBURSEMENT (Tiered Approval)                       │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ CONTRACTOR requests payment                              │   │
│  │ POST /finance/project/{id}/request                       │   │
│  │ - Title (e.g., "Termin 1 - 10%")                         │   │
│  │ - Amount (validated against available budget)            │   │
│  │ - Upload invoice PDF                                     │   │
│  │ Status: SUBMITTED                                        │   │
│  └────────────────────────┬─────────────────────────────────┘   │
│                           ↓                                     │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ AUDITOR reviews (Technical Verification)                 │   │
│  │ PUT /finance/request/{id}/approve-auditor                │   │
│  │ - Verify physical progress matches payment request       │   │
│  │ - APPROVE → Status: AUDITOR_APPROVED                     │   │
│  │ - REJECT → Status: AUDITOR_REJECTED                      │   │
│  │ Digital signature recorded                               │   │
│  └────────────────────────┬─────────────────────────────────┘   │
│                           ↓                                     │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ EXECUTING_AGENCY reviews (Administrative Verification)  │   │
│  │ PUT /finance/request/{id}/approve-agency                 │   │
│  │ - Verify document completeness                           │   │
│  │ - APPROVE → Status: AGENCY_APPROVED                      │   │
│  │ - REJECT → Status: AGENCY_REJECTED                       │   │
│  │ Digital signature recorded                               │   │
│  └────────────────────────┬─────────────────────────────────┘   │
│                           ↓                                     │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ LOAN_PROVIDER disburses (Payment Execution)             │   │
│  │ PUT /finance/request/{id}/disburse                       │   │
│  │ - Upload payment proof (bank transfer receipt)           │   │
│  │ - Status: PAID                                           │   │
│  │ Digital signature recorded                               │   │
│  │ → Amount added to dashboard "Disbursed"                  │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Phase 9: Dashboard & Reporting

```
┌─────────────────────────────────────────────────────────────────┐
│  9. DASHBOARD & ANALYTICS (All Stakeholders)                     │
│                                                                 │
│  GET /support/dashboard/stats/{projectId}                       │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Summary Statistics:                                      │   │
│  │ - Total Budget: Rp 1,000,000,000,000                     │   │
│  │ - Disbursed: Rp 250,000,000,000 (25%)                    │   │
│  │ - Physical Progress: 45.5%                               │   │
│  │ - Tickets: 5 Open, 3 In Progress, 30 Closed              │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                 │
│  GET /support/dashboard/s-curve/{projectId}                     │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ S-Curve Visualization:                                   │   │
│  │                                                          │   │
│  │  100% ─┤                                    ****         │   │
│  │        │                              *****/             │   │
│  │   75% ─┤                        *****/ (Plan)            │   │
│  │        │                  *****/                         │   │
│  │   50% ─┤           *****/ ────── (Actual)                │   │
│  │        │      ****/                                      │   │
│  │   25% ─┤ ****/                                           │   │
│  │        │/                                                │   │
│  │    0% ─┼─────┬────┬────┬────┬────┬────┬────┬────→ Time   │   │
│  │        Jan  Feb  Mar  Apr  May  Jun  Jul  Aug            │   │
│  │                                                          │   │
│  │ Deviation: -5.5% (Behind Schedule)                       │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## API Modules Summary

| Module         | Prefix        | Endpoints | Purpose                       |
| -------------- | ------------- | --------- | ----------------------------- |
| **IAM**        | `/iam`        | 10        | Authentication & Organization |
| **Project**    | `/project`    | 8         | Project & Contract Management |
| **Monitoring** | `/monitoring` | 10        | WBS & Progress Tracking       |
| **Tracker**    | `/tracker`    | 6         | Issue/Ticket Management       |
| **Finance**    | `/finance`    | 7         | Payment Processing            |
| **Support**    | `/support`    | 4         | Upload & Dashboard            |

**Total: 45 Endpoints**

---

## Frontend Implementation Notes

### 1. Response Handling

```typescript
interface BaseResponse<T> {
  metaData: {
    code: number;
    message: string;
    requestId: string;
    path: string;
    method: string;
  };
  responseMessage: string;
  data: T;
  validationErrors: Record<string, string> | null;
}

// Usage
const response = await api.get<Project>("/project/123");
if (response.metaData.code === 200) {
  // Handle success
  const project = response.data;
} else if (response.validationErrors) {
  // Handle validation errors per field
  Object.entries(response.validationErrors).forEach(([field, error]) => {
    setFieldError(field, error);
  });
} else {
  // Handle error by responseMessage
  showError(response.responseMessage);
}
```

### 2. Token Refresh

```typescript
// Intercept 401 responses
if (response.status === 401 && !isRefreshing) {
  const newTokens = await refreshToken();
  if (newTokens) {
    // Retry original request
    return axios(originalRequest);
  } else {
    // Redirect to login
    router.push("/login");
  }
}
```

### 3. Role-Based UI

```typescript
// Show/hide UI based on organization role
const canCreateTask = userRole === "EXECUTING_AGENCY" && memberRole === "ADMIN";
const canSubmitProgress = userRole === "CONTRACTOR";
const canValidateProgress = userRole === "INDEPENDENT_AUDITOR";
```

### 4. File Upload

```typescript
// Use FormData for file uploads
const formData = new FormData();
formData.append("file", file);
const response = await api.post("/support/upload/single", formData, {
  headers: { "Content-Type": "multipart/form-data" },
});
const imageUrl = response.data.url;
```

---

## Environment Variables

| Variable                   | Description            | Example                             |
| -------------------------- | ---------------------- | ----------------------------------- |
| `DATABASE_URL`             | PostgreSQL connection  | `postgres://user:pass@host:5432/db` |
| `JWT_SECRET`               | JWT signing key        | `your-secret-key`                   |
| `JWT_EXPIRES_IN`           | Access token lifetime  | `15m`                               |
| `REFRESH_TOKEN_EXPIRES_IN` | Refresh token lifetime | `7d`                                |
| `CLOUDINARY_CLOUD_NAME`    | Cloudinary account     | `your-cloud`                        |
| `CLOUDINARY_API_KEY`       | Cloudinary key         | `123456`                            |
| `CLOUDINARY_API_SECRET`    | Cloudinary secret      | `abcdef`                            |
