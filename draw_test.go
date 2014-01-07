package draw9

import (
	"image"
	"testing"
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

	//var pix []image.Point

loop:
	for {
		select {
		case r := <-kbd.C:
			// keyboard char
			t.Logf("kbd: %c", r)
			if r == 'q' || r == Kdel {
				break loop
			}
		case m := <-ms.C:
			// mouse move
			t.Logf("ms: %s", m)
			if m.Mb1() {
				var newpt image.Point
				pt := m.Pt()
				for m2 := range ms.C {
					if !m2.Mb1() {
						break
					}
					newpt = m2.Pt()
					if newpt == pt {
						continue
					}
					img.Line(pt, newpt, 0, 0, 3, disp.Black, image.ZP)
					disp.Flush()
					pt = newpt
				}
				//pix = append(pix, pt)
				//img.Draw(image.Rect(pt.X-5, pt.Y-5, pt.X+5, pt.Y+5), disp.Black, nil, image.ZP)
			} else if m.Mb3() {
				//pix = nil
				img.Draw(disp.ScreenImage.R, disp.White, nil, image.ZP)
			}
			disp.Flush()
		case <-ms.Resize:
			// resized
			if err := disp.Attach(Refmesg); err != nil {
				t.Errorf("attach: %s", err)
				break loop
			}
			img = disp.ScreenImage
			img.Draw(disp.ScreenImage.R, disp.White, nil, image.ZP)
			//for _, pt := range pix {
			//	img.Draw(image.Rect(pt.X-5, pt.Y-5, pt.X+5, pt.Y+5), disp.Black, nil, image.ZP)
			//}
			disp.Flush()
		}
	}

	// closing mouse file explicitly seems to break
	// restoring the old window content in rio.

	//ms.Close()
	kbd.Close()
	disp.Close()
}
