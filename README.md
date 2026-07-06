<div align="center">

# Indonesia Seismic Monitor

**Dashboard pemantauan aktivitas seismik Indonesia**  
Data BMKG · Data Historis USGS · Indeks Seismisitas per Provinsi

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?style=flat-square&logo=go&logoColor=white)](./backend)
[![Next.js](https://img.shields.io/badge/Next.js-16-000000?style=flat-square&logo=nextdotjs&logoColor=white)](./frontend)
[![Vercel](https://img.shields.io/badge/Vercel-deployed-000000?style=flat-square&logo=vercel)](https://vercel.com)
[![Render](https://img.shields.io/badge/Render-deployed-46E3B7?style=flat-square&logo=render&logoColor=white)](https://render.com)

[🌐 Live Demo](https://indonesia-seismic-monitor.vercel.app/) · [📖 Backend Docs](backend/README.md) · [🖥️ Frontend Docs](frontend/README.md) · [⚠️ Known Issues](KNOWN_ISSUES.md)

</div>

---

## Tentang Proyek

Dashboard web full-stack untuk memantau aktivitas seismik di seluruh wilayah Indonesia. Menggabungkan data terkini dari BMKG (diperbarui tiap 2 menit) dan data historis enam bulan dari USGS untuk menghasilkan skor seismisitas berbasis frekuensi dan kekuatan gempa per provinsi.

**Fitur utama:**
- **Peta interaktif** — 15 titik gempa terbaru BMKG dengan warna berdasarkan magnitudo, popup detail, tile switch dark/light mode
- **Live feed sidebar** — daftar gempa dengan waktu WIB, kedalaman, dan indikator potensi tsunami
- **Stat cards** — menit sejak gempa terakhir, rata-rata magnitudo, gempa terkuat 30 hari
- **Ranking seismisitas provinsi** — 10 provinsi dengan aktivitas kegempaan tertinggi berdasarkan data historis USGS 6 bulan, dengan indeks relatif 0–100
- **Dark / Light mode** — mengikuti preferensi sistem atau diubah manual
- **Auto-refresh** — data diperbarui otomatis tiap 2 menit tanpa reload halaman

---

## Screenshots

| Dashboard — Light Mode | Dashboard — Dark Mode |
|:---:|:---:|
| <img width="1863" height="1715" alt="screencapture-localhost-3000-2026-07-06-17_13_35" src="https://github.com/user-attachments/assets/a85f3966-e144-476e-a7a3-e1461fa28852" />| <img width="1863" height="1715" alt="screencapture-localhost-3000-2026-07-06-17_13_24" src="https://github.com/user-attachments/assets/5f38948b-e68f-405c-9d79-a2c4dd1875b2" />|

---

## Arsitektur

```
Browser / Mobile
      │
      ▼
 Vercel (Next.js 16)              ← Dashboard: SSR, dark mode, auto-refresh 2 mnt
      │
      ▼
 Render (Go + Fiber)    ← REST API: public, partial degradation
      │
      ├──► BMKG (XML)             ← 15 gempa terbaru, polling tiap 2 menit
      └──► USGS (GeoJSON)         ← Historis 6 bulan, M ≥ 4.5
```

**Backend — alur data:**
```
HTTP Request
    │
    ▼
Handler (earthquakes.go)
    │
    └── FetchBMKG() + FetchUSGS()
                          │
                          ▼
                     MapToProvince()    ← koordinat → nama provinsi
                          │
                          ▼
                     Calculate()        ← indeks seismisitas per provinsi
                          │
                          ▼
                     return response
```

---

## Tech Stack

### Backend [`→ /backend`](./backend)

| Teknologi | Kegunaan |
|-----------|----------|
| Go 1.22 + Fiber v2 | Web framework — routing, middleware, JSON |
| BMKG XML API | 15 gempa terbaru, polling tiap 2 menit |
| USGS FDSN API | Data historis 6 bulan, M ≥ 4.5, wilayah Indonesia |

### Frontend [`→ /frontend`](./frontend)

| Teknologi | Kegunaan |
|-----------|----------|
| Next.js 16 + TypeScript | App Router, SSR, strict types |
| Tailwind CSS v4 | CSS-based config, design token system via `@theme` |
| react-leaflet + Leaflet | Peta interaktif, CircleMarker, tile CARTO dark/light |
| TanStack Query v5 | Data fetching + cache, auto-refetch 2 menit |
| next-themes | Dark / light mode dengan system preference |

### Infrastructure

| Layanan | Platform |
|---------|----------|
| Frontend | Vercel (auto-deploy dari GitHub) |
| Backend | Render (Docker, auto-deploy dari GitHub) |

---

## Sumber Data & Legalitas

| Sumber | Lisensi | Keterangan |
|--------|---------|------------|
| [BMKG](https://data.bmkg.go.id/) | Publik — data pemerintah Indonesia | 15 gempa terbaru, diperbarui tiap 2 menit |
| [USGS FDSN](https://earthquake.usgs.gov/fdsnws/event/1/) | Public Domain — data federal AS | Historis 6 bulan, M ≥ 4.5 |
| [OpenStreetMap / Nominatim](https://www.openstreetmap.org/) | ODbL | Bounding box 38 provinsi |
| [CARTO](https://carto.com/) | Gratis (non-komersial) | Tile peta dark/light mode |

---

## Stats

| | |
|---|---|
| 2 | REST API Endpoints |
| 38 | Provinsi Indonesia (bounding box hardcoded, sumber Nominatim) |
| 2 | Sumber data (BMKG terkini + USGS historis) |
| 6 bulan | Rentang data historis USGS |
| 2 menit | Interval auto-refresh frontend |

---

## Menjalankan Lokal

```bash
# Terminal 1 — Backend
cd backend
cp .env.example .env        # PORT default 9090, tidak perlu diubah untuk dev lokal
go run ./cmd/server
# API: http://localhost:9090

# Terminal 2 — Frontend
cd frontend
cp .env.local.example .env.local   # NEXT_PUBLIC_API_URL=http://localhost:9090
npm install && npm run dev
# http://localhost:3000
```

Backend tidak butuh database maupun cache eksternal. Province mapping hardcoded langsung di kode.

---

## Kontak

**Nurul Izzah Nurhidayat** · Makassar, Sulawesi Selatan

[![LinkedIn](https://img.shields.io/badge/LinkedIn-Connect-0A66C2?style=flat-square&logo=linkedin)](https://linkedin.com/in/nurul-izzah-nurhidayat-397346289)
[![GitHub](https://img.shields.io/badge/GitHub-Profile-181717?style=flat-square&logo=github)](https://github.com/izzahnin)
