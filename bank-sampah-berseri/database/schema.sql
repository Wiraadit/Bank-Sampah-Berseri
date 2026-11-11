CREATE DATABASE IF NOT EXISTS bank_sampah_berseri
  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE bank_sampah_berseri;

CREATE TABLE users (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  username VARCHAR(50) NOT NULL UNIQUE,
  password VARCHAR(255) NOT NULL,            -- hash bcrypt
  role ENUM('admin','nasabah') NOT NULL
) ENGINE=InnoDB;

CREATE TABLE nasabah (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id INT UNSIGNED NOT NULL UNIQUE,
  kode_nasabah VARCHAR(20) NOT NULL UNIQUE,
  nama VARCHAR(100) NOT NULL,
  alamat TEXT,
  no_telepon VARCHAR(20),
  saldo DECIMAL(15,2) NOT NULL DEFAULT 0,
  tunggakan DECIMAL(15,2) NOT NULL DEFAULT 0,
  status ENUM('aktif','nonaktif') NOT NULL DEFAULT 'aktif',
  CONSTRAINT fk_nasabah_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB;

CREATE TABLE jenis_sampah (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  nama_jenis VARCHAR(50) NOT NULL UNIQUE,
  harga_beli DECIMAL(15,2) NOT NULL DEFAULT 0,
  harga_jual DECIMAL(15,2) NOT NULL DEFAULT 0
) ENGINE=InnoDB;

CREATE TABLE pengepul (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  nama VARCHAR(100) NOT NULL,
  no_telepon VARCHAR(20),
  alamat TEXT
) ENGINE=InnoDB;

CREATE TABLE setoran (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  nasabah_id INT UNSIGNED NOT NULL,
  user_id INT UNSIGNED NOT NULL,
  tanggal DATETIME NOT NULL,
  total_nilai DECIMAL(15,2) NOT NULL DEFAULT 0,
  CONSTRAINT fk_setoran_nasabah FOREIGN KEY (nasabah_id) REFERENCES nasabah(id),
  CONSTRAINT fk_setoran_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB;

CREATE TABLE detail_setoran (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  setoran_id INT UNSIGNED NOT NULL,
  jenis_sampah_id INT UNSIGNED NOT NULL,
  berat DECIMAL(10,2) NOT NULL,
  harga_beli DECIMAL(15,2) NOT NULL,         -- snapshot harga
  subtotal DECIMAL(15,2) NOT NULL,
  CONSTRAINT fk_ds_setoran FOREIGN KEY (setoran_id) REFERENCES setoran(id),
  CONSTRAINT fk_ds_jenis FOREIGN KEY (jenis_sampah_id) REFERENCES jenis_sampah(id)
) ENGINE=InnoDB;

CREATE TABLE pemilahan (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id INT UNSIGNED NOT NULL,
  tanggal DATETIME NOT NULL,
  berat_awal DECIMAL(10,2) NOT NULL,
  CONSTRAINT fk_pemilahan_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB;

CREATE TABLE detail_pemilahan (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  pemilahan_id INT UNSIGNED NOT NULL,
  jenis_sampah_id INT UNSIGNED NOT NULL,
  berat_hasil DECIMAL(10,2) NOT NULL,
  CONSTRAINT fk_dp_pemilahan FOREIGN KEY (pemilahan_id) REFERENCES pemilahan(id),
  CONSTRAINT fk_dp_jenis FOREIGN KEY (jenis_sampah_id) REFERENCES jenis_sampah(id)
) ENGINE=InnoDB;

CREATE TABLE penjualan (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  pengepul_id INT UNSIGNED NOT NULL,
  user_id INT UNSIGNED NOT NULL,
  tanggal DATETIME NOT NULL,
  total_pendapatan DECIMAL(15,2) NOT NULL DEFAULT 0,
  CONSTRAINT fk_penjualan_pengepul FOREIGN KEY (pengepul_id) REFERENCES pengepul(id),
  CONSTRAINT fk_penjualan_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB;

-- detail_penjualan: berat = berat stok keluar (mengurangi stok), berat_riil = berat hasil timbangan pengepul (dasar pendapatan & laba)
CREATE TABLE detail_penjualan (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  penjualan_id INT UNSIGNED NOT NULL,
  jenis_sampah_id INT UNSIGNED NOT NULL,
  berat DECIMAL(10,2) NOT NULL,              -- berat stok gudang yang dikeluarkan
  berat_riil DECIMAL(10,2) NOT NULL,         -- berat riil hasil penimbangan di pengepul
  harga_jual DECIMAL(15,2) NOT NULL,
  hpp DECIMAL(15,2) NOT NULL,
  CONSTRAINT fk_dj_penjualan FOREIGN KEY (penjualan_id) REFERENCES penjualan(id),
  CONSTRAINT fk_dj_jenis FOREIGN KEY (jenis_sampah_id) REFERENCES jenis_sampah(id)
) ENGINE=InnoDB;

CREATE TABLE iuran_bulanan (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  nasabah_id INT UNSIGNED NOT NULL,
  bulan TINYINT UNSIGNED NOT NULL,
  tahun SMALLINT UNSIGNED NOT NULL,
  nominal DECIMAL(15,2) NOT NULL,
  dibayar DECIMAL(15,2) NOT NULL DEFAULT 0,
  tunggakan DECIMAL(15,2) NOT NULL DEFAULT 0,
  status ENUM('belum_bayar','sebagian','lunas') NOT NULL DEFAULT 'belum_bayar',
  CONSTRAINT fk_iuran_nasabah FOREIGN KEY (nasabah_id) REFERENCES nasabah(id),
  UNIQUE KEY uq_iuran_periode (nasabah_id, bulan, tahun)  -- cegah tagihan ganda
) ENGINE=InnoDB;

CREATE TABLE transaksi_saldo (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  nasabah_id INT UNSIGNED NOT NULL,
  setoran_id INT UNSIGNED NULL,
  iuran_id INT UNSIGNED NULL,
  nominal DECIMAL(15,2) NOT NULL,
  saldo_sebelum DECIMAL(15,2) NOT NULL,
  saldo_sesudah DECIMAL(15,2) NOT NULL,
  jenis ENUM('setoran','bayar_tunggakan','penarikan','iuran') NOT NULL,
  tanggal DATETIME NOT NULL,
  CONSTRAINT fk_ts_nasabah FOREIGN KEY (nasabah_id) REFERENCES nasabah(id),
  CONSTRAINT fk_ts_setoran FOREIGN KEY (setoran_id) REFERENCES setoran(id),
  CONSTRAINT fk_ts_iuran FOREIGN KEY (iuran_id) REFERENCES iuran_bulanan(id)
) ENGINE=InnoDB;