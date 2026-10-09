package models

import "time"

// ResumeItem adalah item resume medis pendukung klaim (lab, obat, BMHP, dst.).
type ResumeItem struct {
	BaseModel
	SEPNo       string     `gorm:"column:sep_no;size:30" json:"sep_no"`
	ItemName    string     `gorm:"size:255;not null" json:"item_name"`
	SubCategory string     `gorm:"size:100" json:"sub_category"`
	ServiceDate *time.Time `gorm:"type:date" json:"service_date"`
	ResultNote  string     `gorm:"type:text" json:"result_note"`
	Quantity    float64    `gorm:"type:numeric(10,2);not null;default:0" json:"quantity"` // desimal, mis. 0,5 ampul
	Unit        string     `gorm:"size:30" json:"unit"`
	UnitPrice   int64      `gorm:"not null;default:0" json:"unit_price"`
	Subtotal    int64      `gorm:"not null;default:0" json:"subtotal"`
	Reviewed    bool       `gorm:"not null;default:false" json:"reviewed"`

	ClaimID          uint            `gorm:"not null;index" json:"claim_id"`
	Claim            *Claim          `gorm:"foreignKey:ClaimID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"claim,omitempty"`
	ResumeCategoryID uint            `gorm:"not null;index" json:"resume_category_id"`
	ResumeCategory   *ResumeCategory `gorm:"foreignKey:ResumeCategoryID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"resume_category,omitempty"`
}
