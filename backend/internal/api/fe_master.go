package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"medpay_spkd/internal/audit"
	"medpay_spkd/internal/model"
)

// ── Penjamin ─────────────────────────────────────────────────────────────────

type insurerIn struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	AvgPaymentDays int    `json:"avgPaymentDays"`
	DeadlineDays   int    `json:"deadlineDays"`
	CeilingPolicy  string `json:"ceilingPolicy"`
	Verified       bool   `json:"verified"`
}

func (in *insurerIn) validate(db *gorm.DB, exceptID uuid.UUID) error {
	in.Name = strings.TrimSpace(in.Name)
	switch {
	case in.Name == "" || len([]rune(in.Name)) > 150 || len([]rune(in.CeilingPolicy)) > 255:
		return errors.New("Nama wajib diisi (maks. 150), kebijakan plafon maks. 255 karakter")
	case !refValid(db, model.RefInsurerType, in.Type):
		return errors.New("Jenis penjamin tidak dikenal")
	case in.AvgPaymentDays < 0 || in.DeadlineDays < 0 || in.DeadlineDays > 3650:
		return errors.New("Jumlah hari tidak valid")
	}
	var n int64
	if err := db.Model(&model.Insurer{}).Where("LOWER(name) = LOWER(?) AND id <> ?", in.Name, exceptID).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return errDuplicate
	}
	return nil
}

var errDuplicate = errors.New("Nama penjamin sudah terdaftar")

func insurerCreateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if !requirePlatformStaff(c, u, "Master penjamin") {
			return
		}
		var in insurerIn
		if !bind(c, &in) {
			return
		}
		if err := in.validate(db, uuid.Nil); err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, errDuplicate) {
				status = http.StatusConflict
			}
			fail(c, status, err.Error())
			return
		}
		ins := model.Insurer{Name: in.Name, Type: model.InsurerType(in.Type), AvgPaymentDays: in.AvgPaymentDays,
			DeadlineDays: in.DeadlineDays, CeilingPolicy: in.CeilingPolicy, Verified: in.Verified}
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&ins).Error; err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "CREATE_INSURER", Entity: "insurer", EntityID: idStr(ins.ID),
				After: ins.Name}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		created(c, insurerDTO(ins))
	}
}

// ── Notifikasi ───────────────────────────────────────────────────────────────

// notificationReadHandler menandai satu notifikasi milik pengguna sebagai sudah dibaca.
func notificationReadHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		id, valid := parseID(c.Param("id"))
		if !valid {
			fail(c, http.StatusNotFound, "Notifikasi tidak ditemukan")
			return
		}
		res := db.Model(&model.Notification{}).Where("id = ? AND recipient_user_id = ? AND read_at IS NULL", id, u.ID).
			Update("read_at", time.Now())
		if res.Error != nil {
			serverError(c, res.Error)
			return
		}
		if res.RowsAffected == 0 {
			var n int64
			db.Model(&model.Notification{}).Where("id = ? AND recipient_user_id = ?", id, u.ID).Count(&n)
			if n == 0 {
				fail(c, http.StatusNotFound, "Notifikasi tidak ditemukan")
				return
			}
		}
		ok(c, J{"ok": true})
	}
}

// notificationReadAllHandler menandai semua notifikasi pengguna sebagai sudah dibaca.
func notificationReadAllHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		res := db.Model(&model.Notification{}).Where("recipient_user_id = ? AND read_at IS NULL", u.ID).
			Update("read_at", time.Now())
		if res.Error != nil {
			serverError(c, res.Error)
			return
		}
		ok(c, J{"ok": true, "updated": res.RowsAffected})
	}
}
