package acmedb

import (
	"log"
	"os"

	"github.com/boltdb/bolt"
)

var bucketName = []byte("amcedb")

type DB struct {
	*bolt.DB
}

func (db *DB) SaveFileCallback(path string, contents []byte) error {
	log.Printf("save %s", path)
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}
		return b.Put([]byte(path), []byte(contents))
	})
}

func (db *DB) LoadFileCallback(path string) ([]byte, error) {
	log.Printf("load %s", path)
	var contents []byte
	if err := db.View(func(tx *bolt.Tx) error {
		if b := tx.Bucket(bucketName); b != nil {
			contents = b.Get([]byte(path))
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if len(contents) == 0 {
		return nil, os.ErrNotExist
	}
	return contents, nil
}
