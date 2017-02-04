package webcam

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	goji "goji.io"

	"goji.io/pat"
	"goji.io/pattern"

	"golang.org/x/net/context"

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
	db       *bolt.DB
	timer    *time.Timer
	snaps    []*snap
	settings struct {
		Enabled     bool   `json:"enabled"`
		StoredBytes int64  `json:"storedBytes"`
		ImageURL    string `json:"imageUrl"`
		Interval    int    `json:"interval"`
		Threshold   int    `json:"threshold"`
	}
}

func (w *Webcam) ID() string {
	return "webcam"
}

func (w *Webcam) check() {
	b := backoff.Backoff{Max: 5 * time.Minute}
	for {
		w.timer.Reset(time.Duration(w.settings.Interval) * time.Second)
		<-w.timer.C
		//take snap, process, store, etc
		if err := w.snap(); err != nil {
			log.Printf("[webcam] snap failed: %s", err)
			time.Sleep(b.Duration())
		} else {
			b.Reset()
		}
	}
}

func (w *Webcam) snap() error {
	//disabled
	if !w.settings.Enabled {
		return nil
	}
	//no image url defined
	if w.settings.ImageURL == "" {
		return nil
	}
	log.Printf("GET %s", w.settings.ImageURL)
	resp, err := http.Get(w.settings.ImageURL)
	if err != nil {
		return fmt.Errorf("request: %s", err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("download: %s", err)
	}
	//find last
	l := len(w.snaps)
	var last *snap
	if l > 0 {
		last = w.snaps[l-1]
	}
	//create snap
	curr, err := newSnap(b, w.settings.Threshold, last)
	if err != nil {
		return err
	}
	//store last 100 snaps
	w.snaps = append(w.snaps, curr)
	if l+1 == 100 {
		w.snaps = w.snaps[1:]
	}
	//compare last to current, if changed much, store both
	//TODO restore
	// if curr.pdiffNum > 4000 {
	// 	w.store(curr)
	// }
	return nil
}

var bucketName = []byte("snaps")

func (w *Webcam) store(s *snap) {
	if s.stored {
		return
	}
	if err := w.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}
		return b.Put([]byte(s.id), s.raw)
	}); err != nil {
		log.Printf("[webcam] db write failed: %s", err)
		return
	}
	log.Printf("[webcam] wrote snap %s", s.id)
	s.stored = true
}

func (w *Webcam) Get() interface{} {
	return &w.settings
}

func (w *Webcam) Set(j json.RawMessage) error {
	if j == nil {
		//use defaults
		w.settings.StoredBytes = 500e9 //500MB
	} else {
		if err := json.Unmarshal(j, &w.settings); err != nil {
			return err
		}
	}
	if w.settings.Interval < 1 {
		w.settings.Interval = 1
	}
	//do check now!
	w.timer.Reset(0)
	return nil
}

func (wc *Webcam) RegisterRoutes(mux *goji.Mux) {
	mux.HandleC(pat.Get("/snaps"), goji.HandlerFunc(wc.getList))
	mux.HandleC(pat.Get("/snap/:id"), goji.HandlerFunc(wc.getHistorical))
	mux.HandleC(pat.Get("/live/:index/:type"), goji.HandlerFunc(wc.getLive))
	mux.HandleC(pat.Get("/live/:index"), goji.HandlerFunc(wc.getLive))
}

func (wc *Webcam) getList(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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

func (wc *Webcam) getLive(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	i, err := strconv.Atoi(pat.Param(ctx, "index"))
	if err != nil {
		http.Error(w, "index must be an integer", http.StatusBadRequest)
		return
	}
	total := len(wc.snaps)
	if total == 0 || i <= 0 || i > total {
		http.Error(w, "index out of range", http.StatusBadRequest)
		return
	}
	//reverse index order
	s := wc.snaps[total-1-i]
	w.Header().Set("Interval", strconv.Itoa(wc.settings.Interval))
	//find image
	var b []byte

	snaptype := ""
	if v := ctx.Value(pattern.Variable("type")); v != nil {
		if t, ok := v.(string); ok {
			snaptype = t
		}
	}
	switch snaptype {
	case "diff":
		b = s.diff
	default:
		b = s.raw
	}
	//serve
	http.ServeContent(w, r, string(s.id)+".jpg", s.t, bytes.NewReader(b))
}

func (wc *Webcam) getHistorical(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	targetID := []byte(pat.Param(ctx, "id"))
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

func toID(t time.Time) []byte {
	return []byte(t.UTC().Format(time.RFC3339))
}

func fromID(id []byte) time.Time {
	t, _ := time.Parse(time.RFC3339, string(id))
	return t
}