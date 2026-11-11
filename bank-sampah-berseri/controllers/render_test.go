package controllers

import (
	"path/filepath"
	"testing"
)

func TestNewRenderer(t *testing.T) {
	templatesDir := filepath.Join("..", "templates")
	r, err := NewRenderer(templatesDir, false)
	if err != nil {
		t.Fatalf("Gagal memuat template: %v", err)
	}

	expectedTemplates := []string{
		"admin/dashboard.html",
		"admin/nasabah_list.html",
		"admin/nasabah_form.html",
		"admin/nasabah_detail.html",
		"admin/sampah_list.html",
		"admin/sampah_form.html",
		"admin/pengepul_list.html",
		"admin/pengepul_form.html",
		"admin/pengepul_detail.html",
		"admin/setoran_list.html",
		"admin/setoran_form.html",
		"admin/setoran_detail.html",
		"admin/pemilahan_list.html",
		"admin/pemilahan_form.html",
		"admin/pemilahan_detail.html",
		"admin/stok_list.html",
		"admin/penjualan_list.html",
		"admin/penjualan_form.html",
		"admin/penjualan_detail.html",
		"admin/saldo_index.html",
		"admin/saldo_tarik.html",
		"admin/saldo_mutasi.html",
		"admin/iuran_index.html",
		"admin/laporan_index.html",
		"admin/laporan_laba_rugi.html",
		"admin/laporan_stok.html",
		"admin/laporan_mutasi.html",
		"auth/login.html",
		"nasabah/dashboard.html",
		"nasabah/profil.html",
		"nasabah/mutasi.html",
		"nasabah/setoran_list.html",
		"nasabah/setoran_detail.html",
		"nasabah/katalog.html",
	}

	for _, name := range expectedTemplates {
		if _, ok := r.pages[name]; !ok {
			t.Errorf("Template %s tidak terdaftar di Renderer", name)
		}
	}
}

