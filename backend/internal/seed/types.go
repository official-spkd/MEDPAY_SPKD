package seed

// Tipe data demo. ID berupa kode pendek (h-ns, u-bl, CLM-0001) yang dipakai untuk
// menghubungkan antar data; ID database dibuat saat seeding (lihat steps.go).
// Nama pengguna dipakai sebagai referensi aktor (CreatedBy, Actor, Author).

type demoData struct {
	Hospitals     []seedHospital
	Users         []seedUser
	Insurers      []seedInsurer
	Claims        []seedClaim
	Payments      []seedPayment
	BankImports   []seedBankImport
	BankLines     []seedBankLine
	Batches       []seedBatch
	WorkItems     []seedWorkItem
	ResumeItems   []seedResumeItem
	AuditLog      []seedAuditLog
	Partners      []seedPartner
	APIIntakes    []seedAPIIntake
	Regulations   []seedRegulation
	Notifications []seedNotification
}

type seedHospital struct {
	ID, KodeFaskes, Name, City, Class string
	Active, EklaimConfigured          bool
	RegisteredAt                      string
}

type seedUser struct {
	ID, Name, Email, Role, HospitalID, Initials string
	MFAEnabled                                  bool
}

type seedInsurer struct {
	ID, Name, Type string
	AvgPaymentDays int
	DeadlineDays   int
	CeilingPolicy  string
	Verified       bool
}

// Urutan & tipe field harus identik dengan model.HospitalTariff agar bisa dikonversi langsung.
type seedTariff struct {
	ProsedurNonBedah int64
	ProsedurBedah    int64
	Konsultasi       int64
	TenagaAhli       int64
	Keperawatan      int64
	Penunjang        int64
	Radiologi        int64
	Laboratorium     int64
	PelayananDarah   int64
	Rehabilitasi     int64
	KamarAkomodasi   int64
	RawatIntensif    int64
	Obat             int64
	ObatKronis       int64
	ObatKemo         int64
	Alkes            int64
	BMHP             int64
	SewaAlat         int64
}

// Urutan & tipe field harus identik dengan model.SpecialCMG.
type seedCMG struct {
	SpecialProcedure     int64
	SpecialProsthesis    int64
	SpecialInvestigation int64
	SpecialDrug          int64
	SubAcute             int64
	Chronic              int64
}

type seedStatusHistory struct {
	At, Actor, From, To, Note string
}

type seedClaim struct {
	ID, HospitalID, SEPNo, BPJSCardNo, MRN, PatientName, DPJP string
	ParticipantType, CareType, CareClass, Sex                 string
	DateOfBirth, AdmissionDate, DischargeDate                 string
	LOS                                                       int
	PrimaryDiagnosis                                          string
	SecondaryDiagnoses, Procedures                            []string
	INACBGCode, INACBGDescription                             string
	INACBGBaseTariff                                          int64
	SpecialCMG                                                seedCMG
	HospitalTariff                                            seedTariff
	TariffResolvedAt                                          string
	DeclarationChecked                                        bool
	InsurerID, PolicyNo, InsuredName, InsurerNameExtra        string
	InsuranceType, InsuranceMemberNo, CoverageType            string
	PolicyCeiling                                             *int64
	Status, CreatedAt, UpdatedAt                              string
	SubmittedAt, BatchID                                      *string
	CreatedBy                                                 string
	RejectionReasonCode, RejectionReasonNote                  string
	ResolutionSignature                                       string
	StatusHistory                                             []seedStatusHistory
}

type seedPayment struct {
	ID, ClaimID, ReceivedDate        string
	Amount                           int64
	Payer, TransferRef, Notes        string
	CreatedBy, CreatedAt, Allocation string
}

type seedBankImport struct {
	ID, BankName, PeriodLabel, ImportedAt string
	LineCount                             int
}

type seedBankLine struct {
	ID, ImportID, ValueDate, Description string
	Amount                               int64
	Reference, MatchResult               string
	MatchClaimID                         *string
	MatchScore                           int
	MatchNote                            *string
	Confirmed                            bool
}

type seedBatch struct {
	ID, Name, Period, HospitalID, CreatedAt, CreatedBy, Note string
}

type seedWorkComment struct {
	Author, Text, At string
}

type seedWorkHistory struct {
	At, Actor, Action string
}

type seedWorkItem struct {
	ID, Title, Source         string
	ClaimID, Assignee, Role   *string
	DueDate, Priority, Status string
	Comments                  []seedWorkComment
	History                   []seedWorkHistory
	CreatedAt                 string
}

type seedResumeItem struct {
	ClaimID, SEP, Kategori, NamaItem, SubKategori, Tanggal, HasilCatatan string
	Jumlah                                                               float64
	Satuan                                                               string
	HargaSatuan, Subtotal                                                int64
	Reviewed                                                             bool
}

type seedAuditLog struct {
	At, Actor, ActorRole, Action, Entity, EntityID string
	Before, After, IP, CorrelationID, Scope        string
	StatusCode                                     int
	PartnerName                                    string
}

type seedPartner struct {
	ID, Company, Contact                string
	RateLimitPerMin                     int
	WhitelistIPs                        []string
	ConsumerID                          string
	ConsumerSecretMasked, UserKeyMasked string
	Active                              bool
	HospitalIDs                         []string
	CreatedAt                           string
}

type seedAPIIntake struct {
	ID, PartnerID, HospitalID, SEPNo, PatientName, INACBGCode string
	HospitalTotal, JKNTotal, Gap                              int64
	Status, ErrorCode                                         string
	LatencyMs                                                 int
	At                                                        string
	BPJSShare, InsurerShare, PatientShare                     int64
}

type seedRegulation struct {
	Number, Title, Article, Requirement, EffectiveDate string
	Module, Feature, Status, LastReviewed, Reviewer    string
	Note                                               string
}

type seedNotification struct {
	At, Channel, Template string
	ClaimID               *string
	Recipient, Message    string
	Status                string
}

type seedRejectionCode struct {
	Code, Label string
}

type seedScrubRule struct {
	ID, Name, Category, Severity, Message, FixSuggestion, SourceRef string
	Enabled                                                         bool
}

// demoDataset mengumpulkan seluruh data demo dari file seed_*.go.
func demoDataset() *demoData {
	return &demoData{
		Hospitals: seedHospitals, Users: seedUsers, Insurers: seedInsurers,
		Regulations: seedRegulations,
		Batches:     seedBatches, Claims: seedClaims, ResumeItems: seedResumeItems,
		Payments: seedPayments, BankImports: seedBankImports, BankLines: seedBankLines,
		WorkItems: seedWorkItems, Notifications: seedNotifications, AuditLog: seedAuditLogs,
		Partners: seedPartners, APIIntakes: seedAPIIntakes,
	}
}
