package dao

type ShortUrlDao interface {
	IsLikelyOk() bool
	Save(abv string, url string) error
	DeleteAbv(abv string) error
	DeleteUrl(url string) error
	Get(abv string) (string, error)
	Cleanup()
}
