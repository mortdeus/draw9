package main

import (
	"bitbucket.org/mischief/draw9"
	"flag"
	"image"
	"log"
	"time"
)

func main() {
	flag.Parse()
	disp, err := draw9.InitDraw(nil, "", "gopaint")
	if err != nil {
		log.Fatal(err)
	}

	img := disp.ScreenImage

	kbd := draw9.InitCons("")
	ms := draw9.InitMouse("", disp.ScreenImage)

	var pix []image.Point
	timeout := time.After(9 * time.Minute)

loop:
	for {
		select {
		case <-timeout:
			break loop
		case r := <-kbd.C:
			// keyboard char
			if r == 'q' || r == draw9.Kdel {
				break loop
			}
		case m := <-ms.C:
			// mouse move
			if m.Mb1() {
				var newpt image.Point
				pt := m.Pt()
				pix = append(pix, pt)
				for m2 := range ms.C {
					if !m2.Mb1() {
						break
					}
					newpt = m2.Pt()
					if newpt == pt {
						continue
					}
					img.Line(pt, newpt, 1, 1, 3, disp.Black, image.ZP)
					disp.Flush()
					pt = newpt
					pix = append(pix, pt)
				}
			} else if m.Mb3() {
				//pix = nil
				img.Draw(disp.ScreenImage.R, disp.White, nil, image.ZP)
			}
			disp.Flush()
		case <-ms.Resize:
			// resized
			if err := disp.Attach(draw9.Refmesg); err != nil {
				log.Printf("attach: %s", err)
				break loop
			}
			img = disp.ScreenImage
			img.Draw(disp.ScreenImage.R, disp.White, nil, image.ZP)
			for n := 0; n < len(pix)-1; n++ {
				pt1 := pix[n]
				pt2 := pix[n+1]
				img.Line(pt1, pt2, 1, 1, 3, disp.Black, image.ZP)
			}
			disp.Flush()
		}
	}

	// closing mouse file explicitly seems to break
	// restoring the old window content in rio.

	//ms.Close()
	kbd.Close()
	disp.Close()
}
