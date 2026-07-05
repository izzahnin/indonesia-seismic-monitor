package mapper

import (
	"database/sql"
	"log"
)

// bbox mendefinisikan bounding box (kotak pembatas) geografis satu provinsi.
// Koordinat menggunakan sistem WGS84 (standar GPS):
//   - Latitude: sumbu vertikal, negatif = selatan khatulistiwa
//   - Longitude: sumbu horizontal, positif = timur Greenwich
//
// Cara kerja: titik (lat, lng) dianggap masuk ke provinsi ini jika
// MinLat ≤ lat ≤ MaxLat DAN MinLng ≤ lng ≤ MaxLng
type bbox struct {
	Province               string
	MinLat, MaxLat         float64
	MinLng, MaxLng         float64
}

// hardcodedBoxes adalah data bbox fallback untuk 15 provinsi rawan gempa utama.
// Dipakai ketika DATABASE_URL tidak diset atau koneksi PostgreSQL gagal.
// Koordinat ini kasar (bukan batas administratif resmi) — cukup untuk klasifikasi awal.
//
// Untuk akurasi lebih baik, gunakan data dari PostgreSQL (province_bbox table)
// yang diisi via cmd/seed menggunakan data Nominatim + koreksi manual.
var hardcodedBoxes = []bbox{
	{"Aceh", 2.0, 6.0, 95.0, 98.5},
	{"Sumatera Utara", -0.5, 4.5, 97.0, 100.0},
	{"Sumatera Barat", -3.5, 0.5, 98.0, 101.5},
	{"Bengkulu", -5.5, -2.0, 101.0, 104.0},
	{"Lampung", -6.0, -3.5, 103.5, 106.0},
	{"Banten-Jabar", -8.0, -5.5, 105.0, 108.5},
	{"Jawa Tengah-DIY", -8.5, -6.0, 108.5, 111.5},
	{"Jawa Timur", -9.0, -6.5, 111.5, 114.5},
	{"Bali-NTB", -9.5, -7.5, 114.5, 119.0},
	{"NTT", -11.0, -7.5, 119.0, 125.5},
	{"Sulawesi Utara", 0.0, 5.5, 122.5, 127.5},
	{"Sulawesi Tengah", -3.0, 1.5, 119.0, 124.0},
	{"Maluku Utara", -1.0, 3.5, 125.5, 129.0},
	{"Maluku", -8.0, -1.0, 125.5, 135.0},
	{"Papua Barat-Papua", -9.0, 0.5, 130.0, 141.0},
}

// provinceBoxes adalah data aktif yang digunakan MapToProvince.
// Diinisialisasi dengan hardcodedBoxes, kemudian diganti oleh LoadFromDB
// jika PostgreSQL tersedia. Dengan cara ini server selalu punya data meski DB tidak ada.
var provinceBoxes = hardcodedBoxes

// LoadFromDB mengganti provinceBoxes dengan data dari tabel province_bbox di PostgreSQL.
// Dipanggil sekali saat server start di cmd/server/main.go.
//
// Jika tabel kosong, fallback ke hardcodedBoxes tetap dipakai (tidak menimpa dengan slice kosong).
// Jika berhasil, provinceBoxes berisi data 38 provinsi resmi dengan bbox yang lebih akurat.
func LoadFromDB(db *sql.DB) error {
	rows, err := db.Query(`SELECT province, min_lat, max_lat, min_lng, max_lng FROM province_bbox`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var boxes []bbox
	for rows.Next() {
		var b bbox
		if err := rows.Scan(&b.Province, &b.MinLat, &b.MaxLat, &b.MinLng, &b.MaxLng); err != nil {
			return err
		}
		boxes = append(boxes, b)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(boxes) == 0 {
		log.Println("mapper: province_bbox table is empty, using hardcoded fallback")
		return nil
	}

	provinceBoxes = boxes
	log.Printf("mapper: loaded %d province bounding boxes from database", len(boxes))
	return nil
}

// MapToProvince menentukan nama provinsi berdasarkan koordinat (lat, lng).
// Iterasi linear melalui provinceBoxes — cocok pertama yang ditemukan langsung dikembalikan.
//
// Return value:
//   - Nama provinsi (misal "Jawa Barat") jika koordinat masuk dalam bbox-nya
//   - "Wilayah Lain" jika tidak cocok dengan bbox manapun
//
// "Wilayah Lain" bisa berarti:
//   - Gempa di laut antara provinsi (celah antar bbox)
//   - Gempa di negara tetangga yang dimonitor BMKG (Filipina, Timor Leste, PNG)
//   - Provinsi yang tidak ada dalam daftar (jika pakai hardcodedBoxes yang hanya 15 provinsi)
func MapToProvince(lat, lng float64) string {
	for _, b := range provinceBoxes {
		if lat >= b.MinLat && lat <= b.MaxLat && lng >= b.MinLng && lng <= b.MaxLng {
			return b.Province
		}
	}
	return "Wilayah Lain"
}
