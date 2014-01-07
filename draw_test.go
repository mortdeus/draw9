package draw9

import (
	"image"
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

	img := disp.ScreenImage

	//defer disp.Close()

	kbd := InitKeyboard("")
	//defer kbd.Close()

	ms := InitMouse("", disp.ScreenImage)
	//defer ms.Close()

loop:
	for {
		select {
		case r := <-kbd.C:
			// keyboard char
			t.Logf("kbd: %c", r)
		case m := <-ms.C:
			// mouse move
			t.Logf("ms: %s", m)
			pt := m.Pt()
			img.Draw(image.Rect(pt.X-5, pt.Y-5, pt.X+5, pt.Y+5), disp.Black, nil, image.ZP)
			disp.Flush()
		case <-ms.Resize:
			// resized
			if err := disp.Attach(Refmesg); err != nil {
				t.Errorf("attach: %s", err)
				break loop
			}
			img = disp.ScreenImage
			img.Draw(disp.ScreenImage.R, disp.White, nil, image.ZP)
			disp.Flush()
		case <-time.After(5 * time.Second):
			// timeout, die
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
