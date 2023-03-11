package dao

import "time"

type MemoryDB struct {
	urlNdxMap map[string]ShortUrl
	abvNdxMap map[string]ShortUrl
}

func CreateMemoryDB() ShortUrlDao {
	return &MemoryDB{urlNdxMap: map[string]ShortUrl{}, abvNdxMap: map[string]ShortUrl{}}
}

func (d *MemoryDB) IsLikelyOk() bool {
	return true
}

func (d *MemoryDB) Save(abv string, url string) error {
	su := ShortUrl{Abbreviation: abv, Url: url, Hits: 0, DailyHits: map[string]int{}}
	d.urlNdxMap[url] = su
	d.abvNdxMap[abv] = su
	return nil
}

func (d *MemoryDB) DeleteAbv(abv string) error {
	su := d.abvNdxMap[abv]
	delete(d.abvNdxMap, abv)
	delete(d.urlNdxMap, su.Url)
	return nil
}

func (d *MemoryDB) DeleteUrl(url string) error {
	su := d.urlNdxMap[url]
	delete(d.abvNdxMap, su.Abbreviation)
	delete(d.urlNdxMap, url)
	return nil
}

func (d *MemoryDB) GetUrl(abv string) (string, error) {
	su, ok := d.abvNdxMap[abv]
	if ok && len(su.Url) > 0 {
		i := su.Hits
		su.Hits = i + 1
		su.LastAccess = time.Now()
		date := Date()
		su.DailyHits[date] = su.DailyHits[date] + 1
		return su.Url, nil
	}
	return "", nil
}

func (d *MemoryDB) GetAbv(url string) (string, error) {
	return d.urlNdxMap[url].Abbreviation, nil
}

func (d *MemoryDB) GetStats(abv string) (ShortUrl, error) {
	return d.abvNdxMap[abv], nil
}

func (d *MemoryDB) Cleanup() {
	// no op
}
