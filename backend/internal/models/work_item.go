package models

import "time"

const (
	WorkItemNew                 = "New"
	WorkItemAssigned            = "Assigned"
	WorkItemInProgress          = "In Progress"
	WorkItemWaitingVerification = "Waiting Verification"
	WorkItemResolved            = "Resolved"

	PriorityRendah = "Rendah"
	PrioritySedang = "Sedang"
	PriorityTinggi = "Tinggi"
	PriorityKritis = "Kritis"
)

// WorkItem adalah tugas di antrean kerja (dari scrubber, deadline, penolakan, dst.).
type WorkItem struct {
	BaseModel
	ItemNo   string    `gorm:"size:20;not null;uniqueIndex" json:"item_no"`
	Title    string    `gorm:"size:255;not null" json:"title"`
	Source   string    `gorm:"size:30;not null;index" json:"source"`
	Role     string    `gorm:"size:30" json:"role"` // kode peran yang dituju jika belum ada assignee
	DueDate  time.Time `gorm:"type:date;not null;index" json:"due_date"`
	Priority string    `gorm:"size:20;not null;default:Sedang" json:"priority"`
	Status   string    `gorm:"size:30;not null;default:New;index" json:"status"`

	ClaimID     *uint  `gorm:"index" json:"claim_id"`
	Claim       *Claim `gorm:"foreignKey:ClaimID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"claim,omitempty"`
	AssigneeID  *uint  `gorm:"index" json:"assignee_id"`
	Assignee    *User  `gorm:"foreignKey:AssigneeID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"assignee,omitempty"`
	CreatedByID *uint  `gorm:"index" json:"created_by_id"` // nil = dibuat otomatis oleh sistem
	CreatedBy   *User  `gorm:"foreignKey:CreatedByID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"created_by,omitempty"`

	Comments  []WorkItemComment `gorm:"foreignKey:WorkItemID" json:"comments,omitempty"`
	Histories []WorkItemHistory `gorm:"foreignKey:WorkItemID" json:"histories,omitempty"`
}
