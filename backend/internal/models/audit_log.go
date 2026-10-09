package models

import "time"

const (
	AuditScopeWeb       = "WEB"
	AuditScopeEngineAPI = "ENGINE_API"
)

// AuditLog bersifat append-only: sengaja tidak memakai BaseModel (tanpa UpdatedAt
// dan DeletedAt) karena jejak audit tidak boleh diubah atau dihapus.
// Nama & peran aktor disalin agar log tetap bermakna walau user berubah.
type AuditLog struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	CreatedAt     time.Time `gorm:"not null;index" json:"created_at"`
	ActorName     string    `gorm:"size:150;not null" json:"actor_name"`
	ActorRole     string    `gorm:"size:30;not null" json:"actor_role"`
	Action        string    `gorm:"size:50;not null;index" json:"action"`
	Entity        string    `gorm:"size:50;not null;index:idx_audit_logs_entity" json:"entity"`
	EntityID      string    `gorm:"size:100;index:idx_audit_logs_entity" json:"entity_id"`
	Before        string    `gorm:"type:text" json:"before"`
	After         string    `gorm:"type:text" json:"after"`
	IP            string    `gorm:"column:ip;size:45" json:"ip"`
	CorrelationID string    `gorm:"size:64;index" json:"correlation_id"`
	Scope         string    `gorm:"size:20;not null;default:WEB;index" json:"scope"`
	StatusCode    int       `gorm:"not null;default:0" json:"status_code"`

	ActorID   *uint    `gorm:"index" json:"actor_id"`
	Actor     *User    `gorm:"foreignKey:ActorID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"actor,omitempty"`
	PartnerID *uint    `gorm:"index" json:"partner_id"`
	Partner   *Partner `gorm:"foreignKey:PartnerID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"partner,omitempty"`
}
