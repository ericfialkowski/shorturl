package dao

import "time"

type ShortUrl struct {
	Abbreviation string    `json:"abbreviation" bson:"abv"`
	Url          string    `json:"url" bson:"url"`
	Hits         int32     `json:"hits" bson:"hits"`
	LastAccess   time.Time `json:"last_access" bson:"last_access,omitempty"`
}
