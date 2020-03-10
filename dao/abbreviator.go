package dao

import (
	"fmt"
	"shorturl/environment"
	"shorturl/rando"
)

func randString() string {
	for {
		s := rando.RandStrn(environment.GetEnvIntOrDefault("keysize", 5))
		if !BadWord(s) {
			return s
		}
	}
}

func CreateAbbreviation(url string, dao ShortUrlDao) (string, error) {
	abv := randString()

	u, _ := dao.GetUrl(abv)
	for len(u) != 0 && url != u {
		_, err := dao.GetUrl(abv)
		if err != nil {
			return "", fmt.Errorf("error checking abbreviation %v", err)
		}
		abv = randString()
		u, _ = dao.GetUrl(abv)
	}

	return abv, nil
}
