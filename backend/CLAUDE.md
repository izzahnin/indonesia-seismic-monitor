# Backend — Disaster Risk Intelligence

Go service yang menyajikan data gempa BMKG + USGS dengan risk scoring per provinsi.

## Tech Stack

- **Framework:** Fiber v2
- **Module:** `github.com/izzahnin/disaster-risk-intelligence-backend`

## Struktur Paket

```
cmd/server/main.go          entrypoint — init Fiber, routes
internal/
  model/types.go            semua struct: Earthquake, ProvinceSummary, DashboardResponse, dll
  mapper/province.go        MapToProvince(lat, lng) — bounding box 38 provinsi (hardcoded, sumber Nominatim)
  fetcher/bmkg.go           FetchBMKG() — parse XML BMKG, koordinat "lat,long"
  fetcher/usgs.go           FetchUSGS() — parse GeoJSON USGS, koordinat "long,lat,depth"
  scorer/risk.go            Calculate() — group by province, min-max normalization, risk_score
  cache/redis.go            Client wrapper — tidak dipakai di production (no-op)
  handler/earthquakes.go    GET /api/earthquakes, GET /api/health
```

## Env Vars

Salin `.env.example` ke `.env`:
```
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
- `GET /api/earthquakes` → `DashboardResponse` JSON

## Catatan Penting

- Koordinat BMKG: `lat,long` — USGS: `long,lat,depth` — normalisasi dilakukan di masing-masing fetcher
- Partial degradation: kalau BMKG atau USGS gagal, tetap return data dari sumber yang berhasil
- Province mapping tidak butuh database — 38 bbox hardcoded di `internal/mapper/province.go`
