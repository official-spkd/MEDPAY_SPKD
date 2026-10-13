package model

import (
	"time"

	"github.com/google/uuid"
)

// Session adalah sesi login. Yang disimpan hanya hash token agar token
// tidak bisa dipakai ulang bila database bocor.
type Session struct {
	BaseModel
	TokenHash  string     `gorm:"size:255;not null;uniqueIndex" json:"-"`
	IP         string     `gorm:"column:ip;size:45" json:"ip"`
	UserAgent  string     `gorm:"size:255" json:"user_agent"`
	ExpiresAt  time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	LastSeenAt *time.Time `json:"last_seen_at"` // dipakai untuk timeout tanpa aktivitas

	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User   *User     `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"user,omitempty"`
}
