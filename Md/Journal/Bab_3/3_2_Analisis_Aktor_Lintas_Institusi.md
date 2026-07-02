# 3.2 Analisis Aktor Lintas Institusi

Sistem dirancang untuk mendukung proyek yang melibatkan banyak organisasi. Aktor utama yang digunakan dalam model sistem adalah `GOV_SUPERVISOR`, `LOAN_PROVIDER`, `EXECUTING_AGENCY`, `CONTRACTOR`, `INDEPENDENT_AUDITOR`, dan `GOV_PARTNER`. Setiap aktor memiliki kewenangan berbeda terhadap data dan proses proyek.

`GOV_SUPERVISOR` berperan sebagai pengawas proyek, `LOAN_PROVIDER` sebagai pihak pemberi dana, `EXECUTING_AGENCY` sebagai pelaksana utama proyek, `CONTRACTOR` sebagai pelaksana pekerjaan, `INDEPENDENT_AUDITOR` sebagai validator progress, dan `GOV_PARTNER` sebagai instansi pendukung. Role tersebut digunakan untuk menormalisasi peran lintas institusi ke dalam sistem.

Analisis aktor ini menjadi dasar perancangan access control dan workflow. Setiap aksi, seperti submit progress, validasi progress, assignment ticket, approval pembayaran, dan disbursement, harus dikaitkan dengan aktor yang tepat.

[Tabel 3.1 - Pemetaan Aktor dan Kewenangan (berisi aktor, contoh organisasi, dan aksi yang dapat dilakukan)]

[NOTE: Isi kolom "contoh organisasi" sesuai kebutuhan penulisan. Jika ingin aman, gunakan istilah generik seperti pemberi dana asing, pelaksana proyek, kontraktor, auditor, dan instansi pendukung.]
