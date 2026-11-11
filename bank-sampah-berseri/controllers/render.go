package controllers

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"bank-sampah-berseri/middleware"
)

// PageData adalah data yang dikirim ke semua template.
type PageData struct {
	Title   string
	Active  string // menu sidebar yang aktif
	User    *middleware.Session
	CSRF    string // diisi otomatis oleh Render
	Error   string
	Success string
	Data    any // data khusus per halaman
}

// Fungsi bantu yang bisa dipakai di template.
var funcMap = template.FuncMap{
	"rupiah": rupiah,
	"kg":     kg,
}

// rupiah: 1234567.5 -> "Rp 1.234.568"
func rupiah(v float64) string {
	n := int64(math.Round(v))
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(ch)
	}
	if neg {
		return "-Rp " + b.String()
	}
	return "Rp " + b.String()
}

// kg: 12.5 -> "12,50 kg"
func kg(v float64) string {
	return strings.Replace(fmt.Sprintf("%.2f", v), ".", ",", 1) + " kg"
}

// Renderer memuat dan merender template HTML.
type Renderer struct {
	dir   string
	dev   bool // true: template dibaca ulang tiap request (tidak perlu restart saat edit HTML)
	mu    sync.RWMutex
	pages map[string]*template.Template
}

func NewRenderer(dir string, dev bool) (*Renderer, error) {
	r := &Renderer{dir: dir, dev: dev}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

// load menggabungkan layout dengan setiap halaman admin/nasabah.
// Halaman login berdiri sendiri (tanpa sidebar).
func (r *Renderer) load() error {
	pages := map[string]*template.Template{}
	layout := filepath.Join(r.dir, "layouts", "app.html")

	for _, sub := range []string{"admin", "nasabah"} {
		files, err := filepath.Glob(filepath.Join(r.dir, sub, "*.html"))
		if err != nil {
			return err
		}
		for _, f := range files {
			t, err := template.New("page").Funcs(funcMap).ParseFiles(layout, f)
			if err != nil {
				return err
			}
			pages[sub+"/"+filepath.Base(f)] = t
		}
	}

	login, err := template.New("page").Funcs(funcMap).ParseFiles(filepath.Join(r.dir, "auth", "login.html"))
	if err != nil {
		return err
	}
	pages["auth/login.html"] = login

	r.mu.Lock()
	r.pages = pages
	r.mu.Unlock()
	return nil
}

// Render menampilkan template bernama name (contoh: "admin/dashboard.html").
func (r *Renderer) Render(w http.ResponseWriter, status int, name string, data PageData) {
	if r.dev {
		if err := r.load(); err != nil {
			log.Println("gagal memuat template:", err)
			http.Error(w, "Kesalahan template", http.StatusInternalServerError)
			return
		}
	}
	r.mu.RLock()
	t, ok := r.pages[name]
	r.mu.RUnlock()
	if !ok {
		log.Println("template tidak ditemukan:", name)
		http.Error(w, "Halaman tidak ditemukan", http.StatusInternalServerError)
		return
	}

	// Token CSRF diambil dari session sehingga controller tidak perlu mengurusnya.
	if data.User != nil {
		data.CSRF = data.User.CSRFToken
	}

	// Render ke buffer dulu agar kalau error, tidak ada HTML setengah jadi terkirim.
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout", data); err != nil {
		log.Println("gagal render", name, ":", err)
		http.Error(w, "Terjadi kesalahan pada server", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	buf.WriteTo(w)
}
