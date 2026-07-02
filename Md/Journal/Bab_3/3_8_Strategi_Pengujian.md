# 3.8 Strategi Pengujian

Pengujian dilakukan untuk memastikan bahwa rancangan backend berjalan sesuai aturan. Fokus pengujian diarahkan pada empat pilar utama penelitian, yaitu FSM workflow, policy layer, SLA/auditability, dan realtime event delivery.

Pengujian workflow dilakukan untuk memastikan valid transition diterima dan invalid transition ditolak. Pengujian policy dilakukan untuk memastikan aktor yang tidak memiliki kewenangan tidak dapat mengakses atau menjalankan aksi tertentu. Pengujian SLA dan audit dilakukan untuk memastikan status SLA dapat dihitung dan aksi penting dapat dicatat. Pengujian realtime dilakukan untuk memastikan WebSocket dapat connect, subscribe topic, menerima event, dan menangani rejected topic.

Selain itu, contract verification dilakukan untuk memastikan frontend dapat mengonsumsi response backend yang berkaitan dengan fitur utama, seperti SLA ticket, workflow action, dan realtime event. Verifikasi build digunakan untuk memastikan perubahan tidak menyebabkan error kompilasi.

[Tabel 3.3 - Strategi Pengujian Sistem (berisi area pengujian, tujuan, dan contoh skenario)]

[NOTE: Setelah menjalankan test final, tambahkan hasil aktual seperti jumlah test yang pass, command yang digunakan, dan catatan warning jika ada.]
