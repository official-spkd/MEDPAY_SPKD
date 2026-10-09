package models

// PartnerWhitelistIP adalah IP yang diizinkan memanggil Engine API atas nama partner.
type PartnerWhitelistIP struct {
	BaseModel
	IP string `gorm:"column:ip;size:45;not null;uniqueIndex:idx_partner_whitelist_ips_partner_ip,where:deleted_at IS NULL" json:"ip"`

	PartnerID uint     `gorm:"not null;uniqueIndex:idx_partner_whitelist_ips_partner_ip,where:deleted_at IS NULL" json:"partner_id"`
	Partner   *Partner `gorm:"foreignKey:PartnerID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"partner,omitempty"`
}

func (PartnerWhitelistIP) TableName() string { return "partner_whitelist_ips" }
