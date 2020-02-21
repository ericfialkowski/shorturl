package dao

type ShortUrl struct {
	Abbreviation string `json:"abbreviation"`
	Url          string `json:"url"`
	Hits         int    `json:"hits"`
}
