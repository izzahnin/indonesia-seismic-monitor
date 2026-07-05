package handler

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/izzahnin/disaster-risk-intelligence-backend/internal/cache"
	"github.com/izzahnin/disaster-risk-intelligence-backend/internal/fetcher"
	"github.com/izzahnin/disaster-risk-intelligence-backend/internal/model"
	"github.com/izzahnin/disaster-risk-intelligence-backend/internal/scorer"
	"golang.org/x/sync/errgroup"
)

const (
	cacheKey = "earthquakes:combined" // key Redis untuk menyimpan response JSON
	cacheTTL = 10 * time.Minute       // data di-cache 10 menit sebelum di-fetch ulang
)

// Handler memegang dependency yang dibutuhkan HTTP handler — dalam hal ini hanya cache client.
// Pola ini memudahkan testing: dependency disuntik via constructor, bukan global variable.
type Handler struct {
	cache *cache.Client
}

// New membuat Handler baru dengan cache client yang diberikan.
func New(c *cache.Client) *Handler {
	return &Handler{cache: c}
}

// Register mendaftarkan semua route ke Fiber app.
// Dipanggil sekali dari main() setelah app dibuat.
func (h *Handler) Register(app *fiber.App) {
	app.Get("/api/health", h.health)
	app.Get("/api/earthquakes", h.earthquakes)
}

// health adalah endpoint sederhana untuk memverifikasi server berjalan.
// Dipakai oleh monitoring, load balancer, atau saat development untuk cek server hidup.
func (h *Handler) health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

// earthquakes adalah handler utama yang mengorkestrasikan semua layer:
// cache → fetch paralel → score → build stats → cache → response.
//
// Strategi caching:
//   - Cache HIT: return JSON dari Redis langsung, sangat cepat (~1ms)
//   - Cache MISS: fetch BMKG + USGS secara paralel (errgroup), proses, simpan ke cache
//
// Strategi partial degradation:
//   - Jika BMKG gagal: live_feed kosong, historical tetap ada
//   - Jika USGS gagal: historical kosong, live_feed tetap ada
//   - Keduanya gagal: response kosong tapi tidak error 500, PartialData=true
func (h *Handler) earthquakes(c *fiber.Ctx) error {
	ctx := c.Context()

	// Cek cache dulu. Jika hit, langsung kirim tanpa proses apapun.
	// c.SendString lebih efisien dari JSON re-marshal karena string sudah siap kirim.
	if cached, err := h.cache.Get(ctx, cacheKey); err == nil {
		c.Set("Content-Type", "application/json")
		return c.SendString(cached)
	}

	var (
		bmkgData []model.Earthquake
		usgsData []model.Earthquake
		partial  bool
	)

	// errgroup menjalankan dua goroutine secara paralel dengan context bersama.
	// Jika salah satu goroutine return error non-nil, context dibatalkan.
	// Di sini kita sengaja return nil meski error (partial degradation) — kita catat
	// lewat flag `partial` bukan lewat error propagation.
	eg, egCtx := errgroup.WithContext(context.Background())

	eg.Go(func() error {
		data, err := fetcher.FetchBMKG(egCtx)
		if err != nil {
			partial = true
			return nil // partial degradation: lanjut meski BMKG gagal
		}
		bmkgData = data
		return nil
	})

	eg.Go(func() error {
		data, err := fetcher.FetchUSGS(egCtx)
		if err != nil {
			partial = true
			return nil // partial degradation: lanjut meski USGS gagal
		}
		usgsData = data
		return nil
	})

	eg.Wait() // blok sampai kedua goroutine selesai

	// scorer.Calculate hanya memakai data USGS (historis) untuk risk scoring per provinsi.
	// Data BMKG (live feed) tidak dipakai di sini karena hanya 15 data — tidak cukup untuk statistik.
	provinceSummaries := scorer.Calculate(usgsData)

	now := time.Now()
	resp := model.DashboardResponse{
		LiveFeed: bmkgData,
		HistoricalSummary: model.HistoricalSummary{
			Period:      "6 bulan terakhir",
			TotalEvents: len(usgsData),
			ByProvince:  provinceSummaries,
		},
		Stats:       buildStats(bmkgData, usgsData, now),
		CachedAt:    now,
		PartialData: partial,
	}

	b, err := json.Marshal(resp)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "marshal error"})
	}

	// Simpan ke cache untuk request berikutnya. Error diabaikan — cache miss berikutnya
	// hanya berarti fetch ulang, bukan kerusakan data.
	h.cache.Set(ctx, cacheKey, string(b), cacheTTL)

	c.Set("Content-Type", "application/json")
	return c.Send(b)
}

// buildStats menghitung angka-angka ringkasan untuk stat cards di dashboard.
// Menerima data live (BMKG) dan historis (USGS) secara terpisah karena
// masing-masing dipakai untuk stat yang berbeda.
func buildStats(live, historical []model.Earthquake, now time.Time) model.Stats {
	var stats model.Stats

	if len(live) > 0 {
		// Cari gempa paling baru dari live feed untuk menghitung "berapa menit lalu".
		// BMKG tidak menjamin urutan, jadi kita cari manual.
		latest := live[0].Time
		for _, eq := range live[1:] {
			if eq.Time.After(latest) {
				latest = eq.Time
			}
		}
		stats.LatestEventMinutesAgo = int(now.Sub(latest).Minutes())

		// Rata-rata magnitudo dari semua gempa di live feed (max 15 dari BMKG).
		var sumMag float64
		for _, eq := range live {
			sumMag += eq.Magnitude
		}
		stats.AvgMagnitudeLast15 = math.Round(sumMag/float64(len(live))*100) / 100
	}

	// Cari gempa terkuat dalam 30 hari terakhir dari data historis USGS.
	// cutoff = batas waktu — gempa sebelum cutoff diabaikan.
	// StrongestLast30Days tetap nil jika tidak ada gempa dalam periode ini —
	// nil lebih jelas dari {"magnitude":0,"province":""} yang ambigu di frontend.
	cutoff := now.AddDate(0, 0, -30)
	var strongest *model.Earthquake
	for i, eq := range historical {
		if eq.Time.Before(cutoff) {
			continue
		}
		if strongest == nil || eq.Magnitude > strongest.Magnitude {
			strongest = &historical[i]
		}
	}
	if strongest != nil {
		stats.StrongestLast30Days = &model.StrongestEvent{
			Magnitude: strongest.Magnitude,
			Province:  strongest.Province,
		}
	}

	return stats
}
