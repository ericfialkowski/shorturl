package dao

import "time"

type ShortUrl struct {
	Abbreviation string    `json:"abbreviation"`
	Url          string    `json:"url"`
	Hits         int32     `json:"hits"`
	LastAccess   time.Time `json:"last_access"`
}
