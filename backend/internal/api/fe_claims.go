package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"medpay_spkd/internal/audit"
	"medpay_spkd/internal/auth"
	"medpay_spkd/internal/engine"
	"medpay_spkd/internal/model"
)

const claimNotFound = "Klaim tidak ditemukan"

// ── Input klaim ──────────────────────────────────────────────────────────────

type claimIn struct {
	HospitalID         string     `json:"hospitalId"`
	SEPNo              string     `json:"sepNo"`
	BPJSCardNo         string     `json:"bpjsCardNo"`
	MRN                string     `json:"mrn"`
	PatientName        string     `json:"patientName"`
	DPJP               string     `json:"dpjp"`
	ParticipantType    string     `json:"participantType"`
	CareType           string     `json:"careType"`
	CareClass          string     `json:"careClass"`
	Sex                string     `json:"sex"`
	DateOfBirth        string     `json:"dateOfBirth"`
	AdmissionDate      string     `json:"admissionDate"`
	DischargeDate      string     `json:"dischargeDate"`
	PrimaryDiagnosis   string     `json:"primaryDiagnosis"`
	SecondaryDiagnoses []string   `json:"secondaryDiagnoses"`
	Procedures         []string   `json:"procedures"`
	INACBGCode         string     `json:"inacbgCode"`
	INACBGDescription  string     `json:"inacbgDescription"`
	INACBGBaseTariff   int64      `json:"inacbgBaseTariff"`
	SpecialCMG         cmgJSON    `json:"specialCmg"`
	HospitalTariff     tariffJSON `json:"hospitalTariff"`
	DeclarationChecked bool       `json:"declarationChecked"`
	InsurerID          string     `json:"insurerId"`
	PolicyNo           string     `json:"policyNo"`
	InsuredName        string     `json:"insuredName"`
	InsurerNameExtra   string     `json:"insurerNameExtra"`
	InsuranceType      string     `json:"insuranceType"`
	InsuranceMemberNo  string     `json:"insuranceMemberNo"`
	CoverageType       string     `json:"coverageType"`
	PolicyCeiling      *int64     `json:"policyCeiling"`
}

func oneOf(v string, allowed ...string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}

func nonNegative(vals ...int64) bool {
	for _, v := range vals {
		if v < 0 {
			return false
		}
	}
	return true
}

func normCodes(in []string) []string {
	out := []string{}
	for _, s := range in {
		if s = strings.ToUpper(strings.TrimSpace(s)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// validate memeriksa enum, panjang field, dan nominal. Aturan kelengkapan klinis
// sengaja diserahkan ke scrubber agar draft tetap bisa disimpan.
func (in *claimIn) validate() error {
	in.PrimaryDiagnosis = strings.ToUpper(strings.TrimSpace(in.PrimaryDiagnosis))
	in.SecondaryDiagnoses = normCodes(in.SecondaryDiagnoses)
	in.Procedures = normCodes(in.Procedures)
	in.INACBGCode = strings.TrimSpace(in.INACBGCode)

	if !oneOf(in.ParticipantType, model.ParticipantPBI, model.ParticipantNonPBI) {
		return errors.New("participantType harus PBI atau NON_PBI")
	}
	if !oneOf(in.CareType, model.CareTypeRawatInap, model.CareTypeRawatJalan) {
		return errors.New("careType harus RAWAT_INAP atau RAWAT_JALAN")
	}
	if !oneOf(in.CareClass, "I", "II", "III") {
		return errors.New("careClass harus I, II, atau III")
	}
	if !oneOf(in.Sex, "", "L", "P") {
		return errors.New("sex harus L atau P")
	}
	limits := []struct {
		name string
		v    string
		max  int
	}{
		{"sepNo", in.SEPNo, 30}, {"bpjsCardNo", in.BPJSCardNo, 30}, {"mrn", in.MRN, 30},
		{"patientName", in.PatientName, 150}, {"dpjp", in.DPJP, 150}, {"primaryDiagnosis", in.PrimaryDiagnosis, 10},
		{"inacbgCode", in.INACBGCode, 20}, {"inacbgDescription", in.INACBGDescription, 255},
		{"policyNo", in.PolicyNo, 50}, {"insuredName", in.InsuredName, 150},
		{"insurerNameExtra", in.InsurerNameExtra, 150}, {"insuranceType", in.InsuranceType, 50},
		{"insuranceMemberNo", in.InsuranceMemberNo, 50}, {"coverageType", in.CoverageType, 50},
	}
	for _, l := range limits {
		if len([]rune(l.v)) > l.max {
			return fmt.Errorf("%s maksimal %d karakter", l.name, l.max)
		}
	}
	if len(in.SecondaryDiagnoses) > 30 || len(in.Procedures) > 30 {
		return errors.New("maksimal 30 diagnosis sekunder / prosedur")
	}
	for _, code := range append(append([]string{}, in.SecondaryDiagnoses...), in.Procedures...) {
		if len(code) > 10 {
			return fmt.Errorf("kode %q terlalu panjang", code)
		}
	}
	t, cm := in.HospitalTariff, in.SpecialCMG
	if !nonNegative(in.INACBGBaseTariff, t.ProsedurNonBedah, t.ProsedurBedah, t.Konsultasi, t.TenagaAhli,
		t.Keperawatan, t.Penunjang, t.Radiologi, t.Laboratorium, t.PelayananDarah, t.Rehabilitasi,
		t.KamarAkomodasi, t.RawatIntensif, t.Obat, t.ObatKronis, t.ObatKemo, t.Alkes, t.BMHP, t.SewaAlat,
		cm.SpecialProcedure, cm.SpecialProsthesis, cm.SpecialInvestigation, cm.SpecialDrug, cm.SubAcute, cm.Chronic) {
		return errors.New("nominal tarif tidak boleh negatif")
	}
	if in.PolicyCeiling != nil && *in.PolicyCeiling < 0 {
		return errors.New("plafon polis tidak boleh negatif")
	}
	return nil
}

// apply menyalin input ke model (tanpa status, nomor, RS, dan jejak resolve tarif).
func (in *claimIn) apply(db *gorm.DB, c *model.Claim) error {
	var err error
	if c.DateOfBirth, err = parseDate(in.DateOfBirth); err != nil {
		return err
	}
	if c.AdmissionDate, err = parseDate(in.AdmissionDate); err != nil {
		return err
	}
	if c.DischargeDate, err = parseDate(in.DischargeDate); err != nil {
		return err
	}
	insurerID, valid := parseOptID(in.InsurerID)
	if !valid {
		return errors.New("insurerId tidak valid")
	}
	if insurerID != nil {
		var n int64
		if err := db.Model(&model.Insurer{}).Where("id = ?", *insurerID).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			return errors.New("penjamin tidak dikenal")
		}
	}

	c.SEPNo, c.BPJSCardNo, c.MRN = strings.TrimSpace(in.SEPNo), strings.TrimSpace(in.BPJSCardNo), strings.TrimSpace(in.MRN)
	c.PatientName, c.DPJP = strings.TrimSpace(in.PatientName), strings.TrimSpace(in.DPJP)
	c.ParticipantType, c.CareType, c.CareClass, c.Sex = in.ParticipantType, in.CareType, in.CareClass, in.Sex
	c.LOS = engine.LOS(c.AdmissionDate, c.DischargeDate)
	c.PrimaryDiagnosis = in.PrimaryDiagnosis
	c.INACBGCode, c.INACBGDescription, c.INACBGBaseTariff = in.INACBGCode, in.INACBGDescription, in.INACBGBaseTariff
	c.SpecialCMG = model.SpecialCMG(in.SpecialCMG)
	c.HospitalTariff = model.HospitalTariff(in.HospitalTariff)
	c.DeclarationChecked = in.DeclarationChecked
	c.InsurerID = insurerID
	c.Insurer = nil
	c.PolicyNo, c.InsuredName, c.InsurerNameExtra = in.PolicyNo, in.InsuredName, in.InsurerNameExtra
	c.InsuranceType, c.InsuranceMemberNo, c.CoverageType = in.InsuranceType, in.InsuranceMemberNo, in.CoverageType
	c.PolicyCeiling = in.PolicyCeiling
	if c.PolicyCeiling != nil && *c.PolicyCeiling == 0 {
		c.PolicyCeiling = nil // 0 = tidak terbatas
	}

	c.SecondaryDiagnoses = nil
	for i, code := range in.SecondaryDiagnoses {
		c.SecondaryDiagnoses = append(c.SecondaryDiagnoses, model.ClaimSecondaryDiagnosis{ICD10Code: code, Seq: i + 1})
	}
	c.Procedures = nil
	for i, code := range in.Procedures {
		c.Procedures = append(c.Procedures, model.ClaimProcedure{ICD9Code: code, Seq: i + 1})
	}
	return nil
}

// resolveOnSave menjalankan resolusi tarif di server saat klaim disimpan (F-TRF-04):
// nilai INA-CBG dari form tidak dipercaya begitu saja.
//   - Pengguna berizin RESOLVE yang mengisi kode INA-CBG: grouper dijalankan ulang di
//     server dan hasilnya (kode, deskripsi, tarif) menggantikan nilai dari form.
//   - Selain itu, kode/tarif yang berubah dianggap belum di-resolve (R06 muncul).
func resolveOnSave(cl *model.Claim, hospital model.Hospital, u model.User, oldCode string, oldTariff int64) {
	changed := cl.INACBGCode != oldCode || cl.INACBGBaseTariff != oldTariff
	if cl.INACBGCode != "" && auth.HasPermission(u, model.PermResolve) {
		g := engine.DemoGrouper(cl.PrimaryDiagnosis, cl.CareType, cl.CareClass, cl.LOS, hospital.EklaimConfigured)
		if g.OK {
			now := time.Now()
			cl.INACBGCode, cl.INACBGDescription, cl.INACBGBaseTariff = g.Code, g.Description, g.BaseTariff
			cl.TariffResolvedAt = &now
			cl.ResolutionSignature = engine.TariffSignature(cl)
			return
		}
		changed = true
	}
	if changed {
		cl.TariffResolvedAt = nil
		cl.ResolutionSignature = ""
	}
}

// grouperPreviewHandler menjalankan grouper di server untuk form klaim (tanpa menyimpan).
func grouperPreviewHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		var body struct {
			HospitalID       string `json:"hospitalId"`
			PrimaryDiagnosis string `json:"primaryDiagnosis"`
			CareType         string `json:"careType"`
			CareClass        string `json:"careClass"`
			AdmissionDate    string `json:"admissionDate"`
			DischargeDate    string `json:"dischargeDate"`
		}
		if !bind(c, &body) {
			return
		}
		hid, valid := resolveHospital(c, db, u, body.HospitalID)
		if !valid {
			return
		}
		var hospital model.Hospital
		if err := db.First(&hospital, "id = ?", hid).Error; err != nil {
			serverError(c, err)
			return
		}
		adm, err1 := parseDate(body.AdmissionDate)
		dis, err2 := parseDate(body.DischargeDate)
		if err1 != nil || err2 != nil {
			fail(c, http.StatusBadRequest, "Tanggal masuk/pulang wajib berformat YYYY-MM-DD")
			return
		}
		los := engine.LOS(adm, dis)
		g := engine.DemoGrouper(strings.ToUpper(strings.TrimSpace(body.PrimaryDiagnosis)), body.CareType, body.CareClass, los, hospital.EklaimConfigured)
		if !g.OK {
			_ = audit.Log(db, &u, audit.Entry{Action: "GROUPER_PREVIEW_FAILED", Entity: "hospital", EntityID: idStr(hid),
				After: g.Error, HospitalID: &hid, StatusCode: http.StatusUnprocessableEntity}, metaOf(c))
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, J{"error": "Grouper gagal: " + g.Error, "grouper": g})
			return
		}
		ok(c, J{"grouper": g, "los": los})
	}
}

// reloadClaim memuat ulang klaim beserta relasi untuk respons.
func reloadClaim(db *gorm.DB, id uuid.UUID) (*model.Claim, error) {
	var c model.Claim
	err := claimPreloads(db).Preload("StatusHistories.Actor").First(&c, "id = ?", id).Error
	return &c, err
}

func respondClaim(c *gin.Context, db *gorm.DB, id uuid.UUID, status int) {
	cl, err := reloadClaim(db, id)
	if err != nil {
		serverError(c, err)
		return
	}
	sr, err := scrubClaim(db, cl)
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(status, claimDTO(cl, &sr))
}

func claimSummary(c *model.Claim) string {
	b, _ := json.Marshal(J{"status": c.Status, "patient": c.PatientName})
	return string(b)
}

// ── GET /claims ──────────────────────────────────────────────────────────────

func claimListHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, okScope := scopeOf(c)
		if !okScope {
			return
		}
		d, err := loadDataset(db, scope)
		if err != nil {
			serverError(c, err)
			return
		}
		status, careType, batch := c.Query("status"), c.Query("careType"), c.Query("batchId")
		search := strings.ToLower(strings.TrimSpace(c.Query("q")))
		page, _ := strconv.Atoi(c.Query("page"))
		if page < 1 {
			page = 1
		}
		size, _ := strconv.Atoi(c.Query("pageSize"))
		if size < 1 || size > 200 {
			size = 15
		}

		var matched []*model.Claim
		for _, cl := range d.claims {
			batchID := ""
			if cl.BatchID != nil {
				batchID = idStr(*cl.BatchID)
			}
			switch {
			case status != "" && string(cl.Status) != status,
				careType != "" && cl.CareType != careType,
				batch == "none" && batchID != "",
				batch != "" && batch != "none" && batchID != batch,
				search != "" && !strings.Contains(strings.ToLower(cl.ClaimNo+" "+cl.SEPNo+" "+cl.PatientName+" "+cl.MRN), search):
				continue
			}
			matched = append(matched, cl)
		}
		total := len(matched)
		start := min((page-1)*size, total)
		end := min(start+size, total)
		rows := make([]claimJSON, 0, end-start)
		for _, cl := range matched[start:end] {
			sr := d.scrub(cl)
			rows = append(rows, claimDTO(cl, &sr))
		}
		ok(c, J{"rows": rows, "total": total, "page": page, "pageSize": size})
	}
}

// ── GET /claims/:id ──────────────────────────────────────────────────────────

func claimGetHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, err := findClaim(db, currentUser(c), c.Param("id"), true)
		if err != nil {
			notFoundOr500(c, err, claimNotFound)
			return
		}
		sr, err := scrubClaim(db, cl)
		if err != nil {
			serverError(c, err)
			return
		}
		var items []model.ResumeItem
		if err := db.Preload("ResumeCategory").Where("claim_id = ?", cl.ID).Order("id").Find(&items).Error; err != nil {
			serverError(c, err)
			return
		}
		var pays []model.Payment
		if err := db.Preload("CreatedBy").Where("claim_id = ?", cl.ID).Order("received_date, id").Find(&pays).Error; err != nil {
			serverError(c, err)
			return
		}
		resume, payments := []J{}, []J{}
		var paid int64
		for _, it := range items {
			resume = append(resume, resumeDTO(it, cl.ClaimNo))
		}
		for _, p := range pays {
			payments = append(payments, paymentDTO(p, cl.ClaimNo))
			paid += p.Amount
		}
		gap := engine.Gap(cl)
		var hospital, insurer any
		if cl.Hospital != nil {
			hospital = hospitalDTO(*cl.Hospital)
		}
		if cl.Insurer != nil {
			insurer = insurerDTO(*cl.Insurer)
		}
		ok(c, J{"claim": claimDTO(cl, &sr), "transitions": claimTransitionsDTO(db, currentUser(c), cl),
			"editable": engine.Editable(cl.Status), "deletable": cl.Status == model.ClaimDraft && paid == 0,
			"resume": resume, "payments": payments, "gap": gap,
			"outstanding": max(gap-paid, 0), "hospital": hospital, "insurer": insurer})
	}
}

// ── POST /claims ─────────────────────────────────────────────────────────────

func claimCreateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		var in claimIn
		if !bind(c, &in) {
			return
		}
		if err := in.validate(); err != nil {
			fail(c, http.StatusBadRequest, err.Error())
			return
		}
		hid, valid := resolveHospital(c, db, u, in.HospitalID)
		if !valid {
			return
		}
		var hospital model.Hospital
		if err := db.First(&hospital, "id = ?", hid).Error; err != nil {
			serverError(c, err)
			return
		}

		cl := model.Claim{HospitalID: hospital.ID, Status: model.ClaimDraft, CreatedByID: u.ID}
		if err := in.apply(db, &cl); err != nil {
			fail(c, http.StatusBadRequest, err.Error())
			return
		}
		resolveOnSave(&cl, hospital, u, "", 0)
		err := db.Transaction(func(tx *gorm.DB) error {
			no, err := nextNo(tx, "claims", "claim_no", "CLM")
			if err != nil {
				return err
			}
			cl.ClaimNo = no
			cl.StatusHistories = []model.ClaimStatusHistory{{ToStatus: model.ClaimDraft, Note: "Klaim dibuat", ActorID: &u.ID}}
			if err := tx.Create(&cl).Error; err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "CREATE_CLAIM", Entity: "claim", EntityID: cl.ClaimNo,
				After: cl.PatientName, HospitalID: &cl.HospitalID}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		respondClaim(c, db, cl.ID, http.StatusCreated)
	}
}

// ── PUT /claims/:id ──────────────────────────────────────────────────────────

func claimUpdateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		cl, err := findClaim(db, u, c.Param("id"), false)
		if err != nil {
			notFoundOr500(c, err, claimNotFound)
			return
		}
		if !engine.Editable(cl.Status) {
			fail(c, http.StatusConflict, "Klaim dengan status "+claimLabel(db, cl.Status)+" tidak dapat diubah")
			return
		}
		var in claimIn
		if !bind(c, &in) {
			return
		}
		if err := in.validate(); err != nil {
			fail(c, http.StatusBadRequest, err.Error())
			return
		}
		before := claimSummary(cl)
		oldCode, oldTariff := cl.INACBGCode, cl.INACBGBaseTariff
		if err := in.apply(db, cl); err != nil {
			fail(c, http.StatusBadRequest, err.Error())
			return
		}
		var hospital model.Hospital
		if err := db.First(&hospital, "id = ?", cl.HospitalID).Error; err != nil {
			serverError(c, err)
			return
		}
		resolveOnSave(cl, hospital, u, oldCode, oldTariff)

		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("claim_id = ?", cl.ID).Delete(&model.ClaimSecondaryDiagnosis{}).Error; err != nil {
				return err
			}
			if err := tx.Where("claim_id = ?", cl.ID).Delete(&model.ClaimProcedure{}).Error; err != nil {
				return err
			}
			// Omit relasi milik tabel lain; diagnosis & prosedur baru ikut tersimpan.
			if err := tx.Omit("Batch", "Insurer", "RejectionCode", "CreatedBy", "Hospital").Save(cl).Error; err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "UPDATE_CLAIM", Entity: "claim", EntityID: cl.ClaimNo,
				Before: before, After: claimSummary(cl), HospitalID: &cl.HospitalID}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		respondClaim(c, db, cl.ID, http.StatusOK)
	}
}

// ── DELETE /claims/:id ───────────────────────────────────────────────────────

func claimDeleteHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		cl, err := findClaim(db, u, c.Param("id"), false)
		if err != nil {
			notFoundOr500(c, err, claimNotFound)
			return
		}
		if cl.Status != model.ClaimDraft {
			fail(c, http.StatusConflict, "Hanya klaim berstatus Draft yang dapat dihapus (F-CLM-09). Gunakan status Dihapus Buku untuk klaim lain.")
			return
		}
		paid, err := paidAmount(db, cl.ID)
		if err != nil {
			serverError(c, err)
			return
		}
		if paid > 0 {
			fail(c, http.StatusConflict, "Klaim sudah memiliki pembayaran dan tidak dapat dihapus")
			return
		}
		err = db.Transaction(func(tx *gorm.DB) error {
			res := tx.Where("claim_id = ?", cl.ID).Delete(&model.ResumeItem{})
			if res.Error != nil {
				return res.Error
			}
			if err := tx.Model(cl).Update("batch_id", nil).Error; err != nil {
				return err
			}
			if err := tx.Delete(cl).Error; err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "DELETE_CLAIM", Entity: "claim", EntityID: cl.ClaimNo,
				Before: cl.PatientName, After: fmt.Sprintf("menghapus %d item resume", res.RowsAffected),
				HospitalID: &cl.HospitalID}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		ok(c, J{"ok": true})
	}
}

// ── POST /claims/:id/resolve ─────────────────────────────────────────────────

func claimResolveHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		cl, err := findClaim(db, u, c.Param("id"), false)
		if err != nil {
			notFoundOr500(c, err, claimNotFound)
			return
		}
		if !engine.Editable(cl.Status) {
			fail(c, http.StatusConflict, "Tarif klaim berstatus "+claimLabel(db, cl.Status)+" tidak dapat di-resolve ulang")
			return
		}
		var hospital model.Hospital
		if err := db.First(&hospital, "id = ?", cl.HospitalID).Error; err != nil {
			serverError(c, err)
			return
		}
		g := engine.DemoGrouper(cl.PrimaryDiagnosis, cl.CareType, cl.CareClass, cl.LOS, hospital.EklaimConfigured)
		if !g.OK {
			_ = audit.Log(db, &u, audit.Entry{Action: "RESOLVE_TARIFF_FAILED", Entity: "claim", EntityID: cl.ClaimNo,
				After: g.Error, HospitalID: &cl.HospitalID, StatusCode: http.StatusUnprocessableEntity}, metaOf(c))
			fail(c, http.StatusUnprocessableEntity, "Grouper gagal: "+g.Error)
			return
		}
		now := time.Now()
		cl.INACBGCode, cl.INACBGDescription, cl.INACBGBaseTariff = g.Code, g.Description, g.BaseTariff
		cl.TariffResolvedAt = &now
		cl.ResolutionSignature = engine.TariffSignature(cl)
		err = db.Transaction(func(tx *gorm.DB) error {
			err := tx.Model(cl).Updates(map[string]any{
				"inacbg_code": cl.INACBGCode, "inacbg_description": cl.INACBGDescription,
				"inacbg_base_tariff": cl.INACBGBaseTariff, "tariff_resolved_at": now,
				"resolution_signature": cl.ResolutionSignature,
			}).Error
			if err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "RESOLVE_TARIFF", Entity: "claim", EntityID: cl.ClaimNo,
				After: fmt.Sprintf("%s @ %d", g.Code, g.BaseTariff), HospitalID: &cl.HospitalID}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		full, err := reloadClaim(db, cl.ID)
		if err != nil {
			serverError(c, err)
			return
		}
		sr, err := scrubClaim(db, full)
		if err != nil {
			serverError(c, err)
			return
		}
		ok(c, J{"claim": claimDTO(full, &sr), "grouper": g})
	}
}

// ── POST /claims/:id/transition ──────────────────────────────────────────────

func claimTransitionHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		var body struct {
			To            string `json:"to"`
			Note          string `json:"note"`
			RejectionCode string `json:"rejectionCode"`
		}
		if !bind(c, &body) {
			return
		}
		cl, err := findClaim(db, u, c.Param("id"), false)
		if err != nil {
			notFoundOr500(c, err, claimNotFound)
			return
		}
		to := model.ClaimStatus(body.To)
		if to == model.ClaimPaid || to == model.ClaimPartiallyPaid {
			fail(c, http.StatusConflict, "Status Lunas / Sebagian Dibayar dihitung otomatis dari pencatatan pembayaran")
			return
		}
		if need := engine.TransitionPermission(to); !auth.HasPermission(u, need) {
			msg := fmt.Sprintf("Peran Anda (%s) tidak memiliki izin %s untuk status %s", auth.RoleCode(u), need, claimLabel(db, to))
			if to == model.ClaimSubmitted {
				msg = "Pengajuan klaim memerlukan peran Reviewer/Finance/Admin (maker-checker)"
			}
			denyAccess(c, db, u, "claim", c.Param("id"), msg)
			return
		}
		note := strings.TrimSpace(body.Note)
		if len(note) > 1000 {
			fail(c, http.StatusBadRequest, "Catatan maksimal 1000 karakter")
			return
		}
		if !engine.CanTransition(cl.Status, to) {
			fail(c, http.StatusConflict, fmt.Sprintf("Transisi tidak sah: %s → %s", cl.Status, body.To))
			return
		}

		updates := map[string]any{}
		switch to {
		case model.ClaimReadyToSubmit:
			sr, err := scrubClaim(db, cl)
			if err != nil {
				serverError(c, err)
				return
			}
			if sr.HasOpenBlocker {
				fail(c, http.StatusConflict, "Klaim masih memiliki blocker terbuka ("+strings.Join(sr.OpenBlockers(), ", ")+
					") — selesaikan dulu atau minta override Admin RS")
				return
			}
			if !cl.DeclarationChecked {
				fail(c, http.StatusConflict, "Pernyataan kebenaran data belum dicentang")
				return
			}
		case model.ClaimSubmitted:
			if cl.CreatedByID == u.ID {
				fail(c, http.StatusForbidden, "Maker-checker: pembuat klaim tidak boleh mengajukan klaimnya sendiri")
				return
			}
			sr, err := scrubClaim(db, cl)
			if err != nil {
				serverError(c, err)
				return
			}
			if sr.HasOpenBlocker {
				fail(c, http.StatusConflict, "Klaim masih memiliki blocker terbuka — tidak dapat diajukan")
				return
			}
			updates["submitted_at"] = time.Now()
		case model.ClaimRejected:
			if body.RejectionCode == "" {
				fail(c, http.StatusBadRequest, "Alasan penolakan terstruktur wajib diisi")
				return
			}
			var rc model.RejectionCode
			if err := db.Where("code = ?", body.RejectionCode).First(&rc).Error; err != nil {
				notFoundOr500(c, err, "Kode alasan penolakan tidak dikenal")
				return
			}
			updates["rejection_code_id"] = rc.ID
			updates["rejection_reason_note"] = note
		}

		from := cl.Status
		err = db.Transaction(func(tx *gorm.DB) error {
			if len(updates) > 0 {
				if err := tx.Model(cl).Updates(updates).Error; err != nil {
					return err
				}
			}
			if err := setClaimStatus(tx, cl, to, &u, note); err != nil {
				return err
			}
			if to == model.ClaimRejected {
				// F-WQ-04: klaim ditolak → tugas tindak lanjut (sanggah / hapus buku).
				title := fmt.Sprintf("Klaim %s (%s) ditolak penjamin (%s) — putuskan sanggah atau hapus buku",
					cl.ClaimNo, cl.PatientName, body.RejectionCode)
				if err := autoTask(tx, cl.HospitalID, &cl.ID, "REJECTION", model.PriorityTinggi, title, engine.Today().AddDate(0, 0, 5)); err != nil {
					return err
				}
			}
			return audit.Log(tx, &u, audit.Entry{Action: "STATUS_TRANSITION", Entity: "claim", EntityID: cl.ClaimNo,
				Before: string(from), After: string(to) + " — " + note, HospitalID: &cl.HospitalID}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		respondClaim(c, db, cl.ID, http.StatusOK)
	}
}

// ── Scrubber ─────────────────────────────────────────────────────────────────

func claimScrubHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, err := findClaim(db, currentUser(c), c.Param("id"), false)
		if err != nil {
			notFoundOr500(c, err, claimNotFound)
			return
		}
		sr, err := scrubClaim(db, cl)
		if err != nil {
			serverError(c, err)
			return
		}
		ok(c, sr)
	}
}

func scrubTopCausesHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, okScope := scopeOf(c)
		if !okScope {
			return
		}
		d, err := loadDataset(db, scope)
		if err != nil {
			serverError(c, err)
			return
		}
		counts := map[string]int{}
		for _, cl := range d.claims {
			for _, f := range d.scrub(cl).Findings {
				if !f.Overridden {
					counts[f.RuleID]++
				}
			}
		}
		out := []J{}
		for id, n := range counts {
			r := d.rules[id]
			out = append(out, J{"ruleId": id, "count": n, "name": r.Name, "severity": string(r.Severity), "category": r.Category})
		}
		sort.Slice(out, func(i, j int) bool {
			ci, cj := out[i]["count"].(int), out[j]["count"].(int)
			if ci != cj {
				return ci > cj
			}
			return out[i]["ruleId"].(string) < out[j]["ruleId"].(string)
		})
		ok(c, out)
	}
}

func scrubRuleUpdateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if !auth.IsPlatformStaff(u) {
			fail(c, http.StatusForbidden, "Aturan scrub berlaku untuk semua RS; hanya staf SPKD yang dapat mengubahnya")
			return
		}
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if !bind(c, &body) {
			return
		}
		var r model.ScrubRule
		rid, _ := parseID(c.Param("id"))
		if err := db.Where("id = ?", rid).First(&r).Error; err != nil {
			notFoundOr500(c, err, "Aturan tidak ditemukan")
			return
		}
		before := r.Enabled
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&r).Update("enabled", body.Enabled).Error; err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "SCRUB_RULE_TOGGLE", Entity: "scrub_rule", EntityID: r.Code,
				Before: strconv.FormatBool(before), After: strconv.FormatBool(body.Enabled)}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		r.Enabled = body.Enabled
		ok(c, scrubRuleDTO(r))
	}
}

func scrubOverrideHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		var body struct {
			ClaimID string `json:"claimId"`
			RuleID  string `json:"ruleId"`
			Reason  string `json:"reason"`
			Remove  bool   `json:"remove"`
		}
		if !bind(c, &body) {
			return
		}
		reason := strings.TrimSpace(body.Reason)
		if !body.Remove && reason == "" {
			fail(c, http.StatusBadRequest, "Alasan override wajib diisi (tercatat di audit trail)")
			return
		}
		if len(reason) > 1000 {
			fail(c, http.StatusBadRequest, "Alasan maksimal 1000 karakter")
			return
		}
		cl, err := findClaim(db, u, body.ClaimID, false)
		if err != nil {
			notFoundOr500(c, err, claimNotFound)
			return
		}
		var rule model.ScrubRule
		if err := db.Where("code = ?", body.RuleID).First(&rule).Error; err != nil {
			notFoundOr500(c, err, "Aturan tidak ditemukan")
			return
		}
		key := cl.ClaimNo + "|" + rule.Code
		err = db.Transaction(func(tx *gorm.DB) error {
			action := "SCRUB_OVERRIDE"
			if body.Remove {
				action = "SCRUB_OVERRIDE_REMOVED"
				if err := tx.Where("claim_id = ? AND scrub_rule_id = ?", cl.ID, rule.ID).Delete(&model.ClaimScrubOverride{}).Error; err != nil {
					return err
				}
			} else {
				var ov model.ClaimScrubOverride
				err := tx.Where("claim_id = ? AND scrub_rule_id = ?", cl.ID, rule.ID).First(&ov).Error
				switch {
				case err == nil:
					if err := tx.Model(&ov).Updates(map[string]any{"reason": reason, "overridden_by_id": u.ID}).Error; err != nil {
						return err
					}
				case errors.Is(err, gorm.ErrRecordNotFound):
					ov = model.ClaimScrubOverride{ClaimID: cl.ID, ScrubRuleID: rule.ID, Reason: reason, OverriddenByID: u.ID}
					if err := tx.Create(&ov).Error; err != nil {
						return err
					}
				default:
					return err
				}
			}
			return audit.Log(tx, &u, audit.Entry{Action: action, Entity: "claim|rule", EntityID: key,
				After: reason, HospitalID: &cl.HospitalID}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		sr, err := scrubClaim(db, cl)
		if err != nil {
			serverError(c, err)
			return
		}
		ok(c, sr)
	}
}

// ── Resume medis ─────────────────────────────────────────────────────────────

type resumeIn struct {
	SEP          string  `json:"sep"`
	Kategori     string  `json:"kategori"`
	NamaItem     string  `json:"namaItem"`
	SubKategori  string  `json:"subKategori"`
	Tanggal      string  `json:"tanggal"`
	HasilCatatan string  `json:"hasilCatatan"`
	Jumlah       float64 `json:"jumlah"`
	Satuan       string  `json:"satuan"`
	HargaSatuan  int64   `json:"hargaSatuan"`
	Subtotal     int64   `json:"subtotal"`
	Reviewed     bool    `json:"reviewed"`
}

func resumeCategoryIDs(db *gorm.DB) (map[string]uuid.UUID, error) {
	var cats []model.ResumeCategory
	if err := db.Find(&cats).Error; err != nil {
		return nil, err
	}
	out := map[string]uuid.UUID{}
	for _, c := range cats {
		out[c.Code] = c.ID
	}
	return out, nil
}

// apply menyalin input ke item resume. Subtotal 0 dihitung dari jumlah × harga satuan.
func (in resumeIn) apply(it *model.ResumeItem, cats map[string]uuid.UUID) error {
	catID, found := cats[in.Kategori]
	if !found {
		return fmt.Errorf("kategori %q tidak dikenal", in.Kategori)
	}
	name := strings.TrimSpace(in.NamaItem)
	if name == "" || len(name) > 255 || len(in.SubKategori) > 100 || len(in.Satuan) > 30 {
		return errors.New("nama item wajib diisi (maks. 255), sub kategori maks. 100, satuan maks. 30 karakter")
	}
	if in.Jumlah < 0 || in.HargaSatuan < 0 || in.Subtotal < 0 || in.Jumlah > 1e7 {
		return errors.New("jumlah, harga satuan, dan subtotal tidak boleh negatif")
	}
	date, err := parseDate(in.Tanggal)
	if err != nil {
		return err
	}
	it.ItemName, it.SubCategory, it.ServiceDate, it.ResultNote = name, in.SubKategori, date, in.HasilCatatan
	it.Quantity, it.Unit, it.UnitPrice, it.Subtotal, it.Reviewed = in.Jumlah, in.Satuan, in.HargaSatuan, in.Subtotal, in.Reviewed
	it.ResumeCategoryID = catID
	if it.Subtotal == 0 {
		it.Subtotal = int64(math.Round(in.Jumlah * float64(in.HargaSatuan)))
	}
	return nil
}

func resumeBulkHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		var body struct {
			ClaimID string     `json:"claimId"`
			Items   []resumeIn `json:"items"`
		}
		if !bind(c, &body) {
			return
		}
		if len(body.Items) > 1000 {
			fail(c, http.StatusBadRequest, "Maksimal 1000 item per unggahan")
			return
		}
		cl, err := findClaim(db, u, body.ClaimID, false)
		if err != nil {
			notFoundOr500(c, err, claimNotFound)
			return
		}
		cats, err := resumeCategoryIDs(db)
		if err != nil {
			serverError(c, err)
			return
		}
		var items []model.ResumeItem
		skipped := 0
		for _, in := range body.Items {
			if in.SEP != "" && in.SEP != cl.SEPNo {
				skipped++ // item milik SEP lain
				continue
			}
			it := model.ResumeItem{ClaimID: cl.ID, SEPNo: cl.SEPNo}
			if err := in.apply(&it, cats); err != nil {
				skipped++
				continue
			}
			items = append(items, it)
		}
		err = db.Transaction(func(tx *gorm.DB) error {
			if len(items) > 0 {
				if err := tx.CreateInBatches(&items, 200).Error; err != nil {
					return err
				}
			}
			return audit.Log(tx, &u, audit.Entry{Action: "RESUME_UPLOAD", Entity: "claim", EntityID: cl.ClaimNo,
				After: fmt.Sprintf("%d item (skip %d)", len(items), skipped), HospitalID: &cl.HospitalID}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		ok(c, J{"added": len(items), "skipped": skipped})
	}
}

// findResumeItem mencari item resume dalam cakupan RS user.
func findResumeItem(db *gorm.DB, u model.User, id string) (*model.ResumeItem, error) {
	rid, valid := parseID(id)
	if !valid {
		return nil, gorm.ErrRecordNotFound
	}
	var it model.ResumeItem
	if err := db.Preload("Claim").First(&it, "id = ?", rid).Error; err != nil {
		return nil, err
	}
	if it.Claim == nil || !auth.CanAccessHospital(u, it.Claim.HospitalID) {
		return nil, gorm.ErrRecordNotFound
	}
	return &it, nil
}

func resumeUpdateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		it, err := findResumeItem(db, u, c.Param("id"))
		if err != nil {
			notFoundOr500(c, err, "Item tidak ditemukan")
			return
		}
		var in resumeIn
		if !bind(c, &in) {
			return
		}
		cats, err := resumeCategoryIDs(db)
		if err != nil {
			serverError(c, err)
			return
		}
		if err := in.apply(it, cats); err != nil {
			fail(c, http.StatusBadRequest, err.Error())
			return
		}
		claim := it.Claim
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Omit("Claim", "ResumeCategory").Save(it).Error; err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "RESUME_ITEM_EDIT", Entity: "resume_item", EntityID: idStr(it.ID),
				After: it.ItemName, HospitalID: &claim.HospitalID}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		if err := db.Preload("ResumeCategory").First(it, "id = ?", it.ID).Error; err != nil {
			serverError(c, err)
			return
		}
		ok(c, resumeDTO(*it, claim.ClaimNo))
	}
}

func resumeDeleteHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		it, err := findResumeItem(db, u, c.Param("id"))
		if err != nil {
			notFoundOr500(c, err, "Item tidak ditemukan")
			return
		}
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Delete(&model.ResumeItem{}, "id = ?", it.ID).Error; err != nil {
				return err
			}
			return audit.Log(tx, &u, audit.Entry{Action: "RESUME_ITEM_DELETE", Entity: "resume_item", EntityID: idStr(it.ID),
				Before: it.ItemName, HospitalID: &it.Claim.HospitalID}, metaOf(c))
		})
		if err != nil {
			serverError(c, err)
			return
		}
		ok(c, J{"ok": true})
	}
}
