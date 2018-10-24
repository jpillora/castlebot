package castle

import (
	"errors"
	"log"
	"net/http"
	"net/http/pprof"
	"runtime"
	"strconv"
	"sync"
	"time"

	"goji.io"
	"goji.io/pat"

	"github.com/boltdb/bolt"
	"github.com/jpillora/castlebot/castle/modules"
	"github.com/jpillora/castlebot/castle/modules/auth"
	"github.com/jpillora/castlebot/castle/modules/gpio"
	"github.com/jpillora/castlebot/castle/modules/machine"
	"github.com/jpillora/castlebot/castle/modules/radio"
	"github.com/jpillora/castlebot/castle/modules/scanner"
	"github.com/jpillora/castlebot/castle/modules/server"
	"github.com/jpillora/castlebot/castle/modules/webcam"
	"github.com/jpillora/castlebot/castle/static"
	"github.com/jpillora/overseer"
	"github.com/jpillora/requestlog"
	"github.com/jpillora/velox"
)

//Config defines the command-line interface, it contains just enough to access
//the web ui to change the settings database
type Config struct {
	DB      string `help:"castle settings database location"`
	Name    string `help:"name of bot"`
	Port    int    `help:"http listening port, used when not found in settings"`
	Updates bool   `help:"enable automatic updates"`
}

func Run(buildtime string, config Config, state overseer.State) error {
	//validate config
	if config.DB == "" {
		return errors.New("database location is required")
	}
	//setup database
	log.Printf("Open database %s", config.DB)
	db, err := bolt.Open(config.DB, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()
	//velox state
	data := struct {
		velox.State
		sync.Mutex
		Name      string      `json:"name"`
		UpTime    time.Time   `json:"upTime"`
		BuildTime time.Time   `json:"buildTime"`
		GoVersion string      `json:"goVersion"`
		Modules   interface{} `json:"modules"`
	}{
		Name:      config.Name,
		UpTime:    time.Now(),
		GoVersion: runtime.Version(),
	}
	if n, err := strconv.ParseInt(buildtime, 10, 64); err == nil {
		data.BuildTime = time.Unix(n, 0)
		log.Printf("build time: %s", data.BuildTime)
	}
	//root router
	router := goji.NewMux()
	//initialise module container
	m := modules.New(db, router, velox.Pusher(&data))
	data.Modules = m.JSON()
	//initialise modules
	a := auth.New()
	serv := server.New(db, router, config.Port)
	mods := []modules.Identified{
		serv,
		a,
		gpio.New(),
		scanner.New(),
		webcam.New(db),
		machine.New(),
		radio.New(),
	}
	//HACK: let goroutines kick in
	time.Sleep(50 * time.Millisecond)
	//setup middleware
	router.Use(requestlog.Wrap)
	router.Use(a.CookieAuth.Wrap)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//tls hostname check
			if r.TLS != nil && r.TLS.ServerName != serv.Config.HTTPS.Hostname {
				http.NotFound(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
	//register all modules
	for _, mod := range mods {
		m.Register(mod)
	}
	//setup admin routes
	router.Handle(pat.Get("/admin/pprof"), http.HandlerFunc(pprof.Index))
	router.Handle(pat.Get("/admin/pprof/cmdline"), http.HandlerFunc(pprof.Cmdline))
	router.Handle(pat.Get("/admin/pprof/profile"), http.HandlerFunc(pprof.Profile))
	router.Handle(pat.Get("/admin/pprof/symbol"), http.HandlerFunc(pprof.Symbol))
	router.Handle(pat.Get("/admin/pprof/:name"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := pprof.Handler(pat.Param(r, "name"))
		h.ServeHTTP(w, r)
	}))
	//setup velox/static file routes
	router.Handle(pat.Get("/sync"), velox.SyncHandler(&data))
	router.Handle(pat.Get("/js/velox.js"), velox.JS)
	router.Handle(pat.New("/*"), static.Handler())
	//wait till closed
	return serv.Wait()
}
