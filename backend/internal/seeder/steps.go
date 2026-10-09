package seeder

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"medpay_spkd/internal/models"
)

// ── Akses: izin, peran, RS, pengguna ─────────────────────────────────────────

var permissionLabels = []struct{ code, label string }{
	{models.PermView, "Lihat data"},
	{models.PermCreate, "Buat data"},
	{models.PermEdit, "Ubah data"},
	{models.PermDelete, "Hapus data"},
	{models.PermVerify, "Verifikasi"},
	{models.PermApprove, "Setujui / ajukan"},
	{models.PermExport, "Ekspor laporan"},
	{models.PermPrint, "Cetak"},
	{models.PermAssign, "Tugaskan pekerjaan"},
	{models.PermResolve, "Resolusi tarif (grouper)"},
	{models.PermReopen, "Buka kembali"},
	{models.PermConfigure, "Konfigurasi sistem"},
	{models.PermOverride, "Override temuan scrub"},
}

// rolePermissions mengikuti ROLE_PERMS di mock API frontend; label dari ROLE_LABELS.
var rolePermissions = []struct {
	code, label string
	perms       []string
}{
	{models.RoleSuperAdmin, "SPKD Super Admin", nil}, // nil = semua izin
	{models.RoleHospitalAdmin, "Admin RS", nil},
	{models.RoleManagement, "Manajemen RS", []string{"VIEW", "EXPORT"}},
	{models.RoleBillingStaff, "Staf Billing/Klaim", []string{"VIEW", "CREATE", "EDIT", "DELETE", "EXPORT", "PRINT", "ASSIGN"}},
	{models.RoleCoder, "Coder", []string{"VIEW", "CREATE", "EDIT", "RESOLVE"}},
	{models.RoleReviewer, "Reviewer (Maker-Checker)", []string{"VIEW", "VERIFY", "APPROVE", "REOPEN"}},
	{models.RoleFinance, "Finance", []string{"VIEW", "CREATE", "EDIT", "APPROVE", "EXPORT"}},
	{models.RoleAuditor, "Auditor", []string{"VIEW", "EXPORT"}},
	{models.RoleITSIMRS, "IT / SIMRS", []string{"VIEW", "CONFIGURE"}},
	{models.RoleViewer, "Viewer (Read-only)", []string{"VIEW"}},
}

func (s *seeder) seedRBAC() error {
	perms := make([]models.Permission, len(permissionLabels))
	for i, p := range permissionLabels {
		perms[i] = models.Permission{Code: p.code, Label: p.label}
	}
	if err := s.tx.Create(&perms).Error; err != nil {
		return err
	}
	s.permIDs = make(map[string]uint, len(perms))
	for _, p := range perms {
		s.permIDs[p.Code] = p.ID
	}

	s.roleIDs = make(map[string]uint, len(rolePermissions))
	for _, rp := range rolePermissions {
		role := models.Role{Code: rp.code, Label: rp.label}
		if rp.perms == nil {
			role.Permissions = perms
		} else {
			for _, code := range rp.perms {
				role.Permissions = append(role.Permissions, models.Permission{
					BaseModel: models.BaseModel{ID: s.must(s.permIDs, code, "izin")},
				})
			}
		}
		// Omit "Permissions.*": izin sudah ada, cukup isi tabel join role_permissions.
		if err := s.tx.Omit("Permissions.*").Create(&role).Error; err != nil {
			return err
		}
		s.roleIDs[role.Code] = role.ID
	}
	return nil
}

func (s *seeder) seedHospitals() error {
	s.hospitalIDs = make(map[string]uint, len(s.data.Hospitals))
	for _, h := range s.data.Hospitals {
		reg := s.fixed(h.RegisteredAt)
		m := models.Hospital{
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

		// Parameter KAPJ default per RS (nilai DEFAULT_KAPJ di mock).
		kapj := models.KAPJSetting{
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
	s.userIDs = map[string]uint{}
	s.userByName = map[string]uint{}
	s.userByEmail = map[string]uint{}
	for _, u := range s.data.Users {
		m := models.User{
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
	}
	return nil
}

// ── Master data ──────────────────────────────────────────────────────────────

var resumeCategories = []struct{ code, label string }{
	{models.ResumeRekamMedis, "Rekam Medis"},
	{models.ResumeHasilLab, "Hasil Lab"},
	{models.ResumeHasilRadiologi, "Hasil Radiologi"},
	{models.ResumeObat, "Obat"},
	{models.ResumeBMHP, "BMHP"},
	{models.ResumeBilling, "Billing"},
	{models.ResumePenunjangLainnya, "Penunjang Lainnya"},
}

func (s *seeder) seedMasters() error {
	s.insurerIDs = map[string]uint{}
	for _, i := range s.data.Insurers {
		m := models.Insurer{
			Name:           i.Name,
			Type:           models.InsurerType(i.Type),
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

	s.rejectionIDs = map[string]uint{}
	for _, r := range s.data.RejectionCodes {
		m := models.RejectionCode{Code: r.Code, Label: r.Label}
		if err := s.tx.Create(&m).Error; err != nil {
			return err
		}
		s.rejectionIDs[r.Code] = m.ID
	}

	s.ruleIDs = map[string]uint{}
	for _, r := range s.data.ScrubRules {
		m := models.ScrubRule{
			Code:          r.ID,
			Name:          r.Name,
			Category:      r.Category,
			Severity:      models.Severity(r.Severity),
			Message:       r.Message,
			FixSuggestion: r.FixSuggestion,
			SourceRef:     r.SourceRef,
			Enabled:       r.Enabled,
		}
		if err := s.tx.Create(&m).Error; err != nil {
			return err
		}
		s.ruleIDs[m.Code] = m.ID
	}

	s.categoryIDs = map[string]uint{}
	for _, c := range resumeCategories {
		m := models.ResumeCategory{Code: c.code, Label: c.label}
		if err := s.tx.Create(&m).Error; err != nil {
			return err
		}
		s.categoryIDs[m.Code] = m.ID
	}
	return nil
}

func (s *seeder) seedRegulations() error {
	for _, r := range s.data.Regulations {
		m := models.Regulation{
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
	s.partnerIDs = map[string]uint{}
	s.partnerByName = map[string]uint{}
	for _, p := range s.data.Partners {
		m := models.Partner{
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
			m.WhitelistIPs = append(m.WhitelistIPs, models.PartnerWhitelistIP{IP: ip})
		}
		for _, hid := range p.HospitalIDs {
			m.Hospitals = append(m.Hospitals, models.Hospital{
				BaseModel: models.BaseModel{ID: s.must(s.hospitalIDs, hid, "RS")},
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
	}
	return nil
}

// demoSecretHash membuat hash dari secret demo yang deterministik. Secret asli
// partner tidak ada di data mock (hanya versi tersamar), jadi ini sekadar pengisi.
func demoSecretHash(kind, consumerID string) string {
	sum := sha256.Sum256([]byte("demo-" + kind + "-" + consumerID))
	return hex.EncodeToString(sum[:])
}

// ── Klaim ────────────────────────────────────────────────────────────────────

func (s *seeder) seedBatches() error {
	s.batchIDs = map[string]uint{}
	for _, b := range s.data.Batches {
		m := models.Batch{
			BaseModel:   stamp(s.at(b.CreatedAt)),
			BatchNo:     b.ID,
			Name:        b.Name,
			Period:      b.Period,
			Note:        b.Note,
			HospitalID:  s.must(s.hospitalIDs, b.HospitalID, "RS"),
			CreatedByID: s.must(s.userByName, b.CreatedBy, "pengguna"),
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
	s.claimIDs = map[string]uint{}
	s.claimHospitals = map[string]uint{}
	for _, c := range s.data.Claims {
		m := models.Claim{
			BaseModel: models.BaseModel{
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
			SpecialCMG:          models.SpecialCMG(c.SpecialCMG),
			HospitalTariff:      models.HospitalTariff(c.HospitalTariff),
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
			Status:              models.ClaimStatus(c.Status),
			SubmittedAt:         s.atPtr(deref(c.SubmittedAt)),
			RejectionReasonNote: c.RejectionReasonNote,
			HospitalID:          s.must(s.hospitalIDs, c.HospitalID, "RS"),
			InsurerID:           opt(s.insurerIDs, c.InsurerID),
			BatchID:             opt(s.batchIDs, deref(c.BatchID)),
			RejectionCodeID:     opt(s.rejectionIDs, c.RejectionReasonCode),
			CreatedByID:         s.must(s.userByName, c.CreatedBy, "pengguna"),
		}
		for i, dx := range c.SecondaryDiagnoses {
			m.SecondaryDiagnoses = append(m.SecondaryDiagnoses, models.ClaimSecondaryDiagnosis{ICD10Code: dx, Seq: i + 1})
		}
		for i, px := range c.Procedures {
			m.Procedures = append(m.Procedures, models.ClaimProcedure{ICD9Code: px, Seq: i + 1})
		}
		for _, h := range c.StatusHistory {
			m.StatusHistories = append(m.StatusHistories, models.ClaimStatusHistory{
				BaseModel:  stamp(s.at(h.At)),
				FromStatus: models.ClaimStatus(h.From),
				ToStatus:   models.ClaimStatus(h.To),
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
	items := make([]models.ResumeItem, 0, len(s.data.ResumeItems))
	for _, r := range s.data.ResumeItems {
		items = append(items, models.ResumeItem{
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
	s.importIDs = map[string]uint{}
	s.lineByRef = map[string]uint{}
	for _, imp := range s.data.BankImports {
		importedAt := s.at(imp.ImportedAt)
		importer := s.importerOf(imp.ID)
		m := models.BankImport{
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
			line := models.BankLine{
				BaseModel:    stamp(importedAt),
				ValueDate:    s.at(bl.ValueDate),
				Description:  bl.Description,
				Amount:       bl.Amount,
				Reference:    bl.Reference,
				MatchResult:  models.MatchResult(bl.MatchResult),
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
// mock tidak menyimpan RS pada impor, dan seluruh data demonya milik RS pertama.
func (s *seeder) hospitalOfImport(importID string) uint {
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
func (s *seeder) importerOf(importID string) uint {
	for _, a := range s.data.AuditLog {
		if a.Action == "IMPORT_BANK_STATEMENT" && a.EntityID == importID {
			if id, ok := s.userByName[a.Actor]; ok {
				return id
			}
		}
	}
	for _, u := range s.data.Users {
		if u.Role == models.RoleFinance {
			return s.must(s.userIDs, u.ID, "pengguna")
		}
	}
	if s.err == nil {
		s.err = fmt.Errorf("tidak ada pengguna FINANCE untuk impor %s", importID)
	}
	return 0
}

func (s *seeder) seedPayments() error {
	for _, p := range s.data.Payments {
		m := models.Payment{
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
		m := models.WorkItem{
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
			// CreatedByID nil: work item demo dibuat otomatis oleh sistem.
		}
		for _, c := range w.Comments {
			m.Comments = append(m.Comments, models.WorkItemComment{
				BaseModel: stamp(s.at(c.At)),
				Body:      c.Text,
				AuthorID:  s.must(s.userByName, c.Author, "pengguna"),
			})
		}
		for _, h := range w.History {
			m.Histories = append(m.Histories, models.WorkItemHistory{
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

func (s *seeder) seedNotifications() error {
	for _, n := range s.data.Notifications {
		m := models.Notification{
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
	for _, a := range s.data.APIIntakes {
		at := s.at(a.At)
		m := models.APIIntake{
			BaseModel:     stamp(at),
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
	}
	return nil
}

func (s *seeder) seedAuditLogs() error {
	logs := make([]models.AuditLog, 0, len(s.data.AuditLog))
	for _, a := range s.data.AuditLog {
		scope := a.Scope
		if scope == "" {
			scope = models.AuditScopeWeb
		}
		logs = append(logs, models.AuditLog{
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
		})
	}
	if s.err != nil || len(logs) == 0 {
		return nil
	}
	return s.tx.CreateInBatches(&logs, 100).Error
}
