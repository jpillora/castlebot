package webcam

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"goji.io/pat"

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
	w.Config.Interval = 15
	go w.check()
	return w
}

type Webcam struct {
	db     *bolt.DB
	timer  *time.Timer
	snaps  []*snap
	Config struct {
		StoredBytes int64  `json:"storedBytes"`
		ImageURL    string `json:"imageUrl"`
		Interval    int    `json:"interval"`
	}
}

func (w *Webcam) check() {
	b := backoff.Backoff{Max: 5 * time.Minute}
	for {
		w.timer.Reset(time.Duration(w.Config.Interval) * time.Second)
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
	//no image url defined
	if w.Config.ImageURL == "" {
		return nil
	}
	resp, err := http.Get(w.Config.ImageURL)
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
	l := len(w.snaps)
	var last *snap
	if l > 0 {
		last = w.snaps[l-1]
	}
	//store last 100 snaps
	w.snaps = append(w.snaps, curr)
	if l+1 == 100 {
		w.snaps = w.snaps[1:]
	}
	//compare last to current, if changed much, store both
	if last != nil {
		diff := abs(last.n - curr.n)
		// log.Printf("[webcam] diff: %v", diff)
		if diff > 300 {
			w.store(last)
			w.store(curr)
		}
	}
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
	return &w.Config
}

func (w *Webcam) Set(j json.RawMessage) error {
	if j == nil {
		//use defaults
		w.Config.StoredBytes = 500e9 //500MB
	} else {
		if err := json.Unmarshal(j, &w.Config); err != nil {
			return err
		}
	}
	if w.Config.Interval < 1 {
		w.Config.Interval = 1
	}
	//do check now!
	w.timer.Reset(0)
	return nil
}

func (w *Webcam) List(ctx context.Context, writer http.ResponseWriter, r *http.Request) {
	n := 0
	if err := w.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			fmt.Fprintf(writer, "bucket missing\n")
			return nil
		}
		c := b.Cursor()
		for {
			k, v := c.Next()
			if v == nil {
				break
			}
			fmt.Fprintf(writer, "%s = %d\n", k, len(v))
			n++
		}
		return nil
	}); err != nil {
		http.Error(writer, "db view failed", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(writer, "done (#%d)\n", n)
}

func (w *Webcam) GetSnap(ctx context.Context, writer http.ResponseWriter, r *http.Request) {
	id := pat.Param(ctx, "id")
	var snap []byte
	if err := w.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			fmt.Fprintf(writer, "bucket missing\n")
			return nil
		}
		snap = b.Get([]byte(id))
		return nil
	}); err != nil {
		http.Error(writer, "db view failed", http.StatusInternalServerError)
		return
	}
	if len(snap) == 0 {
		http.NotFound(writer, r)
		return
	}
	http.ServeContent(writer, r, id+".jpg", time.Now(), bytes.NewReader(snap))
}
