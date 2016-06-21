package castle

import (
	"errors"

	"goji.io/pat"

	"goji.io"

	"github.com/boltdb/bolt"
	"github.com/jpillora/castlebot/castle/gpio"
	"github.com/jpillora/castlebot/castle/server"
	"github.com/jpillora/castlebot/castle/settings"
	"github.com/jpillora/castlebot/castle/static"
	"github.com/jpillora/castlebot/castle/webcam"
	"github.com/jpillora/overseer"
	"github.com/jpillora/velox"
)

//Config defines the command-line interface, it contains just enough to access
//the web ui to change the settings database
type Config struct {
	SettingsLocation string `help:"castle settings database location"`
	Port             int    `help:"http listening port, used when not found in settings"`
	NoUpdates        bool   `help:"disable automatic updates"`
}

func Run(version string, config Config, state overseer.State) error {
	//validate config
	if config.SettingsLocation == "" {
		return errors.New("database location is required")
	}
	//setup database
	db, err := bolt.Open(config.SettingsLocation, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()
	//settings
	set := settings.New(version, db)
	//webcam
	wc := webcam.New(db)
	//setup routes
	router := goji.NewMux()
	router.Handle(pat.Get("/sync"), velox.SyncHandler(&set.Data))
	router.Handle(pat.Get("/js/velox.js"), velox.JS)
	router.Handle(pat.Get("/gpio"), gpio.New())
	router.HandleC(pat.Put("/settings/:id"), goji.HandlerFunc(set.Update))
	router.HandleC(pat.Get("/webcam/snaps"), goji.HandlerFunc(wc.List))
	router.Handle(pat.New("/*"), static.Handler())
	//http server
	serv := server.New(db, router, config.Port)
	//register all modules
	set.Register("server", serv)
	set.Register("webcam", wc)
	//wait till closed
	return serv.Wait()
}
