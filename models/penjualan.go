package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Penjualan struct {
	ID              int64
	PengepulID      int64
	UserID          int64
	Tanggal         time.Time
	TotalPendapatan float64
	NamaPengepul    string
	TeleponPengepul string
	AlamatPengepul  string
	NamaUser        string
	TotalStokKeluar float64
	TotalBeratRiil  float64
	TotalPenyusutan float64
	TotalLabaRugi   float64
	Items           []DetailPenjualan
}

type DetailPenjualan struct {
	ID            int64
	PenjualanID   int64
	JenisSampahID int64
	NamaJenis     string
	Berat         float64 // berat stok gudang yang dikeluarkan
	BeratRiil     float64 // berat riil hasil penimbangan di pengepul
	HargaJual     float64
	HPP           float64
	Subtotal      float64 // BeratRiil * HargaJual
	Penyusutan    float64 // Berat - BeratRiil
	LabaRugi      float64 // (HargaJual - HPP) * BeratRiil
}

type ItemPenjualanInput struct {
	JenisSampahID int64
	BeratStok     float64
	BeratRiil     float64
	HargaJual     float64
	HPP           float64
}

// ListPenjualan mengembalikan daftar riwayat penjualan sampah ke pengepul.
func ListPenjualan(db *sql.DB, search string) ([]Penjualan, error) {
	q := `
SELECT p.id, p.pengepul_id, p.user_id, p.tanggal, p.total_pendapatan,
       pg.nama, u.username,
       COALESCE((SELECT SUM(dj.berat) FROM detail_penjualan dj WHERE dj.penjualan_id = p.id), 0) AS total_stok_keluar,
       COALESCE((SELECT SUM(dj.berat_riil) FROM detail_penjualan dj WHERE dj.penjualan_id = p.id), 0) AS total_berat_riil,
       COALESCE((SELECT SUM((dj.harga_jual - dj.hpp) * dj.berat_riil) FROM detail_penjualan dj WHERE dj.penjualan_id = p.id), 0) AS total_laba_rugi
FROM penjualan p
JOIN pengepul pg ON pg.id = p.pengepul_id
JOIN users u ON u.id = p.user_id`

	var args []any
	if search != "" {
		like := "%" + search + "%"
		q += " WHERE pg.nama LIKE ?"
		args = append(args, like)
	}
	q += " ORDER BY p.tanggal DESC, p.id DESC"

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Penjualan
	for rows.Next() {
		var p Penjualan
		if err := rows.Scan(&p.ID, &p.PengepulID, &p.UserID, &p.Tanggal, &p.TotalPendapatan,
			&p.NamaPengepul, &p.NamaUser, &p.TotalStokKeluar, &p.TotalBeratRiil, &p.TotalLabaRugi); err != nil {
			return nil, err
		}
		p.TotalPenyusutan = p.TotalStokKeluar - p.TotalBeratRiil
		list = append(list, p)
	}
	return list, rows.Err()
}

// GetPenjualanByID mengambil detail satu transaksi penjualan lengkap dengan rincian per jenis sampah.
func GetPenjualanByID(db *sql.DB, id int64) (*Penjualan, error) {
	var p Penjualan
	err := db.QueryRow(`
SELECT p.id, p.pengepul_id, p.user_id, p.tanggal, p.total_pendapatan,
       pg.nama, COALESCE(pg.no_telepon, ''), COALESCE(pg.alamat, ''), u.username
FROM penjualan p
JOIN pengepul pg ON pg.id = p.pengepul_id
JOIN users u ON u.id = p.user_id
WHERE p.id = ?`, id).Scan(
		&p.ID, &p.PengepulID, &p.UserID, &p.Tanggal, &p.TotalPendapatan,
		&p.NamaPengepul, &p.TeleponPengepul, &p.AlamatPengepul, &p.NamaUser,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT dj.id, dj.penjualan_id, dj.jenis_sampah_id, j.nama_jenis,
       dj.berat, dj.berat_riil, dj.harga_jual, dj.hpp
FROM detail_penjualan dj
JOIN jenis_sampah j ON j.id = dj.jenis_sampah_id
WHERE dj.penjualan_id = ?
ORDER BY dj.id ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var d DetailPenjualan
		if err := rows.Scan(&d.ID, &d.PenjualanID, &d.JenisSampahID, &d.NamaJenis,
			&d.Berat, &d.BeratRiil, &d.HargaJual, &d.HPP); err != nil {
			return nil, err
		}
		d.Subtotal = d.BeratRiil * d.HargaJual
		d.Penyusutan = d.Berat - d.BeratRiil
		d.LabaRugi = (d.HargaJual - d.HPP) * d.BeratRiil

		p.TotalStokKeluar += d.Berat
		p.TotalBeratRiil += d.BeratRiil
		p.TotalPenyusutan += d.Penyusutan
		p.TotalLabaRugi += d.LabaRugi

		p.Items = append(p.Items, d)
	}

	return &p, rows.Err()
}

// CreatePenjualan mengeksekusi transaksi penjualan sampah secara atomik sesuai Bab 5.5.
func CreatePenjualan(db *sql.DB, userID int64, pengepulID int64, items []ItemPenjualanInput) (int64, error) {
	if len(items) == 0 {
		return 0, errors.New("rincian penjualan sampah tidak boleh kosong")
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// 1. Validasi pengepul
	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM pengepul WHERE id = ?)", pengepulID).Scan(&exists)
	if err != nil || !exists {
		return 0, errors.New("pengepul tidak ditemukan")
	}

	// 2. Validasi stok dan hitung total pendapatan
	var totalPendapatan float64
	for _, it := range items {
		if it.BeratStok <= 0 {
			return 0, errors.New("berat stok yang dikeluarkan harus lebih besar dari 0 kg")
		}
		if it.BeratRiil <= 0 {
			return 0, errors.New("berat riil hasil penimbangan harus lebih besar dari 0 kg")
		}
		if it.HargaJual <= 0 {
			return 0, errors.New("harga jual harus lebih besar dari Rp 0")
		}

		// Kunci baris jenis_sampah
		var namaJenis string
		err := tx.QueryRow("SELECT nama_jenis FROM jenis_sampah WHERE id = ? FOR UPDATE", it.JenisSampahID).Scan(&namaJenis)
		if err != nil {
			return 0, fmt.Errorf("jenis sampah ID %d tidak ditemukan", it.JenisSampahID)
		}

		stokTersedia, err := GetStokJenisTx(tx, it.JenisSampahID)
		if err != nil {
			return 0, fmt.Errorf("gagal memeriksa stok %s: %w", namaJenis, err)
		}

		if it.BeratStok > stokTersedia {
			return 0, fmt.Errorf("berat stok keluar untuk %s (%.2f kg) melebihi stok yang tersedia di gudang (%.2f kg)",
				namaJenis, it.BeratStok, stokTersedia)
		}

		totalPendapatan += it.BeratRiil * it.HargaJual
	}

	// 3. Simpan header penjualan
	now := time.Now()
	res, err := tx.Exec("INSERT INTO penjualan (pengepul_id, user_id, tanggal, total_pendapatan) VALUES (?, ?, ?, ?)",
		pengepulID, userID, now, totalPendapatan)
	if err != nil {
		return 0, err
	}
	penjualanID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	// 4. Simpan rincian ke detail_penjualan
	stmt, err := tx.Prepare(`
INSERT INTO detail_penjualan (penjualan_id, jenis_sampah_id, berat, berat_riil, harga_jual, hpp)
VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	for _, it := range items {
		if _, err := stmt.Exec(penjualanID, it.JenisSampahID, it.BeratStok, it.BeratRiil, it.HargaJual, it.HPP); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return penjualanID, nil
}

