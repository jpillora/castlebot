package modules

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	goji "goji.io"

	"goji.io/pat"

	"github.com/boltdb/bolt"
	"github.com/jpillora/velox"
)

type Identified interface {
	ID() string
}

type Statusable interface {
	Status(chan interface{})
}

type Settable interface {
	Get() interface{}
	Set(json.RawMessage) error
}

type Routable interface {
	RegisterRoutes(*goji.Mux)
}

type Module struct {
	ID       string `json:"id"`
	settable Settable
	Settings interface{} `json:"settings,omitempty"`
	Status   interface{} `json:"status,omitempty"`
}

type Modules struct {
	db      *bolt.DB
	router  *goji.Mux
	modules map[string]*Module
	state   velox.Pusher
}

func New(db *bolt.DB, router *goji.Mux, state velox.Pusher) *Modules {
	s := &Modules{}
	s.db = db
	s.router = router
	s.modules = map[string]*Module{}
	s.state = state
	return s
}

func (s *Modules) JSON() interface{} {
	return s.modules
}

func (s *Modules) Register(rawModule Identified) {
	id := rawModule.ID()
	if _, ok := s.modules[id]; ok {
		log.Fatalf("already registered: %s", id)
	}
	module := &Module{
		ID:       id,
		Settings: nil,
		Status:   nil,
	}
	//register subrouter
	subrouter := goji.SubMux()
	s.router.Handle(pat.New("/m/"+id+"/*"), subrouter)
	//load module settings
	if settable, ok := rawModule.(Settable); ok {
		//load from db?
		b := s.dbget(id)
		if len(b) > 0 {
			log.Printf("loaded existing config for: %s", id)
			settable.Set(json.RawMessage(b))
		} else {
			settable.Set(nil) //signal use defaults
		}
		//initial value
		module.Settings = settable.Get()
		module.settable = settable
		//rest api
		subrouter.Handle(pat.Get("/settings"), s.getSettingsHandler(module))
		subrouter.Handle(pat.Put("/settings"), s.updateSettingsHandler(module))
	}
	//pass module status update channel
	if statuser, ok := rawModule.(Statusable); ok {
		updates := make(chan interface{})
		go s.watchUpdates(module, updates)
		go func() {
			time.Sleep(1 * time.Second)
			statuser.Status(updates)
		}()
	}
	//register module routers
	if routable, ok := rawModule.(Routable); ok {
		routable.RegisterRoutes(subrouter)
	}
	//register
	s.modules[id] = module
}

func (s *Modules) watchUpdates(module *Module, updates chan interface{}) {
	for update := range updates {
		module.Status = update
		s.state.Push()
	}
}

func (s *Modules) getSettingsHandler(module *Module) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := json.MarshalIndent(&module.Settings, "", "  ")
		if err != nil {
			http.Error(w, "Settings contain invalid JSON", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
		return
	}
}

func (s *Modules) updateSettingsHandler(module *Module) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var j json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&j); err != nil {
			http.Error(w, "Expecting valid JSON", http.StatusBadRequest)
			return
		}
		//pass to module
		if err := module.settable.Set(j); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		//success! store in db
		if err := s.dbset(module.ID, []byte(j)); err != nil {
			log.Printf("failed to store: %s: %s", module.ID, err)
		}
		module.Settings = module.settable.Get()
		log.Printf("updated settings: %s: %+v", module.ID, module.Settings)
		s.state.Push()
	}
}

var bucketName = []byte("settings")

func (s *Modules) dbget(key string) (contents []byte) {
	s.db.View(func(tx *bolt.Tx) error {
		if b := tx.Bucket(bucketName); b != nil {
			contents = b.Get([]byte(key))
		}
		return nil
	})
	return
}

func (s *Modules) dbset(key string, contents []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}
		return b.Put([]byte(key), []byte(contents))
	})
}
