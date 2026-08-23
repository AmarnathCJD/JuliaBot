package db

import (
	"database/sql"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	sqlOnce sync.Once
	sqlDB   *sql.DB
	sqlErr  error
)

func GetSQLDB() (*sql.DB, error) {
	sqlOnce.Do(func() {
		sqlDB, sqlErr = sql.Open("sqlite", "file:julia.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
		if sqlErr != nil {
			return
		}
		sqlErr = sqlDB.Ping()
		if sqlErr != nil {
			return
		}
		_, sqlErr = sqlDB.Exec(`CREATE TABLE IF NOT EXISTS tamilmv_alerts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chat_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			interval_seconds INTEGER NOT NULL,
			last_seen TEXT NOT NULL DEFAULT '',
			last_checked INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL
		)`)
	})
	return sqlDB, sqlErr
}

func CloseSQLDB() error {
	if sqlDB != nil {
		return sqlDB.Close()
	}
	return nil
}

type TamilMVAlert struct {
	ID, ChatID, UserID    int64
	Title                 string
	Interval, LastChecked int64
	LastSeen              string
}

func SaveTamilMVAlert(a *TamilMVAlert) error {
	d, e := GetSQLDB()
	if e != nil {
		return e
	}
	if a.ID == 0 {
		r, e := d.Exec(`INSERT INTO tamilmv_alerts(chat_id,user_id,title,interval_seconds,created_at) VALUES(?,?,?,?,?)`, a.ChatID, a.UserID, a.Title, a.Interval, time.Now().Unix())
		if e != nil {
			return e
		}
		a.ID, _ = r.LastInsertId()
		return nil
	}
	_, e = d.Exec(`UPDATE tamilmv_alerts SET title=?,interval_seconds=? WHERE id=? AND user_id=?`, a.Title, a.Interval, a.ID, a.UserID)
	return e
}
func ListTamilMVAlerts(uid int64) ([]TamilMVAlert, error) {
	d, e := GetSQLDB()
	if e != nil {
		return nil, e
	}
	rows, e := d.Query(`SELECT id,chat_id,user_id,title,interval_seconds,last_seen,last_checked FROM tamilmv_alerts WHERE user_id=? ORDER BY id`, uid)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []TamilMVAlert
	for rows.Next() {
		var a TamilMVAlert
		if e = rows.Scan(&a.ID, &a.ChatID, &a.UserID, &a.Title, &a.Interval, &a.LastSeen, &a.LastChecked); e == nil {
			out = append(out, a)
		}
	}
	return out, rows.Err()
}
func DeleteTamilMVAlert(id, uid int64) error {
	d, e := GetSQLDB()
	if e != nil {
		return e
	}
	_, e = d.Exec(`DELETE FROM tamilmv_alerts WHERE id=? AND user_id=?`, id, uid)
	return e
}
func DueTamilMVAlerts(now int64) ([]TamilMVAlert, error) {
	d, e := GetSQLDB()
	if e != nil {
		return nil, e
	}
	rows, e := d.Query(`SELECT id,chat_id,user_id,title,interval_seconds,last_seen,last_checked FROM tamilmv_alerts WHERE last_checked=0 OR last_checked+interval_seconds<=?`, now)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []TamilMVAlert
	for rows.Next() {
		var a TamilMVAlert
		if e = rows.Scan(&a.ID, &a.ChatID, &a.UserID, &a.Title, &a.Interval, &a.LastSeen, &a.LastChecked); e == nil {
			out = append(out, a)
		}
	}
	return out, rows.Err()
}
func TouchTamilMVAlert(id int64, seen string, checked int64) error {
	d, e := GetSQLDB()
	if e != nil {
		return e
	}
	_, e = d.Exec(`UPDATE tamilmv_alerts SET last_seen=?,last_checked=? WHERE id=?`, seen, checked, id)
	return e
}
