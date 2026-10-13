package engine

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"medpay_spkd/internal/model"
)

// SumTariff = total tagihan RS (18 komponen).
func SumTariff(t model.HospitalTariff) int64 {
	return t.ProsedurNonBedah + t.ProsedurBedah + t.Konsultasi + t.TenagaAhli + t.Keperawatan +
		t.Penunjang + t.Radiologi + t.Laboratorium + t.PelayananDarah + t.Rehabilitasi +
		t.KamarAkomodasi + t.RawatIntensif + t.Obat + t.ObatKronis + t.ObatKemo + t.Alkes +
		t.BMHP + t.SewaAlat
}

// SumCMG = total top-up special CMG (6 komponen).
func SumCMG(c model.SpecialCMG) int64 {
	return c.SpecialProcedure + c.SpecialProsthesis + c.SpecialInvestigation +
		c.SpecialDrug + c.SubAcute + c.Chronic
}

// TotalJKN = tarif dasar INA-CBG + top-up special CMG.
func TotalJKN(base int64, cmg model.SpecialCMG) int64 { return base + SumCMG(cmg) }

// Gap = tagihan RS − tarif JKN: selisih yang ditagihkan ke penjamin.
func Gap(c *model.Claim) int64 {
	return SumTariff(c.HospitalTariff) - TotalJKN(c.INACBGBaseTariff, c.SpecialCMG)
}

// SecondaryCodes mengembalikan kode diagnosis sekunder sesuai urutan input (Seq).
func SecondaryCodes(c *model.Claim) []string {
	items := append([]model.ClaimSecondaryDiagnosis(nil), c.SecondaryDiagnoses...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].Seq < items[j].Seq })
	out := make([]string, len(items))
	for i, d := range items {
		out[i] = d.ICD10Code
	}
	return out
}

// ProcedureCodes mengembalikan kode prosedur sesuai urutan input (Seq).
func ProcedureCodes(c *model.Claim) []string {
	items := append([]model.ClaimProcedure(nil), c.Procedures...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].Seq < items[j].Seq })
	out := make([]string, len(items))
	for i, p := range items {
		out[i] = p.ICD9Code
	}
	return out
}

// TariffSignature adalah jejak input grouper. Bila berubah setelah tarif di-resolve,
// tarif dianggap basi (aturan R06).
func TariffSignature(c *model.Claim) string {
	return strings.Join([]string{
		c.PrimaryDiagnosis,
		strings.Join(SecondaryCodes(c), ","),
		strings.Join(ProcedureCodes(c), ","),
		strconv.Itoa(c.LOS),
		c.CareType,
		c.CareClass,
	}, "|")
}

// Zona deadline pengajuan.
const (
	ZoneAman    = "aman"
	ZoneWaspada = "waspada" // ≤ 30 hari
	ZoneKritis  = "kritis"  // ≤ 7 hari
	ZoneLewat   = "lewat"   // sudah lewat
)

// DeadlineInfo = batas pengajuan klaim (6 bulan sejak tanggal masuk).
type DeadlineInfo struct {
	Deadline *time.Time // nil bila tanggal masuk kosong
	DaysLeft int
	Zone     string
}

// Near bernilai true untuk zona waspada, kritis, atau lewat.
func (d DeadlineInfo) Near() bool {
	return d.Zone == ZoneWaspada || d.Zone == ZoneKritis || d.Zone == ZoneLewat
}

// Deadline menghitung batas pengajuan 6 bulan sejak tanggal masuk.
func Deadline(admission *time.Time, today time.Time) DeadlineInfo {
	if admission == nil || admission.IsZero() {
		return DeadlineInfo{Zone: ZoneAman}
	}
	dl := Day(*admission).AddDate(0, 6, 0)
	left := DaysBetween(today, dl)
	zone := ZoneAman
	switch {
	case left < 0:
		zone = ZoneLewat
	case left <= 7:
		zone = ZoneKritis
	case left <= 30:
		zone = ZoneWaspada
	}
	return DeadlineInfo{Deadline: &dl, DaysLeft: left, Zone: zone}
}
