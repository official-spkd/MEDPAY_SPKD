package models

// Partner adalah mitra integrasi Engine API (vendor SIMRS, dsb.).
type Partner struct {
	BaseModel
	Company              string `gorm:"size:150;not null" json:"company"`
	Contact              string `gorm:"size:150" json:"contact"`
	RateLimitPerMin      int    `gorm:"not null;default:60" json:"rate_limit_per_min"`
	ConsumerID           string `gorm:"size:100;not null;uniqueIndex:idx_partners_consumer_id,where:deleted_at IS NULL" json:"consumer_id"`
	ConsumerSecretHash   string `gorm:"size:255;not null" json:"-"`
	UserKeyHash          string `gorm:"size:255;not null" json:"-"`
	ConsumerSecretMasked string `gorm:"size:50" json:"consumer_secret_masked"`
	UserKeyMasked        string `gorm:"size:50" json:"user_key_masked"`
	Active               bool   `gorm:"not null;default:true" json:"active"`

	WhitelistIPs []PartnerWhitelistIP `gorm:"foreignKey:PartnerID" json:"whitelist_ips,omitempty"`
	Hospitals    []Hospital           `gorm:"many2many:partner_hospitals;foreignKey:ID;joinForeignKey:PartnerID;references:ID;joinReferences:HospitalID" json:"hospitals,omitempty"`
}
