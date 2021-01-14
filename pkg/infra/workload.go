package infra

import (
	// "fmt"
	"math/rand"
	"time"
)


var chs = []rune("qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM1234567890!@#$%^&*()_=")


func getName(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = chs[rand.Intn(len(chs))]
	}
	return string(b)
}

func generate() []string {
	// idx := getName(64)
	// res := fmt.Sprintf("{\"Args\":[\"create_account\", \"%s\", \"%s\", \"1000000\", \"1000000\"]}", idx, idx)
	var res []string
	res = append(res, "CreateAccount")
	res = append(res, "1217")
	res = append(res, "Bob")
	res = append(res, "100")
	res = append(res, "90")
	// res = append(res, "Query")
	// res = append(res, "1217")

	// res = append(res, "CreateCar")
	// res = append(res, idx)	
	// res = append(res, "china")
	// res = append(res, "tesla")
	// res = append(res, "bluegreen")
	// res = append(res, "yunpeng")
	return res
}

func GenerateWorkload(n int) [][]string {
	rand.Seed(time.Now().UnixNano())
	i := 0
	var res [][]string
	for i = 0; i < n; i++ {
		res = append(res, generate())
	}
	return res
}
