package castle

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/boltdb/bolt"
	"github.com/dkumor/acmewrapper"
	"github.com/jpillora/castlebot/castle/acmedb"
	"github.com/jpillora/castlebot/castle/data"
	"github.com/jpillora/castlebot/castle/gpio"
	"github.com/jpillora/castlebot/castle/settings"
	"github.com/jpillora/castlebot/castle/static"
	"github.com/jpillora/overseer"
	"github.com/jpillora/velox"
)

func Run(version string, config Config, state overseer.State) error {
	//validate config
	if config.DBLocation == "" {
		return errors.New("database location is required")
	}
	//sync data
	data := &data.Data{
		Version: version,
	}
	//setup database
	db, err := bolt.Open(config.DBLocation, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()
	//database dependants
	s := settings.New(db)
	data.Settings = s
	//setup routes
	router := http.NewServeMux()
	router.Handle("/sync", velox.SyncHandler(data))
	router.Handle("/js/velox.js", velox.JS)
	router.Handle("/gpio", gpio.New())
	router.Handle("/settings", s)
	router.Handle("/", static.Handler())
	//setup tls/tcp listener
	var listener net.Listener
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	if config.TLSHostname != "" {
		adb := acmedb.DB{DB: db}
		w, err := acmewrapper.New(acmewrapper.Config{
			Domains:          []string{config.TLSHostname},
			Address:          fmt.Sprintf(":%d", config.Port),
			TLSCertFile:      "cert.pem",
			TLSKeyFile:       "key.pem",
			RegistrationFile: "user.reg",
			PrivateKeyFile:   "user.pem",
			Email:            config.TLSEmail,
			TOSCallback:      acmewrapper.TOSAgree,
			SaveFileCallback: adb.SaveFileCallback,
			LoadFileCallback: adb.LoadFileCallback,
		})
		if err != nil {
			return err
		}
		l, err := tls.Listen("tcp", addr, w.TLSConfig())
		if err != nil {
			return err
		}
		log.Printf("Listening https://%s:%d", config.TLSHostname, config.Port)
		listener = l
	} else {
		l, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		log.Printf("Listening http://%s...", addr)
		listener = l
	}
	//serve router via listener
	return http.Serve(listener, router)
}
