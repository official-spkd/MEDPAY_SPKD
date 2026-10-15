import { reactive } from 'vue'

// ── SPKD MedPay — Formatter (port engine.ts) ────────────────────────────────

export function formatRp(n) {
  if (n === null || n === undefined || Number.isNaN(n)) return '—'
  return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(n)
}

export function formatRpShort(n) {
  const abs = Math.abs(n)
  if (abs >= 1_000_000_000) return 'Rp ' + (n / 1_000_000_000).toFixed(1).replace('.', ',') + ' M'
  if (abs >= 1_000_000) return 'Rp ' + (n / 1_000_000).toFixed(1).replace('.', ',') + ' jt'
  if (abs >= 1_000) return 'Rp ' + Math.round(n / 1_000) + ' rb'
  return 'Rp ' + n
}

export function formatNumber(n) { return new Intl.NumberFormat('id-ID').format(n ?? 0) }

export function formatDate(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short', year: 'numeric' }).format(d)
}

export function formatDateTime(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' }).format(d)
}

export function todayISO() { return new Date().toISOString().slice(0, 10) }

export function daysBetween(a, b) {
  if (!a || !b) return 0
  const da = new Date(a + 'T00:00:00').getTime()
  const db = new Date(b + 'T00:00:00').getTime()
  return Math.round((db - da) / 86_400_000)
}

export function addDays(iso, days) {
  const d = new Date((iso || todayISO()) + 'T00:00:00')
  d.setDate(d.getDate() + days)
  return d.toISOString().slice(0, 10)
}

export function parseMoney(str) {
  if (typeof str === 'number') return Math.round(str)
  const digits = String(str ?? '').replace(/[^\d-]/g, '')
  return digits ? parseInt(digits, 10) : 0
}

export function debounce(fn, ms) {
  let t
  return (...args) => { clearTimeout(t); t = setTimeout(() => fn(...args), ms) }
}

// ── Label & pilihan ─────────────────────────────────────────────────────────
// Nilai di bawah adalah cadangan (mode mock). Saat backend tersedia, isinya
// diganti dari GET /bootstrap → reference (tabel ref_codes) lewat applyReference().

export const CLAIM_LABELS = reactive({
  draft: 'Draft', ready_to_submit: 'Siap Diajukan', submitted: 'Diajukan',
  approved: 'Disetujui', partially_approved: 'Disetujui Sebagian', rejected: 'Ditolak',
  partially_paid: 'Sebagian Dibayar', paid: 'Lunas', disputed: 'Disanggah', written_off: 'Dihapus Buku',
})

export const CLAIM_TONE = reactive({
  draft: 'neutral', ready_to_submit: 'teal', submitted: 'info', approved: 'success',
  partially_approved: 'warning', rejected: 'danger', partially_paid: 'warning',
  paid: 'success', disputed: 'danger', written_off: 'neutral',
})

export const COMPONENT_LABELS = reactive({
  prosedurNonBedah: 'Prosedur Non Bedah', prosedurBedah: 'Prosedur Bedah', konsultasi: 'Konsultasi',
  tenagaAhli: 'Tenaga Ahli', keperawatan: 'Keperawatan', penunjang: 'Penunjang', radiologi: 'Radiologi',
  laboratorium: 'Laboratorium', pelayananDarah: 'Pelayanan Darah', rehabilitasi: 'Rehabilitasi',
  kamarAkomodasi: 'Kamar Akomodasi', rawatIntensif: 'Rawat Intensif', obat: 'Obat',
  obatKronis: 'Obat Kronis', obatKemo: 'Obat Kemo', alkes: 'Alkes', bmhp: 'BMHP', sewaAlat: 'Sewa Alat',
})

// Kunci komponen tetap (struktur data klaim), label boleh berubah.
export const TARIFF_COMPONENT_KEYS = Object.keys(COMPONENT_LABELS)

export const SPECIAL_CMG_LABELS = reactive({
  specialProcedure: 'Special Procedure (SP)', specialProsthesis: 'Special Prosthesis (SR)',
  specialInvestigation: 'Special Investigation (SI)', specialDrug: 'Special Drug (SD)',
  subAcute: 'Sub-acute', chronic: 'Chronic',
})

export const SPECIAL_CMG_KEYS = Object.keys(SPECIAL_CMG_LABELS)

export const RESUME_CATEGORIES = reactive([
  { key: 'REKAM_MEDIS', label: 'Rekam Medis' }, { key: 'HASIL_LAB', label: 'Hasil Lab' },
  { key: 'HASIL_RADIOLOGI', label: 'Hasil Radiologi' }, { key: 'OBAT', label: 'Obat' },
  { key: 'BMHP', label: 'BMHP' }, { key: 'BILLING', label: 'Billing' },
  { key: 'PENUNJANG_LAINNYA', label: 'Penunjang Lainnya' },
])

export const ROLE_LABELS = reactive({
  SUPER_ADMIN: 'SPKD Super Admin', HOSPITAL_ADMIN: 'Admin RS', MANAGEMENT: 'Manajemen RS',
  BILLING_STAFF: 'Staf Billing/Klaim', CODER: 'Coder', REVIEWER: 'Reviewer (Maker-Checker)',
  FINANCE: 'Finance', AUDITOR: 'Auditor', IT_SIMRS: 'IT / SIMRS', VIEWER: 'Viewer (Read-only)',
})

// Pilihan cadangan untuk kategori referensi lain (mode mock).
const FALLBACK_OPTIONS = {
  care_type: [{ code: 'RAWAT_INAP', label: 'Rawat Inap' }, { code: 'RAWAT_JALAN', label: 'Rawat Jalan' }],
  care_class: [{ code: 'I', label: 'Kelas I' }, { code: 'II', label: 'Kelas II' }, { code: 'III', label: 'Kelas III' }],
  participant_type: [{ code: 'NON_PBI', label: 'Non-PBI (eligible COB)' }, { code: 'PBI', label: 'PBI — tidak eligible COB' }],
  sex: [{ code: 'L', label: 'Laki-laki' }, { code: 'P', label: 'Perempuan' }],
  coverage_type: ['Naik Kelas', 'Selisih Kelas', 'Manfaat Rawat Inap', 'Komersial Umum'].map(c => ({ code: c, label: c })),
  insurance_type: ['Individu', 'Keluarga', 'Perusahaan'].map(c => ({ code: c, label: c })),
  hospital_class: ['A', 'B', 'C', 'D'].map(c => ({ code: c, label: 'Kelas ' + c })),
  insurer_type: [{ code: 'ASURANSI_SWASTA', label: 'Asuransi Swasta' }, { code: 'ASURANSI_KANTOR', label: 'Asuransi Kantor' }, { code: 'PERUSAHAAN', label: 'Perusahaan' }],
  work_item_status: [
    { code: 'New', tone: 'teal' }, { code: 'Assigned', tone: 'teal' }, { code: 'In Progress', tone: 'teal' },
    { code: 'Waiting Verification', tone: 'teal' }, { code: 'Resolved', tone: 'success' },
    { code: 'Reopened', tone: 'danger' }, { code: 'Closed', tone: 'success' },
  ].map(x => ({ ...x, label: x.code })),
  work_item_priority: [
    { code: 'Rendah', tone: 'neutral' }, { code: 'Sedang', tone: 'neutral' }, { code: 'Tinggi', tone: 'warning' }, { code: 'Kritis', tone: 'danger' },
  ].map(x => ({ ...x, label: x.code })),
  work_item_source: ['SCRUBBER', 'DEADLINE', 'REJECTION', 'RECONCILIATION', 'AGING', 'MANUAL'].map(c => ({ code: c, label: c })),
  deadline_zone: [
    { code: 'aman', label: 'Aman', tone: 'success' }, { code: 'waspada', label: 'Waspada', tone: 'warning' },
    { code: 'kritis', label: 'Kritis', tone: 'danger' }, { code: 'lewat', label: 'Lewat', tone: 'danger' },
  ],
  match_result: [
    { code: 'matched', label: 'Matched', tone: 'success' }, { code: 'needs_review', label: 'Perlu Review', tone: 'warning' },
    { code: 'unmatched', label: 'Unmatched', tone: 'neutral' },
  ],
  regulation_status: [
    { code: 'TERVERIFIKASI', label: 'Terverifikasi', tone: 'success' }, { code: 'PERLU_VERIFIKASI', label: 'Perlu Verifikasi', tone: 'warning' },
    { code: 'SUPERSEDED', label: 'Superseded', tone: 'neutral' },
  ],
  intake_status: [{ code: 'RESOLVED', label: 'Resolved', tone: 'success' }, { code: 'UNRESOLVED', label: 'Unresolved', tone: 'danger' }],
  payment_allocation: [{ code: 'GAP', label: 'Gap (selisih tarif)' }, { code: 'COPAY', label: 'Co-pay pasien' }],
}

// REF = salinan reaktif data referensi backend (kosong pada mode mock).
export const REF = reactive({ codes: {}, labels: {}, settings: {}, roles: [], businessRules: {}, loaded: false })

function replaceObject(target, src) {
  for (const k of Object.keys(target)) if (!(k in src)) delete target[k]
  Object.assign(target, src)
}

// applyReference mengisi label & pilihan dari GET /bootstrap → reference.
export function applyReference(r) {
  if (!r || !r.codes) return
  REF.codes = r.codes
  REF.labels = r.labels || {}
  REF.settings = r.settings || {}
  REF.roles = r.roles || []
  REF.businessRules = r.businessRules || {}
  REF.loaded = true
  const map = (cat, field) => Object.fromEntries((r.codes[cat] || []).map(c => [c.code, c[field]]))
  if (r.codes.claim_status?.length) {
    replaceObject(CLAIM_LABELS, map('claim_status', 'label'))
    replaceObject(CLAIM_TONE, map('claim_status', 'tone'))
  }
  if (r.codes.tariff_component?.length) Object.assign(COMPONENT_LABELS, map('tariff_component', 'label'))
  if (r.codes.special_cmg?.length) Object.assign(SPECIAL_CMG_LABELS, map('special_cmg', 'label'))
  if (r.resumeCategories?.length) RESUME_CATEGORIES.splice(0, RESUME_CATEGORIES.length, ...r.resumeCategories.map(c => ({ key: c.code, label: c.label })))
  if (r.roles?.length) replaceObject(ROLE_LABELS, Object.fromEntries(r.roles.map(x => [x.code, x.label])))
}

// refOptions = daftar pilihan aktif suatu kategori: [{ code, label, tone, description }].
export function refOptions(category) {
  const list = REF.codes[category]
  if (list && list.length) return list.filter(x => x.active !== false)
  return FALLBACK_OPTIONS[category] || []
}

export function refLabel(category, code) {
  if (code === null || code === undefined || code === '') return '—'
  return refOptions(category).find(x => x.code === code)?.label || code
}

export function refTone(category, code, fallback = 'neutral') {
  return refOptions(category).find(x => x.code === code)?.tone || fallback
}

export function appSetting(key, fallback = '') { return REF.settings[key] || fallback }

// docNo = nomor dokumen untuk tampilan (CLM-0001, BT-0001, …). Backend mengirim `id`
// berupa UUID dan nomor di field terpisah; data mock memakai nomor sebagai `id`.
export function docNo(x, field = 'claimNo') {
  if (!x) return ''
  return x[field] || x.id || ''
}

export function sumTariff(t) { return TARIFF_COMPONENT_KEYS.reduce((s, k) => s + (t[k] || 0), 0) }
export function sumCmg(s0) { return SPECIAL_CMG_KEYS.reduce((a, k) => a + (s0[k] || 0), 0) }
export function jknTotal(base, cmg) { return (base || 0) + sumCmg(cmg) }
export function gapOf(hospital, jkn) { return hospital - jkn }

// Re-export CSV helpers agar view dapat mengimpor dari satu tempat
export { toCsv, downloadBlobCsv } from './api'
