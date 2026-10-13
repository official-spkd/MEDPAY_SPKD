package api

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/mail"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"medpay_spkd/internal/appenv"
	"medpay_spkd/internal/audit"
	"medpay_spkd/internal/auth"
	"medpay_spkd/internal/engine"
	"medpay_spkd/internal/model"
)

func requirePlatformStaff(c *gin.Context, u model.User, what string) bool {
	if !auth.IsPlatformStaff(u) {
		fail(c, http.StatusForbidden, what+" berlaku untuk semua RS; hanya staf SPKD yang dapat mengubahnya")
		return false
	}
	return true
}

// ── Rumah sakit ──────────────────────────────────────────────────────────────

type hospitalIn struct {
	KodeFaskes       string `json:"kodeFaskes"`
	Name             string `json:"name"`
	City             string `json:"city"`
	Class            string `json:"class"`
	Active           bool   `json:"active"`
	EklaimConfigured bool   `json:"eklaimConfigured"`
}

func (in *hospitalIn) validate() error {
	in.KodeFaskes, in.Name, in.City = strings.TrimSpace(in.KodeFaskes), strings.TrimSpace(in.Name), strings.TrimSpace(in.City)
	switch {
	case in.KodeFaskes == "" || len(in.KodeFaskes) > 20:
		return errors.New("Kode Faskes wajib diisi (maks. 20 karakter)")
	case len([]rune(in.Name)) > 200 || len([]rune(in.City)) > 100 || len(in.Class) > 5:
		return errors.New("nama maks. 200, kota maks. 100, kelas maks. 5 karakter")
	}
	return nil
}

func kodeFaskesTaken(db *gorm.DB, kode string, exceptID uuid.UUID) (bool, error) {
	var n int64
	err := db.Model(&model.Hospital{}).Where("kode_faskes = ? AND id <> ?", kode, exceptID).Count(&n).Error
	return n > 0, err
}

func hospitalCreateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if !requirePlatformStaff(c, u, "Registrasi RS") {
			return
		}
		var in hospitalIn
		if !bind(c, &in) {
			return
		}
		if err := in.validate(); err != nil {
			fail(c, http.StatusBadRequest, err.Error())
			return
		}
		taken, err := kodeFaskesTaken(db, in.KodeFaskes, uuid.Nil)
		if err != nil {
			serverError(c, err)
			return
		}
		if taken {
			fail(c, http.StatusConflict, "Kode Faskes sudah terdaftar")
			return
		}
		h := model.Hospital{KodeFaskes: in.KodeFaskes, Name: in.Name, City: in.City, Class: in.Class,
			Active: in.Active, EklaimConfigured: in.EklaimConfigured, RegisteredAt: engine.Today()}
		if h.Name == "" {
			// Nama belum diketahui: tandai untuk diverifikasi, e-Klaim belum boleh dipakai.
			h.Name = "RS (Kode " + in.KodeFaskes + ") — NAMA PERLU DIVERIFIKASI"
			h.EklaimConfigured = false
		}
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&h).Error; err != nil {
				return err
			}
			k := engine.DefaultKAPJ
			setting := model.KAPJSetting{HospitalID: h.ID, BPJSSharePct: k.BPJSSharePct, CopayEnabled: k.CoPayEnabled,
				PatientCopayPct: k.PatientCoPayPct, OutpatientCopayCap: k.OutpatientCoPayCap, InpatientCopayCap: k.InpatientCoPayCap}
			if err := tx.Create(&setting).Error; err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "REGISTER_HOSPITAL", Entity: "hospital", EntityID: idStr(h.ID),
				After: h.KodeFaskes, HospitalID: &h.ID}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		created(c, hospitalDTO(h))
	}
}

func hospitalUpdateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		id, valid := parseID(c.Param("id"))
		if !valid || !auth.CanAccessHospital(u, id) {
			fail(c, http.StatusNotFound, "RS tidak ditemukan")
			return
		}
		var h model.Hospital
		if err := db.First(&h, "id = ?", id).Error; err != nil {
			notFoundOr500(c, err, "RS tidak ditemukan")
			return
		}
		var in hospitalIn
		if !bind(c, &in) {
			return
		}
		if err := in.validate(); err != nil {
			fail(c, http.StatusBadRequest, err.Error())
			return
		}
		// Kode Faskes & status aktif menentukan akses seluruh pengguna RS: hanya staf SPKD.
		if !auth.IsPlatformStaff(u) && (in.KodeFaskes != h.KodeFaskes || in.Active != h.Active) {
			fail(c, http.StatusForbidden, "Kode Faskes dan status aktif RS hanya dapat diubah staf SPKD")
			return
		}
		taken, err := kodeFaskesTaken(db, in.KodeFaskes, h.ID)
		if err != nil {
			serverError(c, err)
			return
		}
		if taken {
			fail(c, http.StatusConflict, "Kode Faskes sudah terdaftar")
			return
		}
		if in.Name == "" {
			in.Name = h.Name
		}
		before := h.Name
		err = db.Transaction(func(tx *gorm.DB) error {
			err := tx.Model(&h).Updates(map[string]any{"kode_faskes": in.KodeFaskes, "name": in.Name, "city": in.City,
				"class": in.Class, "active": in.Active, "eklaim_configured": in.EklaimConfigured}).Error
			if err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "UPDATE_HOSPITAL", Entity: "hospital", EntityID: idStr(h.ID),
				Before: before, After: in.Name, HospitalID: &h.ID}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		h.KodeFaskes, h.Name, h.City, h.Class = in.KodeFaskes, in.Name, in.City, in.Class
		h.Active, h.EklaimConfigured = in.Active, in.EklaimConfigured
		ok(c, hospitalDTO(h))
	}
}

// ── Pengguna ─────────────────────────────────────────────────────────────────

type userIn struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	HospitalID string `json:"hospitalId"`
	MFAEnabled bool   `json:"mfaEnabled"`
	Active     *bool  `json:"active"` // nil = tidak diubah (default aktif saat dibuat)
	Initials   string `json:"initials"`
	Password   string `json:"password"`
}

func initialsOf(name string) string {
	var out []rune
	for _, w := range strings.Fields(name) {
		r := []rune(w)
		if unicode.IsLetter(r[0]) {
			out = append(out, unicode.ToUpper(r[0]))
		}
		if len(out) == 2 {
			break
		}
	}
	return string(out)
}

// resolveUserInput memvalidasi input akun dan aturan hak akses pembuat/pengubah:
//   - pengguna RS hanya mengelola akun RS-nya sendiri;
//   - peran yang diberikan tidak boleh punya izin di luar izin pengelola (anti eskalasi);
//   - Super Admin tidak terikat RS; peran lain wajib terikat RS (kecuali Auditor lintas RS).
func resolveUserInput(c *gin.Context, db *gorm.DB, actor model.User, in *userIn, exceptID uuid.UUID) (*model.Role, *uuid.UUID, bool) {
	in.Name, in.Email = strings.TrimSpace(in.Name), strings.ToLower(strings.TrimSpace(in.Email))
	if in.Name == "" || len([]rune(in.Name)) > 150 {
		fail(c, http.StatusBadRequest, "Nama wajib diisi (maks. 150 karakter)")
		return nil, nil, false
	}
	if addr, err := mail.ParseAddress(in.Email); err != nil || addr.Address != in.Email || len(in.Email) > 150 {
		fail(c, http.StatusBadRequest, "Format email tidak valid")
		return nil, nil, false
	}
	var n int64
	if err := db.Model(&model.User{}).Where("email = ? AND id <> ?", in.Email, exceptID).Count(&n).Error; err != nil {
		serverError(c, err)
		return nil, nil, false
	}
	if n > 0 {
		fail(c, http.StatusConflict, "Email sudah terdaftar")
		return nil, nil, false
	}
	var role model.Role
	if err := db.Preload("Permissions").Where("code = ?", in.Role).First(&role).Error; err != nil {
		notFoundOr500(c, err, "Peran tidak dikenal")
		return nil, nil, false
	}
	if !auth.CanGrantRole(actor, role) {
		denyAccess(c, db, actor, "role", role.Code, "Anda tidak dapat memberikan peran "+role.Label+" (izinnya melebihi izin Anda)")
		return nil, nil, false
	}

	hid, valid := parseOptID(in.HospitalID)
	if !valid {
		fail(c, http.StatusBadRequest, "hospitalId tidak valid")
		return nil, nil, false
	}
	if actor.HospitalID != nil {
		hid = actor.HospitalID // pengguna RS hanya membuat akun untuk RS-nya
	}
	switch {
	case role.Code == model.RoleSuperAdmin && hid != nil:
		fail(c, http.StatusBadRequest, "Super Admin tidak boleh terikat ke RS")
		return nil, nil, false
	case role.Code != model.RoleSuperAdmin && role.Code != model.RoleAuditor && hid == nil:
		fail(c, http.StatusBadRequest, "Peran ini wajib terikat ke RS (hospitalId)")
		return nil, nil, false
	}
	if hid != nil {
		var h int64
		if err := db.Model(&model.Hospital{}).Where("id = ?", *hid).Count(&h).Error; err != nil {
			serverError(c, err)
			return nil, nil, false
		}
		if h == 0 {
			fail(c, http.StatusBadRequest, "RS tidak dikenal")
			return nil, nil, false
		}
	}
	return &role, hid, true
}

// randomPassword membuat kata sandi sementara yang memenuhi kebijakan (huruf + angka).
func randomPassword() string {
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz"
	const digits = "23456789"
	pick := func(set string) byte {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(set))))
		return set[n.Int64()]
	}
	b := make([]byte, 14)
	for i := range b {
		if i%4 == 3 {
			b[i] = pick(digits)
		} else {
			b[i] = pick(letters)
		}
	}
	return string(b)
}

func userCreateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		var in userIn
		if !bind(c, &in) {
			return
		}
		role, hid, valid := resolveUserInput(c, db, u, &in, uuid.Nil)
		if !valid {
			return
		}
		if in.MFAEnabled && appenv.IsProduction() {
			fail(c, http.StatusBadRequest, "MFA belum tersedia di produksi (hanya kode demo di lingkungan non-produksi)")
			return
		}
		password, temp := in.Password, ""
		if password == "" {
			password = randomPassword()
			temp = password
		}
		if err := auth.ValidatePasswordPolicy(password, in.Email); err != nil {
			fail(c, http.StatusBadRequest, "Kata sandi tidak memenuhi kebijakan: "+err.Error())
			return
		}
		hash, err := auth.HashPassword(password)
		if err != nil {
			serverError(c, err)
			return
		}
		nu := model.User{Name: in.Name, Email: in.Email, Initials: initialsOf(in.Name), PasswordHash: hash,
			MFAEnabled: in.MFAEnabled, HospitalID: hid, RoleID: role.ID, Active: true,
			// Kata sandi dibuat oleh admin → pengguna wajib menggantinya saat pertama masuk.
			MustChangePassword: true}
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&nu).Error; err != nil {
				return err
			}
			if in.Active != nil && !*in.Active {
				nu.Active = false
				if err := tx.Model(&nu).Update("active", false).Error; err != nil {
					return err
				}
			}
			return audit.Log(tx, &u, audit.Entry{Action: "CREATE_USER", Entity: "user", EntityID: idStr(nu.ID),
				After: nu.Email + " (" + role.Code + ")", HospitalID: hid}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		nu.Role = role
		out := userDTO(nu)
		if temp != "" {
			// Ditampilkan sekali; pengguna wajib menggantinya saat pertama masuk.
			out["tempPassword"] = temp
		}
		created(c, out)
	}
}

func userUpdateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		id, valid := parseID(c.Param("id"))
		if !valid {
			fail(c, http.StatusNotFound, "Pengguna tidak ditemukan")
			return
		}
		var target model.User
		if err := db.Preload("Role.Permissions").First(&target, "id = ?", id).Error; err != nil {
			notFoundOr500(c, err, "Pengguna tidak ditemukan")
			return
		}
		if u.HospitalID != nil && (target.HospitalID == nil || *target.HospitalID != *u.HospitalID) {
			fail(c, http.StatusNotFound, "Pengguna tidak ditemukan")
			return
		}
		// Akun dengan izin melebihi pengelola (mis. Admin RS dikelola IT) tidak boleh diubah.
		if target.Role != nil && !auth.CanGrantRole(u, *target.Role) {
			denyAccess(c, db, u, "user", idStr(target.ID), "Anda tidak dapat mengubah akun dengan peran "+target.Role.Label)
			return
		}
		var in userIn
		if !bind(c, &in) {
			return
		}
		role, hid, valid := resolveUserInput(c, db, u, &in, target.ID)
		if !valid {
			return
		}
		active := target.Active
		if in.Active != nil {
			active = *in.Active
		}
		if target.ID == u.ID && (role.ID != target.RoleID || !active) {
			fail(c, http.StatusForbidden, "Anda tidak dapat mengubah peran atau menonaktifkan akun Anda sendiri")
			return
		}
		mfaEnabled := in.MFAEnabled
		if mfaEnabled && !target.MFAEnabled && appenv.IsProduction() {
			fail(c, http.StatusBadRequest, "MFA belum tersedia di produksi (hanya kode demo di lingkungan non-produksi)")
			return
		}
		initials := strings.TrimSpace(in.Initials)
		if initials == "" || len([]rune(initials)) > 4 {
			initials = initialsOf(in.Name)
		}
		before := fmt.Sprintf("%s (%s) aktif=%v mfa=%v", target.Email, auth.RoleCode(target), target.Active, target.MFAEnabled)
		after := fmt.Sprintf("%s (%s) aktif=%v mfa=%v", in.Email, role.Code, active, mfaEnabled)
		err := db.Transaction(func(tx *gorm.DB) error {
			err := tx.Model(&model.User{}).Where("id = ?", target.ID).Updates(map[string]any{
				"name": in.Name, "email": in.Email, "initials": initials, "mfa_enabled": mfaEnabled,
				"hospital_id": hid, "role_id": role.ID, "active": active,
			}).Error
			if err != nil {
				return err
			}
			if role.ID != target.RoleID || !equalUintPtr(hid, target.HospitalID) || !active {
				// Peran / RS berubah atau akun dinonaktifkan: paksa login ulang.
				if err := tx.Model(&model.Session{}).Where("user_id = ? AND revoked_at IS NULL", target.ID).
					Update("revoked_at", gorm.Expr("NOW()")).Error; err != nil {
					return err
				}
			}
			return audit.Log(tx, &u, audit.Entry{Action: "UPDATE_USER", Entity: "user", EntityID: idStr(target.ID),
				Before: before, After: after, HospitalID: hid}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		target.Name, target.Email, target.Initials, target.MFAEnabled = in.Name, in.Email, initials, mfaEnabled
		target.HospitalID, target.RoleID, target.Role, target.Active = hid, role.ID, role, active
		ok(c, userDTO(target))
	}
}

// userResetPasswordHandler membuat kata sandi sementara baru untuk pengguna lain.
func userResetPasswordHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		id, valid := parseID(c.Param("id"))
		if !valid {
			fail(c, http.StatusNotFound, "Pengguna tidak ditemukan")
			return
		}
		var target model.User
		if err := db.Preload("Role.Permissions").First(&target, "id = ?", id).Error; err != nil {
			notFoundOr500(c, err, "Pengguna tidak ditemukan")
			return
		}
		if u.HospitalID != nil && (target.HospitalID == nil || *target.HospitalID != *u.HospitalID) {
			fail(c, http.StatusNotFound, "Pengguna tidak ditemukan")
			return
		}
		if target.Role != nil && !auth.CanGrantRole(u, *target.Role) {
			denyAccess(c, db, u, "user", idStr(target.ID), "Anda tidak dapat mereset kata sandi akun dengan peran "+target.Role.Label)
			return
		}
		temp := randomPassword()
		hash, err := auth.HashPassword(temp)
		if err != nil {
			serverError(c, err)
			return
		}
		err = db.Transaction(func(tx *gorm.DB) error {
			err := tx.Model(&model.User{}).Where("id = ?", target.ID).Updates(map[string]any{
				"password_hash": hash, "must_change_password": true, "failed_logins": 0, "locked_until": nil,
			}).Error
			if err != nil {
				return err
			}
			if err := tx.Model(&model.Session{}).Where("user_id = ? AND revoked_at IS NULL", target.ID).
				Update("revoked_at", gorm.Expr("NOW()")).Error; err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "RESET_PASSWORD", Entity: "user", EntityID: idStr(target.ID),
				HospitalID: target.HospitalID}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		ok(c, J{"ok": true, "tempPassword": temp})
	}
}

func equalUintPtr(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// ── Penjamin ─────────────────────────────────────────────────────────────────

func insurerUpdateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if !requirePlatformStaff(c, u, "Master penjamin") {
			return
		}
		id, valid := parseID(c.Param("id"))
		if !valid {
			fail(c, http.StatusNotFound, "Penjamin tidak ditemukan")
			return
		}
		var ins model.Insurer
		if err := db.First(&ins, "id = ?", id).Error; err != nil {
			notFoundOr500(c, err, "Penjamin tidak ditemukan")
			return
		}
		var in insurerIn
		if !bind(c, &in) {
			return
		}
		if err := in.validate(db, ins.ID); err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, errDuplicate) {
				status = http.StatusConflict
			}
			fail(c, status, err.Error())
			return
		}
		err := db.Transaction(func(tx *gorm.DB) error {
			err := tx.Model(&ins).Updates(map[string]any{"name": in.Name, "type": in.Type, "avg_payment_days": in.AvgPaymentDays,
				"deadline_days": in.DeadlineDays, "ceiling_policy": in.CeilingPolicy, "verified": in.Verified}).Error
			if err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "UPDATE_INSURER", Entity: "insurer", EntityID: idStr(ins.ID),
				After: in.Name + " deadline=" + itoa(in.DeadlineDays) + "d"}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		ins.Name, ins.Type, ins.AvgPaymentDays = in.Name, model.InsurerType(in.Type), in.AvgPaymentDays
		ins.DeadlineDays, ins.CeilingPolicy, ins.Verified = in.DeadlineDays, in.CeilingPolicy, in.Verified
		ok(c, insurerDTO(ins))
	}
}

// ── Partner Engine API ───────────────────────────────────────────────────────

func partnerUpdateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if !auth.IsSuperAdmin(u) {
			fail(c, http.StatusForbidden, "Partner Engine API hanya dapat dikelola Super Admin SPKD")
			return
		}
		id, valid := parseID(c.Param("id"))
		if !valid {
			fail(c, http.StatusNotFound, "Partner tidak ditemukan")
			return
		}
		var p model.Partner
		if err := db.Preload("WhitelistIPs").Preload("Hospitals").Preload("Keys").First(&p, "id = ?", id).Error; err != nil {
			notFoundOr500(c, err, "Partner tidak ditemukan")
			return
		}
		var in struct {
			Company         string   `json:"company"`
			Contact         string   `json:"contact"`
			RateLimitPerMin int      `json:"rateLimitPerMin"`
			WhitelistIPs    []string `json:"whitelistIps"`
			Active          bool     `json:"active"`
			HospitalIDs     []string `json:"hospitalIds"`
		}
		if !bind(c, &in) {
			return
		}
		in.Company = strings.TrimSpace(in.Company)
		if in.Company == "" || len([]rune(in.Company)) > 150 || len([]rune(in.Contact)) > 150 {
			fail(c, http.StatusBadRequest, "Nama perusahaan wajib diisi (maks. 150 karakter)")
			return
		}
		if in.RateLimitPerMin < 1 || in.RateLimitPerMin > 10000 {
			fail(c, http.StatusBadRequest, "Rate limit harus 1–10000 per menit")
			return
		}
		seen := map[string]bool{}
		var ips []model.PartnerWhitelistIP
		for _, raw := range in.WhitelistIPs {
			ip := net.ParseIP(strings.TrimSpace(raw))
			if ip == nil {
				fail(c, http.StatusBadRequest, "IP whitelist tidak valid: "+raw)
				return
			}
			if s := ip.String(); !seen[s] {
				seen[s] = true
				ips = append(ips, model.PartnerWhitelistIP{PartnerID: p.ID, IP: s})
			}
		}
		hospitals := p.Hospitals
		if len(in.HospitalIDs) > 0 {
			hospitals = nil
			for _, raw := range in.HospitalIDs {
				hid, valid := parseID(raw)
				if !valid {
					fail(c, http.StatusBadRequest, "hospitalIds tidak valid")
					return
				}
				var h model.Hospital
				if err := db.First(&h, "id = ?", hid).Error; err != nil {
					notFoundOr500(c, err, "RS tidak dikenal")
					return
				}
				hospitals = append(hospitals, h)
			}
		}

		err := db.Transaction(func(tx *gorm.DB) error {
			err := tx.Model(&p).Updates(map[string]any{"company": in.Company, "contact": in.Contact,
				"rate_limit_per_min": in.RateLimitPerMin, "active": in.Active}).Error
			if err != nil {
				return err
			}
			// Hapus permanen agar IP yang sama bisa didaftarkan ulang (indeks unik).
			if err := tx.Unscoped().Where("partner_id = ?", p.ID).Delete(&model.PartnerWhitelistIP{}).Error; err != nil {
				return err
			}
			if len(ips) > 0 {
				if err := tx.Create(&ips).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(&p).Omit("Hospitals.*").Association("Hospitals").Replace(hospitals); err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "UPDATE_PARTNER", Entity: "partner", EntityID: idStr(p.ID),
				After: in.Company}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		p.WhitelistIPs, p.Hospitals = ips, hospitals
		ok(c, partnerDTO(p))
	}
}
