package draw9

type Screen struct {
	Display *Display
	ID      uint32
	Image   *Image
	Fill    *Image
}

func (s *Screen) Free() error {
	s.Display.mu.Lock()
	defer s.Display.mu.Unlock()
	return s.free()
}

func (s *Screen) free() error {
	if s == nil {
		return nil
	}
	d := s.Display
	a := d.bufimage(1 + 4)
	a[0] = 'F'
	bplong(a[1:], s.ID)
	// flush(true) because screen is likely holding the last reference to window,
	// and we want it to disappear visually.
	return d.flush(true)
}
