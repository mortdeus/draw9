package draw9

import (
	"testing"
)

var (
	atoitest = map[string]int{
		"        124 ": 124,
	}
)

func TestAtoi(t *testing.T) {
	for in, out := range atoitest {
		res := atoi([]byte(in))
		if res != out {
			t.Errorf("atoi(%q) expected %d got %d", in, out, res)
		}
	}
}
