package main

import (
	"log"
	"net/http"
	"time"

	"bank-sampah-berseri/config"
	"bank-sampah-berseri/controllers"
	"bank-sampah-berseri/models"
	"bank-sampah-berseri/routes"
)

func main() {
	cfg, err := config.LoadDBConfig("config.json")
	if err != nil {
		log.Fatal(err)
	}

	db, err := config.ConnectDB(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Println("Koneksi database berhasil")

	// Catch-up iuran bulanan otomatis (Bab 5.7)
	if err := models.CatchUpIuranOtomatis(db); err != nil {
		log.Println("Peringatan catch-up iuran:", err)
	} else {
		log.Println("Pemeriksaan catch-up iuran bulanan selesai")
	}

	// true = mode pengembangan (template dimuat ulang tiap request)
	views, err := controllers.NewRenderer("templates", true)
	if err != nil {
		log.Fatal("gagal memuat template: ", err)
	}

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           routes.New(db, views),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Println("Server berjalan di http://localhost:8080")
	log.Fatal(srv.ListenAndServe())
}
