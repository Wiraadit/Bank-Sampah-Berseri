# Sistem Informasi Bank Sampah Berseri

Aplikasi web untuk membantu pengelolaan operasional Bank Sampah Berseri. Aplikasi dibuat menggunakan Go, MySQL/MariaDB, HTML, CSS, dan JavaScript.

## Menjalankan aplikasi di komputer sendiri

Aplikasi ini dirancang untuk dijalankan secara lokal. Setiap orang yang ingin mencoba aplikasi perlu menjalankan backend Go dan database MySQL/MariaDB di komputernya sendiri. Tidak perlu deploy ke Vercel.

### 1. Persiapan perangkat lunak

Install:
- Go sesuai versi minimum yang tercantum pada `go.mod` (saat dokumen ini dibuat, `go.mod` menetapkan Go 1.27.0).
- XAMPP atau server MySQL/MariaDB lain.
- Git (opsional, jika mengambil kode dengan `git clone`).

Pastikan perintah `go version` dan `git --version` dapat dijalankan dari terminal.

### 2. Unduh source code

Dengan Git:

```bash
git clone https://github.com/USERNAME/bank-sampah-berseri.git
cd bank-sampah-berseri
```

Ganti `USERNAME` dengan username GitHub pemilik repository. Alternatifnya, buka repository GitHub, pilih **Code → Download ZIP**, lalu ekstrak ZIP dan buka terminal pada folder proyek yang berisi `main.go` dan `go.mod`.

### 3. Siapkan database

1. Buka XAMPP Control Panel.
2. Jalankan **MySQL**. Apache hanya diperlukan jika ingin membuka phpMyAdmin melalui XAMPP.
3. Buka `http://localhost/phpmyadmin`.
4. Pilih tab **Import**, lalu impor `database/schema.sql`.
5. Setelah schema berhasil dibuat, impor `database/seed.sql` untuk memasukkan data awal jenis sampah.

File `schema.sql` membuat database `bank_sampah_berseri`. Pastikan nama database tersebut tidak diubah tanpa menyesuaikan konfigurasi aplikasi.

### 4. Buat konfigurasi database lokal

Salin `config.example.json` menjadi `config.json` di folder utama proyek (sejajar dengan `main.go`). Contoh melalui terminal:

Windows PowerShell:

```powershell
Copy-Item config.example.json config.json
```

Windows Command Prompt:

```cmd
copy config.example.json config.json
```

Linux/macOS:

```bash
cp config.example.json config.json
```

Isi `config.json` sesuai konfigurasi MySQL di komputer masing-masing. Contoh konfigurasi XAMPP standar:

```json
{
  "host": "127.0.0.1",
  "port": "3306",
  "user": "root",
  "password": "",
  "name": "bank_sampah_berseri"
}
```

Jika MySQL menggunakan password atau port berbeda, sesuaikan nilainya. Jangan mengunggah `config.json` yang berisi kredensial pribadi ke repository publik.

### 5. Unduh dependensi dan jalankan aplikasi

Jalankan dari folder utama proyek:

```bash
go mod download
go run .
```

Jika berhasil, server berjalan pada port `8080`. Buka browser dan kunjungi:

```text
http://localhost:8080
```

Biarkan terminal dan MySQL tetap berjalan selama aplikasi digunakan. Untuk menghentikan server, tekan `Ctrl + C` pada terminal.

### 6. Membuat akun admin atau nasabah

Repository tidak menyertakan akun dengan password default. Buat akun melalui utilitas yang tersedia setelah database aktif dan konfigurasi `config.json` benar.

Membuat akun admin:

```bash
go run ./cmd/buatadmin -u admin -p "PasswordAman123"
```

Membuat akun nasabah:

```bash
go run ./cmd/buatadmin -u nasabah01 -p "PasswordAman123" -role nasabah -nama "Nama Nasabah" -kode NSB001
```

Gunakan password pribadi yang kuat. Jangan menggunakan contoh password di atas untuk penggunaan sebenarnya.

## Struktur folder utama

```text
.
├── cmd/buatadmin/   # Utilitas membuat akun
├── config/          # Koneksi dan konfigurasi database
├── controllers/     # Handler halaman dan proses aplikasi
├── database/        # SQL schema dan data awal
├── middleware/      # Middleware autentikasi dan otorisasi
├── models/          # Akses data dan aturan bisnis
├── routes/          # Pendaftaran route HTTP
├── static/          # CSS dan JavaScript
├── templates/       # Template HTML
├── config.example.json
├── main.go
├── go.mod
└── go.sum
```

## Troubleshooting

- **`gagal membaca config.json`**: pastikan `config.json` sudah dibuat dari `config.example.json` dan berada di folder yang sama dengan `main.go`.
- **`connection refused` / gagal terhubung ke database**: pastikan MySQL di XAMPP sudah berjalan, host dan port benar, serta tidak ada aplikasi lain yang memakai port tersebut.
- **`Access denied for user`**: periksa username dan password MySQL di `config.json`.
- **`Unknown database`**: impor `database/schema.sql` terlebih dahulu.
- **Port 8080 sudah digunakan**: hentikan aplikasi lain yang menggunakan port tersebut sebelum menjalankan aplikasi.
- **Versi Go tidak sesuai**: periksa versi Go yang dibutuhkan pada `go.mod`, lalu install versi yang sesuai.

## Catatan untuk kontributor

- Jangan commit `config.json`, file rahasia, kredensial, atau hasil build.
- Perubahan skema database perlu disertai pembaruan SQL dan kode yang terkait.
- Sebelum mengirim perubahan, jalankan pengujian dengan `go test ./...`.

## Teknologi

- Go (`net/http`, `html/template`)
- MySQL/MariaDB
- HTML, CSS, JavaScript
