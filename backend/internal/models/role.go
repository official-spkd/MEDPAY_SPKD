package models

// Kode peran yang dikenal aplikasi (kolom roles.code).
const (
	RoleSuperAdmin    = "SUPER_ADMIN"
	RoleHospitalAdmin = "HOSPITAL_ADMIN"
	RoleManagement    = "MANAGEMENT"
	RoleBillingStaff  = "BILLING_STAFF"
	RoleCoder         = "CODER"
	RoleReviewer      = "REVIEWER"
	RoleFinance       = "FINANCE"
	RoleAuditor       = "AUDITOR"
	RoleITSIMRS       = "IT_SIMRS"
	RoleViewer        = "VIEWER"
)

// Role adalah peran pengguna; izinnya diatur lewat tabel join role_permissions.
type Role struct {
	BaseModel
	Code  string `gorm:"size:30;not null;uniqueIndex:idx_roles_code,where:deleted_at IS NULL" json:"code"`
	Label string `gorm:"size:100;not null" json:"label"`

	Permissions []Permission `gorm:"many2many:role_permissions;foreignKey:ID;joinForeignKey:RoleID;references:ID;joinReferences:PermissionID" json:"permissions,omitempty"`
}
