package db

import (
	"fmt"

	"gorm.io/gorm"

	"medpay_spkd/internal/model"
)

// Models adalah daftar seluruh model yang dikelola AutoMigrate.
// GORM mengurutkan sendiri berdasarkan dependensi foreign key.
func Models() []any {
	return []any{
		// Master & akses
		&model.Hospital{},
		&model.RefCode{},
		&model.AppSetting{},
		&model.Permission{},
		&model.Role{},
		&model.User{},
		&model.Session{},
		&model.Insurer{},
		&model.RejectionCode{},
		&model.ScrubRule{},
		&model.ResumeCategory{},
		&model.Regulation{},
		&model.KAPJSetting{},

		// Klaim
		&model.Batch{},
		&model.Claim{},
		&model.ClaimSecondaryDiagnosis{},
		&model.ClaimProcedure{},
		&model.ClaimStatusHistory{},
		&model.ResumeItem{},
		&model.ClaimScrubOverride{},

		// Uang & rekonsiliasi
		&model.BankImport{},
		&model.BankLine{},
		&model.Payment{},

		// Operasional & sistem
		&model.WorkItem{},
		&model.WorkItemComment{},
		&model.WorkItemHistory{},
		&model.Notification{},
		&model.Partner{},
		&model.PartnerWhitelistIP{},
		&model.PartnerKey{},
		&model.APIIntake{},
		&model.AuditLog{},
	}
}

// joinTables adalah tabel many-to-many yang dibuat otomatis oleh GORM.
var joinTables = []string{"role_permissions", "partner_hospitals"}

// upgrades = perubahan skema untuk database yang sudah terpasang sebelum kolom
// baru ditambahkan (aman dijalankan berulang). Kolom NOT NULL diisi dulu dari data
// lama supaya AutoMigrate tidak gagal pada tabel yang sudah berisi baris.
var upgrades = []string{
	// Fitur TOTP dihapus: kolom secret-nya tidak dipakai lagi.
	`ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS mfa_secret, DROP COLUMN IF EXISTS mfa_pending_secret`,
	`DO $$ BEGIN
		IF to_regclass('public.api_intakes') IS NOT NULL AND NOT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'api_intakes' AND column_name = 'intake_no') THEN
			ALTER TABLE api_intakes ADD COLUMN intake_no varchar(20);
			UPDATE api_intakes SET intake_no = 'AI-' || LPAD(id::text, 4, '0');
			ALTER TABLE api_intakes ALTER COLUMN intake_no SET NOT NULL;
		END IF;
	END $$`,
	`DO $$ BEGIN
		IF to_regclass('public.bank_lines') IS NOT NULL AND NOT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'bank_lines' AND column_name = 'line_no') THEN
			ALTER TABLE bank_lines ADD COLUMN line_no varchar(20);
			UPDATE bank_lines SET line_no = 'BL-' || LPAD(id::text, 4, '0');
			ALTER TABLE bank_lines ALTER COLUMN line_no SET NOT NULL;
		END IF;
	END $$`,
}

// Migrate menjalankan upgrade skema lalu GORM AutoMigrate untuk seluruh model.
func Migrate(d *gorm.DB) error {
	// Sejak primary key memakai UUID, skema lama (ID angka) tidak bisa dikonversi otomatis.
	var idType string
	err := d.Raw(`SELECT data_type FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'hospitals' AND column_name = 'id'`).Scan(&idType).Error
	if err != nil {
		return err
	}
	if idType != "" && idType != "uuid" {
		return fmt.Errorf("skema lama terdeteksi (ID bertipe %s, sekarang UUID): jalankan `go run ./cmd/server -fresh` "+
			"untuk membuat ulang tabel (SEMUA DATA DIHAPUS), atau pakai database baru", idType)
	}

	for _, stmt := range upgrades {
		if err := d.Exec(stmt).Error; err != nil {
			return fmt.Errorf("upgrade skema: %w", err)
		}
	}
	if err := d.AutoMigrate(Models()...); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	for _, stmt := range backfills {
		if err := d.Exec(stmt).Error; err != nil {
			return fmt.Errorf("backfill data: %w", err)
		}
	}
	return nil
}

// backfills mengisi kolom baru pada data lama setelah AutoMigrate (aman diulang).
var backfills = []string{
	`UPDATE work_items w SET hospital_id = c.hospital_id
		FROM claims c WHERE w.claim_id = c.id AND w.hospital_id IS NULL`,
	`UPDATE work_items w SET hospital_id = u.hospital_id
		FROM users u WHERE w.assignee_id = u.id AND w.hospital_id IS NULL`,
}

// Reset menghapus seluruh tabel aplikasi (termasuk tabel join) lalu migrasi ulang.
// HANYA untuk development: semua data hilang.
func Reset(d *gorm.DB) error {
	tables := make([]any, 0, len(joinTables)+len(Models()))
	for _, t := range joinTables {
		tables = append(tables, t)
	}
	tables = append(tables, Models()...)

	if err := d.Migrator().DropTable(tables...); err != nil {
		return fmt.Errorf("drop tabel: %w", err)
	}
	return Migrate(d)
}
