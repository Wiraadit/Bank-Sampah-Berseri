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

type SaldoController struct {
	DB    *sql.DB
	Views *Renderer
}

func NewSaldoController(db *sql.DB, views *Renderer) *SaldoController {
	return &SaldoController{DB: db, Views: views}
}

type saldoIndexData struct {
	NasabahList   []models.Nasabah
	MutasiTerbaru []models.MutasiGlobal
	TotalSaldo    float64
	TotalTunggakan float64
	Q             string
}

type saldoTarikFormData struct {
	NasabahList       []models.Nasabah
	SelectedNasabahID int64
}

type saldoMutasiData struct {
	Nasabah *models.Nasabah
	Mutasi  []models.Mutasi
}

func (c *SaldoController) page(w http.ResponseWriter, r *http.Request, title string, data any) PageData {
	ok, errMsg := PopFlash(w, r)
	return PageData{
		Title:   title,
		Active:  "saldo",
		User:    middleware.CurrentUser(r),
		Success: ok,
		Error:   errMsg,
		Data:    data,
	}
}

// GET /admin/saldo
func (c *SaldoController) Index(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	nasabahList, err := models.ListNasabah(c.DB, q)
	if err != nil {
		serverError(w, "list nasabah saldo", err)
		return
	}

	mutasiList, err := models.ListMutasiGlobal(c.DB, 20)
	if err != nil {
		serverError(w, "list mutasi global", err)
		return
	}

	var totSaldo, totTunggakan float64
	for _, n := range nasabahList {
		totSaldo += n.Saldo
		totTunggakan += n.Tunggakan
	}

	c.Views.Render(w, http.StatusOK, "admin/saldo_index.html",
		c.page(w, r, "Keuangan & Saldo Nasabah", saldoIndexData{
			NasabahList:    nasabahList,
			MutasiTerbaru:  mutasiList,
			TotalSaldo:     totSaldo,
			TotalTunggakan: totTunggakan,
			Q:              q,
		}))
}

// GET /admin/saldo/tarik
func (c *SaldoController) CreateTarik(w http.ResponseWriter, r *http.Request) {
	nasabahList, err := models.ListNasabah(c.DB, "")
	if err != nil {
		serverError(w, "ambil list nasabah tarik", err)
		return
	}

	var aktif []models.Nasabah
	for _, n := range nasabahList {
		if n.Status == "aktif" {
			aktif = append(aktif, n)
		}
	}

	selectedID, _ := strconv.ParseInt(r.URL.Query().Get("nasabah_id"), 10, 64)

	c.Views.Render(w, http.StatusOK, "admin/saldo_tarik.html",
		c.page(w, r, "Penarikan Saldo Nasabah", saldoTarikFormData{
			NasabahList:       aktif,
			SelectedNasabahID: selectedID,
		}))
}

// POST /admin/saldo/tarik
func (c *SaldoController) StoreTarik(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form tidak valid", http.StatusBadRequest)
		return
	}

	nasabahID, err := strconv.ParseInt(r.PostFormValue("nasabah_id"), 10, 64)
	if err != nil || nasabahID <= 0 {
		SetFlash(w, "error", "Silakan pilih nasabah yang valid")
		http.Redirect(w, r, "/admin/saldo/tarik", http.StatusSeeOther)
		return
	}

	nomStr := strings.ReplaceAll(strings.TrimSpace(r.PostFormValue("nominal")), ",", ".")
	nomVal, err := strconv.ParseFloat(nomStr, 64)
	if err != nil || nomVal <= 0 {
		SetFlash(w, "error", "Nominal penarikan harus lebih besar dari Rp 0")
		http.Redirect(w, r, fmt.Sprintf("/admin/saldo/tarik?nasabah_id=%d", nasabahID), http.StatusSeeOther)
		return
	}

	saldoAkhir, err := models.TarikSaldo(c.DB, nasabahID, nomVal)
	if err != nil {
		SetFlash(w, "error", err.Error())
		http.Redirect(w, r, fmt.Sprintf("/admin/saldo/tarik?nasabah_id=%d", nasabahID), http.StatusSeeOther)
		return
	}

	SetFlash(w, "success", fmt.Sprintf("Penarikan Rp %.0f berhasil. Sisa saldo nasabah: Rp %.0f", nomVal, saldoAkhir))
	http.Redirect(w, r, fmt.Sprintf("/admin/saldo/mutasi/%d", nasabahID), http.StatusSeeOther)
}

// GET /admin/saldo/mutasi/{id}
func (c *SaldoController) MutasiNasabah(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}

	nasabah, err := models.GetNasabahByID(c.DB, id)
	if errors.Is(err, models.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, "muat nasabah mutasi", err)
		return
	}

	mutasiList, err := models.ListSemuaMutasiNasabah(c.DB, id)
	if err != nil {
		serverError(w, "muat mutasi nasabah", err)
		return
	}

	c.Views.Render(w, http.StatusOK, "admin/saldo_mutasi.html",
		c.page(w, r, fmt.Sprintf("Buku Tabungan - %s (%s)", nasabah.Nama, nasabah.KodeNasabah), saldoMutasiData{
			Nasabah: nasabah,
			Mutasi:  mutasiList,
		}))
}

