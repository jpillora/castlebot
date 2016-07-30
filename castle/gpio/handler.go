package gpio

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/stianeikeland/go-rpio"
)

func New() http.Handler {
	err := rpio.Open()
	return &handler{active: err == nil}
}

type handler struct {
	active bool
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.active {
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
	//actuate
	go func() {
		pin := rpio.Pin(p)
		pin.Output()
		pin.High()
		time.Sleep(d)
		pin.Low()
		log.Printf("activated pin %d for %s\n", p, d)
	}()
	//done
	w.WriteHeader(200)
	fmt.Fprintf(w, "activating pin %d for %s\n", p, d)
}
