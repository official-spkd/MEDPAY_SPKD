// Package api berisi handler HTTP & router MedPay. Endpoint mengikuti kontrak
// API yang dipanggil frontend Vue (frontend/src/api.js) di bawah prefix /api.
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"medpay_spkd/internal/appenv"
	"medpay_spkd/internal/auth"
	pdb "medpay_spkd/internal/db"
	"medpay_spkd/internal/model"
)

// ---------- middleware ----------

// Correlation ID dari klien: maksimal 64 karakter, tanpa spasi/baris baru.
var correlationIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,64}$`)

func newCorrelationID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "c-" + hex.EncodeToString(b)
}

func correlationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Correlation-Id")
		if !correlationIDPattern.MatchString(id) {
			id = newCorrelationID()
		}
		c.Set("correlation_id", id)
		c.Header("X-Correlation-Id", id)
		c.Next()
	}
}

// cors hanya mengizinkan origin yang ada di CORS_ORIGIN (pisahkan dengan koma).
func cors() gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, o := range appenv.List("CORS_ORIGIN", "http://localhost:5173") {
		if o = strings.TrimRight(o, "/"); o != "*" {
			allowed[o] = true
		}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		c.Header("Vary", "Origin")
		if origin != "" && allowed[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Correlation-Id")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Expose-Headers", "X-Correlation-Id, Content-Disposition")
			c.Header("Access-Control-Max-Age", "600")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// securityHeaders: header keamanan dasar.
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Cache-Control", "no-store")
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

// maxBody membatasi ukuran body request.
func maxBody(n int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, n)
		}
		c.Next()
	}
}

func healthHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := pdb.Ping(ctx, db); err != nil {
			c.JSON(http.StatusServiceUnavailable, J{"status": "up", "database": "down"})
			return
		}
		ok(c, J{"status": "up", "database": "up"})
	}
}

// NewRouter merakit semua endpoint.
func NewRouter(db *gorm.DB) (*gin.Engine, error) {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	// TRUSTED_PROXIES kosong = tidak percaya proxy mana pun (IP klien = remote address).
	if err := r.SetTrustedProxies(appenv.List("TRUSTED_PROXIES", "")); err != nil {
		return nil, fmt.Errorf("TRUSTED_PROXIES tidak valid: %w", err)
	}
	r.Use(securityHeaders(), correlationID(), cors(), maxBody(4<<20))
	r.NoRoute(func(c *gin.Context) { fail(c, http.StatusNotFound, "Endpoint tidak ditemukan") })

	r.GET("/", func(c *gin.Context) { ok(c, J{"app": "SPKD MedPay API", "health": "/api/health"}) })

	api := r.Group("/api")
	api.GET("/health", healthHandler(db))

	// Publik (tanpa sesi).
	api.POST("/auth/login", loginHandler(db))
	api.POST("/auth/verify", verifyMFAHandler(db))
	api.POST("/auth/logout", logoutHandler(db))

	// Engine API untuk SIMRS partner: autentikasi kredensial partner, bukan sesi pengguna.
	api.POST("/engine/claims", engineClaimHandler(db))

	s := api.Group("", auth.Wrap(db))
	// Setiap penolakan izin dicatat di audit trail (F-AUD-03).
	perm := func(p string) gin.HandlerFunc { return requirePerm(db, p) }

	s.GET("/me", meHandler())
	s.GET("/bootstrap", bootstrapHandler(db))
	s.GET("/reference", referenceHandler(db))
	s.PUT("/notifications/:id/read", notificationReadHandler(db))
	s.POST("/notifications/read-all", notificationReadAllHandler(db))
	s.PUT("/reference/:category/:code", perm(model.PermConfigure), referenceUpdateHandler(db))
	s.PUT("/app-settings/:key", perm(model.PermConfigure), appSettingUpdateHandler(db))

	// Akun sendiri: ganti kata sandi
	s.POST("/auth/change-password", changePasswordHandler(db))

	// Dashboard & wawasan
	s.GET("/dashboard", perm(model.PermView), dashboardHandler(db))
	s.GET("/analytics", perm(model.PermView), analyticsHandler(db))
	s.GET("/aging", perm(model.PermView), agingHandler(db))
	s.POST("/kapj/simulate", perm(model.PermView), kapjSimulateHandler(db))

	// Klaim
	s.GET("/claims", perm(model.PermView), claimListHandler(db))
	s.POST("/claims", perm(model.PermCreate), claimCreateHandler(db))
	s.GET("/claims/:id", perm(model.PermView), claimGetHandler(db))
	s.PUT("/claims/:id", perm(model.PermEdit), claimUpdateHandler(db))
	s.DELETE("/claims/:id", perm(model.PermDelete), claimDeleteHandler(db))
	s.POST("/claims/:id/resolve", perm(model.PermResolve), claimResolveHandler(db))
	// Izin dicek per transisi di handler: pengajuan (submitted) butuh APPROVE, lainnya EDIT.
	s.POST("/claims/:id/transition", perm(model.PermView), claimTransitionHandler(db))
	s.POST("/claims/:id/scrub", perm(model.PermView), claimScrubHandler(db))
	s.POST("/grouper/preview", perm(model.PermResolve), grouperPreviewHandler(db))

	// Scrubber
	s.GET("/scrubber/top-causes", perm(model.PermView), scrubTopCausesHandler(db))
	s.PUT("/scrub-rules/:id", perm(model.PermConfigure), scrubRuleUpdateHandler(db))
	s.POST("/scrub-rules/override", perm(model.PermOverride), scrubOverrideHandler(db))

	// Resume medis
	s.POST("/resume/bulk", perm(model.PermCreate), resumeBulkHandler(db))
	s.PUT("/resume/:id", perm(model.PermEdit), resumeUpdateHandler(db))
	s.DELETE("/resume/:id", perm(model.PermDelete), resumeDeleteHandler(db))

	// Batch
	s.GET("/batches", perm(model.PermView), batchListHandler(db))
	s.POST("/batches", perm(model.PermCreate), batchCreateHandler(db))
	s.GET("/batches/:id", perm(model.PermView), batchGetHandler(db))
	s.DELETE("/batches/:id", perm(model.PermDelete), batchDeleteHandler(db))
	s.POST("/batches/:id/claims", perm(model.PermEdit), batchAddClaimsHandler(db))
	s.DELETE("/batches/:id/claims/:claimId", perm(model.PermEdit), batchRemoveClaimHandler(db))
	s.POST("/batches/:id/submit", perm(model.PermApprove), batchSubmitHandler(db))

	// Pembayaran & rekonsiliasi
	s.GET("/payments", perm(model.PermView), paymentListHandler(db))
	s.POST("/payments", perm(model.PermCreate), paymentCreateHandler(db))
	s.DELETE("/payments/:id", perm(model.PermEdit), paymentDeleteHandler(db))
	s.GET("/bank-imports", perm(model.PermView), bankImportListHandler(db))
	s.POST("/bank-imports", perm(model.PermCreate), bankImportCreateHandler(db))
	s.GET("/bank-lines", perm(model.PermView), bankLineListHandler(db))
	s.GET("/bank-lines/:id/candidates", perm(model.PermView), bankLineCandidatesHandler(db))
	s.POST("/bank-lines/:id/confirm", perm(model.PermApprove), bankLineConfirmHandler(db))
	s.POST("/bank-lines/:id/unconfirm", perm(model.PermApprove), bankLineUnconfirmHandler(db))

	// Antrean kerja
	s.GET("/work-items", perm(model.PermView), workItemListHandler(db))
	s.POST("/work-items", perm(model.PermCreate), workItemCreateHandler(db))
	s.PUT("/work-items/:id", perm(model.PermEdit), workItemUpdateHandler(db))
	s.POST("/work-items/:id/comments", perm(model.PermView), workItemCommentHandler(db))

	// Sistem
	s.GET("/audit", perm(model.PermView), auditListHandler(db))
	s.GET("/engine-api/summary", perm(model.PermView), engineAPISummaryHandler(db))
	s.POST("/hospitals", perm(model.PermConfigure), hospitalCreateHandler(db))
	s.PUT("/hospitals/:id", perm(model.PermConfigure), hospitalUpdateHandler(db))
	s.POST("/users", perm(model.PermConfigure), userCreateHandler(db))
	s.PUT("/users/:id", perm(model.PermConfigure), userUpdateHandler(db))
	s.POST("/users/:id/reset-password", perm(model.PermConfigure), userResetPasswordHandler(db))
	s.POST("/insurers", perm(model.PermConfigure), insurerCreateHandler(db))
	s.PUT("/insurers/:id", perm(model.PermConfigure), insurerUpdateHandler(db))
	s.PUT("/partners/:id", perm(model.PermConfigure), partnerUpdateHandler(db))
	s.POST("/partners/:id/keys", perm(model.PermConfigure), partnerKeyCreateHandler(db))
	s.POST("/partners/:id/rotate-secret", perm(model.PermConfigure), partnerRotateSecretHandler(db))

	// Ekspor CSV
	s.GET("/export/claims.csv", perm(model.PermExport), exportClaimsHandler(db))
	s.GET("/export/aging.csv", perm(model.PermExport), exportAgingHandler(db))
	s.GET("/export/audit.csv", perm(model.PermExport), exportAuditHandler(db))

	return r, nil
}
