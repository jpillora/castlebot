package gpio

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jpillora/go433"
	goji "goji.io"
	"goji.io/pat"
)

func New() *GPIO {
	return &GPIO{Enabled: false}
}

type GPIO struct {
	Enabled bool `json:"enabled"`
}

func (h *GPIO) ID() string {
	return "gpio"
}

func (h *GPIO) Get() interface{} {
	return h
}

func (h *GPIO) Set(j json.RawMessage) error {
	if j != nil {
		if err := json.Unmarshal(j, &h); err != nil {
			return err
		}
	}
	return nil
}

func (g *GPIO) RegisterRoutes(mux *goji.Mux) {
	mux.Handle(pat.Get("/actuate"), http.HandlerFunc(g.actuate))
}

func (h *GPIO) actuate(w http.ResponseWriter, r *http.Request) {
	if !h.Enabled {
		http.Error(w, "GPIO not active", 500)
		return
	}
	q := r.URL.Query()
	var err error
	p := 17
	if s := q.Get("p"); s != "" {
		p, err = strconv.Atoi(s)
		if err != nil {
			http.Error(w, "Invalid pin", 400)
			return
		}
	}
	d := 1 * time.Second
	if s := q.Get("d"); s != "" {
		d, err = time.ParseDuration(s)
		if err != nil {
			http.Error(w, "Invalid duration", 400)
			return
		}
	}
	pin, err := go433.OpenPinOut(p)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	//actuate
	go func() {
		pin.Write(true)
		time.Sleep(d)
		pin.Write(false)
		log.Printf("activated pin %d for %s\n", p, d)
	}()
	//done
	w.WriteHeader(200)
	fmt.Fprintf(w, "activating pin %d for %s\n", p, d)
}
