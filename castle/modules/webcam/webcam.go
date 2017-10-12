package webcam

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	goji "goji.io"
	"goji.io/pat"

	"github.com/boltdb/bolt"
	"github.com/jpillora/backoff"
	"github.com/jpillora/castlebot/castle/util"
	dropbox "github.com/jpillora/go-dropbox"
)

func New(db *bolt.DB) *Webcam {
	w := &Webcam{}
	w.db = db
	w.timer = time.NewTimer(time.Duration(0))
	w.timer.Stop()
	w.snaps = []*snap{}
	w.drop.queue = nil
	w.drop.client = nil
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
	drop      struct {
		queue   chan *snap
		client  *dropbox.Client
		lastDir string
	}
	settings struct {
		Enabled     bool          `json:"enabled"`
		StoredBytes int64         `json:"storedBytes"`
		Host        string        `json:"host"`
		User        string        `json:"user"`
		Pass        string        `json:"pass"`
		Interval    util.Duration `json:"interval"`
		Threshold   int           `json:"threshold"`
		DropboxAPI  string        `json:"dropboxApi"`
		DropboxBase string        `json:"dropboxBase"`
	}
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
	//store locally to database
	if err := w.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}
		return b.Put([]byte(s.id), s.raw)
	}); err != nil {
		log.Printf("[webcam] db write failed: %s", err)
	}
	//store to dropbox (if connected)
	w.dropenque(s)
	//stored!
	log.Printf("[webcam] wrote snap %s (diff: %d)", s.id, s.pdiffNum)
}

func (wc *Webcam) RegisterRoutes(mux *goji.Mux) {
	mux.Handle(pat.Get("/snaps"), http.HandlerFunc(wc.getList))
	mux.Handle(pat.Get("/snap/:id"), http.HandlerFunc(wc.getHistorical))
	mux.Handle(pat.Get("/live/:index/:type"), http.HandlerFunc(wc.getLive))
	mux.Handle(pat.Get("/live/:index"), http.HandlerFunc(wc.getLive))
	mux.Handle(pat.Put("/move/:dir"), http.HandlerFunc(wc.move))
}

func (wc *Webcam) getList(w http.ResponseWriter, r *http.Request) {
	n := 0
	if err := wc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			fmt.Fprintf(w, "bucket missing\n")
			return nil
		}
		c := b.Cursor()
		k, v := c.First()
		for k != nil && v != nil {
			fmt.Fprintf(w, "%s = %d\n", k, len(v))
			n++
			k, v = c.Next()
		}
		return nil
	}); err != nil {
		http.Error(w, "db view failed", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "done (#%d)\n", n)
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

func (wc *Webcam) getHistorical(w http.ResponseWriter, r *http.Request) {
	targetID := []byte(pat.Param(r, "id"))
	if err := wc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			fmt.Fprintf(w, "bucket missing\n")
			return nil
		}
		c := b.Cursor()
		h := w.Header()
		//after last?
		lastID, lastImg := c.Last()
		if bytes.Compare(targetID, lastID) >= 0 {
			h.Set("Curr", string(lastID))
			prevID, _ := c.Prev()
			h.Set("Prev", string(prevID))
			http.ServeContent(w, r, string(lastID)+".jpg", fromID(lastID), bytes.NewReader(lastImg))
			return nil
		}
		//before first?
		firstID, firstImg := c.First()
		if bytes.Compare(firstID, targetID) >= 0 {
			h.Set("Curr", string(firstID))
			nextID, _ := c.Next()
			h.Set("Next", string(nextID))
			http.ServeContent(w, r, string(firstID)+".jpg", fromID(firstID), bytes.NewReader(firstImg))
			return nil
		}
		//middle
		currID, currImg := c.Seek(targetID)
		h.Set("Curr", string(currID))
		prevID, _ := c.Prev()
		h.Set("Prev", string(prevID))
		c.Next()
		nextID, _ := c.Next()
		h.Set("Next", string(nextID))
		http.ServeContent(w, r, string(currID)+".jpg", fromID(currID), bytes.NewReader(currImg))
		return nil
	}); err != nil {
		http.Error(w, "db view failed", http.StatusInternalServerError)
		return
	}
}

func (wc *Webcam) move(w http.ResponseWriter, r *http.Request) {
	dir := pat.Param(r, "dir")
	cmd := ""
	switch dir {
	case "up":
		cmd = "0"
	case "down":
		cmd = "2"
	case "right":
		cmd = "4"
	case "left":
		cmd = "6"
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
