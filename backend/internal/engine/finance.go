package engine

import (
	"math"
	"strings"
	"time"

	"medpay_spkd/internal/model"
)

// ── Aging piutang & DSO ──────────────────────────────────────────────────────

// Bucket = satu kelompok umur piutang.
type Bucket struct {
	Label  string `json:"label"`
	Count  int    `json:"count"`
	Amount int64  `json:"amount"`
}

const BucketOver90 = "> 90 hari"

var bucketRanges = []struct {
	label    string
	min, max int
}{
	{"0–30 hari", 0, 30},
	{"31–60 hari", 31, 60},
	{"61–90 hari", 61, 90},
	{BucketOver90, 91, math.MaxInt},
}

// Receivable = satu piutang terbuka beserta tanggal dasar perhitungan umur.
type Receivable struct {
	Amount int64
	Basis  time.Time
}

// AgingBuckets mengelompokkan piutang berdasarkan umur (hari sejak tanggal dasar).
// Piutang dengan tanggal dasar di masa depan tidak dihitung.
func AgingBuckets(items []Receivable, today time.Time) ([]Bucket, int64) {
	out := make([]Bucket, len(bucketRanges))
	for i, b := range bucketRanges {
		out[i].Label = b.label
	}
	var total int64
	for _, it := range items {
		age := DaysBetween(it.Basis, today)
		for i, b := range bucketRanges {
			if age >= b.min && age <= b.max {
				out[i].Count++
				out[i].Amount += it.Amount
				total += it.Amount
				break
			}
		}
	}
	return out, total
}

// PaidPair = tanggal pengajuan dan tanggal pembayaran terakhir klaim lunas.
type PaidPair struct{ Submitted, Paid time.Time }

// DSO = rata-rata hari dari pengajuan sampai lunas; nil bila belum ada data.
func DSO(pairs []PaidPair) *int {
	sum, n := 0, 0
	for _, p := range pairs {
		if !Day(p.Paid).Before(Day(p.Submitted)) {
			sum += DaysBetween(p.Submitted, p.Paid)
			n++
		}
	}
	if n == 0 {
		return nil
	}
	v := sum / n
	return &v
}

// ── Status pembayaran ────────────────────────────────────────────────────────

// StatusAfterPayment menghitung status klaim setelah pembayaran dicatat.
// Hanya berlaku untuk klaim disetujui / dibayar sebagian; klaim lain tidak berubah.
func StatusAfterPayment(cur model.ClaimStatus, gap, paid int64) model.ClaimStatus {
	switch cur {
	case model.ClaimApproved, model.ClaimPartiallyApproved, model.ClaimPartiallyPaid:
	default:
		return cur
	}
	if paid <= 0 {
		return cur
	}
	if paid >= gap {
		return model.ClaimPaid
	}
	return model.ClaimPartiallyPaid
}

// StatusAfterUndo menghitung status klaim setelah pembayaran dibatalkan.
func StatusAfterUndo(cur model.ClaimStatus, paid int64) model.ClaimStatus {
	if cur != model.ClaimPaid && cur != model.ClaimPartiallyPaid {
		return cur
	}
	if paid > 0 {
		return model.ClaimPartiallyPaid
	}
	return model.ClaimApproved
}

// ── Rekonsiliasi mutasi bank ─────────────────────────────────────────────────

// MatchTolerance = toleransi selisih nominal (Rupiah) saat mencocokkan mutasi.
const MatchTolerance int64 = 50_000

// Ambang skor auto-match.
const (
	MatchAutoScore    = 75
	MatchReviewScore  = 40
	MatchMinCandidate = 25
)

// MatchScore menilai kecocokan satu baris mutasi bank dengan sebuah klaim:
// SEP (6 digit terakhir) +40, nama tertanggung +25, no. polis +20, nominal ± toleransi +15.
func MatchScore(description string, amount int64, c *model.Claim, tolerance int64) (int, string) {
	score := 0
	var reasons []string
	desc := strings.ToUpper(description)
	if description != "" && len(c.SEPNo) >= 6 && strings.Contains(desc, c.SEPNo[len(c.SEPNo)-6:]) {
		score += 40
		reasons = append(reasons, "SEP pada deskripsi")
	}
	if description != "" && c.InsuredName != "" && strings.Contains(desc, strings.ToUpper(c.InsuredName)) {
		score += 25
		reasons = append(reasons, "nama tertanggung")
	}
	if description != "" && c.PolicyNo != "" && strings.Contains(desc, strings.ToUpper(c.PolicyNo)) {
		score += 20
		reasons = append(reasons, "no. polis")
	}
	if target := Gap(c); target > 0 {
		diff := amount - target
		if diff < 0 {
			diff = -diff
		}
		if diff <= tolerance {
			score += 15
			reasons = append(reasons, "nominal ± toleransi")
		}
	}
	if score > 100 {
		score = 100
	}
	if len(reasons) == 0 {
		return score, "tanpa indikator kuat"
	}
	return score, strings.Join(reasons, ", ")
}
