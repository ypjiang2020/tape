package utils

import (
	"math"
	"math/rand"
)

type Zipf struct {
	accountNumber int
	zipfs         float64

	probability []float64
}

func NewZipf(accout int, zipfs float64) *Zipf {
	zipf := &Zipf{
		accountNumber: accout,
		zipfs:         -zipfs,
		probability: make([]float64, accout),
	}
	zipf.init()
	return zipf
}

func (zipf *Zipf) init() {
	sump := 0.0
	for i := 0; i < zipf.accountNumber; i++ {
		p := math.Pow(float64(1+i), zipf.zipfs)
		sump += p
		zipf.probability[i] = sump
	}
	for i := 0; i < zipf.accountNumber; i++ {
		zipf.probability[i] /= sump
	}
}

func (zipf *Zipf) Generate() int {
	p := rand.Float64()
	l, r := 0, zipf.accountNumber
	mid := -1
	for l < r {
		mid = (l+r) / 2
		if zipf.probability[mid] < p {
			l = mid+1
		} else {
			r = mid
		}
	}
	return l
}
