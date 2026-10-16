package api

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"medpay_spkd/internal/audit"
	"medpay_spkd/internal/engine"
	"medpay_spkd/internal/model"
)

// Penjamin membayar satu batch sekaligus. Pembayaran dicatat per batch lalu dibagi ke
// klaim-klaim di dalamnya (urut nomor klaim, maks. sisa gap tiap klaim), sehingga
// status klaim, aging, dan dashboard tetap bekerja per klaim.

// batchPayable: klaim di batch yang masih menunggu pembayaran (Lunas tidak ikut).
func batchPayable(s model.ClaimStatus) bool {
	return isPayable(s) && s != model.ClaimPaid
}

func remainingOf(gap, paid int64) int64 {
	if r := gap - paid; r > 0 {
		return r
	}
	return 0
}

// GET /payments/batches — batch yang punya klaim menunggu pembayaran beserta sisa tagihannya.
func batchPaymentSummaryHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, okScope := scopeOf(c)
		if !okScope {
			return
		}
		var batches []model.Batch
		if err := whereHospital(db.Preload("CreatedBy"), "hospital_id", scope).Order("id").Find(&batches).Error; err != nil {
			serverError(c, err)
			return
		}
		d, err := loadDataset(db, scope)
		if err != nil {
			serverError(c, err)
			return
		}
		type agg struct {
			claims, payable    int
			gap, paid, pending int64
		}
		sum := map[uuid.UUID]*agg{}
		for _, cl := range d.claims {
			if cl.BatchID == nil {
				continue
			}
			a := sum[*cl.BatchID]
			if a == nil {
				a = &agg{}
				sum[*cl.BatchID] = a
			}
			a.claims++
			if !batchPayable(cl.Status) {
				continue
			}
			gap, paid := engine.Gap(cl), d.paid[cl.ID]
			a.payable++
			a.gap += gap
			a.paid += paid
			a.pending += remainingOf(gap, paid)
		}
		out := []J{}
		for _, b := range batches {
			a := sum[b.ID]
			if a == nil || a.payable == 0 {
				continue
			}
			out = append(out, J{"batch": batchDTO(b), "claimCount": a.claims, "payableCount": a.payable,
				"gap": a.gap, "paid": a.paid, "remaining": a.pending})
		}
		ok(c, out)
	}
}

// POST /payments/batch — catat pembayaran untuk satu batch (boleh dicicil, tidak boleh
// melebihi sisa tagihan batch).
func batchPaymentCreateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		var body struct {
			BatchID      string `json:"batchId"`
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
		if len(body.Payer) > 150 || len(body.TransferRef) > 100 {
			fail(c, http.StatusBadRequest, "Pembayar maks. 150 dan referensi maks. 100 karakter")
			return
		}
		b, err := findBatch(db, u, body.BatchID)
		if err != nil {
			notFoundOr500(c, err, batchNotFound)
			return
		}

		type part struct {
			ClaimNo string `json:"claimNo"`
			Amount  int64  `json:"amount"`
			Status  string `json:"status"`
		}
		var (
			parts     []part
			remaining int64
			warning   string
		)
		err = db.Transaction(func(tx *gorm.DB) error {
			var ids []uuid.UUID
			if err := tx.Model(&model.Claim{}).Where("batch_id = ?", b.ID).Order("claim_no").Pluck("id", &ids).Error; err != nil {
				return err
			}
			type slot struct {
				cl   *model.Claim
				paid int64
				rem  int64
			}
			var slots []slot
			var total int64
			for _, id := range ids {
				cl, err := lockClaim(tx, id)
				if err != nil {
					return err
				}
				if !batchPayable(cl.Status) {
					continue
				}
				paid, err := paidAmount(tx, cl.ID)
				if err != nil {
					return err
				}
				rem := remainingOf(engine.Gap(cl), paid)
				if rem > 0 {
					slots = append(slots, slot{cl, paid, rem})
					total += rem
				}
			}
			sort.SliceStable(slots, func(i, j int) bool { return slots[i].cl.ClaimNo < slots[j].cl.ClaimNo })
			if total == 0 {
				return errConflict("Batch ini tidak memiliki tagihan yang menunggu pembayaran")
			}
			if body.Amount > total {
				return errConflict(fmt.Sprintf("Nominal melebihi sisa tagihan batch (sisa Rp %d)", total))
			}
			if ref := strings.TrimSpace(body.TransferRef); ref != "" {
				var dup int64
				if err := scopedPayments(tx, &b.HospitalID).Where("transfer_ref = ?", ref).Count(&dup).Error; err != nil {
					return err
				}
				if dup > 0 {
					warning = "Potensi duplikasi: referensi transfer sudah pernah dicatat"
				}
			}
			note := strings.TrimSpace(body.Notes + " [Batch " + b.BatchNo + "]")
			left := body.Amount
			for _, s := range slots {
				if left == 0 {
					break
				}
				amt := s.rem
				if amt > left {
					amt = left
				}
				left -= amt
				no, err := nextNo(tx, "payments", "payment_no", "PAY")
				if err != nil {
					return err
				}
				p := model.Payment{PaymentNo: no, ReceivedDate: *received, Amount: amt, Payer: body.Payer,
					TransferRef: strings.TrimSpace(body.TransferRef), Notes: note, Allocation: allocation,
					ClaimID: s.cl.ID, CreatedByID: u.ID}
				if err := tx.Create(&p).Error; err != nil {
					return err
				}
				next := engine.StatusAfterPayment(s.cl.Status, engine.Gap(s.cl), s.paid+amt)
				if err := setClaimStatus(tx, s.cl, next, &u, "Pembayaran "+p.PaymentNo+" (batch "+b.BatchNo+") dicatat"); err != nil {
					return err
				}
				if err := audit.Log(tx, &u, audit.Entry{Action: "PAYMENT_RECORD", Entity: "claim", EntityID: s.cl.ClaimNo,
					After: fmt.Sprintf("%s → %d (%s, batch %s)", p.PaymentNo, p.Amount, p.Payer, b.BatchNo), HospitalID: &s.cl.HospitalID}, metaOf(c)); err != nil {
					return err
				}
				parts = append(parts, part{s.cl.ClaimNo, amt, string(s.cl.Status)})
			}
			remaining = total - body.Amount
			if len(parts) > 0 {
				if err := notifyFinance(tx, b.HospitalID, slots[0].cl.ID, "PAYMENT_RECEIVED",
					fmt.Sprintf("Pembayaran batch %s diterima: Rp %d untuk %d klaim", b.BatchNo, body.Amount, len(parts))); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			txError(c, err)
			return
		}
		created(c, J{"batch": batchDTO(*b), "parts": parts, "totalPaid": body.Amount, "remaining": remaining, "warning": warning})
	}
}
