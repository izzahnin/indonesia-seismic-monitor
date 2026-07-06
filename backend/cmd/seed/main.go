package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/izzahnin/disaster-risk-intelligence-backend/internal/db"
	"github.com/joho/godotenv"
)

var provinces = []string{
	"Aceh",
	"Sumatera Utara",
	"Sumatera Barat",
	"Riau",
	"Kepulauan Riau",
	"Jambi",
	"Sumatera Selatan",
	"Kepulauan Bangka Belitung",
	"Bengkulu",
	"Lampung",
	"DKI Jakarta",
	"Banten",
	"Jawa Barat",
	"Jawa Tengah",
	"DI Yogyakarta",
	"Jawa Timur",
	"Bali",
	"Nusa Tenggara Barat",
	"Nusa Tenggara Timur",
	"Kalimantan Barat",
	"Kalimantan Tengah",
	"Kalimantan Selatan",
	"Kalimantan Timur",
	"Kalimantan Utara",
	"Sulawesi Utara",
	"Gorontalo",
	"Sulawesi Tengah",
	"Sulawesi Barat",
	"Sulawesi Selatan",
	"Sulawesi Tenggara",
	"Maluku Utara",
	"Maluku",
	"Papua Barat Daya",
	"Papua Barat",
	"Papua Pegunungan",
	"Papua Tengah",
	"Papua Selatan",
	"Papua",
}

// hardcodedBBox untuk provinsi yang belum ada di Nominatim (pemekaran baru, dll)
// format: [minLat, maxLat, minLng, maxLng]
var hardcodedBBox = map[string][4]float64{
	"Bali":                   {-8.9, -8.05, 114.35, 115.85},
	"Nusa Tenggara Timur":    {-11.1, -8.0, 118.9, 125.05},
	"Sulawesi Utara":         {-1.0, 4.8, 123.0, 126.9},
	"Sulawesi Tenggara":      {-6.2, -2.55, 120.6, 125.8},
	"Maluku":                 {-8.5, -2.8, 126.0, 135.6},
	"Papua Selatan":          {-9.3, -4.3, 136.0, 141.0},
}

// maxBBoxDegrees adalah batas dimensi bbox yang dianggap masih wajar untuk satu provinsi.
// Nominatim kadang mengembalikan bbox seluruh negara atau wilayah yang terlalu luas
// ketika query tidak cocok dengan satu provinsi spesifik — bbox seperti itu akan merusak
// MapToProvince karena satu provinsi menutupi hampir seluruh Indonesia.
// 10 derajat ≈ ~1100 km — cukup besar untuk provinsi terluas (Papua) tapi menolak bbox negara.
const maxBBoxDegrees = 10.0

type nominatimResult struct {
	BoundingBox []string `json:"boundingbox"`
	Class       string   `json:"class"`
	Type        string   `json:"type"`
}

func parseBBox(bb []string) (minLat, maxLat, minLng, maxLng float64, err error) {
	if len(bb) < 4 {
		return 0, 0, 0, 0, fmt.Errorf("bbox has fewer than 4 elements")
	}
	vals := make([]float64, 4)
	for i, s := range bb[:4] {
		v, e := strconv.ParseFloat(s, 64)
		if e != nil {
			return 0, 0, 0, 0, fmt.Errorf("parse bbox[%d]: %w", i, e)
		}
		vals[i] = v
	}
	return vals[0], vals[1], vals[2], vals[3], nil
}

func fetchBBox(province string) (minLat, maxLat, minLng, maxLng float64, err error) {
	query := province + " Indonesia"
	reqURL := "https://nominatim.openstreetmap.org/search?q=" + url.QueryEscape(query) + "&format=json&limit=10"

	req, _ := http.NewRequest("GET", reqURL, nil)
	req.Header.Set("User-Agent", "indonesia-seismic-monitor/1.0 (cacaizzah2003@gmail.com)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var results []nominatimResult
	if err := json.Unmarshal(body, &results); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("json: %w", err)
	}

	// Pilih hasil dengan class=boundary, type=administrative, dan area terbesar
	var bestMinLat, bestMaxLat, bestMinLng, bestMaxLng float64
	bestArea := -1.0

	for i := range results {
		r := &results[i]
		if r.Class != "boundary" || r.Type != "administrative" {
			continue
		}
		mn, mx, ml, mxl, e := parseBBox(r.BoundingBox)
		if e != nil {
			continue
		}
		area := math.Abs(mx-mn) * math.Abs(mxl-ml)
		if area > bestArea {
			bestArea = area
			bestMinLat, bestMaxLat, bestMinLng, bestMaxLng = mn, mx, ml, mxl
		}
	}

	if bestArea < 0 {
		return 0, 0, 0, 0, fmt.Errorf("no administrative boundary found for %q", province)
	}

	// Tolak bbox yang terlalu luas — kemungkinan Nominatim mengembalikan batas negara
	// atau wilayah yang jauh lebih besar dari satu provinsi.
	latSpan := math.Abs(bestMaxLat - bestMinLat)
	lngSpan := math.Abs(bestMaxLng - bestMinLng)
	if latSpan > maxBBoxDegrees || lngSpan > maxBBoxDegrees {
		return 0, 0, 0, 0, fmt.Errorf(
			"bbox for %q too large (lat=%.2f°, lng=%.2f°) — kemungkinan bukan boundary provinsi",
			province, latSpan, lngSpan,
		)
	}

	return bestMinLat, bestMaxLat, bestMinLng, bestMaxLng, nil
}

func upsert(sqlDB *sql.DB, province string, minLat, maxLat, minLng, maxLng float64) error {
	_, err := sqlDB.Exec(`
		INSERT INTO province_bbox (province, min_lat, max_lat, min_lng, max_lng)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (province) DO UPDATE SET
			min_lat = EXCLUDED.min_lat,
			max_lat = EXCLUDED.max_lat,
			min_lng = EXCLUDED.min_lng,
			max_lng = EXCLUDED.max_lng
	`, province, minLat, maxLat, minLng, maxLng)
	return err
}

func main() {
	_ = godotenv.Load()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL env var is required")
	}

	sqlDB, err := db.Open(databaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer sqlDB.Close()

	if err := db.Migrate(sqlDB); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	log.Printf("Seeding %d provinces from Nominatim...\n", len(provinces))

	ok, failed := 0, 0
	for i, province := range provinces {
		var minLat, maxLat, minLng, maxLng float64
		var fetchErr error

		// Cek hardcoded fallback dulu
		if hc, found := hardcodedBBox[province]; found {
			minLat, maxLat, minLng, maxLng = hc[0], hc[1], hc[2], hc[3]
			log.Printf("HC   [%d/%d] %s  bbox=[%.4f, %.4f, %.4f, %.4f] (hardcoded)",
				i+1, len(provinces), province, minLat, maxLat, minLng, maxLng)
		} else {
			minLat, maxLat, minLng, maxLng, fetchErr = fetchBBox(province)
		}

		if fetchErr != nil {
			log.Printf("FAIL [%d/%d] %s: %v", i+1, len(provinces), province, fetchErr)
			failed++
		} else {
			if err := upsert(sqlDB, province, minLat, maxLat, minLng, maxLng); err != nil {
				log.Printf("FAIL [%d/%d] %s (db): %v", i+1, len(provinces), province, err)
				failed++
			} else if _, isHC := hardcodedBBox[province]; !isHC {
				log.Printf("OK   [%d/%d] %s  bbox=[%.4f, %.4f, %.4f, %.4f]",
					i+1, len(provinces), province, minLat, maxLat, minLng, maxLng)
				ok++
			} else {
				ok++
			}
		}

		if i < len(provinces)-1 {
			// Tidak perlu sleep untuk hardcoded (tidak hit API)
			if _, isHC := hardcodedBBox[province]; !isHC {
				time.Sleep(2 * time.Second)
			}
		}
	}

	log.Printf("\nDone. %d OK, %d failed.", ok, failed)
}
