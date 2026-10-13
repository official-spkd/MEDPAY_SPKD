package model

import (
	"time"

	"github.com/google/uuid"
)

// APIIntake adalah klaim yang masuk melalui Engine API dari partner.
type APIIntake struct {
	BaseModel
	IntakeNo      string    `gorm:"size:20;not null;uniqueIndex" json:"intake_no"`
	SEPNo         string    `gorm:"column:sep_no;size:30;not null;index" json:"sep_no"`
	PatientName   string    `gorm:"size:150" json:"patient_name"`
	INACBGCode    string    `gorm:"column:inacbg_code;size:50" json:"inacbg_code"` // bisa berisi kode error grouper bila gagal
	HospitalTotal int64     `gorm:"not null;default:0" json:"hospital_total"`
	JKNTotal      int64     `gorm:"column:jkn_total;not null;default:0" json:"jkn_total"`
	Gap           int64     `gorm:"not null;default:0" json:"gap"`
	BPJSShare     int64     `gorm:"column:bpjs_share;not null;default:0" json:"bpjs_share"`
	InsurerShare  int64     `gorm:"not null;default:0" json:"insurer_share"`
	PatientShare  int64     `gorm:"not null;default:0" json:"patient_share"`
	Status        string    `gorm:"size:20;not null;index" json:"status"` // RESOLVED / UNRESOLVED
	ErrorCode     string    `gorm:"size:50" json:"error_code"`
	LatencyMs     int       `gorm:"not null;default:0" json:"latency_ms"`
	ReceivedAt    time.Time `gorm:"not null;index" json:"received_at"`

	PartnerID  uuid.UUID `gorm:"type:uuid;not null;index" json:"partner_id"`
	Partner    *Partner  `gorm:"foreignKey:PartnerID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"partner,omitempty"`
	HospitalID uuid.UUID `gorm:"type:uuid;not null;index" json:"hospital_id"`
	Hospital   *Hospital `gorm:"foreignKey:HospitalID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"hospital,omitempty"`
}
