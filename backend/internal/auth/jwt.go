// Package auth: token login (JWT + tabel sessions), MFA, peran & izin (RBAC),
// serta isolasi data per rumah sakit.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"medpay_spkd/internal/appenv"
	"medpay_spkd/internal/model"
)

const (
	tokenIssuer     = "medpay"
	sessionAudience = "medpay-api"
	mfaAudience     = "medpay-mfa"
	SessionLifetime = 12 * time.Hour
	mfaLifetime     = 5 * time.Minute
	minSecretLength = 32
	maxTokenLength  = 4096
)

const devSecret = "dev-only-medpay-secret-change-me"

var (
	secretOnce sync.Once
	secretKey  []byte
)

// CheckConfig: di produksi JWT_SECRET wajib ada dan minimal 32 karakter.
func CheckConfig() error {
	v := os.Getenv("JWT_SECRET")
	if appenv.IsProduction() {
		switch {
		case v == "":
			return errors.New("JWT_SECRET wajib diisi saat APP_ENV=production")
		case len(v) < minSecretLength:
			return errors.New("JWT_SECRET terlalu pendek (minimal 32 karakter)")
		case v == devSecret:
			return errors.New("JWT_SECRET masih memakai secret dev")
		}
	}
	return nil
}

// jwtSecret dibaca dari env JWT_SECRET. Secret dev hanya dipakai di luar produksi.
func jwtSecret() []byte {
	secretOnce.Do(func() {
		if v := os.Getenv("JWT_SECRET"); v != "" {
			secretKey = []byte(v)
			return
		}
		if appenv.IsProduction() {
			log.Fatal("JWT_SECRET kosong di produksi")
		}
		log.Println("PERINGATAN: JWT_SECRET belum diisi, memakai secret dev. WAJIB diisi di produksi.")
		secretKey = []byte(devSecret)
	})
	return secretKey
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// HashToken: yang disimpan di tabel sessions hanya hash token, bukan token aslinya.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func sign(userID uuid.UUID, audience string, lifetime time.Duration) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(lifetime)
	claims := jwt.RegisteredClaims{
		ID:        randomID(),
		Issuer:    tokenIssuer,
		Subject:   userID.String(),
		Audience:  jwt.ClaimStrings{audience},
		ExpiresAt: jwt.NewNumericDate(exp),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret())
	return signed, exp, err
}

// parse memvalidasi token dan mengembalikan ID user (subject).
func parse(token, audience string) (uuid.UUID, error) {
	if len(token) > maxTokenLength {
		return uuid.Nil, errors.New("token terlalu panjang")
	}
	claims := &jwt.RegisteredClaims{}
	t, err := jwt.ParseWithClaims(token, claims,
		func(*jwt.Token) (interface{}, error) { return jwtSecret(), nil },
		jwt.WithValidMethods([]string{"HS256"}), // tolak alg=none / alg confusion
		jwt.WithIssuer(tokenIssuer),
		jwt.WithAudience(audience),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil || !t.Valid {
		return uuid.Nil, errors.New("token tidak valid")
	}
	id, err := uuid.Parse(claims.Subject)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, errors.New("token tidak valid")
	}
	return id, nil
}

// IssueSession membuat token login dan mencatatnya di tabel sessions.
func IssueSession(db *gorm.DB, userID uuid.UUID, ip, userAgent string) (string, time.Time, error) {
	token, exp, err := sign(userID, sessionAudience, SessionLifetime)
	if err != nil {
		return "", time.Time{}, err
	}
	if len(userAgent) > 255 {
		userAgent = userAgent[:255]
	}
	s := model.Session{TokenHash: HashToken(token), IP: ip, UserAgent: userAgent, ExpiresAt: exp, UserID: userID}
	if err := db.Create(&s).Error; err != nil {
		return "", time.Time{}, err
	}
	return token, exp, nil
}

// IssueMFAChallenge membuat token sementara setelah kata sandi benar. Token ini
// wajib dikirim saat verifikasi kode MFA, sehingga langkah sandi tidak bisa dilewati.
func IssueMFAChallenge(userID uuid.UUID) (string, error) {
	t, _, err := sign(userID, mfaAudience, mfaLifetime)
	return t, err
}

// ParseMFAChallenge memvalidasi token MFA dan mengembalikan ID user.
func ParseMFAChallenge(token string) (uuid.UUID, error) { return parse(token, mfaAudience) }

// SessionCookie = nama cookie httpOnly berisi token sesi (N-SEC-02).
const SessionCookie = "medpay_session"

// BearerToken mengambil token dari header Authorization.
func BearerToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if len(h) < 8 || !strings.EqualFold(h[:7], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(h[7:])
}

// RequestToken mengambil token sesi dari header Authorization, lalu dari cookie httpOnly.
func RequestToken(c *gin.Context) string {
	if t := BearerToken(c); t != "" {
		return t
	}
	t, _ := c.Cookie(SessionCookie)
	return t
}

// SetSessionCookie menyimpan token di cookie httpOnly + SameSite=Strict sehingga
// tidak dapat dibaca skrip di browser.
func SetSessionCookie(c *gin.Context, token string, exp time.Time) {
	secure := appenv.IsProduction() || c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	http.SetCookie(c.Writer, &http.Cookie{
		Name: SessionCookie, Value: token, Path: "/api", Expires: exp, MaxAge: int(time.Until(exp).Seconds()),
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode,
	})
}

// ClearSessionCookie menghapus cookie sesi.
func ClearSessionCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: SessionCookie, Value: "", Path: "/api", MaxAge: -1,
		HttpOnly: true, SameSite: http.SameSiteStrictMode,
	})
}

// IdleTimeout = batas waktu tanpa aktivitas sebelum sesi berakhir (F-AUT-10).
// SESSION_IDLE_MINUTES, default 30 menit; 0 = nonaktif.
func IdleTimeout() time.Duration {
	if v, err := strconv.Atoi(appenv.Get("SESSION_IDLE_MINUTES", "30")); err == nil && v >= 0 {
		return time.Duration(v) * time.Minute
	}
	return 30 * time.Minute
}

// RevokeSession menandai sesi token sebagai dicabut (logout).
func RevokeSession(db *gorm.DB, token string) error {
	return db.Model(&model.Session{}).
		Where("token_hash = ? AND revoked_at IS NULL", HashToken(token)).
		Update("revoked_at", time.Now()).Error
}

// RevokeOtherSessions mencabut semua sesi user kecuali sesi saat ini.
func RevokeOtherSessions(db *gorm.DB, userID uuid.UUID, keepToken string) error {
	return db.Model(&model.Session{}).
		Where("user_id = ? AND token_hash <> ? AND revoked_at IS NULL", userID, HashToken(keepToken)).
		Update("revoked_at", time.Now()).Error
}

func deny(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": msg})
}

// Wrap = middleware autentikasi: validasi JWT (header atau cookie), cek sesi di
// database (belum dicabut, belum kedaluwarsa, masih ada aktivitas), lalu muat ulang
// user beserta peran & izinnya. Akun nonaktif/terhapus atau RS nonaktif langsung
// ditolak walau token belum kedaluwarsa.
func Wrap(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := RequestToken(c)
		if token == "" {
			deny(c, "Sesi tidak valid atau kosong. Silakan masuk kembali.")
			return
		}
		userID, err := parse(token, sessionAudience)
		if err != nil {
			deny(c, "Sesi kedaluwarsa atau tidak valid. Silakan masuk kembali.")
			return
		}

		now := time.Now()
		var sess model.Session
		err = db.Where("token_hash = ? AND user_id = ? AND revoked_at IS NULL AND expires_at > ?", HashToken(token), userID, now).
			First(&sess).Error
		if err != nil {
			deny(c, "Sesi telah berakhir. Silakan masuk kembali.")
			return
		}
		last := sess.CreatedAt
		if sess.LastSeenAt != nil {
			last = *sess.LastSeenAt
		}
		if idle := IdleTimeout(); idle > 0 && now.Sub(last) > idle {
			db.Model(&sess).Update("revoked_at", now)
			deny(c, "Sesi berakhir karena tidak ada aktivitas. Silakan masuk kembali.")
			return
		}
		if now.Sub(last) > time.Minute {
			db.Model(&sess).Update("last_seen_at", now)
		}

		user, err := LoadUser(db, userID)
		if err != nil || !user.Active {
			deny(c, "Akun tidak ditemukan atau telah dinonaktifkan.")
			return
		}
		if user.Hospital != nil && !user.Hospital.Active {
			deny(c, "Rumah sakit Anda sedang dinonaktifkan. Hubungi SPKD.")
			return
		}

		// Peran, izin & RS selalu diambil dari database, bukan dari klaim token.
		c.Set(ctxUser, user)
		c.Set(ctxToken, token)
		c.Next()
	}
}

// CurrentToken = token sesi request saat ini (diisi Wrap).
func CurrentToken(c *gin.Context) string {
	v, _ := c.Get(ctxToken)
	s, _ := v.(string)
	return s
}

// LoadUser memuat user beserta RS, peran, dan izin perannya.
func LoadUser(db *gorm.DB, id uuid.UUID) (model.User, error) {
	var u model.User
	err := db.Preload("Hospital").Preload("Role.Permissions").First(&u, "id = ?", id).Error
	return u, err
}
