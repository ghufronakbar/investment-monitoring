# Module: FINANCE (Disbursement)

## Epics

Pencairan dana termin yang akuntabel dengan persetujuan berjenjang (Tiered Approval). Flow ini melibatkan validasi teknis (Auditor), administratif (Agency), dan eksekusi pembayaran (Loan Provider).

---

### Story 1: Request Payment (Termin)

**As a** CONTRACTOR (Admin),
**I want to** submit a payment request for a specific milestone,
**So that** the payment process can begin.

#### Acceptance Criteria

1.  **Validation:**
    - Pastikan User adalah `CONTRACTOR` yang terdaftar pada Contract tersebut (`contracts.vendor_participant_id`).
    - **[IMPLEMENTED]** Input `amount` tidak boleh melebihi `contract_value`.
    - **[IMPLEMENTED]** Input `amount` tidak boleh melebihi sisa budget yang belum dibayarkan (`contract_value - paid - pending`).
    - **[IMPLEMENTED]** Tidak boleh mengajukan jika ada request yang masih pending (belum PAID atau REJECTED).
2.  **Input Data:**
    - User mengisi Judul Termin (e.g., "Termin 1 - 10%").
    - User mengisi Nominal (`amount`).
    - **Mandatory:** Upload file Invoice (`invoice_file_url`).
3.  **Initial Status:**
    - Sistem menyimpan data dengan status `SUBMITTED`.
    - **[NOTE]** Status `DRAFT` dihilangkan, default langsung `SUBMITTED`.

#### Technical Notes & Schema Mapping

- **Endpoint:** `POST /finance/project/{projectId}/request`
- **Logic:** Lookup `contract_id` via `projectId`.
- **Validations:**
  - `AMOUNT_EXCEEDS_CONTRACT_VALUE` - amount > contract_value
  - `AMOUNT_EXCEEDS_AVAILABLE` - amount > available amount
  - `PENDING_REQUEST_EXISTS` - ada request yang masih pending
  - `CONTRACT_VALUE_NOT_SET` - contract_value belum diisi
- **Table `payment_requests` Mapping:**
  - `title`: Input User
  - `amount`: Input User (BigInt)
  - `status`: `'SUBMITTED'` (Enum)
  - `invoice_file_url`: Cloudinary URL
  - `contract_id`: From Lookup
  - `auditor_...`, `executing_agency_...`, `loan_provider_...`: NULL

---

### Story 2: Technical Approval (Auditor)

**As an** INDEPENDENT_AUDITOR,
**I want to** approve or reject the payment request based on physical progress,
**So that** payment is only made for verified work.

#### Acceptance Criteria

1.  **Permission Check:** Hanya user dari Organisasi `INDEPENDENT_AUDITOR` yang terdaftar di kontrak (`auditor_participant_id`) yang boleh akses.
2.  **Action: APPROVE:**
    - Mengubah status menjadi `AUDITOR_APPROVED`.
    - **Digital Signature:** Sistem WAJIB mencatat ID Auditor, ID Organisasi, dan Timestamp saat tombol ditekan.
3.  **Action: REJECT:**
    - Mengubah status menjadi `AUDITOR_REJECTED`.
    - Alur berhenti (dikembalikan ke kontraktor).

#### Technical Notes & Schema Mapping

- **Endpoint:** `PUT /finance/request/{requestId}/approve-auditor`
- **Body:** `{ "action": "APPROVE" | "REJECT" }`
- **Table `payment_requests` Update:**
  - `status`: `'AUDITOR_APPROVED'` OR `'AUDITOR_REJECTED'`
  - `auditor_org_id`: Get from JWT (Org ID)
  - `auditor_user_id`: Get from JWT (User ID)
  - `auditor_at`: `NOW()`

---

### Story 3: Administrative Approval (Agency)

**As an** EXECUTING_AGENCY (Finance Team),
**I want to** verify the document completeness,
**So that** the request can be forwarded to the Lender.

#### Acceptance Criteria

1.  **Pre-condition:** Request harus berstatus `AUDITOR_APPROVED`. Jika tidak, tolak aksi.
2.  **Permission Check:** Hanya user dari Organisasi `EXECUTING_AGENCY` yang terdaftar di kontrak (`executing_agency_participant_id`).
3.  **Action: APPROVE:**
    - Mengubah status menjadi `AGENCY_APPROVED`.
    - Mencatat Digital Signature Agency.
4.  **Action: REJECT:**
    - Mengubah status menjadi `AGENCY_REJECTED`.

#### Technical Notes & Schema Mapping

- **Endpoint:** `PUT /finance/request/{requestId}/approve-agency`
- **Body:** `{ "action": "APPROVE" | "REJECT" }`
- **Table `payment_requests` Update:**
  - `status`: `'AGENCY_APPROVED'` OR `'AGENCY_REJECTED'`
  - `executing_agency_org_id`: Get from JWT
  - `executing_agency_user_id`: Get from JWT
  - `executing_agency_at`: `NOW()`

---

### Story 4: Disbursement (JICA)

**As a** LOAN_PROVIDER (JICA Admin),
**I want to** mark the request as paid after transferring funds,
**So that** the system reflects the actual cash flow.

#### Acceptance Criteria

1.  **Pre-condition:** Request harus berstatus `AGENCY_APPROVED`.
2.  **Action: DISBURSE (PAY):**
    - User melakukan transfer bank secara offline.
    - **[IMPLEMENTED]** User WAJIB upload bukti pembayaran (`paymentProofImageUrls`, min 1 URL).
    - Di sistem, user klik "Mark as Paid".
    - Status berubah menjadi `PAID`.
    - Mencatat Digital Signature Loan Provider.
3.  **Post-Action:** Nilai ini akan menambah "Financial Progress" di Dashboard.

#### Technical Notes & Schema Mapping

- **Endpoint:** `PUT /finance/request/{requestId}/disburse`
- **Body:** `{ "paymentProofImageUrls": ["url1", "url2", ...] }`
- **Table `payment_requests` Update:**
  - `status`: `'PAID'`
  - `payment_proof_image_urls`: Array of URLs (TEXT[])
  - `loan_provider_org_id`: Get from JWT
  - `loan_provider_user_id`: Get from JWT
  - `loan_provider_at`: `NOW()`

---

### Story 5: Payment Summary (NEW)

**As a** Project Participant,
**I want to** see the payment summary and available amount for new requests,
**So that** I know how much can be requested.

#### Acceptance Criteria

1.  **Access:** Any project participant can view.
2.  **Response Data:**
    - `contractValue`: Total nilai kontrak
    - `paidAmount`: Jumlah yang sudah PAID
    - `pendingAmount`: Jumlah yang masih dalam proses (tidak PAID, tidak REJECTED)
    - `availableAmount`: contractValue - paidAmount - pendingAmount
    - `canSubmit`: Boolean, true jika tidak ada pending request dan availableAmount > 0
    - `currency`: Currency dari project settings

#### Technical Notes & Schema Mapping

- **Endpoint:** `GET /finance/project/{projectId}/summary`
- **Response:**

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

---

## Schema Notes

### Status Enum

```sql
CREATE TYPE payment_request_status_enum AS ENUM (
    'SUBMITTED',           -- Initial (DRAFT removed)
    'AUDITOR_APPROVED',
    'AUDITOR_REJECTED',
    'AGENCY_APPROVED',
    'AGENCY_REJECTED',
    'PAID'
);
```

### Payment Proof (Migration)

```sql
-- Migration: 20260115173102_add_payment_proof_image_urls.sql
ALTER TABLE payment_requests
ADD COLUMN payment_proof_image_urls TEXT[] NOT NULL DEFAULT '{}';
```

### Response Format

- `amount` dikembalikan sebagai **String** untuk preserve BigInt precision.
- `paymentProofImageUrls` selalu dikembalikan sebagai **array** (bukan null).
