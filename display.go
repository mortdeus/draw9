package draw9

import (
	"fmt"
	"io"
	"os"
	"sync"
)

type Display struct {
	// debugging
	debug bool
	// sync
	mu      sync.Mutex
	locking bool

	dirno int

	// control files
	fd    *os.File
	reffd *os.File
	ctlfd *os.File

	// image ids
	imageid uint32
	qmask   *Image

	local    bool
	devdir   string
	windir   string
	oldlabel string

	// colors
	White       *Image
	Black       *Image
	Opaque      *Image
	Transparent *Image

	Image       *Image
	Screen      *Screen
	ScreenImage *Image
	Windows     *Image

	// devdraw message buffering
	bufsize int
	buf     []byte

	// fonts
	DefaultFont    int
	DefaultSubfont int

	_isnew bool
}

//func (d *Display)
func (d *Display) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.close()
}

func (d *Display) close() error {
	if d.oldlabel != "" {
		if f, err := os.OpenFile(d.windir+"/label", os.O_WRONLY, 0); err == nil {
			io.WriteString(f, d.oldlabel)
			f.Close()
		}
	}

	return nil

	d.White.free()
	d.Black.free()

	err1 := d.fd.Close()
	err2 := d.ctlfd.Close()
	err3 := d.reffd.Close()

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	if err3 != nil {
		return err3
	}
	return nil
}

func (d *Display) readctl() ([]byte, error) {
	buf := make([]byte, 12*12)
	d.ctlfd.Seek(0, 0)
	_, err := d.ctlfd.Read(buf)
	return buf, err
}

func (d *Display) bufimage(n int) []byte {
	if d == nil || n < 0 || n > d.bufsize {
		panic("bad count in bufimage")
	}

	if len(d.buf)+n > d.bufsize {
		if err := d.doflush(); err != nil {
			panic("bufimage flush: " + err.Error())
		}
	}

	i := len(d.buf)
	d.buf = d.buf[:i+n]
	return d.buf[i:]
}

// Flush writes any pending data to the screen.
func (d *Display) Flush() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.flush(true)
}

// flush data, maybe make visible
func (d *Display) flush(vis bool) error {
	if vis {
		d.bufsize++
		a := d.bufimage(1)
		d.bufsize--
		a[0] = 'v'
	}

	return d.doflush()
}

// write out any buffered data to the devdraw data fd
func (d *Display) doflush() error {
	if len(d.buf) == 0 {
		return nil
	}

	if d.debug {
		fmt.Fprintf(os.Stderr, "flushing %d %c %x\n", len(d.buf), rune(d.buf[0]), d.buf[1:len(d.buf)])
	}

	_, err := d.fd.Write(d.buf)
	d.buf = d.buf[:0]

	if err != nil {
		if d.debug {
			fmt.Fprintf(os.Stderr, "doflush: %s\n", err)
		}
		return fmt.Errorf("doflush: %s", err)
	}

	return nil
}
