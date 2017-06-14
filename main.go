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

//VERSION and BUILDTIME will be set by the compiler
var (
	VERSION   = ""
	BUILDTIME = ""
)

func main() {
	overseer.SanityCheck()
	//displayed version
	v := VERSION
	if v == "" {
		v = BUILDTIME
	}
	//parse config
	opts.New(&config).
		Name("castle").
		Version(v).
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
	if err := castle.Run(VERSION, BUILDTIME, config, state); err != nil {
		log.Fatal(err)
	}
}
