package models

import "time"

// Regulation adalah pemetaan regulasi ke modul/fitur untuk halaman Kepatuhan.
type Regulation struct {
	BaseModel
	Number        string     `gorm:"size:100;not null" json:"number"`
	Title         string     `gorm:"size:255;not null" json:"title"`
	Article       string     `gorm:"size:100" json:"article"`
	Requirement   string     `gorm:"type:text" json:"requirement"`
	EffectiveDate *time.Time `gorm:"type:date" json:"effective_date"`
	Module        string     `gorm:"size:100" json:"module"`
	Feature       string     `gorm:"size:150" json:"feature"`
	Status        string     `gorm:"size:30;not null;index" json:"status"`
	LastReviewed  *time.Time `gorm:"type:date" json:"last_reviewed"`
	Note          string     `gorm:"type:text" json:"note"`

	ReviewerID *uint `gorm:"index" json:"reviewer_id"`
	Reviewer   *User `gorm:"foreignKey:ReviewerID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"reviewer,omitempty"`
}
