package models

import "time"

// User adalah pengguna aplikasi. HospitalID nil berarti staf SPKD (lintas RS).
type User struct {
	BaseModel
	Name         string     `gorm:"size:150;not null" json:"name"`
	Email        string     `gorm:"size:150;not null;uniqueIndex:idx_users_email,where:deleted_at IS NULL" json:"email"`
	Initials     string     `gorm:"size:4" json:"initials"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	MFAEnabled   bool       `gorm:"not null;default:false" json:"mfa_enabled"`
	MFASecret    string     `gorm:"size:255" json:"-"`
	LastLoginAt  *time.Time `json:"last_login_at"`

	HospitalID *uint     `gorm:"index" json:"hospital_id"`
	Hospital   *Hospital `gorm:"foreignKey:HospitalID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"hospital,omitempty"`
	RoleID     uint      `gorm:"not null;index" json:"role_id"`
	Role       *Role     `gorm:"foreignKey:RoleID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"role,omitempty"`
}
