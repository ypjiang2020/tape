package smallbank

import "testing"

func Test(t *testing.T) {
	gen := NewSmallBank()
	gen.Init()
	gen.GenerateTransaction()
}
