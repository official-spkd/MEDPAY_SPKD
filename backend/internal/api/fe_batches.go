package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"medpay_spkd/internal/audit"
	"medpay_spkd/internal/auth"
	"medpay_spkd/internal/engine"
	"medpay_spkd/internal/model"
)

const batchNotFound = "Batch tidak ditemukan"

var periodPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// findBatch mencari batch berdasarkan ID (UUID) dalam cakupan RS user.
func findBatch(db *gorm.DB, u model.User, id string) (*model.Batch, error) {
	bid, valid := parseID(id)
	if !valid {
		return nil, gorm.ErrRecordNotFound
	}
	q := db.Preload("CreatedBy").Where("id = ?", bid)
	if u.HospitalID != nil {
		q = q.Where("hospital_id = ?", *u.HospitalID)
	}
	var b model.Batch
	if err := q.First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// resolveHospital menentukan RS target untuk data baru: user RS selalu ke RS-nya,
// staf SPKD wajib memilih RS.
func resolveHospital(c *gin.Context, db *gorm.DB, u model.User, requested string) (uuid.UUID, bool) {
	hid, valid := parseOptID(requested)
	if !valid {
		fail(c, http.StatusBadRequest, "hospitalId tidak valid")
		return uuid.Nil, false
	}
	if hid == nil {
		hid = u.HospitalID
	}
	if hid == nil {
		fail(c, http.StatusBadRequest, "Akun SPKD wajib memilih RS (hospitalId)")
		return uuid.Nil, false
	}
	if !auth.CanAccessHospital(u, *hid) {
		fail(c, http.StatusForbidden, "Di luar cakupan RS Anda")
		return uuid.Nil, false
	}
	var n int64
	if err := db.Model(&model.Hospital{}).Where("id = ?", *hid).Count(&n).Error; err != nil {
		serverError(c, err)
		return uuid.Nil, false
	}
	if n == 0 {
		fail(c, http.StatusBadRequest, "RS tidak dikenal")
		return uuid.Nil, false
	}
	return *hid, true
}

func batchListHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, okScope := scopeOf(c)
		if !okScope {
			return
		}
		_, _ = syncAutoBatches(c, db, currentUser(c), scope)
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
		count := map[uuid.UUID]int{}
		gap := map[uuid.UUID]int64{}
		for _, cl := range d.claims {
			if cl.BatchID != nil {
				count[*cl.BatchID]++
				gap[*cl.BatchID] += engine.Gap(cl)
			}
		}
		out := []J{}
		for _, b := range batches {
			out = append(out, J{"batch": batchDTO(b), "claimCount": count[b.ID], "totalGap": gap[b.ID]})
		}
		ok(c, out)
	}
}

func batchCreateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		var body struct {
			Name       string `json:"name"`
			Period     string `json:"period"`
			HospitalID string `json:"hospitalId"`
			Note       string `json:"note"`
			InsurerID  string `json:"insurerId"`
			DateFrom   string `json:"dateFrom"`
			DateTo     string `json:"dateTo"`
		}
		if !bind(c, &body) {
			return
		}
		hid, valid := resolveHospital(c, db, u, body.HospitalID)
		if !valid {
			return
		}
		b := model.Batch{Note: body.Note, HospitalID: hid, CreatedByID: u.ID}

		// Mode penjamin + rentang tanggal: nama dibuat otomatis, klaim otomatis masuk.
		auto := strings.TrimSpace(body.InsurerID) != ""
		var ins model.Insurer
		var from, to time.Time
		if auto {
			iid, okID := parseID(body.InsurerID)
			if !okID || db.First(&ins, "id = ?", iid).Error != nil {
				fail(c, http.StatusBadRequest, "Penjamin tidak dikenal")
				return
			}
			var e1, e2 error
			from, e1 = time.ParseInLocation(dateLayout, body.DateFrom, engine.Location())
			to, e2 = time.ParseInLocation(dateLayout, body.DateTo, engine.Location())
			if e1 != nil || e2 != nil || to.Before(from) {
				fail(c, http.StatusBadRequest, "Tanggal mulai/akhir tidak valid (YYYY-MM-DD, akhir ≥ mulai)")
				return
			}
			b.InsurerID = &ins.ID
			b.PeriodStart, b.PeriodEnd = &from, &to
			b.Period = to.Format("2006-01")
			b.Name = fmt.Sprintf("%s — %s s/d %s", ins.Name, body.DateFrom, body.DateTo)
		} else {
			name := strings.TrimSpace(body.Name)
			if name == "" || len([]rune(name)) > 150 {
				fail(c, http.StatusBadRequest, "Nama batch wajib diisi (maks. 150 karakter)")
				return
			}
			if !periodPattern.MatchString(body.Period) {
				fail(c, http.StatusBadRequest, "Periode wajib berformat YYYY-MM")
				return
			}
			b.Name, b.Period = name, body.Period
		}

		// Kandidat otomatis: klaim RS ini, penjamin sama, belum ber-batch, Draft/Siap Diajukan,
		// tanggal pulang dalam rentang, dan tanpa blocker terbuka.
		var autoIDs []uuid.UUID
		autoBlocked := 0
		if auto {
			d, err := loadDataset(db, &hid)
			if err != nil {
				serverError(c, err)
				return
			}
			for _, cl := range d.claims {
				if cl.BatchID != nil || cl.InsurerID == nil || *cl.InsurerID != ins.ID ||
					(cl.Status != model.ClaimDraft && cl.Status != model.ClaimReadyToSubmit) || cl.DischargeDate == nil {
					continue
				}
				day := time.Date(cl.DischargeDate.Year(), cl.DischargeDate.Month(), cl.DischargeDate.Day(), 0, 0, 0, 0, from.Location())
				if day.Before(from) || day.After(to) {
					continue
				}
				if d.scrub(cl).HasOpenBlocker {
					autoBlocked++
					continue
				}
				autoIDs = append(autoIDs, cl.ID)
			}
		}

		err := db.Transaction(func(tx *gorm.DB) error {
			no, err := nextNo(tx, "batches", "batch_no", "BT")
			if err != nil {
				return err
			}
			b.BatchNo = no
			if err := tx.Create(&b).Error; err != nil {
				return err
			}
			if len(autoIDs) > 0 {
				if err := tx.Model(&model.Claim{}).Where("id IN ? AND batch_id IS NULL", autoIDs).Update("batch_id", b.ID).Error; err != nil {
					return err
				}
			}
			return audit.Log(tx, &u, audit.Entry{Action: "CREATE_BATCH", Entity: "batch", EntityID: b.BatchNo,
				After: fmt.Sprintf("%s (otomatis +%d, ditolak blocker %d)", b.Name, len(autoIDs), autoBlocked), HospitalID: &hid}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		b.CreatedBy = &u
		out := batchDTO(b)
		out["autoAdded"] = len(autoIDs)
		out["autoBlocked"] = autoBlocked
		created(c, out)
	}
}

func batchGetHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		b, err := findBatch(db, currentUser(c), c.Param("id"))
		if err != nil {
			notFoundOr500(c, err, batchNotFound)
			return
		}
		hid := b.HospitalID
		_, _ = syncAutoBatches(c, db, currentUser(c), &hid)
		d, err := loadDataset(db, &hid)
		if err != nil {
			serverError(c, err)
			return
		}
		claims, unbatched := []claimJSON{}, []claimJSON{}
		var totalGap int64
		blockerHits := 0
		for _, cl := range d.claims {
			sr := d.scrub(cl)
			switch {
			case cl.BatchID != nil && *cl.BatchID == b.ID:
				if sr.HasOpenBlocker {
					blockerHits++
				}
				totalGap += engine.Gap(cl)
				claims = append(claims, claimDTO(cl, &sr))
			case cl.BatchID == nil && (cl.Status == model.ClaimReadyToSubmit || cl.Status == model.ClaimDraft) && sameInsurer(b, cl):
				unbatched = append(unbatched, claimDTO(cl, &sr))
			}
		}
		ok(c, J{"batch": batchDTO(*b), "claims": claims, "unbatched": unbatched, "totalGap": totalGap, "blockerHits": blockerHits})
	}
}

func batchDeleteHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		b, err := findBatch(db, u, c.Param("id"))
		if err != nil {
			notFoundOr500(c, err, batchNotFound)
			return
		}
		var submitted int64
		err = db.Model(&model.Claim{}).
			Where("batch_id = ? AND status NOT IN ?", b.ID, []model.ClaimStatus{model.ClaimDraft, model.ClaimReadyToSubmit}).
			Count(&submitted).Error
		if err != nil {
			serverError(c, err)
			return
		}
		if submitted > 0 {
			fail(c, http.StatusConflict, fmt.Sprintf("Batch berisi %d klaim yang sudah diajukan dan tidak dapat dihapus", submitted))
			return
		}
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&model.Claim{}).Where("batch_id = ?", b.ID).Update("batch_id", nil).Error; err != nil {
				return err
			}
			if err := tx.Delete(&model.Batch{}, "id = ?", b.ID).Error; err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "DELETE_BATCH", Entity: "batch", EntityID: b.BatchNo,
				After: "klaim dilepas (tidak dihapus)", HospitalID: &b.HospitalID}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		ok(c, J{"ok": true})
	}
}

func batchAddClaimsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		var body struct {
			ClaimIDs []string `json:"claimIds"`
		}
		if !bind(c, &body) {
			return
		}
		if len(body.ClaimIDs) > 500 {
			fail(c, http.StatusBadRequest, "Maksimal 500 klaim per permintaan")
			return
		}
		b, err := findBatch(db, u, c.Param("id"))
		if err != nil {
			notFoundOr500(c, err, batchNotFound)
			return
		}
		hid := b.HospitalID
		d, err := loadDataset(db, &hid)
		if err != nil {
			serverError(c, err)
			return
		}
		var ids []uuid.UUID
		blocked, otherInsurer := 0, 0
		for _, raw := range body.ClaimIDs {
			id, _ := parseID(raw)
			cl := d.byID[id]
			if cl == nil || cl.BatchID != nil || (cl.Status != model.ClaimDraft && cl.Status != model.ClaimReadyToSubmit) {
				continue
			}
			if !sameInsurer(b, cl) {
				otherInsurer++
				continue
			}
			if d.scrub(cl).HasOpenBlocker {
				blocked++
				continue
			}
			ids = append(ids, cl.ID)
		}
		err = db.Transaction(func(tx *gorm.DB) error {
			if len(ids) > 0 {
				// batch_id IS NULL mencegah klaim masuk dua batch saat request bersamaan.
				if err := tx.Model(&model.Claim{}).Where("id IN ? AND batch_id IS NULL", ids).Update("batch_id", b.ID).Error; err != nil {
					return err
				}
			}
			return audit.Log(tx, &u, audit.Entry{Action: "BATCH_ADD_CLAIMS", Entity: "batch", EntityID: b.BatchNo,
				After: fmt.Sprintf("+%d (ditolak blocker: %d)", len(ids), blocked), HospitalID: &hid}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		ok(c, J{"added": len(ids), "blocked": blocked, "otherInsurer": otherInsurer})
	}
}

func batchRemoveClaimHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		b, err := findBatch(db, u, c.Param("id"))
		if err != nil {
			notFoundOr500(c, err, batchNotFound)
			return
		}
		claimID, _ := parseID(c.Param("claimId"))
		var claimNo string
		db.Model(&model.Claim{}).Where("id = ?", claimID).Pluck("claim_no", &claimNo)
		err = db.Transaction(func(tx *gorm.DB) error {
			res := tx.Model(&model.Claim{}).Where("id = ? AND batch_id = ?", claimID, b.ID).Update("batch_id", nil)
			if res.Error != nil || res.RowsAffected == 0 {
				return res.Error
			}
			return audit.Log(tx, &u, audit.Entry{Action: "BATCH_REMOVE_CLAIM", Entity: "batch", EntityID: b.BatchNo,
				Before: claimNo, HospitalID: &b.HospitalID}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		ok(c, J{"ok": true})
	}
}

// batchSubmitHandler mengajukan seluruh klaim Siap Diajukan di batch dengan satu aksi
// (F-BAT-04). Aturan pengajuan sama dengan pengajuan satu per satu: izin APPROVE,
// maker-checker (pengaju ≠ pembuat), dan tidak ada blocker terbuka.
func batchSubmitHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		b, err := findBatch(db, u, c.Param("id"))
		if err != nil {
			notFoundOr500(c, err, batchNotFound)
			return
		}
		hid := b.HospitalID
		d, err := loadDataset(db, &hid)
		if err != nil {
			serverError(c, err)
			return
		}
		type skip struct {
			ID      string `json:"id"`
			ClaimNo string `json:"claimNo"`
			Reason  string `json:"reason"`
		}
		// Satu blocker terbuka di klaim mana pun dalam batch menahan seluruh batch.
		var blocked []string
		for _, cl := range d.claims {
			if cl.BatchID == nil || *cl.BatchID != b.ID || (cl.Status != model.ClaimReadyToSubmit && cl.Status != model.ClaimDraft) {
				continue
			}
			if sr := d.scrub(cl); sr.HasOpenBlocker {
				blocked = append(blocked, cl.ClaimNo+" ("+strings.Join(sr.OpenBlockers(), ", ")+")")
			}
		}
		if len(blocked) > 0 {
			fail(c, http.StatusConflict, fmt.Sprintf("Batch tidak dapat diajukan: %d klaim masih punya blocker terbuka — %s. Selesaikan blocker atau keluarkan klaimnya dari batch.", len(blocked), strings.Join(blocked, "; ")))
			return
		}
		var toSubmit []*model.Claim
		skipped := []skip{}
		for _, cl := range d.claims {
			if cl.BatchID == nil || *cl.BatchID != b.ID {
				continue
			}
			switch {
			case cl.Status != model.ClaimReadyToSubmit:
				skipped = append(skipped, skip{idStr(cl.ID), cl.ClaimNo, "status " + claimLabel(db, cl.Status)})
			case cl.CreatedByID == u.ID:
				skipped = append(skipped, skip{idStr(cl.ID), cl.ClaimNo, "maker-checker: Anda pembuat klaim ini"})
			case d.scrub(cl).HasOpenBlocker:
				skipped = append(skipped, skip{idStr(cl.ID), cl.ClaimNo, "blocker terbuka: " + strings.Join(d.scrub(cl).OpenBlockers(), ", ")})
			default:
				toSubmit = append(toSubmit, cl)
			}
		}
		submitted := []string{}
		err = db.Transaction(func(tx *gorm.DB) error {
			now := time.Now()
			for _, cl := range toSubmit {
				if err := tx.Model(cl).Update("submitted_at", now).Error; err != nil {
					return err
				}
				if err := setClaimStatus(tx, cl, model.ClaimSubmitted, &u, "Diajukan via batch "+b.BatchNo); err != nil {
					return err
				}
				submitted = append(submitted, idStr(cl.ID))
			}
			return audit.Log(tx, &u, audit.Entry{Action: "BATCH_SUBMIT", Entity: "batch", EntityID: b.BatchNo,
				After: fmt.Sprintf("diajukan %d, dilewati %d", len(submitted), len(skipped)), HospitalID: &hid}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		ok(c, J{"submitted": submitted, "skipped": skipped})
	}
}
