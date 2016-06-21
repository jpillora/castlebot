package settings

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"
	"sync"

	"goji.io/pat"
	"golang.org/x/net/context"

	"github.com/boltdb/bolt"
	"github.com/jpillora/velox"
)

type Module interface {
	Get() interface{}
	Set(json.RawMessage) error
}

type Settings struct {
	db      *bolt.DB
	modules map[string]Module
	Data    struct {
		velox.State
		sync.Mutex
		Version   string                 `json:"version"`
		GoVersion string                 `json:"goVersion"`
		Settings  map[string]interface{} `json:"settings"`
	}
}

func New(version string, db *bolt.DB) *Settings {
	s := &Settings{}
	s.db = db
	s.modules = map[string]Module{}
	s.Data.Version = version
	s.Data.GoVersion = runtime.Version()
	s.Data.Settings = map[string]interface{}{}
	return s
}

func (s *Settings) Register(id string, m Module) {
	if _, ok := s.modules[id]; ok {
		log.Fatalf("already registered: %s", id)
	}
	//load from db?
	b := s.dbget(id)
	if len(b) > 0 {
		log.Printf("loaded existing config for: %s", id)
		m.Set(json.RawMessage(b))
	} else {
		m.Set(nil) //signal use defaults
	}
	//initial value
	j := m.Get()
	s.Data.Settings[id] = j
	//register
	s.modules[id] = m
}

func (s *Settings) Update(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	id := pat.Param(ctx, "id")
	m, ok := s.modules[id]
	if !ok {
		http.Error(w, "Module not found: "+id, http.StatusNotFound)
		return
	}
	var j json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&j); err != nil {
		http.Error(w, "Expecting valid JSON", http.StatusBadRequest)
		return
	}
	//pass to module
	if err := m.Set(j); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//success! store in db
	if err := s.dbset(id, []byte(j)); err != nil {
		log.Printf("failed to store: %s: %s", id, err)
	}
	v := m.Get()
	s.Data.Settings[id] = v
	s.Data.Push()
}

var bucketName = []byte("settings")

func (s *Settings) dbget(key string) (contents []byte) {
	s.db.View(func(tx *bolt.Tx) error {
		if b := tx.Bucket(bucketName); b != nil {
			contents = b.Get([]byte(key))
		}
		return nil
	})
	return
}

func (s *Settings) dbset(key string, contents []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}
		return b.Put([]byte(key), []byte(contents))
	})
}
