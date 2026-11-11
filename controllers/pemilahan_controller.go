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

type PemilahanController struct {
	DB    *sql.DB
	Views *Renderer
}

func NewPemilahanController(db *sql.DB, views *Renderer) *PemilahanController {
	return &PemilahanController{DB: db, Views: views}
}

type pemilahanListData struct {
	Items []models.Pemilahan
}

type pemilahanFormData struct {
	StokPlastikCampur float64
	JenisList         []models.JenisSampah
}

type stokPageData struct {
	Items       []models.StokItem
	TotalBerat  float64
	TotalAsetBeli float64
}

func (c *PemilahanController) page(w http.ResponseWriter, r *http.Request, title, active string, data any) PageData {
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

// GET /admin/pemilahan
func (c *PemilahanController) Index(w http.ResponseWriter, r *http.Request) {
	items, err := models.ListPemilahan(c.DB)
	if err != nil {
		serverError(w, "list pemilahan", err)
		return
	}
	c.Views.Render(w, http.StatusOK, "admin/pemilahan_list.html",
		c.page(w, r, "Pemilahan Sampah", "pemilahan", pemilahanListData{Items: items}))
}

// GET /admin/pemilahan/tambah
func (c *PemilahanController) Create(w http.ResponseWriter, r *http.Request) {
	_, stokPlastik, err := models.GetInfoPlastikCampur(c.DB)
	if err != nil {
		serverError(w, "ambil stok plastik campur", err)
		return
	}

	jenisList, err := models.ListJenisSampah(c.DB)
	if err != nil {
		serverError(w, "ambil master sampah", err)
		return
	}

	// Filter jenis sampah agar "Plastik Campur" tidak muncul di daftar hasil pemilahan
	var hasilJenis []models.JenisSampah
	for _, j := range jenisList {
		if j.NamaJenis != models.NamaPlastikCampur {
			hasilJenis = append(hasilJenis, j)
		}
	}

	c.Views.Render(w, http.StatusOK, "admin/pemilahan_form.html",
		c.page(w, r, "Catat Pemilahan Sampah Campur", "pemilahan", pemilahanFormData{
			StokPlastikCampur: stokPlastik,
			JenisList:         hasilJenis,
		}))
}

// POST /admin/pemilahan
func (c *PemilahanController) Store(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form tidak valid", http.StatusBadRequest)
		return
	}

	beratAwalStr := strings.ReplaceAll(strings.TrimSpace(r.PostFormValue("berat_awal")), ",", ".")
	beratAwal, err := strconv.ParseFloat(beratAwalStr, 64)
	if err != nil || beratAwal <= 0 {
		SetFlash(w, "error", "Berat awal harus berupa angka lebih besar dari 0 kg")
		http.Redirect(w, r, "/admin/pemilahan/tambah", http.StatusSeeOther)
		return
	}

	jenisIDs := r.PostForm["jenis_id[]"]
	berats := r.PostForm["berat_hasil[]"]

	if len(jenisIDs) == 0 || len(jenisIDs) != len(berats) {
		SetFlash(w, "error", "Minimal satu hasil pemilahan harus diisi")
		http.Redirect(w, r, "/admin/pemilahan/tambah", http.StatusSeeOther)
		return
	}

	var items []models.ItemHasilPemilahanInput
	for i := range jenisIDs {
		jid, err1 := strconv.ParseInt(jenisIDs[i], 10, 64)
		bStr := strings.ReplaceAll(strings.TrimSpace(berats[i]), ",", ".")
		bVal, err2 := strconv.ParseFloat(bStr, 64)

		if err1 != nil || err2 != nil {
			SetFlash(w, "error", "Input jenis atau berat hasil tidak valid")
			http.Redirect(w, r, "/admin/pemilahan/tambah", http.StatusSeeOther)
			return
		}

		if bVal > 0 {
			items = append(items, models.ItemHasilPemilahanInput{
				JenisSampahID: jid,
				BeratHasil:    bVal,
			})
		}
	}

	user := middleware.CurrentUser(r)
	pemilahanID, err := models.CreatePemilahan(c.DB, user.UserID, beratAwal, items)
	if err != nil {
		SetFlash(w, "error", err.Error())
		http.Redirect(w, r, "/admin/pemilahan/tambah", http.StatusSeeOther)
		return
	}

	SetFlash(w, "success", fmt.Sprintf("Pemilahan berhasil dicatat (#PML-%05d). Stok gudang telah diperbarui.", pemilahanID))
	http.Redirect(w, r, fmt.Sprintf("/admin/pemilahan/%d", pemilahanID), http.StatusSeeOther)
}

// GET /admin/pemilahan/{id}
func (c *PemilahanController) Show(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}

	pemilahan, err := models.GetPemilahanByID(c.DB, id)
	if errors.Is(err, models.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, "muat detail pemilahan", err)
		return
	}

	c.Views.Render(w, http.StatusOK, "admin/pemilahan_detail.html",
		c.page(w, r, fmt.Sprintf("Detail Pemilahan #%05d", pemilahan.ID), "pemilahan", pemilahan))
}

// GET /admin/stok
func (c *PemilahanController) Stok(w http.ResponseWriter, r *http.Request) {
	stokList, err := models.ListStok(c.DB)
	if err != nil {
		serverError(w, "list stok", err)
		return
	}

	var totalBerat float64
	var totalAset float64
	for _, s := range stokList {
		if s.Stok > 0 {
			totalBerat += s.Stok
			totalAset += s.Stok * s.HargaBeli
		}
	}

	c.Views.Render(w, http.StatusOK, "admin/stok_list.html",
		c.page(w, r, "Stok Sampah Gudang", "stok", stokPageData{
			Items:         stokList,
			TotalBerat:    totalBerat,
			TotalAsetBeli: totalAset,
		}))
}

