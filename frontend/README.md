# Indonesia Seismic Monitor — Frontend

Dashboard visualisasi aktivitas seismik Indonesia. Menampilkan peta interaktif, ranking seismisitas per provinsi, dan statistik terkini dari data BMKG + USGS.

## Tech Stack

| Layer | Teknologi |
|---|---|
| Framework | Next.js 16 (App Router) + TypeScript |
| Styling | Tailwind CSS v4 (CSS-based config) |
| Tema | next-themes (dark / light / system) |
| Peta | react-leaflet + leaflet |
| Data Fetching | TanStack Query v5 (auto-refresh 2 menit) |
| Testing | Vitest + React Testing Library |
| Deploy | Vercel |

## Setup & Run

### Prasyarat
- Node.js 20+
- Backend Go service berjalan di port 9090

### 1. Install dependencies

```bash
npm install
```

### 2. Konfigurasi environment

```bash
cp .env.local.example .env.local
# Isi NEXT_PUBLIC_API_URL dengan URL backend
```

| Variable | Default | Keterangan |
|---|---|---|
| `NEXT_PUBLIC_API_URL` | `http://localhost:9090` | URL backend Go service |

### 3. Jalankan dev server

```bash
npm run dev
# Buka http://localhost:3000
```

## Scripts

```bash
npm run dev        # development server
npm run build      # production build
npm run start      # jalankan production build
npm run test       # Vitest
npm run lint       # ESLint
```

## Fitur Dashboard

- **Peta Sebaran Gempa** — 15 titik gempa terbaru BMKG, warna berdasarkan magnitudo, popup detail, tile switch CartoDB dark/light otomatis, zoom dibatasi ke wilayah Indonesia
- **Stat Cards** — 3 angka ringkas: menit sejak gempa terakhir, rata-rata magnitudo 15 gempa terkini, gempa terkuat 30 hari terakhir
- **Gempa Terkini** — daftar 15 gempa terbaru dari BMKG dengan waktu WIB, kedalaman, dan indikator potensi tsunami
- **Tabel Ranking** — 10 provinsi dengan aktivitas seismik tertinggi berdasarkan data historis USGS 6 bulan; scroll horizontal di mobile
- **Dark / Light Mode** — tema mengikuti preferensi sistem atau diubah manual via toggle
- **Auto-refresh** — data diperbarui otomatis tiap 2 menit tanpa reload halaman
- **Partial data banner** — notifikasi jika salah satu sumber data tidak tersedia
- **Loading & 404** — halaman skeleton dan not-found bertema seismik

## Struktur Folder

```
app/
  layout.tsx              root layout + ThemeProvider + QueryClientProvider
  page.tsx                halaman dashboard utama
  loading.tsx             skeleton saat initial load (Next.js convention)
  not-found.tsx           halaman 404 bertema seismik (Next.js convention)
  globals.css             design tokens (light/dark), animasi seismograf
components/
  Map/EarthquakeMap       peta Leaflet, dynamic import (SSR disabled), tile dark/light
  Stats/StatCard          big-number card
  Table/ProvinceRankTable tabel ranking dengan progress bar indeks seismisitas, scroll mobile
  ThemeToggle             tombol dark/light mode
hooks/
  useEarthquakeData.ts    TanStack Query hook — fetch + cache
lib/
  types.ts                TypeScript interfaces (mirror Go structs)
```

## Deploy ke Vercel

1. Push repo ke GitHub
2. Hubungkan di [vercel.com](https://vercel.com)
3. Set environment variable: `NEXT_PUBLIC_API_URL=https://<url-backend>`
4. Deploy otomatis dari setiap push ke `main`
