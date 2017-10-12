package main

import (
	"log"
	"strconv"
	"time"

	"github.com/jpillora/castlebot/castle"
	"github.com/jpillora/opts"
	"github.com/jpillora/overseer"
	"github.com/jpillora/overseer/fetcher"
)

//initial config
var config = castle.Config{
	DB:      "castle.db",
	Name:    "Castlebot",
	Port:    3000,
	Updates: false,
}

//BuildTime will be set by the compiler
var (
	BuildTime = strconv.FormatInt(time.Now().Unix(), 10)
)

func main() {
	overseer.SanityCheck()
	//parse config
	opts.New(&config).
		Name("castle").
		Version(BuildTime).
		PkgRepo().
		Parse()
	//no overseer
	if !config.Updates {
		prog(overseer.DisabledState)
		return
	}
	//start overseer
	overseer.Run(overseer.Config{
		Program: prog,
		Fetcher: &fetcher.Github{
			User: "jpillora",
			Repo: "castlebot",
		},
		Debug: true,
	})
}

func prog(state overseer.State) {
	if err := castle.Run(BuildTime, config, state); err != nil {
		log.Fatal(err)
	}
}
