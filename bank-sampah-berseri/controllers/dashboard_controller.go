package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"bank-sampah-berseri/middleware"
	"bank-sampah-berseri/models"
)

type DashboardController struct {
	DB    *sql.DB
	Views *Renderer
}

func NewDashboardController(db *sql.DB, views *Renderer) *DashboardController {
	return &DashboardController{DB: db, Views: views}
}

var namaBulan = [...]string{
	"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

type adminDashData struct {
	Stats       *models.DashboardStats
	Bulan       string
	ChartLabels string // JSON, dibaca oleh static/js/dashboard.js
	ChartValues string // JSON
	AdaSetoran  bool
}

// GET /admin/dashboard
func (c *DashboardController) Admin(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	awalBulan := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	stats, err := models.GetDashboardStats(c.DB, awalBulan, awalBulan.AddDate(0, 1, 0))
	if err != nil {
		serverError(w, "statistik dashboard", err)
		return
	}

	// Grafik: 7 hari terakhir (termasuk hari ini).
	hariIni := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dari := hariIni.AddDate(0, 0, -6)
	harian, err := models.GetSetoranHarian(c.DB, dari, hariIni.AddDate(0, 0, 1))
	if err != nil {
		serverError(w, "setoran harian", err)
		return
	}

	labels := make([]string, 0, 7)
	values := make([]float64, 0, 7)
	ada := false
	for i := 0; i < 7; i++ {
		d := dari.AddDate(0, 0, i)
		v := harian[d.Format("2006-01-02")]
		if v > 0 {
			ada = true
		}
		labels = append(labels, d.Format("02/01"))
		values = append(values, v)
	}
	lb, _ := json.Marshal(labels)
	vb, _ := json.Marshal(values)

	c.Views.Render(w, http.StatusOK, "admin/dashboard.html", PageData{
		Title: "Dashboard Admin", Active: "dashboard", User: middleware.CurrentUser(r),
		Data: adminDashData{
			Stats:       stats,
			Bulan:       namaBulan[now.Month()-1] + " " + now.Format("2006"),
			ChartLabels: string(lb),
			ChartValues: string(vb),
			AdaSetoran:  ada,
		},
	})
}

type nasabahDashData struct {
	Nasabah   *models.Nasabah
	Ringkasan *models.RingkasanSetoran
	Mutasi    []models.Mutasi
}

// GET /nasabah/dashboard
// Semua data diambil dengan NasabahID dari session, bukan dari parameter URL.
func (c *DashboardController) Nasabah(w http.ResponseWriter, r *http.Request) {
	u := middleware.CurrentUser(r)

	n, err := models.GetNasabahByID(c.DB, u.NasabahID)
	if errors.Is(err, models.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, "dashboard nasabah", err)
		return
	}
	ring, err := models.GetRingkasanSetoranNasabah(c.DB, n.ID)
	if err != nil {
		serverError(w, "ringkasan setoran", err)
		return
	}
	mutasi, err := models.ListMutasiTerakhir(c.DB, n.ID, 5)
	if err != nil {
		serverError(w, "mutasi terakhir", err)
		return
	}

	c.Views.Render(w, http.StatusOK, "nasabah/dashboard.html", PageData{
		Title: "Dashboard Nasabah", Active: "dashboard", User: u,
		Data: nasabahDashData{Nasabah: n, Ringkasan: ring, Mutasi: mutasi},
	})
}
