package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"bank-sampah-berseri/config"
	"bank-sampah-berseri/models"
)

func main() {
	username := flag.String("u", "", "username")
	password := flag.String("p", "", "password (8-72 karakter)")
	role := flag.String("role", "admin", "admin atau nasabah")
	nama := flag.String("nama", "", "nama lengkap (wajib untuk role nasabah)")
	kode := flag.String("kode", "", "kode nasabah (wajib untuk role nasabah)")
	flag.Parse()

	*username = strings.TrimSpace(*username)
	if *username == "" || len(*password) < 8 || len(*password) > 72 {
		fmt.Println("Pemakaian: go run ./cmd/buatadmin -u USERNAME -p PASSWORD [-role admin|nasabah -nama NAMA -kode KODE]")
		fmt.Println("Password harus 8-72 karakter.")
		os.Exit(1)
	}
	if *role != "admin" && *role != "nasabah" {
		log.Fatal("role harus 'admin' atau 'nasabah'")
	}
	if *role == "nasabah" && (*nama == "" || *kode == "") {
		log.Fatal("role nasabah membutuhkan -nama dan -kode")
	}

	cfg, err := config.LoadDBConfig("config.json")
	if err != nil {
		log.Fatal(err)
	}
	db, err := config.ConnectDB(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	// Dua tabel terlibat, jadi pakai transaksi.
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback() // tidak berefek jika Commit sudah berhasil

	userID, err := models.CreateUser(tx, *username, string(hash), *role)
	if err != nil {
		log.Fatal("gagal membuat user (username mungkin sudah dipakai): ", err)
	}
	if *role == "nasabah" {
		_, err = tx.Exec(
			"INSERT INTO nasabah (user_id, kode_nasabah, nama, status) VALUES (?, ?, ?, 'aktif')",
			userID, *kode, *nama,
		)
		if err != nil {
			log.Fatal("gagal membuat nasabah (kode mungkin sudah dipakai): ", err)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Berhasil membuat akun '%s' (role: %s)\n", *username, *role)
}
