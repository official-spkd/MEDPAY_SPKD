package model

import "github.com/google/uuid"

// Batch mengelompokkan klaim untuk pengajuan ke penjamin.
type Batch struct {
	BaseModel
	BatchNo string `gorm:"size:20;not null;uniqueIndex" json:"batch_no"`
	Name    string `gorm:"size:150;not null" json:"name"`
	Period  string `gorm:"size:7;not null;index" json:"period"` // format YYYY-MM
	Note    string `gorm:"type:text" json:"note"`

	HospitalID  uuid.UUID `gorm:"type:uuid;not null;index" json:"hospital_id"`
	Hospital    *Hospital `gorm:"foreignKey:HospitalID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"hospital,omitempty"`
	CreatedByID uuid.UUID `gorm:"type:uuid;not null;index" json:"created_by_id"`
	CreatedBy   *User     `gorm:"foreignKey:CreatedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"created_by,omitempty"`

	Claims []Claim `gorm:"foreignKey:BatchID" json:"claims,omitempty"`
}
