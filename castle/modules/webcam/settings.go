package webcam

import (
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/jpillora/castlebot/castle/util"
)

type settings struct {
	Enabled     bool          `json:"enabled"`
	Host        string        `json:"host"`
	User        string        `json:"user"`
	Pass        string        `json:"pass"`
	Interval    util.Duration `json:"interval"`
	Threshold   int           `json:"threshold"`
	DropboxAPI  string        `json:"dropboxApi"`
	DropboxBase string        `json:"dropboxBase"`
	DiskBase    string        `json:"diskBase"`
	DiskForce   bool          `json:"diskForce"`
}

func (w *Webcam) Get() interface{} {
	return &w.settings
}

func (w *Webcam) Set(j json.RawMessage) error {
	if j != nil {
		if err := json.Unmarshal(j, &w.settings); err != nil {
			return err
		}
	}
	if w.settings.Interval.D() < 100*time.Millisecond {
		w.settings.Interval = util.Duration(100 * time.Millisecond)
	}
	ready := false
	if w.settings.Host != "" {
		origin := "http://" + w.settings.Host + "/"
		if _, err := url.Parse(origin); err != nil {
			return errors.New("Invalid host")
		}
		w.origin = origin
		ready = true
	}
	if !ready {
		w.settings.Enabled = false
	}
	//validate disk
	if base := w.settings.DiskBase; base != "" {
		s, err := os.Stat(base)
		if err != nil {
			return errors.New("Invalid disk base")
		} else if !s.IsDir() {
			return errors.New("Invalid disk base dir")
		}
	}
	//validate dropbox
	if w.settings.DropboxBase == "" {
		w.settings.DropboxBase = "/"
	}
	//close last one
	if w.dropcam != nil {
		w.dropcam.close()
		w.dropcam = nil
	}
	//initialise dropbox
	if api := w.settings.DropboxAPI; api != "" {
		base := w.settings.DropboxBase
		dc, err := newDropcam(api, base)
		if err != nil {
			log.Printf("[webcam] dropbox login failed: %s", err)
			return errors.New("Dropbox login failed")
		}
		w.dropcam = dc
	}
	//do check now!
	w.timer.Reset(0)
	return nil
}
