package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"medpay_spkd/internal/audit"
	"medpay_spkd/internal/auth"
	"medpay_spkd/internal/model"
)

// ---------- pembatas percobaan login (brute force) ----------

const (
	maxIPLoginFails = 30 // per IP (penguncian per akun disimpan di tabel users)
	loginLockFor    = 15 * time.Minute
	loginWindow     = 15 * time.Minute
	maxLimiterKeys  = 50000
	maxEmailLength  = 254
)

type failState struct {
	count int
	first time.Time
	until time.Time
}

type loginLimiter struct {
	mu sync.Mutex
	m  map[string]*failState
}

func newLoginLimiter() *loginLimiter { return &loginLimiter{m: map[string]*failState{}} }

func (l *loginLimiter) blocked(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	st := l.m[key]
	if st == nil || st.until.IsZero() {
		return false, 0
	}
	if time.Now().After(st.until) {
		delete(l.m, key)
		return false, 0
	}
	return true, time.Until(st.until)
}

func (l *loginLimiter) fail(key string, limit int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if len(l.m) >= maxLimiterKeys {
		l.pruneLocked(now)
	}
	st := l.m[key]
	if st == nil || (st.until.IsZero() && now.Sub(st.first) > loginWindow) {
		st = &failState{first: now}
		l.m[key] = st
	}
	st.count++
	if st.count >= limit {
		st.until = now.Add(loginLockFor)
	}
}

// pruneLocked membuang entri yang sudah kedaluwarsa.
func (l *loginLimiter) pruneLocked(now time.Time) {
	for k, st := range l.m {
		if (!st.until.IsZero() && now.After(st.until)) || (st.until.IsZero() && now.Sub(st.first) > loginWindow) {
			delete(l.m, k)
		}
	}
}

func (l *loginLimiter) reset(key string) {
	l.mu.Lock()
	delete(l.m, key)
	l.mu.Unlock()
}

func (l *loginLimiter) check(c *gin.Context, keys ...string) bool {
	for _, k := range keys {
		if blocked, wait := l.blocked(k); blocked {
			fail(c, http.StatusTooManyRequests,
				fmt.Sprintf("Terlalu banyak percobaan gagal. Coba lagi dalam %d menit.", int(wait.Minutes())+1))
			return false
		}
	}
	return true
}

// Penguncian akun persisten (N-SEC-05): 5 kali gagal berturut-turut → terkunci 15 menit.
const (
	maxAccountFails = 5
	accountLockFor  = 15 * time.Minute
)

func lockedMessage(until time.Time) string {
	return fmt.Sprintf("Terlalu banyak percobaan gagal. Coba lagi dalam %d menit.", int(time.Until(until).Minutes())+1)
}

// registerFailure menaikkan hitungan gagal akun dan menguncinya bila melewati batas.
func registerFailure(c *gin.Context, db *gorm.DB, u *model.User) {
	fails := u.FailedLogins + 1
	updates := map[string]any{"failed_logins": fails}
	if fails >= maxAccountFails {
		until := time.Now().Add(accountLockFor)
		updates = map[string]any{"failed_logins": 0, "locked_until": until}
		_ = audit.Log(db, u, audit.Entry{Action: "ACCOUNT_LOCKED", Entity: "user", EntityID: idStr(u.ID),
			After: "terkunci sampai " + fmtStamp(until), StatusCode: http.StatusTooManyRequests}, metaOf(c))
	}
	db.Model(&model.User{}).Where("id = ?", u.ID).Updates(updates)
}

func isLocked(u model.User) bool { return u.LockedUntil != nil && time.Now().Before(*u.LockedUntil) }

// startSession membuat sesi login, menyimpan token di cookie httpOnly, dan mengirim
// token + profil user (token di body tetap dikirim untuk klien non-browser/Postman).
func startSession(c *gin.Context, db *gorm.DB, u model.User, action string) {
	token, exp, err := auth.IssueSession(db, u.ID, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		serverError(c, err)
		return
	}
	db.Model(&model.User{}).Where("id = ?", u.ID).
		Updates(map[string]any{"last_login_at": time.Now(), "failed_logins": 0, "locked_until": nil})
	_ = audit.Log(db, &u, audit.Entry{Action: action, Entity: "user", EntityID: idStr(u.ID)}, metaOf(c))
	auth.SetSessionCookie(c, token, exp)
	ok(c, J{"token": token, "expiresAt": fmtStamp(exp), "user": userDTO(u)})
}

// ---------- POST /api/auth/login ----------

func loginHandler(db *gorm.DB) gin.HandlerFunc {
	lim := newLoginLimiter()
	return func(c *gin.Context) {
		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Email) == "" || body.Password == "" {
			fail(c, http.StatusBadRequest, "Email dan kata sandi wajib diisi.")
			return
		}
		email := strings.ToLower(strings.TrimSpace(body.Email))
		if len(email) > maxEmailLength || len(body.Password) > auth.MaxPasswordLength {
			fail(c, http.StatusUnauthorized, "Email atau kata sandi salah")
			return
		}
		ipKey := "ip|" + c.ClientIP()
		if !lim.check(c, ipKey) {
			return
		}

		var user model.User
		err := db.Preload("Hospital").Preload("Role.Permissions").Where("email = ?", email).First(&user).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			serverError(c, err)
			return
		}
		found := err == nil
		if found && isLocked(user) {
			fail(c, http.StatusTooManyRequests, lockedMessage(*user.LockedUntil))
			return
		}
		// Pesan dan waktu sama untuk email/sandi salah (F-AUT-03).
		hash := user.PasswordHash
		if !found {
			hash = auth.DummyHash
		}
		if !auth.VerifyPassword(hash, body.Password) || !found {
			lim.fail(ipKey, maxIPLoginFails)
			if found {
				registerFailure(c, db, &user)
			}
			_ = audit.LogAnonymous(db, email, audit.Entry{
				Action: "LOGIN_FAILED", Entity: "user", EntityID: email, StatusCode: http.StatusUnauthorized,
				HospitalID: user.HospitalID,
			}, metaOf(c))
			fail(c, http.StatusUnauthorized, "Email atau kata sandi salah")
			return
		}
		if !user.Active {
			_ = audit.Log(db, &user, audit.Entry{Action: "LOGIN_REJECTED", Entity: "user", EntityID: idStr(user.ID),
				After: "akun nonaktif", StatusCode: http.StatusForbidden}, metaOf(c))
			fail(c, http.StatusForbidden, "Akun Anda telah dinonaktifkan. Hubungi Admin RS.")
			return
		}
		if user.Hospital != nil && !user.Hospital.Active {
			fail(c, http.StatusForbidden, "Rumah sakit Anda sedang dinonaktifkan. Hubungi SPKD.")
			return
		}
		// Hash lama (bcrypt) di-upgrade ke Argon2id saat kata sandi diketahui benar.
		if auth.NeedsRehash(user.PasswordHash) {
			if h, err := auth.HashPassword(body.Password); err == nil {
				db.Model(&model.User{}).Where("id = ?", user.ID).Update("password_hash", h)
			}
		}

		if user.MFAEnabled {
			challenge, err := auth.IssueMFAChallenge(user.ID)
			if err != nil {
				serverError(c, err)
				return
			}
			ok(c, J{"mfaRequired": true, "email": user.Email, "mfaToken": challenge})
			return
		}
		startSession(c, db, user, "LOGIN")
	}
}

// ---------- POST /api/auth/verify ----------

func verifyMFAHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Email    string `json:"email"`
			Code     string `json:"code"`
			MFAToken string `json:"mfaToken"`
		}
		if !bind(c, &body) {
			return
		}
		userID, err := auth.ParseMFAChallenge(body.MFAToken)
		if err != nil {
			fail(c, http.StatusUnauthorized, "Sesi verifikasi MFA tidak valid atau kedaluwarsa. Silakan masuk ulang.")
			return
		}
		user, err := auth.LoadUser(db, userID)
		if err != nil || !user.Active || (body.Email != "" && !strings.EqualFold(strings.TrimSpace(body.Email), user.Email)) {
			fail(c, http.StatusUnauthorized, "Pengguna tidak ditemukan")
			return
		}
		if isLocked(user) {
			fail(c, http.StatusTooManyRequests, lockedMessage(*user.LockedUntil))
			return
		}
		if !auth.VerifyMFA(user, body.Code) {
			registerFailure(c, db, &user)
			_ = audit.Log(db, &user, audit.Entry{
				Action: "MFA_FAILED", Entity: "user", EntityID: idStr(user.ID), StatusCode: http.StatusUnauthorized,
			}, metaOf(c))
			fail(c, http.StatusUnauthorized, "Kode MFA salah")
			return
		}
		startSession(c, db, user, "LOGIN_MFA")
	}
}

// ---------- POST /api/auth/logout ----------

func logoutHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if tok := auth.RequestToken(c); tok != "" {
			if err := auth.RevokeSession(db, tok); err != nil {
				serverError(c, err)
				return
			}
		}
		auth.ClearSessionCookie(c)
		ok(c, J{"ok": true})
	}
}

// ---------- GET /api/me ----------

func meHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		ok(c, J{"user": userDTO(u), "permissions": auth.Permissions(u)})
	}
}

// ---------- POST /api/auth/change-password ----------

func changePasswordHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		var body struct {
			CurrentPassword string `json:"currentPassword"`
			NewPassword     string `json:"newPassword"`
		}
		if !bind(c, &body) {
			return
		}
		if !auth.VerifyPassword(u.PasswordHash, body.CurrentPassword) {
			registerFailure(c, db, &u)
			_ = audit.Log(db, &u, audit.Entry{Action: "PASSWORD_CHANGE_FAILED", Entity: "user", EntityID: idStr(u.ID),
				StatusCode: http.StatusUnauthorized}, metaOf(c))
			fail(c, http.StatusBadRequest, "Kata sandi saat ini salah")
			return
		}
		if err := auth.ValidatePasswordPolicy(body.NewPassword, u.Email); err != nil {
			fail(c, http.StatusBadRequest, "Kata sandi baru tidak memenuhi kebijakan: "+err.Error())
			return
		}
		if body.NewPassword == body.CurrentPassword {
			fail(c, http.StatusBadRequest, "Kata sandi baru harus berbeda dari kata sandi saat ini")
			return
		}
		hash, err := auth.HashPassword(body.NewPassword)
		if err != nil {
			serverError(c, err)
			return
		}
		err = db.Transaction(func(tx *gorm.DB) error {
			err := tx.Model(&model.User{}).Where("id = ?", u.ID).Updates(map[string]any{
				"password_hash": hash, "password_changed_at": time.Now(), "must_change_password": false, "failed_logins": 0,
			}).Error
			if err != nil {
				return err
			}
			// Sesi lain (perangkat lain) dicabut; sesi saat ini tetap berlaku.
			if err := auth.RevokeOtherSessions(tx, u.ID, auth.CurrentToken(c)); err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "PASSWORD_CHANGED", Entity: "user", EntityID: idStr(u.ID)}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		ok(c, J{"ok": true})
	}
}
