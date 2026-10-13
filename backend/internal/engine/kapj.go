package engine

import (
	"math"

	"medpay_spkd/internal/model"
)

// KAPJParams = parameter simulasi Koordinasi Antar Penyelenggara Jaminan.
type KAPJParams struct {
	BPJSSharePct       float64 `json:"bpjsSharePct"`
	CoPayEnabled       bool    `json:"coPayEnabled"`
	PatientCoPayPct    float64 `json:"patientCoPayPct"`
	OutpatientCoPayCap int64   `json:"outpatientCoPayCap"`
	InpatientCoPayCap  int64   `json:"inpatientCoPayCap"`
}

// DefaultKAPJ dipakai bila RS belum punya pengaturan KAPJ.
var DefaultKAPJ = KAPJParams{
	BPJSSharePct:       75,
	CoPayEnabled:       true,
	PatientCoPayPct:    5,
	OutpatientCoPayCap: 300_000,
	InpatientCoPayCap:  3_000_000,
}

// KAPJFromSetting mengubah pengaturan KAPJ RS menjadi parameter simulasi.
func KAPJFromSetting(s model.KAPJSetting) KAPJParams {
	return KAPJParams{
		BPJSSharePct:       s.BPJSSharePct,
		CoPayEnabled:       s.CopayEnabled,
		PatientCoPayPct:    s.PatientCopayPct,
		OutpatientCoPayCap: s.OutpatientCopayCap,
		InpatientCoPayCap:  s.InpatientCopayCap,
	}
}

// Status hasil pembagian KAPJ.
const (
	KAPJLunasPenuh     = "Lunas Penuh"
	KAPJDibatasiPlafon = "Dibatasi Plafon"
	KAPJCoPayMaks      = "Co-pay Maks"
	KAPJSebagian       = "Sebagian (Plafon)"
)

// KAPJSplit = pembagian tagihan antara BPJS, penjamin, dan pasien.
type KAPJSplit struct {
	TotalHospitalBill int64  `json:"totalHospitalBill"`
	BPJSPays          int64  `json:"bpjsPays"`
	InsurerPays       int64  `json:"insurerPays"`
	PatientPays       int64  `json:"patientPays"`
	Status            string `json:"status"`
}

func pct(amount int64, p float64) int64 {
	return int64(math.Trunc(float64(amount) * p / 100))
}

// SplitKAPJ menghitung porsi BPJS (persentase tarif JKN), penjamin (sisa tagihan,
// dibatasi plafon polis bila ada), dan co-pay pasien (persentase porsi penjamin,
// dibatasi cap rawat jalan / rawat inap). Plafon nil atau 0 = tidak terbatas.
func SplitKAPJ(totalBill, jknTotal int64, policyCeiling *int64, careType string, p KAPJParams) KAPJSplit {
	bpjs := pct(jknTotal, p.BPJSSharePct)
	if bpjs > jknTotal {
		bpjs = jknTotal
	}
	remaining := totalBill - bpjs
	if remaining < 0 {
		remaining = 0
	}

	insurer := remaining
	status := KAPJLunasPenuh
	if policyCeiling != nil && *policyCeiling > 0 && insurer > *policyCeiling {
		insurer = *policyCeiling
		status = KAPJDibatasiPlafon
	}

	var patient int64
	if p.CoPayEnabled && p.PatientCoPayPct > 0 {
		patient = pct(insurer, p.PatientCoPayPct)
		limit := p.OutpatientCoPayCap
		if careType == model.CareTypeRawatInap {
			limit = p.InpatientCoPayCap
		}
		if patient > limit {
			patient = limit
			if status == KAPJLunasPenuh {
				status = KAPJCoPayMaks
			}
		}
	}
	if status == KAPJLunasPenuh && p.CoPayEnabled && patient > 0 {
		status = KAPJSebagian
	}
	return KAPJSplit{
		TotalHospitalBill: totalBill,
		BPJSPays:          bpjs,
		InsurerPays:       insurer - patient,
		PatientPays:       patient,
		Status:            status,
	}
}
