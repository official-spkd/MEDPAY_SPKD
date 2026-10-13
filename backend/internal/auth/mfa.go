package auth

import (
	"crypto/subtle"
	"strings"

	"medpay_spkd/internal/appenv"
	"medpay_spkd/internal/model"
)

// DemoMFACode = kode verifikasi MFA tetap untuk lingkungan non-produksi.
// Bisa diganti lewat env MFA_DEMO_CODE.
const DemoMFACode = "246810"

// VerifyMFA memeriksa kode MFA langkah kedua login. Belum ada penyedia kode
// sungguhan (TOTP/OTP), sehingga di produksi verifikasi selalu ditolak.
func VerifyMFA(_ model.User, code string) bool {
	if appenv.IsProduction() {
		return false
	}
	want := appenv.Get("MFA_DEMO_CODE", DemoMFACode)
	return subtle.ConstantTimeCompare([]byte(strings.TrimSpace(code)), []byte(want)) == 1
}
