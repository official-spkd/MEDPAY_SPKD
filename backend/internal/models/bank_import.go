package models

// BankImport adalah satu kali impor mutasi rekening bank untuk rekonsiliasi.
type BankImport struct {
	BaseModel
	ImportNo    string `gorm:"size:20;not null;uniqueIndex" json:"import_no"`
	BankName    string `gorm:"size:150;not null" json:"bank_name"`
	PeriodLabel string `gorm:"size:100" json:"period_label"`
	LineCount   int    `gorm:"not null;default:0" json:"line_count"`

	HospitalID   uint      `gorm:"not null;index" json:"hospital_id"`
	Hospital     *Hospital `gorm:"foreignKey:HospitalID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"hospital,omitempty"`
	ImportedByID uint      `gorm:"not null;index" json:"imported_by_id"`
	ImportedBy   *User     `gorm:"foreignKey:ImportedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"imported_by,omitempty"`

	Lines []BankLine `gorm:"foreignKey:BankImportID" json:"lines,omitempty"`
}
