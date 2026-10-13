// Package appenv berisi helper pembacaan environment variable.
package appenv

import (
	"os"
	"strings"
	"time"
)

// IsProduction bernilai true bila APP_ENV=production.
func IsProduction() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))) {
	case "production", "prod":
		return true
	}
	return false
}

// Bool membaca env bernilai true/1/yes.
func Bool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "y", "on":
		return true
	}
	return false
}

// Get membaca env; bila kosong memakai fallback.
func Get(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// List membaca env berisi daftar dipisah koma (spasi & entri kosong dibuang).
func List(key, fallback string) []string {
	var out []string
	for _, p := range strings.Split(Get(key, fallback), ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Location adalah zona waktu aplikasi (DB_TIMEZONE, default Asia/Jakarta).
// Seluruh "hari ini", deadline, dan aging dihitung dalam zona ini.
func Location() *time.Location {
	loc, err := time.LoadLocation(Get("DB_TIMEZONE", "Asia/Jakarta"))
	if err != nil {
		return time.FixedZone("WIB", 7*60*60)
	}
	return loc
}
