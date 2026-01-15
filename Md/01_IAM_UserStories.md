# Module: IAM (Identity & Access Management)

## Epics

Manajemen identitas yang ketat berbasis undangan (Invite-Only) untuk menjamin keamanan data proyek strategis negara.

---

## Auth Routes

### 1. Check Email

**Endpoint:** `POST /iam/auth/check-email`

**Description:** Mengecek status email untuk menentukan flow autentikasi selanjutnya.

**Authentication:** ❌ Tidak diperlukan (Public)

**Request Body:**

```json
{
  "email": "string (required, valid email format)"
}
```

**Response:**

```json
{
  "status": "NOT_FOUND" | "SETUP_REQUIRED" | "LOGIN_READY"
}
```

**Flow:**

1. `NOT_FOUND` → Email tidak terdaftar, tolak akses
2. `SETUP_REQUIRED` → Email terdaftar tapi belum set password, arahkan ke set-password flow
3. `LOGIN_READY` → Email terdaftar dan sudah set password, tampilkan input password

**Success Code:** `SUCCESS_CHECK_EMAIL`

---

### 2. Login

**Endpoint:** `POST /iam/auth/login`

**Description:** Login dengan email dan password untuk mendapatkan JWT tokens.

**Authentication:** ❌ Tidak diperlukan (Public)

**Request Body:**

```json
{
  "email": "string (required, valid email format)",
  "password": "string (required, min 8 characters)"
}
```

**Response:**

```json
{
  "accessToken": "string (JWT)",
  "refreshToken": "string (JWT)",
  "user": {
    "id": "uuid",
    "name": "string",
    "email": "string",
    "imageUrl": "string | null"
  },
  "memberships": [
    {
      "id": "uuid (membership id)",
      "memberRole": "ADMIN | MEMBER",
      "orgId": "uuid",
      "orgName": "string",
      "orgRole": "CONTRACTOR | GOV_PARTNER | INDEPENDENT_AUDITOR | LOAN_PROVIDER | EXECUTING_AGENCY",
      "orgLogoUrl": "string | null",
      "createdAt": "timestamp"
    }
  ]
}
```

**Flow:**

1. Validasi email terdaftar
2. Verifikasi password
3. Generate Access Token (short-lived) dan Refresh Token (long-lived)
4. Return user data dengan semua organization memberships

**Error Codes:**

- `USER_NOT_FOUND` - Email tidak terdaftar
- `INVALID_CREDENTIALS` - Password salah
- `PASSWORD_NOT_SET` - Belum set password

**Success Code:** `SUCCESS_LOGIN`

---

### 3. Set Password

**Endpoint:** `POST /iam/auth/set-password`

**Description:** Set password untuk user baru yang diundang.

**Authentication:** ❌ Tidak diperlukan (Public)

**Request Body:**

```json
{
  "email": "string (required, valid email format)",
  "password": "string (required, min 8 characters)",
  "confirmPassword": "string (required, must match password)"
}
```

**Response:**

```json
{
  "message": "Password set successfully"
}
```

**Flow:**

1. Validasi email terdaftar
2. Validasi password belum diset sebelumnya
3. Hash password dan simpan ke database
4. User dapat login setelah set password

**Error Codes:**

- `USER_NOT_FOUND` - Email tidak terdaftar
- `PASSWORD_ALREADY_SET` - Password sudah diset

**Success Code:** `SUCCESS_SET_PASSWORD`

---

### 4. Refresh Token

**Endpoint:** `POST /iam/auth/refresh-token`

**Description:** Mendapatkan access token baru menggunakan refresh token.

**Authentication:** ❌ Tidak diperlukan (Public, tapi butuh valid refresh token)

**Request Body:**

```json
{
  "refreshToken": "string (required, valid JWT refresh token)"
}
```

**Response:**

```json
{
  "accessToken": "string (new JWT access token)",
  "refreshToken": "string (rotated refresh token)"
}
```

**Flow:**

1. Validasi refresh token
2. Generate access token baru
3. Rotate refresh token (old refresh token menjadi invalid)
4. Return tokens baru

**Error Codes:**

- `INVALID_REFRESH_TOKEN` - Token tidak valid atau expired

**Success Code:** `SUCCESS_REFRESH_TOKEN`

---

## Organization Routes

### 5. Get Registered Organizations (Public)

**Endpoint:** `GET /iam/organization/registered`

**Description:** Mendapatkan daftar semua organisasi yang terdaftar di sistem.

**Authentication:** ❌ Tidak diperlukan (Public)

**Request Body:** None

**Response:**

```json
[
  {
    "id": "uuid",
    "name": "string",
    "role": "CONTRACTOR | GOV_PARTNER | ...",
    "logoUrl": "string | null",
    "createdAt": "timestamp"
  }
]
```

**Use Case:** Digunakan di halaman login/registrasi untuk menampilkan organisasi yang tersedia.

**Success Code:** `SUCCESS_GET_REGISTERED_ORGANIZATIONS`

---

### 6. Get My Organizations

**Endpoint:** `GET /iam/organization/my`

**Description:** Mendapatkan semua organisasi yang di-join oleh user yang sedang login.

**Authentication:** ✅ Bearer Token (Access Token)

**Request Body:** None

**Response:**

```json
[
  {
    "membershipId": "uuid",
    "memberRole": "ADMIN | MEMBER",
    "orgId": "uuid",
    "orgName": "string",
    "orgRole": "CONTRACTOR | GOV_PARTNER | ...",
    "orgLogoUrl": "string | null",
    "joinedAt": "timestamp"
  }
]
```

**Use Case:** Digunakan untuk menampilkan organization switcher di dashboard.

**Success Code:** `SUCCESS_GET_MY_ORGANIZATIONS`

---

### 7. Create Organization

**Endpoint:** `POST /iam/organization/{orgId}`

**Description:** Membuat organisasi baru beserta Admin pertama (auto-invite).

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Allowed Organization Roles:** `GOV_SUPERVISOR`, `EXECUTING_AGENCY`
- **Allowed Member Roles:** `ADMIN` only

**URL Parameters:**

- `orgId`: UUID of the caller's organization (for authorization check)

**Request Body:**

```json
{
  "name": "string (required, organization name)",
  "role": "CONTRACTOR | GOV_PARTNER | INDEPENDENT_AUDITOR | LOAN_PROVIDER | EXECUTING_AGENCY (required)",
  "logoUrl": "string (optional, URL to organization logo)",
  "adminName": "string (required, name of first admin)",
  "adminEmail": "string (required, valid email format)"
}
```

**Response:**

```json
{
  "organization": {
    "id": "uuid",
    "name": "string",
    "role": "string",
    "logoUrl": "string | null",
    "createdAt": "timestamp"
  },
  "admin": {
    "id": "uuid",
    "name": "string",
    "email": "string",
    "imageUrl": "null"
  }
}
```

**Flow:**

1. Validasi caller adalah ADMIN dari GOV_SUPERVISOR atau EXECUTING_AGENCY
2. Buat organisasi baru
3. Buat user baru dengan email admin (password = null)
4. Buat organization_members dengan role ADMIN
5. Kirim email undangan ke admin
6. Return data organisasi dan admin

**Error Codes:**

- `FORBIDDEN` - Caller bukan ADMIN atau bukan dari org yang diperbolehkan
- `EMAIL_ALREADY_EXISTS` - Email admin sudah terdaftar

**Success Code:** `SUCCESS_CREATE_ORGANIZATION`

---

### 8. Get Organization Detail

**Endpoint:** `GET /iam/organization/{orgId}`

**Description:** Mendapatkan detail organisasi berdasarkan ID.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Allowed Organization Roles:** Semua (Any Organization)
- **Allowed Member Roles:** Semua (ADMIN atau MEMBER)

**URL Parameters:**

- `orgId`: UUID of the organization to retrieve

**Request Body:** None

**Response:**

```json
{
  "id": "uuid",
  "name": "string",
  "role": "CONTRACTOR | GOV_PARTNER | ...",
  "logoUrl": "string | null",
  "createdAt": "timestamp"
}
```

**Access Control:** User harus menjadi member dari organisasi tersebut.

**Error Codes:**

- `FORBIDDEN` - User bukan member dari organisasi
- `ORGANIZATION_NOT_FOUND` - Organisasi tidak ditemukan

**Success Code:** `SUCCESS_GET_ORGANIZATION`

---

### 9. Get Organization Members

**Endpoint:** `GET /iam/organization/{orgId}/member`

**Description:** Mendapatkan daftar semua member dari organisasi.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Allowed Organization Roles:** Semua (Any Organization)
- **Allowed Member Roles:** Semua (ADMIN atau MEMBER)

**URL Parameters:**

- `orgId`: UUID of the organization

**Request Body:** None

**Response:**

```json
[
  {
    "id": "uuid (membership id)",
    "userId": "uuid",
    "name": "string",
    "email": "string",
    "imageUrl": "string | null",
    "memberRole": "ADMIN | MEMBER",
    "createdAt": "timestamp"
  }
]
```

**Access Control:** User harus menjadi member dari organisasi tersebut.

**Success Code:** `SUCCESS_GET_MEMBERS`

---

### 10. Invite Member

**Endpoint:** `POST /iam/organization/{orgId}/member/invite`

**Description:** Mengundang user baru untuk bergabung ke organisasi.

**Authentication:** ✅ Bearer Token (Access Token)

**Authorization:**

- **Allowed Organization Roles:** Semua (Any Organization)
- **Allowed Member Roles:** `ADMIN` only

**URL Parameters:**

- `orgId`: UUID of the organization

**Request Body:**

```json
{
  "name": "string (required, name of new member)",
  "email": "string (required, valid email format)"
}
```

**Response:**

```json
{
  "member": {
    "id": "uuid (membership id)",
    "name": "string",
    "email": "string",
    "memberRole": "MEMBER",
    "createdAt": "timestamp"
  }
}
```

**Flow:**

1. Validasi caller adalah ADMIN dari organisasi
2. Cek apakah email sudah terdaftar
   - Jika belum: buat user baru dengan password null
   - Jika sudah: gunakan user existing
3. Buat organization_members dengan role MEMBER
4. Kirim email undangan
5. Return data member baru

**Error Codes:**

- `FORBIDDEN` - Caller bukan ADMIN
- `MEMBER_ALREADY_EXISTS` - User sudah menjadi member organisasi

**Success Code:** `SUCCESS_INVITE_MEMBER`

---

## Summary Table

| #   | Endpoint                                  | Method | Auth | Authorization                           | Description                   |
| --- | ----------------------------------------- | ------ | ---- | --------------------------------------- | ----------------------------- |
| 1   | `/iam/auth/check-email`                   | POST   | ❌   | -                                       | Check email status            |
| 2   | `/iam/auth/login`                         | POST   | ❌   | -                                       | Login with password           |
| 3   | `/iam/auth/set-password`                  | POST   | ❌   | -                                       | Set password for invited user |
| 4   | `/iam/auth/refresh-token`                 | POST   | ❌   | Refresh Token                           | Get new access token          |
| 5   | `/iam/organization/registered`            | GET    | ❌   | -                                       | List all organizations        |
| 6   | `/iam/organization/my`                    | GET    | ✅   | -                                       | List user's organizations     |
| 7   | `/iam/organization/{orgId}`               | POST   | ✅   | GOV_SUPERVISOR/EXECUTING_AGENCY + ADMIN | Create organization           |
| 8   | `/iam/organization/{orgId}`               | GET    | ✅   | Member of org                           | Get organization detail       |
| 9   | `/iam/organization/{orgId}/member`        | GET    | ✅   | Member of org                           | List organization members     |
| 10  | `/iam/organization/{orgId}/member/invite` | POST   | ✅   | ADMIN of org                            | Invite new member             |
