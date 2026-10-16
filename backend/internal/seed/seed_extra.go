package seed

// Data demo tambahan: klaim bersih (bebas blocker) lintas status, batch per-penjamin,
// pembayaran per batch, impor mutasi, antrean kerja, dan notifikasi — agar setiap halaman
// (Batch & Pengajuan, Pembayaran, Rekonsiliasi, Piutang, Dashboard, Antrean Kerja) terisi.
// DATA SINTETIS untuk lokal/UAT. Tanggal ditulis relatif terhadap seedDate dan digeser
// ke hari ini saat seeding (lihat seed.go). Klaim dibangun dari tabel spesifikasi kecil
// (extraSpecs) lewat buildExtraClaim sehingga tarif, riwayat status, dan tanda tangan
// resolusi tarif selalu konsisten.

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type extraSpec struct {
	ID, SEP, Card, MRN, Name string
	InsurerID, BatchID       string
	Status                   string
	Admission, Discharge     string
	Dx                       string
	Secondary, Procedures    []string
	Code, Desc               string
	Base, Gap                int64
	RejectCode               string
}

var extraSpecs = []extraSpec{
	// BT-0004 — AIA FINANCIAL, siap diajukan (demo: Ajukan Semua).
	{ID: "CLM-0035", SEP: "0301R001001035", Card: "000210003501", MRN: "RM-23501", Name: "Sinta Maharani", InsurerID: "i-01", BatchID: "BT-0004",
		Status: "ready_to_submit", Admission: "2026-09-24", Discharge: "2026-09-28", Dx: "I25.1", Secondary: []string{"E11.9"}, Procedures: []string{"88.55"},
		Code: "I-1-11-I", Desc: "GROUP JANTUNG-ISKEMIK", Base: 6285000, Gap: 4200000},
	{ID: "CLM-0036", SEP: "0301R001001036", Card: "000210003602", MRN: "RM-23502", Name: "Bagas Prakoso", InsurerID: "i-01", BatchID: "BT-0004",
		Status: "ready_to_submit", Admission: "2026-09-26", Discharge: "2026-09-30", Dx: "J18.9", Secondary: []string{"I10"},
		Code: "J-1-12-I", Desc: "GROUP INFEKSI PARU", Base: 5100000, Gap: 3100000},
	{ID: "CLM-0037", SEP: "0301R001001037", Card: "000210003703", MRN: "RM-23503", Name: "Citra Kirana", InsurerID: "i-01", BatchID: "BT-0004",
		Status: "draft", Admission: "2026-09-27", Discharge: "2026-10-01", Dx: "N39.0",
		Code: "N-1-10-I", Desc: "GROUP INFEKSI SALURAN KEMIH", Base: 4800000, Gap: 2650000},
	// BT-0005 — ALLIANZ, sudah diajukan, menunggu keputusan penjamin.
	{ID: "CLM-0038", SEP: "0301R001001038", Card: "000210003804", MRN: "RM-23504", Name: "Dimas Anggara", InsurerID: "i-02", BatchID: "BT-0005",
		Status: "submitted", Admission: "2026-09-08", Discharge: "2026-09-12", Dx: "K35.8", Procedures: []string{"47.09"},
		Code: "K-1-13-I", Desc: "GROUP APENDISITIS", Base: 7200000, Gap: 5400000},
	{ID: "CLM-0039", SEP: "0301R001001039", Card: "000210003905", MRN: "RM-23505", Name: "Eka Putri", InsurerID: "i-02", BatchID: "BT-0005",
		Status: "submitted", Admission: "2026-09-14", Discharge: "2026-09-18", Dx: "I10",
		Code: "I-4-10-I", Desc: "GROUP HIPERTENSI", Base: 4300000, Gap: 2300000},
	// BT-0006 — MANULIFE, disetujui / dibayar sebagian (tampil di dropdown Pembayaran).
	{ID: "CLM-0040", SEP: "0301R001001040", Card: "000210004006", MRN: "RM-23506", Name: "Fajar Nugroho", InsurerID: "i-09", BatchID: "BT-0006",
		Status: "approved", Admission: "2026-08-05", Discharge: "2026-08-09", Dx: "E11.65", Secondary: []string{"I10"},
		Code: "E-4-10-I", Desc: "GROUP DIABETES", Base: 5500000, Gap: 6100000},
	{ID: "CLM-0041", SEP: "0301R001001041", Card: "000210004107", MRN: "RM-23507", Name: "Gita Permata", InsurerID: "i-09", BatchID: "BT-0006",
		Status: "partially_approved", Admission: "2026-08-12", Discharge: "2026-08-17", Dx: "K80.2", Procedures: []string{"51.23"},
		Code: "K-1-15-II", Desc: "GROUP BATU EMPEDU", Base: 6600000, Gap: 4800000},
	{ID: "CLM-0042", SEP: "0301R001001042", Card: "000210004208", MRN: "RM-23508", Name: "Hendro Wibowo", InsurerID: "i-09", BatchID: "BT-0006",
		Status: "partially_paid", Admission: "2026-08-19", Discharge: "2026-08-23", Dx: "I63.9",
		Code: "G-1-12-II", Desc: "GROUP STROKE", Base: 5900000, Gap: 7000000},
	// BT-0007 — BNI LIFE, sudah lunas.
	{ID: "CLM-0043", SEP: "0301R001001043", Card: "000210004309", MRN: "RM-23509", Name: "Indah Lestari", InsurerID: "i-03", BatchID: "BT-0007",
		Status: "paid", Admission: "2026-07-03", Discharge: "2026-07-07", Dx: "J45.9",
		Code: "J-1-14-I", Desc: "GROUP ASMA", Base: 6100000, Gap: 3500000},
	{ID: "CLM-0044", SEP: "0301R001001044", Card: "000210004410", MRN: "RM-23510", Name: "Joko Prasetyo", InsurerID: "i-03", BatchID: "BT-0007",
		Status: "paid", Admission: "2026-07-10", Discharge: "2026-07-14", Dx: "A09",
		Code: "A-1-10-I", Desc: "GROUP GASTROENTERITIS", Base: 5200000, Gap: 2900000},
	// Tanpa batch — penolakan, sanggahan, hapus buku.
	{ID: "CLM-0045", SEP: "0301R001001045", Card: "000210004511", MRN: "RM-23511", Name: "Kartika Dewi", InsurerID: "i-06",
		Status: "rejected", Admission: "2026-09-02", Discharge: "2026-09-06", Dx: "M54.5",
		Code: "M-1-10-I", Desc: "GROUP NYERI PUNGGUNG", Base: 4400000, Gap: 2800000, RejectCode: "EXCL_PREEXIST"},
	{ID: "CLM-0046", SEP: "0301R001001046", Card: "000210004612", MRN: "RM-23512", Name: "Lukman Hakim", InsurerID: "i-11",
		Status: "disputed", Admission: "2026-08-25", Discharge: "2026-08-30", Dx: "N18.9",
		Code: "N-1-12-II", Desc: "GROUP GAGAL GINJAL", Base: 8100000, Gap: 6300000, RejectCode: "DOC_INCOMPLETE"},
	{ID: "CLM-0047", SEP: "0301R001001047", Card: "000210004713", MRN: "RM-23513", Name: "Maya Sari", InsurerID: "i-12",
		Status: "written_off", Admission: "2026-05-04", Discharge: "2026-05-08", Dx: "S72.0", Procedures: []string{"79.35"},
		Code: "S-1-14-III", Desc: "GROUP FRAKTUR", Base: 9000000, Gap: 3000000, RejectCode: "LATE_SUBMISSION"},
}

// concat menggabungkan data dasar dengan data tambahan tanpa mengubah slice aslinya.
func concat[T any](a, b []T) []T {
	out := make([]T, 0, len(a)+len(b))
	return append(append(out, a...), b...)
}

func shiftStr(date string, days int) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date // dideteksi oleh tes seed_test.go
	}
	return t.AddDate(0, 0, days).Format("2006-01-02")
}

func losBetween(a, b string) int {
	ta, e1 := time.Parse("2006-01-02", a)
	tb, e2 := time.Parse("2006-01-02", b)
	if e1 != nil || e2 != nil {
		return 0
	}
	return int(tb.Sub(ta).Hours() / 24)
}

// extraChains = urutan status setelah draft untuk tiap status akhir.
var extraChains = map[string][]string{
	"draft":              nil,
	"ready_to_submit":    {"ready_to_submit"},
	"submitted":          {"ready_to_submit", "submitted"},
	"approved":           {"ready_to_submit", "submitted", "approved"},
	"partially_approved": {"ready_to_submit", "submitted", "partially_approved"},
	"partially_paid":     {"ready_to_submit", "submitted", "approved", "partially_paid"},
	"paid":               {"ready_to_submit", "submitted", "approved", "paid"},
	"rejected":           {"ready_to_submit", "submitted", "rejected"},
	"disputed":           {"ready_to_submit", "submitted", "rejected", "disputed"},
	"written_off":        {"ready_to_submit", "submitted", "rejected", "written_off"},
}

func extraActor(to string) string {
	switch to {
	case "ready_to_submit":
		return "Rina Wijaya"
	case "submitted":
		return "Sari Kusuma" // checker ≠ pembuat klaim (maker-checker)
	case "partially_paid", "paid":
		return "Sistem"
	}
	return "Budi Santoso"
}

func buildExtraClaim(sp extraSpec) seedClaim {
	total := sp.Base + sp.Gap
	nonBedah, kamar, lab, obat := total*20/100, total*40/100, total*15/100, total*15/100
	tariff := seedTariff{
		ProsedurNonBedah: nonBedah, KamarAkomodasi: kamar, Laboratorium: lab, Obat: obat,
		Konsultasi: total - nonBedah - kamar - lab - obat,
	}
	los := losBetween(sp.Admission, sp.Discharge)
	sec, procs := sp.Secondary, sp.Procedures
	sigParts := []string{sp.Dx, strings.Join(sec, ","), strings.Join(procs, ","), strconv.Itoa(los), "RAWAT_INAP", "I"}

	created := sp.Discharge + "T09:00:00"
	hist := []seedStatusHistory{{At: created, Actor: "Rina Wijaya", To: "draft", Note: "Klaim dibuat"}}
	prev, last := "draft", created
	var submittedAt *string
	for i, to := range extraChains[sp.Status] {
		day := shiftStr(sp.Discharge, i+1)
		if to == "approved" || to == "partially_approved" || to == "rejected" {
			day = shiftStr(sp.Discharge, 10)
		}
		if to == "partially_paid" || to == "paid" {
			day = shiftStr(sp.Discharge, 20)
		}
		if to == "disputed" || to == "written_off" {
			day = shiftStr(sp.Discharge, 30)
		}
		at := day + "T10:00:00"
		note := ""
		if to == "rejected" {
			note = "Ditolak: " + sp.RejectCode
		}
		hist = append(hist, seedStatusHistory{At: at, Actor: extraActor(to), From: prev, To: to, Note: note})
		if to == "submitted" {
			s := at
			submittedAt = &s
		}
		prev, last = to, at
	}
	c := seedClaim{
		ID: sp.ID, HospitalID: "h-ns", SEPNo: sp.SEP, BPJSCardNo: sp.Card, MRN: sp.MRN, PatientName: sp.Name,
		DPJP: "dr. Sri Wahyuni, Sp.PD", ParticipantType: "NON_PBI", CareType: "RAWAT_INAP", CareClass: "I", Sex: "L",
		DateOfBirth: "1980-01-15", AdmissionDate: sp.Admission, DischargeDate: sp.Discharge, LOS: los,
		PrimaryDiagnosis: sp.Dx, SecondaryDiagnoses: sec, Procedures: procs,
		INACBGCode: sp.Code, INACBGDescription: sp.Desc, INACBGBaseTariff: sp.Base, HospitalTariff: tariff,
		TariffResolvedAt: sp.Discharge + "T12:00:00", DeclarationChecked: true,
		InsurerID: sp.InsurerID, PolicyNo: "POL-" + sp.SEP[len(sp.SEP)-6:], InsuredName: sp.Name,
		InsuranceType: "Individu", InsuranceMemberNo: "MBR-" + sp.SEP[len(sp.SEP)-6:], CoverageType: "Naik Kelas",
		PolicyCeiling: ptr[int64](60000000), Status: sp.Status, CreatedAt: created, UpdatedAt: last,
		SubmittedAt: submittedAt, CreatedBy: "Rina Wijaya", RejectionReasonCode: sp.RejectCode,
		ResolutionSignature: strings.Join(sigParts, "|"), StatusHistory: hist,
	}
	if sp.RejectCode != "" {
		c.RejectionReasonNote = "Alasan penolakan demo: " + sp.RejectCode
	}
	if sp.BatchID != "" {
		c.BatchID = ptr(sp.BatchID)
	}
	if sp.Status == "draft" {
		c.SubmittedAt = nil
	}
	return c
}

func extraClaims() []seedClaim {
	out := make([]seedClaim, 0, len(extraSpecs))
	for _, sp := range extraSpecs {
		out = append(out, buildExtraClaim(sp))
	}
	return out
}

// Resume medis bersih (2 item, sudah direview) untuk setiap klaim tambahan → tidak ada blocker R10.
func extraResume() []seedResumeItem {
	out := make([]seedResumeItem, 0, len(extraSpecs)*2)
	for _, sp := range extraSpecs {
		out = append(out,
			seedResumeItem{ClaimID: sp.ID, SEP: sp.SEP, Kategori: "REKAM_MEDIS", NamaItem: "Resume Medis Rawat Inap", SubKategori: "-",
				Tanggal: sp.Discharge, HasilCatatan: "Ringkasan pulang", Jumlah: 1, Satuan: "kali", HargaSatuan: 250000, Subtotal: 250000, Reviewed: true},
			seedResumeItem{ClaimID: sp.ID, SEP: sp.SEP, Kategori: "HASIL_LAB", NamaItem: "Darah Lengkap", SubKategori: "Hematologi",
				Tanggal: sp.Admission, HasilCatatan: "Dalam batas normal", Jumlah: 2, Satuan: "kali", HargaSatuan: 150000, Subtotal: 300000, Reviewed: true},
		)
	}
	return out
}

// Batch per-penjamin: nama & periode (YYYY-MM) dibentuk saat seeding dari tanggal yang sudah digeser.
var extraBatches = []seedBatch{
	{ID: "BT-0004", HospitalID: "h-ns", InsurerID: "i-01", PeriodStart: "2026-09-20", PeriodEnd: "2026-10-03", CreatedAt: "2026-10-02T09:00:00", CreatedBy: "Rina Wijaya", Note: "Batch AIA periode berjalan"},
	{ID: "BT-0005", HospitalID: "h-ns", InsurerID: "i-02", PeriodStart: "2026-09-01", PeriodEnd: "2026-09-30", CreatedAt: "2026-09-30T15:00:00", CreatedBy: "Rina Wijaya", Note: "Sudah diajukan, menunggu keputusan"},
	{ID: "BT-0006", HospitalID: "h-ns", InsurerID: "i-09", PeriodStart: "2026-08-01", PeriodEnd: "2026-08-31", CreatedAt: "2026-09-01T09:00:00", CreatedBy: "Rina Wijaya", Note: "Disetujui, pembayaran sebagian"},
	{ID: "BT-0007", HospitalID: "h-ns", InsurerID: "i-03", PeriodStart: "2026-07-01", PeriodEnd: "2026-07-31", CreatedAt: "2026-08-01T09:00:00", CreatedBy: "Rina Wijaya", Note: "Lunas"},
}

var extraPayments = []seedPayment{
	{ID: "PAY-0007", ClaimID: "CLM-0042", ReceivedDate: "2026-09-12", Amount: 3000000, Payer: "MANULIFE INDONESIA", TransferRef: "TRF-440003",
		Notes: "Cicilan 1 — sisa menyusul [Batch BT-0006]", CreatedBy: "Joko Handoko", CreatedAt: "2026-09-12T10:00:00", Allocation: "GAP"},
	{ID: "PAY-0008", ClaimID: "CLM-0043", ReceivedDate: "2026-08-12", Amount: 3500000, Payer: "BNI LIFE INSURANCE", TransferRef: "TRF-440001",
		Notes: "Pembayaran penuh [Batch BT-0007]", CreatedBy: "Joko Handoko", CreatedAt: "2026-08-12T09:30:00", Allocation: "GAP"},
	{ID: "PAY-0009", ClaimID: "CLM-0044", ReceivedDate: "2026-08-12", Amount: 2900000, Payer: "BNI LIFE INSURANCE", TransferRef: "TRF-440002",
		Notes: "Pembayaran penuh [Batch BT-0007]", CreatedBy: "Joko Handoko", CreatedAt: "2026-08-12T09:31:00", Allocation: "GAP"},
}

var extraBankImports = []seedBankImport{
	{ID: "BI-0003", BankName: "Bank BCA — Rek Giro RS", PeriodLabel: "BT-0007 — BNI LIFE INSURANCE", ImportedAt: "2026-08-13T09:00:00", LineCount: 3},
}

var extraBankLines = []seedBankLine{
	{ID: "BL-0010", ImportID: "BI-0003", ValueDate: "2026-08-12", Description: "TRANSFER MASUK BNI LIFE INSURANCE KLAIM 001043 Indah Lestari POL-001043",
		Amount: 3500000, Reference: "TRF-440001", MatchResult: "matched", MatchClaimID: ptr("CLM-0043"), MatchScore: 100,
		MatchNote: ptr("auto-match: SEP pada deskripsi, nama tertanggung, no. polis, nominal ± toleransi"), Confirmed: true},
	{ID: "BL-0011", ImportID: "BI-0003", ValueDate: "2026-08-12", Description: "TRANSFER MASUK MANULIFE KLAIM 001040 Fajar Nugroho",
		Amount: 4000000, Reference: "BCA/FT/2608/021", MatchResult: "needs_review", MatchClaimID: ptr("CLM-0040"), MatchScore: 65,
		MatchNote: ptr("skor 65 (SEP pada deskripsi, nama tertanggung) — nominal beda")},
	{ID: "BL-0012", ImportID: "BI-0003", ValueDate: "2026-08-12", Description: "BIAYA ADMINISTRASI BULANAN",
		Amount: 25000, Reference: "BCA/ADM/2608", MatchResult: "unmatched", MatchScore: 0},
}

var extraWorkItems = []seedWorkItem{
	{ID: "WI-0007", Title: "Tagih sisa pembayaran batch BT-0006 ke MANULIFE (klaim Hendro Wibowo)", Source: "AGING", ClaimID: ptr("CLM-0042"),
		Assignee: ptr("u-fi"), Role: ptr("FINANCE"), DueDate: "2026-10-07", Priority: "Tinggi", Status: "In Progress",
		Comments: []seedWorkComment{{Author: "Joko Handoko", Text: "Sudah menghubungi MANULIFE, pelunasan dijanjikan minggu depan.", At: "2026-10-02T11:00:00"}},
		CreatedAt: "2026-10-01T09:00:00"},
	{ID: "WI-0008", Title: "Susun sanggahan klaim Kartika Dewi (EXCL_PREEXIST)", Source: "REJECTION", ClaimID: ptr("CLM-0045"),
		Assignee: ptr("u-bl"), Role: ptr("BILLING_STAFF"), DueDate: "2026-10-09", Priority: "Tinggi", Status: "Assigned", CreatedAt: "2026-09-20T08:00:00"},
	{ID: "WI-0009", Title: "Ajukan batch BT-0004 (AIA FINANCIAL) — 2 klaim siap diajukan", Source: "DEADLINE",
		Assignee: ptr("u-bl"), Role: ptr("BILLING_STAFF"), DueDate: "2026-10-05", Priority: "Sedang", Status: "New", CreatedAt: "2026-10-02T09:30:00"},
	{ID: "WI-0010", Title: "Konfirmasi mutasi BCA Rp 4.000.000 untuk klaim Fajar Nugroho", Source: "RECONCILIATION", ClaimID: ptr("CLM-0040"),
		Assignee: ptr("u-fi"), Role: ptr("FINANCE"), DueDate: "2026-10-06", Priority: "Sedang", Status: "Waiting Verification", CreatedAt: "2026-09-29T10:00:00"},
	{ID: "WI-0011", Title: "Bukti pelunasan batch BT-0007 sudah diarsipkan", Source: "AGING", ClaimID: ptr("CLM-0043"),
		Assignee: ptr("u-fi"), Role: ptr("FINANCE"), DueDate: "2026-09-20", Priority: "Rendah", Status: "Closed",
		History: []seedWorkHistory{{At: "2026-09-18T15:00:00", Actor: "Joko Handoko", Action: "Status → Closed"}}, CreatedAt: "2026-09-10T08:00:00"},
	{ID: "WI-0012", Title: "Dokumen sanggahan klaim Lukman Hakim perlu dilengkapi ulang", Source: "REJECTION", ClaimID: ptr("CLM-0046"),
		Assignee: ptr("u-bl"), Role: ptr("BILLING_STAFF"), DueDate: "2026-10-03", Priority: "Kritis", Status: "Reopened",
		History: []seedWorkHistory{{At: "2026-10-01T09:00:00", Actor: "Budi Santoso", Action: "Status → Reopened"}}, CreatedAt: "2026-09-15T08:00:00"},
}

var extraNotifications = []seedNotification{
	{At: "2026-09-12T10:05:00", Channel: "IN_APP", Template: "PAYMENT_RECEIVED", ClaimID: ptr("CLM-0042"), Recipient: "finance@nusantara.id",
		Message: "Pembayaran batch BT-0006 diterima: Rp 3000000 untuk 1 klaim", Status: "TERKIRIM"},
	{At: "2026-08-12T09:35:00", Channel: "IN_APP", Template: "PAYMENT_RECEIVED", ClaimID: ptr("CLM-0043"), Recipient: "finance@nusantara.id",
		Message: "Pembayaran batch BT-0007 diterima: Rp 6400000 untuk 2 klaim", Status: "TERKIRIM"},
	{At: "2026-09-10T10:00:00", Channel: "IN_APP", Template: "CLAIM_REJECTED", ClaimID: ptr("CLM-0045"), Recipient: "billing@nusantara.id",
		Message: "Klaim Kartika Dewi ditolak: EXCL_PREEXIST — susun sanggahan.", Status: "TERKIRIM"},
}

// extraSummary dipakai tes untuk memastikan jumlah data tambahan tidak berubah diam-diam.
func extraSummary() string {
	return fmt.Sprintf("claims=%d batches=%d payments=%d", len(extraSpecs), len(extraBatches), len(extraPayments))
}
