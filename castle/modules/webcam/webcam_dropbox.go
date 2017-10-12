package webcam

import (
	"bytes"
	"log"
	"path"
	"strings"
	"time"

	dropbox "github.com/jpillora/go-dropbox"
)

func (w *Webcam) dropenque(s *snap) {
	if w.drop.queue != nil {
		w.drop.queue <- s
	}
}

func (w *Webcam) dropdeque() {
	for s := range w.drop.queue {
		w.dropupload(s)
	}
}

var sydney, _ = time.LoadLocation("Australia/Sydney")

func (w *Webcam) dropupload(s *snap) {
	c := w.drop.client
	if c == nil {
		return
	}
	dir := s.t.In(sydney).Format("2006-01-02")
	baseDir := path.Join(w.settings.DropboxBase, dir)
	if baseDir != w.drop.lastDir {
		if _, err := c.Files.CreateFolder(&dropbox.CreateFolderInput{
			Path: baseDir,
		}); err == nil {
			log.Printf("[webcam] created: %s", baseDir)
		} else if !strings.Contains(err.Error(), "path/conflict/folder") {
			log.Printf("[webcam] dropbox mkdir fail: %s", err)
			return
		}
		w.drop.lastDir = baseDir
	}
	filename := s.t.In(sydney).Format("15-04-05.000") + ".jpg"
	filepath := path.Join(baseDir, filename)
	log.Printf("[webcam] upload: %s", filepath)
	_, err := c.Files.Upload(&dropbox.UploadInput{
		Path:       filepath,
		Mode:       dropbox.WriteModeAdd,
		AutoRename: true,
		Mute:       true,
		Reader:     bytes.NewReader(s.raw),
	})
	if err != nil {
		log.Printf("[webcam] dropbox upload fail: %s", err)
	}
}
