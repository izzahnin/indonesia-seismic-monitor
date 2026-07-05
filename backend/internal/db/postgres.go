package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Open membuka koneksi ke PostgreSQL dan mengonfigurasi connection pool.
//
// Connection pool mencegah dua masalah umum:
//   - Terlalu banyak koneksi terbuka sekaligus → PostgreSQL kehabisan slot (default max 100)
//   - Koneksi idle terlalu lama → server menutupnya secara diam-diam, kita dapat error "broken pipe"
//
// Nilai pool dipilih untuk layanan kecil (1 instance, traffic rendah-menengah):
//   - MaxOpenConns(10): batas atas koneksi aktif — sesuai untuk Upstash/Neon free tier
//   - MaxIdleConns(5): koneksi yang tetap terbuka saat idle untuk menghindari overhead reconnect
//   - ConnMaxLifetime(5m): paksa reconnect setiap 5 menit — cegah koneksi stale di belakang load balancer
func Open(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db open: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}
