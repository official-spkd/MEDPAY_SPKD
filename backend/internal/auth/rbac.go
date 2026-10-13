package auth

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"medpay_spkd/internal/model"
)

const (
	ctxUser  = "user"
	ctxToken = "session_token"
)

// CurrentUser mengambil user yang sudah divalidasi oleh Wrap().
func CurrentUser(c *gin.Context) (model.User, bool) {
	v, ok := c.Get(ctxUser)
	if !ok {
		return model.User{}, false
	}
	u, ok := v.(model.User)
	return u, ok
}

// RoleCode mengembalikan kode peran user (mis. "BILLING_STAFF").
func RoleCode(u model.User) string {
	if u.Role == nil {
		return ""
	}
	return u.Role.Code
}

// Permissions = daftar kode izin peran user (dari tabel role_permissions).
func Permissions(u model.User) []string {
	out := []string{}
	if u.Role == nil {
		return out
	}
	for _, p := range u.Role.Permissions {
		out = append(out, p.Code)
	}
	return out
}

// HasPermission memeriksa izin peran user.
func HasPermission(u model.User, perm string) bool {
	if u.Role == nil {
		return false
	}
	for _, p := range u.Role.Permissions {
		if p.Code == perm {
			return true
		}
	}
	return false
}

// RequirePermission = middleware tingkat route.
func RequirePermission(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := CurrentUser(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid."})
			return
		}
		if !HasPermission(u, perm) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": fmt.Sprintf("Peran Anda (%s) tidak memiliki izin %s", RoleCode(u), perm),
			})
			return
		}
		c.Next()
	}
}

// ── Isolasi multi-RS ─────────────────────────────────────────────────────────

var ErrCrossTenant = errors.New("akses lintas rumah sakit tidak diizinkan")

// IsPlatformStaff: akun SPKD tidak terikat RS dan boleh melihat semua RS.
func IsPlatformStaff(u model.User) bool { return u.HospitalID == nil }

// IsSuperAdmin: staf SPKD dengan peran Super Admin.
func IsSuperAdmin(u model.User) bool {
	return IsPlatformStaff(u) && RoleCode(u) == model.RoleSuperAdmin
}

// CanAccessHospital = pusat isolasi data: user RS hanya boleh mengakses RS miliknya.
func CanAccessHospital(u model.User, hospitalID uuid.UUID) bool {
	return u.HospitalID == nil || *u.HospitalID == hospitalID
}

// ScopeHospital menentukan filter RS untuk query data. Hasil nil = semua RS
// (hanya untuk staf SPKD). User RS selalu terkunci ke RS miliknya; parameter
// hospitalId milik RS lain ditolak.
func ScopeHospital(u model.User, requested string) (*uuid.UUID, error) {
	if requested == "" {
		return u.HospitalID, nil
	}
	id, err := uuid.Parse(requested)
	if err != nil || id == uuid.Nil {
		return nil, errors.New("hospitalId tidak valid")
	}
	if !CanAccessHospital(u, id) {
		return nil, ErrCrossTenant
	}
	return &id, nil
}

// CanGrantRole: pengguna hanya boleh memberikan peran yang izinnya bagian dari
// izinnya sendiri (mencegah eskalasi hak akses). Peran Super Admin hanya boleh
// diberikan oleh Super Admin. role.Permissions harus sudah di-preload.
func CanGrantRole(actor model.User, role model.Role) bool {
	if role.Code == model.RoleSuperAdmin {
		return IsSuperAdmin(actor)
	}
	for _, p := range role.Permissions {
		if !HasPermission(actor, p.Code) {
			return false
		}
	}
	return true
}
