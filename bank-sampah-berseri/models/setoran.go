package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type RingkasanSetoran struct {
	Jumlah     int
	TotalNilai float64
}

// GetRingkasanSetoranNasabah: jumlah dan total nilai setoran milik satu nasabah.
func GetRingkasanSetoranNasabah(db *sql.DB, nasabahID int64) (*RingkasanSetoran, error) {
	var r RingkasanSetoran
	err := db.QueryRow(
		"SELECT COUNT(*), COALESCE(SUM(total_nilai),0) FROM setoran WHERE nasabah_id = ?", nasabahID,
	).Scan(&r.Jumlah, &r.TotalNilai)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// GetSetoranHarian mengembalikan total nilai setoran per tanggal ("2006-01-02") dalam [dari, sampai).
// Tanggal tanpa setoran tidak ada di map; pemanggil mengisinya dengan 0.
func GetSetoranHarian(db *sql.DB, dari, sampai time.Time) (map[string]float64, error) {
	rows, err := db.Query(`
SELECT DATE_FORMAT(tanggal, '%Y-%m-%d'), COALESCE(SUM(total_nilai), 0)
FROM setoran
WHERE tanggal >= ? AND tanggal < ?
GROUP BY DATE_FORMAT(tanggal, '%Y-%m-%d')`, dari, sampai)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hasil := map[string]float64{}
	for rows.Next() {
		var tgl string
		var nilai float64
		if err := rows.Scan(&tgl, &nilai); err != nil {
			return nil, err
		}
		hasil[tgl] = nilai
	}
	return hasil, rows.Err()
}

// Setoran merepresentasikan header transaksi setoran sampah.
type Setoran struct {
	ID                 int64
	NasabahID          int64
	UserID             int64
	Tanggal            time.Time
	TotalNilai         float64
	KodeNasabah        string
	NamaNasabah        string
	NamaUser           string
	PelunasanTunggakan float64
	PenambahanSaldo    float64
	SaldoSaatIni       float64
	TunggakanSaatIni   float64
	Items              []DetailSetoran
}

// DetailSetoran merepresentasikan rincian per baris jenis sampah yang disetor.
type DetailSetoran struct {
	ID            int64
	SetoranID     int64
	JenisSampahID int64
	NamaJenis     string
	Berat         float64
	HargaBeli     float64
	Subtotal      float64
}

// ItemSetoranInput adalah data input dari form setoran.
type ItemSetoranInput struct {
	JenisSampahID int64
	Berat         float64
}

// HasilSetoran adalah ringkasan hasil kalkulasi transaksi setoran.
type HasilSetoran struct {
	SetoranID          int64
	TotalNilai         float64
	PelunasanTunggakan float64
	PenambahanSaldo    float64
	SaldoAkhir         float64
	TunggakanAkhir     float64
}

// ListSetoran mengembalikan daftar riwayat setoran terbaru.
func ListSetoran(db *sql.DB, search string) ([]Setoran, error) {
	q := `
SELECT s.id, s.nasabah_id, s.user_id, s.tanggal, s.total_nilai,
       n.kode_nasabah, n.nama, u.username
FROM setoran s
JOIN nasabah n ON n.id = s.nasabah_id
JOIN users u ON u.id = s.user_id`
	var args []any
	if search != "" {
		like := "%" + search + "%"
		q += " WHERE n.nama LIKE ? OR n.kode_nasabah LIKE ?"
		args = append(args, like, like)
	}
	q += " ORDER BY s.tanggal DESC, s.id DESC"

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Setoran
	for rows.Next() {
		var s Setoran
		if err := rows.Scan(&s.ID, &s.NasabahID, &s.UserID, &s.Tanggal, &s.TotalNilai,
			&s.KodeNasabah, &s.NamaNasabah, &s.NamaUser); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// ListSetoranByNasabahID mengembalikan seluruh transaksi setoran milik satu nasabah tertentu.
func ListSetoranByNasabahID(db *sql.DB, nasabahID int64) ([]Setoran, error) {
	const q = `
SELECT s.id, s.nasabah_id, s.user_id, s.tanggal, s.total_nilai,
       n.kode_nasabah, n.nama, u.username
FROM setoran s
JOIN nasabah n ON n.id = s.nasabah_id
JOIN users u ON u.id = s.user_id
WHERE s.nasabah_id = ?
ORDER BY s.tanggal DESC, s.id DESC`

	rows, err := db.Query(q, nasabahID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Setoran
	for rows.Next() {
		var s Setoran
		if err := rows.Scan(&s.ID, &s.NasabahID, &s.UserID, &s.Tanggal, &s.TotalNilai,
			&s.KodeNasabah, &s.NamaNasabah, &s.NamaUser); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// GetSetoranByID mengambil detail satu transaksi setoran lengkap beserta rincian itemnya.
func GetSetoranByID(db *sql.DB, id int64) (*Setoran, error) {
	var s Setoran
	err := db.QueryRow(`
SELECT s.id, s.nasabah_id, s.user_id, s.tanggal, s.total_nilai,
       n.kode_nasabah, n.nama, n.saldo, n.tunggakan, u.username
FROM setoran s
JOIN nasabah n ON n.id = s.nasabah_id
JOIN users u ON u.id = s.user_id
WHERE s.id = ?`, id).Scan(
		&s.ID, &s.NasabahID, &s.UserID, &s.Tanggal, &s.TotalNilai,
		&s.KodeNasabah, &s.NamaNasabah, &s.SaldoSaatIni, &s.TunggakanSaatIni, &s.NamaUser,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Ambil rincian detail_setoran
	rows, err := db.Query(`
SELECT ds.id, ds.setoran_id, ds.jenis_sampah_id, j.nama_jenis, ds.berat, ds.harga_beli, ds.subtotal
FROM detail_setoran ds
JOIN jenis_sampah j ON j.id = ds.jenis_sampah_id
WHERE ds.setoran_id = ?
ORDER BY ds.id ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var d DetailSetoran
		if err := rows.Scan(&d.ID, &d.SetoranID, &d.JenisSampahID, &d.NamaJenis, &d.Berat, &d.HargaBeli, &d.Subtotal); err != nil {
			return nil, err
		}
		s.Items = append(s.Items, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Ambil mutasi pelunasan tunggakan jika ada
	var pelunasan float64
	_ = db.QueryRow(
		"SELECT COALESCE(SUM(nominal), 0) FROM transaksi_saldo WHERE setoran_id = ? AND jenis = 'bayar_tunggakan'",
		id,
	).Scan(&pelunasan)

	s.PelunasanTunggakan = pelunasan
	s.PenambahanSaldo = s.TotalNilai - pelunasan
	return &s, nil
}

// CreateSetoran menjalankan transaksi atomik setoran sampah sesuai aturan Bab 5.2.
func CreateSetoran(db *sql.DB, adminUserID int64, nasabahID int64, items []ItemSetoranInput) (*HasilSetoran, error) {
	if len(items) == 0 {
		return nil, errors.New("rincian setoran tidak boleh kosong")
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. Kunci data nasabah
	var n Nasabah
	err = tx.QueryRow(
		"SELECT id, saldo, tunggakan, status FROM nasabah WHERE id = ? FOR UPDATE",
		nasabahID,
	).Scan(&n.ID, &n.Saldo, &n.Tunggakan, &n.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("nasabah tidak ditemukan")
	}
	if err != nil {
		return nil, err
	}
	if n.Status != "aktif" {
		return nil, errors.New("nasabah berstatus nonaktif dan tidak dapat bertransaksi")
	}

	// 2. Ambil master jenis_sampah untuk snapshot harga beli
	jenisMap := make(map[int64]JenisSampah)
	rows, err := tx.Query("SELECT id, nama_jenis, harga_beli, harga_jual FROM jenis_sampah")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var j JenisSampah
		if err := rows.Scan(&j.ID, &j.NamaJenis, &j.HargaBeli, &j.HargaJual); err != nil {
			return nil, err
		}
		jenisMap[j.ID] = j
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 3. Validasi & hitung total
	type barisItem struct {
		jenisID   int64
		berat     float64
		hargaBeli float64
		subtotal  float64
	}
	var rincian []barisItem
	var totalNilai float64

	for _, item := range items {
		if item.Berat <= 0 {
			return nil, errors.New("berat sampah harus lebih besar dari 0 kg")
		}
		js, ok := jenisMap[item.JenisSampahID]
		if !ok {
			return nil, fmt.Errorf("jenis sampah dengan ID %d tidak valid", item.JenisSampahID)
		}
		subtotal := item.Berat * js.HargaBeli
		totalNilai += subtotal
		rincian = append(rincian, barisItem{
			jenisID:   item.JenisSampahID,
			berat:     item.Berat,
			hargaBeli: js.HargaBeli,
			subtotal:  subtotal,
		})
	}

	// 4. Hitung pelunasan tunggakan dan sisa untuk saldo
	pelunasanTunggakan := 0.0
	if n.Tunggakan > 0 {
		if totalNilai >= n.Tunggakan {
			pelunasanTunggakan = n.Tunggakan
		} else {
			pelunasanTunggakan = totalNilai
		}
	}
	penambahanSaldo := totalNilai - pelunasanTunggakan
	saldoSebelum := n.Saldo
	saldoSesudahSetoran := saldoSebelum + totalNilai
	saldoAkhir := saldoSebelum + penambahanSaldo
	tunggakanAkhir := n.Tunggakan - pelunasanTunggakan

	// 5. Simpan ke tabel setoran
	now := time.Now()
	res, err := tx.Exec(
		"INSERT INTO setoran (nasabah_id, user_id, tanggal, total_nilai) VALUES (?, ?, ?, ?)",
		nasabahID, adminUserID, now, totalNilai,
	)
	if err != nil {
		return nil, err
	}
	setoranID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	// 6. Simpan setiap baris detail_setoran
	stmtDetail, err := tx.Prepare(
		"INSERT INTO detail_setoran (setoran_id, jenis_sampah_id, berat, harga_beli, subtotal) VALUES (?, ?, ?, ?, ?)",
	)
	if err != nil {
		return nil, err
	}
	defer stmtDetail.Close()

	for _, b := range rincian {
		if _, err := stmtDetail.Exec(setoranID, b.jenisID, b.berat, b.hargaBeli, b.subtotal); err != nil {
			return nil, err
		}
	}

	// 7. Update saldo dan tunggakan nasabah
	_, err = tx.Exec(
		"UPDATE nasabah SET saldo = ?, tunggakan = ? WHERE id = ?",
		saldoAkhir, tunggakanAkhir, nasabahID,
	)
	if err != nil {
		return nil, err
	}

	// 8. Jika ada pelunasan tunggakan, alokasikan juga ke catatan iuran_bulanan yang belum lunas
	if pelunasanTunggakan > 0 {
		iuranRows, err := tx.Query(`
SELECT id, nominal, dibayar, tunggakan
FROM iuran_bulanan
WHERE nasabah_id = ? AND status != 'lunas'
ORDER BY tahun ASC, bulan ASC, id ASC
FOR UPDATE`, nasabahID)
		if err != nil {
			return nil, err
		}
		defer iuranRows.Close()

		type iuranItem struct {
			id        int64
			nominal   float64
			dibayar   float64
			tunggakan float64
		}
		var tagihan []iuranItem
		for iuranRows.Next() {
			var it iuranItem
			if err := iuranRows.Scan(&it.id, &it.nominal, &it.dibayar, &it.tunggakan); err != nil {
				return nil, err
			}
			tagihan = append(tagihan, it)
		}
		_ = iuranRows.Close()

		sisaAlokasi := pelunasanTunggakan
		for _, tag := range tagihan {
			if sisaAlokasi <= 0 {
				break
			}
			bayar := tag.tunggakan
			if sisaAlokasi < bayar {
				bayar = sisaAlokasi
			}
			newDibayar := tag.dibayar + bayar
			newTunggakan := tag.tunggakan - bayar
			newStatus := "sebagian"
			if newTunggakan <= 0 {
				newStatus = "lunas"
			}
			_, err = tx.Exec(
				"UPDATE iuran_bulanan SET dibayar = ?, tunggakan = ?, status = ? WHERE id = ?",
				newDibayar, newTunggakan, newStatus, tag.id,
			)
			if err != nil {
				return nil, err
			}
			sisaAlokasi -= bayar
		}
	}

	// 9. Catat riwayat di transaksi_saldo
	// a. Kredit nilai total setoran
	_, err = tx.Exec(`
INSERT INTO transaksi_saldo (nasabah_id, setoran_id, iuran_id, nominal, saldo_sebelum, saldo_sesudah, jenis, tanggal)
VALUES (?, ?, NULL, ?, ?, ?, 'setoran', ?)`,
		nasabahID, setoranID, totalNilai, saldoSebelum, saldoSesudahSetoran, now,
	)
	if err != nil {
		return nil, err
	}

	// b. Jika ada pelunasan tunggakan, catat debit bayar_tunggakan
	if pelunasanTunggakan > 0 {
		_, err = tx.Exec(`
INSERT INTO transaksi_saldo (nasabah_id, setoran_id, iuran_id, nominal, saldo_sebelum, saldo_sesudah, jenis, tanggal)
VALUES (?, ?, NULL, ?, ?, ?, 'bayar_tunggakan', ?)`,
			nasabahID, setoranID, pelunasanTunggakan, saldoSesudahSetoran, saldoAkhir, now,
		)
		if err != nil {
			return nil, err
		}
	}

	// 10. Commit transaksi
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &HasilSetoran{
		SetoranID:          setoranID,
		TotalNilai:         totalNilai,
		PelunasanTunggakan: pelunasanTunggakan,
		PenambahanSaldo:    penambahanSaldo,
		SaldoAkhir:         saldoAkhir,
		TunggakanAkhir:     tunggakanAkhir,
	}, nil
}
