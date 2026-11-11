package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Pemilahan struct {
	ID         int64
	UserID     int64
	Tanggal    time.Time
	BeratAwal  float64
	NamaUser   string
	TotalHasil float64
	Items      []DetailPemilahan
}

type DetailPemilahan struct {
	ID            int64
	PemilahanID   int64
	JenisSampahID int64
	NamaJenis     string
	BeratHasil    float64
}

type ItemHasilPemilahanInput struct {
	JenisSampahID int64
	BeratHasil    float64
}

// ListPemilahan mengembalikan riwayat pemilahan terbaru.
func ListPemilahan(db *sql.DB) ([]Pemilahan, error) {
	const q = `
SELECT p.id, p.user_id, p.tanggal, p.berat_awal, u.username,
       COALESCE((SELECT SUM(dp.berat_hasil) FROM detail_pemilahan dp WHERE dp.pemilahan_id = p.id), 0) AS total_hasil
FROM pemilahan p
JOIN users u ON u.id = p.user_id
ORDER BY p.tanggal DESC, p.id DESC`

	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Pemilahan
	for rows.Next() {
		var p Pemilahan
		if err := rows.Scan(&p.ID, &p.UserID, &p.Tanggal, &p.BeratAwal, &p.NamaUser, &p.TotalHasil); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// GetPemilahanByID mengambil satu transaksi pemilahan lengkap dengan rincian hasil pilahnya.
func GetPemilahanByID(db *sql.DB, id int64) (*Pemilahan, error) {
	var p Pemilahan
	err := db.QueryRow(`
SELECT p.id, p.user_id, p.tanggal, p.berat_awal, u.username
FROM pemilahan p
JOIN users u ON u.id = p.user_id
WHERE p.id = ?`, id).Scan(&p.ID, &p.UserID, &p.Tanggal, &p.BeratAwal, &p.NamaUser)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT dp.id, dp.pemilahan_id, dp.jenis_sampah_id, j.nama_jenis, dp.berat_hasil
FROM detail_pemilahan dp
JOIN jenis_sampah j ON j.id = dp.jenis_sampah_id
WHERE dp.pemilahan_id = ?
ORDER BY dp.id ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var d DetailPemilahan
		if err := rows.Scan(&d.ID, &d.PemilahanID, &d.JenisSampahID, &d.NamaJenis, &d.BeratHasil); err != nil {
			return nil, err
		}
		p.Items = append(p.Items, d)
		p.TotalHasil += d.BeratHasil
	}
	return &p, rows.Err()
}

// CreatePemilahan menyimpan transaksi pemilahan dan rincian hasil pilah secara atomik sesuai Bab 5.3.
func CreatePemilahan(db *sql.DB, userID int64, beratAwal float64, items []ItemHasilPemilahanInput) (int64, error) {
	if beratAwal <= 0 {
		return 0, errors.New("berat awal pemilahan harus lebih besar dari 0 kg")
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// 1. Ambil ID Plastik Campur dan kunci/periksa stoknya di dalam tx
	var plastikCampurID int64
	err = tx.QueryRow("SELECT id FROM jenis_sampah WHERE nama_jenis = ? FOR UPDATE", NamaPlastikCampur).Scan(&plastikCampurID)
	if err != nil {
		return 0, fmt.Errorf("master jenis sampah '%s' tidak ditemukan", NamaPlastikCampur)
	}

	stokPlastikCampur, err := GetStokJenisTx(tx, plastikCampurID)
	if err != nil {
		return 0, fmt.Errorf("gagal memeriksa stok plastik campur: %w", err)
	}

	if beratAwal > stokPlastikCampur {
		return 0, fmt.Errorf("berat awal (%.2f kg) melebihi stok Plastik Campur yang tersedia (%.2f kg)", beratAwal, stokPlastikCampur)
	}

	// 2. Validasi rincian hasil pemilahan
	var totalHasil float64
	var validItems []ItemHasilPemilahanInput
	for _, it := range items {
		if it.BeratHasil < 0 {
			return 0, errors.New("berat hasil tidak boleh bernilai negatif")
		}
		if it.BeratHasil > 0 {
			if it.JenisSampahID == plastikCampurID {
				return 0, errors.New("plastik campur tidak boleh menjadi hasil pemilahan dirinya sendiri")
			}
			totalHasil += it.BeratHasil
			validItems = append(validItems, it)
		}
	}

	if len(validItems) == 0 {
		return 0, errors.New("minimal satu jenis sampah hasil pemilahan harus memiliki berat lebih dari 0 kg")
	}

	// Sesuai aturan 5.3: Total Hasil Pemilahan <= Berat Awal
	if totalHasil > beratAwal+0.0001 {
		return 0, fmt.Errorf("total berat hasil pemilahan (%.2f kg) melebihi berat awal (%.2f kg)", totalHasil, beratAwal)
	}

	// 3. Simpan header pemilahan
	now := time.Now()
	res, err := tx.Exec("INSERT INTO pemilahan (user_id, tanggal, berat_awal) VALUES (?, ?, ?)", userID, now, beratAwal)
	if err != nil {
		return 0, err
	}
	pemilahanID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	// 4. Simpan ke detail_pemilahan
	stmt, err := tx.Prepare("INSERT INTO detail_pemilahan (pemilahan_id, jenis_sampah_id, berat_hasil) VALUES (?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	for _, it := range validItems {
		if _, err := stmt.Exec(pemilahanID, it.JenisSampahID, it.BeratHasil); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return pemilahanID, nil
}

