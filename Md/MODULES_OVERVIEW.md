# System Modules Overview

**Project:** MRT Jakarta Phase 2 Monitoring System
**Architecture:** Modular Monolith (Logical Separation)

Dokumen ini menjelaskan batasan domain (_Domain Boundaries_) dan logika bisnis untuk setiap modul dalam sistem.

---

## 🔐 1. MODULE IAM (Identity & Access Management)

**Core Responsibility:** Mengelola identitas pengguna, struktur organisasi, dan hak akses (RBAC).

### Konsep Kunci

- **Invitation-Only System:** Tidak ada fitur registrasi publik. Akun hanya bisa dibuat melalui undangan email dari Admin.
- **Organization-Centric:** User tidak berdiri sendiri, melainkan melekat pada sebuah `Organization`. Hak akses user ditentukan oleh:
  1.  **Organization Role:** Peran instansi (misal: JICA sebagai `LOAN_PROVIDER`, Shimizu sebagai `CONTRACTOR`).
  2.  **Member Role:** Peran individu di dalam instansi (misal: `ADMIN` vs `MEMBER`).
- **Maker-Checker (Internal):** Struktur `OrganizationMember` memungkinkan skenario di mana staf (`MEMBER`) membuat draft, dan atasan (`ADMIN`) melakukan submit.

### Key Entities

- `User`: Data akun login (Email, Password, Profile).
- `Organization`: Entitas hukum/instansi (Nama, Logo, Tipe Role).
- `OrganizationMember`: Tabel penghubung (_Pivot_) antara User dan Organization beserta role internalnya.

---

## 🏗️ 2. MODULE PROJECT (Core Data)

**Core Responsibility:** Menyimpan definisi proyek dan konfigurasi parameter dasar.

### Konsep Kunci

- **G-to-G Representation:** Modul ini merepresentasikan kesepakatan level tinggi antar negara.
- **Legal Baseline:** Menyimpan dokumen "Loan Agreement" sebagai acuan hukum.
- **S-Curve Parameters:** Menyimpan konfigurasi `Total Budget`, `Start Date`, dan `Deadline` yang menjadi sumbu X dan Y pada grafik Kurva S.

### Key Entities

- `Project`: Data header proyek (Nama, Deskripsi, File Loan Agreement).
- `ProjectSetting`: Konfigurasi teknis (Budget, Currency, Timeline Baseline).

---

## 🤝 3. MODULE CONTRACTING (Assignment & Participants)

**Core Responsibility:** Mengelola penunjukan pelaksana (Kontraktor) dan pengawas (Auditor/Partner) dalam sebuah proyek.

### Konsep Kunci

- **Direct Assignment:** Menggantikan proses tender yang rumit. Admin MRT ("Executing Agency") secara langsung menunjuk organisasi mana yang menjadi Kontraktor dan Auditor di sistem.
- **Participant Mapping:** Memetakan organisasi ke dalam peran spesifik di proyek:
  - _Vendor Participant:_ Siapa yang kerja? (Contractor).
  - _Auditor Participant:_ Siapa yang mengawasi? (Independent Auditor).
  - _Gov Participants:_ Siapa yang membantu izin? (BPN, PLN, dll).
- **Contract Value:** Menyimpan nilai kontrak konstruksi final (`contractValue`) dan dokumen kontrak fisik.

### Key Entities

- `Contract`: Pusat relasi antar-organisasi dalam satu proyek.
- `GovParticipant`: Tabel pivot untuk _Multi-Agency Support_ (banyak instansi pemerintah dalam satu kontrak).

---

## 📈 4. MODULE MONITORING (WBS & Progress)

**Core Responsibility:** Memantau progres fisik lapangan berdasarkan jadwal kerja (_Master Schedule_).

### Konsep Kunci

- **WBS (Work Breakdown Structure):** Struktur pekerjaan hirarkis (Tree Structure).
  - Setiap item pekerjaan (`TaskItem`) memiliki **Bobot (Weight)**. Total bobot satu proyek = 100%.
- **Log & Snapshot:** Ketika Kontraktor melapor progres, sistem menyimpan "Snapshot" (Judul & Bobot saat itu). Ini menjaga integritas data sejarah jika WBS berubah di masa depan.
- **Validation Flow:**
  1.  **Claim:** Kontraktor lapor % progres + Foto Bukti.
  2.  **Validate:** Auditor cek lapangan -> Approve/Reject.
  3.  **Calculation:** Hanya data yang _Approved_ yang dihitung masuk ke "Realisasi Kurva S".

### Key Entities

- `TaskItem`: Item pekerjaan (Judul, Bobot, Deadline, Parent ID).
- `ProgressLog`: Laporan klaim dari kontraktor (History).
- `ProgressValidation`: Keputusan auditor atas klaim tersebut (Single Source of Truth).

---

## 🚧 5. MODULE TRACKER (Ticketing System)

**Core Responsibility:** Menyelesaikan hambatan (_Bottleneck_) di lapangan yang melibatkan lintas instansi.

### Konsep Kunci

- **Obstruction Reporting:** Kontraktor melaporkan masalah yang menghambat pekerjaan fisik (misal: Lahan belum bebas).
- **Delegated Resolution:** MRT tidak menyelesaikan masalah sendiri, tapi menugaskan (_Assign_) tiket ke instansi berwenang (`GOV_PARTNER`).
  - _Contoh:_ Tiket "Pipa Gas Bocor" -> Di-assign ke PGN/Pertamina.
- **Transparency:** JICA (Loan Provider) bisa melihat daftar tiket yang statusnya `OPEN` atau `STUCK`, memberikan visibilitas terhadap kinerja birokrasi Indonesia.

### Key Entities

- `Ticket`: Data masalah (Judul, Lokasi, Foto Bukti, Status).
- `TicketLog`: Riwayat aktivitas penyelesaian (Komentar, Ubah Status, Lampiran Surat).

---

## 💸 6. MODULE FINANCE (Disbursement)

**Core Responsibility:** Mengelola pencairan dana termin pembayaran (_Payment Milestone_).

### Konsep Kunci

- **Tiered Approval (Persetujuan Bertingkat):** Uang tidak cair sembarangan. Harus melalui 3 lapis validasi:
  1.  **Technical Approval (Auditor):** Memastikan fisik di lapangan sesuai tagihan.
  2.  **Administrative Approval (MRT):** Memastikan dokumen lengkap.
  3.  **Disbursement (JICA):** Eksekusi transfer dana.
- **Financial Progress:** Data pembayaran yang sukses (`PAID`) akan menaikkan persentase "Serapan Anggaran" di dashboard.

### Key Entities

- `PaymentRequest`: Pengajuan tagihan (Nominal, Invoice PDF, Status Approval).

---

## 🛠️ 7. MODULE SUPPORT (Utility)

**Core Responsibility:** Fungsi pendukung non-bisnis.

### Konsep Kunci

- **Centralized File Management:** Semua upload file (Gambar/PDF) ditangani modul ini untuk dikirim ke Cloud Storage (Cloudinary). Modul lain hanya menyimpan URL string-nya.
- **Dashboard Aggregator:** Menyediakan endpoint khusus untuk menghitung statistik berat seperti Kurva S (Plan vs Actual) agar tidak membebani modul transaksional.

### Key Entities

- _(Tidak ada tabel spesifik, menggunakan logic kalkulasi)_
