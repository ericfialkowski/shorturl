package dao

import (
	"sync"
	"time"
)

type MemoryDB struct {
	mu        sync.RWMutex
	urlNdxMap map[string]*ShortUrl
	abvNdxMap map[string]*ShortUrl
}

func CreateMemoryDB() ShortUrlDao {
	return &MemoryDB{
		urlNdxMap: make(map[string]*ShortUrl),
		abvNdxMap: make(map[string]*ShortUrl),
	}
}

func (d *MemoryDB) IsLikelyOk() bool {
	return true
}

func (d *MemoryDB) Save(abv string, url string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	su := &ShortUrl{
		Abbreviation: abv,
		Url:          url,
		Hits:         0,
		DailyHits:    make(map[string]int),
	}
	d.urlNdxMap[url] = su
	d.abvNdxMap[abv] = su
	return nil
}

func (d *MemoryDB) DeleteAbv(abv string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	su, ok := d.abvNdxMap[abv]
	if ok {
		delete(d.urlNdxMap, su.Url)
		delete(d.abvNdxMap, abv)
	}
	return nil
}

func (d *MemoryDB) DeleteUrl(url string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	su, ok := d.urlNdxMap[url]
	if ok {
		delete(d.abvNdxMap, su.Abbreviation)
		delete(d.urlNdxMap, url)
	}
	return nil
}

func (d *MemoryDB) GetUrl(abv string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	su, ok := d.abvNdxMap[abv]
	if ok && len(su.Url) > 0 {
		su.Hits++
		su.LastAccess = time.Now()
		date := Date()
		su.DailyHits[date]++
		return su.Url, nil
	}
	return "", nil
}

func (d *MemoryDB) GetAbv(url string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	su, ok := d.urlNdxMap[url]
	if ok {
		return su.Abbreviation, nil
	}
	return "", nil
}

func (d *MemoryDB) GetStats(abv string) (ShortUrl, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	su, ok := d.abvNdxMap[abv]
	if ok {
		// Return a copy to avoid external modifications
		return *su, nil
	}
	return ShortUrl{}, nil
}

func (d *MemoryDB) Cleanup() {
	// no op
}
