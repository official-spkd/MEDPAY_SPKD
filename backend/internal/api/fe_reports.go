package api

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"medpay_spkd/internal/engine"
	"medpay_spkd/internal/model"
)

const auditPageSize = 100

// filteredAudit menerapkan cakupan RS dan filter halaman Audit Trail.
func filteredAudit(c *gin.Context, db *gorm.DB, scope *uuid.UUID) *gorm.DB {
	q := scopedAudit(db, scope)
	if s := c.Query("scope"); s != "" {
		q = q.Where("audit_logs.scope = ?", s)
	}
	if c.Query("failed") == "1" {
		q = q.Where("audit_logs.status_code >= 400")
	}
	if s := strings.TrimSpace(c.Query("q")); s != "" {
		like := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(strings.ToLower(s)) + "%"
		q = q.Joins("LEFT JOIN partners ON partners.id = audit_logs.partner_id").
			Where(`(LOWER(audit_logs.actor_name) LIKE ? OR LOWER(audit_logs.action) LIKE ? OR LOWER(audit_logs.entity_id) LIKE ?
				OR LOWER(audit_logs.ip) LIKE ? OR LOWER(COALESCE(partners.company, '')) LIKE ?)`, like, like, like, like, like)
	}
	return q
}

// ── GET /audit ───────────────────────────────────────────────────────────────

func auditListHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, okScope := scopeOf(c)
		if !okScope {
			return
		}
		page, _ := strconv.Atoi(c.Query("page"))
		if page < 1 {
			page = 1
		}

		var summary struct {
			Total     int64
			Rejected  int64
			UniqueIPs int64
		}
		err := filteredAudit(c, db, scope).
			Select("COUNT(*) AS total, COUNT(*) FILTER (WHERE audit_logs.status_code >= 400) AS rejected, COUNT(DISTINCT audit_logs.ip) AS unique_ips").
			Scan(&summary).Error
		if err != nil {
			serverError(c, err)
			return
		}
		var logs []model.AuditLog
		err = filteredAudit(c, db, scope).Preload("Partner").
			Order("audit_logs.created_at DESC, audit_logs.id DESC").
			Offset((page - 1) * auditPageSize).Limit(auditPageSize).Find(&logs).Error
		if err != nil {
			serverError(c, err)
			return
		}

		// Partner yang memanggil Engine API dari lebih dari satu IP (indikasi kebocoran kredensial).
		var multi []struct {
			Partner string
			IPs     string
			Count   int
		}
		err = scopedAudit(db, scope).
			Joins("JOIN partners ON partners.id = audit_logs.partner_id").
			Select("partners.company AS partner, STRING_AGG(DISTINCT audit_logs.ip, ',') AS ips, COUNT(DISTINCT audit_logs.ip) AS count").
			Where("audit_logs.scope = ?", model.AuditScopeEngineAPI).
			Group("partners.company").Having("COUNT(DISTINCT audit_logs.ip) > 1").Scan(&multi).Error
		if err != nil {
			serverError(c, err)
			return
		}
		multiIP := []J{}
		for _, m := range multi {
			multiIP = append(multiIP, J{"partner": m.Partner, "ips": strings.Split(m.IPs, ","), "count": m.Count})
		}

		ok(c, J{
			"rows":    mapSlice(logs, auditDTO),
			"summary": J{"total": summary.Total, "rejected": summary.Rejected, "uniqueIps": summary.UniqueIPs},
			"multiIp": multiIP, "page": page, "pageSize": auditPageSize,
		})
	}
}

// ── Ekspor CSV ───────────────────────────────────────────────────────────────

func exportClaimsHandler(db *gorm.DB) gin.HandlerFunc {
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
		rows := [][]string{{"ID", "SEP", "Pasien", "Status", "Jenis", "Kode INA-CBG", "Tarif JKN", "Tagihan RS", "Gap", "Penjamin", "Readiness"}}
		for _, cl := range d.claims {
			rows = append(rows, []string{cl.ClaimNo, cl.SEPNo, cl.PatientName, string(cl.Status), cl.CareType, cl.INACBGCode,
				itoa(engine.TotalJKN(cl.INACBGBaseTariff, cl.SpecialCMG)), itoa(engine.SumTariff(cl.HospitalTariff)),
				itoa(engine.Gap(cl)), d.insurerName(cl), itoa(d.scrub(cl).Score)})
		}
		sendCSV(c, "klaim-medpay.csv", rows)
	}
}

func exportAgingHandler(db *gorm.DB) gin.HandlerFunc {
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
		list, _ := agingRows(d, c.Query("basis") == "service")
		rows := [][]string{{"ID", "Pasien", "Penjamin", "Status", "Nilai Tertunggak", "Umur (hari)"}}
		for _, r := range list {
			rows = append(rows, []string{r.ID, r.PatientName, r.Insurer, r.Status, itoa(r.Amount), itoa(r.AgeDays)})
		}
		sendCSV(c, "piutang-medpay.csv", rows)
	}
}

func exportAuditHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, okScope := scopeOf(c)
		if !okScope {
			return
		}
		var logs []model.AuditLog
		err := filteredAudit(c, db, scope).Order("audit_logs.created_at, audit_logs.id").Limit(50000).Find(&logs).Error
		if err != nil {
			serverError(c, err)
			return
		}
		rows := [][]string{{"Waktu", "Aktor", "Peran", "Aksi", "Entitas", "ID", "Scope", "Status", "IP"}}
		for _, a := range logs {
			rows = append(rows, []string{fmtStamp(a.CreatedAt), a.ActorName, a.ActorRole, a.Action, a.Entity,
				a.EntityID, a.Scope, itoa(a.StatusCode), a.IP})
		}
		sendCSV(c, "audit-medpay.csv", rows)
	}
}
