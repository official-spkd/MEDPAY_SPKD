package models

// RejectionCode adalah master alasan penolakan klaim yang terstruktur.
type RejectionCode struct {
	BaseModel
	Code  string `gorm:"size:50;not null;uniqueIndex:idx_rejection_codes_code,where:deleted_at IS NULL" json:"code"`
	Label string `gorm:"size:150;not null" json:"label"`
}
