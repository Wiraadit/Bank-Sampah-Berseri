package config

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// DBConfig menampung pengaturan koneksi database.
type DBConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// LoadDBConfig membaca pengaturan dari file JSON.
func LoadDBConfig(path string) (*DBConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca %s: %w", path, err)
	}
	var cfg DBConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("format %s tidak valid: %w", path, err)
	}
	return &cfg, nil
}

// ConnectDB membuka koneksi ke MySQL dan memastikan koneksi berhasil (Ping).
func ConnectDB(cfg *DBConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
	)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("tidak dapat terhubung ke database: %w", err)
	}

	// Migrasi otomatis: pastikan kolom berat_riil tersedia di detail_penjualan (Tahap 6)
	_, _ = db.Exec("ALTER TABLE detail_penjualan ADD COLUMN berat_riil DECIMAL(10,2) NOT NULL DEFAULT 0 AFTER berat")

	return db, nil
}
