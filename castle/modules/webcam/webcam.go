package webcam

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	goji "goji.io"
	"goji.io/pat"

	"github.com/boltdb/bolt"
	"github.com/jpillora/backoff"
)

func New(db *bolt.DB) *Webcam {
	w := &Webcam{}
	w.db = db
	w.timer = time.NewTimer(time.Duration(0))
	w.timer.Stop()
	w.snaps = []*snap{}
	w.settings.Enabled = false
	w.settings.Interval = 1
	w.settings.Threshold = 4000
	go w.check()
	return w
}

type Webcam struct {
	db        *bolt.DB
	timer     *time.Timer
	snaps     []*snap
	computing uint32
	computed  *snap
	origin    string
	dropcam   *dropcam
	settings  settings
}

func (w *Webcam) ID() string {
	return "webcam"
}

func (w *Webcam) check() {
	b := backoff.Backoff{Max: 5 * time.Minute}
	for {
		//take snap, process, store, etc
		t0 := time.Now()
		if err := w.snap(); err != nil {
			log.Printf("[webcam] snap failed: %s", err)
			time.Sleep(b.Duration())
		} else {
			b.Reset()
		}
		dur := time.Now().Sub(t0)
		interval := time.Duration(w.settings.Interval) - dur
		if interval < 0 {
			interval = 0
		}
		//set timer to
		w.timer.Reset(interval)
		//wait for timer
		<-w.timer.C
	}
}

func (w *Webcam) snap() error {
	//disabled
	if !w.settings.Enabled || w.origin == "" {
		return nil
	}
	//build url
	q := url.Values{}
	q.Set("user", w.settings.User)
	q.Set("pwd", w.settings.Pass)
	snapshotURL := w.origin + "/snapshot.cgi?" + q.Encode()
	//
	resp, err := http.Get(snapshotURL)
	if err != nil {
		return fmt.Errorf("request: %s", err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("download: %s", err)
	}
	//create snap
	curr, err := newSnap(b)
	if err != nil {
		return err
	}
	//store last 100 snaps
	if len(w.snaps) == 100 {
		w.snaps = append(w.snaps[1:], curr)
	} else {
		w.snaps = append(w.snaps, curr)
	}
	//attempt to mark compute in progress
	if atomic.CompareAndSwapUint32(&w.computing, 0, 1) {
		go w.computeDiff(curr)
	}
	return nil
}

func (w *Webcam) computeDiff(curr *snap) {
	//has previosu?
	if w.computed != nil {
		//compare with last computed
		diff := curr.computeDiff(w.settings.Threshold, w.computed)
		// log.Printf("compute: %s -> %s: %d", w.computed.id, curr.id, diff)
		if diff > w.settings.Threshold {
			//compare last to current, if changed much, store both
			go w.store(w.computed)
			go w.store(curr)
		}
	}
	w.computed = curr
	//mark complete
	atomic.StoreUint32(&w.computing, 0)
}

var bucketName = []byte("snaps")

func (w *Webcam) store(s *snap) {
	if s.stored {
		return
	}
	s.stored = true
	//store to dropbox
	enqueued := w.dropcam != nil && w.dropcam.enque(s)
	//store to disk?
	disk := w.settings.DiskBase != "" && (!enqueued || w.settings.DiskForce)
	if disk {
		dir := dateDir(w.settings.DiskBase, s.t)
		if s, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.Mkdir(dir, 0755); err != nil {
				log.Printf("mkdir dir failed: %s", err)
				return
			}
		} else if err != nil {
			log.Printf("stat dir failed: %s", err)
			return
		} else if !s.IsDir() {
			log.Printf("expected date dir")
			return
		}
		//write image into dir
		filepath := timeJpg(dir, s.t)
		if err := ioutil.WriteFile(filepath, s.raw, 0755); err != nil {
			log.Printf("write jpg failed: %s", err)
			return
		}
	}
	//stored!
	log.Printf("[webcam] wrote snap %s (diff: %d)", s.id, s.pdiffNum)
}

func (wc *Webcam) RegisterRoutes(mux *goji.Mux) {
	mux.Handle(pat.Get("/snap"), http.HandlerFunc(wc.getSnap))
	mux.Handle(pat.Get("/snap/:day"), http.HandlerFunc(wc.getSnap))
	mux.Handle(pat.Get("/snap/:day/:time"), http.HandlerFunc(wc.getSnap))
	mux.Handle(pat.Get("/live/:index/:type"), http.HandlerFunc(wc.getLive))
	mux.Handle(pat.Get("/live/:index"), http.HandlerFunc(wc.getLive))
	mux.Handle(pat.Put("/move/:dir"), http.HandlerFunc(wc.move))
}

func (wc *Webcam) getSnap(w http.ResponseWriter, r *http.Request) {
	base := wc.settings.DiskBase
	if base == "" {
		w.WriteHeader(404)
		w.Write([]byte("disk disabled"))
		return
	}
	//move base?
	day := pat.Param(r, "day")
	listFiles := day != ""
	if listFiles {
		base = filepath.Join(base, day)
	}
	//grab file?
	time := pat.Param(r, "time")
	readFile := time != ""
	//do
	if readFile {
		filepath := filepath.Join(base, time)
		b, err := ioutil.ReadFile(filepath)
		if err != nil {
			http.Error(w, "read file fail", 404)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(200)
		w.Write(b)
	} else {
		infos, err := ioutil.ReadDir(base)
		if err != nil {
			http.Error(w, "read dir fail", 404)
			return
		}
		dates := []string{}
		for _, info := range infos {
			if info.IsDir() {
				dates = append(dates, info.Name())
			}
		}
		b, _ := json.Marshal(dates)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(b)
	}
}

func (wc *Webcam) getLive(w http.ResponseWriter, r *http.Request) {
	istr := pat.Param(r, "index")
	i, err := strconv.Atoi(istr)
	if err != nil {
		http.Error(w, "index must be an integer", http.StatusBadRequest)
		return
	}
	total := len(wc.snaps)
	if total == 0 || i < 0 || i > total {
		http.Error(w, "index out of range: "+istr, http.StatusBadRequest)
		return
	}
	//reverse index order
	s := wc.snaps[total-1-i]
	intervalMs := wc.settings.Interval / 1e6
	w.Header().Set("Interval-Millis", strconv.Itoa(int(intervalMs)))
	//find image
	var b []byte
	snaptype := pat.Param(r, "type")
	switch snaptype {
	// case "diff":
	// 	b = s.diff
	default:
		b = s.raw
	}
	//serve
	http.ServeContent(w, r, string(s.id)+".jpg", s.t, bytes.NewReader(b))
}

func (wc *Webcam) move(w http.ResponseWriter, r *http.Request) {
	dir := pat.Param(r, "dir")
	cmd := ""
	switch dir {
	case "up":
		cmd = "2"
	case "down":
		cmd = "0"
	case "right":
		cmd = "6"
	case "left":
		cmd = "4"
	default:
		http.Error(w, "invalid dir", 400)
		return
	}
	//build url
	q := url.Values{}
	q.Set("loginuse", wc.settings.User)
	q.Set("loginpas", wc.settings.Pass)
	q.Set("command", cmd)
	q.Set("onestep", "1")
	decoderURL := wc.origin + "/decoder_control.cgi?" + q.Encode()
	resp, err := http.Get(decoderURL)
	if err != nil {
		http.Error(w, "req invalid", 400)
		return
	}
	if resp.StatusCode != 200 {
		http.Error(w, "req failed: "+resp.Status, 400)
		return
	}
	w.Write([]byte("success"))
	log.Printf("[webcam] move: %s", dir)
}

func toID(t time.Time) []byte {
	return []byte(t.UTC().Format(time.RFC3339))
}

func fromID(id []byte) time.Time {
	t, _ := time.Parse(time.RFC3339, string(id))
	return t
}
