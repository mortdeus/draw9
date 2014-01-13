package draw9

import (
	"bitbucket.org/mischief/draw9/color9"
	"image"
	"image/color"
	"log"
	"runtime"
)

type Image struct {
	Display  *Display
	ID       uint32
	Pix      color9.Pix
	R, Clipr image.Rectangle
	Depth    int
	Chan     uint
	Repl     bool
	Next     *Image
	Screen   *Screen
}

// Free implements the 'f' draw(3) message.
func (i *Image) Free() error {
	if i == nil {
		return nil
	}
	if i.Display != nil && i == i.Display.ScreenImage {
		panic("freeimage of ScreenImage")
	}
	i.Display.mu.Lock()
	defer i.Display.mu.Unlock()
	return i.free()
}

func (i *Image) free() error {
	if i == nil || i.Display == nil {
		return nil
	}
	// make sure no refresh events occur on this if we block in the write
	d := i.Display
	// flush pending data so we don't get error deleting the image
	d.flush(false)
	a := d.bufimage(1 + 4)
	a[0] = 'f'
	bplong(a[1:], i.ID)
	if i.Screen != nil {
		w := d.Windows
		if w == i {
			d.Windows = i.Next
		} else {
			for ; w != nil; w = w.Next {
				if w.Next == i {
					w.Next = i.Next
					break
				}
			}
		}
	}
	i.Display = nil // So a second free (perhaps through a Finalizer) will be OK.
	runtime.SetFinalizer(i, nil)
	return d.flush(i.Screen != nil)
}

// ColorModel returns the color model of the Image.
func (i *Image) ColorModel() color.Model {
	switch i.Pix {
	case color9.GREY1:
		return color9.Gray1Model
	case color9.GREY2:
		return color9.Gray2Model
	case color9.GREY4:
		return color9.Gray4Model
	case color9.GREY8:
		return color.GrayModel
	case color9.CMAP8:
		return color9.CMap8Model
	case color9.RGB15:
		return color9.CRGB15Model
	case color9.RGB16:
		return color9.CRGB16Model
	case color9.RGB24, color9.RGBA32, color9.ARGB32, color9.ABGR32, color9.XRGB32, color9.XBGR32:
		return color.RGBAModel
	}
	panic("unknown image Pix type")
}

/*
 * Support for the Image type so it can satisfy the standard Color and Image interfaces.
 */

// At returns the standard Color value for the pixel at (x, y).
// If the location is outside the clipping rectangle, it returns color.Transparent.
// This operation does a round trip to the image server and can be expensive.
func (i *Image) At(x, y int) color.Color {
	if !(image.Point{x, y}.In(i.Clipr)) {
		return color.Transparent
	}
	if i.Repl && !(image.Point{x, y}.In(i.R)) {
		// Translate (x, y) to be within i.R.
		x = (x-i.R.Min.X)%(i.R.Max.X-i.R.Min.X) + i.R.Min.X
		y = (y-i.R.Min.Y)%(i.R.Max.Y-i.R.Min.Y) + i.R.Min.Y
	}
	var buf [4]byte
	_, err := i.Unload(image.Rect(x, y, x+1, y+1), buf[:])
	if err != nil {
		println("image.At: error in Unload: ", err.Error())
		return color.Transparent // As good a value as any.
	}
	// For multi-byte pixels, the ordering is little-endian.
	// For sub-byte pixels, the ordering is big-endian (0x80 is the first bit).
	// Three cheers for PCs.
	switch i.Pix {
	case color9.GREY1:
		// CGrey, 1
		mask := uint8(1 << uint8(7-x&7))
		return color9.Gray1{(buf[0] & mask) != 0}
	case color9.GREY2:
		// CGrey, 2
		shift := uint(x&3) << 1
		// Place pixel at top of word.
		y := buf[0] << shift
		y &= 0xC0
		// Replicate throughout.
		y |= y >> 2
		y |= y >> 4
		return color9.Gray2{y}
	case color9.GREY4:
		// CGrey, 4
		shift := uint(x&1) << 2
		// Place pixel at top of word.
		y := buf[0] << shift
		y &= 0xF0
		// Replicate throughout.
		y |= y >> 4
		return color9.Gray4{y}
	case color9.GREY8:
		// CGrey, 8
		return color.Gray{buf[0]}
	case color9.CMAP8:
		// CMap, 8
		return color9.CMap8{buf[0]}
	case color9.RGB15:
		v := uint16(buf[0]) | uint16(buf[1])<<8
		return color9.CRGB15{v}
	case color9.RGB16:
		v := uint16(buf[0]) | uint16(buf[1])<<8
		return color9.CRGB16{v}
	case color9.RGB24:
		// CRed, 8, CGreen, 8, CBlue, 8
		return color.RGBA{buf[2], buf[1], buf[0], 0xFF}
	case color9.BGR24:
		// CBlue, 8, CGreen, 8, CRed, 8
		return color.RGBA{buf[0], buf[1], buf[2], 0xFF}
	case color9.RGBA32:
		// CRed, 8, CGreen, 8, CBlue, 8, CAlpha, 8
		return color.RGBA{buf[3], buf[2], buf[1], buf[0]}
	case color9.ARGB32:
		// CAlpha, 8, CRed, 8, CGreen, 8, CBlue, 8 // stupid VGAs
		return color.RGBA{buf[2], buf[1], buf[0], buf[3]}
	case color9.ABGR32:
		// CAlpha, 8, CBlue, 8, CGreen, 8, CRed, 8
		return color.RGBA{buf[0], buf[1], buf[2], buf[3]}
	case color9.XRGB32:
		// CIgnore, 8, CRed, 8, CGreen, 8, CBlue, 8
		return color.RGBA{buf[2], buf[1], buf[0], 0xFF}
	case color9.XBGR32:
		// CIgnore, 8, CBlue, 8, CGreen, 8, CRed, 8
		return color.RGBA{buf[0], buf[1], buf[2], 0xFF}
	default:
		panic("unknown color")
	}
}

func (i *Image) Bounds() image.Rectangle {
	return i.Clipr
}

// Set implements the image/draw.Image interface.
// It can change a single pixel in an Image.
// The coordinates x and y are relative to the Image's upper left pixel.
// The color.Color will be converted to the Image's color space.
func (i *Image) Set(x, y int, c color.Color) {
	pt := image.Pt(x, y)

	model := i.ColorModel()

	convc := model.Convert(c)

	r, g, b, _ := convc.RGBA()

	colr := color9.Color((r << 24) | (g << 16) | (b << 8) | 0xFF)

	img, err := i.Display.AllocImage(image.Rect(0, 0, 1, 1), i.Pix, true, colr)

	if err != nil {
		panic(err)
	}

	i.Draw(image.Rect(pt.X, pt.Y, pt.X+1, pt.Y+1), img, nil, image.ZP)
}
