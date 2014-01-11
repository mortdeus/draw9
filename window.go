package draw9

import (
	"bitbucket.org/mischief/draw9/color9"
	"fmt"
	"image"
)

var (
	screenid uint32 = 0
)

func (i *Image) AllocScreen(fill *Image, public bool) (*Screen, error) {
	i.Display.mu.Lock()
	defer i.Display.mu.Unlock()
	return i.allocScreen(fill, public)
}

func (i *Image) allocScreen(fill *Image, public bool) (*Screen, error) {
	var id uint32

	d := i.Display
	if d != fill.Display {
		return nil, fmt.Errorf("AllocScreen: image and fill on different displays")
	}

	for try := 0; ; try++ {
		if try > 25 {
			return nil, fmt.Errorf("no screen")
		}
		a := d.bufimage(1 + 4 + 4 + 4 + 1)
		screenid++
		id = screenid
		a[0] = 'A'
		bplong(a[1:], id)
		bplong(a[5:], i.ID)
		bplong(a[9:], fill.ID)
		if public {
			a[13] = 1
		} else {
			a[13] = 0
		}

		if err := d.flush(false); err == nil {
			break
		}
	}

	s := &Screen{
		Display: d,
		ID:      id,
		Image:   i,
		Fill:    fill,
	}

	return s, nil
}

func allocwindow(i *Image, s *Screen, r image.Rectangle, ref int, val color9.Color) (*Image, error) {
	d := s.Display
	i, err := allocImage(d, i, r, d.ScreenImage.Pix, false, val, s.ID, ref)
	if err != nil {
		return nil, err
	}
	i.Screen = s
	i.Next = s.Display.Windows
	s.Display.Windows = i
	return i, nil
}
