package smallbank

import "testing"

func Test(t *testing.T) {
	gen := NewSmallBank(119, 20, 0.01, 1.0, ".")
	gen.Init()
	gen.GenerateTransaction()
}
