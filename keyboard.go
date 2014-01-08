package draw9

import (
	"io"
	"log"
	"os"
	"syscall"
	"unicode/utf8"
)

const (
	KF    = 0xF000 /* Rune: beginning of private Unicode space */
	Spec  = 0xF800
	PF    = Spec | 0x20 /* num pad function key */
	Kview = Spec | 0x00 /* view (shift window up) */
	/* KF|1, KF|2, ..., KF|0xC is F1, F2, ..., F12 */
	Khome   = KF | 0x0D
	Kup     = KF | 0x0E
	Kdown   = Kview
	Kpgup   = KF | 0x0F
	Kprint  = KF | 0x10
	Kleft   = KF | 0x11
	Kright  = KF | 0x12
	Kpgdown = KF | 0x13
	Kins    = KF | 0x14

	Kalt   = KF | 0x15
	Kshift = KF | 0x16
	Kctl   = KF | 0x17

	Kend           = KF | 0x18
	Kscroll        = KF | 0x19
	Kscrolloneup   = KF | 0x20
	Kscrollonedown = KF | 0x21

	Ksoh  = 0x01
	Kstx  = 0x02
	Ketx  = 0x03
	Keof  = 0x04
	Kenq  = 0x05
	Kack  = 0x06
	Kbs   = 0x08
	Knack = 0x15
	Ketb  = 0x17
	Kdel  = 0x7f
	Kesc  = 0x1b

	Kbreak  = Spec | 0x61
	Kcaps   = Spec | 0x64
	Knum    = Spec | 0x65
	Kmiddle = Spec | 0x66
	Kaltgr  = Spec | 0x67
	Kmouse  = Spec | 0x100
)

type Keyboardctl struct {
	C chan rune

	quit chan bool
	cons *os.File
	ctl  *os.File
}

//func (d *Display) InitKeyboard(file string) *Keyboardctl {
func InitKeyboard(file string) *Keyboardctl {
	var err error
	var kbd Keyboardctl

	if file == "" {
		file = "/dev/cons"
	}

	if kbd.cons, err = os.OpenFile(file, os.O_RDWR|syscall.O_CLOEXEC, 0666); err != nil {
		log.Fatal(err)
	}

	if kbd.ctl, err = os.OpenFile(file+"ctl", os.O_WRONLY|syscall.O_CLOEXEC, 0666); err != nil {
		log.Fatal(err)
	}

	io.WriteString(kbd.ctl, "rawon")

	kbd.C = make(chan rune, 20)
	kbd.quit = make(chan bool, 1)
	go kbd.readproc()
	return &kbd
}

func (kbd *Keyboardctl) Close() {
	io.WriteString(kbd.ctl, "rawoff")
	close(kbd.quit)
	kbd.ctl.Close()
	kbd.cons.Close()
	//	<-kbd.C
}

func (kbd *Keyboardctl) readproc() {
	buf := make([]byte, 20)
	n := 0

loop:
	for {
		select {
		case <-kbd.quit:
			break loop
		default:
			for n > 0 && utf8.FullRune(buf) {
				r, size := utf8.DecodeRune(buf)
				n -= size
				copy(buf, buf[size:])
				kbd.C <- r
			}
			m, err := kbd.cons.Read(buf[n:])
			if err != nil {
				log.Fatal(err)
			}
			n += m
		}
	}

	close(kbd.C)
}
