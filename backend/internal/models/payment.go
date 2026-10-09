package models

import "time"

// Payment adalah pembayaran gap dari penjamin untuk satu klaim.
type Payment struct {
	BaseModel
	PaymentNo    string    `gorm:"size:20;not null;uniqueIndex" json:"payment_no"`
	ReceivedDate time.Time `gorm:"type:date;not null;index" json:"received_date"`
	Amount       int64     `gorm:"not null" json:"amount"`
	Payer        string    `gorm:"size:150" json:"payer"`
	TransferRef  string    `gorm:"size:100;index" json:"transfer_ref"`
	Notes        string    `gorm:"type:text" json:"notes"`
	Allocation   string    `gorm:"size:20;not null;default:GAP" json:"allocation"`

	ClaimID     uint      `gorm:"not null;index" json:"claim_id"`
	Claim       *Claim    `gorm:"foreignKey:ClaimID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"claim,omitempty"`
	CreatedByID uint      `gorm:"not null;index" json:"created_by_id"`
	CreatedBy   *User     `gorm:"foreignKey:CreatedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"created_by,omitempty"`
	BankLineID  *uint     `gorm:"index" json:"bank_line_id"` // terisi jika berasal dari konfirmasi rekonsiliasi
	BankLine    *BankLine `gorm:"foreignKey:BankLineID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"bank_line,omitempty"`
}
