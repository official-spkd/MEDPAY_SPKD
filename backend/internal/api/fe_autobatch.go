package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"medpay_spkd/internal/audit"
	"medpay_spkd/internal/model"
)

// claimFitsBatch: klaim memenuhi syarat batch per-penjamin bila penjamin sama, status
// Draft/Siap Diajukan, belum ber-batch, dan tanggal pulang (akhir episode) dalam rentang batch.
func claimFitsBatch(cl *model.Claim, b *model.Batch) bool {
	if b.InsurerID == nil || b.PeriodStart == nil || b.PeriodEnd == nil {
		return false
	}
	if cl.BatchID != nil || cl.InsurerID == nil || *cl.InsurerID != *b.InsurerID || cl.DischargeDate == nil {
		return false
	}
	if cl.Status != model.ClaimDraft && cl.Status != model.ClaimReadyToSubmit {
		return false
	}
	day := fmtDatePtr(cl.DischargeDate)
	return day >= fmtDatePtr(b.PeriodStart) && day <= fmtDatePtr(b.PeriodEnd)
}

// sameInsurer: batch per-penjamin hanya menerima klaim penjamin yang sama
// (batch lama tanpa penjamin menerima semua).
func sameInsurer(b *model.Batch, cl *model.Claim) bool {
	return b.InsurerID == nil || (cl.InsurerID != nil && *cl.InsurerID == *b.InsurerID)
}

// syncAutoBatches memasukkan klaim yang sudah bebas blocker ke batch per-penjamin yang
// cocok dan masih terbuka (belum berisi klaim yang diajukan). Dipanggil setelah perubahan
// yang bisa menyelesaikan blocker dan saat daftar/detail batch dibuka. scope nil = semua RS.
func syncAutoBatches(c *gin.Context, db *gorm.DB, u model.User, scope *uuid.UUID) (int, error) {
	var batches []model.Batch
	q := whereHospital(db, "hospital_id", scope).Where("insurer_id IS NOT NULL AND period_start IS NOT NULL")
	if err := q.Order("created_at, id").Find(&batches).Error; err != nil {
		return 0, err
	}
	byHospital := map[uuid.UUID][]*model.Batch{}
	for i := range batches {
		byHospital[batches[i].HospitalID] = append(byHospital[batches[i].HospitalID], &batches[i])
	}
	total := 0
	for hid, list := range byHospital {
		h := hid
		d, err := loadDataset(db, &h)
		if err != nil {
			return total, err
		}
		closed := map[uuid.UUID]bool{}
		for _, cl := range d.claims {
			if cl.BatchID != nil && cl.Status != model.ClaimDraft && cl.Status != model.ClaimReadyToSubmit {
				closed[*cl.BatchID] = true
			}
		}
		assign := map[uuid.UUID][]uuid.UUID{}
		for _, cl := range d.claims {
			for _, b := range list {
				if closed[b.ID] || !claimFitsBatch(cl, b) {
					continue
				}
				if !d.scrub(cl).HasOpenBlocker {
					assign[b.ID] = append(assign[b.ID], cl.ID)
				}
				break // satu klaim hanya untuk batch terawal yang cocok
			}
		}
		for _, b := range list {
			ids := assign[b.ID]
			if len(ids) == 0 {
				continue
			}
			bb := b
			err := db.Transaction(func(tx *gorm.DB) error {
				if err := tx.Model(&model.Claim{}).Where("id IN ? AND batch_id IS NULL", ids).Update("batch_id", bb.ID).Error; err != nil {
					return err
				}
				return audit.Log(tx, &u, audit.Entry{Action: "BATCH_AUTO_ADD", Entity: "batch", EntityID: bb.BatchNo,
					After: fmt.Sprintf("+%d klaim bebas blocker masuk otomatis", len(ids)), HospitalID: &bb.HospitalID}, metaOf(c))
			})
			if err != nil {
				return total, err
			}
			total += len(ids)
		}
	}
	return total, nil
}

// autoBatchSync = middleware: setelah aksi yang bisa menyelesaikan blocker berhasil,
// klaim yang sudah bersih langsung dimasukkan ke batch per-penjamin yang sesuai.
func autoBatchSync(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if c.Writer.Status() >= 300 {
			return
		}
		u := currentUser(c)
		_, _ = syncAutoBatches(c, db, u, u.HospitalID)
	}
}
