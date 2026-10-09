package models

import "time"

// Hospital adalah rumah sakit / faskes tenant MedPay.
type Hospital struct {
	BaseModel
	KodeFaskes       string    `gorm:"size:20;not null;uniqueIndex:idx_hospitals_kode_faskes,where:deleted_at IS NULL" json:"kode_faskes"`
	Name             string    `gorm:"size:200;not null" json:"name"`
	City             string    `gorm:"size:100" json:"city"`
	Class            string    `gorm:"size:5" json:"class"`
	Active           bool      `gorm:"not null;default:true" json:"active"`
	EklaimConfigured bool      `gorm:"not null;default:false" json:"eklaim_configured"`
	RegisteredAt     time.Time `gorm:"type:date;not null" json:"registered_at"`

	Users []User `gorm:"foreignKey:HospitalID" json:"users,omitempty"`
}
