package db

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	bolt "go.etcd.io/bbolt"
)

var (
	stickerDB     *bolt.DB
	stickerDBOnce sync.Once
	dbPath        = "stickers.db"
)

type PackInfo struct {
	ShortName    string `json:"short_name"`
	Title        string `json:"title"`
	Type         string `json:"type"`
	StickerCount int    `json:"sticker_count"`
	PackNumber   int    `json:"pack_number"`
}

func InitStickerDB() error {
	var err error
	stickerDBOnce.Do(func() {
		stickerDB, err = bolt.Open(dbPath, 0600, nil)
		if err != nil {
			return
		}

		err = stickerDB.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists([]byte("users"))
			if err != nil {
				return err
			}
			_, err = tx.CreateBucketIfNotExists([]byte("packs"))
			return err
		})
	})
	return err
}

func GetUserPacks(userID int64) (map[string][]*PackInfo, error) {
	if stickerDB == nil {
		if err := InitStickerDB(); err != nil {
			return nil, err
		}
	}

	packs := make(map[string][]*PackInfo)
	packs["normal"] = []*PackInfo{}
	packs["webm"] = []*PackInfo{}
	packs["tgs"] = []*PackInfo{}

	err := stickerDB.View(func(tx *bolt.Tx) error {
		userBucket := tx.Bucket([]byte("users")).Bucket([]byte(strconv.FormatInt(userID, 10)))
		if userBucket == nil {
			return nil
		}

		for _, packType := range []string{"normal", "webm", "tgs"} {
			typeBucket := userBucket.Bucket([]byte(packType))
			if typeBucket == nil {
				continue
			}

			c := typeBucket.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				var pack PackInfo
				if err := json.Unmarshal(v, &pack); err != nil {
					continue
				}
				packs[packType] = append(packs[packType], &pack)
			}
		}
		return nil
	})

	return packs, err
}

func GetActivePack(userID int64, packType string) (*PackInfo, error) {
	if stickerDB == nil {
		if err := InitStickerDB(); err != nil {
			return nil, err
		}
	}

	var pack *PackInfo
	err := stickerDB.View(func(tx *bolt.Tx) error {
		userBucket := tx.Bucket([]byte("users")).Bucket([]byte(strconv.FormatInt(userID, 10)))
		if userBucket == nil {
			return nil
		}

		typeBucket := userBucket.Bucket([]byte(packType))
		if typeBucket == nil {
			return nil
		}

		c := typeBucket.Cursor()
		var lastKey []byte
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			lastKey = k
		}

		if lastKey != nil {
			v := typeBucket.Get(lastKey)
			if v != nil {
				pack = &PackInfo{}
				return json.Unmarshal(v, pack)
			}
		}
		return nil
	})

	return pack, err
}

func SavePack(userID int64, pack *PackInfo) error {
	if stickerDB == nil {
		if err := InitStickerDB(); err != nil {
			return err
		}
	}

	return stickerDB.Update(func(tx *bolt.Tx) error {
		usersBucket := tx.Bucket([]byte("users"))
		userBucket, err := usersBucket.CreateBucketIfNotExists([]byte(strconv.FormatInt(userID, 10)))
		if err != nil {
			return err
		}

		typeBucket, err := userBucket.CreateBucketIfNotExists([]byte(pack.Type))
		if err != nil {
			return err
		}

		packData, err := json.Marshal(pack)
		if err != nil {
			return err
		}

		key := []byte(fmt.Sprintf("%d", pack.PackNumber))
		return typeBucket.Put(key, packData)
	})
}

func IncrementPackCount(userID int64, pack *PackInfo) error {
	pack.StickerCount++
	return SavePack(userID, pack)
}

func DecrementPackCount(userID int64, pack *PackInfo) error {
	if pack.StickerCount > 0 {
		pack.StickerCount--
	}
	return SavePack(userID, pack)
}

func GetPackByShortName(userID int64, shortName string) (*PackInfo, error) {
	if stickerDB == nil {
		if err := InitStickerDB(); err != nil {
			return nil, err
		}
	}

	var pack *PackInfo
	err := stickerDB.View(func(tx *bolt.Tx) error {
		userBucket := tx.Bucket([]byte("users")).Bucket([]byte(strconv.FormatInt(userID, 10)))
		if userBucket == nil {
			return fmt.Errorf("no packs found")
		}

		for _, packType := range []string{"normal", "webm", "tgs"} {
			typeBucket := userBucket.Bucket([]byte(packType))
			if typeBucket == nil {
				continue
			}

			c := typeBucket.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				var p PackInfo
				if err := json.Unmarshal(v, &p); err != nil {
					continue
				}
				if p.ShortName == shortName {
					pack = &p
					return nil
				}
			}
		}
		return fmt.Errorf("pack not found")
	})

	return pack, err
}

func CloseStickerDB() error {
	if stickerDB != nil {
		return stickerDB.Close()
	}
	return nil
}
