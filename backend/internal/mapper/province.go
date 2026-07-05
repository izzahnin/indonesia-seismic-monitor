package mapper

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

// provinceBoxes adalah bounding box 38 provinsi Indonesia.
// Koordinat bersumber dari Nominatim OpenStreetMap — sama persis dengan yang
// dihasilkan cmd/seed. 6 provinsi pakai nilai hardcoded karena Nominatim tidak
// mengembalikan batas administratif yang akurat untuk wilayah tersebut:
// Bali, NTT, Sulawesi Utara, Sulawesi Tenggara, Maluku, Papua Selatan.
var provinceBoxes = []bbox{
	// ── Sumatera ──
	{"Aceh", 1.4586943, 6.2744496, 94.7717124, 98.6866758},
	{"Sumatera Utara", -1.0336376, 4.4395487, 96.7203878, 100.5496202},
	{"Sumatera Barat", -3.8839573, 0.9067222, 98.2364008, 101.8928544},
	{"Riau", -1.1281595, 3.2269013, 100.0248488, 103.9519995},
	{"Kepulauan Riau", -1.1825041, 4.9966383, 103.0646322, 109.7135102},
	{"Jambi", -2.7700765, -0.6436003, 101.1305567, 105.0122093},
	{"Sumatera Selatan", -4.9241592, -1.5138437, 102.0638889, 106.6026347},
	{"Kepulauan Bangka Belitung", -4.9993635, -0.2732799, 104.9996082, 109.3897948},
	{"Bengkulu", -5.7189866, -2.2886667, 100.6204863, 103.7810669},
	{"Lampung", -6.4550344, -3.7237393, 103.5068774, 106.8466516},

	// ── Jawa ──
	{"DKI Jakarta", -6.3744575, -4.9993635, 106.3146732, 106.9739750},
	{"Banten", -7.4565894, -5.4996381, 104.6513179, 106.7800127},
	{"Jawa Barat", -8.0207481, -4.0387936, 106.0509508, 109.0697907},
	{"Jawa Tengah", -8.4411879, -4.0387936, 108.5558548, 111.8689695},
	{"DI Yogyakarta", -8.4159039, -7.5412887, 109.9017890, 110.8386897},
	{"Jawa Timur", -9.0301357, -4.8926893, 110.8815987, 116.4841801},

	// ── Bali & Nusa Tenggara ──
	{"Bali", -8.9, -8.05, 114.35, 115.85},
	{"Nusa Tenggara Barat", -9.3098441, -7.5120245, 115.5700063, 119.4887427},
	{"Nusa Tenggara Timur", -11.1, -8.0, 118.9, 125.05},

	// ── Kalimantan ──
	{"Kalimantan Barat", -4.7153056, 2.3148604, 108.1386521, 114.2053845},
	{"Kalimantan Tengah", -5.1882715, 0.7910090, 110.6795452, 115.8493588},
	{"Kalimantan Selatan", -5.4138916, -1.3125795, 113.9911308, 117.6465697},
	{"Kalimantan Timur", -2.4540290, 2.5690230, 113.8343353, 119.6680859},
	{"Kalimantan Utara", 1.1140414, 4.4078230, 114.5651640, 118.7650776},

	// ── Sulawesi ──
	{"Sulawesi Utara", -1.0, 4.8, 123.0, 126.9},
	{"Gorontalo", -0.0665628, 1.3647141, 121.1612292, 123.5519226},
	{"Sulawesi Tengah", -3.6514082, 1.5835540, 118.8772126, 124.9619235},
	{"Sulawesi Barat", -3.9747153, -0.2274500, 116.9904732, 119.8748281},
	{"Sulawesi Selatan", -7.9722136, -1.8906412, 116.3438210, 122.2749425},
	{"Sulawesi Tenggara", -6.2, -2.55, 120.6, 125.8},

	// ── Maluku ──
	{"Maluku Utara", -2.7956171, 3.4075964, 123.9232297, 130.0686205},
	{"Maluku", -8.5, -2.8, 126.0, 135.6},

	// ── Papua (termasuk provinsi hasil pemekaran 2022) ──
	{"Papua Barat Daya", -2.9766065, 1.2874153, 129.0848423, 133.5217666},
	{"Papua Barat", -4.8517205, 0.5605096, 131.2767210, 135.3341667},
	{"Papua Tengah", -5.4600206, -2.1825333, 134.5852579, 138.3017785},
	{"Papua Pegunungan", -5.2648234, -3.1069562, 137.8269916, 141.0000000},
	{"Papua Selatan", -9.3, -4.3, 136.0, 141.0},
	{"Papua", -3.9567276, 1.1369057, 133.5217666, 141.0130219},
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
func MapToProvince(lat, lng float64) string {
	for _, b := range provinceBoxes {
		if lat >= b.MinLat && lat <= b.MaxLat && lng >= b.MinLng && lng <= b.MaxLng {
			return b.Province
		}
	}
	return "Wilayah Lain"
}
