package engine

import (
	"regexp"
	"strings"
	"time"

	"medpay_spkd/internal/model"
)

var (
	icd10Pattern = regexp.MustCompile(`^[A-Z]\d{2}(\.\d{1,2})?$`)
	icd9Pattern  = regexp.MustCompile(`^\d{2}(\.\d{1,2})?$`)
)

// ScrubContext = data di luar klaim yang dibutuhkan scrubber.
type ScrubContext struct {
	Rules        map[string]model.ScrubRule // kunci = kode aturan (R01..R14)
	Overrides    map[string]bool            // kode aturan yang di-override untuk klaim ini
	Resume       []model.ResumeItem         // item resume medis klaim
	DuplicateSEP bool                       // ada klaim lain (RS sama, belum dihapus buku) dengan SEP sama
	Insurer      *model.Insurer
	Today        time.Time
}

// Finding = satu temuan scrubber.
type Finding struct {
	RuleID        string `json:"ruleId"`
	Severity      string `json:"severity"`
	Message       string `json:"message"`
	FixSuggestion string `json:"fixSuggestion"`
	SourceRef     string `json:"sourceRef"`
	Overridden    bool   `json:"overridden"`
}

// ScrubResult = hasil scrub satu klaim.
type ScrubResult struct {
	Findings       []Finding `json:"findings"`
	Score          int       `json:"score"` // readiness score 0–100
	HasOpenBlocker bool      `json:"hasOpenBlocker"`
}

// OpenBlockers mengembalikan kode aturan BLOCKER yang belum di-override.
func (r ScrubResult) OpenBlockers() []string {
	var out []string
	for _, f := range r.Findings {
		if f.Severity == string(model.SeverityBlocker) && !f.Overridden {
			out = append(out, f.RuleID)
		}
	}
	return out
}

// Scrub menjalankan 14 aturan validasi klaim. Aturan yang dinonaktifkan dilewati;
// temuan yang di-override tetap tampil tetapi tidak mengurangi skor.
// Skor = 100 − 25 × blocker − 8 × warning (minimal 0).
func Scrub(c *model.Claim, ctx ScrubContext) ScrubResult {
	res := ScrubResult{Findings: []Finding{}}
	run := func(code string, cond bool) {
		if !cond {
			return
		}
		r, ok := ctx.Rules[code]
		if !ok || !r.Enabled {
			return
		}
		res.Findings = append(res.Findings, Finding{
			RuleID:        r.Code,
			Severity:      string(r.Severity),
			Message:       r.Message,
			FixSuggestion: r.FixSuggestion,
			SourceRef:     r.SourceRef,
			Overridden:    ctx.Overrides[r.Code],
		})
	}

	// R01 — field wajib.
	run("R01", c.SEPNo == "" || c.BPJSCardNo == "" || c.MRN == "" || c.PatientName == "" ||
		c.AdmissionDate == nil || c.DischargeDate == nil || c.DateOfBirth == nil)

	// R02 — LOS tidak sesuai tanggal.
	run("R02", c.AdmissionDate != nil && c.DischargeDate != nil && LOS(c.AdmissionDate, c.DischargeDate) != c.LOS)

	// R03 — peserta PBI tidak eligible.
	run("R03", c.ParticipantType == model.ParticipantPBI)

	// R04 — SEP ganda.
	run("R04", c.SEPNo != "" && ctx.DuplicateSEP)

	// R05 — format ICD-10 / ICD-9-CM.
	secondary := SecondaryCodes(c)
	procedures := ProcedureCodes(c)
	primaryUp := strings.ToUpper(c.PrimaryDiagnosis)
	bad := c.PrimaryDiagnosis != "" && !icd10Pattern.MatchString(c.PrimaryDiagnosis)
	for _, d := range secondary {
		if !icd10Pattern.MatchString(d) || (primaryUp != "" && strings.ToUpper(d) == primaryUp) {
			bad = true
		}
	}
	for _, p := range procedures {
		if p != "" && !icd9Pattern.MatchString(p) {
			bad = true
		}
	}
	run("R05", bad)

	// R06 — tarif belum di-resolve atau basi.
	stale := c.ResolutionSignature != "" && c.ResolutionSignature != TariffSignature(c)
	run("R06", c.INACBGCode == "" || c.TariffResolvedAt == nil || stale)

	// R07 — kode INA-CBG tidak sesuai jenis rawat (kode rawat jalan berakhiran "-0").
	if c.INACBGCode != "" {
		outCode := strings.HasSuffix(strings.TrimSpace(c.INACBGCode), "-0")
		run("R07", (c.CareType == model.CareTypeRawatInap && outCode) ||
			(c.CareType == model.CareTypeRawatJalan && !outCode))
	}

	// R08 — tagihan RS di bawah tarif JKN.
	hospitalTotal := SumTariff(c.HospitalTariff)
	jknTotal := TotalJKN(c.INACBGBaseTariff, c.SpecialCMG)
	run("R08", hospitalTotal != 0 && hospitalTotal < jknTotal)

	// R09 — gap melebihi plafon polis.
	gap := hospitalTotal - jknTotal
	run("R09", gap > 0 && c.PolicyCeiling != nil && *c.PolicyCeiling > 0 && *c.PolicyCeiling < gap)

	// R10 — resume medis kosong / belum direview.
	unreviewed := len(ctx.Resume) == 0
	for _, r := range ctx.Resume {
		if !r.Reviewed {
			unreviewed = true
		}
	}
	run("R10", unreviewed)

	// R11 — mendekati / melewati batas 6 bulan.
	run("R11", Deadline(c.AdmissionDate, ctx.Today).Near())

	// R12 — selisih kelas tidak berlaku untuk kelas III.
	run("R12", c.CareClass == "III" && strings.Contains(strings.ToLower(c.CoverageType), "selisih kelas"))

	// R13 — deklarasi & data polis.
	run("R13", !c.DeclarationChecked ||
		(c.Status != model.ClaimDraft && (c.PolicyNo == "" || c.InsuredName == "")))

	// R14 — melewati batas hari pengajuan penjamin.
	if ctx.Insurer != nil && c.AdmissionDate != nil &&
		(c.Status == model.ClaimDraft || c.Status == model.ClaimReadyToSubmit) {
		run("R14", DaysBetween(*c.AdmissionDate, ctx.Today) > ctx.Insurer.DeadlineDays)
	}

	blockers, warnings := 0, 0
	for _, f := range res.Findings {
		if f.Overridden {
			continue
		}
		switch f.Severity {
		case string(model.SeverityBlocker):
			blockers++
		case string(model.SeverityWarning):
			warnings++
		}
	}
	res.Score = 100 - blockers*25 - warnings*8
	if res.Score < 0 {
		res.Score = 0
	}
	res.HasOpenBlocker = blockers > 0
	return res
}
