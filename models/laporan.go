package models

import (
	"database/sql"
	"time"
)

// LaporanLabaRugiItem rincian per baris item penjualan.
type LaporanLabaRugiItem struct {
	ID           int64
	Tanggal      time.Time
	NamaPengepul string
	NamaJenis    string
	StokKeluar   float64
	BeratRiil    float64
	Penyusutan   float64
	HargaJual    float64
	HPP          float64
	Pendapatan   float64
	TotalHPP     float64
	LabaRugi     float64
}

type LaporanLabaRugi struct {
	Dari            time.Time
	Sampai          time.Time
	JumlahTransaksi int
	TotalStokKeluar float64
	TotalBeratRiil  float64
	TotalPenyusutan float64
	TotalPendapatan float64
	TotalHPP        float64
	TotalLabaRugi   float64
	Items           []LaporanLabaRugiItem
}

// GetLaporanLabaRugi mengambil rekapitulasi laba rugi penjualan dalam rentang [dari, sampai].
func GetLaporanLabaRugi(db *sql.DB, dari, sampai time.Time) (*LaporanLabaRugi, error) {
	// sampai ditambah 1 hari agar mencakup hingga akhir hari yang dipilih
	sampaiAkhir := sampai.Add(24 * time.Hour)

	const q = `
SELECT p.id, p.tanggal, pg.nama, j.nama_jenis,
       dj.berat, dj.berat_riil, dj.harga_jual, dj.hpp
FROM penjualan p
JOIN pengepul pg ON pg.id = p.pengepul_id
JOIN detail_penjualan dj ON dj.penjualan_id = p.id
JOIN jenis_sampah j ON j.id = dj.jenis_sampah_id
WHERE p.tanggal >= ? AND p.tanggal < ?
ORDER BY p.tanggal ASC, p.id ASC`

	rows, err := db.Query(q, dari, sampaiAkhir)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lap := &LaporanLabaRugi{
		Dari:   dari,
		Sampai: sampai,
	}

	for rows.Next() {
		var it LaporanLabaRugiItem
		if err := rows.Scan(&it.ID, &it.Tanggal, &it.NamaPengepul, &it.NamaJenis,
			&it.StokKeluar, &it.BeratRiil, &it.HargaJual, &it.HPP); err != nil {
			return nil, err
		}

		it.Penyusutan = it.StokKeluar - it.BeratRiil
		it.Pendapatan = it.BeratRiil * it.HargaJual
		it.TotalHPP = it.BeratRiil * it.HPP
		it.LabaRugi = (it.HargaJual - it.HPP) * it.BeratRiil

		lap.JumlahTransaksi++
		lap.TotalStokKeluar += it.StokKeluar
		lap.TotalBeratRiil += it.BeratRiil
		lap.TotalPenyusutan += it.Penyusutan
		lap.TotalPendapatan += it.Pendapatan
		lap.TotalHPP += it.TotalHPP
		lap.TotalLabaRugi += it.LabaRugi

		lap.Items = append(lap.Items, it)
	}

	return lap, rows.Err()
}

// LaporanStokItem rincian pergerakan stok per jenis sampah.
type LaporanStokItem struct {
	ID             int64
	NamaJenis      string
	HargaBeli      float64
	HargaJual      float64
	TotalSetoran   float64
	TotalHasilPilah float64
	TotalBahanPilah float64
	TotalPenjualan float64
	SisaStok       float64
	NilaiAset      float64
}

type LaporanStokGudang struct {
	TotalBobot     float64
	TotalNilaiAset float64
	Items          []LaporanStokItem
}

// GetLaporanStokGudang menghitung rekapitulasi lengkap keluar-masuk stok gudang.
func GetLaporanStokGudang(db *sql.DB) (*LaporanStokGudang, error) {
	const q = `
SELECT j.id, j.nama_jenis, j.harga_beli, j.harga_jual,
  COALESCE((SELECT SUM(ds.berat)       FROM detail_setoran   ds WHERE ds.jenis_sampah_id = j.id), 0) AS tot_setoran,
  COALESCE((SELECT SUM(dp.berat_hasil) FROM detail_pemilahan dp WHERE dp.jenis_sampah_id = j.id), 0) AS tot_hasil_pilah,
  CASE WHEN j.nama_jenis = ?
       THEN COALESCE((SELECT SUM(p.berat_awal) FROM pemilahan p), 0)
       ELSE 0 END AS tot_bahan_pilah,
  COALESCE((SELECT SUM(dj.berat)       FROM detail_penjualan dj WHERE dj.jenis_sampah_id = j.id), 0) AS tot_penjualan
FROM jenis_sampah j
ORDER BY j.id ASC`

	rows, err := db.Query(q, NamaPlastikCampur)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lap := &LaporanStokGudang{}
	for rows.Next() {
		var it LaporanStokItem
		if err := rows.Scan(&it.ID, &it.NamaJenis, &it.HargaBeli, &it.HargaJual,
			&it.TotalSetoran, &it.TotalHasilPilah, &it.TotalBahanPilah, &it.TotalPenjualan); err != nil {
			return nil, err
		}

		it.SisaStok = it.TotalSetoran + it.TotalHasilPilah - it.TotalBahanPilah - it.TotalPenjualan
		if it.SisaStok > 0 {
			it.NilaiAset = it.SisaStok * it.HargaBeli
			lap.TotalBobot += it.SisaStok
			lap.TotalNilaiAset += it.NilaiAset
		}

		lap.Items = append(lap.Items, it)
	}

	return lap, rows.Err()
}

type LaporanMutasiData struct {
	Dari         time.Time
	Sampai       time.Time
	NasabahID    int64
	NamaNasabah  string
	TotalKredit  float64
	TotalDebit   float64
	Items        []MutasiGlobal
}

// GetLaporanMutasi mengambil rekapitulasi mutasi saldo dalam rentang tanggal.
func GetLaporanMutasi(db *sql.DB, nasabahID int64, dari, sampai time.Time) (*LaporanMutasiData, error) {
	sampaiAkhir := sampai.Add(24 * time.Hour)

	q := `
SELECT ts.id, ts.nasabah_id, n.kode_nasabah, n.nama,
       ts.tanggal, ts.jenis, ts.nominal, ts.saldo_sebelum, ts.saldo_sesudah
FROM transaksi_saldo ts
JOIN nasabah n ON n.id = ts.nasabah_id
WHERE ts.tanggal >= ? AND ts.tanggal < ?`

	var args []any
	args = append(args, dari, sampaiAkhir)

	if nasabahID > 0 {
		q += " AND ts.nasabah_id = ?"
		args = append(args, nasabahID)
	}
	q += " ORDER BY ts.tanggal ASC, ts.id ASC"

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lap := &LaporanMutasiData{
		Dari:      dari,
		Sampai:    sampai,
		NasabahID: nasabahID,
	}

	for rows.Next() {
		var m MutasiGlobal
		if err := rows.Scan(&m.ID, &m.NasabahID, &m.KodeNasabah, &m.NamaNasabah,
			&m.Tanggal, &m.Jenis, &m.Nominal, &m.SaldoSebelum, &m.SaldoSesudah); err != nil {
			return nil, err
		}

		if m.Jenis == "setoran" {
			lap.TotalKredit += m.Nominal
		} else {
			lap.TotalDebit += m.Nominal
		}

		lap.Items = append(lap.Items, m)
	}

	if nasabahID > 0 && len(lap.Items) > 0 {
		lap.NamaNasabah = lap.Items[0].NamaNasabah
	}

	return lap, rows.Err()
}

