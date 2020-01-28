package rando

import (
	"math/rand"
	"strings"
)

const chars string = "abcdefghijklmnopqrstuvwxyz0123456789"

func RandStrn(length int) string {
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteByte(chars[rand.Intn(len(chars))])
	}
	return b.String()
}
