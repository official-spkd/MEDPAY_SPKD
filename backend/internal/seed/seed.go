// Package seed mengisi database: data master (selalu, juga produksi) dan data demo (opsional).
//
// Seluruh data demo ditulis sebagai kode Go di file seed_*.go:
//
//	master_data.go   DATA MASTER (selalu diisi, juga di produksi): izin, peran, kategori resume,
//	                 kode penolakan, aturan scrubber R01–R14, kode referensi, pengaturan aplikasi
//	seed_org.go      [demo] rumah sakit, pengguna, penjamin, regulasi
//	seed_claims.go   [demo] batch, klaim, override scrubber
//	seed_resume.go   [demo] item resume medis
//	seed_finance.go  [demo] pembayaran, impor & baris mutasi bank
//	seed_ops.go      [demo] antrean kerja, notifikasi, audit trail
//	seed_platform.go [demo] partner Engine API & klaim masuk via Engine API
//
// master.go berisi Master() & BootstrapAdmin(); steps.go berisi langkah pengisian data demo.
package seed

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"medpay_spkd/internal/auth"
	"medpay_spkd/internal/model"
)

// DemoPassword adalah kata sandi semua akun demo.
const DemoPassword = "medpay2026"

// seedDate = tanggal acuan data demo. Seluruh tanggal transaksi di seed_*.go digeser
// sejauh (hari ini − seedDate) saat seeding agar deadline & aging tetap bermakna.
// Tanggal lahir, tanggal berlaku regulasi, dan tanggal registrasi RS tidak digeser.
const seedDate = "2026-10-03"

// ErrAlreadySeeded dikembalikan bila database sudah berisi data.
var ErrAlreadySeeded = errors.New("database sudah berisi data; jalankan dengan -fresh untuk mengulang dari awal")

type seeder struct {
	tx        *gorm.DB
	data      *demoData
	loc       *time.Location
	shiftDays int
	err       error // error parsing/lookup pertama; dicek setelah tiap langkah

	permIDs        map[string]uuid.UUID // kode izin
	roleIDs        map[string]uuid.UUID // kode peran
	hospitalIDs    map[string]uuid.UUID // id data demo
	userIDs        map[string]uuid.UUID // id data demo
	userByName     map[string]uuid.UUID
	userByEmail    map[string]uuid.UUID
	insurerIDs     map[string]uuid.UUID // id data demo
	rejectionIDs   map[string]uuid.UUID // kode penolakan
	ruleIDs        map[string]uuid.UUID // kode aturan (R01)
	categoryIDs    map[string]uuid.UUID // kode kategori resume
	partnerIDs     map[string]uuid.UUID // id data demo
	partnerByName  map[string]uuid.UUID
	batchIDs       map[string]uuid.UUID // id data demo
	claimIDs       map[string]uuid.UUID // id data demo (CLM-0001)
	claimHospitals map[string]uuid.UUID // id klaim demo → id RS
	importIDs      map[string]uuid.UUID // id data demo
	lineByRef      map[string]uuid.UUID // referensi mutasi bank (hanya yang terkonfirmasi)
	userHospital   map[string]uuid.UUID // nama pengguna → id RS (untuk konteks audit)
	intakeHospital map[string]uuid.UUID // id intake demo → id RS
}

// Run mengisi seluruh tabel dalam satu transaksi. Bila satu langkah gagal,
// tidak ada data yang tersimpan.
func Run(db *gorm.DB, loc *time.Location) error {
	var count int64
	if err := db.Model(&model.Hospital{}).Count(&count).Error; err != nil {
		return fmt.Errorf("cek data awal: %w", err)
	}
	if count > 0 {
		return ErrAlreadySeeded
	}

	data := demoDataset()

	base, err := time.ParseInLocation("2006-01-02", seedDate, loc)
	if err != nil {
		return fmt.Errorf("parsing seedDate: %w", err)
	}
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	return db.Transaction(func(tx *gorm.DB) error {
		s := &seeder{
			tx:        tx,
			data:      data,
			loc:       loc,
			shiftDays: int(today.Sub(base).Hours() / 24),
		}
		steps := []struct {
			name string
			fn   func() error
		}{
			{"master ids", s.loadMaster},
			{"hospitals & kapj_settings", s.seedHospitals},
			{"users", s.seedUsers},
			{"insurers", s.seedInsurers},
			{"regulations", s.seedRegulations},
			{"partners", s.seedPartners},
			{"batches", s.seedBatches},
			{"claims", s.seedClaims},
			{"resume_items", s.seedResumeItems},
			{"claim_scrub_overrides", s.seedScrubOverrides},
			{"bank_imports & bank_lines", s.seedBank},
			{"payments", s.seedPayments},
			{"work_items", s.seedWorkItems},
			{"notifications", s.seedNotifications},
			{"api_intakes", s.seedAPIIntakes},
			{"audit_logs", s.seedAuditLogs},
		}
		for _, st := range steps {
			if err := st.fn(); err != nil {
				return fmt.Errorf("seed %s: %w", st.name, err)
			}
			if s.err != nil {
				return fmt.Errorf("seed %s: %w", st.name, s.err)
			}
		}
		return nil
	})
}

// ── Helper tanggal ───────────────────────────────────────────────────────────

func (s *seeder) parse(v string, shift bool) time.Time {
	if s.err != nil {
		return time.Time{}
	}
	layout := "2006-01-02"
	if len(v) > len(layout) {
		layout = "2006-01-02T15:04:05"
	}
	t, err := time.ParseInLocation(layout, v, s.loc)
	if err != nil {
		s.err = fmt.Errorf("format tanggal %q: %w", v, err)
		return time.Time{}
	}
	if shift {
		t = t.AddDate(0, 0, s.shiftDays)
	}
	return t
}

// at mengembalikan tanggal transaksi yang sudah digeser ke "hari ini".
func (s *seeder) at(v string) time.Time { return s.parse(v, true) }

func (s *seeder) atPtr(v string) *time.Time {
	if v == "" {
		return nil
	}
	t := s.at(v)
	return &t
}

// fixed mengembalikan tanggal tanpa digeser (tanggal lahir, tanggal berlaku regulasi, dst.).
func (s *seeder) fixed(v string) time.Time { return s.parse(v, false) }

func (s *seeder) fixedPtr(v string) *time.Time {
	if v == "" {
		return nil
	}
	t := s.fixed(v)
	return &t
}

func stamp(t time.Time) model.BaseModel {
	return model.BaseModel{CreatedAt: t, UpdatedAt: t}
}

// ── Helper lookup ID ─────────────────────────────────────────────────────────

// must mengambil ID wajib; bila tidak ditemukan, error dicatat di s.err.
func (s *seeder) must(m map[string]uuid.UUID, key, what string) uuid.UUID {
	id, ok := m[key]
	if !ok && s.err == nil {
		s.err = fmt.Errorf("%s %q tidak ditemukan", what, key)
	}
	return id
}

// opt mengambil ID opsional; nil bila kunci kosong atau tidak dikenal.
func opt(m map[string]uuid.UUID, key string) *uuid.UUID {
	if id, ok := m[key]; ok {
		return &id
	}
	return nil
}

func ptr[T any](v T) *T { return &v }

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func hashPassword(pw string) (string, error) {
	h, err := auth.HashPassword(pw)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return h, nil
}
