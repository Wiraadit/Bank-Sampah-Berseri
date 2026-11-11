package routes

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"bank-sampah-berseri/controllers"
)

func TestUnauthenticatedRoutes(t *testing.T) {
	templatesDir := filepath.Join("..", "templates")
	views, err := controllers.NewRenderer(templatesDir, false)
	if err != nil {
		t.Fatalf("Gagal memuat template: %v", err)
	}

	handler := New(nil, views)

	tests := []struct {
		method         string
		url            string
		expectedStatus int
		expectedLoc    string
	}{
		{http.MethodGet, "/", http.StatusSeeOther, "/login"},
		{http.MethodGet, "/login", http.StatusOK, ""},
		{http.MethodGet, "/admin/dashboard", http.StatusSeeOther, "/login"},
		{http.MethodGet, "/admin/nasabah", http.StatusSeeOther, "/login"},
		{http.MethodGet, "/admin/setoran", http.StatusSeeOther, "/login"},
		{http.MethodGet, "/admin/pemilahan", http.StatusSeeOther, "/login"},
		{http.MethodGet, "/admin/penjualan", http.StatusSeeOther, "/login"},
		{http.MethodGet, "/admin/saldo", http.StatusSeeOther, "/login"},
		{http.MethodGet, "/admin/iuran", http.StatusSeeOther, "/login"},
		{http.MethodGet, "/admin/laporan", http.StatusSeeOther, "/login"},
		{http.MethodGet, "/nasabah/dashboard", http.StatusSeeOther, "/login"},
		{http.MethodGet, "/nasabah/profil", http.StatusSeeOther, "/login"},
		{http.MethodGet, "/nasabah/setoran", http.StatusSeeOther, "/login"},
		{http.MethodGet, "/nasabah/mutasi", http.StatusSeeOther, "/login"},
		{http.MethodGet, "/nasabah/katalog", http.StatusSeeOther, "/login"},
	}

	for _, tc := range tests {
		req := httptest.NewRequest(tc.method, tc.url, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != tc.expectedStatus {
			t.Errorf("%s %s: status salah: diharapkan %d, didapat %d", tc.method, tc.url, tc.expectedStatus, rec.Code)
		}
		if tc.expectedLoc != "" {
			if loc := rec.Header().Get("Location"); loc != tc.expectedLoc {
				t.Errorf("%s %s: redirect location salah: diharapkan %s, didapat %s", tc.method, tc.url, tc.expectedLoc, loc)
			}
		}
	}
}
