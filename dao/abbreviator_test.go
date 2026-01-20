package dao

import (
	"testing"
)

func TestCreateAbbreviation(t *testing.T) {
	dao := CreateMemoryDB()
	defer dao.Cleanup()

	abv, err := CreateAbbreviation("https://example.com", dao)
	if err != nil {
		t.Fatalf("CreateAbbreviation() error = %v", err)
	}

	if abv == "" {
		t.Error("CreateAbbreviation() returned empty string")
	}

	if !AcceptableWord(abv) {
		t.Errorf("CreateAbbreviation() returned unacceptable word: %s", abv)
	}
}

func TestCreateAbbreviation_UniquePerURL(t *testing.T) {
	dao := CreateMemoryDB()
	defer dao.Cleanup()

	url1 := "https://example1.com"
	url2 := "https://example2.com"

	abv1, err := CreateAbbreviation(url1, dao)
	if err != nil {
		t.Fatalf("CreateAbbreviation() error = %v", err)
	}
	_ = dao.Save(abv1, url1)

	abv2, err := CreateAbbreviation(url2, dao)
	if err != nil {
		t.Fatalf("CreateAbbreviation() error = %v", err)
	}

	if abv1 == abv2 {
		t.Errorf("CreateAbbreviation() returned same abbreviation for different URLs: %s", abv1)
	}
}

func TestCreateAbbreviation_AvoidsCollision(t *testing.T) {
	dao := CreateMemoryDB()
	defer dao.Cleanup()

	// Save several URLs first
	for i := 0; i < 10; i++ {
		abv, err := CreateAbbreviation("https://test"+string(rune('a'+i))+".com", dao)
		if err != nil {
			t.Fatalf("CreateAbbreviation() error = %v", err)
		}
		_ = dao.Save(abv, "https://test"+string(rune('a'+i))+".com")
	}

	// Create another and ensure it doesn't collide
	newAbv, err := CreateAbbreviation("https://new.com", dao)
	if err != nil {
		t.Fatalf("CreateAbbreviation() error = %v", err)
	}

	// The new abbreviation should not exist in the DAO yet
	existingURL, _ := dao.GetUrl(newAbv)
	if existingURL != "" {
		t.Errorf("CreateAbbreviation() returned existing abbreviation: %s -> %s", newAbv, existingURL)
	}
}

func TestCreateAbbreviation_ReturnsAcceptableWords(t *testing.T) {
	dao := CreateMemoryDB()
	defer dao.Cleanup()

	// Generate many abbreviations and verify they're all acceptable
	for i := 0; i < 50; i++ {
		abv, err := CreateAbbreviation("https://test"+string(rune(i))+".com", dao)
		if err != nil {
			t.Fatalf("CreateAbbreviation() error = %v", err)
		}
		if !AcceptableWord(abv) {
			t.Errorf("CreateAbbreviation() returned unacceptable word: %s", abv)
		}
	}
}

func BenchmarkCreateAbbreviation(b *testing.B) {
	dao := CreateMemoryDB()
	defer dao.Cleanup()

	for i := 0; i < b.N; i++ {
		_, _ = CreateAbbreviation("https://benchmark.com/"+string(rune(i)), dao)
	}
}
