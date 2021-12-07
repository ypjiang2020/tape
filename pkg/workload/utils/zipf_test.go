package utils

import (
	"fmt"
	"testing"
)

func TestZipf(t *testing.T) {
	account := 1000
	num := 10000
	zipf := NewZipf(account, 1)
	done := make(chan struct{})
	cnt := make([][]float64, 10)

	f := func(id int) {
		cnt[id] = make([]float64, account)
		for i := 0; i < num; i++ {
			cnt[id][zipf.Generate()] += 1
		}
		for i := 0; i < account; i++ {
			cnt[id][i] /= float64(num)
		}
		done <- struct{}{}
	}

	for i := 0; i < 10; i++ {
		go f(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}

	res := make([]float64, account)

	for j := 0; j < account; j++ {
		for i := 0; i < 10; i++ {
			res[j] += cnt[i][j]
		}
		res[j] /= 10
	}

	for i := 0; i < 20; i++ {
		fmt.Println(res[i])
	}

}
