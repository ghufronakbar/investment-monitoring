# 4.1 Implementasi Arsitektur Backend Workflow Governance

Implementasi backend pada penelitian ini menggunakan pendekatan modular monolith. Seluruh domain berjalan dalam satu aplikasi backend, tetapi dipisahkan ke dalam module yang memiliki tanggung jawab berbeda. Pemisahan tersebut diterapkan pada module IAM, Project, Monitoring, Tracker, Finance, Support, Policy, Workflow, Audit, Outbox, dan Realtime. Pendekatan ini dipilih agar sistem tetap sederhana untuk dikembangkan sebagai prototipe, tetapi tetap memiliki batas domain yang jelas.

Backend dibangun menggunakan Go dengan PostgreSQL sebagai basis data utama. SQLC digunakan untuk menghasilkan akses database yang type-safe berdasarkan query SQL eksplisit. Redis digunakan sebagai komponen pendukung untuk kebutuhan runtime tertentu, sedangkan WebSocket digunakan sebagai channel realtime delivery. REST API tetap menjadi kontrak utama untuk pengambilan dan perubahan data.

Arsitektur ini mendukung kebutuhan workflow governance karena business rule tidak hanya berada pada controller. Validasi workflow ditempatkan pada layer workflow, authorization dipusatkan pada policy layer, aktivitas penting dicatat melalui audit service, dan event domain dikirim melalui outbox. Dengan struktur tersebut, sistem dapat memisahkan tanggung jawab antara transport HTTP, business logic, access control, auditability, dan event delivery.

[Gambar 4.1 - Implementasi Arsitektur Backend Workflow Governance (berisi module IAM, Project, Monitoring, Tracker, Finance, Policy, Workflow, Audit, Outbox, Realtime, PostgreSQL, Redis, REST API, dan WebSocket)]

[NOTE: Jika ingin lebih konkret, tambahkan nama folder implementasi seperti `internal/policy`, `internal/workflow`, `internal/audit`, `internal/outbox`, dan `internal/module/realtime`.]
