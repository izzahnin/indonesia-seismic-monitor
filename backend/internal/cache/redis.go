package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client adalah wrapper tipis di atas go-redis yang mendukung pola "no-op saat tidak tersedia".
// Jika Redis tidak dikonfigurasi atau tidak bisa diakses, semua operasi tetap bisa dipanggil
// tanpa crash — Get selalu miss, Set selalu berhasil (diabaikan).
//
// Ini memungkinkan development lokal tanpa Redis: app berjalan normal, hanya tanpa caching.
type Client struct {
	rdb *redis.Client // nil jika Redis tidak dikonfigurasi
}

// NewClient membuat cache Client dari URL Redis.
// Jika redisURL kosong atau tidak valid, mengembalikan Client dengan rdb=nil (mode no-op).
//
// Format URL yang didukung: "redis://user:pass@host:port/db" atau "rediss://" untuk TLS.
// Upstash (Redis managed cloud) menggunakan format "rediss://".
func NewClient(redisURL string) *Client {
	if redisURL == "" {
		return &Client{}
	}
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return &Client{}
	}
	return &Client{rdb: redis.NewClient(opt)}
}

// Get mengambil nilai dari Redis berdasarkan key.
// Mengembalikan (value, nil) jika key ditemukan.
// Mengembalikan ("", redis.Nil) jika key tidak ada — ini adalah "cache miss" standar Redis.
// Jika rdb nil (no-op mode), selalu returns redis.Nil sehingga caller selalu anggap cache miss.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if c.rdb == nil {
		return "", redis.Nil
	}
	return c.rdb.Get(ctx, key).Result()
}

// Set menyimpan nilai ke Redis dengan TTL (time-to-live).
// Setelah TTL habis, key otomatis dihapus oleh Redis.
// Jika rdb nil (no-op mode), langsung return nil tanpa melakukan apapun.
func (c *Client) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if c.rdb == nil {
		return nil
	}
	return c.rdb.Set(ctx, key, value, ttl).Err()
}
