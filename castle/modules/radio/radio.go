package radio

import (
	"log"
	"net/http"
	"strconv"

	"golang.org/x/net/context"

	"github.com/jpillora/go433"

	goji "goji.io"
	"goji.io/pat"
)

func New() *Radio {
	a := &Radio{}
	return a
}

type Radio struct {
	settings struct {
	}
}

func (rd *Radio) ID() string {
	return "radio"
}

func (rd *Radio) RegisterRoutes(mux *goji.Mux) {
	mux.HandleC(pat.Get("/send"), goji.HandlerFunc(rd.send))
}

func (rd *Radio) send(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseUint(r.URL.Query().Get("code"), 10, 32)
	if err != nil {
		http.Error(w, "invalid code", 400)
		return
	}
	if err := go433.Send(17, uint32(code)); err != nil {
		log.Printf("[radio] send error: %s", err)
		http.Error(w, "send failed", 400)
		return
	}
	log.Printf("[radio] sent: %d", code)
	w.Write([]byte("success"))
}

// func (a *Radio) Get() interface{} {
// 	return &a.settings
// }

// func (a *Radio) Set(j json.RawMessage) error {
// 	if err := json.Unmarshal(j, &a.settings); err != nil {
// 		return err
// 	}
// 	//do stuff
// 	return nil
// }
