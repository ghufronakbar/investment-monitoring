# 3.5 Perancangan Policy Layer

Policy layer dirancang untuk memusatkan aturan authorization. Pada sistem multi-stakeholder, authorization tidak cukup hanya melihat role pengguna. Sistem juga perlu memeriksa organisasi pengguna, membership, role organisasi, keterlibatan organisasi pada project, serta posisi organisasi dalam contract.

Aturan policy digunakan sebagai guard sebelum suatu aksi dijalankan. Contohnya, contractor hanya dapat melakukan submit progress atau payment request pada project yang terkait dengannya. Auditor hanya dapat melakukan validasi atau approval pada tahap auditor. Executing agency dapat mengelola assignment atau approval administratif, sedangkan loan provider dapat melakukan disbursement pada tahap yang valid.

Dengan policy layer, aturan akses tidak tersebar di banyak controller. Hal ini membuat authorization lebih konsisten, lebih mudah diuji, dan lebih mudah dijelaskan sebagai bagian dari workflow governance.

[Gambar 3.4 - Alur Policy Check (berisi request user, membership, organization role, project access, contract role, dan keputusan allow/deny)]

[NOTE: Jika nanti ada diagram sequence, tampilkan policy check sebelum service menjalankan business logic.]
