# 3.6 Perancangan SLA Tracking dan Auditability

SLA tracking dirancang untuk mengukur durasi suatu entity berada pada state tertentu. Data yang dibutuhkan meliputi entity type, entity id, state aktif, owner organization, waktu mulai state, batas waktu penyelesaian, dan waktu penyelesaian. Dari data tersebut, sistem dapat menentukan apakah proses masih normal, mendekati batas waktu, atau sudah melanggar SLA.

Auditability dirancang melalui audit trail yang mencatat aktivitas penting pada sistem. Audit trail menyimpan informasi seperti actor, organisasi actor, action, entity, waktu, dan konteks request. Catatan ini digunakan untuk meningkatkan traceability terhadap perubahan data dan keputusan yang terjadi dalam workflow.

SLA tracking dan audit trail saling melengkapi. SLA menunjukkan proses mana yang berpotensi terlambat, sedangkan audit trail menunjukkan siapa yang melakukan tindakan terhadap proses tersebut. Keduanya penting untuk mendukung accountability pada proyek lintas lembaga.

[Gambar 3.5 - Rancangan SLA Tracking dan Audit Trail (berisi state aktif, due time, SLA status, action, dan audit log)]

[NOTE: Jika ada aturan SLA spesifik per status, isi durasi SLA sesuai implementasi atau asumsi penelitian.]
