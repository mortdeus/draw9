package image9

import (
	"bitbucket.org/mischief/draw9/color9"
	"fmt"
	"image"
	"image/color"
	"io"
)

func Encode(w io.Writer, i image.Image) error {
	cm := color.RGBAModel
	pix := color9.RGBA32
	r := i.Bounds()
	fmt.Fprintf(w, "%11s %11d %11d %11d %11d ", pix, r.Min.X, r.Min.Y, r.Max.X, r.Max.Y)

	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			colr := cm.Convert(i.At(x, y))
			r, g, b, a := colr.RGBA()

			w.Write([]byte{uint8(a), uint8(b), uint8(g), uint8(r)})
		}
	}
	return nil
}
