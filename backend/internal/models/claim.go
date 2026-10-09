package models

import "time"

type ClaimStatus string

const (
	ClaimDraft             ClaimStatus = "draft"
	ClaimReadyToSubmit     ClaimStatus = "ready_to_submit"
	ClaimSubmitted         ClaimStatus = "submitted"
	ClaimApproved          ClaimStatus = "approved"
	ClaimPartiallyApproved ClaimStatus = "partially_approved"
	ClaimRejected          ClaimStatus = "rejected"
	ClaimPartiallyPaid     ClaimStatus = "partially_paid"
	ClaimPaid              ClaimStatus = "paid"
	ClaimDisputed          ClaimStatus = "disputed"
	ClaimWrittenOff        ClaimStatus = "written_off"
)

const (
	CareTypeRawatInap  = "RAWAT_INAP"
	CareTypeRawatJalan = "RAWAT_JALAN"

	ParticipantPBI    = "PBI"
	ParticipantNonPBI = "NON_PBI"
)

// HospitalTariff adalah 18 komponen tagihan RS (Rupiah). Disimpan sebagai kolom tariff_*.
type HospitalTariff struct {
	ProsedurNonBedah int64 `gorm:"not null;default:0" json:"prosedur_non_bedah"`
	ProsedurBedah    int64 `gorm:"not null;default:0" json:"prosedur_bedah"`
	Konsultasi       int64 `gorm:"not null;default:0" json:"konsultasi"`
	TenagaAhli       int64 `gorm:"not null;default:0" json:"tenaga_ahli"`
	Keperawatan      int64 `gorm:"not null;default:0" json:"keperawatan"`
	Penunjang        int64 `gorm:"not null;default:0" json:"penunjang"`
	Radiologi        int64 `gorm:"not null;default:0" json:"radiologi"`
	Laboratorium     int64 `gorm:"not null;default:0" json:"laboratorium"`
	PelayananDarah   int64 `gorm:"not null;default:0" json:"pelayanan_darah"`
	Rehabilitasi     int64 `gorm:"not null;default:0" json:"rehabilitasi"`
	KamarAkomodasi   int64 `gorm:"not null;default:0" json:"kamar_akomodasi"`
	RawatIntensif    int64 `gorm:"not null;default:0" json:"rawat_intensif"`
	Obat             int64 `gorm:"not null;default:0" json:"obat"`
	ObatKronis       int64 `gorm:"not null;default:0" json:"obat_kronis"`
	ObatKemo         int64 `gorm:"not null;default:0" json:"obat_kemo"`
	Alkes            int64 `gorm:"not null;default:0" json:"alkes"`
	BMHP             int64 `gorm:"column:bmhp;not null;default:0" json:"bmhp"`
	SewaAlat         int64 `gorm:"not null;default:0" json:"sewa_alat"`
}

// SpecialCMG adalah 6 komponen top-up INA-CBG (Rupiah). Disimpan sebagai kolom cmg_*.
type SpecialCMG struct {
	SpecialProcedure     int64 `gorm:"not null;default:0" json:"special_procedure"`
	SpecialProsthesis    int64 `gorm:"not null;default:0" json:"special_prosthesis"`
	SpecialInvestigation int64 `gorm:"not null;default:0" json:"special_investigation"`
	SpecialDrug          int64 `gorm:"not null;default:0" json:"special_drug"`
	SubAcute             int64 `gorm:"not null;default:0" json:"sub_acute"`
	Chronic              int64 `gorm:"not null;default:0" json:"chronic"`
}

// Claim adalah klaim selisih biaya (tagihan RS − tarif JKN) yang ditagihkan ke penjamin.
// Hasil scrub & readiness score tidak disimpan; dihitung ulang dari data klaim.
type Claim struct {
	BaseModel
	ClaimNo string `gorm:"size:20;not null;uniqueIndex" json:"claim_no"`

	// Data pasien & SEP
	SEPNo           string     `gorm:"column:sep_no;size:30;not null;index" json:"sep_no"`
	BPJSCardNo      string     `gorm:"column:bpjs_card_no;size:30" json:"bpjs_card_no"`
	MRN             string     `gorm:"column:mrn;size:30;index" json:"mrn"`
	PatientName     string     `gorm:"size:150;not null;index" json:"patient_name"`
	DPJP            string     `gorm:"column:dpjp;size:150" json:"dpjp"`
	ParticipantType string     `gorm:"size:20;not null" json:"participant_type"`
	CareType        string     `gorm:"size:20;not null;index" json:"care_type"`
	CareClass       string     `gorm:"size:5;not null" json:"care_class"`
	Sex             string     `gorm:"size:1" json:"sex"`
	DateOfBirth     *time.Time `gorm:"type:date" json:"date_of_birth"`
	AdmissionDate   *time.Time `gorm:"type:date;index" json:"admission_date"`
	DischargeDate   *time.Time `gorm:"type:date" json:"discharge_date"`
	LOS             int        `gorm:"column:los;not null;default:0" json:"los"`

	// Koding & tarif INA-CBG
	PrimaryDiagnosis    string         `gorm:"size:10;index" json:"primary_diagnosis"`
	INACBGCode          string         `gorm:"column:inacbg_code;size:20;index" json:"inacbg_code"`
	INACBGDescription   string         `gorm:"column:inacbg_description;size:255" json:"inacbg_description"`
	INACBGBaseTariff    int64          `gorm:"column:inacbg_base_tariff;not null;default:0" json:"inacbg_base_tariff"`
	SpecialCMG          SpecialCMG     `gorm:"embedded;embeddedPrefix:cmg_" json:"special_cmg"`
	HospitalTariff      HospitalTariff `gorm:"embedded;embeddedPrefix:tariff_" json:"hospital_tariff"`
	TariffResolvedAt    *time.Time     `json:"tariff_resolved_at"`
	ResolutionSignature string         `gorm:"size:255" json:"resolution_signature"` // jejak input grouper untuk deteksi tarif basi
	DeclarationChecked  bool           `gorm:"not null;default:false" json:"declaration_checked"`

	// Data penjamin / polis
	PolicyNo          string `gorm:"size:50" json:"policy_no"`
	InsuredName       string `gorm:"size:150" json:"insured_name"`
	InsurerNameExtra  string `gorm:"size:150" json:"insurer_name_extra"`
	InsuranceType     string `gorm:"size:50" json:"insurance_type"`
	InsuranceMemberNo string `gorm:"size:50" json:"insurance_member_no"`
	CoverageType      string `gorm:"size:50" json:"coverage_type"`
	PolicyCeiling     *int64 `json:"policy_ceiling"` // nil = plafon tidak terbatas

	// Status & alur
	Status              ClaimStatus `gorm:"size:30;not null;default:draft;index" json:"status"`
	SubmittedAt         *time.Time  `json:"submitted_at"`
	RejectionReasonNote string      `gorm:"type:text" json:"rejection_reason_note"`

	HospitalID      uint           `gorm:"not null;index" json:"hospital_id"`
	Hospital        *Hospital      `gorm:"foreignKey:HospitalID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"hospital,omitempty"`
	InsurerID       *uint          `gorm:"index" json:"insurer_id"`
	Insurer         *Insurer       `gorm:"foreignKey:InsurerID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"insurer,omitempty"`
	BatchID         *uint          `gorm:"index" json:"batch_id"`
	Batch           *Batch         `gorm:"foreignKey:BatchID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"batch,omitempty"`
	RejectionCodeID *uint          `gorm:"index" json:"rejection_code_id"`
	RejectionCode   *RejectionCode `gorm:"foreignKey:RejectionCodeID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"rejection_code,omitempty"`
	CreatedByID     uint           `gorm:"not null;index" json:"created_by_id"`
	CreatedBy       *User          `gorm:"foreignKey:CreatedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"created_by,omitempty"`

	SecondaryDiagnoses []ClaimSecondaryDiagnosis `gorm:"foreignKey:ClaimID" json:"secondary_diagnoses,omitempty"`
	Procedures         []ClaimProcedure          `gorm:"foreignKey:ClaimID" json:"procedures,omitempty"`
	StatusHistories    []ClaimStatusHistory      `gorm:"foreignKey:ClaimID" json:"status_histories,omitempty"`
	ResumeItems        []ResumeItem              `gorm:"foreignKey:ClaimID" json:"resume_items,omitempty"`
	Payments           []Payment                 `gorm:"foreignKey:ClaimID" json:"payments,omitempty"`
	ScrubOverrides     []ClaimScrubOverride      `gorm:"foreignKey:ClaimID" json:"scrub_overrides,omitempty"`
}
