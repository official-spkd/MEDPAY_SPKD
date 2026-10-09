package database

import (
	"fmt"

	"gorm.io/gorm"

	"medpay_spkd/internal/models"
)

// Models adalah daftar seluruh model yang dikelola AutoMigrate.
// GORM mengurutkan sendiri berdasarkan dependensi foreign key.
func Models() []any {
	return []any{
		// Master & akses
		&models.Hospital{},
		&models.Permission{},
		&models.Role{},
		&models.User{},
		&models.Session{},
		&models.Insurer{},
		&models.RejectionCode{},
		&models.ScrubRule{},
		&models.ResumeCategory{},
		&models.Regulation{},
		&models.KAPJSetting{},

		// Klaim
		&models.Batch{},
		&models.Claim{},
		&models.ClaimSecondaryDiagnosis{},
		&models.ClaimProcedure{},
		&models.ClaimStatusHistory{},
		&models.ResumeItem{},
		&models.ClaimScrubOverride{},

		// Uang & rekonsiliasi
		&models.BankImport{},
		&models.BankLine{},
		&models.Payment{},

		// Operasional & sistem
		&models.WorkItem{},
		&models.WorkItemComment{},
		&models.WorkItemHistory{},
		&models.Notification{},
		&models.Partner{},
		&models.PartnerWhitelistIP{},
		&models.APIIntake{},
		&models.AuditLog{},
	}
}

// joinTables adalah tabel many-to-many yang dibuat otomatis oleh GORM.
var joinTables = []string{"role_permissions", "partner_hospitals"}

// Migrate menjalankan GORM AutoMigrate untuk seluruh model.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(Models()...); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}

// Reset menghapus seluruh tabel aplikasi (termasuk tabel join) lalu migrasi ulang.
// HANYA untuk development: semua data hilang.
func Reset(db *gorm.DB) error {
	tables := make([]any, 0, len(joinTables)+len(Models()))
	for _, t := range joinTables {
		tables = append(tables, t)
	}
	tables = append(tables, Models()...)

	if err := db.Migrator().DropTable(tables...); err != nil {
		return fmt.Errorf("drop tabel: %w", err)
	}
	return Migrate(db)
}
