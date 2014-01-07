package draw9

import (
	"fmt"
	"image"
	"log"
	"os"
	"syscall"
)

type Menu struct {
	// ???
}

// Mouse is a record of one event from /dev/mouse.
type Mouse struct {
	buttons int
	xy      image.Point
	msec    uint
}

func (m Mouse) String() string {
	out := ""

	switch m.buttons {
	case 1:
		out += "Mb1"
	case 2:
		out += "Mb2"
	case 4:
		out += "Mb3"
	case 8:
		out += "SUp"
	case 16:
		out += "SDn"
	default:
		out += "nil"
	}

	out += fmt.Sprintf(" %s %d", m.xy, m.msec)

	return out
}

// Mb1 returns true if the left mouse button was pressed.
func (m *Mouse) Mb1() bool {
	return m.buttons == 1
}

// Mb2 returns true if the middle mouse button was pressed.
func (m *Mouse) Mb2() bool {
	return m.buttons == 2
}

// Mb3 returns true if the right mouse button was pressed.
func (m *Mouse) Mb3() bool {
	return m.buttons == 4
}

// ScrollUp returns true if the scroll wheel went up.
func (m *Mouse) ScrollUp() bool {
	return m.buttons == 8
}

// ScrollDown returns true if the scroll wheel went down.
func (m *Mouse) ScrollDown() bool {
	return m.buttons == 16
}

// Pt returns the point on the window at which the event happened.
func (m *Mouse) Pt() image.Point {
	return m.xy
}

// Ms returns the millisecond timestamp of the Mouse event.
func (m *Mouse) Ms() uint {
	return m.msec
}

type Mousectl struct {
	Mouse
	C      chan Mouse
	Resize chan int
	quit   chan bool

	// /dev/mouse
	mfd *os.File
	// /dev/cursor
	cfd *os.File
	// window/display
	image *Image
}

func InitMouse(file string, img *Image) *Mousectl {
	var err error
	var ms Mousectl

	if file == "" {
		file = "/dev/mouse"
	}

	ms.mfd, err = os.OpenFile(file, os.O_RDWR|syscall.O_CLOEXEC, 0666)
	if err != nil && file == "/dev/mouse" {
		syscall.Bind("#m", "/dev", syscall.MAFTER)
		ms.mfd, err = os.OpenFile(file, os.O_RDWR|syscall.O_CLOEXEC, 0666)
	}

	if err != nil {
		log.Fatal(err)
	}

	ms.cfd, err = os.OpenFile("/dev/cursor", os.O_RDWR|syscall.O_CLOEXEC, 0666)

	ms.image = img
	ms.C = make(chan Mouse)
	ms.Resize = make(chan int, 2)
	ms.quit = make(chan bool, 1)
	go ms.readproc()
	return &ms
}

func (ms *Mousectl) MoveTo(pt image.Point) {
	fmt.Fprintf(ms.mfd, "m%d %d", pt.X, pt.Y)
	ms.xy = pt
}

func (ms *Mousectl) Close() {
	close(ms.quit)
	ms.mfd.Close()
	ms.cfd.Close()
}

func (ms *Mousectl) ReadMouse() Mouse {
	if ms.image != nil {
		ms.image.Display.flush(true)
	}

	return <-ms.C
}

func (ms *Mousectl) readproc() {
	var m Mouse
	buf := make([]byte, 1+5*12)

loop:
	for {
		select {
		case <-ms.quit:
			break loop
		default:
			n, err := ms.mfd.Read(buf)
			if n != 1+4*12 {
				log.Fatalf("mouse: bad count %d: %s", n, err)
			}

			switch buf[0] {
			// resize
			case 'r':
				ms.Resize <- 1
				fallthrough
			// mouse move
			case 'm':
				m.xy.X = atoi(buf[1+0*12:])
				m.xy.Y = atoi(buf[1+1*12:])
				m.buttons = atoi(buf[1+2*12:])
				m.msec = uint(atoi(buf[1+3*12:]))
				ms.C <- m
			}
		}
	}

	close(ms.C)
	close(ms.Resize)
}

/* TODO: setcursor */
