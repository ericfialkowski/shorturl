package dao

type MemoryDB struct {
	db map[string]bool
}

func CreateMemoryDB() MemoryDB {
	return MemoryDB{db: map[string]bool{}}
}

func (d *MemoryDB) IsLikelyOk() bool {
	return true
}

func (d *MemoryDB) Save(app string) error {
	d.db[app] = true
	return nil
}

func (d *MemoryDB) Delete(app string) error {
	delete(d.db, app)
	return nil
}
func (d *MemoryDB) Exists(app string) (bool, error) {
	return d.db[app], nil
}

func (d *MemoryDB) List() ([]string, error) {
	rtn := make([]string, 0)
	for app, val := range d.db {
		if val {
			rtn = append(rtn, app)
		}
	}
	return rtn, nil
}

func (d *MemoryDB) Cleanup() {
	// no op
}
