package draw9

import (
	"testing"
	"time"
)

// try me!
// go test -v -run InitDraw

func TestInitDraw(t *testing.T) {
	disp, err := InitDraw(nil, "", "test")
	if err != nil {
		t.Fatal(err)
	}

	//defer disp.Close()

	kbd := InitKeyboard("")
	//defer kbd.Close()

	ms := InitMouse("", disp.ScreenImage)
	//defer ms.Close()

loop:
	for {
		select {
		case r := <-kbd.C:
			t.Logf("kbd: %c", r)
		case m := <-ms.C:
			t.Logf("ms: %+v", m)
		case <-time.After(1 * time.Second):
			t.Logf("timeout")
			break loop
		}
	}

	// closing mouse file explicitly seems to break
	// restoring the old window content in rio.

	//ms.Close()
	kbd.Close()
	disp.Close()
}
