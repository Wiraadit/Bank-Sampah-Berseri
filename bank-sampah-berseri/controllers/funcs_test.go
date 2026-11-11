package controllers

import (
	"testing"
	"time"
)

func TestFormatHelpers(t *testing.T) {
	// 1. Uji fungsi rupiah
	testsRupiah := []struct {
		input    float64
		expected string
	}{
		{0, "Rp 0"},
		{5000, "Rp 5.000"},
		{1250000, "Rp 1.250.000"},
		{1250000.4, "Rp 1.250.000"},
		{1250000.6, "Rp 1.250.001"},
		{-15000, "-Rp 15.000"},
	}

	for _, tc := range testsRupiah {
		got := rupiah(tc.input)
		if got != tc.expected {
			t.Errorf("rupiah(%.2f): diharapkan %q, didapat %q", tc.input, tc.expected, got)
		}
	}

	// 2. Uji fungsi kg
	testsKg := []struct {
		input    float64
		expected string
	}{
		{0, "0,00 kg"},
		{12.5, "12,50 kg"},
		{3.14159, "3,14 kg"},
		{100, "100,00 kg"},
	}

	for _, tc := range testsKg {
		got := kg(tc.input)
		if got != tc.expected {
			t.Errorf("kg(%.2f): diharapkan %q, didapat %q", tc.input, tc.expected, got)
		}
	}

	// 3. Uji fungsi persen
	persenFn, ok := funcMap["persen"].(func(float64, float64) string)
	if !ok {
		t.Fatal("funcMap['persen'] tidak bertipe func(float64, float64) string")
	}

	testsPersen := []struct {
		bagian   float64
		total    float64
		expected string
	}{
		{10, 0, "0,0%"},
		{25, 100, "25,0%"},
		{1, 3, "33,3%"},
		{50, 200, "25,0%"},
	}

	for _, tc := range testsPersen {
		got := persenFn(tc.bagian, tc.total)
		if got != tc.expected {
			t.Errorf("persen(%.2f, %.2f): diharapkan %q, didapat %q", tc.bagian, tc.total, tc.expected, got)
		}
	}

	// 4. Uji fungsi labelJenis & isKredit
	labelJenisFn := funcMap["labelJenis"].(func(string) string)
	isKreditFn := funcMap["isKredit"].(func(string) bool)

	if labelJenisFn("setoran") != "Setoran sampah" {
		t.Errorf("labelJenis(setoran) salah: %s", labelJenisFn("setoran"))
	}
	if labelJenisFn("penarikan") != "Penarikan saldo" {
		t.Errorf("labelJenis(penarikan) salah: %s", labelJenisFn("penarikan"))
	}
	if labelJenisFn("iuran") != "Iuran bulanan" {
		t.Errorf("labelJenis(iuran) salah: %s", labelJenisFn("iuran"))
	}

	if !isKreditFn("setoran") {
		t.Error("isKredit(setoran) harus true")
	}
	if isKreditFn("penarikan") {
		t.Error("isKredit(penarikan) harus false")
	}
	if isKreditFn("iuran") {
		t.Error("isKredit(iuran) harus false")
	}

	// 5. Uji fungsi tgl
	tglFn := funcMap["tgl"].(func(time.Time) string)
	sampleTime := time.Date(2026, 11, 11, 14, 30, 0, 0, time.UTC)
	expectedTgl := "11-11-2026 14:30"
	if got := tglFn(sampleTime); got != expectedTgl {
		t.Errorf("tgl() salah: diharapkan %s, didapat %s", expectedTgl, got)
	}
}

