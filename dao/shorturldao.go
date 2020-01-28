package dao

type ShortUrlDao interface {
	IsLikelyOk() bool
	Save(abv string, url string) error
	DeleteAbv(abv string) error
	DeleteUrl(url string) error
	GetUrl(abv string) (string, error)
	GetAbv(url string) (string, error)
	Cleanup()
}
