# 4.2 Implementasi FSM Transition Engine

FSM transition engine diterapkan untuk memastikan perubahan status pada workflow utama berjalan melalui transisi yang valid. Workflow yang digunakan sebagai contoh implementasi adalah monitoring progress, ticket hambatan, dan finance approval. Ketiga workflow tersebut memiliki karakteristik yang sama, yaitu memiliki state, action, actor, dan aturan transisi yang harus dipenuhi sebelum status berubah.

Pada monitoring progress, laporan yang dikirim oleh contractor tidak langsung dianggap sebagai realisasi valid sebelum melalui validasi auditor. Pada ticket hambatan, status bergerak secara bertahap dari ticket dibuat hingga ditutup. Pada finance approval, pengajuan pembayaran harus melalui tahap approval auditor dan executing agency sebelum dapat diproses oleh loan provider. Dengan FSM, backend dapat menolak transisi yang tidak sesuai, misalnya payment request yang langsung berpindah dari `SUBMITTED` ke `PAID`.

Implementasi ini membuat workflow lebih deterministik. Setiap action memiliki state awal, state tujuan, serta aktor yang berwenang. Jika state saat ini tidak sesuai dengan action yang diminta, sistem mengembalikan error workflow. Hal ini mengurangi risiko status loncat dan menjaga konsistensi proses lintas organisasi.

[Tabel 4.1 - Contoh Transisi FSM Workflow (berisi workflow, current state, action, next state, dan actor)]

[NOTE: Isi tabel dengan status aktual dari backend. Contoh finance: `SUBMITTED -> AUDITOR_APPROVED -> AGENCY_APPROVED -> PAID`.]
