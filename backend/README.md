# Indonesia Seismic Monitor — Backend

REST API untuk dashboard aktivitas seismik Indonesia. Menggabungkan data terkini BMKG dan data historis USGS, menghasilkan skor seismisitas per provinsi.

## Tech Stack

| Layer | Teknologi |
|---|---|
| Language | Go 1.22+ |
| Framework | Fiber v2 |
| Testing | Go `testing` package |

## Setup & Run

### Prasyarat
- Go 1.22+

### 1. Clone & install dependencies

```bash
cd backend
go mod download
```

### 2. Konfigurasi environment

```bash
cp .env.example .env
```

| Variable | Default | Keterangan |
|---|---|---|
| `PORT` | `9090` | Port HTTP server |

### 3. Jalankan server

```bash
go run ./cmd/server
```

Server berjalan di `http://localhost:9090`.

## API Endpoints

### `GET /api/health`

```json
{ "status": "ok" }
```

### `GET /api/earthquakes`

Data gabungan BMKG (15 gempa terbaru) + USGS (historis 6 bulan), skor seismisitas per provinsi, dan statistik ringkas.

Jika salah satu sumber data gagal, endpoint tetap merespons dengan data yang tersedia (`partial_data: true`).

**Contoh response:**
```json
{
  "live_feed": [
    {
      "id": "bmkg-2026-07-04T061500+0700--6.99_125.83",
      "source": "bmkg",
      "time": "2026-07-04T06:15:00+07:00",
      "latitude": -6.99,
      "longitude": 125.83,
      "magnitude": 5.6,
      "depth_km": 545,
      "region": "202 km TimurLaut ALOR-NTT",
      "province": "Nusa Tenggara Timur",
      "tsunami_potential": false
    }
  ],
  "historical_summary": {
    "period": "6 bulan terakhir",
    "total_events": 1247,
    "by_province": [
      {
        "province": "Maluku",
        "count": 187,
        "avg_magnitude": 5.12,
        "max_magnitude": 6.8,
        "risk_score": 95.4
      }
    ]
  },
  "stats": {
    "latest_event_minutes_ago": 23,
    "avg_magnitude_last_15": 5.1,
    "strongest_last_30_days": {
      "magnitude": 6.8,
      "province": "Maluku"
    }
  },
  "cached_at": "2026-07-04T06:20:00Z",
  "partial_data": false
}
```

## Testing

```bash
go test ./...
```

Unit test mencakup:
- `internal/mapper` — `MapToProvince()` dengan berbagai koordinat
- `internal/scorer` — kalkulasi skor seismisitas, normalisasi, urutan descending

## Province Mapping

Bounding box 38 provinsi Indonesia di-hardcode langsung di `internal/mapper/province.go` — tidak butuh database. Koordinat bersumber dari Nominatim OpenStreetMap dengan koreksi manual untuk provinsi hasil pemekaran 2022.

## Sumber Data

| Sumber | Endpoint | Format | Catatan |
|---|---|---|---|
| BMKG | `data.bmkg.go.id/DataMKG/TEWS/gempaterkini.xml` | XML | 15 gempa terbaru, koordinat `lat,long` |
| USGS | `earthquake.usgs.gov/fdsnws/event/1/query` | GeoJSON | Historis 6 bulan M≥4.5, koordinat `long,lat,depth` |

> Urutan koordinat BMKG dan USGS **terbalik** — normalisasi dilakukan di masing-masing fetcher.

Kedua sumber data berstatus **public domain** / data publik pemerintah dan legal untuk ditampilkan.
