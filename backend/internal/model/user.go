package model

import (
	"time"

	"github.com/google/uuid"
)

// User adalah pengguna aplikasi. HospitalID nil berarti staf SPKD (lintas RS).
type User struct {
	BaseModel
	Name         string     `gorm:"size:150;not null" json:"name"`
	Email        string     `gorm:"size:150;not null;uniqueIndex:idx_users_email,where:deleted_at IS NULL" json:"email"`
	Initials     string     `gorm:"size:4" json:"initials"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	MFAEnabled   bool       `gorm:"not null;default:false" json:"mfa_enabled"`
	LastLoginAt  *time.Time `json:"last_login_at"`

	// Active = false: akun dinonaktifkan (tidak bisa login, sesi lama ditolak).
	Active bool `gorm:"not null;default:true" json:"active"`
	// MustChangePassword = true untuk akun dengan kata sandi sementara.
	MustChangePassword bool `gorm:"not null;default:false" json:"must_change_password"`
	// FailedLogins & LockedUntil = penguncian sementara setelah login gagal berturut-turut.
	FailedLogins      int        `gorm:"not null;default:0" json:"-"`
	LockedUntil       *time.Time `json:"-"`
	PasswordChangedAt *time.Time `json:"password_changed_at"`

	HospitalID *uuid.UUID `gorm:"type:uuid;index" json:"hospital_id"`
	Hospital   *Hospital  `gorm:"foreignKey:HospitalID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"hospital,omitempty"`
	RoleID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"role_id"`
	Role       *Role      `gorm:"foreignKey:RoleID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"role,omitempty"`
}
