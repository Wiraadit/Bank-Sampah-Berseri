package models

import "database/sql"

// Nama jenis yang diperlakukan sebagai bahan pemilahan (sama dengan yang dikunci di controllers/sampah_controller.go).
const NamaPlastikCampur = "Plastik Campur"

type StokItem struct {
	ID        int64
	NamaJenis string
	HargaBeli float64
	HargaJual float64
	Stok      float64
}

// ListStok menghitung stok setiap jenis sampah dari transaksi yang tersimpan (tanpa tabel mutasi stok):
//
//	Stok = Setoran + Hasil Pemilahan - (Berat Awal Pemilahan, khusus Plastik Campur) - Penjualan
//
// Catatan: kolom detail_penjualan.berat diperlakukan sebagai berat stok yang keluar.
func ListStok(db *sql.DB) ([]StokItem, error) {
	const q = `
SELECT j.id, j.nama_jenis, j.harga_beli, j.harga_jual,
  COALESCE((SELECT SUM(ds.berat)       FROM detail_setoran   ds WHERE ds.jenis_sampah_id = j.id), 0)
+ COALESCE((SELECT SUM(dp.berat_hasil) FROM detail_pemilahan dp WHERE dp.jenis_sampah_id = j.id), 0)
- CASE WHEN j.nama_jenis = ?
       THEN COALESCE((SELECT SUM(p.berat_awal) FROM pemilahan p), 0)
       ELSE 0 END
- COALESCE((SELECT SUM(dj.berat)       FROM detail_penjualan dj WHERE dj.jenis_sampah_id = j.id), 0)
  AS stok
FROM jenis_sampah j
ORDER BY j.id`

	rows, err := db.Query(q, NamaPlastikCampur)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []StokItem
	for rows.Next() {
		var s StokItem
		if err := rows.Scan(&s.ID, &s.NamaJenis, &s.HargaBeli, &s.HargaJual, &s.Stok); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// GetStokJenisTx mengembalikan sisa stok untuk satu jenis sampah di dalam transaksi aktif.
func GetStokJenisTx(tx *sql.Tx, jenisID int64) (float64, error) {
	const q = `
SELECT
  COALESCE((SELECT SUM(ds.berat)       FROM detail_setoran   ds WHERE ds.jenis_sampah_id = j.id), 0)
+ COALESCE((SELECT SUM(dp.berat_hasil) FROM detail_pemilahan dp WHERE dp.jenis_sampah_id = j.id), 0)
- CASE WHEN j.nama_jenis = ?
       THEN COALESCE((SELECT SUM(p.berat_awal) FROM pemilahan p), 0)
       ELSE 0 END
- COALESCE((SELECT SUM(dj.berat)       FROM detail_penjualan dj WHERE dj.jenis_sampah_id = j.id), 0)
  AS stok
FROM jenis_sampah j
WHERE j.id = ?`

	var stok float64
	err := tx.QueryRow(q, NamaPlastikCampur, jenisID).Scan(&stok)
	if err != nil {
		return 0, err
	}
	return stok, nil
}

// GetStokJenis mengembalikan sisa stok untuk satu jenis sampah.
func GetStokJenis(db *sql.DB, jenisID int64) (float64, error) {
	const q = `
SELECT
  COALESCE((SELECT SUM(ds.berat)       FROM detail_setoran   ds WHERE ds.jenis_sampah_id = j.id), 0)
+ COALESCE((SELECT SUM(dp.berat_hasil) FROM detail_pemilahan dp WHERE dp.jenis_sampah_id = j.id), 0)
- CASE WHEN j.nama_jenis = ?
       THEN COALESCE((SELECT SUM(p.berat_awal) FROM pemilahan p), 0)
       ELSE 0 END
- COALESCE((SELECT SUM(dj.berat)       FROM detail_penjualan dj WHERE dj.jenis_sampah_id = j.id), 0)
  AS stok
FROM jenis_sampah j
WHERE j.id = ?`

	var stok float64
	err := db.QueryRow(q, NamaPlastikCampur, jenisID).Scan(&stok)
	if err != nil {
		return 0, err
	}
	return stok, nil
}

// GetInfoPlastikCampur mengembalikan ID dan stok Plastik Campur saat ini.
func GetInfoPlastikCampur(db *sql.DB) (int64, float64, error) {
	var id int64
	err := db.QueryRow("SELECT id FROM jenis_sampah WHERE nama_jenis = ?", NamaPlastikCampur).Scan(&id)
	if err != nil {
		return 0, 0, err
	}
	stok, err := GetStokJenis(db, id)
	if err != nil {
		return 0, 0, err
	}
	return id, stok, nil
}
