package dao

type MemoryDB struct {
	urlNdxMap map[string]string
	abvNdxMap map[string]string
	hitMap    map[string]int
}

func CreateMemoryDB() ShortUrlDao {
	return &MemoryDB{urlNdxMap: map[string]string{}, abvNdxMap: map[string]string{}, hitMap: map[string]int{}}
}

func (d *MemoryDB) IsLikelyOk() bool {
	return true
}

func (d *MemoryDB) Save(abv string, url string) error {
	d.urlNdxMap[url] = abv
	d.abvNdxMap[abv] = url
	d.hitMap[abv] = 0
	return nil
}

func (d *MemoryDB) DeleteAbv(abv string) error {
	url := d.abvNdxMap[abv]
	delete(d.abvNdxMap, abv)
	delete(d.urlNdxMap, url)
	return nil
}

func (d *MemoryDB) DeleteUrl(url string) error {
	abv := d.urlNdxMap[url]
	delete(d.abvNdxMap, abv)
	delete(d.urlNdxMap, url)
	return nil
}

func (d *MemoryDB) GetUrl(abv string) (string, error) {
	u := d.abvNdxMap[abv]
	if len(u) > 0 {
		i := d.hitMap[abv]
		d.hitMap[abv] = i + 1
	}
	return u, nil
}

func (d *MemoryDB) GetAbv(url string) (string, error) {
	return d.urlNdxMap[url], nil
}

func (d *MemoryDB) GetStats(abv string) (ShortUrl, error) {
	return ShortUrl{Abbreviation: abv, Url: d.abvNdxMap[abv], Hits: d.hitMap[abv]}, nil
}

func (d *MemoryDB) Cleanup() {
	// no op
}
