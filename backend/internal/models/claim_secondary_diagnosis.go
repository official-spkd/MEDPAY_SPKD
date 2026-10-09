package models

// ClaimSecondaryDiagnosis adalah diagnosis sekunder (ICD-10) klaim; Seq menjaga urutan input.
type ClaimSecondaryDiagnosis struct {
	BaseModel
	ICD10Code string `gorm:"column:icd10_code;size:10;not null;index" json:"icd10_code"`
	Seq       int    `gorm:"not null;default:0" json:"seq"`

	ClaimID uint   `gorm:"not null;index" json:"claim_id"`
	Claim   *Claim `gorm:"foreignKey:ClaimID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"claim,omitempty"`
}
