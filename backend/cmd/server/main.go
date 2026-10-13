// Command server menjalankan SPKD MedPay API.
//
//	go run ./cmd/server          # migrasi, seed demo (bila SEED_DEMO=true & DB kosong), lalu jalan
//	go run ./cmd/server -fresh   # HAPUS semua tabel, migrasi ulang, isi data demo, lalu jalan (dev saja)
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	_ "time/tzdata" // database zona waktu tertanam, agar LoadLocation jalan di Windows

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"medpay_spkd/internal/api"
	"medpay_spkd/internal/appenv"
	"medpay_spkd/internal/auth"
	"medpay_spkd/internal/db"
	"medpay_spkd/internal/engine"
	"medpay_spkd/internal/seed"
)

func main() {
	fresh := flag.Bool("fresh", false, "hapus semua tabel aplikasi, migrasi & seed ulang (data hilang; dev saja)")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Println("file .env tidak ditemukan, memakai environment bawaan")
	}

	prod := appenv.IsProduction()
	if prod {
		gin.SetMode(gin.ReleaseMode)
	}
	loc := appenv.Location()
	engine.SetLocation(loc)

	// Cek JWT_SECRET sebelum server jalan.
	if err := auth.CheckConfig(); err != nil {
		log.Fatal("konfigurasi keamanan: ", err)
	}

	database, err := db.Open()
	if err != nil {
		log.Fatal("gagal konek database: ", err)
	}
	defer func() {
		if err := db.Close(database); err != nil {
			log.Printf("menutup database: %v", err)
		}
	}()
	log.Println("database terhubung")

	switch {
	case *fresh && prod:
		log.Fatal("-fresh ditolak saat APP_ENV=production")
	case *fresh:
		log.Println("menghapus semua tabel & migrasi ulang...")
		err = db.Reset(database)
	default:
		err = db.Migrate(database)
	}
	if err != nil {
		log.Fatal("migrasi database: ", err)
	}
	log.Println("migrasi database selesai")

	// Data master (izin, peran, kode referensi, aturan scrubber, dst.) wajib ada di
	// semua lingkungan; hanya baris yang belum ada yang ditambahkan.
	if n, err := seed.Master(database); err != nil {
		log.Fatal("seed data master gagal: ", err)
	} else if n > 0 {
		log.Printf("data master: %d baris ditambahkan", n)
	}
	if created, err := seed.BootstrapAdmin(database); err != nil {
		log.Fatal("membuat Super Admin awal gagal: ", err)
	} else if created {
		log.Println("Super Admin awal dibuat dari INITIAL_ADMIN_EMAIL (wajib ganti kata sandi saat masuk)")
	}

	// Seed demo hanya bila diminta, tidak pernah di produksi.
	switch {
	case prod && appenv.Bool("SEED_DEMO"):
		log.Fatal("SEED_DEMO=true tidak boleh dipakai saat APP_ENV=production (akun demo bersandi publik)")
	case *fresh || appenv.Bool("SEED_DEMO"):
		start := time.Now()
		switch err := seed.Run(database, loc); {
		case errors.Is(err, seed.ErrAlreadySeeded):
			log.Println("seed demo dilewati: database sudah berisi data")
		case err != nil:
			log.Fatal("seed demo gagal (semua perubahan di-rollback): ", err)
		default:
			log.Printf("seed demo selesai dalam %s — password semua akun demo: %s",
				time.Since(start).Round(time.Millisecond), seed.DemoPassword)
		}
	default:
		log.Println("seed demo dilewati (set SEED_DEMO=true untuk mengisi data uji di lokal)")
	}

	// Tugas otomatis antrean kerja (F-WQ-04). AUTO_TASKS_INTERVAL_MIN=0 menonaktifkan.
	if every, err := strconv.Atoi(appenv.Get("AUTO_TASKS_INTERVAL_MIN", "60")); err == nil && every > 0 {
		api.StartAutoTasks(database, time.Duration(every)*time.Minute)
	}

	r, err := api.NewRouter(database)
	if err != nil {
		log.Fatal("gagal menyiapkan router: ", err)
	}

	port := appenv.Get("APP_PORT", "8080")
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	// Shutdown rapi saat menerima Ctrl+C / SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("SPKD MedPay API jalan di http://localhost:%s (produksi=%v)", port, prod)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("menjalankan server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("mematikan server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown server: %v", err)
	}
}
