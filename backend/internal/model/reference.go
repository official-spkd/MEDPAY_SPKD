package model

// Kategori kode referensi (kolom ref_codes.category).
const (
	RefClaimStatus         = "claim_status"
	RefTariffComponent     = "tariff_component"
	RefSpecialCMG          = "special_cmg"
	RefCareType            = "care_type"
	RefCareClass           = "care_class"
	RefParticipantType     = "participant_type"
	RefSex                 = "sex"
	RefInsuranceType       = "insurance_type"
	RefCoverageType        = "coverage_type"
	RefHospitalClass       = "hospital_class"
	RefInsurerType         = "insurer_type"
	RefWorkItemStatus      = "work_item_status"
	RefWorkItemPriority    = "work_item_priority"
	RefWorkItemSource      = "work_item_source"
	RefDeadlineZone        = "deadline_zone"
	RefMatchResult         = "match_result"
	RefPaymentAllocation   = "payment_allocation"
	RefAuditScope          = "audit_scope"
	RefIntakeStatus        = "intake_status"
	RefRegulationStatus    = "regulation_status"
	RefNotificationChannel = "notification_channel"
	RefNotificationStatus  = "notification_status"
	RefScrubSeverity       = "scrub_severity"
	RefClaimAction         = "claim_action" // label tombol aksi menuju status tujuan
)

// RefCode adalah kode referensi untuk pilihan & label tampilan (status, jenis rawat,
// komponen tarif, dst.). Frontend membaca label, warna (tone), dan urutan dari sini,
// sehingga teks dapat diubah tanpa mengubah kode program (N-MNT-01).
type RefCode struct {
	BaseModel
	Category    string `gorm:"size:50;not null;uniqueIndex:idx_ref_codes_category_code,where:deleted_at IS NULL" json:"category"`
	Code        string `gorm:"size:50;not null;uniqueIndex:idx_ref_codes_category_code,where:deleted_at IS NULL" json:"code"`
	Label       string `gorm:"size:150;not null" json:"label"`
	Description string `gorm:"type:text" json:"description"`
	Tone        string `gorm:"size:20" json:"tone"` // neutral | info | teal | success | warning | danger
	SortOrder   int    `gorm:"not null;default:0" json:"sort_order"`
	Active      bool   `gorm:"not null;default:true" json:"active"`
}

// Kunci pengaturan aplikasi (kolom app_settings.key).
const (
	SettingKAPJDisclaimer = "kapj_disclaimer"
	SettingAppName        = "app_name"
	SettingAppTagline     = "app_tagline"
	SettingSupportContact = "support_contact"
)

// AppSetting adalah pengaturan/teks aplikasi berbentuk key–value.
type AppSetting struct {
	BaseModel
	Key         string `gorm:"size:100;not null;uniqueIndex:idx_app_settings_key,where:deleted_at IS NULL" json:"key"`
	Value       string `gorm:"type:text;not null" json:"value"`
	Description string `gorm:"type:text" json:"description"`
}
