package engine

import (
	"strings"
	"testing"
	"time"

	"medpay_spkd/internal/model"
)

func date(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02", s, Location())
	if err != nil {
		panic(err)
	}
	return t
}

func datePtr(s string) *time.Time {
	t := date(s)
	return &t
}

func i64(v int64) *int64 { return &v }

// allRules mengaktifkan R01..R14 dengan severity sesuai data demo.
func allRules() map[string]model.ScrubRule {
	sev := map[string]model.Severity{
		"R01": model.SeverityBlocker, "R02": model.SeverityWarning, "R03": model.SeverityBlocker,
		"R04": model.SeverityBlocker, "R05": model.SeverityBlocker, "R06": model.SeverityBlocker,
		"R07": model.SeverityBlocker, "R08": model.SeverityWarning, "R09": model.SeverityWarning,
		"R10": model.SeverityWarning, "R11": model.SeverityWarning, "R12": model.SeverityWarning,
		"R13": model.SeverityBlocker, "R14": model.SeverityWarning,
	}
	out := map[string]model.ScrubRule{}
	for code, s := range sev {
		out[code] = model.ScrubRule{Code: code, Severity: s, Enabled: true, Message: "msg " + code}
	}
	return out
}

// cleanClaim = klaim rawat inap lengkap tanpa temuan scrub.
func cleanClaim() *model.Claim {
	c := &model.Claim{
		SEPNo: "0301R001000188", BPJSCardNo: "000210000137", MRN: "RM-1", PatientName: "Pasien Uji",
		ParticipantType: model.ParticipantNonPBI, CareType: model.CareTypeRawatInap, CareClass: "I",
		DateOfBirth: datePtr("1960-01-01"), AdmissionDate: datePtr("2026-09-01"), DischargeDate: datePtr("2026-09-05"),
		LOS: 4, PrimaryDiagnosis: "I25.1",
		SecondaryDiagnoses: []model.ClaimSecondaryDiagnosis{{ICD10Code: "I10", Seq: 2}, {ICD10Code: "E11.9", Seq: 1}},
		Procedures:         []model.ClaimProcedure{{ICD9Code: "88.55", Seq: 1}},
		INACBGCode:         "I-1-11-I", INACBGBaseTariff: 6_000_000,
		HospitalTariff:     model.HospitalTariff{ProsedurBedah: 9_000_000, Obat: 1_000_000},
		DeclarationChecked: true, PolicyNo: "POL-1", InsuredName: "Pasien Uji",
		Status: model.ClaimDraft,
	}
	now := date("2026-09-06")
	c.TariffResolvedAt = &now
	c.ResolutionSignature = TariffSignature(c)
	return c
}

func cleanCtx() ScrubContext {
	return ScrubContext{
		Rules:   allRules(),
		Resume:  []model.ResumeItem{{Reviewed: true}},
		Insurer: &model.Insurer{DeadlineDays: 150},
		Today:   date("2026-10-01"),
	}
}

func ruleIDs(r ScrubResult) string {
	var ids []string
	for _, f := range r.Findings {
		ids = append(ids, f.RuleID)
	}
	return strings.Join(ids, ",")
}

func TestTariffChain(t *testing.T) {
	c := cleanClaim()
	c.SpecialCMG = model.SpecialCMG{SpecialProcedure: 500_000, Chronic: 100_000}
	if got := SumTariff(c.HospitalTariff); got != 10_000_000 {
		t.Fatalf("SumTariff = %d", got)
	}
	if got := TotalJKN(c.INACBGBaseTariff, c.SpecialCMG); got != 6_600_000 {
		t.Fatalf("TotalJKN = %d", got)
	}
	if got := Gap(c); got != 3_400_000 {
		t.Fatalf("Gap = %d", got)
	}
}

func TestTariffSignatureFollowsSeq(t *testing.T) {
	c := cleanClaim()
	if got, want := TariffSignature(c), "I25.1|E11.9,I10|88.55|4|RAWAT_INAP|I"; got != want {
		t.Fatalf("signature = %q, want %q", got, want)
	}
}

func TestLOSAndDays(t *testing.T) {
	if got := LOS(datePtr("2026-09-01"), datePtr("2026-09-05")); got != 4 {
		t.Fatalf("LOS = %d", got)
	}
	if got := LOS(datePtr("2026-09-05"), datePtr("2026-09-01")); got != 0 {
		t.Fatalf("LOS negatif harus 0, dapat %d", got)
	}
	if got := LOS(nil, datePtr("2026-09-01")); got != 0 {
		t.Fatalf("LOS tanpa tanggal masuk = %d", got)
	}
	// Jam tidak memengaruhi selisih hari kalender.
	a := time.Date(2026, 1, 1, 23, 59, 0, 0, Location())
	b := time.Date(2026, 1, 2, 0, 1, 0, 0, Location())
	if got := DaysBetween(a, b); got != 1 {
		t.Fatalf("DaysBetween = %d", got)
	}
}

func TestDeadlineZones(t *testing.T) {
	adm := datePtr("2026-04-10") // deadline 2026-10-10
	cases := []struct {
		today string
		zone  string
		left  int
	}{
		{"2026-08-01", ZoneAman, 70},
		{"2026-09-15", ZoneWaspada, 25},
		{"2026-10-05", ZoneKritis, 5},
		{"2026-10-11", ZoneLewat, -1},
	}
	for _, tc := range cases {
		d := Deadline(adm, date(tc.today))
		if d.Zone != tc.zone || d.DaysLeft != tc.left {
			t.Errorf("today %s: zone=%s left=%d, want %s %d", tc.today, d.Zone, d.DaysLeft, tc.zone, tc.left)
		}
	}
	if d := Deadline(nil, date("2026-01-01")); d.Zone != ZoneAman || d.Deadline != nil {
		t.Errorf("tanpa tanggal masuk harus aman, dapat %+v", d)
	}
	// Akhir bulan: 31 Agustus + 6 bulan dinormalisasi ke awal Maret.
	if d := Deadline(datePtr("2025-08-31"), date("2026-01-01")); d.Deadline.Format("2006-01-02") != "2026-03-03" {
		t.Errorf("deadline akhir bulan = %s", d.Deadline.Format("2006-01-02"))
	}
}

func TestScrubCleanClaim(t *testing.T) {
	r := Scrub(cleanClaim(), cleanCtx())
	if len(r.Findings) != 0 || r.Score != 100 || r.HasOpenBlocker {
		t.Fatalf("klaim bersih harus tanpa temuan, dapat %s skor %d", ruleIDs(r), r.Score)
	}
}

func TestScrubRules(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(c *model.Claim, ctx *ScrubContext)
		want   string
	}{
		{"R01 field wajib", func(c *model.Claim, _ *ScrubContext) { c.MRN = "" }, "R01"},
		{"R02 LOS salah", func(c *model.Claim, ctx *ScrubContext) {
			c.LOS = 9
			c.ResolutionSignature = TariffSignature(c)
		}, "R02"},
		{"R03 PBI", func(c *model.Claim, _ *ScrubContext) { c.ParticipantType = model.ParticipantPBI }, "R03"},
		{"R04 SEP ganda", func(_ *model.Claim, ctx *ScrubContext) { ctx.DuplicateSEP = true }, "R04"},
		{"R05 diagnosis sekunder = primer", func(c *model.Claim, _ *ScrubContext) {
			c.SecondaryDiagnoses = []model.ClaimSecondaryDiagnosis{{ICD10Code: "i25.1", Seq: 1}}
			c.ResolutionSignature = TariffSignature(c)
		}, "R05"},
		{"R05 prosedur salah format", func(c *model.Claim, _ *ScrubContext) {
			c.Procedures = []model.ClaimProcedure{{ICD9Code: "8855", Seq: 1}}
			c.ResolutionSignature = TariffSignature(c)
		}, "R05"},
		{"R06 tarif basi", func(c *model.Claim, _ *ScrubContext) { c.PrimaryDiagnosis = "I21.0" }, "R06"},
		{"R07 kode rawat jalan untuk rawat inap", func(c *model.Claim, _ *ScrubContext) { c.INACBGCode = "I-3-11-0" }, "R07"},
		{"R08 tagihan di bawah JKN", func(c *model.Claim, _ *ScrubContext) {
			c.HospitalTariff = model.HospitalTariff{Obat: 1_000_000}
		}, "R08"},
		{"R09 gap melebihi plafon", func(c *model.Claim, _ *ScrubContext) { c.PolicyCeiling = i64(1_000_000) }, "R09"},
		{"R10 resume belum direview", func(_ *model.Claim, ctx *ScrubContext) {
			ctx.Resume = []model.ResumeItem{{Reviewed: true}, {Reviewed: false}}
		}, "R10"},
		{"R10 resume kosong", func(_ *model.Claim, ctx *ScrubContext) { ctx.Resume = nil }, "R10"},
		{"R11 tidak muncul jauh sebelum deadline", func(_ *model.Claim, ctx *ScrubContext) {
			ctx.Today = date("2026-02-20")
			ctx.Insurer = nil
		}, ""},
		{"R12 selisih kelas kelas III", func(c *model.Claim, _ *ScrubContext) {
			c.CareClass = "III"
			c.CoverageType = "Selisih Kelas"
			c.INACBGCode = "I-1-13-III"
			c.ResolutionSignature = TariffSignature(c)
		}, "R12"},
		{"R13 deklarasi", func(c *model.Claim, _ *ScrubContext) { c.DeclarationChecked = false }, "R13"},
		{"R13 polis kosong setelah draft", func(c *model.Claim, _ *ScrubContext) {
			c.Status = model.ClaimSubmitted
			c.PolicyNo = ""
		}, "R13"},
		{"R14 lewat batas penjamin", func(_ *model.Claim, ctx *ScrubContext) {
			ctx.Insurer = &model.Insurer{DeadlineDays: 10}
		}, "R14"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, ctx := cleanClaim(), cleanCtx()
			tc.mutate(c, &ctx)
			got := ruleIDs(Scrub(c, ctx))
			if got != tc.want {
				t.Fatalf("temuan = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestScrubDeadlineWindow(t *testing.T) {
	c, ctx := cleanClaim(), cleanCtx()
	ctx.Insurer = nil
	ctx.Today = date("2027-02-20") // deadline 2027-03-01 → waspada
	if got := ruleIDs(Scrub(c, ctx)); got != "R11" {
		t.Fatalf("temuan = %q, want R11", got)
	}
}

func TestScrubScoreAndOverride(t *testing.T) {
	c, ctx := cleanClaim(), cleanCtx()
	c.MRN = ""               // R01 blocker
	c.PolicyCeiling = i64(1) // R09 warning
	ctx.Resume = nil         // R10 warning
	r := Scrub(c, ctx)
	if r.Score != 100-25-8-8 || !r.HasOpenBlocker {
		t.Fatalf("skor = %d blocker=%v", r.Score, r.HasOpenBlocker)
	}
	if got := strings.Join(r.OpenBlockers(), ","); got != "R01" {
		t.Fatalf("open blockers = %q", got)
	}

	ctx.Overrides = map[string]bool{"R01": true}
	r = Scrub(c, ctx)
	if r.HasOpenBlocker || r.Score != 100-8-8 {
		t.Fatalf("setelah override: skor=%d blocker=%v", r.Score, r.HasOpenBlocker)
	}
	if !r.Findings[0].Overridden {
		t.Fatal("temuan R01 harus tetap tampil dengan tanda overridden")
	}
}

func TestScrubDisabledRule(t *testing.T) {
	c, ctx := cleanClaim(), cleanCtx()
	c.MRN = ""
	r := ctx.Rules["R01"]
	r.Enabled = false
	ctx.Rules["R01"] = r
	if got := ruleIDs(Scrub(c, ctx)); got != "" {
		t.Fatalf("aturan nonaktif tidak boleh menghasilkan temuan, dapat %q", got)
	}
}

func TestSplitKAPJ(t *testing.T) {
	p := DefaultKAPJ
	cases := []struct {
		name           string
		bill, jkn      int64
		ceiling        *int64
		care           string
		params         KAPJParams
		bpjs, ins, pat int64
		status         string
	}{
		{"co-pay persentase", 10_000_000, 8_000_000, nil, model.CareTypeRawatInap, p, 6_000_000, 3_800_000, 200_000, KAPJSebagian},
		{"co-pay dibatasi cap rawat jalan", 20_000_000, 4_000_000, nil, model.CareTypeRawatJalan, p, 3_000_000, 16_700_000, 300_000, KAPJCoPayMaks},
		{"dibatasi plafon", 20_000_000, 8_000_000, i64(5_000_000), model.CareTypeRawatInap, p, 6_000_000, 4_750_000, 250_000, KAPJDibatasiPlafon},
		{"tanpa co-pay", 10_000_000, 8_000_000, nil, model.CareTypeRawatInap,
			KAPJParams{BPJSSharePct: 100}, 8_000_000, 2_000_000, 0, KAPJLunasPenuh},
		{"tagihan di bawah porsi BPJS", 1_000_000, 8_000_000, nil, model.CareTypeRawatInap,
			KAPJParams{BPJSSharePct: 100}, 8_000_000, 0, 0, KAPJLunasPenuh},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := SplitKAPJ(tc.bill, tc.jkn, tc.ceiling, tc.care, tc.params)
			if s.BPJSPays != tc.bpjs || s.InsurerPays != tc.ins || s.PatientPays != tc.pat || s.Status != tc.status {
				t.Fatalf("dapat %+v", s)
			}
		})
	}
}

func TestAgingAndDSO(t *testing.T) {
	today := date("2026-10-01")
	buckets, total := AgingBuckets([]Receivable{
		{Amount: 100, Basis: date("2026-09-20")}, // 11 hari
		{Amount: 200, Basis: date("2026-08-01")}, // 61 hari
		{Amount: 300, Basis: date("2026-01-01")}, // > 90
		{Amount: 999, Basis: date("2026-12-01")}, // masa depan: tidak dihitung
	}, today)
	if total != 600 {
		t.Fatalf("total = %d", total)
	}
	want := []int64{100, 0, 200, 300}
	for i, b := range buckets {
		if b.Amount != want[i] {
			t.Errorf("bucket %s = %d, want %d", b.Label, b.Amount, want[i])
		}
	}

	if DSO(nil) != nil {
		t.Fatal("DSO tanpa data harus nil")
	}
	d := DSO([]PaidPair{
		{date("2026-01-01"), date("2026-01-31")},
		{date("2026-02-01"), date("2026-02-11")},
		{date("2026-03-10"), date("2026-03-01")}, // dibayar sebelum diajukan: diabaikan
	})
	if d == nil || *d != 20 {
		t.Fatalf("DSO = %v", d)
	}
}

func TestPaymentStatus(t *testing.T) {
	if s := StatusAfterPayment(model.ClaimApproved, 1000, 400); s != model.ClaimPartiallyPaid {
		t.Errorf("sebagian: %s", s)
	}
	if s := StatusAfterPayment(model.ClaimPartiallyPaid, 1000, 1000); s != model.ClaimPaid {
		t.Errorf("lunas: %s", s)
	}
	if s := StatusAfterPayment(model.ClaimSubmitted, 1000, 1000); s != model.ClaimSubmitted {
		t.Errorf("klaim belum disetujui tidak boleh berubah: %s", s)
	}
	if s := StatusAfterUndo(model.ClaimPaid, 0); s != model.ClaimApproved {
		t.Errorf("undo semua: %s", s)
	}
	if s := StatusAfterUndo(model.ClaimPaid, 10); s != model.ClaimPartiallyPaid {
		t.Errorf("undo sebagian: %s", s)
	}
}

func TestMatchScore(t *testing.T) {
	c := cleanClaim() // gap 4.000.000
	score, reason := MatchScore("TRF AIA KLAIM SEP 000188 PASIEN UJI POL-1", 4_020_000, c, MatchTolerance)
	if score != 100 || !strings.Contains(reason, "SEP") {
		t.Fatalf("skor = %d (%s)", score, reason)
	}
	score, reason = MatchScore("TRANSFER LAIN", 1, c, MatchTolerance)
	if score != 0 || reason != "tanpa indikator kuat" {
		t.Fatalf("skor = %d (%s)", score, reason)
	}
}

func TestTransitions(t *testing.T) {
	if !CanTransition(model.ClaimDraft, model.ClaimReadyToSubmit) {
		t.Error("draft → ready_to_submit harus sah")
	}
	if CanTransition(model.ClaimDraft, model.ClaimPaid) {
		t.Error("draft → paid tidak boleh sah")
	}
	if CanTransition(model.ClaimWrittenOff, model.ClaimDraft) {
		t.Error("written_off adalah status akhir")
	}
	// Lunas/Sebagian Dibayar hanya dari pembayaran, bukan aksi manual (Gambar B.3).
	if CanTransition(model.ClaimApproved, model.ClaimPaid) || CanTransition(model.ClaimApproved, model.ClaimPartiallyPaid) {
		t.Error("status pembayaran tidak boleh diubah manual")
	}
	// Disanggah hanya dari Ditolak (banding), lalu diajukan ulang.
	if CanTransition(model.ClaimSubmitted, model.ClaimDisputed) || !CanTransition(model.ClaimDisputed, model.ClaimSubmitted) {
		t.Error("alur banding tidak sesuai diagram")
	}
	if TransitionPermission(model.ClaimSubmitted) != model.PermApprove ||
		TransitionPermission(model.ClaimRejected) != model.PermVerify ||
		TransitionPermission(model.ClaimDisputed) != model.PermReopen ||
		TransitionPermission(model.ClaimReadyToSubmit) != model.PermEdit {
		t.Error("izin transisi tidak sesuai")
	}
}

func TestDemoGrouper(t *testing.T) {
	g := DemoGrouper("I25.1", model.CareTypeRawatInap, "I", 5, true)
	if !g.OK || g.Code != "I-1-11-I" {
		t.Fatalf("grouper = %+v", g)
	}
	// Seed dari kode ASCII 'I'(73) dan '2'(50): (73*7+50*13)%900 = 261.
	if want := int64(4_500_000 + 261*5000 + 5*120_000); g.BaseTariff != want {
		t.Fatalf("tarif = %d, want %d", g.BaseTariff, want)
	}
	if g := DemoGrouper("J18.9", model.CareTypeRawatJalan, "", 0, true); g.Code != "J-3-11-0" {
		t.Fatalf("rawat jalan = %+v", g)
	}
	if g := DemoGrouper("I25.1", model.CareTypeRawatInap, "I", 1, false); g.Error != GrouperConfigMissing {
		t.Fatalf("tanpa konfigurasi e-Klaim = %+v", g)
	}
	if g := DemoGrouper("123", model.CareTypeRawatInap, "I", 1, true); g.Error != GrouperUngroupable {
		t.Fatalf("diagnosis tidak valid = %+v", g)
	}
}
