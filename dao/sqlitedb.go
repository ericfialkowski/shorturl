package dao

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ericfialkowski/shorturl/env"
	_ "modernc.org/sqlite"
)

type SQLiteDB struct {
	db *sql.DB
	mu sync.RWMutex
}

func newSqliteContext() (time.Duration, func()) {
	timeout := env.DurationOrDefault("sqlite_timeout", 10*time.Second)
	return timeout, func() {}
}

// CreateSQLiteDB creates a new SQLite-backed ShortUrlDao.
// The dbPath should be a path to the SQLite database file, e.g.:
// "./shorturl.db" or ":memory:" for in-memory database
func CreateSQLiteDB(dbPath string) ShortUrlDao {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Unable to open SQLite database: %v", err)
	}

	// SQLite performance tuning
	db.SetMaxOpenConns(1) // SQLite only supports one writer at a time
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0) // Keep connection open

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Printf("Warning: could not enable WAL mode: %v", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		log.Printf("Warning: could not set busy timeout: %v", err)
	}

	sqliteDB := &SQLiteDB{db: db}
	sqliteDB.initSchema()

	return sqliteDB
}

func (d *SQLiteDB) initSchema() {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Create the main short_urls table
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS short_urls (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			abbreviation TEXT NOT NULL UNIQUE,
			url TEXT NOT NULL UNIQUE,
			hits INTEGER NOT NULL DEFAULT 0,
			last_access DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_short_urls_abbreviation ON short_urls(abbreviation);
		CREATE INDEX IF NOT EXISTS idx_short_urls_url ON short_urls(url);
	`

	if _, err := d.db.Exec(createTableSQL); err != nil {
		log.Printf("Error creating short_urls table: %v", err)
	}

	// Create the daily_hits table for tracking hits per day
	createDailyHitsSQL := `
		CREATE TABLE IF NOT EXISTS daily_hits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			short_url_id INTEGER NOT NULL REFERENCES short_urls(id) ON DELETE CASCADE,
			hit_date DATE NOT NULL,
			hits INTEGER NOT NULL DEFAULT 0,
			UNIQUE(short_url_id, hit_date)
		);
		CREATE INDEX IF NOT EXISTS idx_daily_hits_short_url_id ON daily_hits(short_url_id);
		CREATE INDEX IF NOT EXISTS idx_daily_hits_date ON daily_hits(hit_date);
	`

	if _, err := d.db.Exec(createDailyHitsSQL); err != nil {
		log.Printf("Error creating daily_hits table: %v", err)
	}

	// Enable foreign key support
	if _, err := d.db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		log.Printf("Warning: could not enable foreign keys: %v", err)
	}
}

func (d *SQLiteDB) Cleanup() {
	d.db.Close()
}

func (d *SQLiteDB) IsLikelyOk() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if err := d.db.Ping(); err != nil {
		log.Printf("Ping failed: %v", err)
		return false
	}
	return true
}

func (d *SQLiteDB) Save(abv string, url string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	sqlStmt := `
		INSERT INTO short_urls (abbreviation, url, hits)
		VALUES (?, ?, 0)
		ON CONFLICT (abbreviation) DO NOTHING
	`

	result, err := d.db.Exec(sqlStmt, abv, url)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil // Treat duplicate as success
		}
		return fmt.Errorf("couldn't store (%s, %s): %v", abv, url, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Check if it was a conflict on abbreviation vs url
		var existingUrl string
		err := d.db.QueryRow("SELECT url FROM short_urls WHERE abbreviation = ?", abv).Scan(&existingUrl)
		if err == nil && existingUrl != url {
			return fmt.Errorf("abbreviation %s already exists with different URL", abv)
		}
	}

	return nil
}

func (d *SQLiteDB) DeleteAbv(abv string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	sqlStmt := `DELETE FROM short_urls WHERE abbreviation = ?`
	if _, err := d.db.Exec(sqlStmt, abv); err != nil {
		return fmt.Errorf("couldn't delete abbreviation %s: %v", abv, err)
	}
	return nil
}

func (d *SQLiteDB) DeleteUrl(url string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	sqlStmt := `DELETE FROM short_urls WHERE url = ?`
	if _, err := d.db.Exec(sqlStmt, url); err != nil {
		return fmt.Errorf("couldn't delete URL %s: %v", url, err)
	}
	return nil
}

func (d *SQLiteDB) GetUrl(abv string) (string, error) {
	d.mu.RLock()
	var url string
	var shortUrlId int
	sqlStmt := `SELECT id, url FROM short_urls WHERE abbreviation = ?`
	err := d.db.QueryRow(sqlStmt, abv).Scan(&shortUrlId, &url)
	d.mu.RUnlock()

	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("error getting URL for %s: %v", abv, err)
	}

	// Update stats asynchronously
	go func() {
		d.mu.Lock()
		defer d.mu.Unlock()

		// Update total hits and last_access in short_urls
		updateSQL := `
			UPDATE short_urls
			SET hits = hits + 1,
				last_access = CURRENT_TIMESTAMP
			WHERE id = ?
		`
		if _, err := d.db.Exec(updateSQL, shortUrlId); err != nil {
			log.Printf("Error updating short_urls stats: %v", err)
		}

		// Insert or update daily hit count
		dailyHitSQL := `
			INSERT INTO daily_hits (short_url_id, hit_date, hits)
			VALUES (?, DATE('now'), 1)
			ON CONFLICT (short_url_id, hit_date)
			DO UPDATE SET hits = daily_hits.hits + 1
		`
		if _, err := d.db.Exec(dailyHitSQL, shortUrlId); err != nil {
			log.Printf("Error updating daily_hits: %v", err)
		}
	}()

	return url, nil
}

func (d *SQLiteDB) GetAbv(url string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var abv string
	sqlStmt := `SELECT abbreviation FROM short_urls WHERE url = ?`
	err := d.db.QueryRow(sqlStmt, url).Scan(&abv)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("no abbreviation found for URL %s", url)
			return "", nil
		}
		return "", fmt.Errorf("error getting abbreviation for %s: %v", url, err)
	}

	return abv, nil
}

func (d *SQLiteDB) GetStats(abv string) (ShortUrl, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var data ShortUrl
	var shortUrlId int
	var lastAccess sql.NullTime

	// Get main short_url data
	sqlStmt := `
		SELECT id, abbreviation, url, hits, last_access
		FROM short_urls
		WHERE abbreviation = ?
	`
	err := d.db.QueryRow(sqlStmt, abv).Scan(
		&shortUrlId,
		&data.Abbreviation,
		&data.Url,
		&data.Hits,
		&lastAccess,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("no stats found for %s", abv)
			return ShortUrl{}, nil
		}
		return ShortUrl{}, fmt.Errorf("error getting stats for %s: %v", abv, err)
	}

	if lastAccess.Valid {
		data.LastAccess = lastAccess.Time
	}

	// Get daily hits from separate table
	data.DailyHits = make(map[string]int)
	dailyHitsSQL := `
		SELECT hit_date, hits
		FROM daily_hits
		WHERE short_url_id = ?
		ORDER BY hit_date DESC
	`
	rows, err := d.db.Query(dailyHitsSQL, shortUrlId)
	if err != nil {
		log.Printf("Error querying daily_hits: %v", err)
		return data, nil
	}
	defer rows.Close()

	for rows.Next() {
		var hitDate time.Time
		var hits int
		if err := rows.Scan(&hitDate, &hits); err != nil {
			log.Printf("Error scanning daily_hits row: %v", err)
			continue
		}
		data.DailyHits[hitDate.Format("2006-01-02")] = hits
	}

	return data, nil
}
