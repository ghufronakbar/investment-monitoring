# 3.7 Perancangan Realtime Event Delivery

Realtime event delivery dirancang untuk mengirim sinyal perubahan data kepada client. Alur dimulai dari aksi domain, seperti progress divalidasi, ticket dibuat, atau pembayaran disetujui. Setelah transaksi utama berhasil, sistem menyimpan event ke outbox. Worker kemudian membaca outbox dan mengirim event ke topic yang sesuai.

WebSocket digunakan sebagai channel delivery event. Client dapat subscribe ke topic tertentu, misalnya topic project monitoring, ticket, atau finance. Ketika event diterima, client tidak langsung menjadikan payload WebSocket sebagai data final, tetapi melakukan refetch melalui REST API. Dengan pendekatan ini, sistem tetap responsif tanpa mengorbankan konsistensi data.

Perancangan ini juga mencakup topic authorization. Client hanya boleh subscribe ke topic yang sesuai dengan aksesnya. Jika topic tidak diizinkan, sistem menolak subscription tanpa harus memutus koneksi WebSocket.

[Gambar 3.6 - Alur Realtime Event Delivery (berisi domain action, outbox event, worker, topic authorization, WebSocket, dan REST refetch)]

[NOTE: Gunakan istilah realtime delivery signal atau near-real-time jika reviewer meminta kehati-hatian terkait klaim realtime.]
