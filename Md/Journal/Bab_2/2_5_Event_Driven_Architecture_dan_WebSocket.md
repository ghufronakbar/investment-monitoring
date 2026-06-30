# 2.5 Event-Driven Architecture dan WebSocket

Event-driven architecture adalah pendekatan arsitektur yang menggunakan event sebagai representasi perubahan penting pada sistem. Dalam sistem monitoring proyek, event dapat muncul ketika progress dikirim, progress divalidasi, ticket dibuat, ticket diselesaikan, atau pembayaran disetujui. Event tersebut dapat digunakan untuk memberi tahu bagian lain dari sistem bahwa terdapat data yang perlu diperbarui.

Salah satu tantangan dalam event-driven architecture adalah menjaga konsistensi antara transaksi utama dan pengiriman event. Jika data berhasil disimpan tetapi event gagal dikirim, client dapat kehilangan informasi perubahan. Untuk mengurangi risiko tersebut, outbox pattern digunakan. Pada pola ini, event disimpan terlebih dahulu ke tabel outbox dalam transaksi yang sama dengan perubahan data utama. Setelah itu, worker membaca outbox dan mengirim event ke channel yang sesuai.

WebSocket dapat digunakan sebagai channel realtime delivery karena menyediakan koneksi dua arah antara client dan server. Client dapat melakukan subscribe ke topic tertentu, misalnya topic project monitoring, ticket, atau finance. Ketika event diproses, server dapat mengirim notifikasi ke client yang berhak menerima topic tersebut.

Namun, WebSocket tidak harus menjadi sumber kebenaran data. Pada penelitian ini, WebSocket diposisikan sebagai sinyal perubahan. Setelah menerima event, client mengambil ulang data melalui REST API. Dengan pendekatan ini, sistem tetap memperoleh responsivitas realtime, sementara konsistensi data tetap bergantung pada REST API dan database utama.

[Gambar 2.5 - Alur Outbox Pattern dan WebSocket Delivery (berisi business transaction, outbox event, worker, topic, WebSocket client, dan REST refetch)]
