package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/izzahnin/disaster-risk-intelligence-backend/internal/cache"
	"github.com/izzahnin/disaster-risk-intelligence-backend/internal/db"
	"github.com/izzahnin/disaster-risk-intelligence-backend/internal/handler"
	"github.com/izzahnin/disaster-risk-intelligence-backend/internal/mapper"
)

// main adalah entrypoint server. Urutan inisialisasi penting:
// env vars → database (opsional) → cache client → HTTP server
//
// Filosofi error handling di sini: gagal di komponen opsional (DB, Redis) tidak
// menghentikan server. Sebaliknya, server berjalan dengan degradasi:
//   - Tanpa DB → mapper pakai hardcoded bbox 15 provinsi
//   - Tanpa Redis → setiap request fetch langsung ke BMKG/USGS (lebih lambat)
//   - Tanpa keduanya → app tetap berjalan, hanya lebih lambat
func main() {
	// godotenv.Load membaca file .env ke environment variables.
	// Error diabaikan (dengan _) karena .env tidak wajib ada — di production
	// env vars biasanya diset langsung oleh platform (Docker, Railway, dll).
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "9090" // default port jika PORT tidak diset
	}

	// PostgreSQL — opsional. Jika DATABASE_URL ada, server mencoba koneksi.
	// Jika gagal, fallback ke hardcodedBoxes di mapper (15 provinsi).
	// Alur: Open → Migrate (buat tabel jika belum ada) → LoadFromDB (isi provinceBoxes)
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		sqlDB, err := db.Open(databaseURL)
		if err != nil {
			log.Printf("WARNING: cannot connect to PostgreSQL: %v — using hardcoded province bounding boxes", err)
		} else {
			defer sqlDB.Close()
			if err := db.Migrate(sqlDB); err != nil {
				log.Printf("WARNING: migrate failed: %v", err)
			} else if err := mapper.LoadFromDB(sqlDB); err != nil {
				log.Printf("WARNING: load province bbox from DB failed: %v — using hardcoded fallback", err)
			}
		}
	} else {
		log.Println("WARNING: DATABASE_URL not set, using hardcoded province bounding boxes")
	}

	redisURL := os.Getenv("REDIS_URL")

	// Fiber adalah HTTP framework yang dipakai sebagai pengganti net/http standar.
	// Lebih cepat untuk API JSON karena dibangun di atas fasthttp.
	app := fiber.New(fiber.Config{
		AppName: "disaster-risk-intelligence",
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
