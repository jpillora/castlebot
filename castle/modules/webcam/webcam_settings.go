package webcam

import (
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"time"

	"github.com/jpillora/castlebot/castle/util"
	dropbox "github.com/jpillora/go-dropbox"
)

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
	if w.settings.StoredBytes < 1e9 {
		w.settings.StoredBytes = 500e9 //500MB
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
	//validate dropbox
	if w.settings.DropboxBase == "" {
		w.settings.DropboxBase = "/"
	}
	if w.drop.queue != nil {
		close(w.drop.queue) //kill old queue
		w.drop.queue = nil
	}
	if api := w.settings.DropboxAPI; api != "" {
		//create dropbox client and test authentication
		client := dropbox.New(dropbox.NewConfig(api))
		u, err := client.Users.GetCurrentAccount()
		if err == nil {
			log.Printf("[webcam] dropbox user: %s", u.Name)
			w.drop.client = client
			w.drop.queue = make(chan *snap)
			go w.dropdeque()
		} else {
			w.drop.client = nil
		}
	}
	//do check now!
	w.timer.Reset(0)
	return nil
}
