package scorer

import (
	"math"
	"sort"

	"github.com/izzahnin/disaster-risk-intelligence-backend/internal/model"
)

// Calculate menghitung skor seismisitas per provinsi dari slice gempa historis (USGS).
// Output diurutkan descending by risk_score — provinsi dengan aktivitas seismik tertinggi di indeks 0.
//
// Skor seismisitas mengukur aktivitas kegempaan, bukan risiko penuh (R = H × V).
// Dua faktor dengan bobot 50:50:
//   - Frekuensi (count): log10(count), dinormalisasi min-max antar provinsi
//   - Intensitas (avg_magnitude): skala tetap (avgMag - 4.5) / (8.0 - 4.5) * 100
func Calculate(earthquakes []model.Earthquake) []model.ProvinceSummary {
	if len(earthquakes) == 0 {
		return []model.ProvinceSummary{}
	}

	// agg adalah struct internal untuk akumulasi data per provinsi sebelum dinormalisasi.
	// Didefinisikan di dalam fungsi karena hanya dipakai di sini.
	type agg struct {
		count        int
		sumMagnitude float64
		maxMagnitude float64
	}

	// Group semua gempa berdasarkan nama provinsinya.
	// "Wilayah Lain" dilewati karena bukan provinsi nyata — koordinatnya jatuh di luar
	// semua bbox yang diketahui (laut lepas, perbatasan negara lain, dll).
	// Tanpa filter ini, "Wilayah Lain" sering muncul #1 di ranking karena volume tinggi.
	byProvince := make(map[string]*agg)
	for _, eq := range earthquakes {
		if eq.Province == "Wilayah Lain" {
			continue
		}
		a, ok := byProvince[eq.Province]
		if !ok {
			a = &agg{}
			byProvince[eq.Province] = a
		}
		a.count++
		a.sumMagnitude += eq.Magnitude
		if eq.Magnitude > a.maxMagnitude {
			a.maxMagnitude = eq.Magnitude
		}
	}

	// Konversi map ke slice agar bisa diurutkan dan dinormalisasi.
	summaries := make([]model.ProvinceSummary, 0, len(byProvince))
	for province, a := range byProvince {
		summaries = append(summaries, model.ProvinceSummary{
			Province:     province,
			Count:        a.count,
			// Pembulatan 2 desimal agar JSON tidak terlalu panjang (misal 5.847284... → 5.85)
			AvgMagnitude: math.Round(a.sumMagnitude/float64(a.count)*100) / 100,
			MaxMagnitude: a.maxMagnitude,
		})
	}

	// Isi field RiskScore di setiap summary.
	calcRiskScores(summaries)

	// Urutkan descending: provinsi risk_score tertinggi tampil pertama di tabel/chart.
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].RiskScore > summaries[j].RiskScore
	})

	return summaries
}

// calcRiskScores menghitung risk_score untuk setiap ProvinceSummary.
//
// Dua komponen dengan bobot 50:50:
//   - Frekuensi : min-max normalization terhadap log10(count)
//   - Intensitas: skala tetap (avgMag - 4.5) / (8.0 - 4.5) * 100
//
// log10(count) dipakai agar satu provinsi dengan count sangat tinggi (misal 217)
// tidak mendominasi dan mengkompresi semua provinsi lain ke nol. Dengan log,
// perbedaan antara count=3 dan count=20 tetap bermakna meski ada count=217.
//
// Skala tetap dipakai untuk avgMag agar perbedaan kecil antar provinsi
// (misal 4.8 vs 5.0) tidak diperbesar secara artifisial oleh min-max.
// Rentang 4.5–8.0 mencakup semua gempa yang dimonitor USGS (M≥4.5)
// hingga gempa besar yang pernah terjadi di Indonesia.
//
// Skor gabungan dinormalisasi akhir ke 0–100 agar provinsi teratas selalu 100.
//
// Edge case: jika hanya 1 provinsi, risk_score = 100.
func calcRiskScores(summaries []model.ProvinceSummary) {
	if len(summaries) == 1 {
		summaries[0].RiskScore = 100
		return
	}

	// log10(count) — min paling kecil adalah log10(1)=0.
	logCounts := make([]float64, len(summaries))
	for i, s := range summaries {
		logCounts[i] = math.Log10(float64(s.Count))
	}
	minLog, maxLog := logCounts[0], logCounts[0]
	for _, v := range logCounts[1:] {
		if v < minLog {
			minLog = v
		}
		if v > maxLog {
			maxLog = v
		}
	}

	// Hitung skor gabungan mentah, lalu normalisasi akhir ke 0–100.
	raw := make([]float64, len(summaries))
	for i := range summaries {
		normCount := normalize(logCounts[i], minLog, maxLog)
		normAvg := (summaries[i].AvgMagnitude - 4.5) / (8.0 - 4.5) * 100
		raw[i] = normCount*0.5 + normAvg*0.5
	}

	minRaw, maxRaw := raw[0], raw[0]
	for _, v := range raw[1:] {
		if v < minRaw {
			minRaw = v
		}
		if v > maxRaw {
			maxRaw = v
		}
	}

	for i := range summaries {
		summaries[i].RiskScore = math.Round(normalize(raw[i], minRaw, maxRaw)*100) / 100
	}
}

// normalize mengaplikasikan rumus min-max: (val - min) / (max - min) * 100
// Hasilnya adalah angka 0–100.
//
// Edge case: jika max == min (semua nilai sama), semua provinsi dapat skor 100
// karena tidak ada yang lebih buruk dari yang lain.
func normalize(val, min, max float64) float64 {
	if max == min {
		return 100
	}
	return (val - min) / (max - min) * 100
}
