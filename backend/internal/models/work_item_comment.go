package models

type WorkItemComment struct {
	BaseModel
	Body string `gorm:"type:text;not null" json:"body"`

	WorkItemID uint      `gorm:"not null;index" json:"work_item_id"`
	WorkItem   *WorkItem `gorm:"foreignKey:WorkItemID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"work_item,omitempty"`
	AuthorID   uint      `gorm:"not null;index" json:"author_id"`
	Author     *User     `gorm:"foreignKey:AuthorID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"author,omitempty"`
}
