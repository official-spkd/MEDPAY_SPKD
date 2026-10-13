package api

import (
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"medpay_spkd/internal/audit"
	"medpay_spkd/internal/auth"
	"medpay_spkd/internal/engine"
	"medpay_spkd/internal/model"
)

// ── Cache kode referensi & pengaturan ────────────────────────────────────────
// Data referensi jarang berubah: dimuat sekali dari DB lalu disegarkan saat diubah
// lewat endpoint admin.

type refStore struct {
	mu       sync.RWMutex
	loaded   bool
	codes    map[string][]model.RefCode // kategori → kode terurut
	settings map[string]string
}

var refs = &refStore{}

func (r *refStore) ensure(db *gorm.DB) error {
	r.mu.RLock()
	ok := r.loaded
	r.mu.RUnlock()
	if ok {
		return nil
	}
	return r.reload(db)
}

func (r *refStore) reload(db *gorm.DB) error {
	var codes []model.RefCode
	if err := db.Order("category, sort_order, id").Find(&codes).Error; err != nil {
		return err
	}
	var settings []model.AppSetting
	if err := db.Find(&settings).Error; err != nil {
		return err
	}
	byCat := map[string][]model.RefCode{}
	for _, c := range codes {
		byCat[c.Category] = append(byCat[c.Category], c)
	}
	sm := map[string]string{}
	for _, s := range settings {
		sm[s.Key] = s.Value
	}
	r.mu.Lock()
	r.codes, r.settings, r.loaded = byCat, sm, true
	r.mu.Unlock()
	return nil
}

func (r *refStore) invalidate() {
	r.mu.Lock()
	r.loaded = false
	r.mu.Unlock()
}

// refLabel = label kode referensi; kode aslinya bila tidak ditemukan.
func refLabel(db *gorm.DB, category, code string) string {
	if refs.ensure(db) != nil {
		return code
	}
	refs.mu.RLock()
	defer refs.mu.RUnlock()
	for _, c := range refs.codes[category] {
		if c.Code == code {
			return c.Label
		}
	}
	return code
}

// refValid memeriksa bahwa kode ada dan aktif di kategori referensi.
func refValid(db *gorm.DB, category, code string) bool {
	if refs.ensure(db) != nil {
		return false
	}
	refs.mu.RLock()
	defer refs.mu.RUnlock()
	for _, c := range refs.codes[category] {
		if c.Code == code && c.Active {
			return true
		}
	}
	return false
}

// refLabels = peta kode → label untuk satu kategori.
func refLabels(db *gorm.DB, category string) map[string]string {
	out := map[string]string{}
	if refs.ensure(db) != nil {
		return out
	}
	refs.mu.RLock()
	defer refs.mu.RUnlock()
	for _, c := range refs.codes[category] {
		out[c.Code] = c.Label
	}
	return out
}

func setting(db *gorm.DB, key string) string {
	if refs.ensure(db) != nil {
		return ""
	}
	refs.mu.RLock()
	defer refs.mu.RUnlock()
	return refs.settings[key]
}

func claimLabel(db *gorm.DB, s model.ClaimStatus) string {
	return refLabel(db, model.RefClaimStatus, string(s))
}

func refCodeDTO(c model.RefCode) J {
	return J{"code": c.Code, "label": c.Label, "tone": c.Tone, "description": c.Description,
		"sortOrder": c.SortOrder, "active": c.Active}
}

// claimTransitionsDTO = aksi status yang tersedia untuk klaim dan apakah user boleh menjalankannya.
func claimTransitionsDTO(db *gorm.DB, u model.User, c *model.Claim) []J {
	out := []J{}
	for _, to := range engine.NextStatuses(c.Status) {
		perm := engine.TransitionPermission(to)
		allowed, reason := auth.HasPermission(u, perm), ""
		if !allowed {
			reason = "Memerlukan izin " + perm
		} else if to == model.ClaimSubmitted && c.CreatedByID == u.ID {
			allowed, reason = false, "Maker-checker: pembuat klaim tidak boleh mengajukan klaimnya sendiri"
		}
		out = append(out, J{
			"to": string(to), "label": refLabel(db, model.RefClaimAction, string(to)),
			"statusLabel": claimLabel(db, to), "permission": perm, "allowed": allowed, "reason": reason,
			"confirm":               refDescription(db, model.RefClaimAction, string(to)),
			"tone":                  refTone(db, model.RefClaimAction, string(to)),
			"requiresRejectionCode": to == model.ClaimRejected, "final": to == model.ClaimWrittenOff,
		})
	}
	return out
}

func refFind(db *gorm.DB, category, code string) (model.RefCode, bool) {
	if refs.ensure(db) != nil {
		return model.RefCode{}, false
	}
	refs.mu.RLock()
	defer refs.mu.RUnlock()
	for _, c := range refs.codes[category] {
		if c.Code == code {
			return c, true
		}
	}
	return model.RefCode{}, false
}

func refDescription(db *gorm.DB, category, code string) string {
	c, _ := refFind(db, category, code)
	return c.Description
}

func refTone(db *gorm.DB, category, code string) string {
	c, _ := refFind(db, category, code)
	return c.Tone
}

// referencePayload = seluruh data referensi yang dibutuhkan frontend: pilihan & label,
// pengaturan teks, peran & izin, kategori resume, kode penolakan, aturan scrubber,
// serta konstanta aturan bisnis SRS (supaya frontend tidak menulis ulang angkanya).
func referencePayload(db *gorm.DB) (J, error) {
	if err := refs.ensure(db); err != nil {
		return nil, err
	}
	var roles []model.Role
	var perms []model.Permission
	var cats []model.ResumeCategory
	var rejections []model.RejectionCode
	var rules []model.ScrubRule
	for _, q := range []func() error{
		func() error { return db.Preload("Permissions").Order("id").Find(&roles).Error },
		func() error { return db.Order("id").Find(&perms).Error },
		func() error { return db.Order("id").Find(&cats).Error },
		func() error { return db.Order("id").Find(&rejections).Error },
		func() error { return db.Order("code").Find(&rules).Error },
	} {
		if err := q(); err != nil {
			return nil, err
		}
	}

	refs.mu.RLock()
	codes, labels := J{}, J{}
	for cat, list := range refs.codes {
		items, lm := []J{}, map[string]string{}
		for _, c := range list {
			items = append(items, refCodeDTO(c))
			lm[c.Code] = c.Label
		}
		codes[cat], labels[cat] = items, lm
	}
	settings := J{}
	for k, v := range refs.settings {
		settings[k] = v
	}
	refs.mu.RUnlock()

	roleOut := []J{}
	for _, r := range roles {
		p := []string{}
		for _, x := range r.Permissions {
			p = append(p, x.Code)
		}
		sort.Strings(p)
		roleOut = append(roleOut, J{"code": r.Code, "label": r.Label, "permissions": p})
	}
	permOut := []J{}
	for _, p := range perms {
		permOut = append(permOut, J{"code": p.Code, "label": p.Label})
	}
	catOut := []J{}
	for _, c := range cats {
		catOut = append(catOut, J{"code": c.Code, "label": c.Label})
	}
	rejOut := []J{}
	for _, r := range rejections {
		rejOut = append(rejOut, J{"code": r.Code, "label": r.Label})
	}

	transitions := J{}
	transPerm := J{}
	statusRows, _ := codes[model.RefClaimStatus].([]J)
	for _, row := range statusRows {
		from := model.ClaimStatus(row["code"].(string))
		next := []string{}
		for _, to := range engine.NextStatuses(from) {
			next = append(next, string(to))
			transPerm[string(to)] = engine.TransitionPermission(to)
		}
		transitions[string(from)] = next
	}

	return J{
		"codes": codes, "labels": labels, "settings": settings,
		"roles": roleOut, "permissions": permOut, "resumeCategories": catOut, "rejectionCodes": rejOut,
		"scrubRules":       mapSlice(rules, scrubRuleDTO),
		"claimTransitions": transitions, "transitionPermissions": transPerm,
		"editableStatuses": []string{string(model.ClaimDraft), string(model.ClaimReadyToSubmit), string(model.ClaimDisputed)},
		"payableStatuses":  []string{string(model.ClaimApproved), string(model.ClaimPartiallyApproved), string(model.ClaimPartiallyPaid), string(model.ClaimPaid)},
		"businessRules": J{
			"claimDeadlineMonths": 6,
			"deadlineZones":       J{"waspadaDays": 30, "kritisDays": 7},
			"readinessScore":      J{"start": 100, "blockerPenalty": 25, "warningPenalty": 8},
			"bankMatch": J{"autoMatchScore": engine.MatchAutoScore, "reviewScore": engine.MatchReviewScore,
				"candidateScore": engine.MatchMinCandidate, "amountTolerance": engine.MatchTolerance,
				"weights": J{"sep": 40, "insuredName": 25, "policyNo": 20, "amount": 15}},
			"agingBuckets":        []string{"0–30 hari", "31–60 hari", "61–90 hari", engine.BucketOver90},
			"kapjDefault":         engine.DefaultKAPJ,
			"insurerApprovalTone": J{"success": 80, "warning": 60},
			"passwordMinLength":   auth.MinPasswordLength,
			"sessionIdleMinutes":  int(auth.IdleTimeout().Minutes()),
		},
	}, nil
}

// ── GET /reference ───────────────────────────────────────────────────────────

func referenceHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		out, err := referencePayload(db)
		if err != nil {
			serverError(c, err)
			return
		}
		ok(c, out)
	}
}

// ── PUT /reference/:category/:code (Super Admin) ─────────────────────────────

func referenceUpdateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if !auth.IsSuperAdmin(u) {
			denyAccess(c, db, u, "ref_code", c.Param("category")+"/"+c.Param("code"), "Kode referensi hanya dapat diubah Super Admin SPKD")
			return
		}
		var body struct {
			Label       *string `json:"label"`
			Tone        *string `json:"tone"`
			Description *string `json:"description"`
			SortOrder   *int    `json:"sortOrder"`
			Active      *bool   `json:"active"`
		}
		if !bind(c, &body) {
			return
		}
		var rc model.RefCode
		if err := db.Where("category = ? AND code = ?", c.Param("category"), c.Param("code")).First(&rc).Error; err != nil {
			notFoundOr500(c, err, "Kode referensi tidak ditemukan")
			return
		}
		updates := map[string]any{}
		if body.Label != nil {
			l := strings.TrimSpace(*body.Label)
			if l == "" || len([]rune(l)) > 150 {
				fail(c, http.StatusBadRequest, "Label wajib diisi (maks. 150 karakter)")
				return
			}
			updates["label"] = l
		}
		if body.Tone != nil {
			if !oneOf(*body.Tone, "", "neutral", "info", "teal", "success", "warning", "danger", "outline") {
				fail(c, http.StatusBadRequest, "Tone tidak dikenal")
				return
			}
			updates["tone"] = *body.Tone
		}
		if body.Description != nil {
			updates["description"] = *body.Description
		}
		if body.SortOrder != nil {
			updates["sort_order"] = *body.SortOrder
		}
		if body.Active != nil {
			updates["active"] = *body.Active
		}
		before := rc.Label
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&rc).Updates(updates).Error; err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "UPDATE_REF_CODE", Entity: "ref_code",
				EntityID: rc.Category + "/" + rc.Code, Before: before, After: rc.Label}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		refs.invalidate()
		if err := db.First(&rc, "id = ?", rc.ID).Error; err != nil {
			serverError(c, err)
			return
		}
		ok(c, refCodeDTO(rc))
	}
}

// ── PUT /app-settings/:key (Super Admin) ─────────────────────────────────────

func appSettingUpdateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if !auth.IsSuperAdmin(u) {
			denyAccess(c, db, u, "app_setting", c.Param("key"), "Pengaturan aplikasi hanya dapat diubah Super Admin SPKD")
			return
		}
		var body struct {
			Value string `json:"value"`
		}
		if !bind(c, &body) {
			return
		}
		var st model.AppSetting
		if err := db.Where("key = ?", c.Param("key")).First(&st).Error; err != nil {
			notFoundOr500(c, err, "Pengaturan tidak ditemukan")
			return
		}
		before := st.Value
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&st).Update("value", body.Value).Error; err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "UPDATE_APP_SETTING", Entity: "app_setting",
				EntityID: st.Key, Before: before, After: body.Value}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		refs.invalidate()
		ok(c, J{"key": st.Key, "value": body.Value})
	}
}
