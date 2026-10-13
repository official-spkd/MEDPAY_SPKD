package seed

import "medpay_spkd/internal/model"

// Data master yang WAJIB ada di setiap lingkungan (termasuk produksi). Diisi oleh
// Master() setiap server start: baris yang belum ada ditambahkan, baris yang sudah
// ada (termasuk yang sudah diubah admin) tidak ditimpa.
//
// Kode yang dipakai logika (status klaim, jenis rawat, kode aturan R01–R14, dst.)
// harus sama dengan konstanta di package model / engine.

type seedRef struct {
	Code, Label, Tone, Description string
}

var seedRefCodes = []struct {
	Category string
	Items    []seedRef
}{
	{model.RefClaimStatus, []seedRef{
		{Code: "draft", Label: "Draft", Tone: "neutral", Description: "Klaim baru, masih dapat diubah"},
		{Code: "ready_to_submit", Label: "Siap Diajukan", Tone: "teal", Description: "Lolos validasi, menunggu pengajuan (checker)"},
		{Code: "submitted", Label: "Diajukan", Tone: "info", Description: "Sudah dikirim ke penjamin"},
		{Code: "approved", Label: "Disetujui", Tone: "success", Description: "Disetujui penjamin, menunggu pembayaran"},
		{Code: "partially_approved", Label: "Disetujui Sebagian", Tone: "warning", Description: "Disetujui sebagian oleh penjamin"},
		{Code: "rejected", Label: "Ditolak", Tone: "danger", Description: "Ditolak penjamin dengan kode alasan"},
		{Code: "partially_paid", Label: "Sebagian Dibayar", Tone: "warning", Description: "Dihitung otomatis dari pembayaran"},
		{Code: "paid", Label: "Lunas", Tone: "success", Description: "Dihitung otomatis dari pembayaran"},
		{Code: "disputed", Label: "Disanggah", Tone: "danger", Description: "Banding atas penolakan; dapat diedit & diajukan ulang"},
		{Code: "written_off", Label: "Dihapus Buku", Tone: "neutral", Description: "Status final"},
	}},
	{model.RefClaimAction, []seedRef{
		{Code: "ready_to_submit", Label: "Siapkan Pengajuan", Tone: "outline"},
		{Code: "draft", Label: "Kembalikan ke Draft", Tone: "outline"},
		{Code: "submitted", Label: "Ajukan", Tone: "teal", Description: "Tindakan tercatat di audit trail. Memerlukan peran Reviewer/Finance/Admin (maker-checker)."},
		{Code: "approved", Label: "Setujui", Tone: "outline"},
		{Code: "partially_approved", Label: "Setujui Sebagian", Tone: "outline"},
		{Code: "rejected", Label: "Tandai Ditolak", Tone: "danger", Description: "Pilih kode alasan penolakan terstruktur untuk analitik."},
		{Code: "disputed", Label: "Disanggah", Tone: "outline"},
		{Code: "written_off", Label: "Hapus Buku", Tone: "danger", Description: "Status berubah menjadi Dihapus Buku dan tidak dapat dikembalikan."},
	}},
	{model.RefTariffComponent, []seedRef{
		{Code: "prosedurNonBedah", Label: "Prosedur Non Bedah"}, {Code: "prosedurBedah", Label: "Prosedur Bedah"},
		{Code: "konsultasi", Label: "Konsultasi"}, {Code: "tenagaAhli", Label: "Tenaga Ahli"},
		{Code: "keperawatan", Label: "Keperawatan"}, {Code: "penunjang", Label: "Penunjang"},
		{Code: "radiologi", Label: "Radiologi"}, {Code: "laboratorium", Label: "Laboratorium"},
		{Code: "pelayananDarah", Label: "Pelayanan Darah"}, {Code: "rehabilitasi", Label: "Rehabilitasi"},
		{Code: "kamarAkomodasi", Label: "Kamar Akomodasi"}, {Code: "rawatIntensif", Label: "Rawat Intensif"},
		{Code: "obat", Label: "Obat"}, {Code: "obatKronis", Label: "Obat Kronis"},
		{Code: "obatKemo", Label: "Obat Kemo"}, {Code: "alkes", Label: "Alkes"},
		{Code: "bmhp", Label: "BMHP"}, {Code: "sewaAlat", Label: "Sewa Alat"},
	}},
	{model.RefSpecialCMG, []seedRef{
		{Code: "specialProcedure", Label: "Special Procedure (SP)"}, {Code: "specialProsthesis", Label: "Special Prosthesis (SR)"},
		{Code: "specialInvestigation", Label: "Special Investigation (SI)"}, {Code: "specialDrug", Label: "Special Drug (SD)"},
		{Code: "subAcute", Label: "Sub-acute"}, {Code: "chronic", Label: "Chronic"},
	}},
	{model.RefCareType, []seedRef{
		{Code: model.CareTypeRawatInap, Label: "Rawat Inap"},
		{Code: model.CareTypeRawatJalan, Label: "Rawat Jalan"},
	}},
	{model.RefCareClass, []seedRef{{Code: "I", Label: "Kelas I"}, {Code: "II", Label: "Kelas II"}, {Code: "III", Label: "Kelas III"}}},
	{model.RefParticipantType, []seedRef{
		{Code: model.ParticipantNonPBI, Label: "Non-PBI (eligible COB)", Tone: "success"},
		{Code: model.ParticipantPBI, Label: "PBI — tidak eligible COB", Tone: "danger"},
	}},
	{model.RefSex, []seedRef{{Code: "L", Label: "Laki-laki"}, {Code: "P", Label: "Perempuan"}}},
	{model.RefInsuranceType, []seedRef{{Code: "Individu", Label: "Individu"}, {Code: "Keluarga", Label: "Keluarga"}, {Code: "Perusahaan", Label: "Perusahaan"}}},
	{model.RefCoverageType, []seedRef{
		{Code: "Naik Kelas", Label: "Naik Kelas"},
		{Code: "Selisih Kelas", Label: "Selisih Kelas", Description: "Tidak berlaku untuk kelas III (scrubber R12)"},
		{Code: "Manfaat Rawat Inap", Label: "Manfaat Rawat Inap"},
		{Code: "Komersial Umum", Label: "Komersial Umum"},
	}},
	{model.RefHospitalClass, []seedRef{{Code: "A", Label: "Kelas A"}, {Code: "B", Label: "Kelas B"}, {Code: "C", Label: "Kelas C"}, {Code: "D", Label: "Kelas D"}}},
	{model.RefInsurerType, []seedRef{
		{Code: string(model.InsurerAsuransiSwasta), Label: "Asuransi Swasta"},
		{Code: string(model.InsurerAsuransiKantor), Label: "Asuransi Kantor"},
		{Code: string(model.InsurerPerusahaan), Label: "Perusahaan"},
	}},
	{model.RefWorkItemStatus, []seedRef{
		{Code: model.WorkItemNew, Label: "New", Tone: "teal"},
		{Code: model.WorkItemAssigned, Label: "Assigned", Tone: "teal"},
		{Code: model.WorkItemInProgress, Label: "In Progress", Tone: "teal"},
		{Code: model.WorkItemWaitingVerification, Label: "Waiting Verification", Tone: "teal"},
		{Code: model.WorkItemResolved, Label: "Resolved", Tone: "success"},
		{Code: model.WorkItemReopened, Label: "Reopened", Tone: "danger"},
		{Code: model.WorkItemClosed, Label: "Closed", Tone: "success"},
	}},
	{model.RefWorkItemPriority, []seedRef{
		{Code: model.PriorityRendah, Label: "Rendah", Tone: "neutral"},
		{Code: model.PrioritySedang, Label: "Sedang", Tone: "neutral"},
		{Code: model.PriorityTinggi, Label: "Tinggi", Tone: "warning"},
		{Code: model.PriorityKritis, Label: "Kritis", Tone: "danger"},
	}},
	{model.RefWorkItemSource, []seedRef{
		{Code: "SCRUBBER", Label: "Scrubber", Description: "Klaim memiliki blocker"},
		{Code: "DEADLINE", Label: "Deadline", Description: "Mendekati / melewati batas pengajuan"},
		{Code: "REJECTION", Label: "Penolakan", Description: "Klaim ditolak penjamin"},
		{Code: "RECONCILIATION", Label: "Rekonsiliasi", Description: "Mutasi bank perlu review"},
		{Code: "AGING", Label: "Aging", Description: "Piutang berumur > 90 hari"},
		{Code: "MANUAL", Label: "Manual", Description: "Dibuat pengguna"},
	}},
	{model.RefDeadlineZone, []seedRef{
		{Code: "aman", Label: "Aman", Tone: "success", Description: "> 30 hari"},
		{Code: "waspada", Label: "Waspada", Tone: "warning", Description: "8–30 hari"},
		{Code: "kritis", Label: "Kritis", Tone: "danger", Description: "0–7 hari"},
		{Code: "lewat", Label: "Lewat", Tone: "danger", Description: "Melewati batas 6 bulan"},
	}},
	{model.RefMatchResult, []seedRef{
		{Code: string(model.MatchMatched), Label: "Matched", Tone: "success", Description: "Skor ≥ 75"},
		{Code: string(model.MatchNeedsReview), Label: "Perlu Review", Tone: "warning", Description: "Skor 40–74"},
		{Code: string(model.MatchUnmatched), Label: "Unmatched", Tone: "neutral", Description: "Skor < 40"},
	}},
	{model.RefPaymentAllocation, []seedRef{{Code: "GAP", Label: "Gap (selisih tarif)"}, {Code: "COPAY", Label: "Co-pay pasien"}}},
	{model.RefAuditScope, []seedRef{{Code: model.AuditScopeWeb, Label: "WEB (aksi pengguna)"}, {Code: model.AuditScopeEngineAPI, Label: "ENGINE_API (partner)"}}},
	{model.RefIntakeStatus, []seedRef{{Code: "RESOLVED", Label: "Resolved", Tone: "success"}, {Code: "UNRESOLVED", Label: "Unresolved", Tone: "danger"}}},
	{model.RefRegulationStatus, []seedRef{
		{Code: "TERVERIFIKASI", Label: "Terverifikasi", Tone: "success"},
		{Code: "PERLU_VERIFIKASI", Label: "Perlu Verifikasi", Tone: "warning"},
		{Code: "SUPERSEDED", Label: "Superseded", Tone: "neutral"},
	}},
	{model.RefNotificationChannel, []seedRef{{Code: "IN_APP", Label: "In-app"}, {Code: "EMAIL", Label: "Email"}, {Code: "WHATSAPP", Label: "WhatsApp"}}},
	{model.RefNotificationStatus, []seedRef{{Code: "TERKIRIM", Label: "Terkirim", Tone: "success"}, {Code: "ANTRIAN", Label: "Antrian", Tone: "warning"}}},
	{model.RefScrubSeverity, []seedRef{
		{Code: string(model.SeverityBlocker), Label: "Blocker", Tone: "danger", Description: "Menghalangi pengajuan (−25 skor)"},
		{Code: string(model.SeverityWarning), Label: "Warning", Tone: "warning", Description: "Menurunkan skor kesiapan (−8 skor)"},
		{Code: string(model.SeverityInfo), Label: "Info", Tone: "info"},
	}},
}

var seedSettings = []model.AppSetting{
	{Key: model.SettingAppName, Value: "SPKD MedPay", Description: "Nama aplikasi"},
	{Key: model.SettingAppTagline, Value: "Dari selisih tarif sampai uang masuk.", Description: "Tagline aplikasi"},
	{Key: model.SettingSupportContact, Value: "support@spkd.co.id", Description: "Kontak dukungan"},
	{Key: model.SettingKAPJDisclaimer, Description: "Disclaimer simulasi KAPJ",
		Value: "Hasil simulasi bersifat indikatif, bukan angka final. Co-pay bersifat opsional sesuai produk. " +
			"Plafon kosong diperlakukan tak terbatas sehingga angka dapat lebih tinggi. Regulasi dalam masa transisi hingga " +
			"22 Desember 2026. [VERIFIKASI REGULASI]"},
}

var permissionLabels = []struct{ code, label string }{
	{model.PermView, "Lihat data"},
	{model.PermCreate, "Buat data"},
	{model.PermEdit, "Ubah data"},
	{model.PermDelete, "Hapus data"},
	{model.PermVerify, "Verifikasi"},
	{model.PermApprove, "Setujui / ajukan"},
	{model.PermExport, "Ekspor laporan"},
	{model.PermPrint, "Cetak"},
	{model.PermAssign, "Tugaskan pekerjaan"},
	{model.PermResolve, "Resolusi tarif (grouper)"},
	{model.PermReopen, "Buka kembali"},
	{model.PermConfigure, "Konfigurasi sistem"},
	{model.PermOverride, "Override temuan scrub"},
}

// rolePermissions = matriks peran & izin (SRS 2.3).
var rolePermissions = []struct {
	code, label string
	perms       []string
}{
	{model.RoleSuperAdmin, "SPKD Super Admin", nil}, // nil = semua izin
	{model.RoleHospitalAdmin, "Admin RS", nil},
	{model.RoleManagement, "Manajemen RS", []string{"VIEW", "EXPORT"}},
	{model.RoleBillingStaff, "Staf Billing/Klaim", []string{"VIEW", "CREATE", "EDIT", "DELETE", "EXPORT", "PRINT", "ASSIGN"}},
	{model.RoleCoder, "Coder", []string{"VIEW", "CREATE", "EDIT", "RESOLVE"}},
	{model.RoleReviewer, "Reviewer (Maker-Checker)", []string{"VIEW", "VERIFY", "APPROVE", "REOPEN"}},
	{model.RoleFinance, "Finance", []string{"VIEW", "CREATE", "EDIT", "APPROVE", "EXPORT"}},
	{model.RoleAuditor, "Auditor", []string{"VIEW", "EXPORT"}},
	{model.RoleITSIMRS, "IT / SIMRS", []string{"VIEW", "CONFIGURE"}},
	{model.RoleViewer, "Viewer (Read-only)", []string{"VIEW"}},
}

var resumeCategories = []struct{ code, label string }{
	{model.ResumeRekamMedis, "Rekam Medis"},
	{model.ResumeHasilLab, "Hasil Lab"},
	{model.ResumeHasilRadiologi, "Hasil Radiologi"},
	{model.ResumeObat, "Obat"},
	{model.ResumeBMHP, "BMHP"},
	{model.ResumeBilling, "Billing"},
	{model.ResumePenunjangLainnya, "Penunjang Lainnya"},
}

var seedRejectionCodes = []seedRejectionCode{
	{Code: "DOC_INCOMPLETE", Label: "Dokumen tidak lengkap"},
	{Code: "EXCL_PREEXIST", Label: "Penyakit pre-existing (pengecualian polis)"},
	{Code: "LIM_EXHAUSTED", Label: "Limit/plafon manfaat habis"},
	{Code: "WAITING_PERIOD", Label: "Masih dalam masa tunggu"},
	{Code: "WRONG_CLASS", Label: "Kelas rawat tidak sesuai manfaat"},
	{Code: "LATE_SUBMISSION", Label: "Pengajuan melewati batas waktu"},
	{Code: "ID_MISMATCH", Label: "Identitas tertanggung tidak cocok"},
	{Code: "DUP_BENEFIT", Label: "Manfaat ganda dengan penjamin lain"},
}

var seedScrubRules = []seedScrubRule{
	{
		ID:            "R01",
		Name:          "Field wajib kosong / format salah",
		Category:      "Kelengkapan Data",
		Severity:      "BLOCKER",
		Message:       "Field wajib belum lengkap atau format salah (SEP, No. Kartu BPJS, MRN, tanggal).",
		FixSuggestion: "Lengkapi field wajib pada Langkah 1; pastikan format tanggal YYYY-MM-DD.",
		SourceRef:     "Panduan e-Klaim INA-CBG — struktur field",
		Enabled:       true,
	},
	{
		ID:            "R02",
		Name:          "Tanggal pulang sebelum masuk / LOS tak konsisten",
		Category:      "Konsistensi",
		Severity:      "BLOCKER",
		Message:       "Tanggal pulang lebih awal dari tanggal masuk, atau LOS tidak sama dengan selisih tanggal.",
		FixSuggestion: "Perbaiki tanggal masuk/pulang; LOS dihitung otomatis dan tidak boleh diketik manual.",
		SourceRef:     "Kodefikasi klaim INA-CBG (LOS-driven)",
		Enabled:       true,
	},
	{
		ID:            "R03",
		Name:          "Peserta PBI tidak eligible COB",
		Category:      "Eligibilitas",
		Severity:      "BLOCKER",
		Message:       "Jenis peserta PBI (Penerima Bantuan Iuran) tidak eligible untuk klaim COB.",
		FixSuggestion: "Klaim COB hanya untuk peserta Non-PBI. Hapus klaim atau perbaiki jenis peserta.",
		SourceRef:     "Ketentuan COB JKN — eligibilitas peserta [VERIFIKASI REGULASI]",
		Enabled:       true,
	},
	{
		ID:            "R04",
		Name:          "Klaim duplikat (SEP sama)",
		Category:      "Duplikasi",
		Severity:      "BLOCKER",
		Message:       "No. SEP yang sama sudah terdaftar pada klaim lain, atau SEP yang sama diajukan ke lebih dari satu penjamin.",
		FixSuggestion: "Periksa daftar klaim dengan No. SEP tersebut; satu SEP hanya boleh satu klaim COB aktif.",
		SourceRef:     "Prinsip anti-duplikasi penagihan ganda",
		Enabled:       true,
	},
	{
		ID:            "R05",
		Name:          "Format/validitas ICD-10 & ICD-9-CM",
		Category:      "Kodefikasi",
		Severity:      "WARNING",
		Message:       "Kode ICD-10/ICD-9-CM berformat tidak standar, diagnosa sekunder sama dengan utama, atau kombinasi tidak lazim untuk jenis kelamin/umur.",
		FixSuggestion: "Verifikasi kode pada ICD-10/ICD-9-CM resmi; pastikan diagnosa sekunder berbeda dari utama.",
		SourceRef:     "ICD-10 WHO 2010; ICD-9-CM vol.3 [VERIFIKASI REGULASI]",
		Enabled:       true,
	},
	{
		ID:            "R06",
		Name:          "Resolusi INA-CBG tidak ada / basi",
		Category:      "Tarif",
		Severity:      "BLOCKER",
		Message:       "Tarif INA-CBG belum diresolve, atau resolusi basi karena diagnosa/prosedur/LOS berubah setelah resolusi terakhir.",
		FixSuggestion: "Klik Resolusi Tarif INA-CBG ulang setelah perubahan diagnosa, prosedur, atau LOS.",
		SourceRef:     "Operasional grouper e-Klaim",
		Enabled:       true,
	},
	{
		ID:            "R07",
		Name:          "Jenis grup CBG ≠ jenis rawat",
		Category:      "Kodefikasi",
		Severity:      "WARNING",
		Message:       "Kode INA-CBG berakhiran '-0' (rawat jalan) tetapi jenis rawat adalah rawat inap, atau sebaliknya.",
		FixSuggestion: "Resolve ulang tarif dengan jenis rawat yang benar (rawat jalan → kode berakhiran '-0').",
		SourceRef:     "Struktur kode INA-CBG",
		Enabled:       true,
	},
	{
		ID:            "R08",
		Name:          "Tagihan RS di bawah Tarif INA-CBG",
		Category:      "Tarif",
		Severity:      "WARNING",
		Message:       "Total Tarif RS lebih rendah dari Total Tarif JKN — tidak ada selisih untuk diklaim; BPJS dapat mem-pending klaim.",
		FixSuggestion: "Periksa kembali 18 komponen tarif RS atau resolusi tarif INA-CBG.",
		SourceRef:     "Mekanisme verifikasi e-Klaim",
		Enabled:       true,
	},
	{
		ID:            "R09",
		Name:          "Plafon polis di bawah nilai klaim",
		Category:      "Penjamin Kedua",
		Severity:      "WARNING",
		Message:       "Selisih tarif > 0 tetapi plafon polis lebih rendah dari nilai selisih — penagihan akan terpotong plafon.",
		FixSuggestion: "Konfirmasi plafon dengan penjamin; pertimbangkan pembagian tagihan atau update data polis.",
		SourceRef:     "Kontrak kerja sama penjamin kedua",
		Enabled:       true,
	},
	{
		ID:            "R10",
		Name:          "Resume medis belum lengkap / belum direview",
		Category:      "Dokumen",
		Severity:      "BLOCKER",
		Message:       "Resume medis belum diunggah, atau masih ada item yang belum ditandai Reviewed.",
		FixSuggestion: "Unggah CSV resume medis dan review seluruh item sebelum generate paket dokumen.",
		SourceRef:     "Prosedur pengajuan COB — kelengkapan berkas",
		Enabled:       true,
	},
	{
		ID:            "R11",
		Name:          "Dekat/melewati batas pengajuan 6 bulan",
		Category:      "Deadline",
		Severity:      "WARNING",
		Message:       "Pengajuan mendekati atau melewati batas 6 bulan sejak tanggal pelayanan.",
		FixSuggestion: "Segera ajukan, atau minta Override Admin RS dengan alasan tercatat di audit trail.",
		SourceRef:     "[VERIFIKASI REGULASI] batas waktu pengajuan 6 bulan",
		Enabled:       true,
	},
	{
		ID:            "R12",
		Name:          "Kelas rawat vs jenis perlindungan tidak konsisten",
		Category:      "Eligibilitas",
		Severity:      "WARNING",
		Message:       "Kelas rawat tidak konsisten dengan jenis perlindungan polis (mis. selisih kelas tetapi rawat Kelas III).",
		FixSuggestion: "Verifikasi kelas rawat dan jenis perlindungan pada polis penjamin kedua.",
		SourceRef:     "Kontrak penjamin — manfaat selisih kelas",
		Enabled:       true,
	},
	{
		ID:            "R13",
		Name:          "Pernyataan kebenaran data / data penjamin kosong",
		Category:      "Kelengkapan Data",
		Severity:      "BLOCKER",
		Message:       "Pernyataan kebenaran data belum dicentang, atau Nama Tertanggung/No. Polis kosong.",
		FixSuggestion: "Centang pernyataan kebenaran data dan lengkapi identitas penjamin pada Langkah 4.",
		SourceRef:     "Kebijakan internal integritas klaim",
		Enabled:       true,
	},
	{
		ID:            "R14",
		Name:          "Melewati deadline kontraktual penjamin",
		Category:      "Deadline",
		Severity:      "WARNING",
		Message:       "Klaim belum diajukan sebelum batas waktu kontraktual penjamin kedua (dapat dikonfigurasi per penjamin).",
		FixSuggestion: "Ajukan segera atau negosiasikan perpanjangan dengan penjamin.",
		SourceRef:     "Perjanjian kerja sama per penjamin (Payer Rule Library)",
		Enabled:       true,
	},
}
