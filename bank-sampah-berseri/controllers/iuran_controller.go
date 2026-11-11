package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"bank-sampah-berseri/middleware"
	"bank-sampah-berseri/models"
)

type IuranController struct {
	DB    *sql.DB
	Views *Renderer
}

func NewIuranController(db *sql.DB, views *Renderer) *IuranController {
	return &IuranController{DB: db, Views: views}
}

type iuranIndexData struct {
	Items     []models.IuranBulanan
	Ringkasan *models.RingkasanIuranData
	Bulan     int
	Tahun     int
	Status    string
	BulanNama string
}

func getNamaBulan(b int) string {
	if b >= 1 && b <= 12 {
		return namaBulan[b-1]
	}
	return ""
}

func (c *IuranController) page(w http.ResponseWriter, r *http.Request, title string, data any) PageData {
	ok, errMsg := PopFlash(w, r)
	return PageData{
		Title:   title,
		Active:  "iuran",
		User:    middleware.CurrentUser(r),
		Success: ok,
		Error:   errMsg,
		Data:    data,
	}
}

// GET /admin/iuran
func (c *IuranController) Index(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	bulan := int(now.Month())
	tahun := now.Year()

	if b, err := strconv.Atoi(r.URL.Query().Get("bulan")); err == nil && b >= 1 && b <= 12 {
		bulan = b
	}
	if t, err := strconv.Atoi(r.URL.Query().Get("tahun")); err == nil && t >= 2020 {
		tahun = t
	}
	status := r.URL.Query().Get("status")

	items, err := models.ListIuran(c.DB, bulan, tahun, status)
	if err != nil {
		serverError(w, "list iuran", err)
		return
	}

	ringkasan, err := models.GetRingkasanIuranPeriode(c.DB, bulan, tahun)
	if err != nil {
		serverError(w, "ringkasan iuran", err)
		return
	}

	c.Views.Render(w, http.StatusOK, "admin/iuran_index.html",
		c.page(w, r, "Iuran Bulanan", iuranIndexData{
			Items:     items,
			Ringkasan: ringkasan,
			Bulan:     bulan,
			Tahun:     tahun,
			Status:    status,
			BulanNama: getNamaBulan(bulan),
		}))
}

// POST /admin/iuran/proses
func (c *IuranController) Proses(w http.ResponseWriter, r *http.Request) {
	bulan, err1 := strconv.Atoi(r.PostFormValue("bulan"))
	tahun, err2 := strconv.Atoi(r.PostFormValue("tahun"))

	if err1 != nil || err2 != nil || bulan < 1 || bulan > 12 || tahun < 2020 {
		SetFlash(w, "error", "Periode bulan atau tahun tidak valid")
		http.Redirect(w, r, "/admin/iuran", http.StatusSeeOther)
		return
	}

	diproses, sudahAda, err := models.ProsesIuranPeriode(c.DB, bulan, tahun)
	if err != nil {
		SetFlash(w, "error", "Gagal memproses iuran: "+err.Error())
		http.Redirect(w, r, fmt.Sprintf("/admin/iuran?bulan=%d&tahun=%d", bulan, tahun), http.StatusSeeOther)
		return
	}

	msg := fmt.Sprintf("Iuran periode %s %d selesai diproses. %d nasabah baru ditagihkan/dipotong, %d nasabah sudah tercatat sebelumnya.",
		getNamaBulan(bulan), tahun, diproses, sudahAda)
	SetFlash(w, "success", msg)
	http.Redirect(w, r, fmt.Sprintf("/admin/iuran?bulan=%d&tahun=%d", bulan, tahun), http.StatusSeeOther)
}
