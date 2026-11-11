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

type PenjualanController struct {
	DB    *sql.DB
	Views *Renderer
}

func NewPenjualanController(db *sql.DB, views *Renderer) *PenjualanController {
	return &PenjualanController{DB: db, Views: views}
}

type penjualanListData struct {
	Items []models.Penjualan
	Q     string
}

type penjualanFormData struct {
	PengepulList      []models.Pengepul
	StokList          []models.StokItem
	SelectedPengepulID int64
}

func (c *PenjualanController) page(w http.ResponseWriter, r *http.Request, title string, data any) PageData {
	ok, errMsg := PopFlash(w, r)
	return PageData{
		Title:   title,
		Active:  "penjualan",
		User:    middleware.CurrentUser(r),
		Success: ok,
		Error:   errMsg,
		Data:    data,
	}
}

// GET /admin/penjualan
func (c *PenjualanController) Index(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	items, err := models.ListPenjualan(c.DB, q)
	if err != nil {
		serverError(w, "list penjualan", err)
		return
	}
	c.Views.Render(w, http.StatusOK, "admin/penjualan_list.html",
		c.page(w, r, "Penjualan Sampah", penjualanListData{Items: items, Q: q}))
}

// GET /admin/penjualan/tambah
func (c *PenjualanController) Create(w http.ResponseWriter, r *http.Request) {
	pengepulList, err := models.ListPengepul(c.DB)
	if err != nil {
		serverError(w, "ambil data pengepul", err)
		return
	}

	stokList, err := models.ListStok(c.DB)
	if err != nil {
		serverError(w, "ambil stok sampah", err)
		return
	}

	selectedID, _ := strconv.ParseInt(r.URL.Query().Get("pengepul_id"), 10, 64)

	c.Views.Render(w, http.StatusOK, "admin/penjualan_form.html",
		c.page(w, r, "Catat Penjualan ke Pengepul", penjualanFormData{
			PengepulList:       pengepulList,
			StokList:           stokList,
			SelectedPengepulID: selectedID,
		}))
}

// POST /admin/penjualan
func (c *PenjualanController) Store(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form tidak valid", http.StatusBadRequest)
		return
	}

	pengepulID, err := strconv.ParseInt(r.PostFormValue("pengepul_id"), 10, 64)
	if err != nil || pengepulID <= 0 {
		SetFlash(w, "error", "Silakan pilih pengepul tujuan")
		http.Redirect(w, r, "/admin/penjualan/tambah", http.StatusSeeOther)
		return
	}

	jenisIDs := r.PostForm["jenis_id[]"]
	beratStoks := r.PostForm["berat_stok[]"]
	beratRiils := r.PostForm["berat_riil[]"]
	hargaJuals := r.PostForm["harga_jual[]"]
	hpps := r.PostForm["hpp[]"]

	if len(jenisIDs) == 0 || len(jenisIDs) != len(beratStoks) {
		SetFlash(w, "error", "Minimal satu jenis sampah harus diisi dalam penjualan")
		http.Redirect(w, r, fmt.Sprintf("/admin/penjualan/tambah?pengepul_id=%d", pengepulID), http.StatusSeeOther)
		return
	}

	var items []models.ItemPenjualanInput
	for i := range jenisIDs {
		jid, err1 := strconv.ParseInt(jenisIDs[i], 10, 64)

		bStokStr := strings.ReplaceAll(strings.TrimSpace(beratStoks[i]), ",", ".")
		bStokVal, err2 := strconv.ParseFloat(bStokStr, 64)

		bRiilStr := strings.ReplaceAll(strings.TrimSpace(beratRiils[i]), ",", ".")
		bRiilVal, err3 := strconv.ParseFloat(bRiilStr, 64)

		hJualStr := strings.ReplaceAll(strings.TrimSpace(hargaJuals[i]), ",", ".")
		hJualVal, err4 := strconv.ParseFloat(hJualStr, 64)

		hppStr := strings.ReplaceAll(strings.TrimSpace(hpps[i]), ",", ".")
		hppVal, err5 := strconv.ParseFloat(hppStr, 64)

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
			SetFlash(w, "error", "Pastikan semua input angka (berat, harga, HPP) diisi dengan benar")
			http.Redirect(w, r, fmt.Sprintf("/admin/penjualan/tambah?pengepul_id=%d", pengepulID), http.StatusSeeOther)
			return
		}

		if bStokVal <= 0 || bRiilVal <= 0 || hJualVal <= 0 {
			SetFlash(w, "error", "Berat stok keluar, berat riil, dan harga jual harus lebih besar dari 0")
			http.Redirect(w, r, fmt.Sprintf("/admin/penjualan/tambah?pengepul_id=%d", pengepulID), http.StatusSeeOther)
			return
		}

		items = append(items, models.ItemPenjualanInput{
			JenisSampahID: jid,
			BeratStok:     bStokVal,
			BeratRiil:     bRiilVal,
			HargaJual:     hJualVal,
			HPP:           hppVal,
		})
	}

	user := middleware.CurrentUser(r)
	penjualanID, err := models.CreatePenjualan(c.DB, user.UserID, pengepulID, items)
	if err != nil {
		SetFlash(w, "error", err.Error())
		http.Redirect(w, r, fmt.Sprintf("/admin/penjualan/tambah?pengepul_id=%d", pengepulID), http.StatusSeeOther)
		return
	}

	SetFlash(w, "success", fmt.Sprintf("Penjualan sampah berhasil dicatat (#PJL-%05d). Stok gudang telah diperbarui.", penjualanID))
	http.Redirect(w, r, fmt.Sprintf("/admin/penjualan/%d", penjualanID), http.StatusSeeOther)
}

// GET /admin/penjualan/{id}
func (c *PenjualanController) Show(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}

	penjualan, err := models.GetPenjualanByID(c.DB, id)
	if errors.Is(err, models.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, "muat detail penjualan", err)
		return
	}

	c.Views.Render(w, http.StatusOK, "admin/penjualan_detail.html",
		c.page(w, r, fmt.Sprintf("Bukti Penjualan #PJL-%05d", penjualan.ID), penjualan))
}
