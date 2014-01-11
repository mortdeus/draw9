package draw9

import (
	"bitbucket.org/mischief/draw9/color9"
	"bytes"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
	"log"
)

const (
	deffontname = "*default*"
)

func InitDraw(errch chan<- error, fontname, label string) (*Display, error) {
	dev := "/dev"
	if _, err := os.Stat("/dev/draw/new"); err != nil {
		if err := syscall.Bind("#i", "/dev", syscall.MAFTER); err != nil {
			return nil, err
		}
	}

	return geninitdraw(dev, dev, fontname, label, Refnone)
}

func geninitdraw(devdir, windir, fontname, label string, ref int) (d *Display, err error) {
	var buf string

	d, err = initdisplay(devdir, windir)
	if err != nil {
		return nil, fmt.Errorf("initdisplay: %s", err)
	}

	/* default font */
	df, err := getdefont(d)
	if err != nil {
		return nil, err
	}
	d.DefaultSubfont = df

	if d.debug {
		log.Printf("loaded %s", df)
	}

	if fontname == "" {
		fontname = os.Getenv("font")
	}

	var font *Font

	if fontname == "" {
		buf := []byte(fmt.Sprintf("%d %d\n0 %d\t%s\n", df.Height, df.Ascent,
			df.N-1, deffontname))
		font, err = d.buildFont(buf, deffontname)
	} else {
		font, err = d.openFont(fontname)
	}

	if err != nil {
		return nil, fmt.Errorf("can't open default font %s: %s", font, err)
	}

	if d.debug {
		log.Printf("loaded %s", font)
	}

	d.DefaultFont = font

	if label != "" {
		buf = d.windir + "/label"
		if fd, err := os.Open(buf); err == nil {
			old := make([]byte, 64)
			io.ReadFull(fd, old)
			d.oldlabel = string(old)
			fd.Close()
			if fd, err = os.Create(buf); err == nil {
				io.WriteString(fd, label)
				fd.Close()
			}
		}
	}

	buf = d.windir + "/winname"
	return d, gengetwindow(d, buf, ref)
}

const (
	nINFO = 12 * 12
)

/* TODO: setup err chan */
func initdisplay(devdir, windir string) (*Display, error) {
	var err error
	var buf string
	var info []byte
	var isnew bool

	if devdir == "" {
		devdir = "/dev"
	}
	if windir == "" {
		windir = "/dev"
	}

	info = make([]byte, 12*12)

	d := &Display{
//		debug: true,
		devdir: devdir,
		windir: windir,
	}

	buf = devdir + "/draw/new"

	if d.ctlfd, err = os.OpenFile(buf, os.O_RDWR|syscall.O_CLOEXEC, 0666); err != nil {
		if err = syscall.Bind("#i", devdir, syscall.MAFTER); err != nil {
			return nil, err
		}

		if d.ctlfd, err = os.OpenFile(buf, os.O_RDWR|syscall.O_CLOEXEC, 0666); err != nil {
			return nil, err
		}
	}

	n, err := io.ReadFull(d.ctlfd, info)
	if err != nil || n < 12 {
		return nil, err
	}

	isnew = false
	if n < nINFO {
		isnew = true
	}

	id := atoi(info[:1*12])
	buf = devdir + "/draw/" + strconv.Itoa(id) + "/data"
	if d.fd, err = os.OpenFile(buf, os.O_RDWR|syscall.O_CLOEXEC, 0666); err != nil {
		return nil, err
	}

	buf = devdir + "/draw/" + strconv.Itoa(id) + "/refresh"
	if d.reffd, err = os.OpenFile(buf, os.O_RDWR|syscall.O_CLOEXEC, 0666); err != nil {
		return nil, err
	}

	i := &Image{}

	pix, _ := color9.ParsePix(strings.TrimSpace(string(info[2*12 : 3*12])))

	if d.debug {
		fmt.Fprintf(os.Stderr, "display pix: %s %v\n", pix, pix.Depth())
	}

	if n >= nINFO {
		i.Display = d
		i.ID = 0
		i.Pix = pix
		i.Depth = pix.Depth()
		i.Repl = atoi(info[3*12:]) > 0
		i.R = ator(info[4*12:])
		i.Clipr = ator(info[8*12:])
	}

	d._isnew = isnew
	d.bufsize = Iounit(int(d.fd.Fd()))
	if d.bufsize <= 0 {
		d.bufsize = 8000
	}

	d.buf = make([]byte, 0, d.bufsize)
	d.Image = i
	d.dirno = atoi(info[0*12:])

	/* fd, ctlfd, reffd already setup */

	d.windir = windir
	d.devdir = devdir

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.White, err = d.allocImage(image.Rect(0, 0, 1, 1), color9.GREY1, true, color9.DWhite); err != nil {
		return nil, fmt.Errorf("can't allocate white: %s", err)
	}

	if d.Black, err = d.allocImage(image.Rect(0, 0, 1, 1), color9.GREY1, true, color9.DBlack); err != nil {
		return nil, fmt.Errorf("can't allocate black: %s", err)
	}

	d.Opaque = d.White
	d.Transparent = d.Black

	return d, nil
}

// Called when a resize happens, equivalent to getwindow in draw(2)
func (d *Display) Attach(ref int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.getwindow(ref)
}

func (d *Display) getwindow(ref int) error {
	winname := fmt.Sprintf("%s/winname", d.windir)
	return gengetwindow(d, winname, ref)
}

// Attach, or possibly reattach, to window.
// If reattaching, maintain value of screen pointer.
func gengetwindow(d *Display, winname string, ref int) error {
	var i *Image
	var noborder bool
	var buf, obuf []byte
	var err error

	/* this is crazy */
retry:
	buf, err = ioutil.ReadFile(winname)
	if err != nil {
		i = d.Image
		if i == nil {
			fmt.Fprintf(os.Stderr, "gengetwindow: %s\n", err)
			return err
		}
		noborder = true
	} else {
		if i, err = d.namedimage(buf); err != nil {
			if bytes.Compare(buf, obuf) != 0 {
				copy(obuf, buf)
				goto retry
			}

			fmt.Fprintf(os.Stderr, "namedimage %s failed: %s\n", buf, err)
		}

		if d.ScreenImage != nil {
			d.ScreenImage.free()
			d.Screen.Image.free()
			d.Screen.free()
			d.Screen = nil
		}

		if i == nil {
			d.ScreenImage = nil
			return fmt.Errorf("nil image")
		}
	}

	d.Screen, err = i.allocScreen(d.White, false)
	if err != nil {
		return err
	}

	r := i.R
	if !noborder {
		r = i.R.Inset(Borderwidth)
	}

	d.ScreenImage = d.Image
	d.ScreenImage, err = allocwindow(nil, d.Screen, r, 0, color9.DWhite)
	if err != nil {
		return err
	}

	if err := d.flush(true); err != nil {
		return err
	}

	scr := d.ScreenImage
	scr.draw(scr.R, d.White, nil, image.ZP)

	return nil
}
