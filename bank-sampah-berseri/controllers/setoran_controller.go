package controllers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"bank-sampah-berseri/middleware"
	"bank-sampah-berseri/models"
)

type SetoranController struct {
	DB    *sql.DB
	Views *Renderer
}

func NewSetoranController(db *sql.DB, views *Renderer) *SetoranController {
	return &SetoranController{DB: db, Views: views}
}

type setoranListData struct {
	Items []models.Setoran
	Q     string
}

type setoranFormData struct {
	NasabahList       []models.Nasabah
	JenisList         []models.JenisSampah
	SelectedNasabahID int64
}

func (c *SetoranController) page(w http.ResponseWriter, r *http.Request, title string, data any) PageData {
	ok, errMsg := PopFlash(w, r)
	return PageData{
		Title:   title,
		Active:  "setoran",
		User:    middleware.CurrentUser(r),
		Success: ok,
		Error:   errMsg,
		Data:    data,
	}
}

// GET /admin/setoran
func (c *SetoranController) Index(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	items, err := models.ListSetoran(c.DB, q)
	if err != nil {
		serverError(w, "list setoran", err)
		return
	}
	c.Views.Render(w, http.StatusOK, "admin/setoran_list.html",
		c.page(w, r, "Setoran Sampah", setoranListData{Items: items, Q: q}))
}

// GET /admin/setoran/tambah
func (c *SetoranController) Create(w http.ResponseWriter, r *http.Request) {
	nasabahList, err := models.ListNasabah(c.DB, "")
	if err != nil {
		serverError(w, "muat nasabah untuk setoran", err)
		return
	}

	// Filter hanya nasabah aktif
	var aktif []models.Nasabah
	for _, n := range nasabahList {
		if n.Status == "aktif" {
			aktif = append(aktif, n)
		}
	}

	jenisList, err := models.ListJenisSampah(c.DB)
	if err != nil {
		serverError(w, "muat jenis sampah untuk setoran", err)
		return
	}

	selectedID, _ := strconv.ParseInt(r.URL.Query().Get("nasabah_id"), 10, 64)

	c.Views.Render(w, http.StatusOK, "admin/setoran_form.html",
		c.page(w, r, "Catat Setoran Baru", setoranFormData{
			NasabahList:       aktif,
			JenisList:         jenisList,
			SelectedNasabahID: selectedID,
		}))
}

// POST /admin/setoran
func (c *SetoranController) Store(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Data form tidak valid", http.StatusBadRequest)
		return
	}

	nasabahID, err := strconv.ParseInt(r.PostFormValue("nasabah_id"), 10, 64)
	if err != nil || nasabahID <= 0 {
		SetFlash(w, "error", "Silakan pilih nasabah yang valid")
		http.Redirect(w, r, "/admin/setoran/tambah", http.StatusSeeOther)
		return
	}

	jenisIDs := r.PostForm["jenis_id[]"]
	berats := r.PostForm["berat[]"]

	if len(jenisIDs) == 0 || len(jenisIDs) != len(berats) {
		SetFlash(w, "error", "Minimal satu jenis sampah dan berat harus diisi")
		http.Redirect(w, r, fmt.Sprintf("/admin/setoran/tambah?nasabah_id=%d", nasabahID), http.StatusSeeOther)
		return
	}

	var items []models.ItemSetoranInput
	for i := range jenisIDs {
		jid, err1 := strconv.ParseInt(jenisIDs[i], 10, 64)
		beratStr := strings.ReplaceAll(strings.TrimSpace(berats[i]), ",", ".")
		beratVal, err2 := strconv.ParseFloat(beratStr, 64)

		if err1 != nil || err2 != nil || beratVal <= 0 {
			SetFlash(w, "error", "Pastikan jenis sampah dipilih dan berat lebih dari 0 kg")
			http.Redirect(w, r, fmt.Sprintf("/admin/setoran/tambah?nasabah_id=%d", nasabahID), http.StatusSeeOther)
			return
		}

		items = append(items, models.ItemSetoranInput{
			JenisSampahID: jid,
			Berat:         beratVal,
		})
	}

	user := middleware.CurrentUser(r)
	hasil, err := models.CreateSetoran(c.DB, user.UserID, nasabahID, items)
	if err != nil {
		SetFlash(w, "error", err.Error())
		http.Redirect(w, r, fmt.Sprintf("/admin/setoran/tambah?nasabah_id=%d", nasabahID), http.StatusSeeOther)
		return
	}

	msg := "Transaksi setoran berhasil disimpan."
	if hasil.PelunasanTunggakan > 0 {
		msg += fmt.Sprintf(" Terpotong otomatis Rp %.0f untuk pelunasan tunggakan.", hasil.PelunasanTunggakan)
	}
	SetFlash(w, "success", msg)
	http.Redirect(w, r, fmt.Sprintf("/admin/setoran/%d", hasil.SetoranID), http.StatusSeeOther)
}

// GET /admin/setoran/{id}
func (c *SetoranController) Show(w http.ResponseWriter, r *http.Request) {
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
		serverError(w, "muat detail setoran", err)
		return
	}

	c.Views.Render(w, http.StatusOK, "admin/setoran_detail.html",
		c.page(w, r, fmt.Sprintf("Bukti Setoran #%05d", setoran.ID), setoran))
}

