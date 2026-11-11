package controllers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"bank-sampah-berseri/middleware"
	"bank-sampah-berseri/models"
)

type PortalNasabahController struct {
	DB    *sql.DB
	Views *Renderer
}

func NewPortalNasabahController(db *sql.DB, views *Renderer) *PortalNasabahController {
	return &PortalNasabahController{DB: db, Views: views}
}

func (c *PortalNasabahController) page(w http.ResponseWriter, r *http.Request, title, active string, data any) PageData {
	ok, errMsg := PopFlash(w, r)
	return PageData{
		Title:   title,
		Active:  active,
		User:    middleware.CurrentUser(r),
		Success: ok,
		Error:   errMsg,
		Data:    data,
	}
}

// GET /nasabah/profil
func (c *PortalNasabahController) Profil(w http.ResponseWriter, r *http.Request) {
	u := middleware.CurrentUser(r)
	n, err := models.GetNasabahByID(c.DB, u.NasabahID)
	if errors.Is(err, models.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, "profil nasabah", err)
		return
	}

	c.Views.Render(w, http.StatusOK, "nasabah/profil.html",
		c.page(w, r, "Profil Saya", "profil", n))
}

// GET /nasabah/mutasi
func (c *PortalNasabahController) Mutasi(w http.ResponseWriter, r *http.Request) {
	u := middleware.CurrentUser(r)
	n, err := models.GetNasabahByID(c.DB, u.NasabahID)
	if err != nil {
		serverError(w, "mutasi nasabah - info", err)
		return
	}

	mutasiList, err := models.ListSemuaMutasiNasabah(c.DB, u.NasabahID)
	if err != nil {
		serverError(w, "mutasi nasabah - list", err)
		return
	}

	type mutasiPageData struct {
		Nasabah *models.Nasabah
		Items   []models.Mutasi
	}

	c.Views.Render(w, http.StatusOK, "nasabah/mutasi.html",
		c.page(w, r, "Buku Tabungan Saya", "mutasi", mutasiPageData{
			Nasabah: n,
			Items:   mutasiList,
		}))
}

// GET /nasabah/setoran
func (c *PortalNasabahController) Setoran(w http.ResponseWriter, r *http.Request) {
	u := middleware.CurrentUser(r)
	items, err := models.ListSetoranByNasabahID(c.DB, u.NasabahID)
	if err != nil {
		serverError(w, "list setoran nasabah", err)
		return
	}

	c.Views.Render(w, http.StatusOK, "nasabah/setoran_list.html",
		c.page(w, r, "Riwayat Setoran Sampah", "setoran", items))
}

// GET /nasabah/setoran/{id}
func (c *PortalNasabahController) SetoranDetail(w http.ResponseWriter, r *http.Request) {
	u := middleware.CurrentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}

	setoran, err := models.GetSetoranByID(c.DB, id)
	if errors.Is(err, models.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, "detail setoran nasabah", err)
		return
	}

	// Proteksi IDOR ketat: pastikan transaksi setoran milik nasabah yang sedang login
	if setoran.NasabahID != u.NasabahID {
		http.NotFound(w, r)
		return
	}

	c.Views.Render(w, http.StatusOK, "nasabah/setoran_detail.html",
		c.page(w, r, fmt.Sprintf("Bukti Setoran #STR-%05d", setoran.ID), "setoran", setoran))
}

// GET /nasabah/katalog
func (c *PortalNasabahController) Katalog(w http.ResponseWriter, r *http.Request) {
	items, err := models.ListJenisSampah(c.DB)
	if err != nil {
		serverError(w, "katalog sampah nasabah", err)
		return
	}

	c.Views.Render(w, http.StatusOK, "nasabah/katalog.html",
		c.page(w, r, "Katalog Harga Beli Sampah", "katalog", items))
}

