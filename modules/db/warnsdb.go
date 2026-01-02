package db

import (
	"encoding/json"
	"strconv"
	"time"

	bolt "go.etcd.io/bbolt"
)

type WarnAction string

const (
	WarnActionBan  WarnAction = "ban"
	WarnActionMute WarnAction = "mute"
	WarnActionKick WarnAction = "kick"
)

type Warn struct {
	Reason    string    `json:"reason"`
	AdminID   int64     `json:"admin_id"`
	Timestamp time.Time `json:"timestamp"`
}

type WarnSettings struct {
	MaxWarns  int        `json:"max_warns"`
	Action    WarnAction `json:"action"`
	DecayDays int        `json:"decay_days"`
}

func ensureWarnsBuckets(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("warns"))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte("warns_settings"))
		return err
	})
}

func AddWarn(chatID, userID int64, warn *Warn) (int, error) {
	db, err := GetDB()
	if err != nil {
		return 0, err
	}
	if err := ensureWarnsBuckets(db); err != nil {
		return 0, err
	}

	var count int

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("warns"))
		cb, err := b.CreateBucketIfNotExists([]byte(strconv.FormatInt(chatID, 10)))
		if err != nil {
			return err
		}
		ub, err := cb.CreateBucketIfNotExists([]byte(strconv.FormatInt(userID, 10)))
		if err != nil {
			return err
		}

		id, _ := ub.NextSequence()
		data, err := json.Marshal(warn)
		if err != nil {
			return err
		}

		if err := ub.Put(itob(int(id)), data); err != nil {
			return err
		}

		count = ub.Stats().KeyN
		return nil
	})

	return count, err
}

func GetWarns(chatID, userID int64) ([]*Warn, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	if err := ensureWarnsBuckets(db); err != nil {
		return nil, err
	}

	var warns []*Warn
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("warns"))
		if b == nil {
			return nil
		}
		cb := b.Bucket([]byte(strconv.FormatInt(chatID, 10)))
		if cb == nil {
			return nil
		}
		ub := cb.Bucket([]byte(strconv.FormatInt(userID, 10)))
		if ub == nil {
			return nil
		}

		c := ub.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var w Warn
			if err := json.Unmarshal(v, &w); err != nil {
				continue
			}
			warns = append(warns, &w)
		}
		return nil
	})
	return warns, err
}

func ResetWarns(chatID, userID int64) error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	if err := ensureWarnsBuckets(db); err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("warns"))
		if b == nil {
			return nil
		}
		cb := b.Bucket([]byte(strconv.FormatInt(chatID, 10)))
		if cb == nil {
			return nil
		}
		return cb.DeleteBucket([]byte(strconv.FormatInt(userID, 10)))
	})
}

func SetWarnSettings(chatID int64, settings *WarnSettings) error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	if err := ensureWarnsBuckets(db); err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("warns_settings"))
		data, err := json.Marshal(settings)
		if err != nil {
			return err
		}
		return b.Put([]byte(strconv.FormatInt(chatID, 10)), data)
	})
}

func GetWarnSettings(chatID int64) (*WarnSettings, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	if err := ensureWarnsBuckets(db); err != nil {
		return nil, err
	}

	settings := &WarnSettings{
		MaxWarns: 3,
		Action:   WarnActionBan,
	}

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("warns_settings"))
		if b == nil {
			return nil
		}
		data := b.Get([]byte(strconv.FormatInt(chatID, 10)))
		if data == nil {
			return nil
		}
		return json.Unmarshal(data, settings)
	})
	return settings, err
}

func itob(v int) []byte {
	b := make([]byte, 8)
	for i := uint(0); i < 8; i++ {
		b[7-i] = byte(v >> (i * 8))
	}
	return b
}
