package api

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"medpay_spkd/internal/engine"
	"medpay_spkd/internal/model"
)

// Tugas otomatis antrean kerja (F-WQ-04): dibuat oleh sistem saat klaim ber-blocker,
// ditolak, masuk zona deadline kritis, mutasi bank perlu review, atau piutang > 90 hari.
// Satu klaim hanya punya satu tugas terbuka per sumber (tidak dobel).

var closedWorkItem = []string{model.WorkItemResolved, model.WorkItemClosed}

// autoTask membuat tugas sistem bila belum ada tugas terbuka dengan sumber yang sama.
func autoTask(tx *gorm.DB, hospitalID uuid.UUID, claimID *uuid.UUID, source, priority, title string, due time.Time) error {
	q := tx.Model(&model.WorkItem{}).Where("source = ? AND hospital_id = ? AND status NOT IN ?", source, hospitalID, closedWorkItem)
	if claimID != nil {
		q = q.Where("claim_id = ?", *claimID)
	} else {
		q = q.Where("claim_id IS NULL AND title = ?", title)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil || n > 0 {
		return err
	}
	no, err := nextNo(tx, "work_items", "item_no", "WI")
	if err != nil {
		return err
	}
	hid := hospitalID
	w := model.WorkItem{ItemNo: no, Title: title, Source: source, DueDate: due, Priority: priority,
		Status: model.WorkItemNew, ClaimID: claimID, HospitalID: &hid,
		Histories: []model.WorkItemHistory{{Action: "Dibuat otomatis oleh sistem"}}}
	return tx.Create(&w).Error
}

// RunAutoTasks memindai klaim semua RS dan membuat tugas untuk blocker terbuka,
// deadline kritis/lewat, dan piutang > 90 hari. Dipanggil berkala oleh server.
func RunAutoTasks(db *gorm.DB) (int, error) {
	d, err := loadDataset(db, nil)
	if err != nil {
		return 0, err
	}
	var before int64
	db.Model(&model.WorkItem{}).Count(&before)
	today := d.today
	err = db.Transaction(func(tx *gorm.DB) error {
		for _, c := range d.claims {
			id := c.ID
			if c.Status == model.ClaimDraft || c.Status == model.ClaimReadyToSubmit {
				if sr := d.scrub(c); sr.HasOpenBlocker {
					title := fmt.Sprintf("Klaim %s (%s) memiliki blocker terbuka: %v", c.ClaimNo, c.PatientName, sr.OpenBlockers())
					if err := autoTask(tx, c.HospitalID, &id, "SCRUBBER", model.PriorityTinggi, title, today.AddDate(0, 0, 3)); err != nil {
						return err
					}
				}
			}
			if openStatuses[c.Status] {
				if di := engine.Deadline(c.AdmissionDate, today); di.Zone == engine.ZoneKritis || di.Zone == engine.ZoneLewat {
					title := fmt.Sprintf("Klaim %s (%s) %s batas pengajuan 6 bulan (%s)", c.ClaimNo, c.PatientName,
						map[string]string{engine.ZoneKritis: "mendekati", engine.ZoneLewat: "melewati"}[di.Zone], fmtDatePtr(di.Deadline))
					if err := autoTask(tx, c.HospitalID, &id, "DEADLINE", model.PriorityKritis, title, today.AddDate(0, 0, 1)); err != nil {
						return err
					}
				}
			}
			if o, open := openReceivable(d, c); open && o > 0 {
				if b := agingBasis(c, false); b != nil && engine.DaysBetween(*b, today) > 90 {
					title := fmt.Sprintf("Piutang klaim %s (%s) berumur > 90 hari, sisa %s", c.ClaimNo, c.PatientName, rupiah(o))
					if err := autoTask(tx, c.HospitalID, &id, "AGING", model.PrioritySedang, title, today.AddDate(0, 0, 7)); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	var after int64
	db.Model(&model.WorkItem{}).Count(&after)
	return int(after - before), nil
}

// StartAutoTasks menjalankan RunAutoTasks saat start lalu setiap interval.
func StartAutoTasks(db *gorm.DB, every time.Duration) {
	run := func() {
		if n, err := RunAutoTasks(db); err != nil {
			log.Printf("tugas otomatis gagal: %v", err)
		} else if n > 0 {
			log.Printf("tugas otomatis: %d tugas baru", n)
		}
	}
	go func() {
		run()
		for range time.Tick(every) {
			run()
		}
	}()
}
