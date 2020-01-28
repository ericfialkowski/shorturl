package dao

type MemoryDB struct {
	urlNdxMap map[string]string
	abvNdxMap map[string]string
}

func CreateMemoryDB() MemoryDB {
	return MemoryDB{urlNdxMap: map[string]string{}, abvNdxMap: map[string]string{}}
}

func (d *MemoryDB) IsLikelyOk() bool {
	return true
}

func (d *MemoryDB) Save(abv string, url string) error {
	d.urlNdxMap[url] = abv
	d.abvNdxMap[abv] = url
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
	return d.abvNdxMap[abv], nil
}

func (d *MemoryDB) GetAbv(url string) (string, error) {
	return d.urlNdxMap[url], nil
}

func (d *MemoryDB) Cleanup() {
	// no op
}
