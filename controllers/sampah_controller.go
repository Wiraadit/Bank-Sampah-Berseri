package controllers

import (
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"bank-sampah-berseri/middleware"
	"bank-sampah-berseri/models"
)

// Jenis ini dipakai oleh logika pemilahan (Tahap 5), jadi namanya tidak boleh diubah.
const namaPlastikCampur = "Plastik Campur"

// Hanya angka bulat, maksimal 9 digit.
var reHarga = regexp.MustCompile(`^[0-9]{1,9}$`)

type SampahController struct {
	DB    *sql.DB
	Views *Renderer
}

func NewSampahController(db *sql.DB, views *Renderer) *SampahController {
	return &SampahController{DB: db, Views: views}
}

// Data form. Harga disimpan sebagai string agar isian pengguna tidak hilang saat error.
type sampahForm struct {
	IsEdit       bool
	ID           int64
	Nama         string
	HargaBeli    string
	HargaJual    string
	NamaTerkunci bool
}

// validasi mengembalikan pesan error (atau "") beserta harga yang sudah diparse.
func (f *sampahForm) validasi() (msg string, beli, jual float64) {
	f.Nama = strings.TrimSpace(f.Nama)
	f.HargaBeli = strings.TrimSpace(f.HargaBeli)
	f.HargaJual = strings.TrimSpace(f.HargaJual)

	if f.Nama == "" {
		return "Nama jenis sampah wajib diisi", 0, 0
	}
	if utf8.RuneCountInString(f.Nama) > 50 {
		return "Nama jenis sampah maksimal 50 karakter", 0, 0
	}
	if !reHarga.MatchString(f.HargaBeli) {
		return "Harga beli harus berupa angka bulat tanpa titik/koma (contoh: 1500)", 0, 0
	}
	if !reHarga.MatchString(f.HargaJual) {
		return "Harga jual harus berupa angka bulat tanpa titik/koma (contoh: 2000)", 0, 0
	}
	b, _ := strconv.ParseInt(f.HargaBeli, 10, 64)
	j, _ := strconv.ParseInt(f.HargaJual, 10, 64)
	return "", float64(b), float64(j)
}

func (c *SampahController) page(w http.ResponseWriter, r *http.Request, title string, data any) PageData {
	ok, errMsg := PopFlash(w, r)
	return PageData{
		Title: title, Active: "sampah", User: middleware.CurrentUser(r),
		Success: ok, Error: errMsg, Data: data,
	}
}

func (c *SampahController) muat(w http.ResponseWriter, r *http.Request) (*models.JenisSampah, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return nil, false
	}
	j, err := models.GetJenisSampahByID(c.DB, id)
	if errors.Is(err, models.ErrNotFound) {
		http.NotFound(w, r)
		return nil, false
	}
	if err != nil {
		serverError(w, "muat jenis sampah", err)
		return nil, false
	}
	return j, true
}

// GET /admin/sampah
func (c *SampahController) Index(w http.ResponseWriter, r *http.Request) {
	items, err := models.ListJenisSampah(c.DB)
	if err != nil {
		serverError(w, "list jenis sampah", err)
		return
	}
	c.Views.Render(w, http.StatusOK, "admin/sampah_list.html",
		c.page(w, r, "Jenis Sampah", items))
}

// GET /admin/sampah/tambah
func (c *SampahController) Create(w http.ResponseWriter, r *http.Request) {
	c.Views.Render(w, http.StatusOK, "admin/sampah_form.html",
		c.page(w, r, "Tambah Jenis Sampah", sampahForm{HargaBeli: "0", HargaJual: "0"}))
}

// POST /admin/sampah
func (c *SampahController) Store(w http.ResponseWriter, r *http.Request) {
	f := sampahForm{
		Nama:      r.PostFormValue("nama_jenis"),
		HargaBeli: r.PostFormValue("harga_beli"),
		HargaJual: r.PostFormValue("harga_jual"),
	}
	showError := func(status int, msg string) {
		p := c.page(w, r, "Tambah Jenis Sampah", f)
		p.Error = msg
		c.Views.Render(w, status, "admin/sampah_form.html", p)
	}

	msg, beli, jual := f.validasi()
	if msg != "" {
		showError(http.StatusBadRequest, msg)
		return
	}
	if _, err := models.CreateJenisSampah(c.DB, f.Nama, beli, jual); err != nil {
		if errors.Is(err, models.ErrDuplicate) {
			showError(http.StatusBadRequest, "Jenis sampah dengan nama tersebut sudah ada")
			return
		}
		serverError(w, "simpan jenis sampah", err)
		return
	}
	SetFlash(w, "success", "Jenis sampah berhasil ditambahkan")
	http.Redirect(w, r, "/admin/sampah", http.StatusSeeOther)
}

// GET /admin/sampah/{id}/edit
func (c *SampahController) Edit(w http.ResponseWriter, r *http.Request) {
	j, ok := c.muat(w, r)
	if !ok {
		return
	}
	f := sampahForm{
		IsEdit: true, ID: j.ID, Nama: j.NamaJenis,
		HargaBeli:    strconv.FormatInt(int64(j.HargaBeli), 10),
		HargaJual:    strconv.FormatInt(int64(j.HargaJual), 10),
		NamaTerkunci: strings.EqualFold(j.NamaJenis, namaPlastikCampur),
	}
	c.Views.Render(w, http.StatusOK, "admin/sampah_form.html",
		c.page(w, r, "Edit Jenis Sampah", f))
}

// POST /admin/sampah/{id}
func (c *SampahController) Update(w http.ResponseWriter, r *http.Request) {
	j, ok := c.muat(w, r)
	if !ok {
		return
	}
	f := sampahForm{
		IsEdit: true, ID: j.ID,
		Nama:         r.PostFormValue("nama_jenis"),
		HargaBeli:    r.PostFormValue("harga_beli"),
		HargaJual:    r.PostFormValue("harga_jual"),
		NamaTerkunci: strings.EqualFold(j.NamaJenis, namaPlastikCampur),
	}
	if f.NamaTerkunci {
		f.Nama = j.NamaJenis // abaikan nama dari form (input disabled tidak dikirim)
	}

	showError := func(status int, msg string) {
		p := c.page(w, r, "Edit Jenis Sampah", f)
		p.Error = msg
		c.Views.Render(w, status, "admin/sampah_form.html", p)
	}

	msg, beli, jual := f.validasi()
	if msg != "" {
		showError(http.StatusBadRequest, msg)
		return
	}
	if err := models.UpdateJenisSampah(c.DB, j.ID, f.Nama, beli, jual); err != nil {
		if errors.Is(err, models.ErrDuplicate) {
			showError(http.StatusBadRequest, "Jenis sampah dengan nama tersebut sudah ada")
			return
		}
		serverError(w, "update jenis sampah", err)
		return
	}
	SetFlash(w, "success", "Jenis sampah berhasil diperbarui")
	http.Redirect(w, r, "/admin/sampah", http.StatusSeeOther)
}
