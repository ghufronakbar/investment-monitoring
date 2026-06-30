# 2.2 Finite State Machine untuk Workflow

Finite State Machine (FSM) adalah model komputasi yang merepresentasikan sistem sebagai himpunan state dan aturan perpindahan antar state. Dalam konteks workflow, FSM digunakan untuk memastikan bahwa suatu proses hanya dapat berpindah dari satu status ke status lain melalui transisi yang valid. Model ini cocok untuk proses bisnis yang memiliki tahapan jelas, aktor tertentu, dan aturan yang harus dipenuhi sebelum status berubah.

Secara umum, FSM dapat direpresentasikan sebagai:

```text
FSM = (S, I, f, s0, F)
```

Keterangan:

- `S` adalah himpunan state;
- `I` adalah input atau event yang memicu transisi;
- `f` adalah fungsi transisi;
- `s0` adalah state awal;
- `F` adalah himpunan final state.

Pada sistem monitoring proyek, FSM dapat digunakan untuk memodelkan progress validation, ticket hambatan, dan approval pembayaran. Sebagai contoh, ticket dapat berpindah dari `OPEN` ke `ASSIGNED`, kemudian `IN_PROGRESS`, `RESOLVED`, dan `CLOSED`. Jika sistem menerima transisi yang tidak sesuai, misalnya langsung dari `OPEN` ke `CLOSED`, backend dapat menolak aksi tersebut.

Penggunaan FSM membantu membuat workflow lebih deterministik. Setiap perubahan status memiliki aturan yang eksplisit, sehingga sistem dapat mencegah status loncat, invalid transition, dan aksi yang dilakukan oleh aktor yang tidak sesuai. Dengan demikian, FSM menjadi fondasi penting dalam backend workflow governance.

[Gambar 2.2 - Contoh Diagram FSM Ticket Hambatan (berisi state OPEN, ASSIGNED, IN_PROGRESS, RESOLVED, CLOSED dan arah transisinya)]

[Tabel 2.1 - Contoh Matriks Transisi FSM (berisi current state, input/action, next state, dan actor yang berwenang)]
