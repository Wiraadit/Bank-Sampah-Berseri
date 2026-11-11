package models

import (
	"database/sql"
	"errors"
)

// Execer dipenuhi oleh *sql.DB maupun *sql.Tx,
// sehingga fungsi yang sama bisa dipakai di dalam atau di luar transaksi.
type Execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Role         string // "admin" atau "nasabah"
}

func GetUserByUsername(db *sql.DB, username string) (*User, error) {
	var u User
	err := db.QueryRow(
		"SELECT id, username, password, role FROM users WHERE username = ?", username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateUser menyimpan user baru. passwordHash harus sudah berupa hash bcrypt.
func CreateUser(exec Execer, username, passwordHash, role string) (int64, error) {
	res, err := exec.Exec(
		"INSERT INTO users (username, password, role) VALUES (?, ?, ?)",
		username, passwordHash, role,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
