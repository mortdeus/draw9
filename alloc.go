package draw9

import (
	"bitbucket.org/mischief/draw9/color9"
	"bitbucket.org/mischief/draw9/image9"
	"bytes"
	"fmt"
	"image"
	"runtime"
)

// LoadImage loads an image.Image onto the Display, and returns a handle to the Image.
func (d *Display) LoadImage(load image.Image) (i *Image, err error) {
	i, err = d.AllocImage(load.Bounds(), color9.RGBA32, false, 0)

	if err != nil {
		return nil, err
	}

	img9buf := new(bytes.Buffer)

	if err = image9.Encode(img9buf, load); err != nil {
		return nil, err
	}

	if _, err = i.Load(load.Bounds(), img9buf.Bytes()[60:]); err != nil {
		return nil, err
	}

	return i, nil
}

// AllocImage allocates a new Image on display d.
func (d *Display) AllocImage(r image.Rectangle, pix color9.Pix, repl bool, val color9.Color) (*Image, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return allocImage(d, nil, r, pix, repl, val, 0, 0)
}

func (d *Display) allocImage(r image.Rectangle, pix color9.Pix, repl bool, val color9.Color) (i *Image, err error) {
	return allocImage(d, nil, r, pix, repl, val, 0, 0)
}

// implements message 'b'
func allocImage(d *Display, ai *Image, r image.Rectangle, pix color9.Pix, repl bool, val color9.Color, screenid uint32, refresh int) (i *Image, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("allocimage %v %v: %v", r, pix, err)
			i.free()
			i = nil
		}
	}()

	depth := pix.Depth()
	if depth == 0 {
		err = fmt.Errorf("bad channel descriptor")
		panic(err)
		return
	}

	// flush pending data so we don't get error allocating the image
	d.flush(false)
	a := d.bufimage(1 + 4 + 4 + 1 + 4 + 1 + 4*4 + 4*4 + 4)
	d.imageid++
	id := d.imageid
	a[0] = 'b'
	bplong(a[1:], id)
	bplong(a[5:], screenid)
	a[9] = byte(refresh)
	bplong(a[10:], uint32(pix))
	if repl {
		a[14] = 1
	} else {
		a[14] = 0
	}
	bplong(a[15:], uint32(r.Min.X))
	bplong(a[19:], uint32(r.Min.Y))
	bplong(a[23:], uint32(r.Max.X))
	bplong(a[27:], uint32(r.Max.Y))
	clipr := r
	if repl {
		// huge but not infinite, so various offsets will leave it huge, not overflow
		clipr = image.Rect(-0x3FFFFFFF, -0x3FFFFFFF, 0x3FFFFFFF, 0x3FFFFFFF)
	}
	bplong(a[31:], uint32(clipr.Min.X))
	bplong(a[35:], uint32(clipr.Min.Y))
	bplong(a[39:], uint32(clipr.Max.X))
	bplong(a[43:], uint32(clipr.Max.Y))
	bplong(a[47:], uint32(val))
	if err = d.flush(false); err != nil {
		return
	}

	i = ai
	if i == nil {
		i = new(Image)
	}
	*i = Image{
		Display: d,
		ID:      id,
		Pix:     pix,
		Depth:   pix.Depth(),
		R:       r,
		Clipr:   clipr,
		Repl:    repl,
	}
	runtime.SetFinalizer(i, (*Image).Free)
	return i, nil
}

func (d *Display) AllocImageMix(color1, color3 color9.Color) *Image {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.ScreenImage.Depth <= 8 { // create a 2x2 texture
		t, _ := d.allocImage(image.Rect(0, 0, 1, 1), d.ScreenImage.Pix, false, color1)
		b, _ := d.allocImage(image.Rect(0, 0, 2, 2), d.ScreenImage.Pix, true, color3)
		b.draw(image.Rect(0, 0, 1, 1), t, nil, image.ZP)
		t.free()
		return b
	}

	// use a solid color, blended using alpha
	if d.qmask == nil {
		d.qmask, _ = d.allocImage(image.Rect(0, 0, 1, 1), color9.GREY8, true, 0x3F3F3FFF)
	}
	t, _ := d.allocImage(image.Rect(0, 0, 1, 1), d.ScreenImage.Pix, true, color1)
	b, _ := d.allocImage(image.Rect(0, 0, 1, 1), d.ScreenImage.Pix, true, color3)
	b.draw(b.R, t, d.qmask, image.ZP)
	return b
}

/* implements message 'n' */
func (d *Display) namedimage(name []byte) (*Image, error) {
	d.flush(false)
	a := d.bufimage(1 + 4 + 1 + len(name))
	d.imageid++
	id := d.imageid

	a[0] = 'n'
	bplong(a[1:], id)
	a[5] = byte(len(name))
	copy(a[6:], name)
	if err := d.flush(false); err != nil {
		return nil, fmt.Errorf("namedimage: %s", err)
	}

	ctlbuf, err := d.readctl()
	if err != nil {
		return nil, fmt.Errorf("namedimage: %s", err)
	}

	pix, _ := color9.ParsePix(string(ctlbuf[2*12 : 3*12]))
	image := &Image{
		Display: d,
		ID:      id,
		Pix:     pix,
		Depth:   pix.Depth(),
		Repl:    atoi(ctlbuf[3*12:]) > 0,
		R:       ator(ctlbuf[4*12:]),
		Clipr:   ator(ctlbuf[8*12:]),
		Next:    nil,
		Screen:  nil,
	}

	return image, nil
}

/* implements message 'N' */
func nameimage(i *Image, name string, in bool) error {
	a := i.Display.bufimage(1 + 4 + 1 + 1 + len(name))
	a[0] = 'N'
	bplong(a[1:], i.ID)
	if in {
		a[5] = 1
	}
	a[6] = byte(len(name))
	copy(a[7:], name)
	return i.Display.flush(false)
}
