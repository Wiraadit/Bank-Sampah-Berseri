package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Mutasi struct {
	ID           int64
	NasabahID    int64
	SetoranID    sql.NullInt64
	IuranID      sql.NullInt64
	Tanggal      time.Time
	Jenis        string // setoran | bayar_tunggakan | penarikan | iuran
	Nominal      float64
	SaldoSebelum float64
	SaldoSesudah float64
}

type MutasiGlobal struct {
	ID           int64
	NasabahID    int64
	KodeNasabah  string
	NamaNasabah  string
	Tanggal      time.Time
	Jenis        string
	Nominal      float64
	SaldoSebelum float64
	SaldoSesudah float64
}

// SaldoMinimumMengendap batas sisa saldo minimal setelah penarikan (Bab 5.6).
const SaldoMinimumMengendap = 10000.0

// ListMutasiTerakhir mengambil mutasi terbaru milik satu nasabah.
func ListMutasiTerakhir(db *sql.DB, nasabahID int64, limit int) ([]Mutasi, error) {
	rows, err := db.Query(`
SELECT id, nasabah_id, setoran_id, iuran_id, tanggal, jenis, nominal, saldo_sebelum, saldo_sesudah
FROM transaksi_saldo
WHERE nasabah_id = ?
ORDER BY tanggal DESC, id DESC
LIMIT ?`, nasabahID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Mutasi
	for rows.Next() {
		var m Mutasi
		if err := rows.Scan(&m.ID, &m.NasabahID, &m.SetoranID, &m.IuranID,
			&m.Tanggal, &m.Jenis, &m.Nominal, &m.SaldoSebelum, &m.SaldoSesudah); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// ListSemuaMutasiNasabah mengambil seluruh buku tabungan milik satu nasabah (urut dari transaksi pertama atau terbaru).
func ListSemuaMutasiNasabah(db *sql.DB, nasabahID int64) ([]Mutasi, error) {
	rows, err := db.Query(`
SELECT id, nasabah_id, setoran_id, iuran_id, tanggal, jenis, nominal, saldo_sebelum, saldo_sesudah
FROM transaksi_saldo
WHERE nasabah_id = ?
ORDER BY tanggal DESC, id DESC`, nasabahID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Mutasi
	for rows.Next() {
		var m Mutasi
		if err := rows.Scan(&m.ID, &m.NasabahID, &m.SetoranID, &m.IuranID,
			&m.Tanggal, &m.Jenis, &m.Nominal, &m.SaldoSebelum, &m.SaldoSesudah); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// ListMutasiGlobal mengambil daftar mutasi saldo seluruh nasabah untuk halaman admin keuangan.
func ListMutasiGlobal(db *sql.DB, limit int) ([]MutasiGlobal, error) {
	rows, err := db.Query(`
SELECT ts.id, ts.nasabah_id, n.kode_nasabah, n.nama,
       ts.tanggal, ts.jenis, ts.nominal, ts.saldo_sebelum, ts.saldo_sesudah
FROM transaksi_saldo ts
JOIN nasabah n ON n.id = ts.nasabah_id
ORDER BY ts.tanggal DESC, ts.id DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []MutasiGlobal
	for rows.Next() {
		var m MutasiGlobal
		if err := rows.Scan(&m.ID, &m.NasabahID, &m.KodeNasabah, &m.NamaNasabah,
			&m.Tanggal, &m.Jenis, &m.Nominal, &m.SaldoSebelum, &m.SaldoSesudah); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// TarikSaldo mengeksekusi penarikan uang tabungan nasabah dengan validasi batas saldo minimal Rp 10.000.
func TarikSaldo(db *sql.DB, nasabahID int64, nominal float64) (saldoAkhir float64, err error) {
	if nominal <= 0 {
		return 0, errors.New("nominal penarikan harus lebih besar dari Rp 0")
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var n Nasabah
	err = tx.QueryRow(`
SELECT id, saldo, tunggakan, status
FROM nasabah
WHERE id = ? FOR UPDATE`, nasabahID).Scan(&n.ID, &n.Saldo, &n.Tunggakan, &n.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errors.New("nasabah tidak ditemukan")
	}
	if err != nil {
		return 0, err
	}
	if n.Status != "aktif" {
		return 0, errors.New("nasabah berstatus nonaktif dan tidak dapat melakukan penarikan")
	}

	// Validasi aturan Bab 5.6: Saldo setelah penarikan minimal Rp 10.000
	sisaSaldo := n.Saldo - nominal
	if sisaSaldo < SaldoMinimumMengendap {
		maksimalTarik := n.Saldo - SaldoMinimumMengendap
		if maksimalTarik < 0 {
			maksimalTarik = 0
		}
		return 0, fmt.Errorf("penarikan ditolak: saldo setelah penarikan minimal Rp 10.000 (saldo saat ini Rp %.0f, maksimal ditarik Rp %.0f)",
			n.Saldo, maksimalTarik)
	}

	// 1. Update saldo nasabah
	_, err = tx.Exec("UPDATE nasabah SET saldo = ? WHERE id = ?", sisaSaldo, nasabahID)
	if err != nil {
		return 0, err
	}

	// 2. Catat riwayat di transaksi_saldo
	now := time.Now()
	_, err = tx.Exec(`
INSERT INTO transaksi_saldo (nasabah_id, nominal, saldo_sebelum, saldo_sesudah, jenis, tanggal)
VALUES (?, ?, ?, ?, 'penarikan', ?)`,
		nasabahID, nominal, n.Saldo, sisaSaldo, now)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return sisaSaldo, nil
}
