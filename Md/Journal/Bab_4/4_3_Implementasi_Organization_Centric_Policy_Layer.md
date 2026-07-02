# 4.3 Implementasi Organization-Centric Policy Layer

Policy layer diimplementasikan untuk memusatkan aturan authorization pada sistem. Pada sistem multi-stakeholder, authorization tidak cukup ditentukan oleh role pengguna secara umum. Sistem juga harus memeriksa organisasi pengguna, membership, role organisasi, keterlibatan organisasi pada project, serta posisi organisasi dalam contract.

Implementasi policy layer digunakan pada berbagai workflow utama. Contractor hanya dapat melakukan aksi yang berkaitan dengan pelaporan progress atau pengajuan pembayaran pada project yang relevan. Auditor hanya dapat melakukan validasi atau approval pada tahap auditor. Executing agency dapat mengelola assignment dan approval administratif, sedangkan loan provider dapat melakukan disbursement pada tahap yang valid.

Pemusatan authorization pada policy layer membuat controller lebih fokus pada parsing request dan response. Business rule terkait akses tidak tersebar di banyak handler, sehingga lebih mudah diuji dan dijaga konsistensinya. Selain itu, policy layer juga berperan sebagai guard sebelum workflow transition dijalankan.

[Gambar 4.2 - Implementasi Policy Layer (berisi request user, policy check, project access, contract role, workflow action, dan hasil allow/deny)]

[NOTE: Jika dibutuhkan, tambahkan contoh nama policy dari kode seperti `CanSubmitProgress`, `CanValidateProgress`, `CanCreateTicket`, `CanAssignTicket`, atau `CanViewFinanceProject`.]
