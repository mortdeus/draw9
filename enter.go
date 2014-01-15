package draw9

import (
	"bitbucket.org/mischief/draw9/color9"
	"image"
	"log"
)

func (d *Display) Enter(ask, ans string, k *Keyboardctl, m *Mousectl) string {
	var err error
	var save *Image

	// result
	out := []rune(ans)

	// placement of '|'
	tick := len(out)

	// condition for loop below
	done := false

	b := d.ScreenImage
	sc := b.Clipr

	back := d.AllocImageMix(color9.DPurpleblue, color9.DWhite)
	bord, _ := d.AllocImage(image.Rect(0, 0, 1, 1), d.ScreenImage.Pix, true, color9.DPurpleblue)

	p := d.DefaultFont.StringSize(" ")
	w := p.X
	h := p.Y

	o := b.R.Min

	if m != nil {
		o = m.Pt()
	}

drain:
	for {
		select {
		case <-k.C:
		default:
			break drain
		}
	}

	b.ReplClipr(false, b.R)

loop:
	for err == nil && done == false {
		p = d.DefaultFont.StringSize(string(out))
		if ask != "" {
			if len(out) > 0 {
				p.X += w
			}
			p.X += d.DefaultFont.StringWidth(ask)
		}

		r := image.Rect(0, 0, p.X, p.Y).Inset(-4).Add(o)
		p.X = 0
		r = r.Sub(p)

		p = image.ZP

		if r.Min.X < b.R.Min.X {
			p.X = b.R.Min.X - r.Min.X
		}
		if r.Min.Y < b.R.Min.Y {
			p.Y = b.R.Min.Y - r.Min.Y
		}
		r = r.Add(p)
		p = image.ZP
		if r.Max.X > b.R.Max.X {
			p.X = r.Max.X - b.R.Max.X
		}
		if r.Max.Y > b.R.Max.Y {
			p.Y = r.Max.Y - b.R.Max.Y
		}
		r = r.Sub(p).Inset(-2)

		if save == nil {
			if save, err = d.AllocImage(r, b.Pix, false, color9.DNofill); err != nil {
				break loop
			}
			save.Draw(r, b, nil, r.Min)
		}

		b.Draw(r, back, nil, image.ZP)
		b.Border(r, 2, bord, image.ZP)
		p = r.Min.Add(image.Pt(6, 6))
		if ask != "" {
			p = b.String(p, bord, image.ZP, d.DefaultFont, ask)
			if len(out) > 0 {
				p.X += w
			}
		}

		if len(out) > 0 {
			//t := p
			p = b.String(p, d.Black, image.ZP, d.DefaultFont, string(out[:tick]))
			b.Draw(image.Rect(p.X-1, p.Y, p.X+2, p.Y+3), d.Black, nil, image.ZP)
			b.Draw(image.Rect(p.X, p.Y, p.X+1, p.Y+h), d.Black, nil, image.ZP)
			b.Draw(image.Rect(p.X-1, p.Y+h-3, p.X+2, p.Y+h), d.Black, nil, image.ZP)

			p = b.String(p, d.Black, image.ZP, d.DefaultFont, string(out[tick:]))
		}

		d.Flush()

		b.ReplClipr(false, sc)

		// mouse
		var mchan chan Mouse
		// resize
		var rchan chan int

		if m != nil {
			mchan = m.C
			rchan = m.Resize
		}

		select {
		case r := <-k.C:
			// kbd
			if r == 0 || r == Keof || r == '\n' {
				done = true
				break
			}

			if r == Knack || r == Kesc {
				break loop
			}

			if r == Ksoh || r == Khome {
				tick = 0
				continue
			}

			if r == Kenq || r == Kend {
				tick = len(out)
				continue
			}

			if r == Kright {
				if tick < len(out) {
					tick++
				}
				continue
			}

			if r == Kleft {
				tick--
				if tick <= 0 {
					tick = 0
				}
				continue
			}

			if r == Kbs {
				if len(out) <= 0 || tick < 1 {
					continue
				}

				tick--
				out = append(out[:tick], out[tick+1:]...)
				//tick--
				break
			}

			if r < 0x20 || r == Kdel || (r&0xFF00) == KF || (r&0xFF00) == Spec {
				continue
			}

			out = append(out, 0)
			copy(out[tick+1:], out[tick:])
			out[tick] = r
			tick++
		case <-mchan:
			// mouse
		case <-rchan:
			// resize
			if err = d.Attach(Refmesg); err != nil {
				break loop
			}
			b = d.ScreenImage
			save.Free()
			save = nil
		}

		if save != nil {
			b.Draw(save.R, save, nil, save.R.Min)
			save.Free()
			save = nil
		}
	}

	b.ReplClipr(false, sc)

	back.Free()
	bord.Free()

	if err != nil {
		log.Printf("Enter: %s", err)
	}

	return string(out)
}
