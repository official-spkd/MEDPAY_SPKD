package models

// KAPJSetting adalah parameter simulasi Koordinasi Antar Penyelenggara Jaminan per RS (1:1).
type KAPJSetting struct {
	BaseModel
	BPJSSharePct       float64 `gorm:"column:bpjs_share_pct;type:numeric(5,2);not null;default:75" json:"bpjs_share_pct"`
	CopayEnabled       bool    `gorm:"not null;default:true" json:"copay_enabled"`
	PatientCopayPct    float64 `gorm:"type:numeric(5,2);not null;default:5" json:"patient_copay_pct"`
	OutpatientCopayCap int64   `gorm:"not null;default:300000" json:"outpatient_copay_cap"`
	InpatientCopayCap  int64   `gorm:"not null;default:3000000" json:"inpatient_copay_cap"`

	HospitalID uint      `gorm:"not null;uniqueIndex:idx_kapj_settings_hospital,where:deleted_at IS NULL" json:"hospital_id"`
	Hospital   *Hospital `gorm:"foreignKey:HospitalID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"hospital,omitempty"`
}

func (KAPJSetting) TableName() string { return "kapj_settings" }
