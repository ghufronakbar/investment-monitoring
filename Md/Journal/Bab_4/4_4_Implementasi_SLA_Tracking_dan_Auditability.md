# 4.4 Implementasi SLA Tracking dan Auditability

SLA tracking diterapkan pada ticket untuk mengukur durasi penyelesaian hambatan. Setiap ticket dapat memiliki SLA record yang menyimpan entity, state, owner organization, waktu masuk state, due time, resolved time, status SLA, dan remaining time. Informasi ini membuat sistem dapat menampilkan apakah suatu ticket masih dalam batas waktu, mendekati batas waktu, atau telah melanggar SLA.

Auditability diterapkan melalui audit log untuk mencatat aktivitas penting. Audit log menyimpan informasi actor, organisasi actor, action, entity, request id, IP address, user agent, dan waktu kejadian. Dengan catatan tersebut, perubahan penting pada sistem dapat ditelusuri kembali.

SLA tracking dan audit log saling melengkapi. SLA tracking menunjukkan proses mana yang berpotensi menjadi bottleneck, sedangkan audit log menunjukkan siapa yang melakukan aksi pada proses tersebut. Kombinasi ini mendukung accountability pada proyek lintas lembaga karena sistem tidak hanya menampilkan status terakhir, tetapi juga menyediakan konteks waktu dan aktor.

[Gambar 4.3 - Implementasi SLA Tracking dan Auditability (berisi ticket state, SLA record, status SLA, actor action, dan audit log)]

[NOTE: Jika ingin memasukkan contoh response, gunakan field ticket `sla` seperti `dueAt`, `status`, `remainingSeconds`, dan `isViolated`.]
