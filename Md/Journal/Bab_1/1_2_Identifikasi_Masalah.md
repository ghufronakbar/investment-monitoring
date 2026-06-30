# 1.2 Identifikasi Masalah

Berdasarkan latar belakang tersebut, terdapat beberapa masalah utama dari sisi Informatika.

Pertama, workflow proyek lintas lembaga sulit dikontrol jika hanya dimodelkan sebagai operasi CRUD biasa. Progress pekerjaan, ticket hambatan, dan approval pembayaran memiliki tahapan yang harus dilalui secara berurutan. Tanpa aturan transisi yang jelas, sistem berisiko menerima perubahan status yang tidak valid.

Kedua, akses sistem tidak cukup ditentukan oleh role pengguna secara umum. Pada sistem multi-stakeholder, kewenangan bergantung pada organisasi pengguna, role organisasi, membership, relasi organisasi terhadap project, dan posisi organisasi dalam contract.

Ketiga, keterlambatan proses sering tidak terukur secara eksplisit. Status seperti "sedang diproses" tidak menunjukkan apakah proses masih dalam batas wajar atau sudah menjadi bottleneck.

Keempat, perubahan status perlu memiliki riwayat yang dapat ditelusuri. Sistem harus dapat mencatat siapa melakukan aksi, kapan, dari organisasi mana, dan terhadap entity apa.

Kelima, dashboard membutuhkan pembaruan informasi yang responsif, tetapi data realtime tidak boleh menggantikan REST API dan database sebagai sumber kebenaran.
