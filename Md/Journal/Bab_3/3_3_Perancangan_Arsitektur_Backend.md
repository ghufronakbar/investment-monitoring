# 3.3 Perancangan Arsitektur Backend

Arsitektur backend dirancang menggunakan pendekatan modular monolith. Pendekatan ini dipilih karena sistem masih dapat dijalankan sebagai satu aplikasi, tetapi setiap domain tetap dipisahkan ke dalam module yang jelas. Pemisahan domain membantu menjaga struktur kode, memudahkan pengujian, dan mempertahankan konsistensi transaksi.

Lapisan utama backend terdiri dari HTTP router, middleware, controller, service, policy/workflow/audit/outbox, repository, dan database. REST API digunakan sebagai kontrak utama pengambilan data. PostgreSQL digunakan sebagai source of truth, sedangkan Redis digunakan sebagai komponen pendukung untuk beberapa kebutuhan runtime. WebSocket digunakan sebagai channel realtime delivery, tetapi data detail tetap diambil melalui REST API.

Secara teknis, backend menggunakan Go, PostgreSQL, SQLC, Redis, outbox worker, dan WebSocket. SQLC digunakan agar query SQL tetap eksplisit tetapi memiliki type safety pada sisi kode. Arsitektur ini mendukung kebutuhan utama penelitian, yaitu workflow governance, access control, SLA tracking, auditability, dan realtime delivery.

[Gambar 3.2 - Arsitektur Backend Modular Monolith (berisi client, REST API, WebSocket, controller, service, policy/workflow/audit/outbox, repository SQLC, PostgreSQL, dan Redis)]

[NOTE: Jika pada dokumen akhir ingin menyebut stack lengkap, sesuaikan dengan stack project: Go, chi router, PostgreSQL, SQLC, goose, Redis, JWT, dan WebSocket.]
