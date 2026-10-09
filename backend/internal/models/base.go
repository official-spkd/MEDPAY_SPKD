package models

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel berisi kolom standar untuk semua tabel (kecuali audit_logs yang append-only).
type BaseModel struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
