package models

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
)

type Nasabah struct {
	ID          int64
	UserID      int64
	KodeNasabah string
	Nama        string
	Alamat      string
	NoTelepon   string
	Saldo       float64
	Tunggakan   float64
	Status      string // "aktif" atau "nonaktif"
	Username    string
}

const nasabahSelect = `
SELECT n.id, n.user_id, n.kode_nasabah, n.nama,
       COALESCE(n.alamat,''), COALESCE(n.no_telepon,''),
       n.saldo, n.tunggakan, n.status, u.username
FROM nasabah n JOIN users u ON u.id = n.user_id`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanNasabah(s rowScanner) (*Nasabah, error) {
	var n Nasabah
	err := s.Scan(&n.ID, &n.UserID, &n.KodeNasabah, &n.Nama, &n.Alamat,
		&n.NoTelepon, &n.Saldo, &n.Tunggakan, &n.Status, &n.Username)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func getNasabah(db *sql.DB, where string, arg any) (*Nasabah, error) {
	n, err := scanNasabah(db.QueryRow(nasabahSelect+" WHERE "+where, arg))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return n, err
}

func GetNasabahByID(db *sql.DB, id int64) (*Nasabah, error) {
	return getNasabah(db, "n.id = ?", id)
}

func GetNasabahByUserID(db *sql.DB, userID int64) (*Nasabah, error) {
	return getNasabah(db, "n.user_id = ?", userID)
}

// ListNasabah mengembalikan semua nasabah; jika search diisi, cari di nama/kode/telepon.
func ListNasabah(db *sql.DB, search string) ([]Nasabah, error) {
	q := nasabahSelect
	var args []any
	if search != "" {
		like := "%" + search + "%"
		q += " WHERE n.nama LIKE ? OR n.kode_nasabah LIKE ? OR n.no_telepon LIKE ?"
		args = append(args, like, like, like)
	}
	q += " ORDER BY n.kode_nasabah"

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Nasabah
	for rows.Next() {
		n, err := scanNasabah(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *n)
	}
	return list, rows.Err()
}

func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// CreateNasabahWithUser membuat akun (users) dan data nasabah dalam SATU transaksi.
// Kode nasabah dibuat otomatis: NSB-0001, NSB-0002, ...
func CreateNasabahWithUser(db *sql.DB, username, passwordHash, nama, alamat, telepon string) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	userID, err := CreateUser(tx, username, passwordHash, "nasabah")
	if err != nil {
		if isDuplicate(err) {
			return 0, ErrDuplicate
		}
		return 0, err
	}

	var last int
	err = tx.QueryRow(`SELECT COALESCE(MAX(CAST(SUBSTRING(kode_nasabah, 5) AS UNSIGNED)), 0)
	                   FROM nasabah WHERE kode_nasabah LIKE 'NSB-%'`).Scan(&last)
	if err != nil {
		return 0, err
	}
	kode := fmt.Sprintf("NSB-%04d", last+1)

	res, err := tx.Exec(
		`INSERT INTO nasabah (user_id, kode_nasabah, nama, alamat, no_telepon, saldo, tunggakan, status)
		 VALUES (?, ?, ?, ?, ?, 0, 0, 'aktif')`,
		userID, kode, nama, alamat, telepon,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func UpdateNasabah(db *sql.DB, id int64, nama, alamat, telepon string) error {
	_, err := db.Exec(
		"UPDATE nasabah SET nama = ?, alamat = ?, no_telepon = ? WHERE id = ?",
		nama, alamat, telepon, id,
	)
	return err
}

func SetNasabahStatus(db *sql.DB, id int64, status string) error {
	_, err := db.Exec("UPDATE nasabah SET status = ? WHERE id = ?", status, id)
	return err
}
