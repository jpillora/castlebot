package main

import (
	"log"

	"github.com/jpillora/castlebot/castle"
	"github.com/jpillora/opts"
	"github.com/jpillora/overseer"
	"github.com/jpillora/overseer/fetcher"
)

//initial config
var config = castle.Config{
	DB:      "castle.db",
	Port:    3000,
	Updates: false, //os.Getenv("DEV") == "1",
}

//VERSION is set by the compiler
var VERSION = "0.0.0-src"

func main() {
	overseer.SanityCheck()
	//parse config
	opts.New(&config).
		Name("castle").
		Version(VERSION).
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
	if err := castle.Run(VERSION, config, state); err != nil {
		log.Fatal(err)
	}
}
