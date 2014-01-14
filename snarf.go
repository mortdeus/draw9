package draw9

import (
	"os"
)

const snarf = "/dev/snarf"

func ReadSnarf(b []byte) (int, error) {
	if f, e := os.Open(snarf); e == nil {
		defer f.Close()
		return f.Read(b)
	} else {
		return 0, e
	}
}

func WriteSnarf(b []byte) (int, error) {
	if f, e := os.OpenFile(snarf, os.O_WRONLY, 0); e == nil {
		defer f.Close()
		return f.Write(b)
	} else {
		return 0, e
	}
}
