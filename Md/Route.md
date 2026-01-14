# Route Documentation

Schema Route HTTP untuk Sistem Monitoring MRT Jakarta Fase 2.

## [MODULE IAM] (Identity & Access Management)

### [IAM-Auth: Login & Invitation Flow]

- **POST** `/iam/auth/check-email`
- **Auth:** Public
- **Body:** `{ email }`
- **Desc:** Langkah pertama login. Mengecek apakah email terdaftar dan apakah sudah setup password.
- **Response:**
- `exists: false` -> Tolak (Sistem Invite Only).
- `exists: true, isSetup: true` -> Frontend minta input Password.
- `exists: true, isSetup: false` -> Frontend minta user cek email untuk aktivasi akun.

- **POST** `/iam/auth/login`
- **Auth:** Public
- **Body:** `{ email, password }`
- **Desc:** Langkah kedua. Menukar kredensial dengan Access Token & Refresh Token.

- **POST** `/iam/auth/refresh`
- **Auth:** Public (Header: Refresh Token)
- **Desc:** Mendapatkan Access Token baru (Silent refresh).

- **POST** `/iam/auth/logout`
- **Auth:** Bearer
- **Desc:** Revoke refresh token saat ini (Logout).

- **GET** `/iam/auth/invitation/verify`
- **Auth:** Public (Query Param: `?token=xyz`)
- **Desc:** Dipanggil saat user mengklik link di email undangan. Backend memvalidasi token.

- **POST** `/iam/auth/invitation/accept`
- **Auth:** Public
- **Body:** `{ token, newPassword, confirmPassword }`
- **Desc:** User melakukan setup password untuk pertama kalinya. Status user berubah menjadi `ACTIVE`.

### [IAM-Organization]

- **GET** `/iam/organization`
- **Auth:** Bearer
- **Desc:** Mendapatkan list organisasi di mana user tersebut bergabung.

- **POST** `/iam/organization`
- **Auth:** Bearer
- **OrgRole:** `GOV_SUPERVISOR` (Bappenas) atau `EXECUTING_AGENCY` (MRT)
- **Body:** `{ name, type, adminEmail, adminName }`
- **Desc:** Membuat organisasi baru (misal: PT Shimizu / BPN Jakpus) SEKALIGUS mengundang admin pertamanya.
- _Logic:_ Create Org -> Create User (Admin) -> Send Invitation Email.

- **POST** `/iam/organization/{organizationId}/member/invite`
- **Auth:** Bearer
- **MemberRole:** `ADMIN` (Level Organisasi)
- **Body:** `{ email, name, role }`
- **Desc:** Admin organisasi mengundang staff internal mereka sendiri.

---

## [MODULE PROJECT] (Core Data & Contract)

### [PROJECT-Project]

- **GET** `/project`
- **Auth:** Bearer
- **Desc:** Mendapatkan list project.
- _Logic Visibility:_
- `GOV_SUPERVISOR` & `LOAN_PROVIDER`: Melihat **SEMUA** project.
- `EXECUTING_AGENCY`, `CONTRACTOR`, `INDEPENDENT_AUDITOR`, `GOV_PARTNER`: Hanya melihat project di mana organisasi mereka terlibat dalam kontrak.

- **POST** `/project`
- **Auth:** Bearer
- **OrgRole:** `GOV_SUPERVISOR` (Bappenas)
- **MemberRole:** `ADMIN`
- **Desc:** Inisialisasi Project baru (Input: Nama, Deskripsi, Upload Loan Agreement).

- **PUT** `/project/{projectId}/settings`
- **Auth:** Bearer
- **OrgRole:** `EXECUTING_AGENCY` (PT MRT)
- **Desc:** Mengatur parameter teknis: `baselineDeadline`, `totalBudget`, `startDate`, `currency`.

### [PROJECT-Contract Assignment] (Pengganti Bidding)

- **GET** `/project/{projectId}/contract`
- **Auth:** Bearer
- **Desc:** Melihat detail kontrak: Siapa Kontraktornya, Auditornya, dan Partner Pemerintahnya.

- **PUT** `/project/{projectId}/contract/assign-vendor`
- **Auth:** Bearer
- **OrgRole:** `EXECUTING_AGENCY`
- **Body:** `{ vendorOrgId, contractValue, contractDocUrl }`
- **Desc:** Menunjuk pemenang tender (Direct Assignment) yang sudah dilakukan secara offline.

- **PUT** `/project/{projectId}/contract/assign-auditor`
- **Auth:** Bearer
- **OrgRole:** `EXECUTING_AGENCY`
- **Body:** `{ auditorOrgId }`
- **Desc:** Menunjuk konsultan pengawas independen.

- **PUT** `/project/{projectId}/contract/assign-gov-partners`
- **Auth:** Bearer
- **OrgRole:** `EXECUTING_AGENCY`
- **Body:** `{ govPartnerOrgIds: ["id-bpn", "id-pln"] }`
- **Desc:** Menambahkan instansi pendukung ke dalam _loop_ proyek.

---

## [MODULE MONITORING] (WBS & Progress)

### [MONITORING-TaskItem (WBS)]

- **GET** `/monitoring/project/{projectId}/tasks`
- **Auth:** Bearer
- **Desc:** Melihat struktur WBS (Tree structure) dan status progress per item.

- **POST** `/monitoring/project/{projectId}/tasks`
- **Auth:** Bearer
- **OrgRole:** `EXECUTING_AGENCY` (PMO)
- **Desc:** Membuat/Upload struktur WBS (Schedule Baseline). Input: Title, Weight, Deadline, ParentId.

### [MONITORING-ProgressLog]

- **POST** `/monitoring/task/{taskId}/log`
- **Auth:** Bearer
- **OrgRole:** `CONTRACTOR`
- **MemberRole:** `MEMBER` atau `ADMIN`
- **Desc:** Lapor progress mingguan/harian.
- _Input:_ % Progress, Deskripsi, **Upload Foto Bukti (Url)**.

- **GET** `/monitoring/task/{taskId}/log`
- **Auth:** Bearer
- **Desc:** Melihat history laporan pada task tertentu.

### [MONITORING-Validation]

- **POST** `/monitoring/log/{logId}/validate`
- **Auth:** Bearer
- **OrgRole:** `INDEPENDENT_AUDITOR`
- **Desc:** Auditor memvalidasi laporan kontraktor.
- _Input:_ `isAccepted` (Boolean), `realProgressPercentage`, `comment`.
- _Effect:_ Jika Accepted, angka ini masuk ke perhitungan Kurva S.

---

## [MODULE TRACKER] (Ticketing System)

### [TRACKER-Ticket]

- **POST** `/tracker/project/{projectId}/ticket`
- **Auth:** Bearer
- **OrgRole:** `CONTRACTOR`
- **Desc:** Melaporkan masalah lapangan (misal: Lahan belum bebas).
- _Input:_ Judul, Deskripsi, Lokasi, Foto Bukti, `taskItemId` (Opsional).
- _Status Awal:_ `OPEN`.

- **GET** `/tracker/project/{projectId}/ticket`
- **Auth:** Bearer
- **Desc:** List semua tiket. JICA memantau lewat endpoint ini.

### [TRACKER-Assignment & Resolution]

- **PUT** `/tracker/ticket/{ticketId}/assign`
- **Auth:** Bearer
- **OrgRole:** `EXECUTING_AGENCY`
- **Desc:** Meneruskan tiket (Assignment) ke instansi terkait.
- _Input:_ `assignToOrgId` (misal: ID BPN).
- _Status Baru:_ `ASSIGNED`.

- **POST** `/tracker/ticket/{ticketId}/log`
- **Auth:** Bearer
- **OrgRole:** `GOV_PARTNER` (Pihak yang di-assign)
- **Desc:** Update status pengerjaan masalah (misal: "Sedang diukur").
- _Input:_ Komentar, Bukti Dokumen.
- _Status Baru:_ `IN_PROGRESS` atau `RESOLVED`.

- **PUT** `/tracker/ticket/{ticketId}/close`
- **Auth:** Bearer
- **OrgRole:** `EXECUTING_AGENCY`
- **Desc:** Menutup tiket secara final setelah verifikasi lapangan.
- _Status Baru:_ `CLOSED`.

---

## [MODULE FINANCE] (Disbursement)

### [FINANCE-PaymentRequest]

- **POST** `/finance/project/{projectId}/request`
- **Auth:** Bearer
- **OrgRole:** `CONTRACTOR`
- **Desc:** Mengajukan termin pembayaran.
- _Input:_ Nominal, Judul (Termin 1), Invoice PDF Url.
- _Status:_ `SUBMITTED`.

- **GET** `/finance/project/{projectId}/request`
- **Auth:** Bearer
- **Desc:** Melihat list pengajuan pembayaran dan posisi approval saat ini.

### [FINANCE-ApprovalChain]

- **PUT** `/finance/request/{requestId}/approve-auditor`
- **Auth:** Bearer
- **OrgRole:** `INDEPENDENT_AUDITOR`
- **Desc:** Approval Level 1 (Teknis).
- _Status:_ `AUDITOR_APPROVED` / `AUDITOR_REJECTED`.

- **PUT** `/finance/request/{requestId}/approve-agency`
- **Auth:** Bearer
- **OrgRole:** `EXECUTING_AGENCY`
- **Desc:** Approval Level 2 (Administrasi).
- _Status:_ `AGENCY_APPROVED` / `AGENCY_REJECTED`.

- **PUT** `/finance/request/{requestId}/disburse`
- **Auth:** Bearer
- **OrgRole:** `LOAN_PROVIDER` (JICA)
- **Desc:** Konfirmasi Transfer Dana.
- _Status:_ `PAID`.

---

## [MODULE SUPPORT] (Utility)

- **POST** `/support/upload`
- **Auth:** Bearer (Semua Role)
- **Body:** `Multipart/Form-Data` (File)
- **Desc:** Upload file (Gambar/PDF) ke Cloud storage (Cloudinary).
- **Response:** `{ url: "https://res.cloudinary.com/..." }`
- _Usage:_ Endpoint ini dipanggil Frontend _sebelum_ Submit Progress/Ticket/Contract untuk mendapatkan URL file.

- **GET** `/support/dashboard/s-curve/{projectId}`
- **Auth:** Bearer
- **Desc:** Mengembalikan data JSON (Planned vs Actual) untuk grafik Kurva S.

- **GET** `/support/dashboard/stats/{projectId}`
- **Auth:** Bearer
- **Desc:** Mengembalikan statistik ringkas (Budget terserap, Jumlah Tiket Open, dll).
