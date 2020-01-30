package dao

import (
	"fmt"
	"shorturl/rando"
)

func CreateAbbreviation(url string, dao ShortUrlDao) (string, error) {
	u := ""

	abv := rando.RandStrn(5)
	u, err := dao.GetUrl(abv)
	for len(u) != 0 {
		u, err = dao.GetUrl(abv) // TODO: handle error
		if err != nil {
			return "", fmt.Errorf("error checking abbeviation %v", err)
		}
		abv = rando.RandStrn(5)
	}

	return abv, nil
}
