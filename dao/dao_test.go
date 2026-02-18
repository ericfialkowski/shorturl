package dao

import (
	"testing"
	"time"
)

// DAOTestSuite runs the same tests against any ShortUrlDao implementation
func runDAOTests(t *testing.T, name string, createDAO func() ShortUrlDao) {
	t.Run(name, func(t *testing.T) {
		t.Run("Save and GetUrl", func(t *testing.T) {
			dao := createDAO()
			defer dao.Cleanup()

			err := dao.Save("abc", "https://example.com")
			if err != nil {
				t.Fatalf("Save() error = %v", err)
			}

			url, err := dao.GetUrl("abc")
			if err != nil {
				t.Fatalf("GetUrl() error = %v", err)
			}
			if url != "https://example.com" {
				t.Errorf("GetUrl() = %v, want %v", url, "https://example.com")
			}
		})

		t.Run("Save and GetAbv", func(t *testing.T) {
			dao := createDAO()
			defer dao.Cleanup()

			err := dao.Save("xyz", "https://test.com")
			if err != nil {
				t.Fatalf("Save() error = %v", err)
			}

			abv, err := dao.GetAbv("https://test.com")
			if err != nil {
				t.Fatalf("GetAbv() error = %v", err)
			}
			if abv != "xyz" {
				t.Errorf("GetAbv() = %v, want %v", abv, "xyz")
			}
		})

		t.Run("GetUrl returns empty for missing abbreviation", func(t *testing.T) {
			dao := createDAO()
			defer dao.Cleanup()

			url, err := dao.GetUrl("nonexistent")
			if err != nil {
				t.Fatalf("GetUrl() error = %v", err)
			}
			if url != "" {
				t.Errorf("GetUrl() = %v, want empty string", url)
			}
		})

		t.Run("GetAbv returns empty for missing URL", func(t *testing.T) {
			dao := createDAO()
			defer dao.Cleanup()

			abv, err := dao.GetAbv("https://nonexistent.com")
			if err != nil {
				t.Fatalf("GetAbv() error = %v", err)
			}
			if abv != "" {
				t.Errorf("GetAbv() = %v, want empty string", abv)
			}
		})

		t.Run("DeleteAbv", func(t *testing.T) {
			dao := createDAO()
			defer dao.Cleanup()

			_ = dao.Save("del1", "https://delete1.com")

			err := dao.DeleteAbv("del1")
			if err != nil {
				t.Fatalf("DeleteAbv() error = %v", err)
			}

			url, _ := dao.GetUrl("del1")
			if url != "" {
				t.Errorf("After DeleteAbv(), GetUrl() = %v, want empty", url)
			}
		})

		t.Run("DeleteUrl", func(t *testing.T) {
			dao := createDAO()
			defer dao.Cleanup()

			_ = dao.Save("del2", "https://delete2.com")

			err := dao.DeleteUrl("https://delete2.com")
			if err != nil {
				t.Fatalf("DeleteUrl() error = %v", err)
			}

			abv, _ := dao.GetAbv("https://delete2.com")
			if abv != "" {
				t.Errorf("After DeleteUrl(), GetAbv() = %v, want empty", abv)
			}
		})

		t.Run("IsLikelyOk", func(t *testing.T) {
			dao := createDAO()
			defer dao.Cleanup()

			if !dao.IsLikelyOk() {
				t.Error("IsLikelyOk() = false, want true")
			}
		})

		t.Run("GetStats returns empty for missing abbreviation", func(t *testing.T) {
			dao := createDAO()
			defer dao.Cleanup()

			stats, err := dao.GetStats("nonexistent")
			if err != nil {
				t.Fatalf("GetStats() error = %v", err)
			}
			if stats.Abbreviation != "" {
				t.Errorf("GetStats().Abbreviation = %v, want empty", stats.Abbreviation)
			}
		})

		t.Run("GetStats returns correct data", func(t *testing.T) {
			dao := createDAO()
			defer dao.Cleanup()

			_ = dao.Save("stats1", "https://stats.com")

			stats, err := dao.GetStats("stats1")
			if err != nil {
				t.Fatalf("GetStats() error = %v", err)
			}
			if stats.Abbreviation != "stats1" {
				t.Errorf("GetStats().Abbreviation = %v, want %v", stats.Abbreviation, "stats1")
			}
			if stats.Url != "https://stats.com" {
				t.Errorf("GetStats().Url = %v, want %v", stats.Url, "https://stats.com")
			}
		})

		t.Run("Multiple saves and retrieves", func(t *testing.T) {
			dao := createDAO()
			defer dao.Cleanup()

			urls := map[string]string{
				"a1": "https://one.com",
				"b2": "https://two.com",
				"c3": "https://three.com",
			}

			for abv, url := range urls {
				if err := dao.Save(abv, url); err != nil {
					t.Fatalf("Save(%s, %s) error = %v", abv, url, err)
				}
			}

			for abv, expectedUrl := range urls {
				url, err := dao.GetUrl(abv)
				if err != nil {
					t.Fatalf("GetUrl(%s) error = %v", abv, err)
				}
				if url != expectedUrl {
					t.Errorf("GetUrl(%s) = %v, want %v", abv, url, expectedUrl)
				}
			}
		})
	})
}

func TestMemoryDB(t *testing.T) {
	runDAOTests(t, "MemoryDB", func() ShortUrlDao {
		return CreateMemoryDB()
	})
}

func TestSQLiteDB(t *testing.T) {
	runDAOTests(t, "SQLiteDB", func() ShortUrlDao {
		return CreateSQLiteDB(":memory:")
	})
}

func TestSQLiteDB_HitTracking(t *testing.T) {
	dao := CreateSQLiteDB(":memory:")
	defer dao.Cleanup()

	_ = dao.Save("hit1", "https://hittrack.com")

	// Access the URL multiple times
	for range 5 {
		_, _ = dao.GetUrl("hit1")
	}

	// Give async updates time to complete
	time.Sleep(100 * time.Millisecond)

	stats, err := dao.GetStats("hit1")
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	if stats.Hits < 5 {
		t.Errorf("GetStats().Hits = %v, want at least 5", stats.Hits)
	}
}

func TestDate(t *testing.T) {
	result := Date()
	expected := time.Now().Format("2006-01-02")
	if result != expected {
		t.Errorf("Date() = %v, want %v", result, expected)
	}
}
