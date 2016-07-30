package castle

import (
	"errors"
	"net/http"

	"goji.io/pat"

	"goji.io"

	"github.com/boltdb/bolt"
	"github.com/jpillora/castlebot/castle/gpio"
	"github.com/jpillora/castlebot/castle/server"
	"github.com/jpillora/castlebot/castle/settings"
	"github.com/jpillora/castlebot/castle/static"
	"github.com/jpillora/castlebot/castle/webcam"
	"github.com/jpillora/overseer"
	"github.com/jpillora/requestlog"
	"github.com/jpillora/velox"
	"github.com/zenazn/goji/web/middleware"
)

//Config defines the command-line interface, it contains just enough to access
//the web ui to change the settings database
type Config struct {
	DB        string `help:"castle settings database location"`
	Port      int    `help:"http listening port, used when not found in settings"`
	NoUpdates bool   `help:"disable automatic updates"`
}

func Run(version string, config Config, state overseer.State) error {
	//validate config
	if config.DB == "" {
		return errors.New("database location is required")
	}
	//setup database
	db, err := bolt.Open(config.DB, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()
	//settings
	set := settings.New(version, db)
	//webcam
	wc := webcam.New(db)
	//http
	router := goji.NewMux()
	serv := server.New(db, router, config.Port)
	//setup routes
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
	router.Handle(pat.Get("/sync"), velox.SyncHandler(&set.Data))
	router.Handle(pat.Get("/js/velox.js"), velox.JS)
	router.Handle(pat.Get("/gpio"), gpio.New())
	router.HandleC(pat.Put("/settings/:id"), goji.HandlerFunc(set.Update))
	router.HandleC(pat.Get("/webcam/snaps"), goji.HandlerFunc(wc.List))
	router.HandleC(pat.Get("/webcam/snap/:id"), goji.HandlerFunc(wc.GetHistorical))
	router.HandleC(pat.Get("/webcam/live/:index/:type"), goji.HandlerFunc(wc.GetLive))
	router.HandleC(pat.Get("/webcam/live/:index"), goji.HandlerFunc(wc.GetLive))
	router.Handle(pat.New("/*"), static.Handler())
	//register all modules
	set.Register("server", serv)
	set.Register("webcam", wc)
	//wait till closed
	return serv.Wait()
}
