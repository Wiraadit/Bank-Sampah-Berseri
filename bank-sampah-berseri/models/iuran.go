package models

import (
	"database/sql"
	"errors"
	"time"
)

const TarifIuranBulanan = 5000.0

type IuranBulanan struct {
	ID          int64
	NasabahID   int64
	KodeNasabah string
	NamaNasabah string
	Bulan       int
	Tahun       int
	Nominal     float64
	Dibayar     float64
	Tunggakan   float64
	Status      string // "belum_bayar" | "sebagian" | "lunas"
}

type RingkasanIuranData struct {
	TotalTagihan   float64
	TotalTerbayar  float64
	TotalTunggakan float64
	JumlahLunas    int
	JumlahBelum    int
}

// ListIuran mengambil daftar tagihan iuran bulanan untuk periode tertentu.
func ListIuran(db *sql.DB, bulan, tahun int, statusFilter string) ([]IuranBulanan, error) {
	q := `
SELECT ib.id, ib.nasabah_id, n.kode_nasabah, n.nama,
       ib.bulan, ib.tahun, ib.nominal, ib.dibayar, ib.tunggakan, ib.status
FROM iuran_bulanan ib
JOIN nasabah n ON n.id = ib.nasabah_id
WHERE ib.bulan = ? AND ib.tahun = ?`

	var args []any
	args = append(args, bulan, tahun)

	if statusFilter != "" {
		q += " AND ib.status = ?"
		args = append(args, statusFilter)
	}
	q += " ORDER BY n.kode_nasabah ASC"

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []IuranBulanan
	for rows.Next() {
		var ib IuranBulanan
		if err := rows.Scan(&ib.ID, &ib.NasabahID, &ib.KodeNasabah, &ib.NamaNasabah,
			&ib.Bulan, &ib.Tahun, &ib.Nominal, &ib.Dibayar, &ib.Tunggakan, &ib.Status); err != nil {
			return nil, err
		}
		list = append(list, ib)
	}
	return list, rows.Err()
}

// GetRingkasanIuranPeriode menghitung total tagihan, terbayar, dan tunggakan untuk satu periode.
func GetRingkasanIuranPeriode(db *sql.DB, bulan, tahun int) (*RingkasanIuranData, error) {
	var r RingkasanIuranData
	err := db.QueryRow(`
SELECT
  COALESCE(SUM(nominal), 0),
  COALESCE(SUM(dibayar), 0),
  COALESCE(SUM(tunggakan), 0),
  COALESCE(SUM(CASE WHEN status = 'lunas' THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN status != 'lunas' THEN 1 ELSE 0 END), 0)
FROM iuran_bulanan
WHERE bulan = ? AND tahun = ?`, bulan, tahun).Scan(
		&r.TotalTagihan, &r.TotalTerbayar, &r.TotalTunggakan, &r.JumlahLunas, &r.JumlahBelum,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// prosesIuranNasabahTx memproses tagihan 1 nasabah di dalam transaksi aktif.
func prosesIuranNasabahTx(tx *sql.Tx, nasabahID int64, bulan, tahun int, now time.Time) (bool, error) {
	// Cek apakah tagihan periode ini sudah pernah dibuat
	var exists bool
	err := tx.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM iuran_bulanan WHERE nasabah_id = ? AND bulan = ? AND tahun = ?)",
		nasabahID, bulan, tahun,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil // sudah ada, jangan buat tagihan ganda
	}

	// Kunci baris nasabah
	var saldo, tunggakan float64
	var statusNasabah string
	err = tx.QueryRow(
		"SELECT saldo, tunggakan, status FROM nasabah WHERE id = ? FOR UPDATE",
		nasabahID,
	).Scan(&saldo, &tunggakan, &statusNasabah)
	if err != nil {
		return false, err
	}
	if statusNasabah != "aktif" {
		return false, nil
	}

	nominal := TarifIuranBulanan
	var dibayar, sisaTunggakan float64
	var statusIuran string

	if saldo >= nominal {
		// Kasus 1: Saldo cukup -> bayar penuh
		dibayar = nominal
		sisaTunggakan = 0
		statusIuran = "lunas"

		saldoBaru := saldo - nominal
		_, err = tx.Exec("UPDATE nasabah SET saldo = ? WHERE id = ?", saldoBaru, nasabahID)
		if err != nil {
			return false, err
		}

		res, err := tx.Exec(`
INSERT INTO iuran_bulanan (nasabah_id, bulan, tahun, nominal, dibayar, tunggakan, status)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
			nasabahID, bulan, tahun, nominal, dibayar, sisaTunggakan, statusIuran)
		if err != nil {
			return false, err
		}
		iuranID, _ := res.LastInsertId()

		// Catat mutasi saldo debit
		_, err = tx.Exec(`
INSERT INTO transaksi_saldo (nasabah_id, setoran_id, iuran_id, nominal, saldo_sebelum, saldo_sesudah, jenis, tanggal)
VALUES (?, NULL, ?, ?, ?, ?, 'iuran', ?)`,
			nasabahID, iuranID, nominal, saldo, saldoBaru, now)
		if err != nil {
			return false, err
		}

	} else if saldo > 0 {
		// Kasus 2: Saldo ada tapi kurang dari 5.000 -> bayar sebagian, sisanya jadi tunggakan
		dibayar = saldo
		sisaTunggakan = nominal - saldo
		statusIuran = "sebagian"

		tunggakanBaru := tunggakan + sisaTunggakan
		_, err = tx.Exec("UPDATE nasabah SET saldo = 0, tunggakan = ? WHERE id = ?", tunggakanBaru, nasabahID)
		if err != nil {
			return false, err
		}

		res, err := tx.Exec(`
INSERT INTO iuran_bulanan (nasabah_id, bulan, tahun, nominal, dibayar, tunggakan, status)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
			nasabahID, bulan, tahun, nominal, dibayar, sisaTunggakan, statusIuran)
		if err != nil {
			return false, err
		}
		iuranID, _ := res.LastInsertId()

		// Catat mutasi saldo debit sebesar saldo yang terpotong
		_, err = tx.Exec(`
INSERT INTO transaksi_saldo (nasabah_id, setoran_id, iuran_id, nominal, saldo_sebelum, saldo_sesudah, jenis, tanggal)
VALUES (?, NULL, ?, ?, ?, 0, 'iuran', ?)`,
			nasabahID, iuranID, dibayar, saldo, now)
		if err != nil {
			return false, err
		}

	} else {
		// Kasus 3: Saldo 0 -> seluruhnya dicatat sebagai tunggakan
		dibayar = 0
		sisaTunggakan = nominal
		statusIuran = "belum_bayar"

		tunggakanBaru := tunggakan + sisaTunggakan
		_, err = tx.Exec("UPDATE nasabah SET tunggakan = ? WHERE id = ?", tunggakanBaru, nasabahID)
		if err != nil {
			return false, err
		}

		_, err = tx.Exec(`
INSERT INTO iuran_bulanan (nasabah_id, bulan, tahun, nominal, dibayar, tunggakan, status)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
			nasabahID, bulan, tahun, nominal, dibayar, sisaTunggakan, statusIuran)
		if err != nil {
			return false, err
		}
		// Tidak ada pemotongan saldo, jadi tidak ada record transaksi_saldo
	}

	return true, nil
}

// ProsesIuranPeriode memproses tagihan iuran untuk seluruh nasabah aktif pada bulan & tahun yang dipilih.
func ProsesIuranPeriode(db *sql.DB, bulan, tahun int) (diproses int, sudahAda int, err error) {
	if bulan < 1 || bulan > 12 || tahun < 2020 {
		return 0, 0, errors.New("periode bulan atau tahun tidak valid")
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	// Ambil semua nasabah aktif
	rows, err := tx.Query("SELECT id FROM nasabah WHERE status = 'aktif'")
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, 0, err
		}
		ids = append(ids, id)
	}
	_ = rows.Close()

	now := time.Now()
	for _, id := range ids {
		created, err := prosesIuranNasabahTx(tx, id, bulan, tahun, now)
		if err != nil {
			return 0, 0, err
		}
		if created {
			diproses++
		} else {
			sudahAda++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}

	return diproses, sudahAda, nil
}

// CatchUpIuranOtomatis memastikan tagihan bulan berjalan untuk nasabah aktif sudah terbit (skema catch-up Bab 5.7).
func CatchUpIuranOtomatis(db *sql.DB) error {
	now := time.Now()
	bulan := int(now.Month())
	tahun := now.Year()
	_, _, err := ProsesIuranPeriode(db, bulan, tahun)
	return err
}

// ProsesIuranNasabahSingle memproses tagihan bulan berjalan untuk satu nasabah tertentu (misal: nasabah baru / diaktifkan kembali).
func ProsesIuranNasabahSingle(db *sql.DB, nasabahID int64) error {
	now := time.Now()
	bulan := int(now.Month())
	tahun := now.Year()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = prosesIuranNasabahTx(tx, nasabahID, bulan, tahun, now)
	if err != nil {
		return err
	}
	return tx.Commit()
}


