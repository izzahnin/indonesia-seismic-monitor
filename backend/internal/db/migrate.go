package db

import "database/sql"

func Migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS province_bbox (
			province TEXT PRIMARY KEY,
			min_lat  DOUBLE PRECISION NOT NULL,
			max_lat  DOUBLE PRECISION NOT NULL,
			min_lng  DOUBLE PRECISION NOT NULL,
			max_lng  DOUBLE PRECISION NOT NULL
		)
	`)
	return err
}
