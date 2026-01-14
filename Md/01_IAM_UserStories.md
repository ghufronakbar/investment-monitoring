# Module: IAM (Identity & Access Management)

## Epics

Manajemen identitas yang ketat berbasis undangan (Invite-Only) untuk menjamin keamanan data proyek strategis negara.

---

### Story 1: Login & Account Activation

**As a** User (All Roles),
**I want to** login to the system using my email,
**So that** I can access the dashboard securely.

#### Acceptance Criteria

1.  **Check Email First:** Sistem harus mengecek email terlebih dahulu.
    - Jika email tidak ada di database: Tolak login (Error: "Access Denied. Invitation Required").
    - Jika email ada tapi password belum diset (`password = null`): Arahkan flow ke "Check Email for Invitation Link".
    - Jika email ada dan password ada: Tampilkan input password.
2.  **Verify Invitation:** User yang mengklik link dari email harus divalidasi token-nya. Jika valid, user diminta input `New Password` dan `Confirm Password`.
3.  **Security:** Token JWT (Access & Refresh) harus digenerate setelah login sukses.

#### Technical Notes

- Endpoint: `/iam/auth/check-email`, `/iam/auth/login`, `/iam/auth/invitation/verify`.
- Table: `users`.

---

### Story 2: Create Organization & Invite First Admin

**As a** GOV_SUPERVISOR (Bappenas) OR EXECUTING_AGENCY (MRT),
**I want to** create a new partner organization (e.g., Contractor Company) and invite their PIC,
**So that** they can start using the system.

#### Acceptance Criteria

1.  **Single Action:** Input nama organisasi, tipe (e.g., CONTRACTOR), dan email PIC (Admin) dalam satu form.
2.  **Auto-Invite:** Setelah organisasi dibuat, sistem otomatis membuat user baru untuk email PIC tersebut dengan role `ADMIN` dan mengirim email undangan.
3.  **Data Integrity:** Role organisasi harus sesuai Enum (`CONTRACTOR`, `GOV_PARTNER`, etc).

#### Technical Notes

- Endpoint: `POST /iam/organization`.
- Table: `organizations`, `users`, `organization_members`.

---

### Story 3: Internal Team Management

**As an** Organization Admin (e.g., Head of BPN),
**I want to** invite my staff members,
**So that** I can delegate tasks (e.g., land measurement) to them.

#### Acceptance Criteria

1.  Admin bisa input email dan nama staf.
2.  Staf yang di-invite akan menerima email aktivasi.
3.  Staf otomatis masuk ke dalam `organization_members` dengan role `MEMBER`.
4.  Staf tidak bisa mengubah setting organisasi, hanya Admin yang bisa.

#### Technical Notes

- Endpoint: `POST /iam/organization/{id}/member/invite`.
