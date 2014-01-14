package draw9

import (
	"fmt"
	"io"
	"io/ioutil"
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
	DefaultFont    *Font
	DefaultSubfont *Subfont
}

func (d *Display) OpenFont(name string) (*Font, error) {
	// nil display is allowed, for querying font metrics
	// in non-draw program.
	if d != nil {
		d.mu.Lock()
		defer d.mu.Unlock()
	}
	return d.openFont(name)
}

func (d *Display) openFont(name string) (*Font, error) {
	data, err := ioutil.ReadFile(name)

	if err != nil {
		return nil, err
	}

	return d.buildFont(data, name)
}

func (d *Display) readSubfont(name string, fd io.Reader, ai *Image) (*Subfont, error) {
	hdr := make([]byte, 3*12+4)
	i := ai
	if i == nil {
		var err error
		i, err = d.readImage(fd)
		if err != nil {
			return nil, err
		}
	}
	var (
		n   int
		p   []byte
		fc  []Fontchar
		f   *Subfont
		err error
	)
	// Release lock for the I/O - could take a long time.
	if d != nil {
		//d.mu.Unlock()
	}
	_, err = io.ReadFull(fd, hdr[:3*12])
	if d != nil {
		//d.mu.Lock()
	}
	//defer d.mu.Unlock()
	if err != nil {
		err = fmt.Errorf("rdsubfontfile: header read error: %v", err)
		goto Err
	}
	n = atoi(hdr)
	p = make([]byte, 6*(n+1))
	if _, err = io.ReadFull(fd, p); err != nil {
		err = fmt.Errorf("rdsubfontfile: fontchar read error: %v", err)
		goto Err
	}
	fc = make([]Fontchar, n+1)
	unpackinfo(fc, p, n)
	f = d.allocSubfont(name, atoi(hdr[12:]), atoi(hdr[24:]), fc, i)
	return f, nil

Err:
	if ai == nil {
		i.free()
	}
	return nil, err
}

func (d *Display) ReadSubfont(name string, fd io.Reader) (*Subfont, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.readSubfont(name, fd, nil)
}

func unpackinfo(fc []Fontchar, p []byte, n int) {
	for j := 0; j <= n; j++ {
		fc[j].X = int(p[0]) | int(p[1])<<8
		fc[j].Top = uint8(p[2])
		fc[j].Bottom = uint8(p[3])
		fc[j].Left = int8(p[4])
		fc[j].Width = uint8(p[5])
		p = p[6:]
	}
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
	buf := make([]byte, 12*12+1)
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
