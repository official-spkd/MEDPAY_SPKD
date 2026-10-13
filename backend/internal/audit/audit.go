// Package audit mencatat audit trail (append-only). Setiap perubahan data
// dicatat di transaksi yang sama dengan perubahannya, sehingga tidak ada
// perubahan tanpa jejak dan tidak ada jejak tanpa perubahan.
package audit

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"medpay_spkd/internal/model"
)

// Meta = konteks request (diisi dari middleware).
type Meta struct {
	IP            string
	CorrelationID string
}

// Entry = satu kejadian audit.
type Entry struct {
	Action     string // mis. CREATE_CLAIM, STATUS_TRANSITION
	Entity     string // mis. claim, batch, user
	EntityID   string // ID publik entitas (CLM-0001, BT-0002, ...)
	Before     string
	After      string
	HospitalID *uuid.UUID // RS konteks kejadian; kosong = RS milik aktor
	StatusCode int        // diisi untuk kejadian gagal (mis. 401 login gagal)
}

const (
	systemActor = "sistem"
	systemRole  = "SYSTEM"
	anonRole    = "ANONYMOUS"
)

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// Log mencatat kejadian oleh user (actor nil = aksi sistem).
func Log(tx *gorm.DB, actor *model.User, e Entry, meta Meta) error {
	entry := model.AuditLog{
		CreatedAt:     time.Now(),
		ActorName:     systemActor,
		ActorRole:     systemRole,
		Action:        truncate(e.Action, 50),
		Entity:        truncate(e.Entity, 50),
		EntityID:      truncate(e.EntityID, 100),
		Before:        e.Before,
		After:         e.After,
		IP:            truncate(meta.IP, 45),
		CorrelationID: truncate(meta.CorrelationID, 64),
		Scope:         model.AuditScopeWeb,
		StatusCode:    e.StatusCode,
		HospitalID:    e.HospitalID,
	}
	if actor != nil {
		entry.ActorName = truncate(actor.Name, 150)
		if actor.Role != nil {
			entry.ActorRole = actor.Role.Code
		}
		id := actor.ID
		entry.ActorID = &id
		if entry.HospitalID == nil {
			entry.HospitalID = actor.HospitalID
		}
	}
	return tx.Create(&entry).Error
}

// LogAnonymous untuk kejadian sebelum ada sesi (mis. login gagal).
func LogAnonymous(tx *gorm.DB, name string, e Entry, meta Meta) error {
	entry := model.AuditLog{
		CreatedAt:     time.Now(),
		ActorName:     truncate(name, 150),
		ActorRole:     anonRole,
		Action:        truncate(e.Action, 50),
		Entity:        truncate(e.Entity, 50),
		EntityID:      truncate(e.EntityID, 100),
		IP:            truncate(meta.IP, 45),
		CorrelationID: truncate(meta.CorrelationID, 64),
		Scope:         model.AuditScopeWeb,
		StatusCode:    e.StatusCode,
		HospitalID:    e.HospitalID,
	}
	return tx.Create(&entry).Error
}

// LogEngine mencatat panggilan Engine API oleh partner (scope ENGINE_API).
func LogEngine(tx *gorm.DB, partnerName string, partnerID, hospitalID *uuid.UUID, e Entry, meta Meta) error {
	if partnerName == "" {
		partnerName = "partner tidak dikenal"
	}
	entry := model.AuditLog{
		CreatedAt:     time.Now(),
		ActorName:     truncate(partnerName, 150),
		ActorRole:     "PARTNER",
		Action:        truncate(e.Action, 50),
		Entity:        truncate(e.Entity, 50),
		EntityID:      truncate(e.EntityID, 100),
		Before:        e.Before,
		After:         e.After,
		IP:            truncate(meta.IP, 45),
		CorrelationID: truncate(meta.CorrelationID, 64),
		Scope:         model.AuditScopeEngineAPI,
		StatusCode:    e.StatusCode,
		PartnerID:     partnerID,
		HospitalID:    hospitalID,
	}
	return tx.Create(&entry).Error
}
