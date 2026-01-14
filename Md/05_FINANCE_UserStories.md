# Module: FINANCE (Disbursement)

## Epics

Pencairan dana termin yang akuntabel dengan persetujuan berjenjang.

---

### Story 1: Request Payment (Termin)

**As a** CONTRACTOR (Admin),
**I want to** submit a payment request for a milestone,
**So that** we can get paid.

#### Acceptance Criteria

1.  Input: Judul Termin (e.g., "Termin 1 - 10%"), Nominal Uang.
2.  **Mandatory Upload:** Invoice / Berita Acara Fisik (PDF).
3.  Status awal: `SUBMITTED`.

#### Technical Notes

- Endpoint: `POST /finance/project/{id}/request`.
- Table: `payment_requests`.

---

### Story 2: Technical Approval (Auditor)

**As an** INDEPENDENT_AUDITOR,
**I want to** verify if the physical progress matches the payment request,
**So that** we don't pay for work that isn't done.

#### Acceptance Criteria

1.  Auditor mengecek apakah % Realisasi di modul Monitoring >= % Tagihan.
2.  Approve: Status berubah jadi `AUDITOR_APPROVED`.
3.  Reject: Status berubah jadi `AUDITOR_REJECTED` (Kembali ke kontraktor).

#### Technical Notes

- Endpoint: `PUT .../approve-auditor`.

---

### Story 3: Administrative Approval (Agency)

**As an** EXECUTING_AGENCY (Finance Team),
**I want to** verify the documents completeness,
**So that** the request can be forwarded to the lender.

#### Acceptance Criteria

1.  Hanya bisa dieksekusi jika status sudah `AUDITOR_APPROVED`.
2.  Approve: Status berubah jadi `AGENCY_APPROVED`.

#### Technical Notes

- Endpoint: `PUT .../approve-agency`.

---

### Story 4: Disbursement (JICA)

**As a** LOAN_PROVIDER (JICA Admin),
**I want to** mark the request as paid after I transfer the funds,
**So that** the system reflects the actual cash flow.

#### Acceptance Criteria

1.  User melakukan transfer bank di luar sistem.
2.  User mengklik tombol "Mark as Paid".
3.  Status berubah menjadi `PAID`.
4.  Timestamp pembayaran tercatat.

#### Technical Notes

- Endpoint: `PUT .../disburse`.
