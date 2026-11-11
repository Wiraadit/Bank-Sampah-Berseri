package models

import (
	"database/sql"
	"errors"
)

type JenisSampah struct {
	ID        int64
	NamaJenis string
	HargaBeli float64
	HargaJual float64
}

func ListJenisSampah(db *sql.DB) ([]JenisSampah, error) {
	rows, err := db.Query("SELECT id, nama_jenis, harga_beli, harga_jual FROM jenis_sampah ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []JenisSampah
	for rows.Next() {
		var j JenisSampah
		if err := rows.Scan(&j.ID, &j.NamaJenis, &j.HargaBeli, &j.HargaJual); err != nil {
			return nil, err
		}
		list = append(list, j)
	}
	return list, rows.Err()
}

func GetJenisSampahByID(db *sql.DB, id int64) (*JenisSampah, error) {
	var j JenisSampah
	err := db.QueryRow(
		"SELECT id, nama_jenis, harga_beli, harga_jual FROM jenis_sampah WHERE id = ?", id,
	).Scan(&j.ID, &j.NamaJenis, &j.HargaBeli, &j.HargaJual)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// CreateJenisSampah mengembalikan ErrDuplicate jika nama sudah ada.
func CreateJenisSampah(db *sql.DB, nama string, hargaBeli, hargaJual float64) (int64, error) {
	res, err := db.Exec(
		"INSERT INTO jenis_sampah (nama_jenis, harga_beli, harga_jual) VALUES (?, ?, ?)",
		nama, hargaBeli, hargaJual,
	)
	if err != nil {
		if isDuplicate(err) { // didefinisikan di models/nasabah.go
			return 0, ErrDuplicate
		}
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateJenisSampah mengembalikan ErrDuplicate jika nama bentrok dengan jenis lain.
func UpdateJenisSampah(db *sql.DB, id int64, nama string, hargaBeli, hargaJual float64) error {
	_, err := db.Exec(
		"UPDATE jenis_sampah SET nama_jenis = ?, harga_beli = ?, harga_jual = ? WHERE id = ?",
		nama, hargaBeli, hargaJual, id,
	)
	if err != nil && isDuplicate(err) {
		return ErrDuplicate
	}
	return err
}
