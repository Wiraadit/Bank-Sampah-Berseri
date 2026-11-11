package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSessionManagement(t *testing.T) {
	rec := httptest.NewRecorder()
	sessData := Session{
		UserID:   1,
		Username: "admin_test",
		Nama:     "Admin Testing",
		Role:     "admin",
	}

	// 1. Uji CreateSession
	if err := CreateSession(rec, sessData); err != nil {
		t.Fatalf("CreateSession gagal: %v", err)
	}

	cookie := rec.Result().Cookies()
	if len(cookie) == 0 {
		t.Fatal("Cookie session tidak diset oleh CreateSession")
	}

	sessCookie := cookie[0]
	if sessCookie.Name != cookieName {
		t.Fatalf("Nama cookie tidak sesuai: diharapkan %s, didapat %s", cookieName, sessCookie.Name)
	}

	// 2. Uji GetSession
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(sessCookie)

	activeSess := GetSession(req)
	if activeSess == nil {
		t.Fatal("GetSession mengembalikan nil untuk session yang baru dibuat")
	}
	if activeSess.Username != "admin_test" || activeSess.Role != "admin" {
		t.Errorf("Data session tidak sesuai: %+v", activeSess)
	}
	if activeSess.CSRFToken == "" {
		t.Error("CSRFToken dalam session kosong")
	}

	// 3. Uji DestroySession
	recDestroy := httptest.NewRecorder()
	DestroySession(recDestroy, req)

	// Setelah destroy, GetSession harus nil
	if postDestroySess := GetSession(req); postDestroySess != nil {
		t.Error("Session masih ditemukan setelah DestroySession dipanggil")
	}
}

func TestRequireLogin(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	protectedHandler := RequireLogin(dummyHandler)

	// Kasus 1: Request tanpa session -> harus redirect ke /login
	reqUnauth := httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
	recUnauth := httptest.NewRecorder()
	protectedHandler.ServeHTTP(recUnauth, reqUnauth)

	if recUnauth.Code != http.StatusSeeOther {
		t.Errorf("Diharapkan status 303 SeeOther saat belum login, didapat %d", recUnauth.Code)
	}
	if loc := recUnauth.Header().Get("Location"); loc != "/login" {
		t.Errorf("Diharapkan redirect ke /login, didapat %s", loc)
	}

	// Buat session valid untuk pengujian selanjutnya
	recLogin := httptest.NewRecorder()
	_ = CreateSession(recLogin, Session{
		UserID:   2,
		Username: "user_test",
		Role:     "nasabah",
	})
	validCookie := recLogin.Result().Cookies()[0]

	// Kasus 2: GET request dengan session valid -> harus lolos
	reqAuthGet := httptest.NewRequest(http.MethodGet, "/portal", nil)
	reqAuthGet.AddCookie(validCookie)
	recAuthGet := httptest.NewRecorder()
	protectedHandler.ServeHTTP(recAuthGet, reqAuthGet)

	if recAuthGet.Code != http.StatusOK {
		t.Errorf("Diharapkan status 200 OK untuk user terautentikasi, didapat %d", recAuthGet.Code)
	}

	// Kasus 3: POST request dengan CSRF token salah -> harus 403 Forbidden
	formInvalid := url.Values{}
	formInvalid.Set("csrf_token", "invalid_token_123")
	reqAuthPostInvalid := httptest.NewRequest(http.MethodPost, "/portal/profil", strings.NewReader(formInvalid.Encode()))
	reqAuthPostInvalid.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqAuthPostInvalid.AddCookie(validCookie)
	recAuthPostInvalid := httptest.NewRecorder()
	protectedHandler.ServeHTTP(recAuthPostInvalid, reqAuthPostInvalid)

	if recAuthPostInvalid.Code != http.StatusForbidden {
		t.Errorf("Diharapkan status 403 Forbidden saat CSRF salah, didapat %d", recAuthPostInvalid.Code)
	}

	// Kasus 4: POST request dengan CSRF token benar -> harus lolos
	activeSess := GetSession(reqAuthGet)
	formValid := url.Values{}
	formValid.Set("csrf_token", activeSess.CSRFToken)
	reqAuthPostValid := httptest.NewRequest(http.MethodPost, "/portal/profil", strings.NewReader(formValid.Encode()))
	reqAuthPostValid.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqAuthPostValid.AddCookie(validCookie)
	recAuthPostValid := httptest.NewRecorder()
	protectedHandler.ServeHTTP(recAuthPostValid, reqAuthPostValid)

	if recAuthPostValid.Code != http.StatusOK {
		t.Errorf("Diharapkan status 200 OK saat CSRF benar, didapat %d", recAuthPostValid.Code)
	}
}

func TestRequireRole(t *testing.T) {
	dummyAdminHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ADMIN_OK"))
	})

	adminOnlyRoute := RequireRole("admin", dummyAdminHandler)

	// Buat session nasabah
	recNasabah := httptest.NewRecorder()
	_ = CreateSession(recNasabah, Session{
		UserID:    10,
		NasabahID: 5,
		Username:  "nasabah1",
		Role:      "nasabah",
	})
	nasabahCookie := recNasabah.Result().Cookies()[0]

	// 1. Nasabah mencoba akses route admin -> harus 403 Forbidden
	reqNasabahAksesAdmin := httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
	reqNasabahAksesAdmin.AddCookie(nasabahCookie)
	recNasabahAksesAdmin := httptest.NewRecorder()
	adminOnlyRoute.ServeHTTP(recNasabahAksesAdmin, reqNasabahAksesAdmin)

	if recNasabahAksesAdmin.Code != http.StatusForbidden {
		t.Errorf("Diharapkan status 403 saat nasabah mengakses route admin, didapat %d", recNasabahAksesAdmin.Code)
	}

	// Buat session admin
	recAdmin := httptest.NewRecorder()
	_ = CreateSession(recAdmin, Session{
		UserID:   1,
		Username: "admin_boss",
		Role:     "admin",
	})
	adminCookie := recAdmin.Result().Cookies()[0]

	// 2. Admin mengakses route admin -> harus 200 OK
	reqAdminAksesAdmin := httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
	reqAdminAksesAdmin.AddCookie(adminCookie)
	recAdminAksesAdmin := httptest.NewRecorder()
	adminOnlyRoute.ServeHTTP(recAdminAksesAdmin, reqAdminAksesAdmin)

	if recAdminAksesAdmin.Code != http.StatusOK {
		t.Errorf("Diharapkan status 200 OK saat admin mengakses route admin, didapat %d", recAdminAksesAdmin.Code)
	}
}

