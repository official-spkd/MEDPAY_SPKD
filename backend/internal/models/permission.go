package models

// Kode izin yang dicek middleware (kolom permissions.code).
const (
	PermView      = "VIEW"
	PermCreate    = "CREATE"
	PermEdit      = "EDIT"
	PermDelete    = "DELETE"
	PermVerify    = "VERIFY"
	PermApprove   = "APPROVE"
	PermExport    = "EXPORT"
	PermPrint     = "PRINT"
	PermAssign    = "ASSIGN"
	PermResolve   = "RESOLVE"
	PermReopen    = "REOPEN"
	PermConfigure = "CONFIGURE"
	PermOverride  = "OVERRIDE"
)

type Permission struct {
	BaseModel
	Code  string `gorm:"size:30;not null;uniqueIndex:idx_permissions_code,where:deleted_at IS NULL" json:"code"`
	Label string `gorm:"size:100;not null" json:"label"`
}
