package webcam

import (
	"bytes"
	"fmt"
	"image"
	"time"

	_ "image/jpeg"

	"github.com/disintegration/gift"
)

type snap struct {
	id                           string
	n                            int
	avg, min, max, diff, diffsum float64
	changing, newState           bool
	t                            time.Time
	prev                         *snap
	raw                          []byte
	processed                    image.Image
	stored                       bool
}

func newSnap(raw []byte) (*snap, error) {
	s := &snap{}
	s.t = time.Now()
	s.id = s.t.UTC().Format(time.RFC3339)
	s.raw = raw
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("jpg: %s", err)
	}
	processor := gift.New(
		gift.Sobel(),
		gift.Grayscale(),
	)
	processor.Options = gift.Options{Parallelization: true}
	dst := image.NewRGBA(processor.Bounds(src.Bounds()))
	processor.Draw(dst, src)
	//done
	s.processed = dst
	//count white in processed
	s.n = s.countWhite()
	return s, nil
}

func (s *snap) countWhite() int {
	//n is the number of pure white pixels
	n := 0
	bounds := s.processed.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := s.processed.At(x, y).RGBA()
			if r == 0xffff && g == 0xffff && b == 0xffff {
				n++
			}
		}
	}
	return n
}
