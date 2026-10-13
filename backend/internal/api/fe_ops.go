package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"medpay_spkd/internal/audit"
	"medpay_spkd/internal/engine"
	"medpay_spkd/internal/model"
)

const workItemNotFound = "Work item tidak ditemukan"

func findWorkItem(db *gorm.DB, u model.User, id string) (*model.WorkItem, error) {
	wid, valid := parseID(id)
	if !valid {
		return nil, gorm.ErrRecordNotFound
	}
	var w model.WorkItem
	err := workItemPreloads(scopedWorkItems(db, u.HospitalID)).Where("work_items.id = ?", wid).First(&w).Error
	return &w, err
}

// checkAssignee memastikan penerima tugas ada dan berada di RS pemilik tugas.
func checkAssignee(db *gorm.DB, u model.User, id string, itemHospital *uuid.UUID) (*model.User, error) {
	uid, valid := parseID(id)
	if !valid {
		return nil, errors.New("assignee tidak valid")
	}
	var a model.User
	if err := db.First(&a, "id = ?", uid).Error; err != nil {
		return nil, errors.New("pengguna penerima tugas tidak ditemukan")
	}
	hospital := u.HospitalID
	if itemHospital != nil {
		hospital = itemHospital
	}
	if hospital != nil && a.HospitalID != nil && *a.HospitalID != *hospital {
		return nil, errors.New("penerima tugas harus dari RS yang sama")
	}
	if u.HospitalID != nil && a.HospitalID == nil {
		return nil, errors.New("penerima tugas harus dari RS yang sama")
	}
	return &a, nil
}

func workItemListHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, okScope := scopeOf(c)
		if !okScope {
			return
		}
		q := workItemPreloads(scopedWorkItems(db, scope))
		if s := c.Query("status"); s != "" {
			q = q.Where("work_items.status = ?", s)
		}
		if s := c.Query("source"); s != "" {
			q = q.Where("work_items.source = ?", s)
		}
		var items []model.WorkItem
		if err := q.Order("work_items.id").Find(&items).Error; err != nil {
			serverError(c, err)
			return
		}
		out := []J{}
		for _, w := range items {
			out = append(out, J{"workItem": workItemDTO(w)})
		}
		ok(c, out)
	}
}

func respondWorkItem(c *gin.Context, db *gorm.DB, id uuid.UUID, status int) {
	var w model.WorkItem
	if err := workItemPreloads(db).First(&w, "id = ?", id).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(status, workItemDTO(w))
}

func workItemCreateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		var body struct {
			Title      string  `json:"title"`
			Source     string  `json:"source"`
			ClaimID    *string `json:"claimId"`
			Assignee   *string `json:"assignee"`
			Role       *string `json:"role"`
			DueDate    string  `json:"dueDate"`
			Priority   string  `json:"priority"`
			Status     string  `json:"status"`
			HospitalID string  `json:"hospitalId"`
		}
		if !bind(c, &body) {
			return
		}
		title := strings.TrimSpace(body.Title)
		if title == "" || len([]rune(title)) > 255 {
			fail(c, http.StatusBadRequest, "Judul wajib diisi (maks. 255 karakter)")
			return
		}
		if body.Source == "" {
			body.Source = "MANUAL"
		}
		if body.Priority == "" {
			body.Priority = model.PrioritySedang
		}
		if body.Status == "" {
			body.Status = model.WorkItemNew
		}
		if !refValid(db, model.RefWorkItemSource, body.Source) || !refValid(db, model.RefWorkItemPriority, body.Priority) ||
			!refValid(db, model.RefWorkItemStatus, body.Status) {
			fail(c, http.StatusBadRequest, "Sumber, prioritas, atau status tidak dikenal")
			return
		}
		due, err := parseDate(body.DueDate)
		if err != nil {
			fail(c, http.StatusBadRequest, err.Error())
			return
		}
		if due == nil {
			t := engine.Today().AddDate(0, 0, 3)
			due = &t
		}

		w := model.WorkItem{Title: title, Source: body.Source, DueDate: *due, Priority: body.Priority,
			Status: body.Status, CreatedByID: &u.ID}
		// RS pemilik: RS klaim terkait, lalu RS user. Staf SPKD boleh memilih RS
		// (tanpa RS = tugas lintas RS yang hanya terlihat oleh staf SPKD).
		w.HospitalID = u.HospitalID
		if body.ClaimID != nil && *body.ClaimID != "" {
			cl, err := findClaim(db, u, *body.ClaimID, false)
			if err != nil {
				notFoundOr500(c, err, claimNotFound)
				return
			}
			w.ClaimID, w.HospitalID = &cl.ID, &cl.HospitalID
		} else if w.HospitalID == nil && body.HospitalID != "" {
			hid, valid := resolveHospital(c, db, u, body.HospitalID)
			if !valid {
				return
			}
			w.HospitalID = &hid
		}
		if body.Assignee != nil && *body.Assignee != "" {
			a, err := checkAssignee(db, u, *body.Assignee, w.HospitalID)
			if err != nil {
				fail(c, http.StatusBadRequest, err.Error())
				return
			}
			w.AssigneeID = &a.ID
			if w.Status == model.WorkItemNew {
				w.Status = model.WorkItemAssigned
			}
		}
		if body.Role != nil && *body.Role != "" {
			var n int64
			if err := db.Model(&model.Role{}).Where("code = ?", *body.Role).Count(&n).Error; err != nil || n == 0 {
				fail(c, http.StatusBadRequest, "Peran tujuan tidak dikenal")
				return
			}
			w.Role = *body.Role
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			no, err := nextNo(tx, "work_items", "item_no", "WI")
			if err != nil {
				return err
			}
			w.ItemNo = no
			w.Histories = []model.WorkItemHistory{{Action: "Dibuat", ActorID: &u.ID}}
			if err := tx.Create(&w).Error; err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "CREATE_WORK_ITEM", Entity: "work_item", EntityID: w.ItemNo,
				After: w.Title, HospitalID: w.HospitalID}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		respondWorkItem(c, db, w.ID, http.StatusCreated)
	}
}

func workItemUpdateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		var body struct {
			Status   *string `json:"status"`
			Assignee *string `json:"assignee"`
			Priority *string `json:"priority"`
			DueDate  *string `json:"dueDate"`
		}
		if !bind(c, &body) {
			return
		}
		w, err := findWorkItem(db, u, c.Param("id"))
		if err != nil {
			notFoundOr500(c, err, workItemNotFound)
			return
		}
		itemHospital := w.HospitalID

		updates := map[string]any{}
		var changes []string
		if body.Status != nil && *body.Status != w.Status {
			if !refValid(db, model.RefWorkItemStatus, *body.Status) {
				fail(c, http.StatusBadRequest, "Status tidak dikenal")
				return
			}
			changes = append(changes, "status "+w.Status+" → "+*body.Status)
			updates["status"], w.Status = *body.Status, *body.Status
		}
		if body.Assignee != nil && *body.Assignee != "" && *body.Assignee != idPtrStr(w.AssigneeID) {
			a, err := checkAssignee(db, u, *body.Assignee, itemHospital)
			if err != nil {
				fail(c, http.StatusBadRequest, err.Error())
				return
			}
			changes = append(changes, "ditugaskan ke "+a.Name)
			updates["assignee_id"] = a.ID
			if w.Status == model.WorkItemNew {
				updates["status"], w.Status = model.WorkItemAssigned, model.WorkItemAssigned
			}
		}
		if body.Priority != nil && *body.Priority != w.Priority {
			if !refValid(db, model.RefWorkItemPriority, *body.Priority) {
				fail(c, http.StatusBadRequest, "Prioritas tidak dikenal")
				return
			}
			changes = append(changes, "prioritas "+w.Priority+" → "+*body.Priority)
			updates["priority"] = *body.Priority
		}
		if body.DueDate != nil && *body.DueDate != fmtDate(w.DueDate) {
			due, err := parseDate(*body.DueDate)
			if err != nil || due == nil {
				fail(c, http.StatusBadRequest, "Tenggat wajib berformat YYYY-MM-DD")
				return
			}
			changes = append(changes, "tenggat → "+*body.DueDate)
			updates["due_date"] = *due
		}

		if len(changes) > 0 {
			summary := strings.Join(changes, "; ")
			err = db.Transaction(func(tx *gorm.DB) error {
				if err := tx.Model(&model.WorkItem{}).Where("id = ?", w.ID).Updates(updates).Error; err != nil {
					return err
				}
				if err := tx.Create(&model.WorkItemHistory{WorkItemID: w.ID, Action: summary, ActorID: &u.ID}).Error; err != nil {
					return err
				}
				return audit.Log(tx, &u, audit.Entry{Action: "UPDATE_WORK_ITEM", Entity: "work_item", EntityID: w.ItemNo,
					After: summary, HospitalID: itemHospital}, metaOf(c))
			})
			if err != nil {
				serverError(c, err)
				return
			}
		}
		respondWorkItem(c, db, w.ID, http.StatusOK)
	}
}

func workItemCommentHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		var body struct {
			Text string `json:"text"`
		}
		if !bind(c, &body) {
			return
		}
		text := strings.TrimSpace(body.Text)
		if text == "" || len([]rune(text)) > 2000 {
			fail(c, http.StatusBadRequest, "Komentar wajib diisi (maks. 2000 karakter)")
			return
		}
		w, err := findWorkItem(db, u, c.Param("id"))
		if err != nil {
			notFoundOr500(c, err, workItemNotFound)
			return
		}
		itemHospital := w.HospitalID
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&model.WorkItemComment{WorkItemID: w.ID, Body: text, AuthorID: u.ID}).Error; err != nil {
				return err
			}
			if err := tx.Create(&model.WorkItemHistory{WorkItemID: w.ID, Action: "Komentar ditambahkan", ActorID: &u.ID}).Error; err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "WORK_ITEM_COMMENT", Entity: "work_item", EntityID: w.ItemNo,
				After: text, HospitalID: itemHospital}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		respondWorkItem(c, db, w.ID, http.StatusOK)
	}
}

// ── GET /engine-api/summary ──────────────────────────────────────────────────

func engineAPISummaryHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, okScope := scopeOf(c)
		if !okScope {
			return
		}
		var intakes []model.APIIntake
		if err := whereHospital(db, "hospital_id", scope).Order("received_at DESC").Find(&intakes).Error; err != nil {
			serverError(c, err)
			return
		}
		unresolved, latency := 0, 0
		var totalHosp, totalJKN int64
		for _, a := range intakes {
			if a.Status == "UNRESOLVED" {
				unresolved++
			}
			totalHosp += a.HospitalTotal
			totalJKN += a.JKNTotal
			latency += a.LatencyMs
		}
		avgLatency := 0
		if len(intakes) > 0 {
			avgLatency = latency / len(intakes)
		}

		var stats struct {
			Calls    int64
			Rejected int64
			Partners int64
		}
		err := scopedAudit(db, scope).
			Select("COUNT(*) AS calls, COUNT(*) FILTER (WHERE status_code >= 400) AS rejected, COUNT(DISTINCT partner_id) AS partners").
			Where("scope = ?", model.AuditScopeEngineAPI).Scan(&stats).Error
		if err != nil {
			serverError(c, err)
			return
		}
		var coverage int64
		if err := whereHospital(db.Model(&model.Hospital{}), "id", scope).Count(&coverage).Error; err != nil {
			serverError(c, err)
			return
		}
		success := 0.0
		if stats.Calls > 0 {
			success = float64(stats.Calls-stats.Rejected) / float64(stats.Calls) * 100
		}
		ok(c, J{
			"intakes": mapSlice(intakes, intakeDTO), "unresolved": unresolved, "totalHospitalBill": totalHosp,
			"totalJkn": totalJKN, "apiCalls": stats.Calls, "rejected": stats.Rejected, "successPct": success,
			"avgLatencyMs": avgLatency, "partnerCount": stats.Partners, "hospitalCoverage": coverage,
		})
	}
}
