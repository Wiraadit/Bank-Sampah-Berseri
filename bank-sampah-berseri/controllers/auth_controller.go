package controllers

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"bank-sampah-berseri/middleware"
	"bank-sampah-berseri/models"
)

type AuthController struct {
	DB    *sql.DB
	Views *Renderer
}

func NewAuthController(db *sql.DB, views *Renderer) *AuthController {
	return &AuthController{DB: db, Views: views}
}

// Hash palsu: dipakai agar waktu respons tetap mirip walaupun username tidak ada,
// sehingga penyerang tidak bisa menebak username dari kecepatan respons.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)

// DashboardPath menentukan tujuan setelah login berdasarkan role.
func DashboardPath(role string) string {
	if role == "admin" {
		return "/admin/dashboard"
	}
	return "/nasabah/dashboard"
}

// Home: alamat "/" mengarahkan ke dashboard (jika sudah login) atau ke login.
func (c *AuthController) Home(w http.ResponseWriter, r *http.Request) {
	if s := middleware.GetSession(r); s != nil {
		http.Redirect(w, r, DashboardPath(s.Role), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (c *AuthController) ShowLogin(w http.ResponseWriter, r *http.Request) {
	if s := middleware.GetSession(r); s != nil {
		http.Redirect(w, r, DashboardPath(s.Role), http.StatusSeeOther)
		return
	}
	c.Views.Render(w, http.StatusOK, "auth/login.html", PageData{Title: "Login"})
}

func (c *AuthController) loginError(w http.ResponseWriter, status int, msg string) {
	c.Views.Render(w, status, "auth/login.html", PageData{Title: "Login", Error: msg})
}

func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		c.loginError(w, http.StatusBadRequest, "Permintaan tidak valid")
		return
	}
	username := strings.TrimSpace(r.PostFormValue("username"))
	password := r.PostFormValue("password")

	// Validasi di backend (bukan hanya di HTML/JavaScript).
	if username == "" || password == "" {
		c.loginError(w, http.StatusBadRequest, "Username dan password wajib diisi")
		return
	}

	user, err := models.GetUserByUsername(c.DB, username)
	if err != nil && !errors.Is(err, models.ErrNotFound) {
		log.Println("login: gagal query user:", err)
		c.loginError(w, http.StatusInternalServerError, "Terjadi kesalahan pada server")
		return
	}

	hash := dummyHash
	if user != nil {
		hash = []byte(user.PasswordHash)
	}
	// Selalu jalankan bcrypt; pesan error sengaja dibuat sama untuk
	// "username salah" maupun "password salah".
	if bcrypt.CompareHashAndPassword(hash, []byte(password)) != nil || user == nil {
		c.loginError(w, http.StatusUnauthorized, "Username atau password salah")
		return
	}

	sess := middleware.Session{
		UserID:   user.ID,
		Username: user.Username,
		Nama:     user.Username,
		Role:     user.Role,
	}

	if user.Role == "nasabah" {
		n, err := models.GetNasabahByUserID(c.DB, user.ID)
		if errors.Is(err, models.ErrNotFound) {
			c.loginError(w, http.StatusForbidden, "Data nasabah tidak ditemukan. Hubungi pengurus.")
			return
		}
		if err != nil {
			log.Println("login: gagal query nasabah:", err)
			c.loginError(w, http.StatusInternalServerError, "Terjadi kesalahan pada server")
			return
		}
		if n.Status != "aktif" {
			c.loginError(w, http.StatusForbidden, "Akun Anda nonaktif. Hubungi pengurus.")
			return
		}
		sess.NasabahID = n.ID
		sess.Nama = n.Nama
	}

	if err := middleware.CreateSession(w, sess); err != nil {
		log.Println("login: gagal membuat session:", err)
		c.loginError(w, http.StatusInternalServerError, "Terjadi kesalahan pada server")
		return
	}
	http.Redirect(w, r, DashboardPath(user.Role), http.StatusSeeOther)
}

func (c *AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	middleware.DestroySession(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
