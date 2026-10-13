package model

import (
	"time"

	"github.com/google/uuid"
)

// PartnerKey adalah user key Engine API milik partner yang terikat ke tepat satu RS.
// Hanya hash yang disimpan; nilai asli ditampilkan sekali saat dibuat.
type PartnerKey struct {
	BaseModel
	KeyHash    string     `gorm:"size:255;not null;uniqueIndex" json:"-"`
	KeyMasked  string     `gorm:"size:50" json:"key_masked"`
	Active     bool       `gorm:"not null;default:true" json:"active"`
	LastUsedAt *time.Time `json:"last_used_at"`

	PartnerID  uuid.UUID `gorm:"type:uuid;not null;index" json:"partner_id"`
	Partner    *Partner  `gorm:"foreignKey:PartnerID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"partner,omitempty"`
	HospitalID uuid.UUID `gorm:"type:uuid;not null;index" json:"hospital_id"`
	Hospital   *Hospital `gorm:"foreignKey:HospitalID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"hospital,omitempty"`
}
