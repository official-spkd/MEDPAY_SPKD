// Package db mengelola koneksi PostgreSQL dan migrasi skema (GORM AutoMigrate).
package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"medpay_spkd/internal/appenv"
)

// Open membaca konfigurasi DB_* dari environment, membuka koneksi, lalu
// memastikan database dapat dijangkau. Migrasi dijalankan terpisah (Migrate).
func Open() (*gorm.DB, error) {
	prod := appenv.IsProduction()
	host := appenv.Get("DB_HOST", "localhost")
	password := appenv.Get("DB_PASSWORD", "")
	defaultSSL := "disable"
	if prod {
		defaultSSL = "require"
	}
	sslmode := appenv.Get("DB_SSLMODE", defaultSSL)

	var missing []string
	for _, k := range []string{"DB_USER", "DB_NAME"} {
		if appenv.Get(k, "") == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("environment variable wajib belum diisi: %s", strings.Join(missing, ", "))
	}

	if prod {
		if len(password) < 16 {
			return nil, errors.New("DB_PASSWORD wajib diisi minimal 16 karakter saat APP_ENV=production")
		}
		local := host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasPrefix(host, "/")
		if sslmode == "disable" && !local {
			return nil, fmt.Errorf("DB_SSLMODE=disable ke database %q tidak aman di produksi (pakai require / verify-full)", host)
		}
	}

	// TimeZone sengaja tidak dikutip: driver gorm mem-parsing nilainya sendiri
	// dan kutip akan ikut terkirim ke server.
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		dsnQuote(host),
		dsnQuote(appenv.Get("DB_PORT", "5432")),
		dsnQuote(appenv.Get("DB_USER", "")),
		dsnQuote(password),
		dsnQuote(appenv.Get("DB_NAME", "")),
		dsnQuote(sslmode),
		appenv.Location().String(),
	)

	logLevel := logger.Warn
	if prod {
		// Query tidak ditulis ke log di produksi.
		logLevel = logger.Error
	}
	d, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// "record not found" adalah hasil normal (respons 404), bukan error server.
		Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  !prod,
		}),
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("membuka koneksi database: %w", err)
	}

	sqlDB, err := d.DB()
	if err != nil {
		return nil, fmt.Errorf("mengambil *sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return d, nil
}

// dsnQuote membungkus nilai dengan kutip tunggal agar spasi, kutip, atau
// backslash pada password tidak merusak connection string.
func dsnQuote(v string) string {
	r := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
	return "'" + r.Replace(v) + "'"
}

// Ping dipakai health check untuk memastikan koneksi database masih hidup.
func Ping(ctx context.Context, d *gorm.DB) error {
	sqlDB, err := d.DB()
	if err != nil {
		return fmt.Errorf("mengambil *sql.DB: %w", err)
	}
	return sqlDB.PingContext(ctx)
}

// Close menutup pool koneksi database.
func Close(d *gorm.DB) error {
	sqlDB, err := d.DB()
	if err != nil {
		return fmt.Errorf("mengambil *sql.DB: %w", err)
	}
	return sqlDB.Close()
}
