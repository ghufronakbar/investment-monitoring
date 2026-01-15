# Module: FINANCE (Disbursement)

## Epics

Pencairan dana termin yang akuntabel dengan persetujuan berjenjang (Tiered Approval). Flow ini melibatkan validasi teknis (Auditor), administratif (Agency), dan eksekusi pembayaran (Loan Provider).

---

## Payment Request Routes

### 1. Create Payment Request

**Endpoint:** `POST /finance/project/{projectId}/request`

**Description:** Kontraktor mengajukan permintaan pembayaran termin.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus dari CONTRACTOR organization yang terdaftar di project

**URL Parameters:**

- `projectId`: UUID of the project

**Request Body:**

```json
{
  "title": "string (required, e.g. 'Termin 1 - 10%')",
  "amount": 100000000,
  "invoiceFileUrl": "string (required, URL to invoice PDF)"
}
```

**Response:**

```json
{
  "id": "uuid",
  "title": "Termin 1 - 10%",
  "amount": "100000000",
  "status": "SUBMITTED",
  "invoiceFileUrl": "https://cloudinary.com/invoice.pdf",
  "paymentProofImageUrls": [],
  "contractId": "uuid",
  "auditor": null,
  "agency": null,
  "loanProvider": null,
  "createdAt": "timestamp",
  "updatedAt": "timestamp"
}
```

**Flow:**

1. Validasi user adalah CONTRACTOR
2. **Validasi Amount:**
   - Amount tidak boleh > contract_value
   - Amount tidak boleh > available amount (contract_value - paid - pending)
3. Validasi tidak ada pending request
4. Buat payment_request dengan status SUBMITTED
5. Return created request

**Error Codes:**

- `CONTRACTOR_REQUIRED` - User bukan dari CONTRACTOR org
- `CONTRACT_VALUE_NOT_SET` - Contract value belum diisi
- `AMOUNT_EXCEEDS_CONTRACT_VALUE` - Amount > contract value total
- `AMOUNT_EXCEEDS_AVAILABLE` - Amount > available amount
- `PENDING_REQUEST_EXISTS` - Ada request yang belum selesai

**Success Code:** `SUCCESS_CREATE_PAYMENT_REQUEST`

---

### 2. Get Payment Requests

**Endpoint:** `GET /finance/project/{projectId}/request`

**Description:** Mendapatkan daftar semua payment requests untuk project.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:** Any project participant

**URL Parameters:**

- `projectId`: UUID of the project

**Response:**

```json
[
  {
    "id": "uuid",
    "title": "Termin 1 - 10%",
    "amount": "100000000",
    "status": "PAID",
    "invoiceFileUrl": "url",
    "paymentProofImageUrls": ["url1", "url2"],
    "contractId": "uuid",
    "auditor": {
      "orgId": "uuid",
      "orgName": "KAP XYZ",
      "userId": "uuid",
      "approvedAt": "timestamp"
    },
    "agency": {
      "orgId": "uuid",
      "orgName": "PT MRT Jakarta",
      "userId": "uuid",
      "approvedAt": "timestamp"
    },
    "loanProvider": {
      "orgId": "uuid",
      "orgName": "JICA",
      "userId": "uuid",
      "approvedAt": "timestamp"
    },
    "createdAt": "timestamp"
  }
]
```

**Success Code:** `SUCCESS_GET_PAYMENT_REQUESTS`

---

### 3. Get Payment Request Detail

**Endpoint:** `GET /finance/request/{requestId}`

**Description:** Mendapatkan detail payment request.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:** Any project participant

**URL Parameters:**

- `requestId`: UUID of the payment request

**Response:** Single PaymentRequestReturn object

**Error Codes:**

- `PAYMENT_REQUEST_NOT_FOUND` - Request tidak ditemukan

**Success Code:** `SUCCESS_GET_PAYMENT_REQUEST`

---

### 4. Approve by Auditor

**Endpoint:** `PUT /finance/request/{requestId}/approve-auditor`

**Description:** Auditor melakukan verifikasi teknis dan approve/reject request.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus dari INDEPENDENT_AUDITOR organization yang terdaftar di project

**URL Parameters:**

- `requestId`: UUID of the payment request

**Request Body:**

```json
{
  "action": "APPROVE | REJECT"
}
```

**Response:** Updated PaymentRequestReturn

**Flow:**

1. Validasi user adalah INDEPENDENT_AUDITOR
2. Validasi status saat ini adalah SUBMITTED
3. Jika APPROVE: status → AUDITOR_APPROVED
4. Jika REJECT: status → AUDITOR_REJECTED
5. Record digital signature (org_id, user_id, timestamp)

**Valid Status Transitions:**

- `SUBMITTED` → `AUDITOR_APPROVED` ✅
- `SUBMITTED` → `AUDITOR_REJECTED` ✅

**Error Codes:**

- `AUDITOR_REQUIRED` - User bukan dari INDEPENDENT_AUDITOR org
- `INVALID_STATUS_TRANSITION` - Status bukan SUBMITTED

**Success Code:** `SUCCESS_APPROVE_BY_AUDITOR`

---

### 5. Approve by Agency

**Endpoint:** `PUT /finance/request/{requestId}/approve-agency`

**Description:** Executing Agency melakukan verifikasi administratif.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus dari EXECUTING_AGENCY organization yang terdaftar di project

**URL Parameters:**

- `requestId`: UUID of the payment request

**Request Body:**

```json
{
  "action": "APPROVE | REJECT"
}
```

**Response:** Updated PaymentRequestReturn

**Flow:**

1. Validasi user adalah EXECUTING_AGENCY
2. Validasi status saat ini adalah AUDITOR_APPROVED
3. Jika APPROVE: status → AGENCY_APPROVED
4. Jika REJECT: status → AGENCY_REJECTED
5. Record digital signature

**Valid Status Transitions:**

- `AUDITOR_APPROVED` → `AGENCY_APPROVED` ✅
- `AUDITOR_APPROVED` → `AGENCY_REJECTED` ✅

**Error Codes:**

- `EXECUTING_AGENCY_REQUIRED` - User bukan dari EXECUTING_AGENCY org
- `INVALID_STATUS_TRANSITION` - Status bukan AUDITOR_APPROVED

**Success Code:** `SUCCESS_APPROVE_BY_AGENCY`

---

### 6. Disburse (Mark as Paid)

**Endpoint:** `PUT /finance/request/{requestId}/disburse`

**Description:** Loan Provider mencairkan dana dan upload bukti pembayaran.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Required:** User harus dari LOAN_PROVIDER organization yang terdaftar di project

**URL Parameters:**

- `requestId`: UUID of the payment request

**Request Body:**

```json
{
  "paymentProofImageUrls": ["url1", "url2", "..."]
}
```

> Note: Minimal 1 payment proof image URL required

**Response:** Updated PaymentRequestReturn with status PAID

**Flow:**

1. Validasi user adalah LOAN_PROVIDER
2. Validasi status saat ini adalah AGENCY_APPROVED
3. Validasi paymentProofImageUrls minimal 1 item
4. Update status → PAID
5. Simpan payment proof URLs
6. Record digital signature
7. Amount akan dihitung ke "Disbursed" di dashboard

**Valid Status Transitions:**

- `AGENCY_APPROVED` → `PAID` ✅

**Error Codes:**

- `LOAN_PROVIDER_REQUIRED` - User bukan dari LOAN_PROVIDER org
- `INVALID_STATUS_TRANSITION` - Status bukan AGENCY_APPROVED
- `PAYMENT_PROOF_REQUIRED` - Tidak ada bukti pembayaran

**Success Code:** `SUCCESS_DISBURSE`

---

### 7. Get Payment Summary

**Endpoint:** `GET /finance/project/{projectId}/summary`

**Description:** Mendapatkan ringkasan pembayaran dan sisa budget yang bisa diajukan.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:** Any project participant

**URL Parameters:**

- `projectId`: UUID of the project

**Response:**

```json
{
  "contractValue": "1000000000",
  "paidAmount": "200000000",
  "pendingAmount": "100000000",
  "availableAmount": "700000000",
  "canSubmit": false,
  "currency": "IDR"
}
```

**Field Explanations:**

- `contractValue`: Total nilai kontrak dari `contracts.contract_value`
- `paidAmount`: Sum of amount dari requests dengan status PAID
- `pendingAmount`: Sum of amount dari requests yang sedang dalam proses (tidak PAID, tidak REJECTED)
- `availableAmount`: contractValue - paidAmount - pendingAmount
- `canSubmit`: true jika tidak ada pending request DAN availableAmount > 0
- `currency`: Dari project settings

**Success Code:** `SUCCESS_GET_PAYMENT_SUMMARY`

---

## Summary Table

| #   | Endpoint                                       | Method | Auth | Authorization       | Description             |
| --- | ---------------------------------------------- | ------ | ---- | ------------------- | ----------------------- |
| 1   | `/finance/project/{projectId}/request`         | POST   | ✅   | CONTRACTOR          | Create payment request  |
| 2   | `/finance/project/{projectId}/request`         | GET    | ✅   | Project participant | List payment requests   |
| 3   | `/finance/request/{requestId}`                 | GET    | ✅   | Project participant | Get request detail      |
| 4   | `/finance/request/{requestId}/approve-auditor` | PUT    | ✅   | INDEPENDENT_AUDITOR | Technical approval      |
| 5   | `/finance/request/{requestId}/approve-agency`  | PUT    | ✅   | EXECUTING_AGENCY    | Administrative approval |
| 6   | `/finance/request/{requestId}/disburse`        | PUT    | ✅   | LOAN_PROVIDER       | Mark as paid            |
| 7   | `/finance/project/{projectId}/summary`         | GET    | ✅   | Project participant | Payment summary         |

---

## Payment Request Status Flow

```
┌──────────────────────────────────────────────────────────────────────┐
│                    PAYMENT REQUEST LIFECYCLE                          │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  CONTRACTOR submits payment request                                  │
│     ↓                                                                │
│  [SUBMITTED] ────────────────────────────────────────────────────┐   │
│     ↓                                                            │   │
│  AUDITOR reviews (technical verification)                        │   │
│     ↓                                                            │   │
│  [AUDITOR_APPROVED] ───────┬───────→ [AUDITOR_REJECTED] ❌       │   │
│     ↓                      │                                     │   │
│  AGENCY reviews (administrative)                                 │   │
│     ↓                      │                                     │   │
│  [AGENCY_APPROVED] ────────┼───────→ [AGENCY_REJECTED] ❌        │   │
│     ↓                      │                                     │   │
│  LOAN_PROVIDER disburses   │                                     │   │
│     ↓                      │                                     │   │
│  [PAID] ✅                  │                                     │   │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Schema Notes

### Status Enum

```sql
CREATE TYPE payment_request_status_enum AS ENUM (
    'SUBMITTED',           -- Initial state
    'AUDITOR_APPROVED',    -- After auditor approval
    'AUDITOR_REJECTED',    -- After auditor rejection
    'AGENCY_APPROVED',     -- After agency approval
    'AGENCY_REJECTED',     -- After agency rejection
    'PAID'                 -- Final: disbursed
);
```

### Digital Signature Fields

Setiap approval step mencatat:

- `{role}_org_id`: Organization ID yang melakukan
- `{role}_user_id`: User ID yang melakukan
- `{role}_at`: Timestamp saat dilakukan

### Response Format Notes

- `amount` dikembalikan sebagai **String** untuk preserve BigInt precision
- `paymentProofImageUrls` selalu dikembalikan sebagai **array** (tidak pernah null)
