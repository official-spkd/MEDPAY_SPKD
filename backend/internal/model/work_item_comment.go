package model

import "github.com/google/uuid"

type WorkItemComment struct {
	BaseModel
	Body string `gorm:"type:text;not null" json:"body"`

	WorkItemID uuid.UUID `gorm:"type:uuid;not null;index" json:"work_item_id"`
	WorkItem   *WorkItem `gorm:"foreignKey:WorkItemID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"work_item,omitempty"`
	AuthorID   uuid.UUID `gorm:"type:uuid;not null;index" json:"author_id"`
	Author     *User     `gorm:"foreignKey:AuthorID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"author,omitempty"`
}
