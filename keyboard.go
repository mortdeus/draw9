package draw9

import (
	"io"
	"log"
	"os"
	"syscall"
	"unicode/utf8"
	"bytes"
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

type Consctl struct {
	C chan rune

	quit chan bool
	cons *os.File
	ctl  *os.File
}

func InitCons(file string) *Consctl {
	var err error
	var cons Consctl

	if file == "" {
		file = "/dev/cons"
	}

	if cons.cons, err = os.OpenFile(file, os.O_RDWR|syscall.O_CLOEXEC, 0666); err != nil {
		log.Fatal(err)
	}

	if cons.ctl, err = os.OpenFile(file+"ctl", os.O_WRONLY|syscall.O_CLOEXEC, 0666); err != nil {
		log.Fatal(err)
	}

	io.WriteString(cons.ctl, "rawon")

	cons.C = make(chan rune, 20)
	cons.quit = make(chan bool, 1)
	go cons.readproc()
	return &cons
}

func (cons *Consctl) Close() {
	io.WriteString(cons.ctl, "rawoff")
	close(cons.quit)
	cons.ctl.Close()
	cons.cons.Close()
	//	<-cons.C
}

func (cons *Consctl) readproc() {
	buf := make([]byte, 20)
	n := 0

loop:
	for {
		select {
		case <-cons.quit:
			break loop
		default:
			for n > 0 && utf8.FullRune(buf) {
				r, size := utf8.DecodeRune(buf)
				n -= size
				copy(buf, buf[size:])
				cons.C <- r
			}
			m, err := cons.cons.Read(buf[n:])
			if err != nil {
				log.Fatal(err)
			}
			n += m
		}
	}

	close(cons.C)
}

type KbdType int

const (
	KbdDown KbdType = iota
	KbdUp
	KbdChar
)

type Kbd struct {
	Type KbdType
	R rune
}

type Keyboardctl struct {
	C chan Kbd

	quit chan bool
	fd *os.File
}

func InitKeyboard(file string) *Keyboardctl {
	var err error
	var kbd Keyboardctl

	if file == "" {
		file = "/dev/kbd"
	}

	if kbd.fd, err = os.OpenFile(file, os.O_RDWR|syscall.O_CLOEXEC, 0666); err != nil {
		log.Fatal(err)
	}

	kbd.C = make(chan Kbd)
	kbd.quit = make(chan bool, 1)
	go kbd.readproc()
	return &kbd
}

func (k *Keyboardctl) Close() error {
	close(k.quit)
	return k.fd.Close()
}

func (k *Keyboardctl) readproc() {
	var s []byte

	buf := make([]byte, 128)
	buf2 := make([]byte, 128)

loop:
for {
	select {
	case <-k.quit:
		break loop
	default:
		if m, err := k.fd.Read(buf); err == nil && m > 0 {
			var e Kbd
			switch buf[0] {
			case 'c':
				r, _ := utf8.DecodeRune(buf[1:])
				if r != utf8.RuneError {
					e.Type = KbdChar
					e.R = r
					k.C <- e
				}
				fallthrough
			default:
				continue
			case 'k':
				s = buf[1:]
				for utf8.FullRune(s) {
					r, sz := utf8.DecodeRune(s)
					s = s[sz:]
					if bytes.IndexRune(buf2[1:], r) == -1 {
						e.Type = KbdDown
						e.R = r
						k.C <- e
					}
				}
			case 'K':
				s = buf2[1:]
				for utf8.FullRune(s) {
					r, sz := utf8.DecodeRune(s)
					s = s[sz:]
					if bytes.IndexRune(buf[1:], r) == -1 {
						e.Type = KbdUp
						e.R = r
						k.C <- e
					}
				}				
			}
			copy(buf2, buf)
		} else {
			log.Printf("Keyboardctl: %s", err)
			break loop
		}
	}
}

	close(k.C)
}
