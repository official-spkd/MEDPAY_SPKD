package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel berisi kolom standar untuk semua tabel (kecuali audit_logs yang append-only).
// Primary key berupa UUID v7: acak sehingga tidak bisa ditebak/di-enumerasi, tetapi
// tetap berurutan waktu sehingga index B-tree tetap efisien.
type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate mengisi ID bila belum diisi.
func (m *BaseModel) BeforeCreate(*gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = NewID()
	}
	return nil
}

// NewID membuat UUID v7 baru.
func NewID() uuid.UUID { return uuid.Must(uuid.NewV7()) }
