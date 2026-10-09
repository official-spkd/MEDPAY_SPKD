package seeder

// Struct berikut memetakan data/mock_data.json (salinan frontend/src/mock/data.json).
// encoding/json mencocokkan nama field tanpa membedakan huruf besar-kecil,
// sehingga tag hanya diperlukan bila namanya berbeda.

type mockData struct {
	Hospitals      []mockHospital      `json:"hospitals"`
	Users          []mockUser          `json:"users"`
	Insurers       []mockInsurer       `json:"insurers"`
	Claims         []mockClaim         `json:"claims"`
	Payments       []mockPayment       `json:"payments"`
	BankImports    []mockBankImport    `json:"bankImports"`
	BankLines      []mockBankLine      `json:"bankLines"`
	Batches        []mockBatch         `json:"batches"`
	WorkItems      []mockWorkItem      `json:"workItems"`
	ResumeItems    []mockResumeItem    `json:"resumeItems"`
	AuditLog       []mockAuditLog      `json:"auditLog"`
	Partners       []mockPartner       `json:"partners"`
	APIIntakes     []mockAPIIntake     `json:"apiIntakes"`
	Regulations    []mockRegulation    `json:"regulations"`
	Notifications  []mockNotification  `json:"notifications"`
	RejectionCodes []mockRejectionCode `json:"rejectionCodes"`
	ScrubRules     []mockScrubRule     `json:"scrubRules"`
}

type mockHospital struct {
	ID, KodeFaskes, Name, City, Class string
	Active, EklaimConfigured          bool
	RegisteredAt                      string
}

type mockUser struct {
	ID, Name, Email, Role, HospitalID, Initials string
	MFAEnabled                                  bool
}

type mockInsurer struct {
	ID, Name, Type string
	AvgPaymentDays int
	DeadlineDays   int
	CeilingPolicy  string
	Verified       bool
}

// Urutan & tipe field harus identik dengan models.HospitalTariff agar bisa dikonversi langsung.
type mockTariff struct {
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

// Urutan & tipe field harus identik dengan models.SpecialCMG.
type mockCMG struct {
	SpecialProcedure     int64
	SpecialProsthesis    int64
	SpecialInvestigation int64
	SpecialDrug          int64
	SubAcute             int64
	Chronic              int64
}

type mockStatusHistory struct {
	At, Actor, From, To, Note string
}

type mockClaim struct {
	ID, HospitalID, SEPNo, BPJSCardNo, MRN, PatientName, DPJP string
	ParticipantType, CareType, CareClass, Sex                 string
	DateOfBirth, AdmissionDate, DischargeDate                 string
	LOS                                                       int
	PrimaryDiagnosis                                          string
	SecondaryDiagnoses, Procedures                            []string
	INACBGCode, INACBGDescription                             string
	INACBGBaseTariff                                          int64
	SpecialCMG                                                mockCMG
	HospitalTariff                                            mockTariff
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
	StatusHistory                                             []mockStatusHistory
}

type mockPayment struct {
	ID, ClaimID, ReceivedDate        string
	Amount                           int64
	Payer, TransferRef, Notes        string
	CreatedBy, CreatedAt, Allocation string
}

type mockBankImport struct {
	ID, BankName, PeriodLabel, ImportedAt string
	LineCount                             int
}

type mockBankLine struct {
	ID, ImportID, ValueDate, Description string
	Amount                               int64
	Reference, MatchResult               string
	MatchClaimID                         *string
	MatchScore                           int
	MatchNote                            *string
	Confirmed                            bool
}

type mockBatch struct {
	ID, Name, Period, HospitalID, CreatedAt, CreatedBy, Note string
}

type mockWorkComment struct {
	Author, Text, At string
}

type mockWorkHistory struct {
	At, Actor, Action string
}

type mockWorkItem struct {
	ID, Title, Source         string
	ClaimID, Assignee, Role   *string
	DueDate, Priority, Status string
	Comments                  []mockWorkComment
	History                   []mockWorkHistory
	CreatedAt                 string
}

type mockResumeItem struct {
	ClaimID, SEP, Kategori, NamaItem, SubKategori, Tanggal, HasilCatatan string
	Jumlah                                                               float64
	Satuan                                                               string
	HargaSatuan, Subtotal                                                int64
	Reviewed                                                             bool
}

type mockAuditLog struct {
	At, Actor, ActorRole, Action, Entity, EntityID string
	Before, After, IP, CorrelationID, Scope        string
	StatusCode                                     int
	PartnerName                                    string
}

type mockPartner struct {
	ID, Company, Contact                string
	RateLimitPerMin                     int
	WhitelistIPs                        []string
	ConsumerID                          string
	ConsumerSecretMasked, UserKeyMasked string
	Active                              bool
	HospitalIDs                         []string
	CreatedAt                           string
}

type mockAPIIntake struct {
	ID, PartnerID, HospitalID, SEPNo, PatientName, INACBGCode string
	HospitalTotal, JKNTotal, Gap                              int64
	Status, ErrorCode                                         string
	LatencyMs                                                 int
	At                                                        string
	BPJSShare, InsurerShare, PatientShare                     int64
}

type mockRegulation struct {
	Number, Title, Article, Requirement, EffectiveDate string
	Module, Feature, Status, LastReviewed, Reviewer    string
	Note                                               string
}

type mockNotification struct {
	At, Channel, Template string
	ClaimID               *string
	Recipient, Message    string
	Status                string
}

type mockRejectionCode struct {
	Code, Label string
}

type mockScrubRule struct {
	ID, Name, Category, Severity, Message, FixSuggestion, SourceRef string
	Enabled                                                         bool
}
