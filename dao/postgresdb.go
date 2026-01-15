package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ericfialkowski/shorturl/env"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDB struct {
	pool *pgxpool.Pool
}

func newPgContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), env.DurationOrDefault("postgres_timeout", 10*time.Second))
}

// CreatePostgresDB creates a new PostgreSQL-backed ShortUrlDao.
// The connString should be a PostgreSQL connection string, e.g.:
// "postgres://user:password@localhost:5432/shorturl"
func CreatePostgresDB(connString string) ShortUrlDao {
	ctx, cancel := newPgContext()
	defer cancel()

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		log.Fatalf("Unable to parse connection string: %v", err)
	}
	config.MaxConns = int32(env.IntOrDefault("postgres_max_conns", 10))

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}

	db := &PostgresDB{pool: pool}
	db.initSchema()

	return db
}

func (d *PostgresDB) initSchema() {
	ctx, cancel := newPgContext()
	defer cancel()

	// Create the table if it doesn't exist
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS short_urls (
			id SERIAL PRIMARY KEY,
			abbreviation VARCHAR(50) NOT NULL UNIQUE,
			url TEXT NOT NULL UNIQUE,
			hits INTEGER NOT NULL DEFAULT 0,
			last_access TIMESTAMP WITH TIME ZONE,
			daily_hits JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_short_urls_abbreviation ON short_urls(abbreviation);
		CREATE INDEX IF NOT EXISTS idx_short_urls_url ON short_urls(url);
	`

	if _, err := d.pool.Exec(ctx, createTableSQL); err != nil {
		log.Printf("Error creating schema: %v", err)
	}
}

func (d *PostgresDB) Cleanup() {
	d.pool.Close()
}

func (d *PostgresDB) IsLikelyOk() bool {
	ctx, cancel := newPgContext()
	defer cancel()

	if err := d.pool.Ping(ctx); err != nil {
		log.Printf("Ping failed: %v", err)
		return false
	}
	return true
}

func (d *PostgresDB) Save(abv string, url string) error {
	ctx, cancel := newPgContext()
	defer cancel()

	sql := `
		INSERT INTO short_urls (abbreviation, url, hits, daily_hits)
		VALUES ($1, $2, 0, '{}'::jsonb)
		ON CONFLICT (abbreviation) DO NOTHING
	`

	result, err := d.pool.Exec(ctx, sql, abv, url)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return nil // Treat duplicate as success (same as MongoDB impl)
		}
		return fmt.Errorf("couldn't store (%s, %s): %v", abv, url, err)
	}

	if result.RowsAffected() == 0 {
		// Check if it was a conflict on abbreviation vs url
		var existingUrl string
		err := d.pool.QueryRow(ctx, "SELECT url FROM short_urls WHERE abbreviation = $1", abv).Scan(&existingUrl)
		if err == nil && existingUrl != url {
			return fmt.Errorf("abbreviation %s already exists with different URL", abv)
		}
	}

	return nil
}

func (d *PostgresDB) DeleteAbv(abv string) error {
	ctx, cancel := newPgContext()
	defer cancel()

	sql := `DELETE FROM short_urls WHERE abbreviation = $1`
	if _, err := d.pool.Exec(ctx, sql, abv); err != nil {
		return fmt.Errorf("couldn't delete abbreviation %s: %v", abv, err)
	}
	return nil
}

func (d *PostgresDB) DeleteUrl(url string) error {
	ctx, cancel := newPgContext()
	defer cancel()

	sql := `DELETE FROM short_urls WHERE url = $1`
	if _, err := d.pool.Exec(ctx, sql, url); err != nil {
		return fmt.Errorf("couldn't delete URL %s: %v", url, err)
	}
	return nil
}

func (d *PostgresDB) GetUrl(abv string) (string, error) {
	ctx, cancel := newPgContext()
	defer cancel()

	var url string
	sql := `SELECT url FROM short_urls WHERE abbreviation = $1`
	err := d.pool.QueryRow(ctx, sql, abv).Scan(&url)

	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("error getting URL for %s: %v", abv, err)
	}

	// Update stats asynchronously
	go func() {
		ctx, cancel := newPgContext()
		defer cancel()

		today := Date()
		updateSQL := `
			UPDATE short_urls
			SET hits = hits + 1,
				last_access = CURRENT_TIMESTAMP,
				daily_hits = jsonb_set(
					daily_hits,
					$1::text[],
					to_jsonb(COALESCE((daily_hits->>$2)::int, 0) + 1)
				)
			WHERE abbreviation = $3
		`
		if _, err := d.pool.Exec(ctx, updateSQL, []string{today}, today, abv); err != nil {
			log.Printf("Error updating stats: %v", err)
		}
	}()

	return url, nil
}

func (d *PostgresDB) GetAbv(url string) (string, error) {
	ctx, cancel := newPgContext()
	defer cancel()

	var abv string
	sql := `SELECT abbreviation FROM short_urls WHERE url = $1`
	err := d.pool.QueryRow(ctx, sql, url).Scan(&abv)

	if err != nil {
		if err == pgx.ErrNoRows {
			log.Printf("no abbreviation found for URL %s", url)
			return "", nil
		}
		return "", fmt.Errorf("error getting abbreviation for %s: %v", url, err)
	}

	return abv, nil
}

func (d *PostgresDB) GetStats(abv string) (ShortUrl, error) {
	ctx, cancel := newPgContext()
	defer cancel()

	var data ShortUrl
	var dailyHitsJSON []byte
	var lastAccess *time.Time

	sql := `
		SELECT abbreviation, url, hits, last_access, daily_hits
		FROM short_urls
		WHERE abbreviation = $1
	`
	err := d.pool.QueryRow(ctx, sql, abv).Scan(
		&data.Abbreviation,
		&data.Url,
		&data.Hits,
		&lastAccess,
		&dailyHitsJSON,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			log.Printf("no stats found for %s", abv)
			return ShortUrl{}, nil
		}
		return ShortUrl{}, fmt.Errorf("error getting stats for %s: %v", abv, err)
	}

	if lastAccess != nil {
		data.LastAccess = *lastAccess
	}

	// Parse the JSONB daily_hits into map
	data.DailyHits = make(map[string]int)
	if len(dailyHitsJSON) > 0 {
		if err := json.Unmarshal(dailyHitsJSON, &data.DailyHits); err != nil {
			log.Printf("Error parsing daily_hits JSON: %v", err)
		}
	}

	return data, nil
}
