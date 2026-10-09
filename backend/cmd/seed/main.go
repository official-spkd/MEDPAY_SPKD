// Command seed mengisi database dengan data demo.
//
//	go run ./cmd/seed          # migrasi + isi data (gagal bila data sudah ada)
//	go run ./cmd/seed -fresh   # HAPUS semua tabel, migrasi ulang, lalu isi data
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"time"
	_ "time/tzdata" // database zona waktu tertanam, agar LoadLocation jalan di Windows

	"gorm.io/gorm"

	"medpay_spkd/internal/config"
	"medpay_spkd/internal/database"
	"medpay_spkd/internal/seeder"
)

func main() {
	fresh := flag.Bool("fresh", false, "hapus semua tabel aplikasi lalu migrasi & seed ulang (data hilang)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("memuat konfigurasi: %v", err)
	}
	if *fresh && cfg.IsProduction() {
		log.Fatal("-fresh ditolak saat APP_ENV=production")
	}

	loc, err := time.LoadLocation(cfg.DB.TimeZone)
	if err != nil {
		log.Fatalf("zona waktu %q tidak dikenal: %v", cfg.DB.TimeZone, err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("koneksi database: %v", err)
	}
	defer func() {
		if err := database.Close(db); err != nil {
			log.Printf("menutup database: %v", err)
		}
	}()

	if *fresh {
		log.Println("menghapus semua tabel & migrasi ulang...")
		err = database.Reset(db)
	} else {
		err = database.Migrate(db)
	}
	if err != nil {
		log.Fatalf("migrasi: %v", err)
	}

	start := time.Now()
	if err := seeder.Run(db, loc); err != nil {
		if errors.Is(err, seeder.ErrAlreadySeeded) {
			log.Fatal(err)
		}
		log.Fatalf("seeding gagal (semua perubahan di-rollback): %v", err)
	}
	log.Printf("seeding selesai dalam %s", time.Since(start).Round(time.Millisecond))

	if err := printCounts(db); err != nil {
		log.Fatalf("menghitung baris: %v", err)
	}
	fmt.Printf("\nPassword semua akun demo: %s\n", seeder.DemoPassword)
}

// printCounts menampilkan jumlah baris per tabel sebagai verifikasi cepat.
func printCounts(db *gorm.DB) error {
	tables := []string{"role_permissions", "partner_hospitals"}
	for _, m := range database.Models() {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(m); err != nil {
			return fmt.Errorf("parse model: %w", err)
		}
		tables = append(tables, stmt.Schema.Table)
	}

	fmt.Println()
	for _, t := range tables {
		var n int64
		if err := db.Table(t).Count(&n).Error; err != nil {
			return fmt.Errorf("hitung %s: %w", t, err)
		}
		fmt.Printf("  %-28s %5d\n", t, n)
	}
	return nil
}
