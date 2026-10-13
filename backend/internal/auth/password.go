package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// Parameter Argon2id (rekomendasi OWASP: m=19 MiB, t=2, p=1; di sini m=64 MiB).
const (
	argonTime    uint32 = 2
	argonMemory  uint32 = 64 * 1024
	argonThreads uint8  = 1
	argonKeyLen  uint32 = 32
	argonSaltLen        = 16
)

// MinPasswordLength & MaxPasswordLength = batas panjang kata sandi.
const (
	MinPasswordLength = 8
	MaxPasswordLength = 128
)

// HashPassword membuat hash Argon2id berformat PHC:
// $argon2id$v=19$m=65536,t=2,p=1$<salt>$<hash>
func HashPassword(pw string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pw), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// VerifyPassword memeriksa kata sandi terhadap hash Argon2id atau bcrypt (hash lama).
func VerifyPassword(hash, pw string) bool {
	if strings.HasPrefix(hash, "$argon2id$") {
		return verifyArgon2id(hash, pw)
	}
	if strings.HasPrefix(hash, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
	}
	return false
}

// NeedsRehash bernilai true untuk hash lama (bcrypt) atau parameter Argon2id usang.
func NeedsRehash(hash string) bool {
	want := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$", argon2.Version, argonMemory, argonTime, argonThreads)
	return !strings.HasPrefix(hash, want)
}

func verifyArgon2id(hash, pw string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false
	}
	var version int
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil || m == 0 || t == 0 || p == 0 || m > 1<<20 {
		return false
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil || len(want) == 0 {
		return false
	}
	got := argon2.IDKey([]byte(pw), salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// DummyHash dipakai saat email tidak ditemukan agar waktu respons login tetap sama.
var DummyHash, _ = HashPassword("medpay-dummy-password-for-timing")

// ValidatePasswordPolicy: minimal 8 karakter, memuat huruf dan angka, dan tidak sama dengan email.
func ValidatePasswordPolicy(pw, email string) error {
	if len(pw) < MinPasswordLength || len(pw) > MaxPasswordLength {
		return fmt.Errorf("kata sandi harus %d–%d karakter", MinPasswordLength, MaxPasswordLength)
	}
	var letter, digit bool
	for _, r := range pw {
		switch {
		case unicode.IsLetter(r):
			letter = true
		case unicode.IsDigit(r):
			digit = true
		}
	}
	if !letter || !digit {
		return errors.New("kata sandi harus memuat huruf dan angka")
	}
	if email != "" && strings.EqualFold(pw, email) {
		return errors.New("kata sandi tidak boleh sama dengan email")
	}
	return nil
}
