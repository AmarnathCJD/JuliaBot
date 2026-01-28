package db

import (
	"encoding/json"
	"strconv"
	"time"

	bolt "go.etcd.io/bbolt"
)

type Note struct {
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	MediaType string    `json:"media_type,omitempty"`
	FileID    string    `json:"file_id,omitempty"`
	AdminOnly bool      `json:"admin_only"`
	CreatedBy int64     `json:"created_by"`
	ExpiresAt time.Time `json:"expires_at"`
	Buttons   string    `json:"buttons,omitempty"`
}

func ensureNotesBuckets(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {

		_, err := tx.CreateBucketIfNotExists([]byte("notes"))
		return err
	})
}

func SaveNote(chatID int64, note *Note) error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	if err := ensureNotesBuckets(db); err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		notesBucket := tx.Bucket([]byte("notes"))
		chatBucket, err := notesBucket.CreateBucketIfNotExists([]byte(strconv.FormatInt(chatID, 10)))
		if err != nil {
			return err
		}

		noteData, err := json.Marshal(note)
		if err != nil {
			return err
		}

		return chatBucket.Put([]byte(note.Name), noteData)
	})
}

func GetNote(chatID int64, name string) (*Note, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	if err := ensureNotesBuckets(db); err != nil {
		return nil, err
	}

	var note *Note
	err = db.View(func(tx *bolt.Tx) error {
		notesBucket := tx.Bucket([]byte("notes"))
		if notesBucket == nil {
			return nil
		}
		chatBucket := notesBucket.Bucket([]byte(strconv.FormatInt(chatID, 10)))
		if chatBucket == nil {
			return nil
		}

		data := chatBucket.Get([]byte(name))
		if data == nil {
			return nil
		}

		note = &Note{}
		err := json.Unmarshal(data, note)
		if err != nil {
			return err
		}

		if !note.ExpiresAt.IsZero() && time.Now().After(note.ExpiresAt) {
			chatBucket.Delete([]byte(name))
			note = nil
			return nil
		}

		return nil
	})

	return note, err
}

func GetAllNotes(chatID int64) ([]*Note, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	if err := ensureNotesBuckets(db); err != nil {
		return nil, err
	}

	var notes []*Note
	err = db.View(func(tx *bolt.Tx) error {
		notesBucket := tx.Bucket([]byte("notes"))
		if notesBucket == nil {
			return nil
		}
		chatBucket := notesBucket.Bucket([]byte(strconv.FormatInt(chatID, 10)))
		if chatBucket == nil {
			return nil
		}

		c := chatBucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var note Note
			if err := json.Unmarshal(v, &note); err != nil {
				continue
			}
			notes = append(notes, &note)
		}
		return nil
	})

	return notes, err
}

func DeleteNote(chatID int64, name string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	if err := ensureNotesBuckets(db); err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		notesBucket := tx.Bucket([]byte("notes"))
		if notesBucket == nil {
			return nil
		}
		chatBucket := notesBucket.Bucket([]byte(strconv.FormatInt(chatID, 10)))
		if chatBucket == nil {
			return nil
		}

		return chatBucket.Delete([]byte(name))
	})
}

func DeleteAllNotes(chatID int64) error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	if err := ensureNotesBuckets(db); err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		notesBucket := tx.Bucket([]byte("notes"))
		if notesBucket == nil {
			return nil
		}
		return notesBucket.DeleteBucket([]byte(strconv.FormatInt(chatID, 10)))
	})
}

func GetNotesCount(chatID int64) (int, error) {
	db, err := GetDB()
	if err != nil {
		return 0, err
	}
	if err := ensureNotesBuckets(db); err != nil {
		return 0, err
	}

	count := 0
	err = db.View(func(tx *bolt.Tx) error {
		notesBucket := tx.Bucket([]byte("notes"))
		if notesBucket == nil {
			return nil
		}
		chatBucket := notesBucket.Bucket([]byte(strconv.FormatInt(chatID, 10)))
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

func CloseNotesDB() error {
	return nil
}
