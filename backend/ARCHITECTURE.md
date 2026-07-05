# Backend Architecture — Urutan Baca & Alur Kerja

Dokumen ini menjelaskan file-file backend dalam **urutan dependency** — dari layer paling fundamental hingga entrypoint. Membaca dalam urutan ini memberi gambaran lengkap tentang bagaimana data mengalir dari sumber eksternal (BMKG/USGS) hingga response JSON ke client.

---

## Urutan Baca (Bottom-Up)

```
Lapisan 1 — Fondasi Data
└── internal/model/types.go

Lapisan 2 — Pure Logic (tidak ada dependency internal)
├── internal/mapper/province.go
└── internal/cache/redis.go

Lapisan 3 — Transformasi Data (bergantung pada model + mapper)
├── internal/scorer/risk.go
├── internal/fetcher/bmkg.go
└── internal/fetcher/usgs.go

Lapisan 4 — Orkestrasi
└── internal/handler/earthquakes.go

Lapisan 5 — Entrypoint
├── cmd/server/main.go   ← HTTP server
└── cmd/seed/main.go     ← one-shot tool: referensi historis, tidak dipakai di production
```

---

## Detail Per File

### 1. `internal/model/types.go`
**Peran:** Definisi semua struct yang dipakai di seluruh codebase. Tidak ada logika — hanya kontrak data.

**Konseptual:** File ini adalah "bahasa bersama" seluruh sistem. Setiap layer (fetcher, scorer, handler) berkomunikasi menggunakan tipe yang didefinisikan di sini. Dibaca pertama karena semua file lain bergantung padanya.

**Struktural — hierarki struct:**
```
DashboardResponse          ← root response ke client
├── LiveFeed: []Earthquake ← data BMKG (live, 15 terbaru)
├── HistoricalSummary
│   └── ByProvince: []ProvinceSummary ← data USGS setelah di-score
└── Stats
    ├── LatestEventMinutesAgo  ← dari BMKG
    ├── AvgMagnitudeLast15     ← dari BMKG
    └── StrongestLast30Days: *StrongestEvent ← pointer; null jika tidak ada gempa 30 hari
```

Field penting di `Earthquake`:
- `Source` — "bmkg" atau "usgs", membedakan asal data meski struct sama
- `Province` — hasil `MapToProvince()`, bisa "Wilayah Lain" jika di luar semua bbox
- `TsunamiPotential` — hasil normalisasi dari format berbeda (string BMKG vs integer USGS)

---

### 2. `internal/mapper/province.go`
**Peran:** Mengkonversi koordinat `(lat, lng)` → nama provinsi Indonesia menggunakan bounding box.

**Konseptual:** Mapper adalah jembatan antara "koordinat geografis" dan "nama wilayah administratif". Pendekatan bounding box dipilih karena sederhana dan cepat — tidak butuh library GIS atau database. Tradeoffnya: area di perbatasan dua provinsi bisa salah klasifikasi.

Data 38 provinsi di-hardcode langsung di file ini — tidak butuh database. Koordinat bersumber dari Nominatim (via `cmd/seed`) dengan koreksi manual untuk provinsi pemekaran 2022 yang belum tersedia di Nominatim.

**Struktural:**
```
Setiap gempa masuk
    │
    └─ MapToProvince(lat, lng)
           │
           └─ loop provinceBoxes (38 provinsi, hardcoded):
                  cek MinLat ≤ lat ≤ MaxLat && MinLng ≤ lng ≤ MaxLng
                  ├─ cocok → return nama provinsi
                  └─ tidak ada yang cocok → return "Wilayah Lain"
```

Test: `internal/mapper/province_test.go`

---

### 3. `internal/cache/redis.go`
**Peran:** Wrapper tipis di atas `go-redis` dengan pola no-op saat Redis tidak tersedia.

**Konseptual:** Caching diperlukan karena fetch ke BMKG dan USGS memakan waktu (~1-3 detik) dan data gempa tidak berubah setiap detik. Dengan TTL 10 menit, ratusan request dalam satu menit hanya menghasilkan satu fetch ke sumber eksternal.

**Struktural — dua mode:**
```
NewClient("")           → Client{rdb: nil}  (no-op mode)
NewClient("redis://...") → Client{rdb: *redis.Client} (connected mode)

Get(key):
  rdb == nil → return ("", redis.Nil)  ← selalu cache miss
  rdb != nil → return dari Redis

Set(key, value, ttl):
  rdb == nil → return nil  ← diabaikan
  rdb != nil → simpan ke Redis dengan TTL
```

Pola ini memungkinkan development lokal tanpa Redis — app berjalan normal, hanya tanpa caching.

---

### 4. `internal/scorer/risk.go`
**Peran:** Menghitung `risk_score` per provinsi dari slice `[]Earthquake` menggunakan min-max normalization.

**Konseptual:** Risk score bersifat **relatif**, bukan absolut. Angka 80 bukan berarti "80% berbahaya" — melainkan "lebih berisiko dari sebagian besar provinsi dalam dataset ini". Ini penting dipahami saat membaca angka di dashboard.

**Struktural — alur kalkulasi:**
```
Calculate([]Earthquake)
    │
    ├─ Group by Province → map[string]*agg{count, sumMag, maxMag}
    │     └─ SKIP "Wilayah Lain" — bukan provinsi nyata, koordinat di luar bbox manapun.
    │        Tanpa skip ini, "Wilayah Lain" sering muncul #1 karena volume gempa laut tinggi.
    │
    ├─ Konversi ke []ProvinceSummary (hitung avgMag = sumMag/count)
    │
    ├─ minMaxNormalize(summaries):
    │       ├─ Cari min/max untuk count dan avgMag dari semua provinsi
    │       ├─ normalize(val) = (val - min) / (max - min) * 100
    │       └─ risk_score = normalize(count)*0.5 + normalize(avgMag)*0.5
    │
    └─ sort descending by risk_score → return []ProvinceSummary
```

Edge case: 1 provinsi → risk_score=100 (tidak ada pembanding). Semua nilai sama → normalize() return 100.

Test: `internal/scorer/risk_test.go`

---

### 5. `internal/fetcher/bmkg.go`
**Peran:** Fetch dan parse data live feed dari BMKG — 15 gempa terbaru yang dimonitor Indonesia.

**Konseptual:** BMKG adalah badan resmi Indonesia yang memonitor aktivitas seismik di wilayah Indonesia dan sekitarnya. API-nya sederhana — tidak butuh autentikasi, selalu mengembalikan snapshot 15 gempa paling baru saat itu. Karena "terkini", data ini dipakai sebagai live feed, bukan untuk analisis historis.

**Struktural — alur kode:**
```
FetchBMKG(ctx)
  │
  ├─ HTTP GET bmkgURL (timeout 5s)
  │    └─ response: XML dengan elemen <gempa> berulang
  │
  ├─ xml.Unmarshal → bmkgXML{Gempa: []bmkgGempa}
  │
  └─ loop tiap bmkgGempa:
       ├─ parseCoordsBMKG("lat,long") → lat, lng float64
       │     └─ SKIP jika gagal (koordinat rusak)
       ├─ time.Parse(RFC3339, DateTime) → time.Time
       │     └─ SKIP jika gagal — time zero (1 Jan 0001) tidak bermakna di dashboard
       ├─ strconv.ParseFloat(Magnitude) → float64
       │     └─ SKIP jika gagal — magnitude 0 akan merusak stats dan risk scoring
       ├─ parseDepth("545 km") → 545.0  (zero value aman jika gagal)
       ├─ mapper.MapToProvince(lat, lng) → nama provinsi / "Wilayah Lain"
       └─ append ke []model.Earthquake
```

**Format data BMKG yang perlu diperhatikan:**
- Koordinat: string `"lat,long"` — latitude dulu, baru longitude
- Kedalaman: string dengan satuan `"545 km"` — harus di-strip dan di-parse
- Tsunami: string teks bebas `"Berpotensi tsunami"` — dicek dengan `strings.EqualFold` + `strings.TrimSpace` (exact match, bukan contains)
- Waktu: RFC3339 string (`"2024-01-15T06:30:00+07:00"`)
- ID: dibentuk dari `DateTime + Coordinates` (bukan index loop) → stabil lintas fetch.
  Format: `bmkg-2024-01-15T063000+0700--6.99_125.83`. Index-based (`bmkg-0`, `bmkg-1`)
  bermasalah karena bergeser tiap ada gempa baru, membuat frontend tidak bisa track event yang sama.

---

### 6. `internal/fetcher/usgs.go`
**Peran:** Fetch dan parse data historis gempa M ≥ 4.5 dari USGS untuk wilayah Indonesia, 6 bulan terakhir.

**Konseptual:** USGS (United States Geological Survey) mengoperasikan jaringan seismograf global dan menyediakan API publik FDSN yang sangat lengkap. Berbeda dengan BMKG yang hanya 15 data terbaru, USGS memberikan ribuan event historis dengan filter geografis dan magnitudo. Data ini dipakai untuk risk scoring per provinsi — butuh volume besar agar statistik bermakna.

**Struktural — alur kode:**
```
FetchUSGS(ctx)
  │
  ├─ Hitung startTime = hari ini - 6 bulan
  │
  ├─ HTTP GET usgsURL?format=geojson&starttime=...&minmagnitude=4.5
  │           &minlatitude=-11&maxlatitude=6&minlongitude=95&maxlongitude=141
  │    └─ response: GeoJSON FeatureCollection
  │
  ├─ json.Unmarshal → usgsResponse{Features: []usgsFeature}
  │
  └─ loop tiap usgsFeature:
       ├─ Coordinates[0] → lng  ← SWAP! GeoJSON urutannya [lon, lat, depth]
       ├─ Coordinates[1] → lat
       ├─ Coordinates[2] → depth
       ├─ time.UnixMilli(Properties.Time) → time.Time  ← milliseconds, bukan detik
       ├─ mapper.MapToProvince(lat, lng) → nama provinsi / "Wilayah Lain"
       └─ append ke []model.Earthquake
```

**Format data USGS yang perlu diperhatikan:**
- Koordinat: GeoJSON array `[longitude, latitude, depth]` — **longitude dulu**, terbalik dari BMKG dan konvensi peta biasa
- Waktu: Unix **milliseconds** integer (bukan detik) — harus pakai `time.UnixMilli`, bukan `time.Unix`
- Tsunami: integer `0` atau `1` — bukan string, cukup bandingkan dengan `== 1`
- Filter geografis dilakukan di sisi USGS via query params, bukan di kode kita

**Perbandingan BMKG vs USGS:**

| Aspek | BMKG | USGS |
|---|---|---|
| Tujuan | Live feed terkini | Analisis historis |
| Jumlah data | 15 gempa terbaru | Ratusan–ribuan (6 bulan) |
| Format | XML | GeoJSON |
| Urutan koordinat | `lat, lng` | `lng, lat, depth` (terbalik!) |
| Timestamp | RFC3339 string | Unix milliseconds integer |
| Kedalaman | String `"545 km"` | Float langsung di koordinat index 2 |
| Tsunami | String teks bebas | Integer 0/1 |
| Filter geografis | Tidak ada (BMKG pilihkan) | Query param bbox eksplisit |
| Cakupan | Indonesia + sekitar perbatasan | Global, dikropkan ke bbox Indonesia |

---

### 7. `internal/handler/earthquakes.go`
**Peran:** HTTP handler yang mengorkestrasikan semua layer — cache, fetcher, scorer, stats — menjadi satu response.

**Konseptual:** Handler adalah "konduktor orkestra". Ia tidak melakukan logika bisnis sendiri, tapi tahu urutan memanggil komponen lain dan bagaimana menggabungkan hasilnya. Partial degradation adalah keputusan desain penting: lebih baik kirim data sebagian daripada error 500.

**Struktural — alur `GET /api/earthquakes`:**
```
Request masuk
    │
    ├─ cache.Get("earthquakes:combined")
    │     └─ HIT → c.SendString(cached) ──────────────────────────► response (~1ms)
    │
    └─ MISS → errgroup (2 goroutine paralel, timeout 5s)
          │
          ├─ fetcher.FetchBMKG() → bmkgData []Earthquake  (atau partial=true jika gagal)
          └─ fetcher.FetchUSGS() → usgsData []Earthquake  (atau partial=true jika gagal)
          │
          ├─ scorer.Calculate(usgsData) → provinceSummaries
          │
          ├─ buildStats(bmkgData, usgsData, now) → Stats
          │     ├─ live data: cari gempa terbaru, hitung avg mag
          │     └─ historical data: cari gempa terkuat 30 hari terakhir
          │           └─ StrongestLast30Days = nil jika tidak ada → JSON: null (bukan zero value)
          │
          ├─ json.Marshal(DashboardResponse) → []byte
          │
          ├─ cache.Set("earthquakes:combined", json, 10min)
          │
          └─ c.Send(b) ────────────────────────────────────────────► response
```

**Dua handler yang terdaftar:**
- `GET /api/health` → `{"status":"ok"}` — cek server hidup
- `GET /api/earthquakes` → `DashboardResponse` — data lengkap dashboard

---

### 8. `cmd/server/main.go`
**Peran:** Entrypoint — membaca konfigurasi dari environment, inisialisasi semua komponen, start HTTP server.

**Konseptual:** main() adalah "bootstrap" — satu-satunya tempat di mana semua komponen dirakit bersama. Komponen lain tidak saling tahu satu sama lain; mereka hanya tahu interface yang dibutuhkan. Ini disebut dependency injection manual (tanpa framework DI).

**Struktural — urutan inisialisasi:**
```
main()
    │
    ├─ godotenv.Load()            ← baca .env (diabaikan jika tidak ada)
    ├─ os.Getenv("PORT")          ← default "9090" jika kosong
    │
    ├─ fiber.New()                ← buat HTTP server
    ├─ cors.New(AllowOrigins:"*") ← izinkan request dari frontend
    │
    ├─ cache.NewClient(REDIS_URL) ← no-op jika REDIS_URL kosong
    ├─ handler.New(cacheClient)
    ├─ h.Register(app)            ← daftarkan /api/health dan /api/earthquakes
    │
    └─ app.Listen(":PORT")        ← mulai terima request (blocking)
```

Tidak ada koneksi database di server — province mapping sepenuhnya hardcoded di `internal/mapper`.
Satu-satunya env var yang dipakai: `PORT` (default 9090).

---

## Alur Data Lengkap (Request → Response)

```
Client
  │
  │ GET /api/earthquakes
  ▼
handler.earthquakes()
  │
  ├─ cache.Get("earthquakes:combined")
  │     └─ HIT → return JSON ──────────────────────────────► Client
  │
  └─ MISS → errgroup.Go x2 (paralel, timeout 5s)
        │
        ├─ fetcher.FetchBMKG()
        │     └─ BMKG XML → parse → mapper.MapToProvince() → []Earthquake
        │
        └─ fetcher.FetchUSGS()
              └─ USGS GeoJSON → parse → mapper.MapToProvince() → []Earthquake
        │
        ▼
   scorer.Calculate(usgsData) → []ProvinceSummary (risk_score, sorted)
        │
        ▼
   buildStats(bmkgData, usgsData) → Stats
        │
        ▼
   DashboardResponse{LiveFeed, HistoricalSummary, Stats, CachedAt}
        │
        ├─ cache.Set("earthquakes:combined", json, 10min)
        │
        └──────────────────────────────────────────────────► Client
```
