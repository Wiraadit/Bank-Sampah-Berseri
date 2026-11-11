package models

import "errors"

// ErrNotFound dikembalikan ketika data yang dicari tidak ada.
var ErrNotFound = errors.New("data tidak ditemukan")

// ErrDuplicate dikembalikan ketika data melanggar kolom UNIQUE (misalnya username sudah dipakai).
var ErrDuplicate = errors.New("data sudah ada")
