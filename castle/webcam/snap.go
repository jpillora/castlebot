package webcam

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"time"

	"image/color"
	"image/jpeg"
)

var (
	black = &color.RGBA{R: 0, G: 0, B: 0, A: 0xff}
	cyan  = &color.RGBA{R: 0, G: 0xff, B: 0xff, A: 0xff}
	pink  = &color.RGBA{R: 0xff, G: 0, B: 0xe0, A: 0xff}
	white = &color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
)

type snap struct {
	id       []byte
	t        time.Time
	prev     *snap
	raw      []byte
	img      image.Image
	diff     []byte
	pdiffSum int
	pdiffNum int
	stored   bool
}

func newSnap(raw []byte, threshold int, last *snap) (*snap, error) {
	s := &snap{}
	s.t = time.Now()
	s.id = toID(s.t)
	s.raw = raw
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("jpg: %s", err)
	}
	s.img = img
	b := s.img.Bounds()
	if last != nil && b.Eq(last.img.Bounds()) {
		d := image.NewNRGBA(b)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r1, g1, b1, _ := s.img.At(x, y).RGBA()
				r2, g2, b2, _ := last.img.At(x, y).RGBA()

				pdiff := abs(int(r2)-int(r1)) +
					abs(int(g2)-int(g1)) +
					abs(int(b2)-int(b1))

				s.pdiffSum += pdiff
				if pdiff > threshold {
					if r1+g1+b1 > r2+g2+b2 {
						d.Set(x, y, pink)
					} else {
						d.Set(x, y, cyan)
					}
					s.pdiffNum++
				} else {
					d.Set(x, y, black)
				}
			}
		}
		buff := bytes.Buffer{}
		if err := jpeg.Encode(&buff, d, nil); err != nil {
			log.Printf("jpgencode: %s", err)
		} else {
			s.diff = buff.Bytes()
			log.Printf("sum: %d (num: %d, avg: %f)", s.pdiffSum, s.pdiffNum, float64(s.pdiffSum)/float64(b.Max.X*b.Max.Y))
		}
	}
	return s, nil
}
