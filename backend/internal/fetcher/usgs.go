package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/izzahnin/disaster-risk-intelligence-backend/internal/mapper"
	"github.com/izzahnin/disaster-risk-intelligence-backend/internal/model"
)

// usgsURL adalah endpoint FDSN (International Federation of Digital Seismograph Networks)
// yang dioperasikan USGS. API ini bersifat publik, global, dan sangat lengkap secara historis.
// Dokumentasi: https://earthquake.usgs.gov/fdsnws/event/1/
const usgsURL = "https://earthquake.usgs.gov/fdsnws/event/1/query"

// usgsResponse adalah root dari GeoJSON FeatureCollection yang dikembalikan USGS.
// GeoJSON adalah standar format geografis berbasis JSON (RFC 7946).
// Kita hanya butuh array Features — field lain seperti "type" dan "metadata" diabaikan.
type usgsResponse struct {
	Features []usgsFeature `json:"features"`
}

// usgsFeature mewakili satu gempa dalam format GeoJSON Feature.
// Setiap Feature memiliki Properties (atribut gempa) dan Geometry (posisi geografis).
type usgsFeature struct {
	Properties usgsProperties `json:"properties"`
	Geometry   usgsGeometry   `json:"geometry"`
}

// usgsProperties berisi atribut gempa dari USGS.
//   - Time: Unix timestamp dalam MILLISECONDS (bukan detik seperti biasanya)
//   - Tsunami: integer 0/1, bukan boolean — 1 berarti ada peringatan tsunami
type usgsProperties struct {
	Mag     float64 `json:"mag"`
	Place   string  `json:"place"`
	Time    int64   `json:"time"` // milliseconds epoch
	Tsunami int     `json:"tsunami"`
}

// usgsGeometry menyimpan posisi geografis dalam format GeoJSON Point.
//
// PERHATIAN — urutan koordinat GeoJSON adalah [longitude, latitude, depth],
// BUKAN [latitude, longitude] seperti yang lazim di peta atau BMKG.
// Ini adalah standar GeoJSON (RFC 7946 Section 3.1.1) yang sering menjadi sumber bug.
//
// Contoh: [-122.419, 37.774, 10.0] = longitude=-122.419, latitude=37.774, depth=10km
type usgsGeometry struct {
	Coordinates [3]float64 `json:"coordinates"` // [longitude, latitude, depth_km]
}

// FetchUSGS mengambil data gempa historis 6 bulan terakhir dari API USGS
// untuk wilayah Indonesia, lalu mengonversinya ke []model.Earthquake.
//
// Alur kerja:
//  1. Hitung startTime = 6 bulan lalu dari sekarang (dinamis, bukan hardcode)
//  2. Bangun URL query dengan parameter bounding box Indonesia dan filter magnitudo
//  3. GET ke USGS FDSN API, baca response body
//  4. Unmarshal GeoJSON → struct usgsResponse
//  5. Loop setiap feature: swap urutan koordinat, konversi timestamp, tentukan provinsi
func FetchUSGS(ctx context.Context) ([]model.Earthquake, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// startTime dihitung dinamis agar selalu 6 bulan dari waktu request.
	// Format "2006-01-02" adalah cara Go menulis layout tanggal (bukan angka arbitrer).
	startTime := time.Now().AddDate(0, -6, 0).Format("2006-01-02")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, usgsURL, nil)
	if err != nil {
		return nil, err
	}

	// Query parameter menentukan subset data yang dikembalikan USGS.
	// Bounding box Indonesia: lat -11 s/d 6, lon 95 s/d 141
	// minmagnitude 4.5: gempa di bawah ini umumnya tidak dirasakan dan tidak relevan untuk risk scoring.
	q := req.URL.Query()
	q.Set("format", "geojson")
	q.Set("starttime", startTime)
	q.Set("minmagnitude", "4.5")
	q.Set("minlatitude", "-11")
	q.Set("maxlatitude", "6")
	q.Set("minlongitude", "95")
	q.Set("maxlongitude", "141")
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data usgsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	earthquakes := make([]model.Earthquake, 0, len(data.Features))
	for i, f := range data.Features {
		// KRITIS: GeoJSON menyimpan koordinat sebagai [longitude, latitude, depth].
		// Kita harus swap index 0 dan 1 agar sesuai konvensi (lat, lng) yang dipakai mapper.
		lng := f.Geometry.Coordinates[0]
		lat := f.Geometry.Coordinates[1]
		depth := f.Geometry.Coordinates[2]

		// USGS menyimpan waktu sebagai Unix milliseconds, bukan detik.
		// time.UnixMilli mengonversinya ke time.Time. .UTC() memastikan zona waktu konsisten.
		t := time.UnixMilli(f.Properties.Time).UTC()

		earthquakes = append(earthquakes, model.Earthquake{
			ID:               fmt.Sprintf("usgs-%d", i),
			Source:           "usgs",
			Time:             t,
			Latitude:         lat,
			Longitude:        lng,
			Magnitude:        f.Properties.Mag,
			DepthKm:          depth,
			Region:           f.Properties.Place,
			Province:         mapper.MapToProvince(lat, lng),
			// USGS menggunakan integer untuk tsunami: 1 = ada peringatan tsunami.
			// Berbeda dengan BMKG yang menggunakan string teks bebas.
			TsunamiPotential: f.Properties.Tsunami == 1,
		})
	}

	return earthquakes, nil
}
