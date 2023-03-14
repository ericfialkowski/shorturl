package dao

import (
	"fmt"
	"log"
	"shorturl/env"
	"shorturl/rando"
)

var keySize = env.IntOrDefault("startingkeysize", 1)

func randString() string {
	tries := 0
	for {
		s := rando.RandStrn(keySize)
		if AcceptableWord(s) {
			return s
		}
		// if we haven't found a good word in a certain number of tries, we need to grow the keysize for more randomness
		if tries = tries + 1; tries > env.IntOrDefault("keygrowretries", 10) {
			tries = 0
			keySize = keySize + 1
			log.Printf("Growing keySize to be %d", keySize)
		}
	}
}

func CreateAbbreviation(url string, dao ShortUrlDao) (string, error) {
	tries := 0
	abv := randString()
	u, _ := dao.GetUrl(abv)
	for len(u) != 0 && url != u {
		// if we haven't found a good word in a certain number of tries, we need to grow the keysize for more randomness
		if tries = tries + 1; tries > env.IntOrDefault("keygrowretries", 10) {
			tries = 0
			keySize = keySize + 1
			log.Printf("Growing keySize to be %d", keySize)
		}
		_, err := dao.GetUrl(abv)
		if err != nil {
			return "", fmt.Errorf("error checking abbreviation %v", err)
		}
		abv = randString()
		u, _ = dao.GetUrl(abv)
	}

	return abv, nil
}
