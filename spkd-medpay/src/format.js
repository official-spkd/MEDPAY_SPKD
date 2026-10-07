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

export const CLAIM_LABELS = {
  draft: 'Draft', ready_to_submit: 'Siap Diajukan', submitted: 'Diajukan',
  approved: 'Disetujui', partially_approved: 'Disetujui Sebagian', rejected: 'Ditolak',
  partially_paid: 'Sebagian Dibayar', paid: 'Lunas', disputed: 'Disanggah', written_off: 'Dihapus Buku',
}

export const CLAIM_TONE = {
  draft: 'neutral', ready_to_submit: 'teal', submitted: 'info', approved: 'success',
  partially_approved: 'warning', rejected: 'danger', partially_paid: 'warning',
  paid: 'success', disputed: 'danger', written_off: 'neutral',
}

export const COMPONENT_LABELS = {
  prosedurNonBedah: 'Prosedur Non Bedah', prosedurBedah: 'Prosedur Bedah', konsultasi: 'Konsultasi',
  tenagaAhli: 'Tenaga Ahli', keperawatan: 'Keperawatan', penunjang: 'Penunjang', radiologi: 'Radiologi',
  laboratorium: 'Laboratorium', pelayananDarah: 'Pelayanan Darah', rehabilitasi: 'Rehabilitasi',
  kamarAkomodasi: 'Kamar Akomodasi', rawatIntensif: 'Rawat Intensif', obat: 'Obat',
  obatKronis: 'Obat Kronis', obatKemo: 'Obat Kemo', alkes: 'Alkes', bmhp: 'BMHP', sewaAlat: 'Sewa Alat',
}

export const TARIFF_COMPONENT_KEYS = Object.keys(COMPONENT_LABELS)

export const SPECIAL_CMG_LABELS = {
  specialProcedure: 'Special Procedure (SP)', specialProsthesis: 'Special Prosthesis (SR)',
  specialInvestigation: 'Special Investigation (SI)', specialDrug: 'Special Drug (SD)',
  subAcute: 'Sub-acute', chronic: 'Chronic',
}

export const SPECIAL_CMG_KEYS = Object.keys(SPECIAL_CMG_LABELS)

export const RESUME_CATEGORIES = [
  { key: 'REKAM_MEDIS', label: 'Rekam Medis' }, { key: 'HASIL_LAB', label: 'Hasil Lab' },
  { key: 'HASIL_RADIOLOGI', label: 'Hasil Radiologi' }, { key: 'OBAT', label: 'Obat' },
  { key: 'BMHP', label: 'BMHP' }, { key: 'BILLING', label: 'Billing' },
  { key: 'PENUNJANG_LAINNYA', label: 'Penunjang Lainnya' },
]

export const ROLE_LABELS = {
  SUPER_ADMIN: 'SPKD Super Admin', HOSPITAL_ADMIN: 'Admin RS', MANAGEMENT: 'Manajemen RS',
  BILLING_STAFF: 'Staf Billing/Klaim', CODER: 'Coder', REVIEWER: 'Reviewer (Maker-Checker)',
  FINANCE: 'Finance', AUDITOR: 'Auditor', IT_SIMRS: 'IT / SIMRS', VIEWER: 'Viewer (Read-only)',
}

export function sumTariff(t) { return TARIFF_COMPONENT_KEYS.reduce((s, k) => s + (t[k] || 0), 0) }
export function sumCmg(s0) { return SPECIAL_CMG_KEYS.reduce((a, k) => a + (s0[k] || 0), 0) }
export function jknTotal(base, cmg) { return (base || 0) + sumCmg(cmg) }
export function gapOf(hospital, jkn) { return hospital - jkn }

// Re-export CSV helpers agar view dapat mengimpor dari satu tempat
export { toCsv, downloadBlobCsv } from './api'
