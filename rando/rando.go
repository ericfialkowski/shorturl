package rando

import (
	"math/rand"
	"strings"
	"time"
)

const chars string = "abcdefghijklmnopqrstuvwxyz0123456789"

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func RandStrn(length int) string {
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteByte(chars[r.Intn(len(chars))])
	}
	return b.String()
}
