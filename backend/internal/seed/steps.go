package seed

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"medpay_spkd/internal/model"
)

// ── Akses: izin, peran, RS, pengguna ─────────────────────────────────────────

func (s *seeder) seedHospitals() error {
	s.hospitalIDs = make(map[string]uuid.UUID, len(s.data.Hospitals))
	for _, h := range s.data.Hospitals {
		reg := s.fixed(h.RegisteredAt)
		m := model.Hospital{
			BaseModel:        stamp(reg),
			KodeFaskes:       h.KodeFaskes,
			Name:             h.Name,
			City:             h.City,
			Class:            h.Class,
			Active:           h.Active,
			EklaimConfigured: h.EklaimConfigured,
			RegisteredAt:     reg,
		}
		if err := s.tx.Create(&m).Error; err != nil {
			return err
		}
		s.hospitalIDs[h.ID] = m.ID

		// Parameter KAPJ default per RS (BR-09).
		kapj := model.KAPJSetting{
			HospitalID:         m.ID,
			BPJSSharePct:       75,
			CopayEnabled:       true,
			PatientCopayPct:    5,
			OutpatientCopayCap: 300_000,
			InpatientCopayCap:  3_000_000,
		}
		if err := s.tx.Create(&kapj).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *seeder) seedUsers() error {
	// Semua akun demo memakai password yang sama, cukup di-hash sekali.
	hash, err := hashPassword(DemoPassword)
	if err != nil {
		return err
	}
	s.userIDs = map[string]uuid.UUID{}
	s.userByName = map[string]uuid.UUID{}
	s.userByEmail = map[string]uuid.UUID{}
	s.userHospital = map[string]uuid.UUID{}
	for _, u := range s.data.Users {
		m := model.User{
			Name:         u.Name,
			Email:        strings.ToLower(u.Email),
			Initials:     u.Initials,
			PasswordHash: hash,
			MFAEnabled:   u.MFAEnabled,
			HospitalID:   opt(s.hospitalIDs, u.HospitalID),
			RoleID:       s.must(s.roleIDs, u.Role, "peran"),
		}
		if s.err != nil {
			return nil
		}
		if err := s.tx.Create(&m).Error; err != nil {
			return err
		}
		s.userIDs[u.ID] = m.ID
		s.userByName[m.Name] = m.ID
		s.userByEmail[m.Email] = m.ID
		if m.HospitalID != nil {
			s.userHospital[m.Name] = *m.HospitalID
		}
	}
	return nil
}

// ── Penjamin & ID master ─────────────────────────────────────────────────────

func (s *seeder) seedInsurers() error {
	s.insurerIDs = map[string]uuid.UUID{}
	for _, i := range s.data.Insurers {
		m := model.Insurer{
			Name:           i.Name,
			Type:           model.InsurerType(i.Type),
			AvgPaymentDays: i.AvgPaymentDays,
			DeadlineDays:   i.DeadlineDays,
			CeilingPolicy:  i.CeilingPolicy,
			Verified:       i.Verified,
		}
		if err := s.tx.Create(&m).Error; err != nil {
			return err
		}
		s.insurerIDs[i.ID] = m.ID
	}
	return nil
}

// loadMaster memuat ID data master (diisi Master) untuk menghubungkan data demo.
func (s *seeder) loadMaster() error {
	s.roleIDs, s.ruleIDs, s.rejectionIDs, s.categoryIDs = map[string]uuid.UUID{}, map[string]uuid.UUID{}, map[string]uuid.UUID{}, map[string]uuid.UUID{}
	var roles []model.Role
	var rules []model.ScrubRule
	var rejections []model.RejectionCode
	var cats []model.ResumeCategory
	for _, q := range []func() error{
		func() error { return s.tx.Find(&roles).Error },
		func() error { return s.tx.Find(&rules).Error },
		func() error { return s.tx.Find(&rejections).Error },
		func() error { return s.tx.Find(&cats).Error },
	} {
		if err := q(); err != nil {
			return err
		}
	}
	for _, r := range roles {
		s.roleIDs[r.Code] = r.ID
	}
	for _, r := range rules {
		s.ruleIDs[r.Code] = r.ID
	}
	for _, r := range rejections {
		s.rejectionIDs[r.Code] = r.ID
	}
	for _, c := range cats {
		s.categoryIDs[c.Code] = c.ID
	}
	return nil
}

func (s *seeder) seedRegulations() error {
	for _, r := range s.data.Regulations {
		m := model.Regulation{
			Number:        r.Number,
			Title:         r.Title,
			Article:       r.Article,
			Requirement:   r.Requirement,
			EffectiveDate: s.fixedPtr(r.EffectiveDate),
			Module:        r.Module,
			Feature:       r.Feature,
			Status:        r.Status,
			LastReviewed:  s.fixedPtr(r.LastReviewed),
			Note:          r.Note,
			ReviewerID:    opt(s.userByName, r.Reviewer),
		}
		if s.err != nil {
			return nil
		}
		if err := s.tx.Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *seeder) seedPartners() error {
	s.partnerIDs = map[string]uuid.UUID{}
	s.partnerByName = map[string]uuid.UUID{}
	for _, p := range s.data.Partners {
		m := model.Partner{
			BaseModel:            stamp(s.at(p.CreatedAt)),
			Company:              p.Company,
			Contact:              p.Contact,
			RateLimitPerMin:      p.RateLimitPerMin,
			ConsumerID:           p.ConsumerID,
			ConsumerSecretHash:   demoSecretHash("cs", p.ConsumerID),
			UserKeyHash:          demoSecretHash("uk", p.ConsumerID),
			ConsumerSecretMasked: p.ConsumerSecretMasked,
			UserKeyMasked:        p.UserKeyMasked,
			Active:               p.Active,
		}
		for _, ip := range p.WhitelistIPs {
			m.WhitelistIPs = append(m.WhitelistIPs, model.PartnerWhitelistIP{IP: ip})
		}
		for _, hid := range p.HospitalIDs {
			m.Hospitals = append(m.Hospitals, model.Hospital{
				BaseModel: model.BaseModel{ID: s.must(s.hospitalIDs, hid, "RS")},
			})
		}
		if s.err != nil {
			return nil
		}
		// Omit "Hospitals.*": RS sudah ada, cukup isi tabel join partner_hospitals.
		if err := s.tx.Omit("Hospitals.*").Create(&m).Error; err != nil {
			return err
		}
		s.partnerIDs[p.ID] = m.ID
		s.partnerByName[m.Company] = m.ID

		// Satu user key per RS yang dilayani partner (lihat demoUserKey).
		for _, hid := range p.HospitalIDs {
			key := model.PartnerKey{
				PartnerID:  m.ID,
				HospitalID: s.must(s.hospitalIDs, hid, "RS"),
				KeyHash:    sha256Hex(demoUserKey(p.ConsumerID, s.kodeFaskes(hid))),
				KeyMasked:  p.UserKeyMasked,
				Active:     true,
			}
			if err := s.tx.Create(&key).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// Kredensial Engine API demo (deterministik, HANYA untuk lokal). Data demo hanya
// menyimpan versi tersamar, jadi nilai aslinya dibentuk dari consumer ID:
//
//	consumer secret = demo-cs-<consumerId>              mis. demo-cs-CONS-SOL-8811
//	user key        = demo-uk-<consumerId>-<kodeFaskes> mis. demo-uk-CONS-SOL-8811-0304R001
func demoSecretHash(kind, consumerID string) string {
	return sha256Hex("demo-" + kind + "-" + consumerID)
}

func demoUserKey(consumerID, kodeFaskes string) string {
	return "demo-uk-" + consumerID + "-" + kodeFaskes
}

func sha256Hex(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

func (s *seeder) kodeFaskes(seedHospitalID string) string {
	for _, h := range s.data.Hospitals {
		if h.ID == seedHospitalID {
			return h.KodeFaskes
		}
	}
	return ""
}

// ── Klaim ────────────────────────────────────────────────────────────────────

func (s *seeder) seedBatches() error {
	s.batchIDs = map[string]uuid.UUID{}
	for _, b := range s.data.Batches {
		m := model.Batch{
			BaseModel:   stamp(s.at(b.CreatedAt)),
			BatchNo:     b.ID,
			Name:        b.Name,
			Period:      b.Period,
			Note:        b.Note,
			HospitalID:  s.must(s.hospitalIDs, b.HospitalID, "RS"),
			CreatedByID: s.must(s.userByName, b.CreatedBy, "pengguna"),
		}
		if b.InsurerID != "" {
			insName := ""
			for _, in := range s.data.Insurers {
				if in.ID == b.InsurerID {
					insName = in.Name
				}
			}
			from, to := s.at(b.PeriodStart), s.at(b.PeriodEnd)
			m.InsurerID = opt(s.insurerIDs, b.InsurerID)
			m.PeriodStart, m.PeriodEnd = &from, &to
			m.Period = to.Format("2006-01")
			m.Name = fmt.Sprintf("%s — %s s/d %s", insName, from.Format("2006-01-02"), to.Format("2006-01-02"))
		}
		if s.err != nil {
			return nil
		}
		if err := s.tx.Create(&m).Error; err != nil {
			return err
		}
		s.batchIDs[b.ID] = m.ID
	}
	return nil
}

func (s *seeder) seedClaims() error {
	s.claimIDs = map[string]uuid.UUID{}
	s.claimHospitals = map[string]uuid.UUID{}
	for _, c := range s.data.Claims {
		m := model.Claim{
			BaseModel: model.BaseModel{
				CreatedAt: s.at(c.CreatedAt),
				UpdatedAt: s.at(c.UpdatedAt),
			},
			ClaimNo:             c.ID,
			SEPNo:               c.SEPNo,
			BPJSCardNo:          c.BPJSCardNo,
			MRN:                 c.MRN,
			PatientName:         c.PatientName,
			DPJP:                c.DPJP,
			ParticipantType:     c.ParticipantType,
			CareType:            c.CareType,
			CareClass:           c.CareClass,
			Sex:                 c.Sex,
			DateOfBirth:         s.fixedPtr(c.DateOfBirth),
			AdmissionDate:       s.atPtr(c.AdmissionDate),
			DischargeDate:       s.atPtr(c.DischargeDate),
			LOS:                 c.LOS,
			PrimaryDiagnosis:    c.PrimaryDiagnosis,
			INACBGCode:          c.INACBGCode,
			INACBGDescription:   c.INACBGDescription,
			INACBGBaseTariff:    c.INACBGBaseTariff,
			SpecialCMG:          model.SpecialCMG(c.SpecialCMG),
			HospitalTariff:      model.HospitalTariff(c.HospitalTariff),
			TariffResolvedAt:    s.atPtr(c.TariffResolvedAt),
			ResolutionSignature: c.ResolutionSignature,
			DeclarationChecked:  c.DeclarationChecked,
			PolicyNo:            c.PolicyNo,
			InsuredName:         c.InsuredName,
			InsurerNameExtra:    c.InsurerNameExtra,
			InsuranceType:       c.InsuranceType,
			InsuranceMemberNo:   c.InsuranceMemberNo,
			CoverageType:        c.CoverageType,
			PolicyCeiling:       c.PolicyCeiling,
			Status:              model.ClaimStatus(c.Status),
			SubmittedAt:         s.atPtr(deref(c.SubmittedAt)),
			RejectionReasonNote: c.RejectionReasonNote,
			HospitalID:          s.must(s.hospitalIDs, c.HospitalID, "RS"),
			InsurerID:           opt(s.insurerIDs, c.InsurerID),
			BatchID:             opt(s.batchIDs, deref(c.BatchID)),
			RejectionCodeID:     opt(s.rejectionIDs, c.RejectionReasonCode),
			CreatedByID:         s.must(s.userByName, c.CreatedBy, "pengguna"),
		}
		for i, dx := range c.SecondaryDiagnoses {
			m.SecondaryDiagnoses = append(m.SecondaryDiagnoses, model.ClaimSecondaryDiagnosis{ICD10Code: dx, Seq: i + 1})
		}
		for i, px := range c.Procedures {
			m.Procedures = append(m.Procedures, model.ClaimProcedure{ICD9Code: px, Seq: i + 1})
		}
		for _, h := range c.StatusHistory {
			m.StatusHistories = append(m.StatusHistories, model.ClaimStatusHistory{
				BaseModel:  stamp(s.at(h.At)),
				FromStatus: model.ClaimStatus(h.From),
				ToStatus:   model.ClaimStatus(h.To),
				Note:       h.Note,
				ActorID:    opt(s.userByName, h.Actor), // aktor "Sistem" → NULL
			})
		}
		if s.err != nil {
			return nil
		}
		// Diagnosis, prosedur, dan riwayat status ikut tersimpan sebagai asosiasi.
		if err := s.tx.Create(&m).Error; err != nil {
			return err
		}
		s.claimIDs[c.ID] = m.ID
		s.claimHospitals[c.ID] = m.HospitalID
	}
	return nil
}

func (s *seeder) seedResumeItems() error {
	items := make([]model.ResumeItem, 0, len(s.data.ResumeItems))
	for _, r := range s.data.ResumeItems {
		items = append(items, model.ResumeItem{
			SEPNo:            r.SEP,
			ItemName:         r.NamaItem,
			SubCategory:      r.SubKategori,
			ServiceDate:      s.atPtr(r.Tanggal),
			ResultNote:       r.HasilCatatan,
			Quantity:         r.Jumlah,
			Unit:             r.Satuan,
			UnitPrice:        r.HargaSatuan,
			Subtotal:         r.Subtotal,
			Reviewed:         r.Reviewed,
			ClaimID:          s.must(s.claimIDs, r.ClaimID, "klaim"),
			ResumeCategoryID: s.must(s.categoryIDs, r.Kategori, "kategori resume"),
		})
	}
	if s.err != nil || len(items) == 0 {
		return nil
	}
	return s.tx.CreateInBatches(&items, 100).Error
}

// ── Uang & rekonsiliasi ──────────────────────────────────────────────────────

func (s *seeder) seedBank() error {
	s.importIDs = map[string]uuid.UUID{}
	s.lineByRef = map[string]uuid.UUID{}
	for _, imp := range s.data.BankImports {
		importedAt := s.at(imp.ImportedAt)
		importer := s.importerOf(imp.ID)
		m := model.BankImport{
			BaseModel:    stamp(importedAt),
			ImportNo:     imp.ID,
			BankName:     imp.BankName,
			PeriodLabel:  imp.PeriodLabel,
			LineCount:    imp.LineCount,
			HospitalID:   s.hospitalOfImport(imp.ID),
			ImportedByID: importer,
		}
		for _, bl := range s.data.BankLines {
			if bl.ImportID != imp.ID {
				continue
			}
			line := model.BankLine{
				BaseModel:    stamp(importedAt),
				LineNo:       bl.ID,
				ValueDate:    s.at(bl.ValueDate),
				Description:  bl.Description,
				Amount:       bl.Amount,
				Reference:    bl.Reference,
				MatchResult:  model.MatchResult(bl.MatchResult),
				MatchScore:   bl.MatchScore,
				MatchNote:    deref(bl.MatchNote),
				Confirmed:    bl.Confirmed,
				MatchClaimID: opt(s.claimIDs, deref(bl.MatchClaimID)),
			}
			if bl.Confirmed {
				line.ConfirmedAt = &importedAt
				line.ConfirmedByID = &importer
			}
			m.Lines = append(m.Lines, line)
		}
		if s.err != nil {
			return nil
		}
		if err := s.tx.Create(&m).Error; err != nil {
			return err
		}
		s.importIDs[imp.ID] = m.ID
		for _, l := range m.Lines {
			if l.Confirmed && l.Reference != "" {
				s.lineByRef[l.Reference] = l.ID
			}
		}
	}
	return nil
}

// hospitalOfImport menebak RS pemilik impor dari klaim yang dicocokkan;
// data impor demo tidak menyimpan RS, dan seluruh transaksinya milik RS pertama.
func (s *seeder) hospitalOfImport(importID string) uuid.UUID {
	for _, bl := range s.data.BankLines {
		if bl.ImportID == importID && bl.MatchClaimID != nil {
			if hid, ok := s.claimHospitals[*bl.MatchClaimID]; ok {
				return hid
			}
		}
	}
	return s.must(s.hospitalIDs, s.data.Hospitals[0].ID, "RS")
}

// importerOf mencari pengimpor dari audit log; fallback ke pengguna FINANCE pertama.
func (s *seeder) importerOf(importID string) uuid.UUID {
	for _, a := range s.data.AuditLog {
		if a.Action == "IMPORT_BANK_STATEMENT" && a.EntityID == importID {
			if id, ok := s.userByName[a.Actor]; ok {
				return id
			}
		}
	}
	for _, u := range s.data.Users {
		if u.Role == model.RoleFinance {
			return s.must(s.userIDs, u.ID, "pengguna")
		}
	}
	if s.err == nil {
		s.err = fmt.Errorf("tidak ada pengguna FINANCE untuk impor %s", importID)
	}
	return uuid.Nil
}

func (s *seeder) seedPayments() error {
	for _, p := range s.data.Payments {
		m := model.Payment{
			BaseModel:    stamp(s.at(p.CreatedAt)),
			PaymentNo:    p.ID,
			ReceivedDate: s.at(p.ReceivedDate),
			Amount:       p.Amount,
			Payer:        p.Payer,
			TransferRef:  p.TransferRef,
			Notes:        p.Notes,
			Allocation:   p.Allocation,
			ClaimID:      s.must(s.claimIDs, p.ClaimID, "klaim"),
			CreatedByID:  s.must(s.userByName, p.CreatedBy, "pengguna"),
			BankLineID:   opt(s.lineByRef, p.TransferRef),
		}
		if s.err != nil {
			return nil
		}
		if err := s.tx.Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}

// ── Operasional & sistem ─────────────────────────────────────────────────────

func (s *seeder) seedWorkItems() error {
	for _, w := range s.data.WorkItems {
		m := model.WorkItem{
			BaseModel:  stamp(s.at(w.CreatedAt)),
			ItemNo:     w.ID,
			Title:      w.Title,
			Source:     w.Source,
			Role:       deref(w.Role),
			DueDate:    s.at(w.DueDate),
			Priority:   w.Priority,
			Status:     w.Status,
			ClaimID:    opt(s.claimIDs, deref(w.ClaimID)),
			AssigneeID: opt(s.userIDs, deref(w.Assignee)),
			HospitalID: s.hospitalOfWorkItem(w),
			// CreatedByID nil: work item demo dibuat otomatis oleh sistem.
		}
		for _, c := range w.Comments {
			m.Comments = append(m.Comments, model.WorkItemComment{
				BaseModel: stamp(s.at(c.At)),
				Body:      c.Text,
				AuthorID:  s.must(s.userByName, c.Author, "pengguna"),
			})
		}
		if len(w.History) == 0 {
			m.Histories = append(m.Histories, model.WorkItemHistory{
				BaseModel: stamp(s.at(w.CreatedAt)),
				Action:    "Dibuat otomatis oleh sistem (" + w.Source + ")",
			})
		}
		for _, h := range w.History {
			m.Histories = append(m.Histories, model.WorkItemHistory{
				BaseModel: stamp(s.at(h.At)),
				Action:    h.Action,
				ActorID:   opt(s.userByName, h.Actor),
			})
		}
		if s.err != nil {
			return nil
		}
		if err := s.tx.Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}

// hospitalOfWorkItem: RS klaim terkait, lalu RS penerima tugas; data tugas demo tidak
// menyimpan RS, dan seluruh transaksinya milik RS pertama.
func (s *seeder) hospitalOfWorkItem(w seedWorkItem) *uuid.UUID {
	if hid, ok := s.claimHospitals[deref(w.ClaimID)]; ok {
		return &hid
	}
	for _, u := range s.data.Users {
		if u.ID == deref(w.Assignee) && u.HospitalID != "" {
			return opt(s.hospitalIDs, u.HospitalID)
		}
	}
	return opt(s.hospitalIDs, s.data.Hospitals[0].ID)
}

func (s *seeder) seedNotifications() error {
	for _, n := range s.data.Notifications {
		m := model.Notification{
			BaseModel:       stamp(s.at(n.At)),
			Channel:         n.Channel,
			Template:        n.Template,
			Message:         n.Message,
			Status:          n.Status,
			RecipientUserID: s.must(s.userByEmail, strings.ToLower(n.Recipient), "penerima"),
			ClaimID:         opt(s.claimIDs, deref(n.ClaimID)),
		}
		if s.err != nil {
			return nil
		}
		if err := s.tx.Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *seeder) seedAPIIntakes() error {
	s.intakeHospital = map[string]uuid.UUID{}
	for _, a := range s.data.APIIntakes {
		at := s.at(a.At)
		m := model.APIIntake{
			BaseModel:     stamp(at),
			IntakeNo:      a.ID,
			SEPNo:         a.SEPNo,
			PatientName:   a.PatientName,
			INACBGCode:    a.INACBGCode,
			HospitalTotal: a.HospitalTotal,
			JKNTotal:      a.JKNTotal,
			Gap:           a.Gap,
			BPJSShare:     a.BPJSShare,
			InsurerShare:  a.InsurerShare,
			PatientShare:  a.PatientShare,
			Status:        a.Status,
			ErrorCode:     a.ErrorCode,
			LatencyMs:     a.LatencyMs,
			ReceivedAt:    at,
			PartnerID:     s.must(s.partnerIDs, a.PartnerID, "partner"),
			HospitalID:    s.must(s.hospitalIDs, a.HospitalID, "RS"),
		}
		if s.err != nil {
			return nil
		}
		if err := s.tx.Create(&m).Error; err != nil {
			return err
		}
		s.intakeHospital[a.ID] = m.HospitalID
	}
	return nil
}

func (s *seeder) seedAuditLogs() error {
	logs := make([]model.AuditLog, 0, len(s.data.AuditLog))
	for _, a := range s.data.AuditLog {
		scope := a.Scope
		if scope == "" {
			scope = model.AuditScopeWeb
		}
		hospitalID := opt(s.userHospital, a.Actor)
		if scope == model.AuditScopeEngineAPI {
			hospitalID = opt(s.intakeHospital, a.EntityID)
		}
		logs = append(logs, model.AuditLog{
			CreatedAt:     s.at(a.At),
			ActorName:     a.Actor,
			ActorRole:     a.ActorRole,
			Action:        a.Action,
			Entity:        a.Entity,
			EntityID:      a.EntityID,
			Before:        a.Before,
			After:         a.After,
			IP:            a.IP,
			CorrelationID: a.CorrelationID,
			Scope:         scope,
			StatusCode:    a.StatusCode,
			ActorID:       opt(s.userByName, a.Actor),
			PartnerID:     opt(s.partnerByName, a.PartnerName),
			HospitalID:    hospitalID,
		})
	}
	if s.err != nil || len(logs) == 0 {
		return nil
	}
	return s.tx.CreateInBatches(&logs, 100).Error
}

// ── Override scrubber ────────────────────────────────────────────────────────

func (s *seeder) seedScrubOverrides() error {
	for _, o := range seedScrubOverrides {
		m := model.ClaimScrubOverride{
			Reason:         o.Reason,
			ClaimID:        s.must(s.claimIDs, o.ClaimID, "klaim"),
			ScrubRuleID:    s.must(s.ruleIDs, o.RuleID, "aturan scrubber"),
			OverriddenByID: s.must(s.userByName, o.By, "pengguna"),
		}
		if s.err != nil {
			return nil
		}
		if err := s.tx.Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}
