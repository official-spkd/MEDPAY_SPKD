package models

// ClaimProcedure adalah tindakan/prosedur (ICD-9-CM) klaim; Seq menjaga urutan input.
type ClaimProcedure struct {
	BaseModel
	ICD9Code string `gorm:"column:icd9_code;size:10;not null;index" json:"icd9_code"`
	Seq      int    `gorm:"not null;default:0" json:"seq"`

	ClaimID uint   `gorm:"not null;index" json:"claim_id"`
	Claim   *Claim `gorm:"foreignKey:ClaimID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"claim,omitempty"`
}
