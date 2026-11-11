package controllers

import (
	"fmt"
	"strings"
	"time"
)

func init() {
	funcMap["tgl"] = func(t time.Time) string { return t.Format("02-01-2006 15:04") }

	funcMap["labelJenis"] = func(jenis string) string {
		switch jenis {
		case "setoran":
			return "Setoran sampah"
		case "bayar_tunggakan":
			return "Pembayaran tunggakan"
		case "penarikan":
			return "Penarikan saldo"
		case "iuran":
			return "Iuran bulanan"
		}
		return jenis
	}

	// Kredit = saldo bertambah. Selain setoran, saldo berkurang (debit).
	funcMap["isKredit"] = func(jenis string) bool { return jenis == "setoran" }

	funcMap["persen"] = func(bagian, total float64) string {
		if total <= 0 {
			return "0,0%"
		}
		p := (bagian / total) * 100
		return strings.Replace(fmt.Sprintf("%.1f", p), ".", ",", 1) + "%"
	}
}
