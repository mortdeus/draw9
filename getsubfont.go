package draw9

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"log"
)

func getsubfont(d *Display, name string) (*Subfont, error) {
	data, err := ioutil.ReadFile(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "getsubfont: %v\n", err)
		return nil, err
	}
	f, err := d.readSubfont(name, bytes.NewReader(data), nil)
	if err != nil {
		return nil, fmt.Errorf("getsubfont: can't read %s: %v", name, err)
	}

	if d.debug {
		log.Printf("load %s", f)
	}
	return f, err
}
