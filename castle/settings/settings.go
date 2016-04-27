package settings

import (
	"net/http"

	"github.com/boltdb/bolt"
)

type Settings struct {
	db *bolt.DB
}

func New(db *bolt.DB) *Settings {
	return &Settings{db: db}
}

func (s *Settings) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}
