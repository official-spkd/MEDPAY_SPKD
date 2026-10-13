package seed

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"medpay_spkd/internal/appenv"
	"medpay_spkd/internal/auth"
	"medpay_spkd/internal/model"
)

// Master mengisi data master yang wajib ada di semua lingkungan (lihat master_data.go).
// Aman dijalankan berulang setiap server start: baris yang belum ada ditambahkan,
// baris yang sudah ada tidak ditimpa (perubahan admin tetap terjaga).
// Mengembalikan jumlah baris yang ditambahkan.
func Master(db *gorm.DB) (int, error) {
	added := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		exists := func(m any, where string, args ...any) (bool, error) {
			var n int64
			err := tx.Model(m).Where(where, args...).Count(&n).Error
			return n > 0, err
		}
		create := func(v any) error {
			added++
			return tx.Create(v).Error
		}

		// Izin
		for _, p := range permissionLabels {
			if ok, err := exists(&model.Permission{}, "code = ?", p.code); err != nil || ok {
				if err != nil {
					return err
				}
				continue
			}
			if err := create(&model.Permission{Code: p.code, Label: p.label}); err != nil {
				return err
			}
		}
		var perms []model.Permission
		if err := tx.Find(&perms).Error; err != nil {
			return err
		}
		permByCode := map[string]model.Permission{}
		for _, p := range perms {
			permByCode[p.Code] = p
		}

		// Peran: izin hanya dipasang saat peran baru dibuat (matriks bisa diubah admin).
		for _, rp := range rolePermissions {
			if ok, err := exists(&model.Role{}, "code = ?", rp.code); err != nil || ok {
				if err != nil {
					return err
				}
				continue
			}
			role := model.Role{Code: rp.code, Label: rp.label}
			if rp.perms == nil {
				role.Permissions = perms
			} else {
				for _, code := range rp.perms {
					p, found := permByCode[code]
					if !found {
						return fmt.Errorf("izin %q untuk peran %s tidak dikenal", code, rp.code)
					}
					role.Permissions = append(role.Permissions, p)
				}
			}
			added++
			if err := tx.Omit("Permissions.*").Create(&role).Error; err != nil {
				return err
			}
		}

		// Kategori resume medis
		for _, c := range resumeCategories {
			if ok, err := exists(&model.ResumeCategory{}, "code = ?", c.code); err != nil || ok {
				if err != nil {
					return err
				}
				continue
			}
			if err := create(&model.ResumeCategory{Code: c.code, Label: c.label}); err != nil {
				return err
			}
		}

		// Kode alasan penolakan
		for _, r := range seedRejectionCodes {
			if ok, err := exists(&model.RejectionCode{}, "code = ?", r.Code); err != nil || ok {
				if err != nil {
					return err
				}
				continue
			}
			if err := create(&model.RejectionCode{Code: r.Code, Label: r.Label}); err != nil {
				return err
			}
		}

		// Aturan scrubber R01–R14 (logikanya di engine, metadata di tabel)
		for _, r := range seedScrubRules {
			if ok, err := exists(&model.ScrubRule{}, "code = ?", r.ID); err != nil || ok {
				if err != nil {
					return err
				}
				continue
			}
			err := create(&model.ScrubRule{Code: r.ID, Name: r.Name, Category: r.Category, Severity: model.Severity(r.Severity),
				Message: r.Message, FixSuggestion: r.FixSuggestion, SourceRef: r.SourceRef, Enabled: r.Enabled})
			if err != nil {
				return err
			}
		}

		// Kode referensi (pilihan & label tampilan)
		for _, cat := range seedRefCodes {
			for i, r := range cat.Items {
				if ok, err := exists(&model.RefCode{}, "category = ? AND code = ?", cat.Category, r.Code); err != nil || ok {
					if err != nil {
						return err
					}
					continue
				}
				err := create(&model.RefCode{Category: cat.Category, Code: r.Code, Label: r.Label,
					Description: r.Description, Tone: r.Tone, SortOrder: (i + 1) * 10, Active: true})
				if err != nil {
					return err
				}
			}
		}

		// Pengaturan aplikasi
		for _, st := range seedSettings {
			if ok, err := exists(&model.AppSetting{}, "key = ?", st.Key); err != nil || ok {
				if err != nil {
					return err
				}
				continue
			}
			v := st
			if err := create(&v); err != nil {
				return err
			}
		}
		return nil
	})
	return added, err
}

// BootstrapAdmin membuat akun Super Admin pertama dari env bila belum ada pengguna sama
// sekali (instalasi baru tanpa data demo):
//
//	INITIAL_ADMIN_EMAIL, INITIAL_ADMIN_PASSWORD, INITIAL_ADMIN_NAME (opsional)
//
// Pengguna wajib mengganti kata sandi saat pertama masuk.
func BootstrapAdmin(db *gorm.DB) (bool, error) {
	email := strings.ToLower(appenv.Get("INITIAL_ADMIN_EMAIL", ""))
	password := appenv.Get("INITIAL_ADMIN_PASSWORD", "")
	if email == "" || password == "" {
		return false, nil
	}
	var n int64
	if err := db.Model(&model.User{}).Count(&n).Error; err != nil || n > 0 {
		return false, err
	}
	if err := auth.ValidatePasswordPolicy(password, email); err != nil {
		return false, fmt.Errorf("INITIAL_ADMIN_PASSWORD: %w", err)
	}
	var role model.Role
	if err := db.Where("code = ?", model.RoleSuperAdmin).First(&role).Error; err != nil {
		return false, fmt.Errorf("peran Super Admin belum ada: %w", err)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return false, err
	}
	name := appenv.Get("INITIAL_ADMIN_NAME", "Super Admin SPKD")
	u := model.User{Name: name, Email: email, Initials: "SA", PasswordHash: hash, RoleID: role.ID,
		Active: true, MustChangePassword: true}
	return true, db.Create(&u).Error
}
