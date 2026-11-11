package controllers

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"bank-sampah-berseri/middleware"
	"bank-sampah-berseri/models"
)

type NasabahController struct {
	DB    *sql.DB
	Views *Renderer
}

func NewNasabahController(db *sql.DB, views *Renderer) *NasabahController {
	return &NasabahController{DB: db, Views: views}
}

var (
	reTelepon  = regexp.MustCompile(`^\+?[0-9]{8,15}$`)
	reUsername = regexp.MustCompile(`^[a-zA-Z0-9_.]{4,50}$`)
	teleponFix = strings.NewReplacer(" ", "", "-", "")
)

// Data untuk template.
type nasabahListData struct {
	Items []models.Nasabah
	Q     string
}

type nasabahForm struct {
	IsEdit    bool
	ID        int64
	Kode      string
	Nama      string
	Alamat    string
	NoTelepon string
	Username  string
}

func (f *nasabahForm) bersihkan() {
	f.Nama = strings.TrimSpace(f.Nama)
	f.Alamat = strings.TrimSpace(f.Alamat)
	f.NoTelepon = teleponFix.Replace(strings.TrimSpace(f.NoTelepon))
	f.Username = strings.TrimSpace(f.Username)
}

// validasi mengembalikan pesan error, atau "" jika valid.
func (f *nasabahForm) validasi(password string) string {
	if f.Nama == "" {
		return "Nama wajib diisi"
	}
	if utf8.RuneCountInString(f.Nama) > 100 {
		return "Nama maksimal 100 karakter"
	}
	if utf8.RuneCountInString(f.Alamat) > 500 {
		return "Alamat maksimal 500 karakter"
	}
	if f.NoTelepon != "" && !reTelepon.MatchString(f.NoTelepon) {
		return "No. telepon harus 8-15 digit angka (boleh diawali +)"
	}
	if !f.IsEdit {
		if !reUsername.MatchString(f.Username) {
			return "Username 4-50 karakter: huruf, angka, titik, atau garis bawah"
		}
		if len(password) < 8 || len(password) > 72 {
			return "Password harus 8-72 karakter"
		}
	}
	return ""
}

// page menyiapkan PageData sekaligus mengambil pesan flash (jika ada).
func (c *NasabahController) page(w http.ResponseWriter, r *http.Request, title string, data any) PageData {
	ok, errMsg := PopFlash(w, r)
	return PageData{
		Title: title, Active: "nasabah", User: middleware.CurrentUser(r),
		Success: ok, Error: errMsg, Data: data,
	}
}

func serverError(w http.ResponseWriter, context string, err error) {
	log.Println(context+":", err)
	http.Error(w, "Terjadi kesalahan pada server", http.StatusInternalServerError)
}

// muatNasabah membaca {id} dari URL dan mengambil datanya. Mengirim 404/500 sendiri jika gagal.
func (c *NasabahController) muatNasabah(w http.ResponseWriter, r *http.Request) (*models.Nasabah, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return nil, false
	}
	n, err := models.GetNasabahByID(c.DB, id)
	if errors.Is(err, models.ErrNotFound) {
		http.NotFound(w, r)
		return nil, false
	}
	if err != nil {
		serverError(w, "muat nasabah", err)
		return nil, false
	}
	return n, true
}

// GET /admin/nasabah
func (c *NasabahController) Index(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	items, err := models.ListNasabah(c.DB, q)
	if err != nil {
		serverError(w, "list nasabah", err)
		return
	}
	c.Views.Render(w, http.StatusOK, "admin/nasabah_list.html",
		c.page(w, r, "Data Nasabah", nasabahListData{Items: items, Q: q}))
}

// GET /admin/nasabah/tambah
func (c *NasabahController) Create(w http.ResponseWriter, r *http.Request) {
	c.Views.Render(w, http.StatusOK, "admin/nasabah_form.html",
		c.page(w, r, "Tambah Nasabah", nasabahForm{}))
}

// POST /admin/nasabah
func (c *NasabahController) Store(w http.ResponseWriter, r *http.Request) {
	f := nasabahForm{
		Nama:      r.PostFormValue("nama"),
		Alamat:    r.PostFormValue("alamat"),
		NoTelepon: r.PostFormValue("no_telepon"),
		Username:  r.PostFormValue("username"),
	}
	password := r.PostFormValue("password")
	f.bersihkan()

	showError := func(status int, msg string) {
		p := c.page(w, r, "Tambah Nasabah", f)
		p.Error = msg
		c.Views.Render(w, status, "admin/nasabah_form.html", p)
	}

	if msg := f.validasi(password); msg != "" {
		showError(http.StatusBadRequest, msg)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		serverError(w, "hash password", err)
		return
	}

	id, err := models.CreateNasabahWithUser(c.DB, f.Username, string(hash), f.Nama, f.Alamat, f.NoTelepon)
	if errors.Is(err, models.ErrDuplicate) {
		showError(http.StatusBadRequest, "Username sudah dipakai, pilih username lain")
		return
	}
	if err != nil {
		serverError(w, "simpan nasabah", err)
		return
	}

	// Otomatis terbitkan tagihan iuran bulan berjalan untuk nasabah baru
	if err := models.ProsesIuranNasabahSingle(c.DB, id); err != nil {
		log.Println("gagal memproses iuran nasabah baru:", err)
	}

	SetFlash(w, "success", "Nasabah berhasil ditambahkan dan iuran bulan ini telah diterbitkan")
	http.Redirect(w, r, fmt.Sprintf("/admin/nasabah/%d", id), http.StatusSeeOther)
}

// GET /admin/nasabah/{id}
func (c *NasabahController) Show(w http.ResponseWriter, r *http.Request) {
	n, ok := c.muatNasabah(w, r)
	if !ok {
		return
	}
	c.Views.Render(w, http.StatusOK, "admin/nasabah_detail.html",
		c.page(w, r, "Detail Nasabah", n))
}

// GET /admin/nasabah/{id}/edit
func (c *NasabahController) Edit(w http.ResponseWriter, r *http.Request) {
	n, ok := c.muatNasabah(w, r)
	if !ok {
		return
	}
	f := nasabahForm{
		IsEdit: true, ID: n.ID, Kode: n.KodeNasabah, Username: n.Username,
		Nama: n.Nama, Alamat: n.Alamat, NoTelepon: n.NoTelepon,
	}
	c.Views.Render(w, http.StatusOK, "admin/nasabah_form.html",
		c.page(w, r, "Edit Nasabah", f))
}

// POST /admin/nasabah/{id}
func (c *NasabahController) Update(w http.ResponseWriter, r *http.Request) {
	n, ok := c.muatNasabah(w, r)
	if !ok {
		return
	}
	f := nasabahForm{
		IsEdit: true, ID: n.ID, Kode: n.KodeNasabah, Username: n.Username,
		Nama:      r.PostFormValue("nama"),
		Alamat:    r.PostFormValue("alamat"),
		NoTelepon: r.PostFormValue("no_telepon"),
	}
	f.bersihkan()
	f.Username = n.Username // username tidak bisa diubah dari form ini

	if msg := f.validasi(""); msg != "" {
		p := c.page(w, r, "Edit Nasabah", f)
		p.Error = msg
		c.Views.Render(w, http.StatusBadRequest, "admin/nasabah_form.html", p)
		return
	}
	if err := models.UpdateNasabah(c.DB, n.ID, f.Nama, f.Alamat, f.NoTelepon); err != nil {
		serverError(w, "update nasabah", err)
		return
	}
	SetFlash(w, "success", "Data nasabah berhasil diperbarui")
	http.Redirect(w, r, fmt.Sprintf("/admin/nasabah/%d", n.ID), http.StatusSeeOther)
}

// POST /admin/nasabah/{id}/status  (aktif <-> nonaktif)
func (c *NasabahController) ToggleStatus(w http.ResponseWriter, r *http.Request) {
	n, ok := c.muatNasabah(w, r)
	if !ok {
		return
	}
	baru := "nonaktif"
	if n.Status == "nonaktif" {
		baru = "aktif"
	}
	if err := models.SetNasabahStatus(c.DB, n.ID, baru); err != nil {
		serverError(w, "ubah status nasabah", err)
		return
	}
	if baru == "aktif" {
		if err := models.ProsesIuranNasabahSingle(c.DB, n.ID); err != nil {
			log.Println("gagal memproses iuran aktivasi nasabah:", err)
		}
	}
	SetFlash(w, "success", "Status nasabah sekarang: "+baru)
	http.Redirect(w, r, fmt.Sprintf("/admin/nasabah/%d", n.ID), http.StatusSeeOther)
}
