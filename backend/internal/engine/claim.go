package engine

import (
	"fmt"
	"regexp"
	"strings"

	"medpay_spkd/internal/model"
)

// ── Transisi status klaim (SRS Appendix B, Gambar B.3; BR-04) ────────────────

// claimTransitions = transisi yang boleh dilakukan pengguna lewat aksi status.
// Sebagian Dibayar & Lunas TIDAK ada di sini: keduanya dihitung otomatis dari
// pembayaran (BR-05) melalui StatusAfterPayment / StatusAfterUndo.
var claimTransitions = map[model.ClaimStatus][]model.ClaimStatus{
	model.ClaimDraft:             {model.ClaimReadyToSubmit, model.ClaimWrittenOff},
	model.ClaimReadyToSubmit:     {model.ClaimSubmitted, model.ClaimDraft, model.ClaimWrittenOff},
	model.ClaimSubmitted:         {model.ClaimApproved, model.ClaimPartiallyApproved, model.ClaimRejected},
	model.ClaimApproved:          {},
	model.ClaimPartiallyApproved: {model.ClaimRejected},
	model.ClaimRejected:          {model.ClaimDisputed, model.ClaimWrittenOff},
	model.ClaimDisputed:          {model.ClaimSubmitted, model.ClaimWrittenOff},
	model.ClaimPartiallyPaid:     {model.ClaimWrittenOff},
	model.ClaimPaid:              {model.ClaimWrittenOff},
	model.ClaimWrittenOff:        {},
}

// CanTransition memeriksa apakah perpindahan status klaim sah.
func CanTransition(from, to model.ClaimStatus) bool {
	for _, s := range claimTransitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// NextStatuses = status tujuan yang sah dari status saat ini.
func NextStatuses(from model.ClaimStatus) []model.ClaimStatus {
	return append([]model.ClaimStatus(nil), claimTransitions[from]...)
}

// TransitionPermission = izin yang dibutuhkan untuk berpindah ke status tujuan:
//   - Siap Diajukan / kembali ke Draft : EDIT   (Staf Billing — maker)
//   - Diajukan                         : APPROVE (Reviewer/Finance/Admin — checker)
//   - Disetujui / Sebagian / Ditolak   : VERIFY  (mencatat keputusan penjamin)
//   - Disanggah (banding)              : REOPEN
//   - Dihapus Buku                     : APPROVE (keputusan final keuangan)
func TransitionPermission(to model.ClaimStatus) string {
	switch to {
	case model.ClaimSubmitted, model.ClaimWrittenOff:
		return model.PermApprove
	case model.ClaimApproved, model.ClaimPartiallyApproved, model.ClaimRejected:
		return model.PermVerify
	case model.ClaimDisputed:
		return model.PermReopen
	default:
		return model.PermEdit
	}
}

// Editable bernilai true bila data klaim masih boleh diubah.
func Editable(s model.ClaimStatus) bool {
	return s == model.ClaimDraft || s == model.ClaimReadyToSubmit || s == model.ClaimDisputed
}

// ── Grouper INA-CBG (DEMO) ───────────────────────────────────────────────────

// Kode error grouper.
const (
	GrouperConfigMissing = "EKLAIM_CONFIG_MISSING"
	GrouperUngroupable   = "UNGROUPABLE"
)

// GrouperResult = hasil grouping INA-CBG.
type GrouperResult struct {
	OK          bool   `json:"ok"`
	Code        string `json:"code,omitempty"`
	BaseTariff  int64  `json:"baseTariff,omitempty"`
	Description string `json:"description,omitempty"`
	Error       string `json:"error,omitempty"`
}

var groupDescriptions = map[byte]string{
	'I': "GROUP JANTUN-ISKEMIK & PENYAKIT PENYERTA", 'J': "GROUP RESPIRATORI INFEKSI/INFLAMASI",
	'E': "GROUP DIABETES & GANGGUAN METABOLIK", 'K': "GROUP GIT & HEPATOBILIARI",
	'N': "GROUP GINJAL & SALURAN KEMIH", 'C': "GROUP NEOPLASMA GANAS", 'S': "GROUP TRAUMA TULANG & SENDI",
	'A': "GROUP INFEKSI & PARASIT", 'G': "GROUP SARAF PUSAT", 'O': "GROUP OBSTETRIK",
}

var groupablePattern = regexp.MustCompile(`(?i)^[A-Z]\d{2}`)

// DemoGrouper meniru grouper e-Klaim INA-CBG secara deterministik.
// DATA SINTETIS: ganti dengan integrasi e-Klaim resmi sebelum dipakai produksi.
func DemoGrouper(dx, careType, careClass string, los int, eklaimConfigured bool) GrouperResult {
	if !eklaimConfigured {
		return GrouperResult{Error: GrouperConfigMissing}
	}
	if !groupablePattern.MatchString(dx) {
		return GrouperResult{Error: GrouperUngroupable}
	}
	chapter := strings.ToUpper(dx[:1])[0]
	inpatient := careType == model.CareTypeRawatInap
	classDigit := "3"
	switch careClass {
	case "I":
		classDigit = "1"
	case "II":
		classDigit = "2"
	}
	severity := "1"
	if los > 6 {
		severity = "2"
	}

	var code string
	mult := int64(750_000)
	if inpatient {
		code = fmt.Sprintf("%c-1-%s%s-%s", chapter, severity, classDigit, careClass)
		mult = 4_500_000
	} else {
		code = fmt.Sprintf("%c-3-1%s-0", chapter, severity)
	}
	seed := (int64(dx[0])*7 + int64(dx[1])*13) % 900
	desc, ok := groupDescriptions[chapter]
	if !ok {
		desc = "GROUP INA-CBG TERPILIH (DEMO)"
	}
	return GrouperResult{
		OK:          true,
		Code:        code,
		BaseTariff:  mult + seed*5000 + int64(los)*120_000,
		Description: desc,
	}
}
