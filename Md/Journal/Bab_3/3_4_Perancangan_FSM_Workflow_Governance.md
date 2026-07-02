# 3.4 Perancangan FSM Workflow Governance

Workflow utama dimodelkan menggunakan Finite State Machine (FSM). FSM digunakan untuk memastikan setiap proses hanya dapat berpindah status melalui transisi yang valid. Pada penelitian ini, workflow yang dijadikan contoh utama adalah monitoring progress, ticket hambatan, dan finance approval.

Pada workflow monitoring progress, contractor mengirim laporan progress, kemudian auditor melakukan validasi. Status progress tidak langsung dianggap valid sebelum melalui proses validasi. Pada workflow ticket, hambatan bergerak dari status awal hingga selesai melalui tahapan seperti open, assigned, in progress, resolved, dan closed. Pada workflow finance, pengajuan pembayaran harus melewati approval bertingkat sebelum dinyatakan paid.

Setiap workflow memiliki state, input transition, actor, dan guard condition. Jika actor tidak sesuai atau transisi tidak valid, backend menolak aksi tersebut. Dengan demikian, FSM berfungsi sebagai mekanisme kontrol agar proses tidak melompat tahap.

[Gambar 3.3 - Rancangan FSM Workflow Utama (berisi tiga workflow: progress validation, ticket, dan finance approval)]

[Tabel 3.2 - Matriks Transisi Workflow (berisi workflow, current state, action, next state, dan actor)]

[NOTE: Jika ingin lebih detail, isi Tabel 3.2 berdasarkan enum/status aktual di backend agar konsisten dengan implementasi.]
