package routes

import (
	"database/sql"
	"net/http"

	"bank-sampah-berseri/controllers"
	"bank-sampah-berseri/middleware"
)

func New(db *sql.DB, views *controllers.Renderer) http.Handler {
	mux := http.NewServeMux()

	auth := controllers.NewAuthController(db, views)
	dash := controllers.NewDashboardController(db, views)
	nsb := controllers.NewNasabahController(db, views)
	smp := controllers.NewSampahController(db, views)
	pgp := controllers.NewPengepulController(db, views)
	str := controllers.NewSetoranController(db, views)
	pml := controllers.NewPemilahanController(db, views)
	pjl := controllers.NewPenjualanController(db, views)
	sld := controllers.NewSaldoController(db, views)
	iur := controllers.NewIuranController(db, views)
	lap := controllers.NewLaporanController(db, views)
	pn := controllers.NewPortalNasabahController(db, views)

	admin := func(h http.HandlerFunc) http.Handler { return middleware.RequireRole("admin", h) }
	nasabah := func(h http.HandlerFunc) http.Handler { return middleware.RequireRole("nasabah", h) }

	// File statis (CSS, JS, gambar)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Publik
	mux.HandleFunc("GET /{$}", auth.Home)
	mux.HandleFunc("GET /login", auth.ShowLogin)
	mux.HandleFunc("POST /login", auth.Login)

	// Harus login
	mux.Handle("POST /logout", middleware.RequireLogin(http.HandlerFunc(auth.Logout)))

	// Admin
	mux.Handle("GET /admin/dashboard", admin(dash.Admin))

	mux.Handle("GET /admin/nasabah", admin(nsb.Index))
	mux.Handle("GET /admin/nasabah/tambah", admin(nsb.Create))
	mux.Handle("POST /admin/nasabah", admin(nsb.Store))
	mux.Handle("GET /admin/nasabah/{id}", admin(nsb.Show))
	mux.Handle("GET /admin/nasabah/{id}/edit", admin(nsb.Edit))
	mux.Handle("POST /admin/nasabah/{id}", admin(nsb.Update))
	mux.Handle("POST /admin/nasabah/{id}/status", admin(nsb.ToggleStatus))

	mux.Handle("GET /admin/sampah", admin(smp.Index))
	mux.Handle("GET /admin/sampah/tambah", admin(smp.Create))
	mux.Handle("POST /admin/sampah", admin(smp.Store))
	mux.Handle("GET /admin/sampah/{id}/edit", admin(smp.Edit))
	mux.Handle("POST /admin/sampah/{id}", admin(smp.Update))

	mux.Handle("GET /admin/pengepul", admin(pgp.Index))
	mux.Handle("GET /admin/pengepul/tambah", admin(pgp.Create))
	mux.Handle("POST /admin/pengepul", admin(pgp.Store))
	mux.Handle("GET /admin/pengepul/{id}", admin(pgp.Show))
	mux.Handle("GET /admin/pengepul/{id}/edit", admin(pgp.Edit))
	mux.Handle("POST /admin/pengepul/{id}", admin(pgp.Update))

	mux.Handle("GET /admin/setoran", admin(str.Index))
	mux.Handle("GET /admin/setoran/tambah", admin(str.Create))
	mux.Handle("POST /admin/setoran", admin(str.Store))
	mux.Handle("GET /admin/setoran/{id}", admin(str.Show))

	mux.Handle("GET /admin/pemilahan", admin(pml.Index))
	mux.Handle("GET /admin/pemilahan/tambah", admin(pml.Create))
	mux.Handle("POST /admin/pemilahan", admin(pml.Store))
	mux.Handle("GET /admin/pemilahan/{id}", admin(pml.Show))

	mux.Handle("GET /admin/stok", admin(pml.Stok))

	mux.Handle("GET /admin/penjualan", admin(pjl.Index))
	mux.Handle("GET /admin/penjualan/tambah", admin(pjl.Create))
	mux.Handle("POST /admin/penjualan", admin(pjl.Store))
	mux.Handle("GET /admin/penjualan/{id}", admin(pjl.Show))

	mux.Handle("GET /admin/saldo", admin(sld.Index))
	mux.Handle("GET /admin/saldo/tarik", admin(sld.CreateTarik))
	mux.Handle("POST /admin/saldo/tarik", admin(sld.StoreTarik))
	mux.Handle("GET /admin/saldo/mutasi/{id}", admin(sld.MutasiNasabah))

	mux.Handle("GET /admin/iuran", admin(iur.Index))
	mux.Handle("POST /admin/iuran/proses", admin(iur.Proses))

	mux.Handle("GET /admin/laporan", admin(lap.Index))
	mux.Handle("GET /admin/laporan/laba-rugi", admin(lap.LabaRugi))
	mux.Handle("GET /admin/laporan/stok", admin(lap.Stok))
	mux.Handle("GET /admin/laporan/mutasi", admin(lap.Mutasi))

	// Nasabah
	mux.Handle("GET /nasabah/dashboard", nasabah(dash.Nasabah))
	mux.Handle("GET /nasabah/profil", nasabah(pn.Profil))
	mux.Handle("GET /nasabah/mutasi", nasabah(pn.Mutasi))
	mux.Handle("GET /nasabah/setoran", nasabah(pn.Setoran))
	mux.Handle("GET /nasabah/setoran/{id}", nasabah(pn.SetoranDetail))
	mux.Handle("GET /nasabah/katalog", nasabah(pn.Katalog))

	return mux
}
