package utils

import (
	"math/rand/v2"
	"strings"
)

const (
	alphabet = "abcdefghijklmnopqrstuvwxyz"
)

var currencies = []string{"USD", "EUR", "CAD"}

func RandomInt(min, max int64) int64 {
	return rand.Int64N(max-min+1) + min
}

func RandomString(n int) string {
	var b strings.Builder
	for range n {
		b.WriteByte(alphabet[rand.IntN(len(alphabet))])
	}
	return b.String()
}

func RandomOwner() string {
	return RandomString(6)
}

func RandomUsername() string {
	return RandomString(6)
}

func RandomFullName() string {
	return RandomString(6) + " " + RandomString(6)
}

func RandomEmail() string {
	return RandomString(6) + "@example.com"
}

func RandomMoney() int64 {
	return RandomInt(0, 1000)
}

func RandomCurrency() string {
	return currencies[rand.IntN(len(currencies))]
}
