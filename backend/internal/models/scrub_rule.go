package models

type Severity string

const (
	SeverityBlocker Severity = "BLOCKER"
	SeverityWarning Severity = "WARNING"
	SeverityInfo    Severity = "INFO"
)

// ScrubRule adalah aturan validasi klaim (R01–R14). Logika aturan ada di kode;
// tabel ini menyimpan metadata dan status aktif/nonaktif.
type ScrubRule struct {
	BaseModel
	Code          string   `gorm:"size:10;not null;uniqueIndex:idx_scrub_rules_code,where:deleted_at IS NULL" json:"code"`
	Name          string   `gorm:"size:150;not null" json:"name"`
	Category      string   `gorm:"size:100;not null" json:"category"`
	Severity      Severity `gorm:"size:20;not null" json:"severity"`
	Message       string   `gorm:"type:text;not null" json:"message"`
	FixSuggestion string   `gorm:"type:text" json:"fix_suggestion"`
	SourceRef     string   `gorm:"size:255" json:"source_ref"`
	Enabled       bool     `gorm:"not null;default:true" json:"enabled"`
}
