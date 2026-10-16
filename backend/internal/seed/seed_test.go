package seed

import (
	"strings"
	"testing"
	"time"

	"medpay_spkd/internal/engine"
	"medpay_spkd/internal/model"
)

// Tes integritas data demo (tanpa database): menangkap referensi yang putus, ID ganda,
// dan klaim tambahan yang ternyata punya blocker SEBELUM seeder dijalankan. Seeder berjalan
// dalam satu transaksi — kesalahan semacam ini akan membuat seluruh seed di-rollback dan
// server gagal start, jadi dicek di sini.

func idset[T any](items []T, id func(T) string, t *testing.T, what string) map[string]bool {
	t.Helper()
	m := map[string]bool{}
	for _, it := range items {
		k := id(it)
		if k == "" {
			t.Errorf("%s: ada ID kosong", what)
		}
		if m[k] {
			t.Errorf("%s: ID ganda %q", what, k)
		}
		m[k] = true
	}
	return m
}

func mustDate(t *testing.T, what, v string) time.Time {
	t.Helper()
	layout := "2006-01-02"
	if len(v) > len(layout) {
		layout = "2006-01-02T15:04:05"
	}
	d, err := time.Parse(layout, v)
	if err != nil {
		t.Fatalf("%s: tanggal tidak valid %q", what, v)
	}
	return d
}

func TestDemoDataIntegrity(t *testing.T) {
	d := demoDataset()

	hosp := idset(d.Hospitals, func(x seedHospital) string { return x.ID }, t, "hospitals")
	users := idset(d.Users, func(x seedUser) string { return x.ID }, t, "users")
	ins := idset(d.Insurers, func(x seedInsurer) string { return x.ID }, t, "insurers")
	batches := idset(d.Batches, func(x seedBatch) string { return x.ID }, t, "batches")
	claims := idset(d.Claims, func(x seedClaim) string { return x.ID }, t, "claims")
	idset(d.Payments, func(x seedPayment) string { return x.ID }, t, "payments")
	imports := idset(d.BankImports, func(x seedBankImport) string { return x.ID }, t, "bank_imports")
	idset(d.BankLines, func(x seedBankLine) string { return x.ID }, t, "bank_lines")
	idset(d.WorkItems, func(x seedWorkItem) string { return x.ID }, t, "work_items")

	names, emails := map[string]bool{"Sistem": true}, map[string]bool{}
	for _, u := range d.Users {
		names[u.Name] = true
		emails[u.Email] = true
	}
	rejCodes := map[string]bool{}
	for _, r := range seedRejectionCodes {
		rejCodes[r.Code] = true
	}
	cats := map[string]bool{}
	for _, c := range resumeCategories {
		cats[c.code] = true
	}
	insByID := map[string]seedInsurer{}
	for _, i := range d.Insurers {
		insByID[i.ID] = i
	}
	sepSeen := map[string]string{}

	for _, c := range d.Claims {
		if !hosp[c.HospitalID] {
			t.Errorf("klaim %s: RS %q tidak ada", c.ID, c.HospitalID)
		}
		if c.InsurerID != "" && !ins[c.InsurerID] {
			t.Errorf("klaim %s: penjamin %q tidak ada", c.ID, c.InsurerID)
		}
		if c.BatchID != nil && !batches[*c.BatchID] {
			t.Errorf("klaim %s: batch %q tidak ada", c.ID, *c.BatchID)
		}
		if !names[c.CreatedBy] {
			t.Errorf("klaim %s: pembuat %q tidak ada", c.ID, c.CreatedBy)
		}
		if c.RejectionReasonCode != "" && !rejCodes[c.RejectionReasonCode] {
			t.Errorf("klaim %s: kode penolakan %q tidak dikenal", c.ID, c.RejectionReasonCode)
		}
		for _, h := range c.StatusHistory {
			mustDate(t, c.ID+" riwayat", h.At)
			if !names[h.Actor] && !strings.HasPrefix(h.Actor, "Sistem") { // aktor "Sistem…" disimpan NULL oleh seeder
				t.Errorf("klaim %s: aktor riwayat %q tidak ada", c.ID, h.Actor)
			}
		}
		if prev, dup := sepSeen[c.HospitalID+c.SEPNo]; dup {
			t.Errorf("SEP ganda %s antara %s dan %s", c.SEPNo, prev, c.ID)
		}
		sepSeen[c.HospitalID+c.SEPNo] = c.ID
	}
	for _, b := range d.Batches {
		if !hosp[b.HospitalID] || !names[b.CreatedBy] {
			t.Errorf("batch %s: RS/pembuat tidak valid", b.ID)
		}
		if b.InsurerID != "" {
			if !ins[b.InsurerID] {
				t.Errorf("batch %s: penjamin %q tidak ada", b.ID, b.InsurerID)
			}
			from, to := mustDate(t, b.ID, b.PeriodStart), mustDate(t, b.ID, b.PeriodEnd)
			if to.Before(from) {
				t.Errorf("batch %s: periode terbalik", b.ID)
			}
			// Klaim di batch per-penjamin harus milik penjamin yang sama & tanggal pulang dalam periode.
			for _, c := range d.Claims {
				if c.BatchID == nil || *c.BatchID != b.ID {
					continue
				}
				if c.InsurerID != b.InsurerID {
					t.Errorf("klaim %s di batch %s tapi penjamin berbeda", c.ID, b.ID)
				}
				if dd := mustDate(t, c.ID, c.DischargeDate); dd.Before(from) || dd.After(to) {
					t.Errorf("klaim %s: tanggal pulang %s di luar periode batch %s", c.ID, c.DischargeDate, b.ID)
				}
			}
		} else if b.Name == "" || b.Period == "" {
			t.Errorf("batch lama %s wajib punya nama & periode", b.ID)
		}
	}
	for _, r := range d.ResumeItems {
		if !claims[r.ClaimID] || !cats[r.Kategori] {
			t.Errorf("resume %s/%s: klaim atau kategori tidak valid", r.ClaimID, r.Kategori)
		}
	}
	paid := map[string]int64{}
	for _, p := range d.Payments {
		if !claims[p.ClaimID] || !names[p.CreatedBy] {
			t.Errorf("pembayaran %s: klaim/pembuat tidak valid", p.ID)
		}
		mustDate(t, p.ID, p.ReceivedDate)
		paid[p.ClaimID] += p.Amount
	}
	for _, i := range d.BankImports {
		_ = i
	}
	for _, l := range d.BankLines {
		if !imports[l.ImportID] {
			t.Errorf("baris mutasi %s: impor %q tidak ada", l.ID, l.ImportID)
		}
		if l.MatchClaimID != nil && !claims[*l.MatchClaimID] {
			t.Errorf("baris mutasi %s: klaim %q tidak ada", l.ID, *l.MatchClaimID)
		}
	}
	for _, w := range d.WorkItems {
		if w.ClaimID != nil && !claims[*w.ClaimID] {
			t.Errorf("work item %s: klaim %q tidak ada", w.ID, *w.ClaimID)
		}
		if w.Assignee != nil && !users[*w.Assignee] {
			t.Errorf("work item %s: assignee %q tidak ada", w.ID, *w.Assignee)
		}
		for _, c := range w.Comments {
			if !names[c.Author] {
				t.Errorf("work item %s: penulis komentar %q tidak ada", w.ID, c.Author)
			}
		}
	}
	for _, n := range d.Notifications {
		if n.ClaimID != nil && !claims[*n.ClaimID] {
			t.Errorf("notifikasi: klaim %q tidak ada", *n.ClaimID)
		}
		if !emails[n.Recipient] {
			t.Errorf("notifikasi: penerima %q tidak ada", n.Recipient)
		}
	}

	// Konsistensi status ↔ pembayaran untuk klaim tambahan.
	for _, sp := range extraSpecs {
		switch sp.Status {
		case "paid":
			if paid[sp.ID] != sp.Gap {
				t.Errorf("%s Lunas tapi dibayar %d dari gap %d", sp.ID, paid[sp.ID], sp.Gap)
			}
		case "partially_paid":
			if paid[sp.ID] <= 0 || paid[sp.ID] >= sp.Gap {
				t.Errorf("%s Sebagian Dibayar tapi dibayar %d dari gap %d", sp.ID, paid[sp.ID], sp.Gap)
			}
		default:
			if paid[sp.ID] != 0 {
				t.Errorf("%s status %s tidak boleh punya pembayaran", sp.ID, sp.Status)
			}
		}
	}
}

// Klaim tambahan dirancang bebas blocker; tes ini menjalankan scrubber sungguhan terhadapnya.
func TestExtraClaimsHaveNoBlockers(t *testing.T) {
	d := demoDataset()
	today := mustDate(t, "seedDate", seedDate)

	rules := map[string]model.ScrubRule{}
	for _, r := range seedScrubRules {
		rules[r.ID] = model.ScrubRule{Code: r.ID, Severity: model.Severity(r.Severity), Enabled: true}
	}
	sepCount := map[string]int{}
	for _, c := range d.Claims {
		sepCount[c.HospitalID+c.SEPNo]++
	}
	resume := map[string][]model.ResumeItem{}
	for _, r := range d.ResumeItems {
		resume[r.ClaimID] = append(resume[r.ClaimID], model.ResumeItem{Reviewed: r.Reviewed})
	}
	insurers := map[string]seedInsurer{}
	for _, i := range d.Insurers {
		insurers[i.ID] = i
	}

	extra := map[string]bool{}
	for _, sp := range extraSpecs {
		extra[sp.ID] = true
	}
	for _, sc := range d.Claims {
		if !extra[sc.ID] {
			continue
		}
		adm, dis := mustDate(t, sc.ID, sc.AdmissionDate), mustDate(t, sc.ID, sc.DischargeDate)
		dob := mustDate(t, sc.ID, sc.DateOfBirth)
		resolved := mustDate(t, sc.ID, sc.TariffResolvedAt)
		c := &model.Claim{
			SEPNo: sc.SEPNo, BPJSCardNo: sc.BPJSCardNo, MRN: sc.MRN, PatientName: sc.PatientName,
			ParticipantType: sc.ParticipantType, CareType: sc.CareType, CareClass: sc.CareClass,
			DateOfBirth: &dob, AdmissionDate: &adm, DischargeDate: &dis, LOS: sc.LOS, PrimaryDiagnosis: sc.PrimaryDiagnosis,
			INACBGCode: sc.INACBGCode, INACBGBaseTariff: sc.INACBGBaseTariff, SpecialCMG: model.SpecialCMG(sc.SpecialCMG),
			HospitalTariff: model.HospitalTariff(sc.HospitalTariff), TariffResolvedAt: &resolved,
			ResolutionSignature: sc.ResolutionSignature, DeclarationChecked: sc.DeclarationChecked,
			PolicyNo: sc.PolicyNo, InsuredName: sc.InsuredName, CoverageType: sc.CoverageType,
			PolicyCeiling: sc.PolicyCeiling, Status: model.ClaimStatus(sc.Status),
		}
		for i, dx := range sc.SecondaryDiagnoses {
			c.SecondaryDiagnoses = append(c.SecondaryDiagnoses, model.ClaimSecondaryDiagnosis{ICD10Code: dx, Seq: i + 1})
		}
		for i, px := range sc.Procedures {
			c.Procedures = append(c.Procedures, model.ClaimProcedure{ICD9Code: px, Seq: i + 1})
		}
		if sig := engine.TariffSignature(c); sig != sc.ResolutionSignature {
			t.Errorf("%s: tanda tangan resolusi %q ≠ %q", sc.ID, sc.ResolutionSignature, sig)
		}
		if got, want := engine.Gap(c), gapOf(sc.ID); got != want {
			t.Errorf("%s: gap %d ≠ %d", sc.ID, got, want)
		}
		in := insurers[sc.InsurerID]
		res := engine.Scrub(c, engine.ScrubContext{
			Rules: rules, Resume: resume[sc.ID], DuplicateSEP: sepCount[sc.HospitalID+sc.SEPNo] > 1,
			Insurer: &model.Insurer{DeadlineDays: in.DeadlineDays}, Today: today,
		})
		if res.HasOpenBlocker {
			t.Errorf("%s harus bebas blocker, tapi ada: %v", sc.ID, res.OpenBlockers())
		}
	}
}

func gapOf(id string) int64 {
	for _, sp := range extraSpecs {
		if sp.ID == id {
			return sp.Gap
		}
	}
	return -1
}
