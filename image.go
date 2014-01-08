package draw9

import (
	"image"
	"runtime"
)

type Image struct {
	Display  *Display
	ID       uint32
	Pix      Pix
	R, Clipr image.Rectangle
	Depth    int
	Chan     uint
	Repl     bool
	Next     *Image
	Screen   *Screen
}

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
