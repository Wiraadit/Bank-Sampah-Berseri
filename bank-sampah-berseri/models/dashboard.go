package models

import (
	"database/sql"
	"time"
)

type DashboardStats struct {
	TotalNasabah     int
	NasabahAktif     int
	TotalSaldo       float64
	TotalTunggakan   float64
	NasabahMenunggak int
	TotalStok        float64

	SetoranBulanJumlah   int
	SetoranBulanNilai    float64
	PenjualanBulanJumlah int
	PenjualanBulanNilai  float64
}

// GetDashboardStats mengumpulkan angka ringkasan. Periode "bulan ini" = [awal, akhir).
func GetDashboardStats(db *sql.DB, awal, akhir time.Time) (*DashboardStats, error) {
	var s DashboardStats

	err := db.QueryRow(`
SELECT COUNT(*),
       COALESCE(SUM(status = 'aktif'), 0),
       COALESCE(SUM(saldo), 0),
       COALESCE(SUM(tunggakan), 0),
       COALESCE(SUM(tunggakan > 0), 0)
FROM nasabah`).Scan(&s.TotalNasabah, &s.NasabahAktif, &s.TotalSaldo, &s.TotalTunggakan, &s.NasabahMenunggak)
	if err != nil {
		return nil, err
	}

	err = db.QueryRow(
		"SELECT COUNT(*), COALESCE(SUM(total_nilai),0) FROM setoran WHERE tanggal >= ? AND tanggal < ?",
		awal, akhir,
	).Scan(&s.SetoranBulanJumlah, &s.SetoranBulanNilai)
	if err != nil {
		return nil, err
	}

	err = db.QueryRow(
		"SELECT COUNT(*), COALESCE(SUM(total_pendapatan),0) FROM penjualan WHERE tanggal >= ? AND tanggal < ?",
		awal, akhir,
	).Scan(&s.PenjualanBulanJumlah, &s.PenjualanBulanNilai)
	if err != nil {
		return nil, err
	}

	stok, err := ListStok(db)
	if err != nil {
		return nil, err
	}
	for _, it := range stok {
		s.TotalStok += it.Stok
	}
	return &s, nil
}
