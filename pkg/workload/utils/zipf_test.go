package utils

import (
	"fmt"
	"testing"
)

func TestZipf(t *testing.T) {
	account := 1000
	num := 10000
	zipf := NewZipf(account, 1)
	cnt := make([]float64, account)
	for i := 0; i < num; i++ {
		cnt[zipf.Generate()] += 1
	}
	for i := 0; i < account; i++ {
		cnt[i] /= float64(num)
	}
	for i := 0; i < 20; i++ {
		fmt.Println(cnt[i])
	}

}
