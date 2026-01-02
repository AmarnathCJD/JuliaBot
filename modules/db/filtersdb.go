package db

import (
	"encoding/json"
	"strconv"

	bolt "go.etcd.io/bbolt"
)

type Filter struct {
	Keyword   string `json:"keyword"`
	Content   string `json:"content"`
	MediaType string `json:"media_type,omitempty"`
	FileID    string `json:"file_id,omitempty"`
	AddedBy   int64  `json:"added_by"`
}

func ensureFiltersBuckets(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("filters"))
		return err
	})
}

func SaveFilter(chatID int64, filter *Filter) error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	if err := ensureFiltersBuckets(db); err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		filtersBucket := tx.Bucket([]byte("filters"))
		chatBucket, err := filtersBucket.CreateBucketIfNotExists([]byte(strconv.FormatInt(chatID, 10)))
		if err != nil {
			return err
		}

		data, err := json.Marshal(filter)
		if err != nil {
			return err
		}

		return chatBucket.Put([]byte(filter.Keyword), data)
	})
}

func GetFilter(chatID int64, keyword string) (*Filter, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	if err := ensureFiltersBuckets(db); err != nil {
		return nil, err
	}

	var filter *Filter
	err = db.View(func(tx *bolt.Tx) error {
		filtersBucket := tx.Bucket([]byte("filters"))
		if filtersBucket == nil {
			return nil
		}
		chatBucket := filtersBucket.Bucket([]byte(strconv.FormatInt(chatID, 10)))
		if chatBucket == nil {
			return nil
		}

		data := chatBucket.Get([]byte(keyword))
		if data == nil {
			return nil
		}

		filter = &Filter{}
		return json.Unmarshal(data, filter)
	})

	return filter, err
}

func GetAllFilters(chatID int64) ([]*Filter, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	if err := ensureFiltersBuckets(db); err != nil {
		return nil, err
	}

	var filters []*Filter
	err = db.View(func(tx *bolt.Tx) error {
		filtersBucket := tx.Bucket([]byte("filters"))
		if filtersBucket == nil {
			return nil
		}
		chatBucket := filtersBucket.Bucket([]byte(strconv.FormatInt(chatID, 10)))
		if chatBucket == nil {
			return nil
		}

		c := chatBucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var filter Filter
			if err := json.Unmarshal(v, &filter); err != nil {
				continue
			}
			filters = append(filters, &filter)
		}
		return nil
	})

	return filters, err
}

func DeleteFilter(chatID int64, keyword string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	if err := ensureFiltersBuckets(db); err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		filtersBucket := tx.Bucket([]byte("filters"))
		if filtersBucket == nil {
			return nil
		}
		chatBucket := filtersBucket.Bucket([]byte(strconv.FormatInt(chatID, 10)))
		if chatBucket == nil {
			return nil
		}

		return chatBucket.Delete([]byte(keyword))
	})
}

func DeleteAllFilters(chatID int64) error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	if err := ensureFiltersBuckets(db); err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		filtersBucket := tx.Bucket([]byte("filters"))
		if filtersBucket == nil {
			return nil
		}
		return filtersBucket.DeleteBucket([]byte(strconv.FormatInt(chatID, 10)))
	})
}

func GetFiltersCount(chatID int64) (int, error) {
	db, err := GetDB()
	if err != nil {
		return 0, err
	}
	if err := ensureFiltersBuckets(db); err != nil {
		return 0, err
	}

	count := 0
	err = db.View(func(tx *bolt.Tx) error {
		filtersBucket := tx.Bucket([]byte("filters"))
		if filtersBucket == nil {
			return nil
		}
		chatBucket := filtersBucket.Bucket([]byte(strconv.FormatInt(chatID, 10)))
		if chatBucket == nil {
			return nil
		}

		c := chatBucket.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			count++
		}
		return nil
	})

	return count, err
}

func CloseFiltersDB() error {
	return nil
}
