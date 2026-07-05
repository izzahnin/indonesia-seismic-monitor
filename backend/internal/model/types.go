package model

import "time"

// Earthquake merepresentasikan satu event gempa bumi dari sumber manapun (BMKG atau USGS).
// Struct ini adalah "bahasa bersama" seluruh codebase — fetcher mengisi field ini,
// scorer membacanya, handler mengirimkannya ke client.
//
// Field Source membedakan asal data karena format asli BMKG dan USGS sangat berbeda,
// tapi setelah normalisasi di fetcher, keduanya masuk ke struct yang sama ini.
type Earthquake struct {
	ID               string    `json:"id"`
	Source           string    `json:"source"` // "bmkg" = live feed | "usgs" = historis
	Time             time.Time `json:"time"`
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	Magnitude        float64   `json:"magnitude"`
	DepthKm          float64   `json:"depth_km"`
	Region           string    `json:"region"`    // deskripsi lokasi dari sumber asli
	Province         string    `json:"province"`  // hasil mapper: nama provinsi atau "Wilayah Lain"
	TsunamiPotential bool      `json:"tsunami_potential"`
}

// ProvinceSummary adalah hasil agregasi dan risk scoring untuk satu provinsi.
// Dihasilkan oleh scorer.Calculate() dari data USGS historis.
// Dipakai oleh frontend untuk tabel ranking dan bar chart.
type ProvinceSummary struct {
	Province     string  `json:"province"`
	Count        int     `json:"count"`          // jumlah event gempa dalam periode
	AvgMagnitude float64 `json:"avg_magnitude"`  // rata-rata magnitudo
	MaxMagnitude float64 `json:"max_magnitude"`  // magnitudo tertinggi
	RiskScore    float64 `json:"risk_score"`     // 0–100, hasil min-max normalization
}

// HistoricalSummary membungkus semua data historis yang dikirim ke client.
// Period adalah label teks (misal "6 bulan terakhir"), bukan kalkulasi otomatis.
type HistoricalSummary struct {
	Period      string            `json:"period"`
	TotalEvents int               `json:"total_events"`
	ByProvince  []ProvinceSummary `json:"by_province"` // diurutkan descending by risk_score
}

// StrongestEvent menyimpan info gempa terkuat dalam 30 hari terakhir.
// Hanya menyimpan Magnitude dan Province — bukan seluruh Earthquake —
// karena frontend hanya butuh dua field ini untuk stat card.
type StrongestEvent struct {
	Magnitude float64 `json:"magnitude"`
	Province  string  `json:"province"`
}

// Stats berisi angka-angka ringkasan yang ditampilkan di stat cards dashboard.
// Semua dihitung oleh buildStats() di handler dari data live (BMKG) dan historis (USGS).
//
// StrongestLast30Days adalah pointer karena bisa nil — jika tidak ada gempa dalam 30 hari
// terakhir, JSON akan mengirim `null` bukan `{"magnitude":0,"province":""}` yang ambigu.
type Stats struct {
	LatestEventMinutesAgo int             `json:"latest_event_minutes_ago"` // dari BMKG live feed
	AvgMagnitudeLast15    float64         `json:"avg_magnitude_last_15"`    // rata-rata 15 gempa BMKG terbaru
	StrongestLast30Days   *StrongestEvent `json:"strongest_last_30_days"`   // nil jika tidak ada data 30 hari
}

// DashboardResponse adalah shape JSON lengkap yang dikembalikan endpoint GET /api/earthquakes.
// Satu response ini mengandung semua data yang dibutuhkan frontend — live feed, historis, dan stats.
//
// PartialData=true berarti salah satu sumber (BMKG atau USGS) gagal di-fetch;
// data yang berhasil tetap dikembalikan (partial degradation, app tidak crash).
type DashboardResponse struct {
	LiveFeed          []Earthquake      `json:"live_feed"`
	HistoricalSummary HistoricalSummary `json:"historical_summary"`
	Stats             Stats             `json:"stats"`
	CachedAt          time.Time         `json:"cached_at"`
	PartialData       bool              `json:"partial_data,omitempty"` // omitempty: tidak muncul di JSON jika false
}
