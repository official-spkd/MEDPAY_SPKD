package models

import "time"

type MatchResult string

const (
	MatchMatched     MatchResult = "matched"
	MatchNeedsReview MatchResult = "needs_review"
	MatchUnmatched   MatchResult = "unmatched"
)

// BankLine adalah satu baris mutasi bank beserta hasil pencocokan ke klaim.
type BankLine struct {
	BaseModel
	ValueDate   time.Time   `gorm:"type:date;not null;index" json:"value_date"`
	Description string      `gorm:"type:text" json:"description"`
	Amount      int64       `gorm:"not null" json:"amount"`
	Reference   string      `gorm:"size:100;index" json:"reference"`
	MatchResult MatchResult `gorm:"size:20;not null;default:unmatched;index" json:"match_result"`
	MatchScore  int         `gorm:"not null;default:0" json:"match_score"`
	MatchNote   string      `gorm:"type:text" json:"match_note"`
	Confirmed   bool        `gorm:"not null;default:false" json:"confirmed"`
	ConfirmedAt *time.Time  `json:"confirmed_at"`

	BankImportID  uint        `gorm:"not null;index" json:"bank_import_id"`
	BankImport    *BankImport `gorm:"foreignKey:BankImportID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"bank_import,omitempty"`
	MatchClaimID  *uint       `gorm:"index" json:"match_claim_id"`
	MatchClaim    *Claim      `gorm:"foreignKey:MatchClaimID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"match_claim,omitempty"`
	ConfirmedByID *uint       `gorm:"index" json:"confirmed_by_id"`
	ConfirmedBy   *User       `gorm:"foreignKey:ConfirmedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"confirmed_by,omitempty"`
}
