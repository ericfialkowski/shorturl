package dao

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ericfialkowski/shorturl/env"
	_ "github.com/go-sql-driver/mysql"
)

type MySQLDB struct {
	db *sql.DB
}

func newMySQLContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), env.DurationOrDefault("mysql_timeout", 10*time.Second))
}

// CreateMySQLDB creates a new MySQL-backed ShortUrlDao.
// The dsn should be a MySQL DSN string, e.g.:
// "user:password@tcp(localhost:3306)/shorturl?parseTime=true"
func CreateMySQLDB(dsn string) ShortUrlDao {
	// Ensure parseTime=true is set for proper time handling
	if !strings.Contains(dsn, "parseTime") {
		if strings.Contains(dsn, "?") {
			dsn += "&parseTime=true"
		} else {
			dsn += "?parseTime=true"
		}
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Unable to open MySQL database: %v", err)
	}

	db.SetMaxOpenConns(env.IntOrDefault("mysql_max_conns", 10))
	db.SetMaxIdleConns(env.IntOrDefault("mysql_max_idle_conns", 5))
	db.SetConnMaxLifetime(time.Duration(env.IntOrDefault("mysql_conn_max_lifetime_minutes", 5)) * time.Minute)

	ctx, cancel := newMySQLContext()
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Unable to connect to MySQL: %v", err)
	}

	mysqlDB := &MySQLDB{db: db}
	mysqlDB.initSchema()

	return mysqlDB
}

func (d *MySQLDB) initSchema() {
	ctx, cancel := newMySQLContext()
	defer cancel()

	// Create the main short_urls table
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS short_urls (
			id INT AUTO_INCREMENT PRIMARY KEY,
			abbreviation VARCHAR(50) NOT NULL UNIQUE,
			url TEXT NOT NULL,
			hits INT NOT NULL DEFAULT 0,
			last_access DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY idx_url (url(255))
		)
	`

	if _, err := d.db.ExecContext(ctx, createTableSQL); err != nil {
		log.Printf("Error creating short_urls table: %v", err)
	}

	// Create index on abbreviation
	createAbvIndex := `CREATE INDEX IF NOT EXISTS idx_short_urls_abbreviation ON short_urls(abbreviation)`
	if _, err := d.db.ExecContext(ctx, createAbvIndex); err != nil {
		// MySQL might not support IF NOT EXISTS for indexes in older versions, ignore error
		if !strings.Contains(err.Error(), "Duplicate key name") {
			log.Printf("Error creating abbreviation index: %v", err)
		}
	}

	// Create the daily_hits table for tracking hits per day
	createDailyHitsSQL := `
		CREATE TABLE IF NOT EXISTS daily_hits (
			id INT AUTO_INCREMENT PRIMARY KEY,
			short_url_id INT NOT NULL,
			hit_date DATE NOT NULL,
			hits INT NOT NULL DEFAULT 0,
			UNIQUE KEY idx_url_date (short_url_id, hit_date),
			FOREIGN KEY (short_url_id) REFERENCES short_urls(id) ON DELETE CASCADE
		)
	`

	if _, err := d.db.ExecContext(ctx, createDailyHitsSQL); err != nil {
		log.Printf("Error creating daily_hits table: %v", err)
	}
}

func (d *MySQLDB) Cleanup() {
	_ = d.db.Close()
}

func (d *MySQLDB) IsLikelyOk() bool {
	ctx, cancel := newMySQLContext()
	defer cancel()

	if err := d.db.PingContext(ctx); err != nil {
		log.Printf("Ping failed: %v", err)
		return false
	}
	return true
}

func (d *MySQLDB) Save(abv string, url string) error {
	ctx, cancel := newMySQLContext()
	defer cancel()

	sqlStmt := `INSERT IGNORE INTO short_urls (abbreviation, url, hits) VALUES (?, ?, 0)`

	result, err := d.db.ExecContext(ctx, sqlStmt, abv, url)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			return nil // Treat duplicate as success
		}
		return fmt.Errorf("couldn't store (%s, %s): %v", abv, url, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Check if it was a conflict on abbreviation vs url
		var existingUrl string
		err := d.db.QueryRowContext(ctx, "SELECT url FROM short_urls WHERE abbreviation = ?", abv).Scan(&existingUrl)
		if err == nil && existingUrl != url {
			return fmt.Errorf("abbreviation %s already exists with different URL", abv)
		}
	}

	return nil
}

func (d *MySQLDB) DeleteAbv(abv string) error {
	ctx, cancel := newMySQLContext()
	defer cancel()

	sqlStmt := `DELETE FROM short_urls WHERE abbreviation = ?`
	if _, err := d.db.ExecContext(ctx, sqlStmt, abv); err != nil {
		return fmt.Errorf("couldn't delete abbreviation %s: %v", abv, err)
	}
	return nil
}

func (d *MySQLDB) DeleteUrl(url string) error {
	ctx, cancel := newMySQLContext()
	defer cancel()

	sqlStmt := `DELETE FROM short_urls WHERE url = ?`
	if _, err := d.db.ExecContext(ctx, sqlStmt, url); err != nil {
		return fmt.Errorf("couldn't delete URL %s: %v", url, err)
	}
	return nil
}

func (d *MySQLDB) GetUrl(abv string) (string, error) {
	ctx, cancel := newMySQLContext()
	defer cancel()

	var url string
	var shortUrlId int
	sqlStmt := `SELECT id, url FROM short_urls WHERE abbreviation = ?`
	err := d.db.QueryRowContext(ctx, sqlStmt, abv).Scan(&shortUrlId, &url)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("error getting URL for %s: %v", abv, err)
	}

	// Update stats asynchronously
	go func() {
		ctx, cancel := newMySQLContext()
		defer cancel()

		// Update total hits and last_access in short_urls
		updateSQL := `UPDATE short_urls SET hits = hits + 1, last_access = NOW() WHERE id = ?`
		if _, err := d.db.ExecContext(ctx, updateSQL, shortUrlId); err != nil {
			log.Printf("Error updating short_urls stats: %v", err)
		}

		// Insert or update daily hit count
		dailyHitSQL := `
			INSERT INTO daily_hits (short_url_id, hit_date, hits)
			VALUES (?, CURDATE(), 1)
			ON DUPLICATE KEY UPDATE hits = hits + 1
		`
		if _, err := d.db.ExecContext(ctx, dailyHitSQL, shortUrlId); err != nil {
			log.Printf("Error updating daily_hits: %v", err)
		}
	}()

	return url, nil
}

func (d *MySQLDB) GetAbv(url string) (string, error) {
	ctx, cancel := newMySQLContext()
	defer cancel()

	var abv string
	sqlStmt := `SELECT abbreviation FROM short_urls WHERE url = ?`
	err := d.db.QueryRowContext(ctx, sqlStmt, url).Scan(&abv)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("no abbreviation found for URL %s", url)
			return "", nil
		}
		return "", fmt.Errorf("error getting abbreviation for %s: %v", url, err)
	}

	return abv, nil
}

func (d *MySQLDB) GetStats(abv string) (ShortUrl, error) {
	ctx, cancel := newMySQLContext()
	defer cancel()

	var data ShortUrl
	var shortUrlId int
	var lastAccess sql.NullTime

	// Get main short_url data
	sqlStmt := `
		SELECT id, abbreviation, url, hits, last_access
		FROM short_urls
		WHERE abbreviation = ?
	`
	err := d.db.QueryRowContext(ctx, sqlStmt, abv).Scan(
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
	rows, err := d.db.QueryContext(ctx, dailyHitsSQL, shortUrlId)
	if err != nil {
		log.Printf("Error querying daily_hits: %v", err)
		return data, nil
	}
	defer func() {
		_ = rows.Close()
	}()

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
