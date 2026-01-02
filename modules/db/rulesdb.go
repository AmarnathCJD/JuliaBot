package db

import (
	"strconv"

	bolt "go.etcd.io/bbolt"
)

func ensureRulesBuckets(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("rules"))
		return err
	})
}

func SetRules(chatID int64, rules string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	if err := ensureRulesBuckets(db); err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("rules"))
		return bucket.Put([]byte(strconv.FormatInt(chatID, 10)), []byte(rules))
	})
}

func GetRules(chatID int64) (string, error) {
	db, err := GetDB()
	if err != nil {
		return "", err
	}
	if err := ensureRulesBuckets(db); err != nil {
		return "", err
	}

	var rules string
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("rules"))
		data := bucket.Get([]byte(strconv.FormatInt(chatID, 10)))
		if data != nil {
			rules = string(data)
		}
		return nil
	})

	return rules, err
}

func DeleteRules(chatID int64) error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	if err := ensureRulesBuckets(db); err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("rules"))
		return bucket.Delete([]byte(strconv.FormatInt(chatID, 10)))
	})
}

func HasRules(chatID int64) bool {
	rules, err := GetRules(chatID)
	return err == nil && rules != ""
}

func CloseRulesDB() error {
	return nil
}
