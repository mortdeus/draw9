package draw9

import (
	"testing"
)

func TestKeyboard(t *testing.T) {
	kbd := InitKeyboard("")
	defer kbd.Close()

	println("q to end test")

	for r := range kbd.C {
		t.Logf("got rune '%c'", r)
		if r == 'q' {
			break
		}
	}
}
