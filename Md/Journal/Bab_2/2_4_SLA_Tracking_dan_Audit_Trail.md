# 2.4 SLA Tracking dan Audit Trail

Service Level Agreement (SLA) tracking adalah mekanisme untuk mengukur apakah suatu proses berjalan dalam batas waktu yang telah ditentukan. Dalam workflow proyek, SLA tidak hanya dipahami sebagai deadline akhir proyek, tetapi juga sebagai batas waktu pada state tertentu. Misalnya, ticket hambatan dapat memiliki waktu maksimal untuk ditangani sejak status `OPEN` atau `ASSIGNED`.

SLA tracking umumnya membutuhkan informasi seperti entity yang dipantau, state aktif, organisasi pemilik tanggung jawab, waktu mulai state, batas waktu penyelesaian, dan waktu penyelesaian. Dari data tersebut, sistem dapat menghitung apakah suatu proses masih normal, mendekati batas waktu, atau sudah melewati SLA. Informasi ini membantu mendeteksi bottleneck secara lebih objektif.

Audit trail adalah catatan aktivitas penting yang terjadi di sistem. Audit trail biasanya menyimpan informasi actor, organisasi actor, action, entity, waktu, dan konteks request. Pada sistem multi-stakeholder, audit trail diperlukan karena perubahan status atau approval tidak hanya berdampak pada satu pihak, tetapi dapat memengaruhi proses koordinasi lintas organisasi.

SLA tracking dan audit trail saling melengkapi. SLA tracking menjawab pertanyaan "proses mana yang terlambat", sedangkan audit trail menjawab pertanyaan "siapa melakukan apa dan kapan". Kombinasi keduanya meningkatkan traceability dan accountability pada sistem monitoring proyek.

[Gambar 2.4 - Hubungan SLA Tracking dan Audit Trail (berisi alur state aktif, due time, SLA status, actor action, dan audit log)]
