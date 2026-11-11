package middleware

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const (
	cookieName = "bsb_session"
	sessionTTL = 8 * time.Hour
)

// Session berisi data pengguna yang sedang login.
// NasabahID hanya terisi untuk role nasabah. Halaman nasabah nanti memakai
// nilai ini (bukan parameter URL) sehingga nasabah tidak bisa melihat data orang lain.
type Session struct {
	UserID    int64
	NasabahID int64
	Username  string
	Nama      string
	Role      string
	CSRFToken string
	expires   time.Time
}

var (
	mu    sync.Mutex
	store = map[string]*Session{}
)

type ctxKey struct{}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateSession membuat session baru dan mengirim cookie ke browser.
func CreateSession(w http.ResponseWriter, s Session) error {
	token, err := newToken()
	if err != nil {
		return err
	}
	csrf, err := newToken()
	if err != nil {
		return err
	}
	s.CSRFToken = csrf
	s.expires = time.Now().Add(sessionTTL)

	mu.Lock()
	store[token] = &s
	mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,                 // tidak bisa dibaca JavaScript
		SameSite: http.SameSiteLaxMode, // mengurangi risiko CSRF
		MaxAge:   int(sessionTTL.Seconds()),
	})
	return nil
}

// GetSession mengembalikan salinan session aktif, atau nil jika belum login/kedaluwarsa.
func GetSession(r *http.Request) *Session {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	s, ok := store[c.Value]
	if !ok {
		return nil
	}
	if time.Now().After(s.expires) {
		delete(store, c.Value)
		return nil
	}
	cp := *s
	return &cp
}

// DestroySession menghapus session di server dan cookie di browser.
func DestroySession(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		mu.Lock()
		delete(store, c.Value)
		mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
	})
}

// CurrentUser mengambil session dari context (dipakai di dalam handler yang dilindungi).
func CurrentUser(r *http.Request) *Session {
	s, _ := r.Context().Value(ctxKey{}).(*Session)
	return s
}

// RequireLogin: hanya pengguna yang sudah login yang boleh lewat.
// Untuk request yang mengubah data (POST dll.), token CSRF di form wajib cocok.
func RequireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := GetSession(r)
		if s == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
			got := r.PostFormValue("csrf_token")
			if subtle.ConstantTimeCompare([]byte(got), []byte(s.CSRFToken)) != 1 {
				http.Error(w, "403 - Token keamanan tidak valid. Muat ulang halaman lalu coba lagi.", http.StatusForbidden)
				return
			}
		}

		// Cegah browser menyimpan halaman (tombol Back setelah logout).
		w.Header().Set("Cache-Control", "no-store")
		ctx := context.WithValue(r.Context(), ctxKey{}, s)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole: harus login DAN memiliki role tertentu.
func RequireRole(role string, next http.Handler) http.Handler {
	return RequireLogin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if CurrentUser(r).Role != role {
			http.Error(w, "403 - Anda tidak berhak mengakses halaman ini", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}
