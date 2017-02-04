package castle

import (
	"errors"
	"log"
	"net/http"
	"runtime"
	"sync"

	"goji.io/pat"

	"goji.io"

	"github.com/boltdb/bolt"
	"github.com/jpillora/castlebot/castle/modules"
	"github.com/jpillora/castlebot/castle/modules/gpio"
	"github.com/jpillora/castlebot/castle/modules/scanner"
	"github.com/jpillora/castlebot/castle/modules/server"
	"github.com/jpillora/castlebot/castle/modules/webcam"
	"github.com/jpillora/castlebot/castle/static"
	"github.com/jpillora/overseer"
	"github.com/jpillora/requestlog"
	"github.com/jpillora/velox"
	"github.com/zenazn/goji/web/middleware"
)

//Config defines the command-line interface, it contains just enough to access
//the web ui to change the settings database
type Config struct {
	DB      string `help:"castle settings database location"`
	Port    int    `help:"http listening port, used when not found in settings"`
	Updates bool   `help:"enable automatic updates"`
}

func Run(version string, config Config, state overseer.State) error {
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
		Version   string      `json:"version"`
		GoVersion string      `json:"goVersion"`
		Modules   interface{} `json:"modules"`
	}{
		Version:   version,
		GoVersion: runtime.Version(),
	}
	//root router
	router := goji.NewMux()
	//initialise module container
	m := modules.New(db, router, velox.Pusher(&data))
	data.Modules = m.JSON()
	//initialise modules
	g := gpio.New()
	sc := scanner.New()
	serv := server.New(db, router, config.Port)
	wc := webcam.New(db)
	//setup middleware
	router.Use(middleware.RealIP)
	router.Use(requestlog.Wrap)
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
	m.Register(g)
	m.Register(sc)
	m.Register(serv)
	m.Register(wc)
	//setup velox/static file routes
	router.Handle(pat.Get("/sync"), velox.SyncHandler(&data))
	router.Handle(pat.Get("/js/velox.js"), velox.JS)
	router.Handle(pat.New("/*"), static.Handler())
	//wait till closed
	return serv.Wait()
}
