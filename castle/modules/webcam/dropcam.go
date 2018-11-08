package webcam

import (
	"bytes"
	"log"
	"strings"

	dropbox "github.com/jpillora/go-dropbox"
)

const queueSize = 100

type dropcam struct {
	ready   bool
	base    string
	client  *dropbox.Client
	queue   chan *snap
	lastDir string
}

func newDropcam(api, base string) (*dropcam, error) {
	//create dropbox client and test authentication
	client := dropbox.New(dropbox.NewConfig(api))
	u, err := client.Users.GetCurrentAccount()
	if err != nil {
		return nil, err
	}
	log.Printf("[dropcam] dropbox user: %s", u.Name)
	dc := &dropcam{}
	dc.base = base
	dc.client = client
	dc.queue = make(chan *snap, queueSize)
	go dc.deque()
	dc.ready = true
	return dc, nil
}

func (w *dropcam) enque(s *snap) bool {
	if !w.ready || len(w.queue) == queueSize {
		return false //give up
	}
	//should never block
	w.queue <- s
	//enqueued!
	return true
}

func (w *dropcam) deque() {
	for s := range w.queue {
		w.upload(s)
	}
}

func (w *dropcam) upload(s *snap) {
	if !w.ready {
		return
	}
	baseDir := dateDir(w.base, s.t)
	if baseDir != w.lastDir {
		if _, err := w.client.Files.CreateFolder(&dropbox.CreateFolderInput{
			Path: baseDir,
		}); err == nil {
			log.Printf("[dropcam] created: %s", baseDir)
		} else if !strings.Contains(err.Error(), "path/conflict/folder") {
			log.Printf("[dropcam] dropbox mkdir fail: %s", err)
			return
		}
		w.lastDir = baseDir
	}
	filepath := timeJpg(baseDir, s.t)
	log.Printf("[dropcam] upload: %s", filepath)
	_, err := w.client.Files.Upload(&dropbox.UploadInput{
		Path:       filepath,
		Mode:       dropbox.WriteModeAdd,
		AutoRename: true,
		Mute:       true,
		Reader:     bytes.NewReader(s.raw),
	})
	if err != nil {
		log.Printf("[dropcam] dropbox upload fail: %s", err)
	}
	log.Printf("[dropcam] uploaded. %d remaining", len(w.queue))
}

func (w *dropcam) close() {
	close(w.queue)
	w.ready = false
}
