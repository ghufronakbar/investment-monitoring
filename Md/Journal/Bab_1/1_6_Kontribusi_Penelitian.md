# 1.6 Kontribusi Penelitian

Penelitian ini memberikan kontribusi pada perancangan backend untuk sistem monitoring proyek infrastruktur lintas lembaga.

Kontribusi pertama adalah model workflow governance berbasis Finite State Machine. Model ini digunakan untuk memastikan perubahan status pada progress monitoring, ticket hambatan, dan approval pembayaran berjalan melalui transisi yang valid.

Kontribusi kedua adalah penerapan organization-centric access control. Pendekatan ini membatasi kewenangan berdasarkan relasi antara user, organisasi, role, project, dan contract, sehingga lebih sesuai untuk sistem multi-stakeholder.

Kontribusi ketiga adalah integrasi SLA tracking dan audit trail. SLA tracking digunakan untuk mengukur potensi bottleneck, sedangkan audit trail digunakan untuk meningkatkan traceability terhadap aksi penting.

Kontribusi keempat adalah realtime event delivery berbasis outbox pattern dan WebSocket. Mekanisme ini memungkinkan client menerima sinyal perubahan data secara responsif tanpa menjadikan WebSocket sebagai sumber kebenaran data.

Dengan kontribusi tersebut, penelitian ini diharapkan dapat menjadi referensi pengembangan backend workflow governance untuk proyek infrastruktur yang membutuhkan transparansi, accountability, dan koordinasi lintas organisasi.
