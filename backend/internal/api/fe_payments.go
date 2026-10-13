package api

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"medpay_spkd/internal/audit"
	"medpay_spkd/internal/auth"
	"medpay_spkd/internal/engine"
	"medpay_spkd/internal/model"
)

// Klaim yang boleh menerima pembayaran gap.
var payableStatuses = []model.ClaimStatus{
	model.ClaimApproved, model.ClaimPartiallyApproved, model.ClaimPartiallyPaid, model.ClaimPaid,
}

// Klaim yang masih menunggu pembayaran (kandidat auto-match rekonsiliasi).
var awaitingPayment = []model.ClaimStatus{
	model.ClaimApproved, model.ClaimPartiallyApproved, model.ClaimPartiallyPaid,
}

func isPayable(s model.ClaimStatus) bool {
	for _, p := range payableStatuses {
		if p == s {
			return true
		}
	}
	return false
}

// lockClaim mengunci baris klaim (FOR UPDATE) agar perhitungan total bayar & status
// tidak balapan dengan pembayaran lain untuk klaim yang sama.
func lockClaim(tx *gorm.DB, id uuid.UUID) (*model.Claim, error) {
	var c model.Claim
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&c, "id = ?", id).Error
	return &c, err
}

// notifyFinance mengirim notifikasi in-app ke staf Finance RS terkait.
func notifyFinance(tx *gorm.DB, hospitalID, claimID uuid.UUID, template, msg string) error {
	var ids []uuid.UUID
	err := tx.Model(&model.User{}).Joins("JOIN roles ON roles.id = users.role_id").
		Where("users.hospital_id = ? AND roles.code = ?", hospitalID, model.RoleFinance).Pluck("users.id", &ids).Error
	if err != nil {
		return err
	}
	for _, uid := range ids {
		cid := claimID
		n := model.Notification{Channel: "IN_APP", Template: template, Message: msg, Status: "TERKIRIM",
			RecipientUserID: uid, ClaimID: &cid}
		if err := tx.Create(&n).Error; err != nil {
			return err
		}
	}
	return nil
}

// ── Pembayaran ───────────────────────────────────────────────────────────────

func paymentListHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, okScope := scopeOf(c)
		if !okScope {
			return
		}
		var pays []model.Payment
		err := scopedPayments(db, scope).Preload("Claim").Preload("CreatedBy").Order("id DESC").Limit(100).Find(&pays).Error
		if err != nil {
			serverError(c, err)
			return
		}
		out := []J{}
		for _, p := range pays {
			if p.Claim == nil {
				continue
			}
			cl := p.Claim
			out = append(out, J{"payment": paymentDTO(p, cl.ClaimNo), "claim": J{
				"id": idStr(cl.ID), "claimNo": cl.ClaimNo, "patientName": cl.PatientName, "sepNo": cl.SEPNo, "status": string(cl.Status),
				"gap": engine.Gap(cl), "insurerId": idPtrStr(cl.InsurerID),
			}})
		}
		ok(c, out)
	}
}

func paymentCreateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		var body struct {
			ClaimID      string `json:"claimId"`
			ReceivedDate string `json:"receivedDate"`
			Amount       int64  `json:"amount"`
			Payer        string `json:"payer"`
			TransferRef  string `json:"transferRef"`
			Notes        string `json:"notes"`
			Allocation   string `json:"allocation"`
		}
		if !bind(c, &body) {
			return
		}
		if body.Amount <= 0 {
			fail(c, http.StatusBadRequest, "Nominal pembayaran harus > 0")
			return
		}
		received, err := parseDate(body.ReceivedDate)
		if err != nil {
			fail(c, http.StatusBadRequest, err.Error())
			return
		}
		if received == nil {
			t := engine.Today()
			received = &t
		}
		allocation := strings.ToUpper(strings.TrimSpace(body.Allocation))
		if allocation == "" {
			allocation = "GAP"
		}
		if !refValid(db, model.RefPaymentAllocation, allocation) {
			fail(c, http.StatusBadRequest, "Alokasi pembayaran harus GAP atau COPAY")
			return
		}
		if len(body.Payer) > 150 || len(body.TransferRef) > 100 || len(allocation) > 20 {
			fail(c, http.StatusBadRequest, "Pembayar maks. 150, referensi maks. 100, alokasi maks. 20 karakter")
			return
		}
		found, err := findClaim(db, u, body.ClaimID, false)
		if err != nil {
			notFoundOr500(c, err, claimNotFound)
			return
		}

		var (
			p       model.Payment
			warning string
			cl      *model.Claim
			paid    int64
		)
		err = db.Transaction(func(tx *gorm.DB) error {
			var err error
			if cl, err = lockClaim(tx, found.ID); err != nil {
				return err
			}
			if !isPayable(cl.Status) {
				return errConflict("Pembayaran hanya untuk klaim yang sudah disetujui/dibayar sebagian")
			}
			before, err := paidAmount(tx, cl.ID)
			if err != nil {
				return err
			}
			gap := engine.Gap(cl)
			if gap > 0 && before+body.Amount > gap {
				warning = "Lebih bayar: total pembayaran melebihi nilai gap"
			}
			if ref := strings.TrimSpace(body.TransferRef); ref != "" {
				var dup int64
				err := scopedPayments(tx, &cl.HospitalID).Where("transfer_ref = ?", ref).Count(&dup).Error
				if err != nil {
					return err
				}
				if dup > 0 {
					warning = "Potensi duplikasi: referensi transfer sudah pernah dicatat"
				}
			}
			no, err := nextNo(tx, "payments", "payment_no", "PAY")
			if err != nil {
				return err
			}
			p = model.Payment{PaymentNo: no, ReceivedDate: *received, Amount: body.Amount, Payer: body.Payer,
				TransferRef: strings.TrimSpace(body.TransferRef), Notes: body.Notes, Allocation: allocation,
				ClaimID: cl.ID, CreatedByID: u.ID}
			if err := tx.Create(&p).Error; err != nil {
				return err
			}
			paid = before + body.Amount
			next := engine.StatusAfterPayment(cl.Status, gap, paid)
			if err := setClaimStatus(tx, cl, next, &u, "Pembayaran "+p.PaymentNo+" dicatat"); err != nil {
				return err
			}
			if err := notifyFinance(tx, cl.HospitalID, cl.ID, "PAYMENT_RECEIVED",
				fmt.Sprintf("Pembayaran diterima untuk klaim %s (%s)", cl.PatientName, p.PaymentNo)); err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "PAYMENT_RECORD", Entity: "claim", EntityID: cl.ClaimNo,
				After: fmt.Sprintf("%s → %d (%s)", p.PaymentNo, p.Amount, p.Payer), HospitalID: &cl.HospitalID}, metaOf(c))
		})
		if err != nil {
			txError(c, err)
			return
		}
		p.CreatedBy = &u
		created(c, J{"payment": paymentDTO(p, cl.ClaimNo), "warning": warning, "claimStatus": string(cl.Status),
			"totalPaid": paid, "gap": engine.Gap(cl)})
	}
}

func paymentDeleteHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		var p model.Payment
		pid, _ := parseID(c.Param("id"))
		if err := db.Preload("Claim").Where("id = ?", pid).First(&p).Error; err != nil {
			notFoundOr500(c, err, "Pembayaran tidak ditemukan")
			return
		}
		if p.Claim == nil || !auth.CanAccessHospital(u, p.Claim.HospitalID) {
			fail(c, http.StatusNotFound, "Pembayaran tidak ditemukan")
			return
		}
		if p.BankLineID != nil {
			fail(c, http.StatusConflict, "Pembayaran berasal dari rekonsiliasi bank — batalkan lewat menu Rekonsiliasi (undo konfirmasi)")
			return
		}
		var cl *model.Claim
		err := db.Transaction(func(tx *gorm.DB) error {
			var err error
			if cl, err = lockClaim(tx, p.ClaimID); err != nil {
				return err
			}
			if err := tx.Delete(&model.Payment{}, "id = ?", p.ID).Error; err != nil {
				return err
			}
			paid, err := paidAmount(tx, cl.ID)
			if err != nil {
				return err
			}
			if err := setClaimStatus(tx, cl, engine.StatusAfterUndo(cl.Status, paid), &u, "Pembayaran "+p.PaymentNo+" dibatalkan"); err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "PAYMENT_UNDO", Entity: "claim", EntityID: cl.ClaimNo,
				Before: fmt.Sprintf("%s %d", p.PaymentNo, p.Amount), After: "status → " + string(cl.Status),
				HospitalID: &cl.HospitalID}, metaOf(c))
		})
		if err != nil {
			txError(c, err)
			return
		}
		ok(c, J{"ok": true, "claimStatus": string(cl.Status)})
	}
}

// ── Mutasi bank & rekonsiliasi ───────────────────────────────────────────────

func bankImportListHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, okScope := scopeOf(c)
		if !okScope {
			return
		}
		var imps []model.BankImport
		if err := whereHospital(db, "hospital_id", scope).Order("id").Find(&imps).Error; err != nil {
			serverError(c, err)
			return
		}
		ok(c, mapSlice(imps, bankImportDTO))
	}
}

// matchCandidates memuat klaim RS dengan status tertentu untuk pencocokan mutasi.
func matchCandidates(db *gorm.DB, hospitalID uuid.UUID, statuses []model.ClaimStatus) ([]model.Claim, error) {
	var claims []model.Claim
	err := db.Where("hospital_id = ? AND status IN ?", hospitalID, statuses).Order("id").Find(&claims).Error
	return claims, err
}

// autoMatch mencari klaim terbaik untuk satu baris mutasi.
func autoMatch(line *model.BankLine, claims []model.Claim) {
	var best *model.Claim
	bestScore, bestReason := 0, ""
	for i := range claims {
		score, reason := engine.MatchScore(line.Description, line.Amount, &claims[i], engine.MatchTolerance)
		if score > bestScore {
			best, bestScore, bestReason = &claims[i], score, reason
		}
	}
	if best == nil {
		return
	}
	line.MatchScore = bestScore
	line.MatchNote = fmt.Sprintf("skor %d (%s)", bestScore, bestReason)
	switch {
	case bestScore >= engine.MatchAutoScore:
		id := best.ID
		line.MatchResult, line.MatchClaimID, line.MatchNote = model.MatchMatched, &id, "auto-match: "+bestReason
	case bestScore >= engine.MatchReviewScore:
		line.MatchResult = model.MatchNeedsReview
	}
}

func bankImportCreateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		var body struct {
			BankName    string `json:"bankName"`
			PeriodLabel string `json:"periodLabel"`
			HospitalID  string `json:"hospitalId"`
			Lines       []struct {
				ValueDate   string `json:"valueDate"`
				Description string `json:"description"`
				Amount      int64  `json:"amount"`
				Reference   string `json:"reference"`
			} `json:"lines"`
		}
		if !bind(c, &body) {
			return
		}
		bank := strings.TrimSpace(body.BankName)
		if bank == "" || len(bank) > 150 || len(body.PeriodLabel) > 100 {
			fail(c, http.StatusBadRequest, "Nama bank wajib diisi (maks. 150 karakter)")
			return
		}
		if len(body.Lines) == 0 || len(body.Lines) > 5000 {
			fail(c, http.StatusBadRequest, "Jumlah baris mutasi harus 1–5000")
			return
		}
		hid, valid := resolveHospital(c, db, u, body.HospitalID)
		if !valid {
			return
		}

		lines := make([]model.BankLine, 0, len(body.Lines))
		for i, ln := range body.Lines {
			vd, err := parseDate(ln.ValueDate)
			if err != nil || vd == nil {
				fail(c, http.StatusBadRequest, fmt.Sprintf("Baris %d: tanggal valuta wajib berformat YYYY-MM-DD", i+1))
				return
			}
			if len(ln.Reference) > 100 {
				fail(c, http.StatusBadRequest, fmt.Sprintf("Baris %d: referensi maks. 100 karakter", i+1))
				return
			}
			lines = append(lines, model.BankLine{ValueDate: *vd, Description: ln.Description, Amount: ln.Amount,
				Reference: strings.TrimSpace(ln.Reference), MatchResult: model.MatchUnmatched})
		}
		claims, err := matchCandidates(db, hid, awaitingPayment)
		if err != nil {
			serverError(c, err)
			return
		}
		matched, review, unmatched := 0, 0, 0
		for i := range lines {
			autoMatch(&lines[i], claims)
			switch lines[i].MatchResult {
			case model.MatchMatched:
				matched++
			case model.MatchNeedsReview:
				review++
			default:
				unmatched++
			}
		}

		imp := model.BankImport{BankName: bank, PeriodLabel: body.PeriodLabel, LineCount: len(lines),
			HospitalID: hid, ImportedByID: u.ID}
		err = db.Transaction(func(tx *gorm.DB) error {
			no, err := nextNo(tx, "bank_imports", "import_no", "BI")
			if err != nil {
				return err
			}
			imp.ImportNo = no
			if err := tx.Create(&imp).Error; err != nil {
				return err
			}
			last, err := lastNo(tx, "bank_lines", "line_no")
			if err != nil {
				return err
			}
			for i := range lines {
				lines[i].BankImportID = imp.ID
				lines[i].LineNo = formatNo("BL", last+int64(i)+1)
			}
			if err := tx.CreateInBatches(&lines, 500).Error; err != nil {
				return err
			}
			if review > 0 {
				// F-WQ-04: mutasi yang perlu review → tugas rekonsiliasi untuk Finance.
				title := fmt.Sprintf("Impor %s: %d baris mutasi perlu review rekonsiliasi", imp.ImportNo, review)
				if err := autoTask(tx, hid, nil, "RECONCILIATION", model.PrioritySedang, title, engine.Today().AddDate(0, 0, 2)); err != nil {
					return err
				}
			}
			return audit.Log(tx, &u, audit.Entry{Action: "IMPORT_BANK_STATEMENT", Entity: "bank_import", EntityID: imp.ImportNo,
				After:      fmt.Sprintf("%s — %d baris (auto: %d match, %d review, %d unmatched)", bank, len(lines), matched, review, unmatched),
				HospitalID: &hid}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		created(c, J{"import": bankImportDTO(imp), "matched": matched, "needsReview": review, "unmatched": unmatched})
	}
}

// findBankLine mencari baris mutasi (UUID) dalam cakupan RS user.
func findBankLine(db *gorm.DB, u model.User, id string) (*model.BankLine, error) {
	lid, valid := parseID(id)
	if !valid {
		return nil, gorm.ErrRecordNotFound
	}
	var l model.BankLine
	if err := db.Preload("BankImport").Preload("MatchClaim").Where("id = ?", lid).First(&l).Error; err != nil {
		return nil, err
	}
	if l.BankImport == nil || !auth.CanAccessHospital(u, l.BankImport.HospitalID) {
		return nil, gorm.ErrRecordNotFound
	}
	return &l, nil
}

func paidByClaims(db *gorm.DB, ids []uuid.UUID) (map[uuid.UUID]int64, error) {
	out := map[uuid.UUID]int64{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		ClaimID uuid.UUID
		Total   int64
	}
	err := db.Model(&model.Payment{}).Select("claim_id, SUM(amount) AS total").
		Where("claim_id IN ?", ids).Group("claim_id").Scan(&rows).Error
	for _, r := range rows {
		out[r.ClaimID] = r.Total
	}
	return out, err
}

func bankLineListHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, okScope := scopeOf(c)
		if !okScope {
			return
		}
		q := db.Preload("BankImport").Preload("MatchClaim").Joins("JOIN bank_imports ON bank_imports.id = bank_lines.bank_import_id AND bank_imports.deleted_at IS NULL")
		q = whereHospital(q, "bank_imports.hospital_id", scope)
		if imp := c.Query("importId"); imp != "" {
			iid, valid := parseID(imp)
			if !valid {
				fail(c, http.StatusBadRequest, "importId tidak valid")
				return
			}
			q = q.Where("bank_imports.id = ?", iid)
		}
		var lines []model.BankLine
		if err := q.Order("bank_lines.id").Find(&lines).Error; err != nil {
			serverError(c, err)
			return
		}
		var ids []uuid.UUID
		for _, l := range lines {
			if l.MatchClaim != nil {
				ids = append(ids, l.MatchClaim.ID)
			}
		}
		paid, err := paidByClaims(db, ids)
		if err != nil {
			serverError(c, err)
			return
		}
		out := []J{}
		for _, l := range lines {
			row := J{"line": bankLineDTO(l, l.BankImport.ImportNo)}
			if cl := l.MatchClaim; cl != nil {
				row["claim"] = J{"id": idStr(cl.ID), "claimNo": cl.ClaimNo, "patientName": cl.PatientName, "sepNo": cl.SEPNo,
					"status": string(cl.Status), "gap": engine.Gap(cl), "paid": paid[cl.ID]}
			}
			out = append(out, row)
		}
		ok(c, out)
	}
}

func bankLineCandidatesHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		l, err := findBankLine(db, currentUser(c), c.Param("id"))
		if err != nil {
			notFoundOr500(c, err, "Baris mutasi tidak ditemukan")
			return
		}
		claims, err := matchCandidates(db, l.BankImport.HospitalID, payableStatuses)
		if err != nil {
			serverError(c, err)
			return
		}
		ids := make([]uuid.UUID, len(claims))
		for i := range claims {
			ids[i] = claims[i].ID
		}
		paid, err := paidByClaims(db, ids)
		if err != nil {
			serverError(c, err)
			return
		}
		type cand struct {
			ClaimID     string `json:"claimId"`
			ClaimNo     string `json:"claimNo"`
			PatientName string `json:"patientName"`
			SEPNo       string `json:"sepNo"`
			Status      string `json:"status"`
			Gap         int64  `json:"gap"`
			Paid        int64  `json:"paid"`
			Score       int    `json:"score"`
			Reason      string `json:"reason"`
		}
		out := []cand{}
		for i := range claims {
			cl := &claims[i]
			score, reason := engine.MatchScore(l.Description, l.Amount, cl, engine.MatchTolerance)
			if score >= engine.MatchMinCandidate {
				out = append(out, cand{idStr(cl.ID), cl.ClaimNo, cl.PatientName, cl.SEPNo, string(cl.Status), engine.Gap(cl), paid[cl.ID], score, reason})
			}
		}
		sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
		ok(c, out)
	}
}

func bankLineConfirmHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		var body struct {
			ClaimID string `json:"claimId"`
			Force   bool   `json:"force"`
		}
		if !bind(c, &body) {
			return
		}
		l, err := findBankLine(db, u, c.Param("id"))
		if err != nil {
			notFoundOr500(c, err, "Baris mutasi tidak ditemukan")
			return
		}
		if l.Confirmed {
			fail(c, http.StatusConflict, "Baris mutasi ini sudah terkonfirmasi")
			return
		}
		if l.Amount <= 0 {
			fail(c, http.StatusConflict, "Mutasi bernominal ≤ 0 tidak dapat dikonfirmasi sebagai pembayaran")
			return
		}
		var target model.Claim
		targetID, _ := parseID(body.ClaimID)
		err = db.Where("id = ? AND hospital_id = ?", targetID, l.BankImport.HospitalID).First(&target).Error
		if err != nil {
			notFoundOr500(c, err, claimNotFound)
			return
		}

		var (
			pay      model.Payment
			cl       *model.Claim
			warnings []string
		)
		err = db.Transaction(func(tx *gorm.DB) error {
			var err error
			if cl, err = lockClaim(tx, target.ID); err != nil {
				return err
			}
			if !isPayable(cl.Status) {
				return errConflict("Pembayaran hanya untuk klaim yang sudah disetujui/dibayar sebagian")
			}
			// Kunci baris mutasi agar tidak dikonfirmasi dua kali bersamaan.
			var line model.BankLine
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&line, "id = ?", l.ID).Error; err != nil {
				return err
			}
			if line.Confirmed {
				return errConflict("Baris mutasi ini sudah terkonfirmasi")
			}
			paid, err := paidAmount(tx, cl.ID)
			if err != nil {
				return err
			}
			gap := engine.Gap(cl)
			if paid > 0 {
				warnings = append(warnings, "Klaim sudah memiliki pembayaran tercatat — potensi pembayaran ganda")
			}
			if diff := gap - line.Amount; diff > 0 {
				warnings = append(warnings, "Kurang bayar: nominal mutasi "+rupiah(diff)+" di bawah nilai gap")
			} else if diff < 0 {
				warnings = append(warnings, "Lebih bayar: nominal mutasi "+rupiah(-diff)+" di atas nilai gap")
			}
			if len(warnings) > 0 && !body.Force {
				return errNeedsForce
			}

			no, err := nextNo(tx, "payments", "payment_no", "PAY")
			if err != nil {
				return err
			}
			lineID := line.ID
			pay = model.Payment{PaymentNo: no, ReceivedDate: line.ValueDate, Amount: line.Amount, Payer: "Mutasi Bank",
				TransferRef: line.Reference, Notes: "Konfirmasi rekonsiliasi " + line.LineNo, Allocation: "GAP",
				ClaimID: cl.ID, CreatedByID: u.ID, BankLineID: &lineID}
			if err := tx.Create(&pay).Error; err != nil {
				return err
			}
			now := time.Now()
			err = tx.Model(&line).Updates(map[string]any{
				"confirmed": true, "confirmed_at": now, "confirmed_by_id": u.ID,
				"match_result": model.MatchMatched, "match_claim_id": cl.ID,
			}).Error
			if err != nil {
				return err
			}
			next := engine.StatusAfterPayment(cl.Status, gap, paid+line.Amount)
			if err := setClaimStatus(tx, cl, next, &u, "Rekonsiliasi "+line.LineNo+" → "+pay.PaymentNo); err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "RECON_CONFIRM", Entity: "bank_line", EntityID: line.LineNo,
				After: fmt.Sprintf("→ %s / %s", cl.ClaimNo, pay.PaymentNo), HospitalID: &cl.HospitalID}, metaOf(c))
		})
		if errors.Is(err, errNeedsForce) {
			c.AbortWithStatusJSON(http.StatusConflict, J{"error": "Perlu konfirmasi: " + strings.Join(warnings, "; "), "warnings": warnings})
			return
		}
		if err != nil {
			txError(c, err)
			return
		}
		pay.CreatedBy = &u
		ok(c, J{"ok": true, "payment": paymentDTO(pay, cl.ClaimNo), "claimStatus": string(cl.Status)})
	}
}

func bankLineUnconfirmHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		l, err := findBankLine(db, u, c.Param("id"))
		if err != nil {
			notFoundOr500(c, err, "Baris mutasi tidak ditemukan")
			return
		}
		if !l.Confirmed || l.MatchClaimID == nil {
			fail(c, http.StatusConflict, "Baris ini tidak sedang terkonfirmasi")
			return
		}
		err = db.Transaction(func(tx *gorm.DB) error {
			cl, err := lockClaim(tx, *l.MatchClaimID)
			if err != nil {
				return err
			}
			var removed []string
			if err := tx.Model(&model.Payment{}).Where("bank_line_id = ?", l.ID).Pluck("payment_no", &removed).Error; err != nil {
				return err
			}
			if err := tx.Where("bank_line_id = ?", l.ID).Delete(&model.Payment{}).Error; err != nil {
				return err
			}
			err = tx.Model(&model.BankLine{}).Where("id = ?", l.ID).Updates(map[string]any{
				"confirmed": false, "confirmed_at": nil, "confirmed_by_id": nil, "match_note": "undo oleh " + u.Name,
			}).Error
			if err != nil {
				return err
			}
			paid, err := paidAmount(tx, cl.ID)
			if err != nil {
				return err
			}
			if err := setClaimStatus(tx, cl, engine.StatusAfterUndo(cl.Status, paid), &u, "Rekonsiliasi "+l.LineNo+" dibatalkan"); err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "RECON_UNDO", Entity: "bank_line", EntityID: l.LineNo,
				Before: strings.Join(removed, ","), After: "dibatalkan", HospitalID: &cl.HospitalID}, metaOf(c))
		})
		if err != nil {
			txError(c, err)
			return
		}
		ok(c, J{"ok": true})
	}
}

// ── Error transaksi ──────────────────────────────────────────────────────────

// conflictError dipakai untuk membatalkan transaksi dengan respons 409.
type conflictError struct{ msg string }

func (e conflictError) Error() string { return e.msg }

func errConflict(msg string) error { return conflictError{msg} }

var errNeedsForce = errors.New("perlu konfirmasi")

func txError(c *gin.Context, err error) {
	var ce conflictError
	switch {
	case errors.As(err, &ce):
		fail(c, http.StatusConflict, ce.msg)
	case errors.Is(err, gorm.ErrRecordNotFound):
		fail(c, http.StatusNotFound, "Data tidak ditemukan")
	default:
		serverError(c, err)
	}
}
