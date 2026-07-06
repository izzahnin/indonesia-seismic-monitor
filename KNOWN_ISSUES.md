# Known Limitations & Potential Improvements

## Known Limitations

**Province Mapping — Bounding Box**
Pemetaan koordinat gempa ke provinsi menggunakan bounding box (kotak persegi), bukan polygon batas wilayah administratif yang sesungguhnya. Akibatnya:
- Area di perbatasan dua provinsi bisa salah diklasifikasi
- Overlap antar bbox diselesaikan dengan *first-match* berdasarkan urutan data — provinsi yang lebih dulu di list yang "menang"
- Solusi yang lebih akurat membutuhkan data polygon GeoJSON dari GADM atau BIG, beserta library GIS

**Data BMKG — Near Real-Time, Bukan Live**
Data BMKG diambil via polling setiap 2 menit, bukan push/stream langsung dari BMKG. Ada jeda maksimal 2 menit antara kejadian gempa dan tampilannya di dashboard.

**Skor Seismisitas — Bukan Skor Risiko**
Yang ditampilkan adalah **skor seismisitas** (aktivitas kegempaan), bukan skor risiko. Formula R = H × V (Risiko = Bahaya × Kerentanan) yang dipakai lembaga resmi seperti BNPB dan UNDRR juga memperhitungkan kerentanan penduduk, kualitas bangunan, dan kapasitas mitigasi — data yang tidak tersedia secara API publik.

Skor seismisitas dihitung dari dua komponen (bobot 50:50): frekuensi gempa dengan log10(count) dan rata-rata magnitudo pada skala tetap M4.5–8.0. Skor dinormalisasi ke 0–100 relatif antar provinsi dalam dataset dan berubah seiring data USGS 6 bulan bergulir.

**Cakupan Data USGS**
USGS hanya mencatat gempa M≥4.5. Gempa kecil yang sering (M<4.5) — yang bisa relevan untuk beberapa wilayah — tidak masuk dalam perhitungan indeks seismisitas.

---

## Potential Improvements

- **Polygon-based province mapping** — ganti bbox dengan GeoJSON batas provinsi resmi dari BIG/GADM untuk akurasi pemetaan yang jauh lebih baik
- **Kedalaman sebagai faktor risiko** — gempa dangkal (< 70 km) jauh lebih merusak dari gempa dalam; kedalaman bisa dimasukkan ke formula scoring
- **Grafik tren historis** — tampilkan perubahan indeks seismisitas per provinsi dari waktu ke waktu, bukan hanya snapshot 6 bulan
- **Filter & eksplorasi** — filter peta berdasarkan magnitudo, kedalaman, atau rentang waktu
- **Notifikasi gempa besar** — push notification atau alert saat ada gempa M≥6.0 baru di BMKG
- **PWA / mobile app** — tambahkan service worker agar bisa diinstall dan diakses offline
