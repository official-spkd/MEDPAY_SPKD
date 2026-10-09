package models

type InsurerType string

const (
	InsurerAsuransiSwasta InsurerType = "ASURANSI_SWASTA"
	InsurerAsuransiKantor InsurerType = "ASURANSI_KANTOR"
	InsurerPerusahaan     InsurerType = "PERUSAHAAN"
)

// Insurer adalah penjamin non-BPJS (asuransi swasta/kantor/perusahaan) yang membayar gap.
type Insurer struct {
	BaseModel
	Name           string      `gorm:"size:150;not null;uniqueIndex:idx_insurers_name,where:deleted_at IS NULL" json:"name"`
	Type           InsurerType `gorm:"size:30;not null" json:"type"`
	AvgPaymentDays int         `gorm:"not null;default:0" json:"avg_payment_days"`
	DeadlineDays   int         `gorm:"not null;default:0" json:"deadline_days"` // batas hari pengajuan sejak tanggal masuk
	CeilingPolicy  string      `gorm:"size:255" json:"ceiling_policy"`
	Verified       bool        `gorm:"not null;default:false" json:"verified"`
}
