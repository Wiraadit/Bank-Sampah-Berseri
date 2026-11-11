package controllers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"bank-sampah-berseri/middleware"
	"bank-sampah-berseri/models"
)

type LaporanController struct {
	DB    *sql.DB
	Views *Renderer
}

func NewLaporanController(db *sql.DB, views *Renderer) *LaporanController {
	return &LaporanController{DB: db, Views: views}
}

func (c *LaporanController) page(w http.ResponseWriter, r *http.Request, title string, data any) PageData {
	ok, errMsg := PopFlash(w, r)
	return PageData{
		Title:   title,
		Active:  "laporan",
		User:    middleware.CurrentUser(r),
		Success: ok,
		Error:   errMsg,
		Data:    data,
	}
}

func parseDateRange(r *http.Request) (time.Time, time.Time, string, string) {
	now := time.Now()
	// Default dari awal bulan ini
	awalBulan := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	akhirHari := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	dariStr := r.URL.Query().Get("dari")
	sampaiStr := r.URL.Query().Get("sampai")

	dari := awalBulan
	sampai := akhirHari

	if t, err := time.ParseInLocation("2006-01-02", dariStr, now.Location()); err == nil {
		dari = t
	} else {
		dariStr = dari.Format("2006-01-02")
	}

	if t, err := time.ParseInLocation("2006-01-02", sampaiStr, now.Location()); err == nil {
		sampai = t
	} else {
		sampaiStr = sampai.Format("2006-01-02")
	}

	return dari, sampai, dariStr, sampaiStr
}

// GET /admin/laporan
func (c *LaporanController) Index(w http.ResponseWriter, r *http.Request) {
	c.Views.Render(w, http.StatusOK, "admin/laporan_index.html",
		c.page(w, r, "Pusat Laporan", nil))
}

// GET /admin/laporan/laba-rugi
func (c *LaporanController) LabaRugi(w http.ResponseWriter, r *http.Request) {
	dari, sampai, dariStr, sampaiStr := parseDateRange(r)

	laporan, err := models.GetLaporanLabaRugi(c.DB, dari, sampai)
	if err != nil {
		serverError(w, "laporan laba rugi", err)
		return
	}

	type labaRugiViewData struct {
		Laporan   *models.LaporanLabaRugi
		DariStr   string
		SampaiStr string
	}

	c.Views.Render(w, http.StatusOK, "admin/laporan_laba_rugi.html",
		c.page(w, r, "Laporan Laba / Rugi Penjualan", labaRugiViewData{
			Laporan:   laporan,
			DariStr:   dariStr,
			SampaiStr: sampaiStr,
		}))
}

// GET /admin/laporan/stok
func (c *LaporanController) Stok(w http.ResponseWriter, r *http.Request) {
	laporan, err := models.GetLaporanStokGudang(c.DB)
	if err != nil {
		serverError(w, "laporan stok", err)
		return
	}

	c.Views.Render(w, http.StatusOK, "admin/laporan_stok.html",
		c.page(w, r, "Laporan Stok Gudang", laporan))
}

// GET /admin/laporan/mutasi
func (c *LaporanController) Mutasi(w http.ResponseWriter, r *http.Request) {
	dari, sampai, dariStr, sampaiStr := parseDateRange(r)
	nasabahID, _ := strconv.ParseInt(r.URL.Query().Get("nasabah_id"), 10, 64)

	nasabahList, err := models.ListNasabah(c.DB, "")
	if err != nil {
		serverError(w, "ambil list nasabah laporan", err)
		return
	}

	laporan, err := models.GetLaporanMutasi(c.DB, nasabahID, dari, sampai)
	if err != nil {
		serverError(w, "laporan mutasi", err)
		return
	}

	type mutasiViewData struct {
		Laporan           *models.LaporanMutasiData
		NasabahList       []models.Nasabah
		SelectedNasabahID int64
		DariStr           string
		SampaiStr         string
	}

	c.Views.Render(w, http.StatusOK, "admin/laporan_mutasi.html",
		c.page(w, r, "Laporan Buku Tabungan & Mutasi", mutasiViewData{
			Laporan:           laporan,
			NasabahList:       nasabahList,
			SelectedNasabahID: nasabahID,
			DariStr:           dariStr,
			SampaiStr:         sampaiStr,
		}))
}

