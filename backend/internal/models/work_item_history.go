package models

// WorkItemHistory mencatat perubahan pada work item (status, assignee, prioritas, komentar).
type WorkItemHistory struct {
	BaseModel
	Action string `gorm:"type:text;not null" json:"action"`

	WorkItemID uint      `gorm:"not null;index" json:"work_item_id"`
	WorkItem   *WorkItem `gorm:"foreignKey:WorkItemID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"work_item,omitempty"`
	ActorID    *uint     `gorm:"index" json:"actor_id"` // nil = aksi sistem
	Actor      *User     `gorm:"foreignKey:ActorID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"actor,omitempty"`
}
