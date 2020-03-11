package dao

import "time"

type ShortUrlDao interface {
	IsLikelyOk() bool
	Save(abv string, url string) error
	DeleteAbv(abv string) error
	DeleteUrl(url string) error
	GetUrl(abv string) (string, error) // TODO: make new method that doesn't update stats on a "hit"
	GetAbv(url string) (string, error)
	GetStats(abv string) (ShortUrl, error)
	Cleanup()
}

func Date() string {
	return time.Now().Format("2006-01-02")
}
