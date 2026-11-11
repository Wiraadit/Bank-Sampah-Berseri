package models

import (
	"database/sql"
	"errors"
)

type Pengepul struct {
	ID        int64
	Nama      string
	NoTelepon string
	Alamat    string
}

// PengepulRingkasan dipakai di halaman detail pengepul.
type PengepulRingkasan struct {
	JumlahPenjualan int
	TotalPendapatan float64
}

func ListPengepul(db *sql.DB) ([]Pengepul, error) {
	rows, err := db.Query(
		"SELECT id, nama, COALESCE(no_telepon,''), COALESCE(alamat,'') FROM pengepul ORDER BY nama")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Pengepul
	for rows.Next() {
		var p Pengepul
		if err := rows.Scan(&p.ID, &p.Nama, &p.NoTelepon, &p.Alamat); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func GetPengepulByID(db *sql.DB, id int64) (*Pengepul, error) {
	var p Pengepul
	err := db.QueryRow(
		"SELECT id, nama, COALESCE(no_telepon,''), COALESCE(alamat,'') FROM pengepul WHERE id = ?", id,
	).Scan(&p.ID, &p.Nama, &p.NoTelepon, &p.Alamat)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func CreatePengepul(db *sql.DB, nama, telepon, alamat string) (int64, error) {
	res, err := db.Exec(
		"INSERT INTO pengepul (nama, no_telepon, alamat) VALUES (?, ?, ?)",
		nama, telepon, alamat,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func UpdatePengepul(db *sql.DB, id int64, nama, telepon, alamat string) error {
	_, err := db.Exec(
		"UPDATE pengepul SET nama = ?, no_telepon = ?, alamat = ? WHERE id = ?",
		nama, telepon, alamat, id,
	)
	return err
}

func GetPengepulRingkasan(db *sql.DB, id int64) (*PengepulRingkasan, error) {
	var r PengepulRingkasan
	err := db.QueryRow(
		"SELECT COUNT(*), COALESCE(SUM(total_pendapatan),0) FROM penjualan WHERE pengepul_id = ?", id,
	).Scan(&r.JumlahPenjualan, &r.TotalPendapatan)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
