# 4.5 Implementasi Event-Driven Realtime Delivery

Realtime delivery diimplementasikan menggunakan kombinasi domain event, outbox pattern, Redis Pub/Sub, dan WebSocket. Ketika aksi penting terjadi, seperti progress submitted, progress validated, ticket created, ticket resolved, atau payment approved, service menyimpan event ke outbox. Event tersebut kemudian diproses oleh worker dan dipetakan menjadi envelope realtime.

WebSocket digunakan untuk mengirim event kepada client yang melakukan subscribe pada topic tertentu. Topic disusun berdasarkan konteks resource, seperti project monitoring, project ticket, project finance, ticket, payment, atau organization. Sebelum subscription diterima, sistem melakukan topic authorization agar client hanya menerima event yang sesuai dengan hak aksesnya.

Event realtime tidak digunakan sebagai sumber data utama. Payload WebSocket berfungsi sebagai sinyal bahwa suatu data berubah. Setelah menerima event, frontend melakukan refetch data melalui REST API. Dengan pendekatan ini, sistem memperoleh responsivitas realtime, tetapi konsistensi data tetap dijaga oleh PostgreSQL dan REST API.

[Gambar 4.4 - Implementasi Outbox dan WebSocket Realtime Delivery (berisi domain action, outbox event, worker, Redis Pub/Sub, WebSocket topic, dan frontend refetch)]

[NOTE: Gunakan istilah "realtime delivery signal" atau "near-real-time" jika ingin menghindari klaim bahwa event selalu diterima tanpa delay.]
