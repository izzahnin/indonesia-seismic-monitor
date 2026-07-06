# Frontend — Indonesia Seismic Monitor

Next.js 16 App Router dashboard untuk visualisasi data gempa bumi Indonesia.

## Tech Stack

- **Framework:** Next.js 16 (App Router) + TypeScript
- **Styling:** Tailwind CSS v4 (CSS-based config, tidak ada tailwind.config.ts)
- **Peta:** react-leaflet + leaflet (dynamic import wajib — SSR disabled)
- **Charts:** Recharts
- **Data fetching:** TanStack Query v5 — auto-refetch tiap 2 menit
- **Testing:** Vitest + React Testing Library

## Env Vars

Salin `.env.local.example` ke `.env.local`:
```
NEXT_PUBLIC_API_URL=http://localhost:9090
```

## Run & Test

```bash
npm run dev      # dev server (localhost:3000)
npm run test     # Vitest
npm run build    # production build
npm run lint     # ESLint
```

## Struktur

```
app/
  layout.tsx               root layout — QueryClientProvider + Leaflet CSS
  page.tsx                 halaman utama dashboard (single page)
  globals.css              Tailwind v4 directives + CSS variables dark theme
components/
  Map/EarthquakeMap.tsx    react-leaflet, warna marker by magnitude
  Stats/StatCard.tsx       big-number cards (sidebar)
  Table/ProvinceRankTable.tsx  ranking provinsi by risk_score dengan progress bar
  Chart/ProvinceBarChart.tsx   Recharts horizontal bar chart
hooks/
  useEarthquakeData.ts     TanStack Query, refetchInterval 120000ms
lib/
  types.ts                 TypeScript interface mirror dari Go structs
__tests__/                 Vitest test files
```

## Catatan Tailwind v4

- Tidak ada `tailwind.config.ts` — konfigurasi lewat CSS di `globals.css`
- Dark mode: gunakan CSS variables, bukan class `dark:`
- Custom colors didefinisikan via `@theme` block di globals.css

## Catatan Leaflet

- WAJIB dynamic import via `next/dynamic` dengan `ssr: false`
- Import CSS Leaflet di `app/layout.tsx`
- Leaflet marker icons butuh fix manual di Next.js (icon path issue)
