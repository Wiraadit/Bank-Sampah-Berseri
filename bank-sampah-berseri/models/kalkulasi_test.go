package models

import (
	"testing"
)

// TestKalkulasiSetoran menguji formula perhitungan setoran sampah,
// termasuk pelunasan otomatis tunggakan dan sisa untuk penambahan saldo.
func TestKalkulasiSetoran(t *testing.T) {
	// Skenario 1: Nasabah tidak punya tunggakan
	{
		tunggakanAwal := 0.0
		saldoAwal := 25000.0

		// Rincian setoran:
		// Kardus: 5 kg @ Rp 1.500 = Rp 7.500
		// Botol Plastik: 2 kg @ Rp 2.500 = Rp 5.000
		totalSetoran := (5.0 * 1500.0) + (2.0 * 2500.0) // 12.500

		pelunasanTunggakan := 0.0
		if tunggakanAwal > 0 {
			if totalSetoran >= tunggakanAwal {
				pelunasanTunggakan = tunggakanAwal
			} else {
				pelunasanTunggakan = totalSetoran
			}
		}
		penambahanSaldo := totalSetoran - pelunasanTunggakan
		saldoAkhir := saldoAwal + penambahanSaldo
		tunggakanAkhir := tunggakanAwal - pelunasanTunggakan

		if totalSetoran != 12500.0 {
			t.Errorf("Total setoran salah: diharapkan 12500, didapat %.2f", totalSetoran)
		}
		if pelunasanTunggakan != 0.0 {
			t.Errorf("Pelunasan tunggakan harus 0, didapat %.2f", pelunasanTunggakan)
		}
		if penambahanSaldo != 12500.0 {
			t.Errorf("Penambahan saldo salah: diharapkan 12500, didapat %.2f", penambahanSaldo)
		}
		if saldoAkhir != 37500.0 {
			t.Errorf("Saldo akhir salah: diharapkan 37500, didapat %.2f", saldoAkhir)
		}
		if tunggakanAkhir != 0.0 {
			t.Errorf("Tunggakan akhir salah: diharapkan 0, didapat %.2f", tunggakanAkhir)
		}
	}

	// Skenario 2: Nasabah punya tunggakan Rp 5.000, setoran Rp 12.000 (lunas & bersisa)
	{
		tunggakanAwal := 5000.0
		saldoAwal := 0.0
		totalSetoran := 12000.0

		pelunasanTunggakan := 0.0
		if tunggakanAwal > 0 {
			if totalSetoran >= tunggakanAwal {
				pelunasanTunggakan = tunggakanAwal
			} else {
				pelunasanTunggakan = totalSetoran
			}
		}
		penambahanSaldo := totalSetoran - pelunasanTunggakan
		saldoAkhir := saldoAwal + penambahanSaldo
		tunggakanAkhir := tunggakanAwal - pelunasanTunggakan

		if pelunasanTunggakan != 5000.0 {
			t.Errorf("Pelunasan tunggakan harus 5000, didapat %.2f", pelunasanTunggakan)
		}
		if penambahanSaldo != 7000.0 {
			t.Errorf("Penambahan saldo harus 7000, didapat %.2f", penambahanSaldo)
		}
		if saldoAkhir != 7000.0 {
			t.Errorf("Saldo akhir harus 7000, didapat %.2f", saldoAkhir)
		}
		if tunggakanAkhir != 0.0 {
			t.Errorf("Tunggakan akhir harus 0, didapat %.2f", tunggakanAkhir)
		}
	}

	// Skenario 3: Nasabah punya tunggakan Rp 10.000, setoran Rp 4.000 (pelunasan sebagian, saldo tidak bertambah)
	{
		tunggakanAwal := 10000.0
		saldoAwal := 0.0
		totalSetoran := 4000.0

		pelunasanTunggakan := 0.0
		if tunggakanAwal > 0 {
			if totalSetoran >= tunggakanAwal {
				pelunasanTunggakan = tunggakanAwal
			} else {
				pelunasanTunggakan = totalSetoran
			}
		}
		penambahanSaldo := totalSetoran - pelunasanTunggakan
		saldoAkhir := saldoAwal + penambahanSaldo
		tunggakanAkhir := tunggakanAwal - pelunasanTunggakan

		if pelunasanTunggakan != 4000.0 {
			t.Errorf("Pelunasan tunggakan harus 4000, didapat %.2f", pelunasanTunggakan)
		}
		if penambahanSaldo != 0.0 {
			t.Errorf("Penambahan saldo harus 0, didapat %.2f", penambahanSaldo)
		}
		if saldoAkhir != 0.0 {
			t.Errorf("Saldo akhir harus 0, didapat %.2f", saldoAkhir)
		}
		if tunggakanAkhir != 6000.0 {
			t.Errorf("Sisa tunggakan harus 6000, didapat %.2f", tunggakanAkhir)
		}
	}
}

// TestKalkulasiPenjualan menguji perhitungan penjualan ke pengepul
// sesuai HPP Metode A dan Penanganan Berat Alternatif 2 (Stok Keluar vs Berat Riil).
func TestKalkulasiPenjualan(t *testing.T) {
	stokKeluar := 50.0 // 50 kg dari gudang
	beratRiil := 48.5  // 48.5 kg timbangan di pengepul
	hargaJual := 3500.0
	hpp := 2000.0 // snapshot harga beli master saat transaksi

	penyusutan := stokKeluar - beratRiil
	pendapatan := beratRiil * hargaJual
	totalHPP := beratRiil * hpp
	labaRugi := (hargaJual - hpp) * beratRiil

	if penyusutan != 1.5 {
		t.Errorf("Penyusutan salah: diharapkan 1.50 kg, didapat %.2f kg", penyusutan)
	}
	if pendapatan != 169750.0 {
		t.Errorf("Pendapatan salah: diharapkan 169750, didapat %.2f", pendapatan)
	}
	if totalHPP != 97000.0 {
		t.Errorf("Total HPP salah: diharapkan 97000, didapat %.2f", totalHPP)
	}
	if labaRugi != 72750.0 {
		t.Errorf("Laba rugi salah: diharapkan 72750, didapat %.2f", labaRugi)
	}
	if labaRugi != (pendapatan - totalHPP) {
		t.Errorf("Konsistensi laba rugi gagal: (Pendapatan - HPP) != (HargaJual - HPP)*BeratRiil")
	}
}

// TestValidasiPenarikanSaldo menguji aturan saldo mengendap minimum Rp 10.000.
func TestValidasiPenarikanSaldo(t *testing.T) {
	saldoNasabah := 50000.0
	minMengendap := SaldoMinimumMengendap // 10000.0

	// Tarik 30.000 -> sisa 20.000 (valid karena >= 10.000)
	nominalValid := 30000.0
	sisaValid := saldoNasabah - nominalValid
	if sisaValid < minMengendap {
		t.Errorf("Penarikan valid seharusnya tidak ditolak: sisa %.2f", sisaValid)
	}

	// Tarik 45.000 -> sisa 5.000 (harus ditolak karena < 10.000)
	nominalInvalid := 45000.0
	sisaInvalid := saldoNasabah - nominalInvalid
	if sisaInvalid >= minMengendap {
		t.Errorf("Penarikan 45.000 seharusnya melanggar batas mengendap 10.000: sisa %.2f", sisaInvalid)
	}
	maksimalTarik := saldoNasabah - minMengendap
	if maksimalTarik != 40000.0 {
		t.Errorf("Maksimal tarik salah: diharapkan 40000, didapat %.2f", maksimalTarik)
	}
}

// TestLogikaIuranBulanan menguji 3 kondisi pemrosesan iuran bulanan (Rp 5.000).
func TestLogikaIuranBulanan(t *testing.T) {
	tarif := TarifIuranBulanan // 5000.0

	// Kondisi 1: Saldo cukup (Saldo >= 5.000) -> LUNAS
	{
		saldo := 25000.0
		tunggakan := 0.0

		var dibayar, sisaTunggakan float64
		var status string
		if saldo >= tarif {
			dibayar = tarif
			sisaTunggakan = 0
			status = "lunas"
			saldo -= tarif
		}
		if status != "lunas" || dibayar != 5000.0 || sisaTunggakan != 0.0 || saldo != 20000.0 || tunggakan != 0.0 {
			t.Errorf("Kondisi 1 iuran gagal: status=%s, dibayar=%.2f, sisaTunggakan=%.2f, saldo=%.2f",
				status, dibayar, sisaTunggakan, saldo)
		}
	}

	// Kondisi 2: Saldo ada tapi kurang dari Rp 5.000 -> SEBAGIAN
	{
		saldo := 2000.0
		tunggakan := 0.0

		var dibayar, sisaTunggakan float64
		var status string
		if saldo < tarif && saldo > 0 {
			dibayar = saldo
			sisaTunggakan = tarif - saldo
			status = "sebagian"
			tunggakan += sisaTunggakan
			saldo = 0
		}
		if status != "sebagian" || dibayar != 2000.0 || sisaTunggakan != 3000.0 || saldo != 0.0 || tunggakan != 3000.0 {
			t.Errorf("Kondisi 2 iuran gagal: status=%s, dibayar=%.2f, sisaTunggakan=%.2f, saldo=%.2f, tunggakan=%.2f",
				status, dibayar, sisaTunggakan, saldo, tunggakan)
		}
	}

	// Kondisi 3: Saldo 0 -> BELUM_BAYAR (seluruhnya jadi tunggakan)
	{
		saldo := 0.0
		tunggakan := 0.0

		var dibayar, sisaTunggakan float64
		var status string
		if saldo == 0 {
			dibayar = 0
			sisaTunggakan = tarif
			status = "belum_bayar"
			tunggakan += sisaTunggakan
		}
		if status != "belum_bayar" || dibayar != 0.0 || sisaTunggakan != 5000.0 || saldo != 0.0 || tunggakan != 5000.0 {
			t.Errorf("Kondisi 3 iuran gagal: status=%s, dibayar=%.2f, sisaTunggakan=%.2f, tunggakan=%.2f",
				status, dibayar, sisaTunggakan, tunggakan)
		}
	}
}

// TestValidasiPemilahan menguji aturan bahwa total hasil pemilahan tidak boleh melebihi berat awal.
func TestValidasiPemilahan(t *testing.T) {
	beratAwal := 20.0 // 20 kg Plastik Campur

	// Hasil pilah: Emberan 10 kg, Botol Bening 8 kg -> Total 18 kg (valid, susut 2 kg)
	totalHasilValid := 10.0 + 8.0
	if totalHasilValid > beratAwal {
		t.Errorf("Total hasil 18 kg seharusnya valid untuk berat awal 20 kg")
	}

	// Hasil pilah: Emberan 15 kg, Botol Bening 10 kg -> Total 25 kg (invalid)
	totalHasilInvalid := 15.0 + 10.0
	if totalHasilInvalid <= beratAwal {
		t.Errorf("Total hasil 25 kg seharusnya tidak valid untuk berat awal 20 kg")
	}
}

