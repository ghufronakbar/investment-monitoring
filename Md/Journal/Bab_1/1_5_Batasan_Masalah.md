# 1.5 Batasan Masalah

Penelitian ini memiliki beberapa batasan agar pembahasan tetap fokus.

Pertama, sistem yang dibahas merupakan prototipe akademik berbasis studi kasus, bukan sistem resmi yang telah digunakan atau diintegrasikan dengan MRT Jakarta, Pemprov DKI, JICA, maupun kontraktor terkait.

Kedua, fokus penelitian berada pada backend workflow governance. Pembahasan utama dibatasi pada FSM, organization-centric access control, SLA tracking dan auditability, serta realtime event delivery. Fitur pendukung seperti dashboard cache, secure upload, refresh token revocation, rate limiting, dan detail frontend tidak menjadi fokus utama.

Ketiga, WebSocket hanya digunakan sebagai channel sinyal perubahan data. Data detail tetap diambil melalui REST API, sehingga REST API dan database tetap menjadi source of truth.

Keempat, evaluasi difokuskan pada correctness dan contract, bukan benchmark produksi, high availability, atau integrasi eksternal lintas instansi.
