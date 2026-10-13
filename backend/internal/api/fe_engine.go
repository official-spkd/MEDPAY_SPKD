package api

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"medpay_spkd/internal/audit"
	"medpay_spkd/internal/auth"
	"medpay_spkd/internal/engine"
	"medpay_spkd/internal/model"
)

// ── Engine API untuk SIMRS partner (F-ADM-05) ─────────────────────────────────
//
// POST /api/engine/claims
//   Header: X-Consumer-Id, X-Consumer-Secret, X-User-Key
//   - consumer secret = kredensial partner;
//   - user key        = terikat ke tepat satu RS (tabel partner_keys);
//   - IP pemanggil wajib ada di whitelist partner;
//   - dibatasi rate_limit_per_min per partner.
// Setiap panggilan (berhasil maupun ditolak) tercatat di audit dengan scope ENGINE_API.

func sha256Hex(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

func secretMatches(plain, hash string) bool {
	return hash != "" && subtle.ConstantTimeCompare([]byte(sha256Hex(plain)), []byte(hash)) == 1
}

// randomSecret membuat kredensial acak 256-bit dengan prefix (mis. "cs_", "uk_").
func randomSecret(prefix string) string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

func maskSecret(prefix, v string) string { return prefix + "••••••••" + v[len(v)-4:] }

// rateLimiter = jendela tetap per menit per partner (in-memory, per instance server).
type rateLimiter struct {
	mu     sync.Mutex
	window int64
	counts map[uuid.UUID]int
}

func (r *rateLimiter) allow(partnerID uuid.UUID, limit int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	w := time.Now().Unix() / 60
	if w != r.window {
		r.window, r.counts = w, map[uuid.UUID]int{}
	}
	r.counts[partnerID]++
	return r.counts[partnerID] <= limit
}

type engineClaimIn struct {
	SEPNo            string `json:"sepNo"`
	PatientName      string `json:"patientName"`
	PrimaryDiagnosis string `json:"primaryDiagnosis"`
	CareType         string `json:"careType"`
	CareClass        string `json:"careClass"`
	AdmissionDate    string `json:"admissionDate"`
	DischargeDate    string `json:"dischargeDate"`
	HospitalTotal    int64  `json:"hospitalTotal"`
	PolicyCeiling    *int64 `json:"policyCeiling"`
}

func engineClaimHandler(db *gorm.DB) gin.HandlerFunc {
	limiter := &rateLimiter{counts: map[uuid.UUID]int{}}
	return func(c *gin.Context) {
		start := time.Now()
		meta := metaOf(c)
		var (
			partner    model.Partner
			partnerID  *uuid.UUID
			hospitalID *uuid.UUID
		)
		reject := func(status int, msg string) {
			_ = audit.LogEngine(db, partner.Company, partnerID, hospitalID, audit.Entry{
				Action: "ENGINE_API_POST_CLAIMS", Entity: "api_intake", After: msg, StatusCode: status,
			}, meta)
			fail(c, status, msg)
		}

		consumerID := strings.TrimSpace(c.GetHeader("X-Consumer-Id"))
		if consumerID == "" || db.Preload("WhitelistIPs").Where("consumer_id = ?", consumerID).First(&partner).Error != nil {
			reject(http.StatusUnauthorized, "Kredensial partner tidak valid")
			return
		}
		pid := partner.ID
		partnerID = &pid
		if !secretMatches(c.GetHeader("X-Consumer-Secret"), partner.ConsumerSecretHash) {
			reject(http.StatusUnauthorized, "Kredensial partner tidak valid")
			return
		}
		if !partner.Active {
			reject(http.StatusForbidden, "Partner dinonaktifkan")
			return
		}
		ip, allowed := c.ClientIP(), false
		for _, w := range partner.WhitelistIPs {
			if w.IP == ip {
				allowed = true
			}
		}
		if !allowed {
			reject(http.StatusForbidden, "IP "+ip+" tidak terdaftar di whitelist partner")
			return
		}
		if !limiter.allow(partner.ID, max(partner.RateLimitPerMin, 1)) {
			reject(http.StatusTooManyRequests, fmt.Sprintf("Batas %d permintaan per menit terlampaui", partner.RateLimitPerMin))
			return
		}
		var key model.PartnerKey
		err := db.Preload("Hospital").Where("key_hash = ? AND partner_id = ? AND active = ?",
			sha256Hex(c.GetHeader("X-User-Key")), partner.ID, true).First(&key).Error
		if err != nil || key.Hospital == nil {
			reject(http.StatusUnauthorized, "User key tidak valid")
			return
		}
		hid := key.HospitalID
		hospitalID = &hid
		if !key.Hospital.Active {
			reject(http.StatusForbidden, "RS dinonaktifkan")
			return
		}

		var in engineClaimIn
		if err := c.ShouldBindJSON(&in); err != nil {
			reject(http.StatusBadRequest, "Format data tidak valid: "+err.Error())
			return
		}
		adm, err1 := parseDate(in.AdmissionDate)
		dis, err2 := parseDate(in.DischargeDate)
		switch {
		case strings.TrimSpace(in.SEPNo) == "" || len(in.SEPNo) > 30 || len(in.PatientName) > 150:
			reject(http.StatusBadRequest, "sepNo wajib diisi (maks. 30), patientName maks. 150 karakter")
			return
		case !oneOf(in.CareType, model.CareTypeRawatInap, model.CareTypeRawatJalan) || !oneOf(in.CareClass, "I", "II", "III"):
			reject(http.StatusBadRequest, "careType/careClass tidak valid")
			return
		case err1 != nil || err2 != nil || adm == nil || dis == nil:
			reject(http.StatusBadRequest, "admissionDate & dischargeDate wajib berformat YYYY-MM-DD")
			return
		case in.HospitalTotal <= 0 || (in.PolicyCeiling != nil && *in.PolicyCeiling < 0):
			reject(http.StatusBadRequest, "hospitalTotal harus > 0 dan policyCeiling tidak boleh negatif")
			return
		}

		// Grouper + pembagian KAPJ memakai pengaturan RS.
		los := engine.LOS(adm, dis)
		g := engine.DemoGrouper(strings.ToUpper(strings.TrimSpace(in.PrimaryDiagnosis)), in.CareType, in.CareClass, los, key.Hospital.EklaimConfigured)
		intake := model.APIIntake{SEPNo: strings.TrimSpace(in.SEPNo), PatientName: in.PatientName,
			HospitalTotal: in.HospitalTotal, ReceivedAt: start, PartnerID: partner.ID, HospitalID: hid}
		if g.OK {
			params := engine.DefaultKAPJ
			var s model.KAPJSetting
			if db.Where("hospital_id = ?", hid).First(&s).Error == nil {
				params = engine.KAPJFromSetting(s)
			}
			k := engine.SplitKAPJ(in.HospitalTotal, g.BaseTariff, in.PolicyCeiling, in.CareType, params)
			intake.Status, intake.INACBGCode, intake.JKNTotal = "RESOLVED", g.Code, g.BaseTariff
			intake.Gap = in.HospitalTotal - g.BaseTariff
			intake.BPJSShare, intake.InsurerShare, intake.PatientShare = k.BPJSPays, k.InsurerPays, k.PatientPays
		} else {
			intake.Status, intake.INACBGCode, intake.ErrorCode = "UNRESOLVED", g.Error, g.Error
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			no, err := nextNo(tx, "api_intakes", "intake_no", "AI")
			if err != nil {
				return err
			}
			intake.IntakeNo = no
			intake.LatencyMs = int(time.Since(start).Milliseconds())
			if err := tx.Create(&intake).Error; err != nil {
				return err
			}
			now := time.Now()
			if err := tx.Model(&key).Update("last_used_at", now).Error; err != nil {
				return err
			}
			return audit.LogEngine(tx, partner.Company, partnerID, hospitalID, audit.Entry{
				Action: "ENGINE_API_POST_CLAIMS", Entity: "api_intake", EntityID: intake.IntakeNo,
				After: intake.Status + " " + intake.INACBGCode, StatusCode: http.StatusCreated,
			}, meta)
		})
		if err != nil {
			serverError(c, err)
			return
		}
		created(c, intakeDTO(intake))
	}
}

// ── Pengelolaan kredensial partner (Super Admin) ─────────────────────────────

func loadPartnerForAdmin(c *gin.Context, db *gorm.DB) (*model.Partner, bool) {
	u := currentUser(c)
	if !auth.IsSuperAdmin(u) {
		denyAccess(c, db, u, "partner", c.Param("id"), "Partner Engine API hanya dapat dikelola Super Admin SPKD")
		return nil, false
	}
	id, valid := parseID(c.Param("id"))
	if !valid {
		fail(c, http.StatusNotFound, "Partner tidak ditemukan")
		return nil, false
	}
	var p model.Partner
	if err := db.Preload("Hospitals").First(&p, "id = ?", id).Error; err != nil {
		notFoundOr500(c, err, "Partner tidak ditemukan")
		return nil, false
	}
	return &p, true
}

// partnerKeyCreateHandler menerbitkan user key baru untuk satu RS. Key lama partner
// untuk RS yang sama dinonaktifkan. Nilai key hanya ditampilkan sekali.
func partnerKeyCreateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		p, found := loadPartnerForAdmin(c, db)
		if !found {
			return
		}
		var body struct {
			HospitalID string `json:"hospitalId"`
		}
		if !bind(c, &body) {
			return
		}
		hid, valid := parseID(body.HospitalID)
		var h model.Hospital
		if !valid || db.First(&h, "id = ?", hid).Error != nil {
			fail(c, http.StatusBadRequest, "RS tidak dikenal")
			return
		}
		plain := randomSecret("uk_")
		key := model.PartnerKey{PartnerID: p.ID, HospitalID: hid, KeyHash: sha256Hex(plain), KeyMasked: maskSecret("uk_", plain), Active: true}
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&model.PartnerKey{}).Where("partner_id = ? AND hospital_id = ?", p.ID, hid).Update("active", false).Error; err != nil {
				return err
			}
			if err := tx.Create(&key).Error; err != nil {
				return err
			}
			// RS otomatis masuk daftar RS yang dilayani partner.
			if err := tx.Model(p).Omit("Hospitals.*").Association("Hospitals").Append(&h); err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "PARTNER_KEY_ISSUED", Entity: "partner", EntityID: idStr(p.ID),
				After: h.KodeFaskes + " " + key.KeyMasked, HospitalID: &hid}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		created(c, J{"userKey": plain, "keyMasked": key.KeyMasked, "hospitalId": idStr(hid), "partnerId": idStr(p.ID)})
	}
}

// partnerRotateSecretHandler membuat consumer secret baru (yang lama langsung tidak berlaku).
func partnerRotateSecretHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		p, found := loadPartnerForAdmin(c, db)
		if !found {
			return
		}
		plain := randomSecret("cs_")
		masked := maskSecret("cs_", plain)
		err := db.Transaction(func(tx *gorm.DB) error {
			err := tx.Model(&model.Partner{}).Where("id = ?", p.ID).
				Updates(map[string]any{"consumer_secret_hash": sha256Hex(plain), "consumer_secret_masked": masked}).Error
			if err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "PARTNER_SECRET_ROTATED", Entity: "partner", EntityID: idStr(p.ID),
				After: masked}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		ok(c, J{"consumerId": p.ConsumerID, "consumerSecret": plain, "consumerSecretMasked": masked})
	}
}
