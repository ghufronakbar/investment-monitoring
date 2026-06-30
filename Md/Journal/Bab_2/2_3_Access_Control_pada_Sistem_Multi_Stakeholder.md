# 2.3 Access Control pada Sistem Multi-Stakeholder

Access control adalah mekanisme untuk menentukan siapa yang boleh mengakses resource dan menjalankan aksi tertentu pada sistem. Model yang umum digunakan adalah Role-Based Access Control (RBAC), yaitu pemberian hak akses berdasarkan role pengguna. Namun, pada sistem multi-stakeholder, RBAC sederhana sering tidak cukup karena kewenangan pengguna juga bergantung pada organisasi, relasi organisasi terhadap proyek, dan peran organisasi dalam kontrak.

Pada proyek infrastruktur lintas lembaga, satu pengguna dapat menjadi anggota suatu organisasi, sementara organisasi tersebut dapat memiliki peran berbeda pada project tertentu. Misalnya, organisasi dapat bertindak sebagai contractor, auditor, executing agency, atau loan provider. Karena itu, sistem perlu memeriksa bukan hanya role user, tetapi juga membership, role organisasi, project participation, dan contract role.

Pendekatan tersebut dapat disebut sebagai organization-centric access control. Pada model ini, authorization ditentukan dari hubungan antara user, organisasi, project, contract, dan action. Sebagai contoh, kontraktor dapat melakukan submit progress, auditor dapat melakukan validasi progress, executing agency dapat mengelola assignment atau approval administratif, sedangkan loan provider dapat melakukan disbursement pada tahap tertentu.

Model access control seperti ini penting untuk menjaga boundary antar stakeholder. Tanpa pembatasan yang tepat, aktor dari organisasi yang tidak berwenang dapat melihat atau mengubah data yang bukan tanggung jawabnya. Dalam konteks workflow governance, access control juga berfungsi sebagai guard condition agar transisi workflow hanya dapat dilakukan oleh aktor yang sesuai.

[Gambar 2.3 - Model Organization-Centric Access Control (berisi relasi user, organization, membership, project, contract, role, dan action)]
