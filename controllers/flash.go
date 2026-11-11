package controllers

import (
	"encoding/base64"
	"net/http"
	"strings"
)

const flashCookie = "bsb_flash"

// SetFlash menyimpan pesan untuk ditampilkan di halaman berikutnya.
// kind: "success" atau "error".
func SetFlash(w http.ResponseWriter, kind, msg string) {
	val := base64.RawURLEncoding.EncodeToString([]byte(kind + "|" + msg))
	http.SetCookie(w, &http.Cookie{
		Name: flashCookie, Value: val, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 60,
	})
}

// PopFlash membaca pesan lalu menghapus cookie-nya. Harus dipanggil sebelum Render.
func PopFlash(w http.ResponseWriter, r *http.Request) (success, errMsg string) {
	c, err := r.Cookie(flashCookie)
	if err != nil {
		return "", ""
	}
	http.SetCookie(w, &http.Cookie{Name: flashCookie, Value: "", Path: "/", MaxAge: -1})

	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		return "", ""
	}
	kind, msg, ok := strings.Cut(string(raw), "|")
	if !ok {
		return "", ""
	}
	if kind == "success" {
		return msg, ""
	}
	return "", msg
}
