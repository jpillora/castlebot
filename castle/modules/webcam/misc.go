package webcam

import (
	"path"
	"time"
)

func abs(n int) int {
	if n < 0 {
		return n * -1
	}
	return n
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var sydney, _ = time.LoadLocation("Australia/Sydney")

func dateDir(base string, t time.Time) string {
	dir := t.In(sydney).Format("2006-01-02")
	return path.Join(base, dir)
}

func timeJpg(dir string, t time.Time) string {
	filename := t.In(sydney).Format("15-04-05.000") + ".jpg"
	return path.Join(dir, filename)
}
