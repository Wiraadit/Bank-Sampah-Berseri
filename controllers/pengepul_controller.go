package controllers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"bank-sampah-berseri/middleware"
	"bank-sampah-berseri/models"
)

type PengepulController struct {
	DB    *sql.DB
	Views *Renderer
}

func NewPengepulController(db *sql.DB, views *Renderer) *PengepulController {
	return &PengepulController{DB: db, Views: views}
}

type pengepulForm struct {
	IsEdit    bool
	ID        int64
	Nama      string
	NoTelepon string
	Alamat    string
}

func (f *pengepulForm) bersihkan() {
	f.Nama = strings.TrimSpace(f.Nama)
	f.Alamat = strings.TrimSpace(f.Alamat)
	f.NoTelepon = teleponFix.Replace(strings.TrimSpace(f.NoTelepon))
}

func (f *pengepulForm) validasi() string {
	if f.Nama == "" {
		return "Nama pengepul wajib diisi"
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
	return ""
}

type pengepulDetail struct {
	P *models.Pengepul
	R *models.PengepulRingkasan
}

func (c *PengepulController) page(w http.ResponseWriter, r *http.Request, title string, data any) PageData {
	ok, errMsg := PopFlash(w, r)
	return PageData{
		Title: title, Active: "pengepul", User: middleware.CurrentUser(r),
		Success: ok, Error: errMsg, Data: data,
	}
}

func (c *PengepulController) muat(w http.ResponseWriter, r *http.Request) (*models.Pengepul, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return nil, false
	}
	p, err := models.GetPengepulByID(c.DB, id)
	if errors.Is(err, models.ErrNotFound) {
		http.NotFound(w, r)
		return nil, false
	}
	if err != nil {
		serverError(w, "muat pengepul", err)
		return nil, false
	}
	return p, true
}

// GET /admin/pengepul
func (c *PengepulController) Index(w http.ResponseWriter, r *http.Request) {
	items, err := models.ListPengepul(c.DB)
	if err != nil {
		serverError(w, "list pengepul", err)
		return
	}
	c.Views.Render(w, http.StatusOK, "admin/pengepul_list.html", c.page(w, r, "Data Pengepul", items))
}

// GET /admin/pengepul/tambah
func (c *PengepulController) Create(w http.ResponseWriter, r *http.Request) {
	c.Views.Render(w, http.StatusOK, "admin/pengepul_form.html",
		c.page(w, r, "Tambah Pengepul", pengepulForm{}))
}

// POST /admin/pengepul
func (c *PengepulController) Store(w http.ResponseWriter, r *http.Request) {
	f := pengepulForm{
		Nama:      r.PostFormValue("nama"),
		NoTelepon: r.PostFormValue("no_telepon"),
		Alamat:    r.PostFormValue("alamat"),
	}
	f.bersihkan()

	if msg := f.validasi(); msg != "" {
		p := c.page(w, r, "Tambah Pengepul", f)
		p.Error = msg
		c.Views.Render(w, http.StatusBadRequest, "admin/pengepul_form.html", p)
		return
	}
	id, err := models.CreatePengepul(c.DB, f.Nama, f.NoTelepon, f.Alamat)
	if err != nil {
		serverError(w, "simpan pengepul", err)
		return
	}
	SetFlash(w, "success", "Pengepul berhasil ditambahkan")
	http.Redirect(w, r, fmt.Sprintf("/admin/pengepul/%d", id), http.StatusSeeOther)
}

// GET /admin/pengepul/{id}
func (c *PengepulController) Show(w http.ResponseWriter, r *http.Request) {
	p, ok := c.muat(w, r)
	if !ok {
		return
	}
	ring, err := models.GetPengepulRingkasan(c.DB, p.ID)
	if err != nil {
		serverError(w, "ringkasan pengepul", err)
		return
	}
	c.Views.Render(w, http.StatusOK, "admin/pengepul_detail.html",
		c.page(w, r, "Detail Pengepul", pengepulDetail{P: p, R: ring}))
}

// GET /admin/pengepul/{id}/edit
func (c *PengepulController) Edit(w http.ResponseWriter, r *http.Request) {
	p, ok := c.muat(w, r)
	if !ok {
		return
	}
	f := pengepulForm{IsEdit: true, ID: p.ID, Nama: p.Nama, NoTelepon: p.NoTelepon, Alamat: p.Alamat}
	c.Views.Render(w, http.StatusOK, "admin/pengepul_form.html", c.page(w, r, "Edit Pengepul", f))
}

// POST /admin/pengepul/{id}
func (c *PengepulController) Update(w http.ResponseWriter, r *http.Request) {
	p, ok := c.muat(w, r)
	if !ok {
		return
	}
	f := pengepulForm{
		IsEdit: true, ID: p.ID,
		Nama:      r.PostFormValue("nama"),
		NoTelepon: r.PostFormValue("no_telepon"),
		Alamat:    r.PostFormValue("alamat"),
	}
	f.bersihkan()

	if msg := f.validasi(); msg != "" {
		pg := c.page(w, r, "Edit Pengepul", f)
		pg.Error = msg
		c.Views.Render(w, http.StatusBadRequest, "admin/pengepul_form.html", pg)
		return
	}
	if err := models.UpdatePengepul(c.DB, p.ID, f.Nama, f.NoTelepon, f.Alamat); err != nil {
		serverError(w, "update pengepul", err)
		return
	}
	SetFlash(w, "success", "Data pengepul berhasil diperbarui")
	http.Redirect(w, r, fmt.Sprintf("/admin/pengepul/%d", p.ID), http.StatusSeeOther)
}
