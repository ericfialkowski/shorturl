package dao

import (
	"fmt"
	"shorturl/environment"
	"shorturl/rando"
)

func randString() string {
	return rando.RandStrn(environment.GetEnvIntOrDefault("keysize", 5))
}

func CreateAbbreviation(url string, dao ShortUrlDao) (string, error) {
	abv := randString()
	for BadWord(abv) {
		abv = randString()
	}

	u, err := dao.GetUrl(abv)
	for len(u) != 0 && url != u {
		u, err = dao.GetUrl(abv) // TODO: handle error
		if err != nil {
			return "", fmt.Errorf("error checking abbeviation %v", err)
		}
		abv = randString()
	}

	return abv, nil
}
