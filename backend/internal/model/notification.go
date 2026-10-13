package model

import (
	"time"

	"github.com/google/uuid"
)

// Notification adalah notifikasi untuk seorang pengguna (in-app, email, dst.).
type Notification struct {
	BaseModel
	Channel  string     `gorm:"size:20;not null" json:"channel"`
	Template string     `gorm:"size:50;not null;index" json:"template"`
	Message  string     `gorm:"type:text;not null" json:"message"`
	Status   string     `gorm:"size:20;not null;index" json:"status"`
	ReadAt   *time.Time `json:"read_at"` // nil = belum dibaca

	RecipientUserID uuid.UUID  `gorm:"type:uuid;not null;index" json:"recipient_user_id"`
	RecipientUser   *User      `gorm:"foreignKey:RecipientUserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"recipient_user,omitempty"`
	ClaimID         *uuid.UUID `gorm:"type:uuid;index" json:"claim_id"`
	Claim           *Claim     `gorm:"foreignKey:ClaimID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"claim,omitempty"`
}
