package scorer

import (
	"math"
	"sort"

	"github.com/izzahnin/disaster-risk-intelligence-backend/internal/model"
)

// Calculate menghitung risk_score per provinsi dari slice gempa historis (USGS).
// Output diurutkan descending by risk_score — provinsi paling berisiko di indeks 0.
//
// Risk score dihitung dengan menggabungkan dua faktor:
//   - Frekuensi (count): seberapa sering gempa terjadi di provinsi ini
//   - Intensitas (avg_magnitude): rata-rata kekuatan gempanya
//
// Rumus akhir: risk_score = normalize(count) * 0.5 + normalize(avg_magnitude) * 0.5
// Bobot 50:50 antara frekuensi dan intensitas.
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

	// Isi field RiskScore di setiap summary menggunakan min-max normalization.
	minMaxNormalize(summaries)

	// Urutkan descending: provinsi risk_score tertinggi tampil pertama di tabel/chart.
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].RiskScore > summaries[j].RiskScore
	})

	return summaries
}

// minMaxNormalize menghitung risk_score untuk setiap ProvinceSummary menggunakan
// min-max normalization — menskalakan nilai ke rentang 0–100 relatif terhadap semua provinsi.
//
// Cara kerja min-max: nilai terendah → 0, nilai tertinggi → 100, yang lain proporsional.
// Ini berarti risk_score bersifat relatif: provinsi dengan risk_score 80 bukan berarti
// "80% berbahaya", tapi "lebih berbahaya dari 80% provinsi lain dalam dataset ini".
//
// Edge case: jika hanya 1 provinsi (tidak ada pembanding), risk_score = 100 secara default.
func minMaxNormalize(summaries []model.ProvinceSummary) {
	if len(summaries) == 1 {
		summaries[0].RiskScore = 100
		return
	}

	// Cari nilai min dan max untuk count dan avg_magnitude dari semua provinsi.
	minCount, maxCount := float64(summaries[0].Count), float64(summaries[0].Count)
	minAvg, maxAvg := summaries[0].AvgMagnitude, summaries[0].AvgMagnitude

	for _, s := range summaries[1:] {
		c := float64(s.Count)
		if c < minCount {
			minCount = c
		}
		if c > maxCount {
			maxCount = c
		}
		if s.AvgMagnitude < minAvg {
			minAvg = s.AvgMagnitude
		}
		if s.AvgMagnitude > maxAvg {
			maxAvg = s.AvgMagnitude
		}
	}

	for i := range summaries {
		normCount := normalize(float64(summaries[i].Count), minCount, maxCount)
		normAvg := normalize(summaries[i].AvgMagnitude, minAvg, maxAvg)
		// Bobot 50:50 — frekuensi dan intensitas dianggap sama pentingnya.
		summaries[i].RiskScore = math.Round((normCount*0.5+normAvg*0.5)*100) / 100
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
