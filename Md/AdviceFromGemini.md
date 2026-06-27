# OUTLINE JURNAL SINTA
ABSTRAK
Latar Belakang: Proyek MRT Jakarta Fase 2Aberbasis ODA Loan Jepang sempat terhambat red tape birokrasi, namun berhasil akselerasi hingga mencapai ~60% pada tahun 2026.
Masalah: Risiko red tape baru mengintai pada sisa 40% proyek di area kritis (Harmoni-Kota Tua) akibat tantangan integrasi sistem rel/sinyal (CP205) dan pemindahan utilitas sempit.
Metode: Kolaborasi lintas disiplin melalui pendekatan Kualitatif-Deskriptif (Hubungan Internasional) dan Pemodelan Sistem Alur Kerja (Informatika).
Solusi & Hasil: Mengusulkan arsitektur sistem berbasis Finite State Machine (FSM) dan SLA Tracking untuk memotong asimetri informasi dan menjaga kepercayaan bilateral (International Trust).
I. PENDAHULUAN
1.1 Latar Belakang: MRT Fase 2A sebagai simbol diplomasi ekonomi RI-Jepang. Mengapa rute Bundaran HI-Kota jauh lebih kompleks secara birokrasi dan ruang fisik dibanding Fase 1.
1.2 Identifikasi Masalah: Rekam jejak red tape masa lalu (gagal tender CP205 berkali-kali) dan tantangan masa kini (2026) di area sempit Harmoni-Kota Tua yang melibatkan koordinasi berlapis antar-instansi lokal.
1.3 Dampak Bilateral (Sisi HI): Bagaimana kelambatan administrasi di lapisan bawah berdampak pada evaluasi JICA dan kredibilitas investasi Indonesia.
1.4 Batasan Solusi (Sisi IT): Kelemahan koordinasi manual saat ini (rapat fisik/surat dinas berkelanjutan) yang memicu masalah "saling tunggu".
1.5 Tujuan Penelitian: Merumuskan integrasi kebijakan luar negeri dan sistem informasi sebagai model mitigasi risiko birokrasi.
II. TINJAUAN PUSTAKA
 
III. METODE PENELITIAN
3.1 Desain Riset: Studi kasus deskriptif-analitis dengan pendekatan interdisipliner (HI-IT).
3.2 Sumber Data: Laporan progres resmi PT MRT Jakarta per 2026, dokumen evaluasi JICA, dan rilis resmi kontrak CP205 Sojitz Corporation.
3.3 Metode Analisis: Pemetaan alur birokrasi (kondisi berbelit saat ini) disusul dengan perancangan model sistem (solusi digital yang diusulkan).
IV. HASIL DAN PEMBAHASAN
4.1 Analisis Konflik Kepercayaan & Resolusi CP205 (Sisi HI)
Membedah bagaimana penandatanganan CP205 dengan Sojitz Corp perbaikan hubungan RI-Jepang.
Analisis dinamika sisa 40% proyek di kawasan cagar budaya dan utilitas padat yang memerlukan "gaya diplomasi baru" berbasis data riil.
4.2 Pemodelan Sistem Mitigasi "Anti-Red Tape" (Sisi IT)
Use Case 1 (Pembebasan Lahan/Serah Terima Lokasi): Implementasi Finite State Machine (FSM). Menjelaskan matriks perpindahan status secara transparansi bertahap. 
Use Case 2 (Akselerasi Integrasi Sipil-Sistem): Penerapan SLA Tracking (Alarm Birokrasi). Pada dashboard pemantauan JICA dan Pemprov.
4.3 Dampak Penerapan Sistem Terhadap Hubungan Bilateral
Analisis efek (merasa diawasi): Bagaimana transparansi sistem digital "memaksa" instansi lokal bekerja tepat waktu karena datanya dipantau langsung secara internasional oleh pemberi dana (JICA).
V. KESIMPULAN DAN SARAN
5.1 Kesimpulan: Akselerasi MRT Fase 2A di tahun 2026 membuktikan pentingnya kepastian regulasi. Namun, untuk sisa proyek, red tape tidak bisa lagi diselesaikan dengan rapat birokratis biasa, melainkan harus dipangkas lewat sistem penguncian digital (FSM & SLA).
5.2 Saran/Rekomendasi: PT MRT Jakarta dan Pemprov DKI disarankan membangun dashboard pemantauan terintegrasi yang dapat diakses langsung oleh konsultan asing sebagai wujud Digital Diplomacy yang transparan.
 
NOTE: Diatas outline dari teman saya yang membuat jurnal fokus "Hubungan Internasional"

### 1. Rekomendasi Judul (Sudut Pandang Informatika)

Judul harus mencerminkan kontribusi teknis. Contoh:

* *“Pemodelan Alur Kerja Birokrasi Proyek Infrastruktur Menggunakan Finite State Machine dan Tracking SLA Berbasis Web”*
* *“Arsitektur Sistem Monitoring Transparansi Proyek Transnasional Menggunakan Pendekatan Finite State Machine (FSM)”*

---

### 2. Fokus Utama yang Harus Diperdalam (Niche Informatika)

#### A. Formalisasi Pemodelan FSM (Finite State Machine)

Jangan hanya menyebut FSM secara superfisial. Anda harus menjabarkannya secara matematis dan struktural:

* **Definisikan Komponen FSM:** Tentukan secara jelas set statenya ($S$), input transisinya ($I$), dan fungsi transisinya ($f$).
* *Contoh State:* `Draft_Pengajuan`, `Review_Internal_MRT`, `Verifikasi_Pemprov`, `Approval_JICA`, `SLA_Delayed`, `Final_Handover`.


* **Matriks Transisi State:** Buat tabel atau diagram transisi yang menunjukkan bagaimana sebuah dokumen/izin berpindah dari satu instansi ke instansi lain, serta kondisi apa yang memicu perpindahan tersebut (misal: input `approve_by_pemprov` memindahkan state ke `Wait_JICA_Review`).
* **Pencegahan Deadlock:** Fokuskan bagaimana FSM ini dirancang untuk mencegah terjadinya *deadlock* (kondisi saling tunggu antar-instansi yang membuat sistem macet).

#### B. Arsitektur Integrasi Sistem & Multi-Tenancy

Proyek MRT melibatkan banyak pihak eksternal (PT MRTJ, Pemprov DKI, JICA, Kontraktor Sojitz). Fokuskan pada:

* **Manajemen Hak Akses (RBAC - Role-Based Access Control):** Bagaimana sistem memisahkan hak akses data yang sensitif antar-instansi tanpa mengorbankan transparansi.
* **Integrasi Data (API Gateway):** Bagaimana sistem ini dapat bertukar data dengan sistem internal yang sudah dimiliki oleh Pemprov DKI atau JICA secara aman (Secure API).

#### C. Algoritma SLA Tracking (Engine Alarm)

Jelaskan logika komputasi di balik dashboard monitoring:

* **Mekanisme Pemicu (Trigger Mechanism):** Bagaimana sistem menghitung waktu mundur (countdown) sejak suatu state aktif. Apakah menggunakan *cron jobs*, *event-driven architecture*, atau antrean pesan (*message queue*).
* **Logika Eskalasi:** Jika SLA di suatu state (misal: verifikasi utilitas di Harmoni) melewati batas 7x24 jam, bagaimana algoritma sistem memindahkan status ke `SLA_Violated` dan otomatis mengirimkan notifikasi/alert ke level manajemen yang lebih tinggi (Pemprov & JICA).

#### D. Metode Pengujian dan Evaluasi (Wajib untuk Jurnal IT)

Jurnal Informatika membutuhkan validasi bahwa model yang Anda buat benar-benar berjalan. Anda bisa menggunakan:

* **Black-Box / White-Box Testing:** Menguji semua jalur transisi state pada FSM untuk memastikan tidak ada *state* hantu (unreachable state).
* **Simulasi Beban Kerja (Simulation/Prototyping):** Jika sistem belum diimplementasikan nyata, buatlah prototipe fungsional (misal menggunakan framework backend tertentu) lalu simulasikan skenario birokrasi MRT Fase 2A menggunakan data sekunder untuk menghitung efisiensi waktu pemrosesan dibandingkan dengan cara manual.

---

### 3. Penyesuaian Struktur Outline untuk Jurnal Anda

Mari kita ubah struktur outline yang diberikan teman Anda agar sepenuhnya berfokus pada sisi teknis Informatika:

* **I. PENDAHULUAN**
* **1.1 Latar Belakang:** Kompleksitas koordinasi data pada proyek MRT Fase 2A (Harmoni-Kota Tua). Fokus pada masalah *asimetri informasi* dan lambatnya pelaporan manual.
* **1.2 Rumusan Masalah:** Bagaimana merancang model sistem informasi yang dapat melacak status birokrasi secara *real-time* dan otomatis mendeteksi keterlambatan antar-instansi.
* **1.3 Batasan Masalah:** Sistem berfokus pada pemodelan alur kerja persetujuan dokumen/lahan menggunakan FSM dan pelacakan SLA, tidak mencakup sistem keuangan proyek.


* **II. TINJAUAN PUSTAKA**
* Kajian teori tentang *Finite State Machine* (FSM) dalam pemodelan proses bisnis, sistem pelacakan SLA, dan arsitektur sistem informasi berbasis web/cloud untuk multi-stakeholder.


* **III. METODE PENELITIAN**
* **3.1 Analisis Kebutuhan Sistem:** Mengidentifikasi aktor (MRTJ, Pemprov, JICA) dan kebutuhan fungsional sistem.
* **3.2 Perancangan Model (FSM):** Menyusun diagram status perpindahan dokumen/proses kerja.
* **3.3 Perancangan Arsitektur:** Perancangan database (ERD) dan arsitektur alur data (Data Flow Diagram / Use Case Diagram).


* **IV. HASIL DAN PEMBAHASAN**
* **4.1 Implementasi Model FSM pada Alur Kerja:** Menampilkan visualisasi grafis FSM dan matriks transisinya untuk kasus serah terima lokasi proyek.
* **4.2 Rancangan Antarmuka & Dashboard SLA:** Menjelaskan bagaimana dashboard menampilkan grafik performa SLA tiap instansi dan bagaimana sistem *alerting* bekerja secara teknis.
* **4.3 Evaluasi Verifikasi Model:** Hasil pengujian logika FSM untuk memastikan seluruh fungsionalitas sistem bebas dari *bug* logika alur kerja.


* **V. KESIMPULAN**
* Kesimpulan mengenai keberhasilan pemodelan FSM dan tracking SLA dalam memberikan transparansi status proyek dan efisiensi pelacakan birokrasi secara digital.



### Kesimpulan Langkah Selanjutnya

Hal ini sangat aman dilakukan tanpa dianggap melakukan plagiasi satu sama lain (karena *novelty* atau keterbaruan yang ditawarkan berbeda: teman Anda menawarkan solusi dari perspektif manajemen/kebijakan internasional, sedangkan Anda menawarkan solusi konkret berupa blueprint arsitektur perangkat lunaknya). Anda bisa mulai dengan menyusun diagram FSM berdasarkan alur birokrasi nyata yang ditemukan dari data proyek MRT Fase 2A tersebut.