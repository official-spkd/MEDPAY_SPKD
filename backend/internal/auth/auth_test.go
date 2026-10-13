package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"medpay_spkd/internal/model"
)

var (
	rs1 = uuid.MustParse("01900000-0000-7000-8000-000000000001")
	rs2 = uuid.MustParse("01900000-0000-7000-8000-000000000002")
)

func uptr(v uuid.UUID) *uuid.UUID { return &v }

func TestDemoMFA(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	u := model.User{}
	if !VerifyMFA(u, DemoMFACode) {
		t.Fatal("kode demo harus diterima di development")
	}
	t.Setenv("APP_ENV", "production")
	if VerifyMFA(u, DemoMFACode) {
		t.Fatal("kode demo tidak boleh diterima di produksi")
	}
}

func TestScopeHospital(t *testing.T) {
	staff := model.User{}
	rs := model.User{HospitalID: uptr(rs1)}

	if h, err := ScopeHospital(staff, ""); err != nil || h != nil {
		t.Fatalf("staf SPKD tanpa filter = %v, %v", h, err)
	}
	if h, err := ScopeHospital(staff, rs2.String()); err != nil || *h != rs2 {
		t.Fatalf("staf SPKD filter RS 2 = %v, %v", h, err)
	}
	if h, err := ScopeHospital(rs, ""); err != nil || *h != rs1 {
		t.Fatalf("user RS = %v, %v", h, err)
	}
	if _, err := ScopeHospital(rs, rs2.String()); err != ErrCrossTenant {
		t.Fatalf("user RS ke RS lain harus ditolak, dapat %v", err)
	}
	if _, err := ScopeHospital(rs, "x"); err == nil {
		t.Fatal("hospitalId tidak valid harus ditolak")
	}
}

func TestSessionTokenAudience(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	uid := model.NewID()
	tok, _, err := sign(uid, sessionAudience, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if id, err := parse(tok, sessionAudience); err != nil || id != uid {
		t.Fatalf("parse sesi = %d, %v", id, err)
	}
	// Token MFA tidak boleh dipakai sebagai token sesi (dan sebaliknya).
	if _, err := parse(tok, mfaAudience); err == nil {
		t.Fatal("token sesi tidak boleh lolos sebagai token MFA")
	}
	mfa, err := IssueMFAChallenge(uid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parse(mfa, sessionAudience); err == nil {
		t.Fatal("token MFA tidak boleh lolos sebagai token sesi")
	}
}

func TestPasswordHashing(t *testing.T) {
	h, err := HashPassword("Rahasia123")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$") || NeedsRehash(h) {
		t.Fatalf("hash bukan Argon2id terkini: %s", h)
	}
	if !VerifyPassword(h, "Rahasia123") || VerifyPassword(h, "rahasia123") {
		t.Fatal("verifikasi Argon2id salah")
	}
	// Hash bcrypt lama tetap bisa login lalu ditandai perlu di-rehash.
	old, _ := bcrypt.GenerateFromPassword([]byte("medpay2026"), bcrypt.MinCost)
	if !VerifyPassword(string(old), "medpay2026") || !NeedsRehash(string(old)) {
		t.Fatal("kompatibilitas bcrypt gagal")
	}
	if VerifyPassword("bukan-hash", "x") {
		t.Fatal("format hash tak dikenal harus ditolak")
	}
}

func TestPasswordPolicy(t *testing.T) {
	cases := map[string]bool{"Rahasia123": true, "pendek1": false, "tanpaangka": false, "12345678": false, "a@b.co1": false}
	for pw, want := range cases {
		if got := ValidatePasswordPolicy(pw, "x@y.id") == nil; got != want {
			t.Errorf("%q: valid=%v, want %v", pw, got, want)
		}
	}
	if ValidatePasswordPolicy("User1@rs.id", "user1@rs.id") == nil {
		t.Error("kata sandi sama dengan email harus ditolak")
	}
}

func role(code string, perms ...string) model.Role {
	r := model.Role{Code: code}
	for _, p := range perms {
		r.Permissions = append(r.Permissions, model.Permission{Code: p})
	}
	return r
}

func TestCanGrantRole(t *testing.T) {
	it := model.User{HospitalID: uptr(rs1)}
	r := role(model.RoleITSIMRS, "VIEW", "CONFIGURE")
	it.Role = &r
	if !CanGrantRole(it, role(model.RoleViewer, "VIEW")) {
		t.Error("IT boleh membuat Viewer")
	}
	if CanGrantRole(it, role(model.RoleHospitalAdmin, "VIEW", "CONFIGURE", "APPROVE")) {
		t.Error("IT tidak boleh membuat Admin RS (eskalasi)")
	}
	all := role(model.RoleHospitalAdmin, "VIEW", "CONFIGURE", "APPROVE")
	admin := model.User{HospitalID: uptr(rs1), Role: &all}
	if CanGrantRole(admin, role(model.RoleSuperAdmin)) {
		t.Error("Admin RS tidak boleh membuat Super Admin")
	}
	sa := role(model.RoleSuperAdmin, "VIEW")
	if !CanGrantRole(model.User{Role: &sa}, role(model.RoleSuperAdmin)) {
		t.Error("Super Admin boleh membuat Super Admin")
	}
}
