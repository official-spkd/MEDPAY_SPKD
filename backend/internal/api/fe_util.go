package api

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
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

// Kontrak API frontend Vue (src/api.js): respons sukses berupa JSON data langsung,
// respons gagal berupa {"error": "pesan"}. Field camelCase, tanggal "YYYY-MM-DD",
// waktu "YYYY-MM-DDTHH:MM:SS" (zona waktu aplikasi). ID publik klaim, batch,
// pembayaran, dst. memakai nomor dokumen (CLM-0001); entitas master memakai ID numerik
// dalam bentuk string.

type J = map[string]any

const (
	dateLayout  = "2006-01-02"
	stampLayout = "2006-01-02T15:04:05"
)

func ok(c *gin.Context, data any)      { c.JSON(http.StatusOK, data) }
func created(c *gin.Context, data any) { c.JSON(http.StatusCreated, data) }

func fail(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, J{"error": msg})
}

// serverError mencatat detail error di log server; klien hanya menerima pesan umum.
func serverError(c *gin.Context, err error) {
	log.Printf("ERROR %s %s [%s]: %v", c.Request.Method, c.Request.URL.Path, correlationOf(c), err)
	fail(c, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
}

// notFoundOr500 membedakan data tidak ditemukan dengan error lain.
func notFoundOr500(c *gin.Context, err error, msg string) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, http.StatusNotFound, msg)
		return
	}
	serverError(c, err)
}

// bind mem-parsing body JSON; body kosong dianggap objek kosong.
func bind(c *gin.Context, dst any) bool {
	if c.Request.ContentLength == 0 {
		return true
	}
	if err := c.ShouldBindJSON(dst); err != nil {
		fail(c, http.StatusBadRequest, "Format data tidak valid: "+err.Error())
		return false
	}
	return true
}

func currentUser(c *gin.Context) model.User {
	u, _ := auth.CurrentUser(c)
	return u
}

// scopeOf = filter RS untuk user saat ini (nil = semua RS, khusus staf SPKD).
func scopeOf(c *gin.Context) (*uuid.UUID, bool) {
	h, err := auth.ScopeHospital(currentUser(c), c.Query("hospitalId"))
	if err != nil {
		code := http.StatusBadRequest
		if errors.Is(err, auth.ErrCrossTenant) {
			code = http.StatusForbidden
		}
		fail(c, code, err.Error())
		return nil, false
	}
	return h, true
}

// whereHospital menerapkan filter RS pada kolom yang diberikan.
func whereHospital(q *gorm.DB, col string, scope *uuid.UUID) *gorm.DB {
	if scope == nil {
		return q
	}
	return q.Where(col+" = ?", *scope)
}

func correlationOf(c *gin.Context) string {
	v, _ := c.Get("correlation_id")
	s, _ := v.(string)
	return s
}

func metaOf(c *gin.Context) audit.Meta {
	return audit.Meta{IP: c.ClientIP(), CorrelationID: correlationOf(c)}
}

// ── ID ───────────────────────────────────────────────────────────────────────

func idStr(id uuid.UUID) string { return id.String() }

func idPtrStr(p *uuid.UUID) string {
	if p == nil {
		return ""
	}
	return idStr(*p)
}

func idPtrOrNil(p *uuid.UUID) any {
	if p == nil {
		return nil
	}
	return idStr(*p)
}

// parseID mem-parsing UUID dari parameter URL / body.
func parseID(s string) (uuid.UUID, bool) {
	v, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil || v == uuid.Nil {
		return uuid.Nil, false
	}
	return v, true
}

// parseOptID: string kosong = nil; selain itu wajib UUID valid.
func parseOptID(s string) (*uuid.UUID, bool) {
	if strings.TrimSpace(s) == "" {
		return nil, true
	}
	id, ok := parseID(s)
	if !ok {
		return nil, false
	}
	return &id, true
}

// nextNo membuat nomor dokumen berikutnya (mis. CLM-0035). Advisory lock per tabel
// mencegah dua request mendapat nomor yang sama; nomor baris yang sudah dihapus
// (soft delete) tidak dipakai ulang.
func nextNo(tx *gorm.DB, table, column, prefix string) (string, error) {
	n, err := lastNo(tx, table, column)
	if err != nil {
		return "", err
	}
	return formatNo(prefix, n+1), nil
}

// lastNo mengunci penomoran tabel sampai transaksi selesai lalu mengembalikan
// angka nomor dokumen terbesar. table & column selalu konstanta dari kode.
func lastNo(tx *gorm.DB, table, column string) (int64, error) {
	if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", table).Error; err != nil {
		return 0, err
	}
	var n int64
	err := tx.Raw(fmt.Sprintf(
		`SELECT COALESCE(MAX(CAST(SUBSTRING(%s FROM '([0-9]+)$') AS BIGINT)), 0) FROM %s`, column, table,
	)).Scan(&n).Error
	return n, err
}

func formatNo(prefix string, n int64) string { return fmt.Sprintf("%s-%04d", prefix, n) }

// ── Tanggal ──────────────────────────────────────────────────────────────────

func fmtDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(engine.Location()).Format(dateLayout)
}

func fmtDatePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return fmtDate(*t)
}

func fmtStamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(engine.Location()).Format(stampLayout)
}

func fmtStampPtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return fmtStamp(*t)
}

func stampOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return fmtStamp(*t)
}

// parseDate menerima "YYYY-MM-DD" (atau stempel waktu, diambil tanggalnya).
// String kosong = nil.
func parseDate(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	if len(s) > len(dateLayout) {
		s = s[:len(dateLayout)]
	}
	t, err := time.ParseInLocation(dateLayout, s, engine.Location())
	if err != nil {
		return nil, fmt.Errorf("tanggal %q tidak valid (format YYYY-MM-DD)", s)
	}
	return &t, nil
}

func todayStr() string { return engine.Today().Format(dateLayout) }

// ── CSV ──────────────────────────────────────────────────────────────────────

// sendCSV mengirim CSV ber-BOM dengan pemisah titik koma (ramah Excel Indonesia).
func sendCSV(c *gin.Context, filename string, rows [][]string) {
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF}) // BOM UTF-8 agar Excel membaca karakter Indonesia dengan benar
	w := csv.NewWriter(&buf)
	w.Comma = ';'
	for _, r := range rows {
		for i, v := range r {
			r[i] = csvSafe(v)
		}
		_ = w.Write(r)
	}
	w.Flush()
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}

// csvSafe mencegah CSV/formula injection saat file dibuka di spreadsheet.
func csvSafe(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		if _, err := strconv.ParseFloat(v, 64); err != nil {
			return "'" + v
		}
	}
	return v
}

func itoa[T ~int | ~int64](v T) string { return strconv.FormatInt(int64(v), 10) }

func rupiah(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "." + s[i:]
	}
	if neg {
		return "Rp -" + s
	}
	return "Rp " + s
}

// denyAccess menolak aksi (403) dan mencatatnya di audit trail (F-AUD-03).
func denyAccess(c *gin.Context, db *gorm.DB, u model.User, entity, entityID, msg string) {
	_ = audit.Log(db, &u, audit.Entry{Action: "ACCESS_DENIED", Entity: entity, EntityID: entityID,
		Before: c.Request.Method + " " + c.FullPath(), After: msg, StatusCode: http.StatusForbidden}, metaOf(c))
	fail(c, http.StatusForbidden, msg)
}

// requirePerm = middleware izin tingkat route yang mencatat penolakan di audit.
func requirePerm(db *gorm.DB, perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if !auth.HasPermission(u, perm) {
			denyAccess(c, db, u, "permission", perm, fmt.Sprintf("Peran Anda (%s) tidak memiliki izin %s", auth.RoleCode(u), perm))
			return
		}
		c.Next()
	}
}
