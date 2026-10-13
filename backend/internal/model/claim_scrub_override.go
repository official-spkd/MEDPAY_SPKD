package model

import "github.com/google/uuid"

// ClaimScrubOverride menandai temuan scrub yang di-override (dengan alasan) untuk satu klaim.
// Unik per (klaim, aturan) di antara baris yang belum dihapus.
type ClaimScrubOverride struct {
	BaseModel
	Reason string `gorm:"type:text;not null" json:"reason"`

	ClaimID        uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_claim_scrub_overrides_claim_rule,where:deleted_at IS NULL" json:"claim_id"`
	Claim          *Claim     `gorm:"foreignKey:ClaimID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"claim,omitempty"`
	ScrubRuleID    uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_claim_scrub_overrides_claim_rule,where:deleted_at IS NULL" json:"scrub_rule_id"`
	ScrubRule      *ScrubRule `gorm:"foreignKey:ScrubRuleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"scrub_rule,omitempty"`
	OverriddenByID uuid.UUID  `gorm:"type:uuid;not null;index" json:"overridden_by_id"`
	OverriddenBy   *User      `gorm:"foreignKey:OverriddenByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"overridden_by,omitempty"`
}
