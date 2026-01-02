package db

import (
	"fmt"
	"sync"

	bolt "go.etcd.io/bbolt"
)

var (
	sharedDB     *bolt.DB
	sharedDBOnce sync.Once
	sharedDBPath = "database.db"
)

func GetDB() (*bolt.DB, error) {
	var err error
	sharedDBOnce.Do(func() {
		sharedDB, err = bolt.Open(sharedDBPath, 0600, nil)
	})
	if err != nil {
		return nil, err
	}
	if sharedDB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return sharedDB, nil
}

func CloseDB() error {
	if sharedDB != nil {
		return sharedDB.Close()
	}
	return nil
}
