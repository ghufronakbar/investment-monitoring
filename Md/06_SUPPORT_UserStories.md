# Module: SUPPORT (Utility & Dashboard)

## Epics

Fungsi pendukung visualisasi data dan manajemen file.

---

### Story 1: Upload File

**As a** User (All Roles),
**I want to** upload images or PDFs,
**So that** I can attach them to my reports/contracts.

#### Acceptance Criteria

1.  Support format: JPG, PNG, PDF.
2.  Limit size: Max 5MB per file.
3.  Return: Public URL dari Cloudinary.
4.  Secure handling (tidak boleh upload file executable).

#### Technical Notes

- Endpoint: `POST /support/upload`.

---

### Story 2: View S-Curve (Dashboard)

**As a** Stakeholder (JICA/Bappenas/MRT),
**I want to** see the S-Curve graph overlaying Plan vs Actual,
**So that** I can instantly know if the project is delayed.

#### Acceptance Criteria

1.  **Garis Plan:** Dihitung dari akumulasi bobot `TaskItem` berdasarkan `Deadline`.
2.  **Garis Actual:** Dihitung dari akumulasi `real_progress_percentage` (yang sudah divalidasi Auditor) dikali Bobot Task.
3.  **Deviasi:** Menampilkan selisih % (Negatif = Terlambat).
4.  Response berupa JSON data points yang siap dirender oleh charting library di Frontend.

#### Technical Notes

- Endpoint: `GET /support/dashboard/s-curve/{id}`.
