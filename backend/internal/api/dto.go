package api

import (
	"sort"

	"medpay_spkd/internal/auth"
	"medpay_spkd/internal/engine"
	"medpay_spkd/internal/model"
)

// Pemetaan model GORM → bentuk JSON yang dipakai frontend.

func userDTO(u model.User) J {
	return J{
		"id": idStr(u.ID), "name": u.Name, "email": u.Email, "role": auth.RoleCode(u),
		"hospitalId": idPtrStr(u.HospitalID), "mfaEnabled": u.MFAEnabled, "initials": u.Initials,
		"active": u.Active, "mustChangePassword": u.MustChangePassword,
	}
}

func hospitalDTO(h model.Hospital) J {
	return J{
		"id": idStr(h.ID), "kodeFaskes": h.KodeFaskes, "name": h.Name, "city": h.City, "class": h.Class,
		"active": h.Active, "eklaimConfigured": h.EklaimConfigured, "registeredAt": fmtDate(h.RegisteredAt),
	}
}

func insurerDTO(i model.Insurer) J {
	return J{
		"id": idStr(i.ID), "name": i.Name, "type": string(i.Type), "avgPaymentDays": i.AvgPaymentDays,
		"deadlineDays": i.DeadlineDays, "ceilingPolicy": i.CeilingPolicy, "verified": i.Verified,
	}
}

func scrubRuleDTO(r model.ScrubRule) J {
	return J{
		"id": idStr(r.ID), "code": r.Code, "name": r.Name, "category": r.Category, "severity": string(r.Severity),
		"message": r.Message, "fixSuggestion": r.FixSuggestion, "sourceRef": r.SourceRef, "enabled": r.Enabled,
	}
}

func regulationDTO(r model.Regulation) J {
	reviewer := ""
	if r.Reviewer != nil {
		reviewer = r.Reviewer.Name
	}
	return J{
		"id": idStr(r.ID), "number": r.Number, "title": r.Title, "article": r.Article,
		"requirement": r.Requirement, "effectiveDate": fmtDatePtr(r.EffectiveDate), "module": r.Module,
		"feature": r.Feature, "status": r.Status, "lastReviewed": fmtDatePtr(r.LastReviewed),
		"reviewer": reviewer, "note": r.Note,
	}
}

func notificationDTO(n model.Notification) J {
	claimNo := ""
	if n.Claim != nil {
		claimNo = n.Claim.ClaimNo
	}
	recipient := ""
	if n.RecipientUser != nil {
		recipient = n.RecipientUser.Email
	}
	return J{
		"id": idStr(n.ID), "at": fmtStamp(n.CreatedAt), "channel": n.Channel, "template": n.Template,
		"claimId": idPtrOrNil(n.ClaimID), "claimNo": claimNo, "recipient": recipient, "message": n.Message, "status": n.Status,
		"read": n.ReadAt != nil, "recipientUserId": idStr(n.RecipientUserID),
	}
}

func partnerDTO(p model.Partner) J {
	ips := []string{}
	for _, w := range p.WhitelistIPs {
		ips = append(ips, w.IP)
	}
	hospitals := []string{}
	for _, h := range p.Hospitals {
		hospitals = append(hospitals, idStr(h.ID))
	}
	keys := []J{}
	for _, k := range p.Keys {
		keys = append(keys, J{"hospitalId": idStr(k.HospitalID), "keyMasked": k.KeyMasked, "active": k.Active,
			"lastUsedAt": stampOrNil(k.LastUsedAt)})
	}
	return J{
		"id": idStr(p.ID), "company": p.Company, "contact": p.Contact, "rateLimitPerMin": p.RateLimitPerMin,
		"whitelistIps": ips, "consumerId": p.ConsumerID, "consumerSecretMasked": p.ConsumerSecretMasked,
		"userKeyMasked": p.UserKeyMasked, "active": p.Active, "hospitalIds": hospitals,
		"createdAt": fmtStamp(p.CreatedAt), "userKeys": keys,
	}
}

func userName(u *model.User, fallback string) string {
	if u == nil {
		return fallback
	}
	return u.Name
}

// ── Klaim ────────────────────────────────────────────────────────────────────

// tariffJSON & cmgJSON: urutan & tipe field identik dengan model agar bisa dikonversi langsung.
type tariffJSON struct {
	ProsedurNonBedah int64 `json:"prosedurNonBedah"`
	ProsedurBedah    int64 `json:"prosedurBedah"`
	Konsultasi       int64 `json:"konsultasi"`
	TenagaAhli       int64 `json:"tenagaAhli"`
	Keperawatan      int64 `json:"keperawatan"`
	Penunjang        int64 `json:"penunjang"`
	Radiologi        int64 `json:"radiologi"`
	Laboratorium     int64 `json:"laboratorium"`
	PelayananDarah   int64 `json:"pelayananDarah"`
	Rehabilitasi     int64 `json:"rehabilitasi"`
	KamarAkomodasi   int64 `json:"kamarAkomodasi"`
	RawatIntensif    int64 `json:"rawatIntensif"`
	Obat             int64 `json:"obat"`
	ObatKronis       int64 `json:"obatKronis"`
	ObatKemo         int64 `json:"obatKemo"`
	Alkes            int64 `json:"alkes"`
	BMHP             int64 `json:"bmhp"`
	SewaAlat         int64 `json:"sewaAlat"`
}

type cmgJSON struct {
	SpecialProcedure     int64 `json:"specialProcedure"`
	SpecialProsthesis    int64 `json:"specialProsthesis"`
	SpecialInvestigation int64 `json:"specialInvestigation"`
	SpecialDrug          int64 `json:"specialDrug"`
	SubAcute             int64 `json:"subAcute"`
	Chronic              int64 `json:"chronic"`
}

type statusHistoryJSON struct {
	At    string `json:"at"`
	Actor string `json:"actor"`
	From  string `json:"from"`
	To    string `json:"to"`
	Note  string `json:"note"`
}

type claimJSON struct {
	ID                  string              `json:"id"`
	ClaimNo             string              `json:"claimNo"`
	HospitalID          string              `json:"hospitalId"`
	SEPNo               string              `json:"sepNo"`
	BPJSCardNo          string              `json:"bpjsCardNo"`
	MRN                 string              `json:"mrn"`
	PatientName         string              `json:"patientName"`
	DPJP                string              `json:"dpjp"`
	ParticipantType     string              `json:"participantType"`
	CareType            string              `json:"careType"`
	CareClass           string              `json:"careClass"`
	Sex                 string              `json:"sex"`
	DateOfBirth         string              `json:"dateOfBirth"`
	AdmissionDate       string              `json:"admissionDate"`
	DischargeDate       string              `json:"dischargeDate"`
	LOS                 int                 `json:"los"`
	PrimaryDiagnosis    string              `json:"primaryDiagnosis"`
	SecondaryDiagnoses  []string            `json:"secondaryDiagnoses"`
	Procedures          []string            `json:"procedures"`
	INACBGCode          string              `json:"inacbgCode"`
	INACBGDescription   string              `json:"inacbgDescription"`
	INACBGBaseTariff    int64               `json:"inacbgBaseTariff"`
	SpecialCMG          cmgJSON             `json:"specialCmg"`
	HospitalTariff      tariffJSON          `json:"hospitalTariff"`
	TariffResolvedAt    string              `json:"tariffResolvedAt"`
	DeclarationChecked  bool                `json:"declarationChecked"`
	InsurerID           string              `json:"insurerId"`
	PolicyNo            string              `json:"policyNo"`
	InsuredName         string              `json:"insuredName"`
	InsurerNameExtra    string              `json:"insurerNameExtra"`
	InsuranceType       string              `json:"insuranceType"`
	InsuranceMemberNo   string              `json:"insuranceMemberNo"`
	CoverageType        string              `json:"coverageType"`
	PolicyCeiling       *int64              `json:"policyCeiling"`
	Status              string              `json:"status"`
	CreatedAt           string              `json:"createdAt"`
	UpdatedAt           string              `json:"updatedAt"`
	SubmittedAt         any                 `json:"submittedAt"`
	BatchID             any                 `json:"batchId"`
	BatchNo             string              `json:"batchNo"`
	CreatedBy           string              `json:"createdBy"`
	ScrubFindings       []engine.Finding    `json:"scrubFindings"`
	ReadinessScore      int                 `json:"readinessScore"`
	RejectionReasonCode string              `json:"rejectionReasonCode"`
	RejectionReasonNote string              `json:"rejectionReasonNote"`
	ResolutionSignature string              `json:"resolutionSignature"`
	StatusHistory       []statusHistoryJSON `json:"statusHistory"`
	Gap                 int64               `json:"gap"`
	HospitalTotal       int64               `json:"hospitalTotal"`
	JKNTotal            int64               `json:"jknTotal"`
	Deadline            string              `json:"deadline"`         // batas pengajuan 6 bulan (BR-08)
	DeadlineDaysLeft    *int                `json:"deadlineDaysLeft"` // nil bila tanggal masuk kosong
	DeadlineZone        string              `json:"deadlineZone"`     // aman | waspada | kritis | lewat
}

// claimDTO memetakan klaim. sr = hasil scrub (nil = tidak dihitung).
// Riwayat status hanya terisi bila StatusHistories sudah di-preload.
func claimDTO(c *model.Claim, sr *engine.ScrubResult) claimJSON {
	out := claimJSON{
		ID: idStr(c.ID), ClaimNo: c.ClaimNo, HospitalID: idStr(c.HospitalID), SEPNo: c.SEPNo, BPJSCardNo: c.BPJSCardNo,
		MRN: c.MRN, PatientName: c.PatientName, DPJP: c.DPJP, ParticipantType: c.ParticipantType,
		CareType: c.CareType, CareClass: c.CareClass, Sex: c.Sex,
		DateOfBirth: fmtDatePtr(c.DateOfBirth), AdmissionDate: fmtDatePtr(c.AdmissionDate),
		DischargeDate: fmtDatePtr(c.DischargeDate), LOS: c.LOS,
		PrimaryDiagnosis:   c.PrimaryDiagnosis,
		SecondaryDiagnoses: engine.SecondaryCodes(c), Procedures: engine.ProcedureCodes(c),
		INACBGCode: c.INACBGCode, INACBGDescription: c.INACBGDescription, INACBGBaseTariff: c.INACBGBaseTariff,
		SpecialCMG: cmgJSON(c.SpecialCMG), HospitalTariff: tariffJSON(c.HospitalTariff),
		TariffResolvedAt: fmtStampPtr(c.TariffResolvedAt), DeclarationChecked: c.DeclarationChecked,
		InsurerID: idPtrStr(c.InsurerID), PolicyNo: c.PolicyNo, InsuredName: c.InsuredName,
		InsurerNameExtra: c.InsurerNameExtra, InsuranceType: c.InsuranceType,
		InsuranceMemberNo: c.InsuranceMemberNo, CoverageType: c.CoverageType, PolicyCeiling: c.PolicyCeiling,
		Status: string(c.Status), CreatedAt: fmtStamp(c.CreatedAt), UpdatedAt: fmtStamp(c.UpdatedAt),
		SubmittedAt: stampOrNil(c.SubmittedAt), CreatedBy: userName(c.CreatedBy, ""),
		ScrubFindings: []engine.Finding{}, RejectionReasonNote: c.RejectionReasonNote,
		ResolutionSignature: c.ResolutionSignature, StatusHistory: []statusHistoryJSON{},
		Gap: engine.Gap(c), HospitalTotal: engine.SumTariff(c.HospitalTariff),
		JKNTotal: engine.TotalJKN(c.INACBGBaseTariff, c.SpecialCMG),
	}
	if di := engine.Deadline(c.AdmissionDate, engine.Today()); di.Deadline != nil {
		left := di.DaysLeft
		out.Deadline, out.DeadlineDaysLeft, out.DeadlineZone = fmtDate(*di.Deadline), &left, di.Zone
	}
	if c.Batch != nil {
		out.BatchID, out.BatchNo = idStr(c.Batch.ID), c.Batch.BatchNo
	}
	if c.RejectionCode != nil {
		out.RejectionReasonCode = c.RejectionCode.Code
	}
	if sr != nil {
		out.ScrubFindings = sr.Findings
		out.ReadinessScore = sr.Score
	}
	hist := append([]model.ClaimStatusHistory(nil), c.StatusHistories...)
	sort.SliceStable(hist, func(i, j int) bool { return hist[i].CreatedAt.Before(hist[j].CreatedAt) })
	for _, h := range hist {
		out.StatusHistory = append(out.StatusHistory, statusHistoryJSON{
			At: fmtStamp(h.CreatedAt), Actor: userName(h.Actor, "Sistem"),
			From: string(h.FromStatus), To: string(h.ToStatus), Note: h.Note,
		})
	}
	return out
}

// claimBrief = ringkasan klaim untuk badge, pencarian, dan relasi.
func claimBrief(c *model.Claim) J {
	return J{
		"id": idStr(c.ID), "claimNo": c.ClaimNo, "hospitalId": idStr(c.HospitalID), "sepNo": c.SEPNo, "mrn": c.MRN,
		"patientName": c.PatientName, "status": string(c.Status), "insurerId": idPtrStr(c.InsurerID),
	}
}

func resumeDTO(r model.ResumeItem, claimNo string) J {
	kategori := ""
	if r.ResumeCategory != nil {
		kategori = r.ResumeCategory.Code
	}
	return J{
		"id": idStr(r.ID), "claimId": idStr(r.ClaimID), "claimNo": claimNo, "sep": r.SEPNo, "kategori": kategori, "namaItem": r.ItemName,
		"subKategori": r.SubCategory, "tanggal": fmtDatePtr(r.ServiceDate), "hasilCatatan": r.ResultNote,
		"jumlah": r.Quantity, "satuan": r.Unit, "hargaSatuan": r.UnitPrice, "subtotal": r.Subtotal,
		"reviewed": r.Reviewed,
	}
}

// ── Uang ─────────────────────────────────────────────────────────────────────

func paymentDTO(p model.Payment, claimNo string) J {
	return J{
		"id": idStr(p.ID), "paymentNo": p.PaymentNo, "claimId": idStr(p.ClaimID), "claimNo": claimNo, "receivedDate": fmtDate(p.ReceivedDate), "amount": p.Amount,
		"payer": p.Payer, "transferRef": p.TransferRef, "notes": p.Notes,
		"createdBy": userName(p.CreatedBy, ""), "createdAt": fmtStamp(p.CreatedAt), "allocation": p.Allocation,
	}
}

func batchDTO(b model.Batch) J {
	return J{
		"id": idStr(b.ID), "batchNo": b.BatchNo, "name": b.Name, "period": b.Period, "hospitalId": idStr(b.HospitalID),
		"createdAt": fmtStamp(b.CreatedAt), "createdBy": userName(b.CreatedBy, ""), "note": b.Note,
	}
}

func bankImportDTO(b model.BankImport) J {
	return J{
		"id": idStr(b.ID), "importNo": b.ImportNo, "bankName": b.BankName, "periodLabel": b.PeriodLabel,
		"importedAt": fmtStamp(b.CreatedAt), "lineCount": b.LineCount, "hospitalId": idStr(b.HospitalID),
	}
}

func bankLineDTO(l model.BankLine, importNo string) J {
	var note any
	claimNo := ""
	if l.MatchClaim != nil {
		claimNo = l.MatchClaim.ClaimNo
	}
	if l.MatchNote != "" {
		note = l.MatchNote
	}
	return J{
		"id": idStr(l.ID), "lineNo": l.LineNo, "importId": idStr(l.BankImportID), "importNo": importNo, "valueDate": fmtDate(l.ValueDate), "description": l.Description,
		"amount": l.Amount, "reference": l.Reference, "matchResult": string(l.MatchResult),
		"matchClaimId": idPtrOrNil(l.MatchClaimID), "matchClaimNo": claimNo, "matchScore": l.MatchScore, "matchNote": note, "confirmed": l.Confirmed,
	}
}

// ── Operasional & sistem ─────────────────────────────────────────────────────

func workItemDTO(w model.WorkItem) J {
	var assignee, role any
	claimNo := ""
	if w.Claim != nil {
		claimNo = w.Claim.ClaimNo
	}
	if w.AssigneeID != nil {
		assignee = idStr(*w.AssigneeID)
	}
	if w.Role != "" {
		role = w.Role
	}
	comments := append([]model.WorkItemComment(nil), w.Comments...)
	sort.SliceStable(comments, func(i, j int) bool { return comments[i].CreatedAt.Before(comments[j].CreatedAt) })
	cs := []J{}
	for _, c := range comments {
		cs = append(cs, J{"id": idStr(c.ID), "author": userName(c.Author, ""), "text": c.Body, "at": fmtStamp(c.CreatedAt)})
	}
	hist := append([]model.WorkItemHistory(nil), w.Histories...)
	sort.SliceStable(hist, func(i, j int) bool { return hist[i].CreatedAt.Before(hist[j].CreatedAt) })
	hs := []J{}
	for _, h := range hist {
		hs = append(hs, J{"id": idStr(h.ID), "at": fmtStamp(h.CreatedAt), "actor": userName(h.Actor, "Sistem"), "action": h.Action})
	}
	return J{
		"id": idStr(w.ID), "itemNo": w.ItemNo, "title": w.Title, "source": w.Source, "claimId": idPtrOrNil(w.ClaimID), "claimNo": claimNo, "assignee": assignee,
		"role": role, "dueDate": fmtDate(w.DueDate), "priority": w.Priority, "status": w.Status,
		"comments": cs, "history": hs, "createdAt": fmtStamp(w.CreatedAt),
	}
}

func auditDTO(a model.AuditLog) J {
	out := J{
		"id": idStr(a.ID), "at": fmtStamp(a.CreatedAt), "actor": a.ActorName, "actorRole": a.ActorRole,
		"action": a.Action, "entity": a.Entity, "entityId": a.EntityID, "ip": a.IP,
		"correlationId": a.CorrelationID, "scope": a.Scope, "statusCode": a.StatusCode,
	}
	if a.Before != "" {
		out["before"] = a.Before
	}
	if a.After != "" {
		out["after"] = a.After
	}
	if a.Partner != nil {
		out["partnerName"] = a.Partner.Company
	}
	return out
}

func intakeDTO(a model.APIIntake) J {
	return J{
		"id": idStr(a.ID), "intakeNo": a.IntakeNo, "partnerId": idStr(a.PartnerID), "hospitalId": idStr(a.HospitalID), "sepNo": a.SEPNo,
		"patientName": a.PatientName, "inacbgCode": a.INACBGCode, "hospitalTotal": a.HospitalTotal,
		"jknTotal": a.JKNTotal, "gap": a.Gap, "status": a.Status, "errorCode": a.ErrorCode,
		"latencyMs": a.LatencyMs, "at": fmtStamp(a.ReceivedAt), "bpjsShare": a.BPJSShare,
		"insurerShare": a.InsurerShare, "patientShare": a.PatientShare,
	}
}
