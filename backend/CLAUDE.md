# Backend — Disaster Risk Intelligence

Go service yang menyajikan data gempa BMKG + USGS, dengan Redis cache dan risk scoring per provinsi.

## Tech Stack

- **Framework:** Fiber v2
- **Cache:** Redis via `go-redis/v9` (Upstash free tier di prod)
- **Module:** `github.com/izzahnin/disaster-risk-intelligence-backend`

## Struktur Paket

```
cmd/server/main.go          entrypoint — init Fiber, Redis, routes
internal/
  model/types.go            semua struct: Earthquake, ProvinceSummary, DashboardResponse, dll
  mapper/province.go        MapToProvince(lat, lng) — bounding box 15 provinsi rawan gempa
  fetcher/bmkg.go           FetchBMKG() — parse XML BMKG, koordinat "lat,long"
  fetcher/usgs.go           FetchUSGS() — parse GeoJSON USGS, koordinat "long,lat,depth"
  scorer/risk.go            Calculate() — group by province, min-max normalization, risk_score
  cache/redis.go            Client wrapper — Get/Set, no-op kalau REDIS_URL kosong
  handler/earthquakes.go    GET /api/earthquakes, GET /api/health
```

## Env Vars

Salin `.env.example` ke `.env`:
```
REDIS_URL=redis://...   # kosongkan untuk no-op cache (dev tanpa Redis)
PORT=9090
```

## Run & Test

```bash
go run ./cmd/server          # dev server
go test ./...                # semua unit test
go build ./...               # verifikasi kompilasi
```

## Endpoints

- `GET /api/health` → `{"status":"ok"}`
- `GET /api/earthquakes` → `DashboardResponse` JSON (cache TTL 10 menit)

## Catatan Penting

- Koordinat BMKG: `lat,long` — USGS: `long,lat,depth` — normalisasi dilakukan di masing-masing fetcher
- Partial degradation: kalau BMKG atau USGS gagal, tetap return data dari sumber yang berhasil
- Redis down = fallback fetch on-demand tanpa cache, app tidak crash
