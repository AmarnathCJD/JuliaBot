package db

import (
	"encoding/json"
	"strconv"

	bolt "go.etcd.io/bbolt"
)

type WelcomeMessage struct {
	Content        string `json:"content"`
	MediaType      string `json:"media_type,omitempty"`
	FileID         string `json:"file_id,omitempty"`
	DeletePrevious bool   `json:"delete_previous"`
	AutoDeleteSec  int    `json:"auto_delete_sec"`
}

type CaptchaSettings struct {
	Enabled   bool   `json:"enabled"`
	Mode      string `json:"mode"` // "button", "math"
	TimeLimit int    `json:"time_limit"`
}

func ensureWelcomeBuckets(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("welcome"))
		return err
	})
}

func SetWelcome(chatID int64, msg *WelcomeMessage) error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	if err := ensureWelcomeBuckets(db); err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("welcome"))
		cb, err := b.CreateBucketIfNotExists([]byte(strconv.FormatInt(chatID, 10)))
		if err != nil {
			return err
		}
		data, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		return cb.Put([]byte("msg"), data)
	})
}

func GetWelcome(chatID int64) (*WelcomeMessage, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	if err := ensureWelcomeBuckets(db); err != nil {
		return nil, err
	}

	var msg *WelcomeMessage
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("welcome"))
		cb := b.Bucket([]byte(strconv.FormatInt(chatID, 10)))
		if cb == nil {
			return nil
		}
		data := cb.Get([]byte("msg"))
		if data == nil {
			return nil
		}
		msg = &WelcomeMessage{}
		return json.Unmarshal(data, msg)
	})
	return msg, err
}

func SetGoodbye(chatID int64, msg *WelcomeMessage) error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	if err := ensureWelcomeBuckets(db); err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("welcome"))
		cb, err := b.CreateBucketIfNotExists([]byte(strconv.FormatInt(chatID, 10)))
		if err != nil {
			return err
		}
		data, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		return cb.Put([]byte("bye"), data)
	})
}

func GetGoodbye(chatID int64) (*WelcomeMessage, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	if err := ensureWelcomeBuckets(db); err != nil {
		return nil, err
	}

	var msg *WelcomeMessage
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("welcome"))
		cb := b.Bucket([]byte(strconv.FormatInt(chatID, 10)))
		if cb == nil {
			return nil
		}
		data := cb.Get([]byte("bye"))
		if data == nil {
			return nil
		}
		msg = &WelcomeMessage{}
		return json.Unmarshal(data, msg)
	})
	return msg, err
}

func SetCaptchaSettings(chatID int64, settings *CaptchaSettings) error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	if err := ensureWelcomeBuckets(db); err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("welcome"))
		cb, err := b.CreateBucketIfNotExists([]byte(strconv.FormatInt(chatID, 10)))
		if err != nil {
			return err
		}
		data, err := json.Marshal(settings)
		if err != nil {
			return err
		}
		return cb.Put([]byte("captcha"), data)
	})
}

func GetCaptchaSettings(chatID int64) (*CaptchaSettings, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	if err := ensureWelcomeBuckets(db); err != nil {
		return nil, err
	}

	settings := &CaptchaSettings{}
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("welcome"))
		cb := b.Bucket([]byte(strconv.FormatInt(chatID, 10)))
		if cb == nil {
			return nil
		}
		data := cb.Get([]byte("captcha"))
		if data == nil {
			return nil
		}
		return json.Unmarshal(data, settings)
	})
	return settings, err
}

func SetLastWelcomeID(chatID int64, msgID int) error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	if err := ensureWelcomeBuckets(db); err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("welcome"))
		cb, err := b.CreateBucketIfNotExists([]byte(strconv.FormatInt(chatID, 10)))
		if err != nil {
			return err
		}
		return cb.Put([]byte("last_msg_id"), []byte(strconv.Itoa(msgID)))
	})
}

func GetLastWelcomeID(chatID int64) (int, error) {
	db, err := GetDB()
	if err != nil {
		return 0, err
	}
	if err := ensureWelcomeBuckets(db); err != nil {
		return 0, err
	}

	var msgID int
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("welcome"))
		cb := b.Bucket([]byte(strconv.FormatInt(chatID, 10)))
		if cb == nil {
			return nil
		}
		data := cb.Get([]byte("last_msg_id"))
		if data == nil {
			return nil
		}
		msgID, _ = strconv.Atoi(string(data))
		return nil
	})
	return msgID, err
}
