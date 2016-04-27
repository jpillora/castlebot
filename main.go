package main

import (
	"log"
	"time"

	"github.com/jpillora/castlebot/castle"
	"github.com/jpillora/opts"
	"github.com/jpillora/overseer"
	"github.com/jpillora/overseer/fetcher"
)

//initial config
var config = castle.Config{
	DBLocation: "castle.db",
	Host:       "0.0.0.0",
	Port:       3000,
	NoUpdates:  true,
}

var VERSION = "0.0.0-src"

func main() {
	//parse config
	opts.New(&config).
		Name("castle").
		Version(VERSION).
		PkgRepo().
		Parse()
	//no overseer
	if config.NoUpdates {
		prog(overseer.DisabledState)
		return
	}
	//start overseer
	overseer.Run(overseer.Config{
		Program: prog,
		Fetcher: &fetcher.HTTP{
			URL:      "http://localhost:4000/binaries/myapp",
			Interval: 1 * time.Second,
		},
		Debug: true,
	})
}

func prog(state overseer.State) {
	if err := castle.Run(VERSION, config, state); err != nil {
		log.Fatal(err)
	}
}
