package model

import "github.com/google/uuid"

// ClaimStatusHistory mencatat setiap transisi status klaim.
type ClaimStatusHistory struct {
	BaseModel
	FromStatus ClaimStatus `gorm:"size:30" json:"from_status"` // kosong saat klaim dibuat
	ToStatus   ClaimStatus `gorm:"size:30;not null" json:"to_status"`
	Note       string      `gorm:"type:text" json:"note"`

	ClaimID uuid.UUID  `gorm:"type:uuid;not null;index" json:"claim_id"`
	Claim   *Claim     `gorm:"foreignKey:ClaimID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"claim,omitempty"`
	ActorID *uuid.UUID `gorm:"type:uuid;index" json:"actor_id"` // nil = perubahan oleh sistem / update penjamin
	Actor   *User      `gorm:"foreignKey:ActorID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"actor,omitempty"`
}
