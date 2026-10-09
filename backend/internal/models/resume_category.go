package models

// Kode kategori resume medis (kolom resume_categories.code).
const (
	ResumeRekamMedis       = "REKAM_MEDIS"
	ResumeHasilLab         = "HASIL_LAB"
	ResumeHasilRadiologi   = "HASIL_RADIOLOGI"
	ResumeObat             = "OBAT"
	ResumeBMHP             = "BMHP"
	ResumeBilling          = "BILLING"
	ResumePenunjangLainnya = "PENUNJANG_LAINNYA"
)

type ResumeCategory struct {
	BaseModel
	Code  string `gorm:"size:30;not null;uniqueIndex:idx_resume_categories_code,where:deleted_at IS NULL" json:"code"`
	Label string `gorm:"size:100;not null" json:"label"`
}
