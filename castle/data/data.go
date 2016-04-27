package data

import (
	"sync"

	"github.com/jpillora/castlebot/castle/settings"
	"github.com/jpillora/velox"
)

type Data struct {
	velox.State
	sync.Mutex
	Version  string             `json:"version"`
	Settings *settings.Settings `json:"settings"`
}
