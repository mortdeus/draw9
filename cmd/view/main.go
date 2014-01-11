package main

import (
	"bitbucket.org/mischief/draw9"
	"bytes"
	"flag"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
)

var file = flag.String("f", "", "file to view")

func main() {
	flag.Parse()
	disp, err := draw9.InitDraw(nil, "", "view")
	if err != nil {
		log.Fatal(err)
	}

	screen := disp.ScreenImage
	kbd := draw9.InitKeyboard("")
	ms := draw9.InitMouse("", disp.ScreenImage)

	var pic image.Image

	if *file == "" {
		pic, _, err = image.Decode(os.Stdin)
	} else {
		buf := new(bytes.Buffer)
		f, err := os.Open(*file)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		buf.ReadFrom(f)
		pic, _, err = image.Decode(buf)
	}

	if err != nil {
		log.Fatal(err)
	}

	drawpic, err := disp.LoadImage(pic)

	if err != nil {
		log.Fatal(err)
	}

	screen.Draw(disp.ScreenImage.R, drawpic, nil, image.ZP)

loop:
	for {
		select {
		case r := <-kbd.C:
			// keyboard char
			if r == 'q' || r == draw9.Kdel {
				break loop
			}
		case <-ms.C:
			// mouse move
			/*
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
						img.Line(pt, newpt, 1, 1, 3, disp.Black, image.ZP)
						disp.Flush()
						pt = newpt
					}
					//pix = append(pix, pt)
					//img.Draw(image.Rect(pt.X-5, pt.Y-5, pt.X+5, pt.Y+5), disp.Black, nil, image.ZP)
				} else if m.Mb3() {
					//pix = nil
					img.Draw(disp.ScreenImage.R, disp.White, nil, image.ZP)
				}
			*/
			disp.Flush()
		case <-ms.Resize:
			// resized
			if err := disp.Attach(draw9.Refmesg); err != nil {
				log.Printf("attach: %s", err)
				break loop
			}
			screen = disp.ScreenImage
			screen.Draw(disp.ScreenImage.R, drawpic, nil, image.ZP)
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
