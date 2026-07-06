package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/izzahnin/disaster-risk-intelligence-backend/internal/cache"
	"github.com/izzahnin/disaster-risk-intelligence-backend/internal/handler"
)

// main adalah entrypoint server.
// Urutan inisialisasi: env vars → cache client → HTTP server
//
// Province mapping menggunakan bbox 38 provinsi yang di-hardcode di internal/mapper.
// Tanpa Redis → setiap request fetch langsung ke BMKG/USGS (lebih lambat, tapi tetap jalan).
func main() {
	// godotenv.Load membaca file .env ke environment variables.
	// Error diabaikan karena .env tidak wajib ada — di production env vars
	// diset langsung oleh platform (Render, Vercel, dll).
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	redisURL := os.Getenv("REDIS_URL")

	// Fiber adalah HTTP framework yang dipakai sebagai pengganti net/http standar.
	// Lebih cepat untuk API JSON karena dibangun di atas fasthttp.
	app := fiber.New(fiber.Config{
		AppName: "indonesia-seismic-monitor",
	})

	// CORS middleware mengizinkan semua origin ("*") agar frontend di localhost:3000
	// bisa mengakses API di localhost:9090 tanpa browser memblok request.
	// Di production, sebaiknya dibatasi ke domain frontend yang spesifik.
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
	}))

	// Inisialisasi cache dan handler, lalu daftarkan routes.
	// NewClient akan return no-op client jika REDIS_URL kosong.
	cacheClient := cache.NewClient(redisURL)
	h := handler.New(cacheClient)
	h.Register(app) // mendaftarkan /api/health dan /api/earthquakes

	log.Printf("Server starting on :%s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatal(err)
	}
}
