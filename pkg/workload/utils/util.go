package utils

import "math/rand"

var chs = []rune("qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM1234567890")

func GetName(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = chs[rand.Intn(len(chs))]
	}
	return string(b)
}

func RandomId(n int) int {
	res := rand.Intn(n)
	return res
}

func RandomInRange(min, max int) int {
	return rand.Intn(max-min) + min
}
