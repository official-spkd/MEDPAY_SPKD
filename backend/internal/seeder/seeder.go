// Package seeder mengisi database dengan data demo SPKD MedPay.
// Sumber data: data/mock_data.json, salinan seed dari mock API frontend,
// sehingga backend dan mode mock frontend menampilkan data yang sama.
package seeder

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"medpay_spkd/internal/models"
)

// DemoPassword adalah kata sandi semua akun demo.
const DemoPassword = "medpay2026"

// seedDate adalah tanggal pembuatan data mock. Seluruh tanggal transaksi digeser
// relatif ke hari ini (sama seperti mock frontend) agar deadline & aging tetap bermakna.
const seedDate = "2026-10-03"

// ErrAlreadySeeded dikembalikan bila database sudah berisi data.
var ErrAlreadySeeded = errors.New("database sudah berisi data; jalankan dengan -fresh untuk mengulang dari awal")

//go:embed data/mock_data.json
var mockJSON []byte

type seeder struct {
	tx        *gorm.DB
	data      *mockData
	loc       *time.Location
	shiftDays int
	err       error // error parsing/lookup pertama; dicek setelah tiap langkah

	permIDs        map[string]uint // kode izin
	roleIDs        map[string]uint // kode peran
	hospitalIDs    map[string]uint // id mock
	userIDs        map[string]uint // id mock
	userByName     map[string]uint
	userByEmail    map[string]uint
	insurerIDs     map[string]uint // id mock
	rejectionIDs   map[string]uint // kode penolakan
	ruleIDs        map[string]uint // kode aturan (R01)
	categoryIDs    map[string]uint // kode kategori resume
	partnerIDs     map[string]uint // id mock
	partnerByName  map[string]uint
	batchIDs       map[string]uint // id mock
	claimIDs       map[string]uint // id mock (CLM-0001)
	claimHospitals map[string]uint // id mock klaim → id RS
	importIDs      map[string]uint // id mock
	lineByRef      map[string]uint // referensi mutasi bank (hanya yang terkonfirmasi)
}

// Run mengisi seluruh tabel dalam satu transaksi. Bila satu langkah gagal,
// tidak ada data yang tersimpan.
func Run(db *gorm.DB, loc *time.Location) error {
	var count int64
	if err := db.Model(&models.Hospital{}).Count(&count).Error; err != nil {
		return fmt.Errorf("cek data awal: %w", err)
	}
	if count > 0 {
		return ErrAlreadySeeded
	}

	var data mockData
	if err := json.Unmarshal(mockJSON, &data); err != nil {
		return fmt.Errorf("membaca mock_data.json: %w", err)
	}

	base, err := time.ParseInLocation("2006-01-02", seedDate, loc)
	if err != nil {
		return fmt.Errorf("parsing seedDate: %w", err)
	}
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	return db.Transaction(func(tx *gorm.DB) error {
		s := &seeder{
			tx:        tx,
			data:      &data,
			loc:       loc,
			shiftDays: int(today.Sub(base).Hours() / 24),
		}
		steps := []struct {
			name string
			fn   func() error
		}{
			{"permissions & roles", s.seedRBAC},
			{"hospitals & kapj_settings", s.seedHospitals},
			{"users", s.seedUsers},
			{"master data", s.seedMasters},
			{"regulations", s.seedRegulations},
			{"partners", s.seedPartners},
			{"batches", s.seedBatches},
			{"claims", s.seedClaims},
			{"resume_items", s.seedResumeItems},
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

func stamp(t time.Time) models.BaseModel {
	return models.BaseModel{CreatedAt: t, UpdatedAt: t}
}

// ── Helper lookup ID ─────────────────────────────────────────────────────────

// must mengambil ID wajib; bila tidak ditemukan, error dicatat di s.err.
func (s *seeder) must(m map[string]uint, key, what string) uint {
	id, ok := m[key]
	if !ok && s.err == nil {
		s.err = fmt.Errorf("%s %q tidak ditemukan", what, key)
	}
	return id
}

// opt mengambil ID opsional; nil bila kunci kosong atau tidak dikenal.
func opt(m map[string]uint, key string) *uint {
	if id, ok := m[key]; ok {
		return &id
	}
	return nil
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func hashPassword(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(h), nil
}
