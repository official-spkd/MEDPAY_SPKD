package api

import (
	"fmt"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"medpay_spkd/internal/auth"
	"medpay_spkd/internal/engine"
	"medpay_spkd/internal/model"
)

// ── Cakupan data per RS untuk tabel tanpa kolom hospital_id ───────────────────

func scopedWorkItems(db *gorm.DB, scope *uuid.UUID) *gorm.DB {
	return whereHospital(db.Model(&model.WorkItem{}), "work_items.hospital_id", scope)
}

// scopedAudit: kejadian milik RS, ditambah panggilan Engine API tanpa RS (mis. ditolak
// saat autentikasi) dari partner yang melayani RS tersebut.
func scopedAudit(db *gorm.DB, scope *uuid.UUID) *gorm.DB {
	q := db.Model(&model.AuditLog{})
	if scope == nil {
		return q
	}
	h := *scope
	return q.Where(`(audit_logs.hospital_id = ? OR (audit_logs.hospital_id IS NULL AND audit_logs.scope = ?
		AND audit_logs.partner_id IN (SELECT partner_id FROM partner_hospitals WHERE hospital_id = ?)))`,
		h, model.AuditScopeEngineAPI, h)
}

func workItemPreloads(q *gorm.DB) *gorm.DB {
	return q.Preload("Claim").Preload("Comments.Author").Preload("Histories.Actor")
}

func scopedNotifications(db *gorm.DB, scope *uuid.UUID) *gorm.DB {
	q := db.Model(&model.Notification{})
	if scope == nil {
		return q
	}
	h := *scope
	return q.Where(`(notifications.recipient_user_id IN (SELECT id FROM users WHERE hospital_id = ?)
		OR notifications.claim_id IN (SELECT id FROM claims WHERE hospital_id = ?))`, h, h)
}

func scopedPayments(db *gorm.DB, scope *uuid.UUID) *gorm.DB {
	q := db.Model(&model.Payment{})
	if scope == nil {
		return q
	}
	return q.Where("payments.claim_id IN (SELECT id FROM claims WHERE hospital_id = ?)", *scope)
}

// ── GET /bootstrap ───────────────────────────────────────────────────────────

func bootstrapHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		scope := u.HospitalID

		var hospitals []model.Hospital
		var users []model.User
		var insurers []model.Insurer
		var rejections []model.RejectionCode
		var rules []model.ScrubRule
		var regs []model.Regulation
		var notifs []model.Notification
		var cats []model.ResumeCategory
		var claims []model.Claim
		var items []model.WorkItem
		queries := []func() error{
			func() error { return whereHospital(db, "id", scope).Order("id").Find(&hospitals).Error },
			func() error {
				return whereHospital(db.Preload("Role"), "hospital_id", scope).Order("id").Find(&users).Error
			},
			func() error { return db.Order("name").Find(&insurers).Error },
			func() error { return db.Order("id").Find(&rejections).Error },
			func() error { return db.Order("code").Find(&rules).Error },
			func() error { return db.Preload("Reviewer").Order("id").Find(&regs).Error },
			func() error {
				return scopedNotifications(db, scope).Preload("Claim").Preload("RecipientUser").
					Order("created_at DESC").Limit(50).Find(&notifs).Error
			},
			func() error { return db.Order("id").Find(&cats).Error },
			func() error { return whereHospital(db, "hospital_id", scope).Order("id").Find(&claims).Error },
			func() error { return scopedWorkItems(db, scope).Preload("Claim").Order("id").Find(&items).Error },
		}
		for _, q := range queries {
			if err := q(); err != nil {
				serverError(c, err)
				return
			}
		}

		out := J{
			"hospitals": mapSlice(hospitals, hospitalDTO), "users": mapSlice(users, userDTO),
			"insurers": mapSlice(insurers, insurerDTO), "scrubRules": mapSlice(rules, scrubRuleDTO),
			"regulations": mapSlice(regs, regulationDTO), "notifications": mapSlice(notifs, notificationDTO),
			"claimLabels": refLabels(db, model.RefClaimStatus), "componentLabels": refLabels(db, model.RefTariffComponent),
			"specialCmgLabels": refLabels(db, model.RefSpecialCMG), "partners": []J{},
		}
		rc := []J{}
		for _, r := range rejections {
			rc = append(rc, J{"code": r.Code, "label": r.Label})
		}
		out["rejectionCodes"] = rc
		cs := []J{}
		for _, cat := range cats {
			cs = append(cs, J{"Key": cat.Code, "Label": cat.Label})
		}
		out["resumeCategories"] = cs
		cb := []J{}
		for i := range claims {
			cb = append(cb, claimBrief(&claims[i]))
		}
		out["claims"] = cb
		wb := []J{}
		for _, w := range items {
			var assignee, role any
			claimNo := ""
			if w.Claim != nil {
				claimNo = w.Claim.ClaimNo
			}
			if w.AssigneeID != nil {
				assignee = idStr(*w.AssigneeID)
			}
			if w.Role != "" {
				role = w.Role
			}
			wb = append(wb, J{"id": idStr(w.ID), "itemNo": w.ItemNo, "title": w.Title, "status": w.Status, "assignee": assignee,
				"role": role, "priority": w.Priority, "claimId": idPtrOrNil(w.ClaimID), "claimNo": claimNo})
		}
		out["workItems"] = wb

		reference, err := referencePayload(db)
		if err != nil {
			serverError(c, err)
			return
		}
		out["reference"] = reference

		if auth.IsSuperAdmin(u) {
			var partners []model.Partner
			if err := db.Preload("WhitelistIPs").Preload("Hospitals").Preload("Keys").Order("id").Find(&partners).Error; err != nil {
				serverError(c, err)
				return
			}
			out["partners"] = mapSlice(partners, partnerDTO)
		}
		ok(c, out)
	}
}

func mapSlice[T any](xs []T, f func(T) J) []J {
	out := make([]J, 0, len(xs))
	for _, x := range xs {
		out = append(out, f(x))
	}
	return out
}

// ── Kinerja penjamin ─────────────────────────────────────────────────────────

type insurerPerf struct {
	InsurerID     string  `json:"insurerId"`
	Name          string  `json:"name"`
	Submitted     int     `json:"submitted"`
	Approved      int     `json:"approved"`
	Rejected      int     `json:"rejected"`
	ApprovalRate  float64 `json:"approvalRate"`
	RejectionRate float64 `json:"rejectionRate"`
	AvgDaysPay    *int    `json:"avgDaysPay"`
	ReceivedRp    int64   `json:"receivedRp"`
	payDays       []int
}

func insurerPerformance(d *dataset) []*insurerPerf {
	rows := map[uuid.UUID]*insurerPerf{}
	ids := make([]uuid.UUID, 0, len(d.insurers))
	for id, ins := range d.insurers {
		rows[id] = &insurerPerf{InsurerID: idStr(id), Name: ins.Name}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return rows[ids[i]].Name < rows[ids[j]].Name })

	for _, c := range d.claims {
		if c.InsurerID == nil || rows[*c.InsurerID] == nil {
			continue
		}
		row := rows[*c.InsurerID]
		switch c.Status {
		case model.ClaimSubmitted, model.ClaimApproved, model.ClaimPartiallyApproved, model.ClaimPartiallyPaid, model.ClaimPaid:
			row.Submitted++
			row.Approved++
		case model.ClaimRejected:
			row.Submitted++
			row.Rejected++
		}
		if c.Status == model.ClaimPaid && c.SubmittedAt != nil {
			if last, ok := d.lastPay[c.ID]; ok {
				row.payDays = append(row.payDays, engine.DaysBetween(*c.SubmittedAt, last))
			}
		}
		row.ReceivedRp += d.paid[c.ID]
	}

	out := []*insurerPerf{}
	for _, id := range ids {
		row := rows[id]
		if row.Submitted == 0 && row.ReceivedRp == 0 {
			continue
		}
		if row.Submitted > 0 {
			row.ApprovalRate = float64(row.Approved) / float64(row.Submitted) * 100
			row.RejectionRate = float64(row.Rejected) / float64(row.Submitted) * 100
		}
		if n := len(row.payDays); n > 0 {
			sum := 0
			for _, v := range row.payDays {
				sum += v
			}
			avg := sum / n
			row.AvgDaysPay = &avg
		}
		out = append(out, row)
	}
	return out
}

func paidPairs(d *dataset) []engine.PaidPair {
	var pairs []engine.PaidPair
	for _, c := range d.claims {
		if c.Status != model.ClaimPaid || c.SubmittedAt == nil {
			continue
		}
		if last, ok := d.lastPay[c.ID]; ok {
			pairs = append(pairs, engine.PaidPair{Submitted: *c.SubmittedAt, Paid: last})
		}
	}
	return pairs
}

// openReceivable: klaim dengan sisa tagihan yang masih bisa ditagih.
func openReceivable(d *dataset, c *model.Claim) (int64, bool) {
	o := d.outstanding(c)
	if o <= 0 || c.Status == model.ClaimWrittenOff || c.Status == model.ClaimRejected {
		return 0, false
	}
	return o, true
}

func agingBasis(c *model.Claim, byService bool) *time.Time {
	if byService || c.SubmittedAt == nil {
		return c.AdmissionDate
	}
	return c.SubmittedAt
}

var openStatuses = map[model.ClaimStatus]bool{
	model.ClaimDraft: true, model.ClaimReadyToSubmit: true, model.ClaimSubmitted: true,
}

// ── GET /dashboard ───────────────────────────────────────────────────────────

func dashboardHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, okScope := scopeOf(c)
		if !okScope {
			return
		}
		d, err := loadDataset(db, scope)
		if err != nil {
			serverError(c, err)
			return
		}

		var eligible, nearDeadline, blockers, resolutionsToday int
		var outsideBpjs, received int64
		breakdown := map[string]int{}
		nearClaims := []J{}
		var receivables []engine.Receivable
		var outstandingTotal int64
		today := fmtDate(d.today)

		for _, cl := range d.claims {
			breakdown[string(cl.Status)]++
			g := engine.Gap(cl)
			if cl.ParticipantType == model.ParticipantNonPBI && g > 0 && cl.Status != model.ClaimWrittenOff {
				eligible++
				outsideBpjs += g
			}
			if cl.TariffResolvedAt != nil && fmtDate(*cl.TariffResolvedAt) == today {
				resolutionsToday++
			}
			di := engine.Deadline(cl.AdmissionDate, d.today)
			if openStatuses[cl.Status] && di.Near() {
				nearDeadline++
				if len(nearClaims) < 8 {
					nearClaims = append(nearClaims, J{"id": idStr(cl.ID), "claimNo": cl.ClaimNo, "patientName": cl.PatientName,
						"status": string(cl.Status), "deadline": fmtDatePtr(di.Deadline), "daysLeft": di.DaysLeft, "zone": di.Zone})
				}
			}
			if cl.Status != model.ClaimPaid && cl.Status != model.ClaimWrittenOff && d.scrub(cl).HasOpenBlocker {
				blockers++
			}
			if o, open := openReceivable(d, cl); open {
				if b := agingBasis(cl, false); b != nil {
					receivables = append(receivables, engine.Receivable{Amount: o, Basis: *b})
				}
				outstandingTotal += o
			}
			received += d.paid[cl.ID]
		}
		buckets, _ := engine.AgingBuckets(receivables, d.today)

		var queue []model.WorkItem
		err = workItemPreloads(scopedWorkItems(db, scope)).
			Where("work_items.status IN ?", []string{model.WorkItemNew, model.WorkItemAssigned, model.WorkItemInProgress}).
			Order("due_date, id").Find(&queue).Error
		if err != nil {
			serverError(c, err)
			return
		}
		var recent []model.Payment
		if err := scopedPayments(db, scope).Preload("Claim").Preload("CreatedBy").Order("id DESC").Limit(6).Find(&recent).Error; err != nil {
			serverError(c, err)
			return
		}
		rp := []J{}
		for _, p := range recent {
			rp = append(rp, paymentDTO(p, claimNoOf(p.Claim)))
		}

		ok(c, J{
			"eligibleClaims": eligible, "outsideBpjsRp": outsideBpjs, "receivedRp": received,
			"outstandingRp": outstandingTotal, "resolutionsToday": resolutionsToday, "nearDeadline": nearDeadline,
			"blockersCount": blockers, "statusBreakdown": breakdown, "nearDeadlineClaims": nearClaims,
			"workQueueToday": mapSlice(queue, workItemDTO), "agingBuckets": buckets,
			"dso": engine.DSO(paidPairs(d)), "insurerPerf": insurerPerformance(d), "recentPayments": rp,
		})
	}
}

func claimNoOf(c *model.Claim) string {
	if c == nil {
		return ""
	}
	return c.ClaimNo
}

// ── GET /analytics ───────────────────────────────────────────────────────────

func analyticsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, okScope := scopeOf(c)
		if !okScope {
			return
		}
		d, err := loadDataset(db, scope)
		if err != nil {
			serverError(c, err)
			return
		}

		type trendRow struct {
			Month  string `json:"month"`
			Claims int    `json:"claims"`
			Gap    int64  `json:"gap"`
		}
		now := time.Now().In(engine.Location())
		trend := make([]*trendRow, 12)
		idx := map[string]int{}
		for k := 11; k >= 0; k-- {
			m := time.Date(now.Year(), now.Month()-time.Month(k), 1, 0, 0, 0, 0, engine.Location()).Format("2006-01")
			trend[11-k] = &trendRow{Month: m}
			idx[m] = 11 - k
		}

		type careStats struct {
			claims, below int
			gap, topUp    int64
			groups        map[string]bool
		}
		ri := &careStats{groups: map[string]bool{}}
		rj := &careStats{groups: map[string]bool{}}
		segments := map[string]int{}
		window := []J{}

		for _, cl := range d.claims {
			g := engine.Gap(cl)
			if i, ok := idx[cl.CreatedAt.In(engine.Location()).Format("2006-01")]; ok {
				trend[i].Claims++
				trend[i].Gap += g
			}
			hosp := engine.SumTariff(cl.HospitalTariff)
			t := rj
			if cl.CareType == model.CareTypeRawatInap {
				t = ri
			}
			t.claims++
			t.gap += g
			t.topUp += engine.SumCMG(cl.SpecialCMG)
			t.groups[cl.INACBGCode] = true
			if hosp > 0 && hosp < engine.TotalJKN(cl.INACBGBaseTariff, cl.SpecialCMG) {
				t.below++
			}
			seg := "PBI (tidak eligible)"
			if cl.ParticipantType == model.ParticipantNonPBI {
				seg = fmt.Sprintf("Kelas %s Non-PBI", cl.CareClass)
			}
			segments[seg]++

			di := engine.Deadline(cl.AdmissionDate, d.today)
			if openStatuses[cl.Status] && di.Deadline != nil && di.DaysLeft <= 30 && di.DaysLeft >= -365 {
				window = append(window, J{"id": idStr(cl.ID), "claimNo": cl.ClaimNo, "patientName": cl.PatientName,
					"deadline": fmtDatePtr(di.Deadline), "daysLeft": di.DaysLeft, "critical": di.DaysLeft <= 7})
			}
		}
		fin := func(t *careStats) J {
			return J{"claims": t.claims, "gap": t.gap, "topUp": t.topUp, "uniqueGroups": len(t.groups), "belowInacbg": t.below}
		}
		ok(c, J{"trend": trend, "inpatient": fin(ri), "outpatient": fin(rj), "segments": segments, "deadlineWindow": window})
	}
}

// ── GET /aging ───────────────────────────────────────────────────────────────

type agingRow struct {
	ID          string `json:"id"`
	ClaimNo     string `json:"claimNo"`
	PatientName string `json:"patientName"`
	SEPNo       string `json:"sepNo"`
	Insurer     string `json:"insurer"`
	Status      string `json:"status"`
	Amount      int64  `json:"amount"`
	AgeDays     int    `json:"ageDays"`
}

func agingRows(d *dataset, byService bool) ([]agingRow, []engine.Receivable) {
	rows := []agingRow{}
	var recv []engine.Receivable
	for _, cl := range d.claims {
		o, open := openReceivable(d, cl)
		if !open {
			continue
		}
		b := agingBasis(cl, byService)
		if b == nil {
			continue
		}
		recv = append(recv, engine.Receivable{Amount: o, Basis: *b})
		rows = append(rows, agingRow{ID: idStr(cl.ID), ClaimNo: cl.ClaimNo, PatientName: cl.PatientName, SEPNo: cl.SEPNo,
			Insurer: d.insurerName(cl), Status: string(cl.Status), Amount: o, AgeDays: engine.DaysBetween(*b, d.today)})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].AgeDays > rows[j].AgeDays })
	return rows, recv
}

func agingHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, okScope := scopeOf(c)
		if !okScope {
			return
		}
		d, err := loadDataset(db, scope)
		if err != nil {
			serverError(c, err)
			return
		}
		rows, recv := agingRows(d, c.Query("basis") == "service")
		buckets, total := engine.AgingBuckets(recv, d.today)
		var over90 int64
		for _, b := range buckets {
			if b.Label == engine.BucketOver90 {
				over90 = b.Amount
			}
		}
		pct := 0.0
		if total > 0 {
			pct = float64(over90) / float64(total) * 100
		}
		ok(c, J{"buckets": buckets, "total": total, "dso": engine.DSO(paidPairs(d)), "pctOver90": pct,
			"drilldown": rows, "payerPerf": insurerPerformance(d)})
	}
}

// ── POST /kapj/simulate ──────────────────────────────────────────────────────

func kapjSimulateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Params engine.KAPJParams `json:"params"`
		}
		if !bind(c, &body) {
			return
		}
		scope, okScope := scopeOf(c)
		if !okScope {
			return
		}
		p := body.Params
		if p.BPJSSharePct <= 0 {
			// Parameter kosong: pakai pengaturan KAPJ RS (atau default nasional).
			p = engine.DefaultKAPJ
			if scope != nil {
				var s model.KAPJSetting
				if err := db.Where("hospital_id = ?", *scope).First(&s).Error; err == nil {
					p = engine.KAPJFromSetting(s)
				}
			}
		}
		if p.BPJSSharePct > 100 || p.PatientCoPayPct < 0 || p.PatientCoPayPct > 100 ||
			p.OutpatientCoPayCap < 0 || p.InpatientCoPayCap < 0 {
			fail(c, 400, "Parameter KAPJ tidak valid (persentase 0–100, cap tidak boleh negatif)")
			return
		}

		d, err := loadDataset(db, scope)
		if err != nil {
			serverError(c, err)
			return
		}
		rows := []J{}
		var totals engine.KAPJSplit
		for _, cl := range d.claims {
			if cl.Status == model.ClaimWrittenOff || cl.Status == model.ClaimRejected {
				continue
			}
			bill := engine.SumTariff(cl.HospitalTariff)
			jkn := engine.TotalJKN(cl.INACBGBaseTariff, cl.SpecialCMG)
			if bill <= 0 && jkn <= 0 {
				continue
			}
			k := engine.SplitKAPJ(bill, jkn, cl.PolicyCeiling, cl.CareType, p)
			rows = append(rows, J{"claimId": idStr(cl.ID), "claimNo": cl.ClaimNo, "patientName": cl.PatientName, "insurer": d.insurerName(cl),
				"totalHospitalBill": bill, "bpjsPays": k.BPJSPays, "insurerPays": k.InsurerPays,
				"patientPays": k.PatientPays, "status": k.Status})
			totals.TotalHospitalBill += bill
			totals.BPJSPays += k.BPJSPays
			totals.InsurerPays += k.InsurerPays
			totals.PatientPays += k.PatientPays
		}
		ok(c, J{"params": p, "rows": rows, "disclaimer": setting(db, model.SettingKAPJDisclaimer), "totals": J{
			"totalHospitalBill": totals.TotalHospitalBill, "bpjsPays": totals.BPJSPays,
			"insurerPays": totals.InsurerPays, "patientPays": totals.PatientPays,
		}})
	}
}
