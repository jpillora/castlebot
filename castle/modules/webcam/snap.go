package webcam

import (
	"bytes"
	"fmt"
	"image"
	"time"

	"image/color"
	_ "image/jpeg" //needed for jpeg encoding
)

var (
	black = &color.RGBA{R: 0, G: 0, B: 0, A: 0xff}
	cyan  = &color.RGBA{R: 0, G: 0xff, B: 0xff, A: 0xff}
	pink  = &color.RGBA{R: 0xff, G: 0, B: 0xe0, A: 0xff}
	white = &color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
)

type snap struct {
	//image
	id  []byte
	t   time.Time
	raw []byte
	img image.Image
	//diff
	diffed   bool
	pdiffSum int
	pdiffNum int
	// diff     []byte //debug image
	//store
	stored bool
}

func newSnap(raw []byte) (*snap, error) {
	s := &snap{}
	s.t = time.Now()
	s.id = toID(s.t)
	s.raw = raw
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("jpg: %s", err)
	}
	s.img = img
	return s, nil
}

func (s *snap) computeDiff(threshold int, other *snap) int {
	if s.diffed {
		return s.pdiffNum
	}
	s.diffed = true
	b := s.img.Bounds()
	if other != nil && b.Eq(other.img.Bounds()) {
		//NOTE: diff image is for debugging
		// d := image.NewNRGBA(b)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r1, g1, b1, _ := s.img.At(x, y).RGBA()
				r2, g2, b2, _ := other.img.At(x, y).RGBA()
				pdiff := abs(int(r2)-int(r1)) +
					abs(int(g2)-int(g1)) +
					abs(int(b2)-int(b1))
				s.pdiffSum += pdiff
				if pdiff > threshold {
					// if r1+g1+b1 > r2+g2+b2 {
					// 	d.Set(x, y, pink)
					// } else {
					// 	d.Set(x, y, cyan)
					// }
					s.pdiffNum++
				} /* else {
					d.Set(x, y, black)
				}*/
			}
		}
		// buff := bytes.Buffer{}
		// jpeg.Encode(&buff, d, nil)
		// s.diff = buff.Bytes()
	}
	return s.pdiffNum
}
