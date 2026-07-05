# Disaster Risk Intelligence

Dashboard web real-time untuk memantau risiko gempa bumi di Indonesia. Menggabungkan data live dari BMKG dan data historis dari USGS untuk menghasilkan peta sebaran, ranking risiko per provinsi, dan statistik seismik yang mudah dipahami.

---

## Tampilan

> Dashboard menampilkan 15 gempa terbaru BMKG secara real-time, diperbarui otomatis tiap 2 menit.

---

## Struktur Repo

```
disaster-risk-intelligence/
├── backend/     Go REST API — fetching, scoring, caching
└── frontend/    Next.js dashboard — peta interaktif, tabel, statistik
```

Dokumentasi teknis masing-masing ada di subfolder:
- [`backend/README.md`](./backend/README.md)
- [`frontend/README.md`](./frontend/README.md)

---

## Cara Kerja

```
BMKG (XML)  ──┐
               ├─► Go API ─► Redis Cache ─► Next.js Dashboard
USGS (JSON) ──┘
```

1. **BMKG** menyediakan 15 gempa terbaru secara real-time
2. **USGS** menyediakan data historis 6 bulan terakhir (M ≥ 4.5) untuk wilayah Indonesia
3. **Go backend** menggabungkan keduanya, menghitung risk score per provinsi dengan min-max normalization, dan meng-cache hasilnya di Redis selama 10 menit
4. **Next.js frontend** menampilkan data dalam peta Leaflet, tabel ranking, dan stat cards — auto-refresh tiap 2 menit

---

## Tech Stack

| Bagian | Teknologi |
|---|---|
| Backend | Go, Fiber v2, Redis |
| Frontend | Next.js 16, TypeScript, Tailwind CSS v4, react-leaflet, TanStack Query |
| Data | BMKG (real-time), USGS FDSN (historis) |
| Deploy | Railway/Render (backend), Vercel (frontend) |

---

## Sumber Data & Legalitas

| Sumber | Lisensi | Keterangan |
|---|---|---|
| [BMKG](https://data.bmkg.go.id/) | Publik (data pemerintah Indonesia) | 15 gempa terbaru, real-time |
| [USGS](https://earthquake.usgs.gov/fdsnws/event/1/) | Public Domain (data federal AS) | Historis 6 bulan, M ≥ 4.5 |
| [OpenStreetMap](https://www.openstreetmap.org/) | ODbL | Bounding box provinsi via Nominatim |
| [ESRI World Imagery](https://www.esri.com/) | Gratis non-komersial | Tile layer peta |

---

## Menjalankan Lokal

Diperlukan dua terminal — satu untuk backend, satu untuk frontend.

```bash
# Terminal 1 — Backend
cd backend
cp .env.example .env
go run ./cmd/server

# Terminal 2 — Frontend
cd frontend
cp .env.local.example .env.local
npm install
npm run dev
```

Buka [http://localhost:3000](http://localhost:3000).
Backend berjalan di port `9090` secara default.

---

*Data gempa bersumber dari lembaga resmi pemerintah. Proyek ini dibuat untuk keperluan portofolio dan edukasi.*
