package fetcher

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/izzahnin/disaster-risk-intelligence-backend/internal/mapper"
	"github.com/izzahnin/disaster-risk-intelligence-backend/internal/model"
)

// bmkgURL adalah endpoint resmi BMKG untuk data gempa terkini dalam format XML.
// API ini tidak memerlukan autentikasi dan selalu mengembalikan 15 gempa terbaru.
// Dokumentasi: https://data.bmkg.go.id/gempabumi/
const bmkgURL = "https://data.bmkg.go.id/DataMKG/TEWS/gempaterkini.xml"

// bmkgXML adalah representasi Go dari root element XML BMKG.
// Tag `xml:"gempa"` memberitahu decoder untuk mengumpulkan semua elemen <gempa>
// ke dalam slice Gempa.
type bmkgXML struct {
	Gempa []bmkgGempa `xml:"gempa"`
}

// bmkgGempa memetakan satu elemen <gempa> dari XML BMKG ke struct Go.
// Beberapa hal yang perlu diperhatikan:
//   - Coordinates: path `point>coordinates` berarti elemen bersarang <point><coordinates>
//   - Magnitude dan Kedalaman dikirim sebagai string (bukan angka) oleh BMKG
//   - Potensi berisi teks bebas, misal "Tidak berpotensi tsunami" atau "Berpotensi tsunami"
type bmkgGempa struct {
	DateTime    string `xml:"DateTime"`
	Coordinates string `xml:"point>coordinates"`
	Magnitude   string `xml:"Magnitude"`
	Kedalaman   string `xml:"Kedalaman"`
	Wilayah     string `xml:"Wilayah"`
	Potensi     string `xml:"Potensi"`
}

// FetchBMKG mengambil data 15 gempa terbaru dari API BMKG dan mengonversinya
// ke slice []model.Earthquake yang siap dipakai oleh handler.
//
// Alur kerja:
//  1. Buat HTTP request dengan timeout 5 detik (agar tidak blocking lama)
//  2. GET ke endpoint BMKG, baca response body
//  3. Unmarshal XML → struct bmkgXML
//  4. Loop setiap gempa: parse koordinat, waktu, magnitudo, kedalaman
//  5. Tentukan provinsi dengan mapper.MapToProvince(lat, lng)
//  6. Gempa dengan koordinat, waktu, atau magnitudo tidak valid di-skip (continue)
//     — zero value tidak boleh lolos ke response (mag=0 atau time=epoch tidak bermakna)
func FetchBMKG(ctx context.Context) ([]model.Earthquake, error) {
	// Timeout 5 detik mencegah fetch menggantung jika BMKG lambat merespons.
	// Context ini diturunkan dari context request HTTP asli.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bmkgURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// xml.Unmarshal mengisi struct bmkgXML berdasarkan tag xml yang sudah didefinisikan.
	var data bmkgXML
	if err := xml.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	earthquakes := make([]model.Earthquake, 0, len(data.Gempa))
	for _, g := range data.Gempa {
		// parseCoordsBMKG akan error jika format koordinat tidak sesuai.
		// Gempa dengan koordinat rusak di-skip agar tidak mempengaruhi data lain.
		lat, lng, err := parseCoordsBMKG(g.Coordinates)
		if err != nil {
			continue
		}

		// Waktu dan magnitudo adalah field wajib — kalau parse gagal, data gempa ini
		// tidak bisa direpresentasikan dengan benar. Di-skip agar zero value (time=0, mag=0)
		// tidak lolos ke response, konsisten dengan skip koordinat rusak di atas.
		t, err := time.Parse(time.RFC3339, g.DateTime)
		if err != nil {
			continue
		}
		mag, err := strconv.ParseFloat(strings.TrimSpace(g.Magnitude), 64)
		if err != nil {
			continue
		}
		depth := parseDepth(g.Kedalaman)

		// ID dibentuk dari DateTime + koordinat, bukan index loop.
		// Index bergeser tiap fetch (gempa baru masuk → semua ID lama berubah),
		// yang menyebabkan frontend tidak bisa track gempa yang sama antar request.
		// DateTime + koordinat unik per event dan stabil lintas fetch.
		stableID := fmt.Sprintf("bmkg-%s-%s",
			strings.ReplaceAll(g.DateTime, ":", ""),
			strings.ReplaceAll(g.Coordinates, ",", "_"),
		)

		earthquakes = append(earthquakes, model.Earthquake{
			ID:               stableID,
			Source:           "bmkg",
			Time:             t,
			Latitude:         lat,
			Longitude:        lng,
			Magnitude:        mag,
			DepthKm:          depth,
			Region:           g.Wilayah,
			// MapToProvince mencocokkan koordinat dengan bounding box 38 provinsi dari DB.
			// Jika tidak cocok (misal gempa di Filipina yang dimonitor BMKG), returns "Wilayah Lain".
			Province:         mapper.MapToProvince(lat, lng),
			// BMKG mengirim teks bebas: "Berpotensi tsunami" atau "Tidak berpotensi tsunami".
			// strings.Contains saja tidak cukup — "Tidak berpotensi tsunami" juga mengandung
			// substring "berpotensi tsunami", sehingga false positive. Pakai EqualFold agar
			// perbandingan case-insensitive dan tidak terpengaruh variasi kapitalisasi BMKG.
			TsunamiPotential: strings.EqualFold(strings.TrimSpace(g.Potensi), "berpotensi tsunami"),
		})
	}

	return earthquakes, nil
}

// parseCoordsBMKG mem-parse koordinat dari format string BMKG: "lat,long"
// Contoh input: "-6.99,125.83"
//
// Penting: BMKG mengirim latitude dulu baru longitude — urutan ini berbeda dari
// konvensi GeoJSON (yang longitude dulu). Fungsi ini mengikuti urutan BMKG.
func parseCoordsBMKG(s string) (lat, lng float64, err error) {
	parts := strings.SplitN(strings.TrimSpace(s), ",", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid coordinates: %q", s)
	}
	lat, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return
	}
	lng, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	return
}

// parseDepth mengonversi string kedalaman BMKG ke float64 dalam satuan km.
// Contoh: "545 km" → 545.0, "10 Km" → 10.0
//
// BMKG mengirim kedalaman sebagai string dengan satuan, bukan angka murni.
// strings.ToLower dipakai agar "Km", "KM", "km" semua tertangani.
func parseDepth(s string) float64 {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "km", "")
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}
