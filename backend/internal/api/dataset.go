package api

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"medpay_spkd/internal/engine"
	"medpay_spkd/internal/model"
)

// dataset = seluruh klaim dalam cakupan RS beserta data pendukung scrub,
// pembayaran, dan penjamin. Dipakai endpoint agregat (dashboard, analitik, aging,
// daftar klaim, batch, KAPJ) supaya perhitungan cukup satu kali muat data.
type dataset struct {
	today     time.Time
	claims    []*model.Claim
	byID      map[uuid.UUID]*model.Claim
	rules     map[string]model.ScrubRule
	resume    map[uuid.UUID][]model.ResumeItem
	overrides map[uuid.UUID]map[string]bool
	paid      map[uuid.UUID]int64
	lastPay   map[uuid.UUID]time.Time
	insurers  map[uuid.UUID]*model.Insurer
	sepCount  map[string]int
	scrubbed  map[uuid.UUID]*engine.ScrubResult
}

func claimPreloads(q *gorm.DB) *gorm.DB {
	return q.Preload("SecondaryDiagnoses").Preload("Procedures").Preload("Batch").
		Preload("RejectionCode").Preload("CreatedBy").Preload("Insurer")
}

func sepKey(hospitalID uuid.UUID, sep string) string { return idStr(hospitalID) + "|" + sep }

func loadRules(db *gorm.DB) (map[string]model.ScrubRule, error) {
	var rules []model.ScrubRule
	if err := db.Order("code").Find(&rules).Error; err != nil {
		return nil, err
	}
	out := make(map[string]model.ScrubRule, len(rules))
	for _, r := range rules {
		out[r.Code] = r
	}
	return out, nil
}

// loadDataset memuat klaim dalam cakupan (scope nil = semua RS).
func loadDataset(db *gorm.DB, scope *uuid.UUID) (*dataset, error) {
	d := &dataset{
		today:     engine.Today(),
		byID:      map[uuid.UUID]*model.Claim{},
		resume:    map[uuid.UUID][]model.ResumeItem{},
		overrides: map[uuid.UUID]map[string]bool{},
		paid:      map[uuid.UUID]int64{},
		lastPay:   map[uuid.UUID]time.Time{},
		insurers:  map[uuid.UUID]*model.Insurer{},
		sepCount:  map[string]int{},
		scrubbed:  map[uuid.UUID]*engine.ScrubResult{},
	}

	var claims []model.Claim
	if err := claimPreloads(whereHospital(db, "hospital_id", scope)).Order("id").Find(&claims).Error; err != nil {
		return nil, err
	}
	for i := range claims {
		c := &claims[i]
		d.claims = append(d.claims, c)
		d.byID[c.ID] = c
		if c.SEPNo != "" && c.Status != model.ClaimWrittenOff {
			d.sepCount[sepKey(c.HospitalID, c.SEPNo)]++
		}
	}
	if len(claims) == 0 {
		var err error
		d.rules, err = loadRules(db)
		return d, err
	}

	ids := whereHospital(db.Model(&model.Claim{}).Select("id"), "hospital_id", scope)

	var err error
	if d.rules, err = loadRules(db); err != nil {
		return nil, err
	}

	var items []model.ResumeItem
	if err := db.Where("claim_id IN (?)", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	for _, it := range items {
		d.resume[it.ClaimID] = append(d.resume[it.ClaimID], it)
	}

	var ovs []model.ClaimScrubOverride
	if err := db.Preload("ScrubRule").Where("claim_id IN (?)", ids).Find(&ovs).Error; err != nil {
		return nil, err
	}
	for _, o := range ovs {
		if o.ScrubRule == nil {
			continue
		}
		if d.overrides[o.ClaimID] == nil {
			d.overrides[o.ClaimID] = map[string]bool{}
		}
		d.overrides[o.ClaimID][o.ScrubRule.Code] = true
	}

	var pays []struct {
		ClaimID uuid.UUID
		Total   int64
		Last    time.Time
	}
	err = db.Model(&model.Payment{}).
		Select("claim_id, SUM(amount) AS total, MAX(received_date) AS last").
		Where("claim_id IN (?)", ids).Group("claim_id").Scan(&pays).Error
	if err != nil {
		return nil, err
	}
	for _, p := range pays {
		d.paid[p.ClaimID] = p.Total
		d.lastPay[p.ClaimID] = p.Last
	}

	var insurers []model.Insurer
	if err := db.Find(&insurers).Error; err != nil {
		return nil, err
	}
	for i := range insurers {
		d.insurers[insurers[i].ID] = &insurers[i]
	}
	return d, nil
}

// scrub menghitung (dan menyimpan sementara) hasil scrub sebuah klaim.
func (d *dataset) scrub(c *model.Claim) engine.ScrubResult {
	if r, ok := d.scrubbed[c.ID]; ok {
		return *r
	}
	dup := false
	if c.SEPNo != "" {
		n := d.sepCount[sepKey(c.HospitalID, c.SEPNo)]
		if c.Status != model.ClaimWrittenOff {
			n-- // klaim ini sendiri
		}
		dup = n > 0
	}
	var ins *model.Insurer
	if c.InsurerID != nil {
		ins = d.insurers[*c.InsurerID]
	}
	r := engine.Scrub(c, engine.ScrubContext{
		Rules: d.rules, Overrides: d.overrides[c.ID], Resume: d.resume[c.ID],
		DuplicateSEP: dup, Insurer: ins, Today: d.today,
	})
	d.scrubbed[c.ID] = &r
	return r
}

func (d *dataset) outstanding(c *model.Claim) int64 {
	if o := engine.Gap(c) - d.paid[c.ID]; o > 0 {
		return o
	}
	return 0
}

func (d *dataset) insurerName(c *model.Claim) string {
	if c.InsurerID != nil {
		if i := d.insurers[*c.InsurerID]; i != nil {
			return i.Name
		}
	}
	return ""
}

// ── Satu klaim ───────────────────────────────────────────────────────────────

// findClaim mencari klaim berdasarkan ID (UUID) dalam cakupan RS user.
// Klaim RS lain diperlakukan sebagai tidak ditemukan.
func findClaim(db *gorm.DB, u model.User, id string, withHistory bool) (*model.Claim, error) {
	cid, valid := parseID(id)
	if !valid {
		return nil, gorm.ErrRecordNotFound
	}
	q := claimPreloads(db).Where("claims.id = ?", cid)
	if u.HospitalID != nil {
		q = q.Where("hospital_id = ?", *u.HospitalID)
	}
	if withHistory {
		q = q.Preload("StatusHistories.Actor").Preload("Hospital")
	}
	var c model.Claim
	if err := q.First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// scrubClaim menjalankan scrubber untuk satu klaim.
func scrubClaim(db *gorm.DB, c *model.Claim) (engine.ScrubResult, error) {
	rules, err := loadRules(db)
	if err != nil {
		return engine.ScrubResult{}, err
	}
	var items []model.ResumeItem
	if err := db.Where("claim_id = ?", c.ID).Find(&items).Error; err != nil {
		return engine.ScrubResult{}, err
	}
	var ovs []model.ClaimScrubOverride
	if err := db.Preload("ScrubRule").Where("claim_id = ?", c.ID).Find(&ovs).Error; err != nil {
		return engine.ScrubResult{}, err
	}
	overrides := map[string]bool{}
	for _, o := range ovs {
		if o.ScrubRule != nil {
			overrides[o.ScrubRule.Code] = true
		}
	}
	var dup int64
	if c.SEPNo != "" {
		err := db.Model(&model.Claim{}).
			Where("hospital_id = ? AND sep_no = ? AND id <> ? AND status <> ?", c.HospitalID, c.SEPNo, c.ID, model.ClaimWrittenOff).
			Count(&dup).Error
		if err != nil {
			return engine.ScrubResult{}, err
		}
	}
	ins := c.Insurer
	if ins == nil && c.InsurerID != nil {
		var i model.Insurer
		if err := db.First(&i, "id = ?", *c.InsurerID).Error; err == nil {
			ins = &i
		}
	}
	return engine.Scrub(c, engine.ScrubContext{
		Rules: rules, Overrides: overrides, Resume: items, DuplicateSEP: dup > 0, Insurer: ins, Today: engine.Today(),
	}), nil
}

// paidAmount = total pembayaran (belum dibatalkan) untuk sebuah klaim.
func paidAmount(db *gorm.DB, claimID uuid.UUID) (int64, error) {
	var total int64
	err := db.Model(&model.Payment{}).Select("COALESCE(SUM(amount), 0)").Where("claim_id = ?", claimID).Scan(&total).Error
	return total, err
}

// setClaimStatus mengubah status klaim dan mencatat riwayatnya (actor nil = sistem).
func setClaimStatus(tx *gorm.DB, c *model.Claim, to model.ClaimStatus, actor *model.User, note string) error {
	if c.Status == to {
		return nil
	}
	from := c.Status
	if err := tx.Model(c).Update("status", to).Error; err != nil {
		return err
	}
	c.Status = to
	h := model.ClaimStatusHistory{ClaimID: c.ID, FromStatus: from, ToStatus: to, Note: note}
	if actor != nil {
		id := actor.ID
		h.ActorID = &id
	}
	return tx.Create(&h).Error
}
