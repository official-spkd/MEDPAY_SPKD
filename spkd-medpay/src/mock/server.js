// ── SPKD MedPay — Mock API (berjalan di browser) ────────────────────────────
// Port dari backend Go (mini-services/medpay-api) agar frontend dapat di-deploy
// statis (mis. Vercel) tanpa backend. Semua data SINTETIS / DEMO.
//
//  • Data awal  : ./data.json (salinan seed backend Go, tanpa hash kata sandi).
//  • Tanggal    : seluruh tanggal digeser relatif terhadap "hari ini" (seed dibuat
//                 2026-10-03) sehingga deadline, aging, dan KPI tetap bermakna.
//  • Persistensi: perubahan disimpan di sessionStorage (bertahan saat refresh,
//                 kembali segar pada tab/sesi baru).
//  • Sesi       : token stateless "mock.<userId>".

import seedData from './data.json'

const SEED_DATE = '2026-10-03'
const STORAGE_KEY = 'spkd_mock_db_v1'
const DEMO_PASSWORD = 'medpay2026'
const MFA_CODE = '246810'
const LATENCY_MS = 90

// ── Waktu ────────────────────────────────────────────────────────────────────
const pad = (n) => String(n).padStart(2, '0')
const fmtUTC = (d) => `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())}`
function parseISO(s) {
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(s || '')
  return m ? new Date(Date.UTC(+m[1], +m[2] - 1, +m[3])) : null
}
function todayISO() {
  const d = new Date()
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}
function nowStamp() {
  const d = new Date()
  return `${todayISO()}T${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}
function addDays(iso, days) {
  const t = parseISO(iso)
  if (!t) return iso
  t.setUTCDate(t.getUTCDate() + days)
  return fmtUTC(t)
}
function daysBetween(a, b) {
  const ta = parseISO(a), tb = parseISO(b)
  if (!ta || !tb) return 0
  return Math.round((tb - ta) / 86400000)
}

// ── Konstanta domain ─────────────────────────────────────────────────────────
const TARIFF_KEYS = [
  'prosedurNonBedah', 'prosedurBedah', 'konsultasi', 'tenagaAhli', 'keperawatan', 'penunjang',
  'radiologi', 'laboratorium', 'pelayananDarah', 'rehabilitasi', 'kamarAkomodasi', 'rawatIntensif',
  'obat', 'obatKronis', 'obatKemo', 'alkes', 'bmhp', 'sewaAlat',
]
const CMG_KEYS = ['specialProcedure', 'specialProsthesis', 'specialInvestigation', 'specialDrug', 'subAcute', 'chronic']

const COMPONENT_LABELS = {
  prosedurNonBedah: 'Prosedur Non Bedah', prosedurBedah: 'Prosedur Bedah', konsultasi: 'Konsultasi',
  tenagaAhli: 'Tenaga Ahli', keperawatan: 'Keperawatan', penunjang: 'Penunjang', radiologi: 'Radiologi',
  laboratorium: 'Laboratorium', pelayananDarah: 'Pelayanan Darah', rehabilitasi: 'Rehabilitasi',
  kamarAkomodasi: 'Kamar Akomodasi', rawatIntensif: 'Rawat Intensif', obat: 'Obat', obatKronis: 'Obat Kronis',
  obatKemo: 'Obat Kemo', alkes: 'Alkes', bmhp: 'BMHP', sewaAlat: 'Sewa Alat',
}
const SPECIAL_CMG_LABELS = {
  specialProcedure: 'Special Procedure (SP)', specialProsthesis: 'Special Prosthesis (SR)',
  specialInvestigation: 'Special Investigation (SI)', specialDrug: 'Special Drug (SD)',
  subAcute: 'Sub-acute', chronic: 'Chronic',
}
const CLAIM_LABELS = {
  draft: 'Draft', ready_to_submit: 'Siap Diajukan', submitted: 'Diajukan', approved: 'Disetujui',
  partially_approved: 'Disetujui Sebagian', rejected: 'Ditolak', partially_paid: 'Sebagian Dibayar',
  paid: 'Lunas', disputed: 'Disanggah', written_off: 'Dihapus Buku',
}
const RESUME_CATEGORIES = [
  { Key: 'REKAM_MEDIS', Label: 'Rekam Medis' }, { Key: 'HASIL_LAB', Label: 'Hasil Lab' },
  { Key: 'HASIL_RADIOLOGI', Label: 'Hasil Radiologi' }, { Key: 'OBAT', Label: 'Obat' },
  { Key: 'BMHP', Label: 'BMHP' }, { Key: 'BILLING', Label: 'Billing' },
  { Key: 'PENUNJANG_LAINNYA', Label: 'Penunjang Lainnya' },
]

const ALL_PERMS = ['VIEW', 'CREATE', 'EDIT', 'DELETE', 'VERIFY', 'APPROVE', 'EXPORT', 'PRINT', 'ASSIGN', 'RESOLVE', 'REOPEN', 'CONFIGURE', 'OVERRIDE']
const ROLE_PERMS = {
  SUPER_ADMIN: ALL_PERMS,
  HOSPITAL_ADMIN: ALL_PERMS,
  MANAGEMENT: ['VIEW', 'EXPORT'],
  BILLING_STAFF: ['VIEW', 'CREATE', 'EDIT', 'DELETE', 'EXPORT', 'PRINT', 'ASSIGN'],
  CODER: ['VIEW', 'CREATE', 'EDIT', 'RESOLVE'],
  REVIEWER: ['VIEW', 'VERIFY', 'APPROVE', 'REOPEN'],
  FINANCE: ['VIEW', 'CREATE', 'EDIT', 'APPROVE', 'EXPORT'],
  AUDITOR: ['VIEW', 'EXPORT'],
  IT_SIMRS: ['VIEW', 'CONFIGURE'],
  VIEWER: ['VIEW'],
}

const CLAIM_TRANSITIONS = {
  draft: ['ready_to_submit', 'written_off'],
  ready_to_submit: ['submitted', 'draft', 'written_off'],
  submitted: ['approved', 'partially_approved', 'rejected', 'disputed'],
  approved: ['partially_paid', 'paid', 'disputed'],
  partially_approved: ['partially_paid', 'paid', 'rejected', 'disputed'],
  rejected: ['disputed', 'written_off'],
  disputed: ['submitted', 'ready_to_submit', 'written_off'],
  partially_paid: ['paid', 'disputed', 'written_off'],
  paid: ['written_off'],
  written_off: [],
}
const canTransition = (from, to) => (CLAIM_TRANSITIONS[from] || []).includes(to)

const DEFAULT_KAPJ = { bpjsSharePct: 75, coPayEnabled: true, patientCoPayPct: 5, outpatientCoPayCap: 300000, inpatientCoPayCap: 3000000 }

// ── Store (in-memory + sessionStorage) ───────────────────────────────────────
let db = null

const DATE_RE = /^\d{4}-\d{2}-\d{2}(T\d{2}:\d{2}:\d{2})?$/
const SKIP_SHIFT_KEYS = new Set(['dateOfBirth', 'effectiveDate', 'lastReviewed', 'registeredAt'])

function shiftDates(node, delta) {
  if (!delta) return
  const walk = (obj) => {
    for (const k of Object.keys(obj)) {
      const v = obj[k]
      if (typeof v === 'string') {
        if (!SKIP_SHIFT_KEYS.has(k) && DATE_RE.test(v)) obj[k] = addDays(v.slice(0, 10), delta) + v.slice(10)
      } else if (v && typeof v === 'object') walk(v)
    }
  }
  walk(node)
}

function initSeq(d) {
  d.seq = d.seq || {}
  const bump = (prefix, id) => {
    const m = new RegExp('^' + prefix + '-(\\d+)$').exec(id || '')
    if (m) d.seq[prefix] = Math.max(d.seq[prefix] || 0, parseInt(m[1], 10))
  }
  const cols = {
    CLM: d.claims, PAY: d.payments, BT: d.batches, BI: d.bankImports, BL: d.bankLines,
    WI: d.workItems, RS: d.resumeItems, AL: d.auditLog, NT: d.notifications,
  }
  for (const [p, arr] of Object.entries(cols)) (arr || []).forEach((x) => bump(p, x.id))
  ;(d.workItems || []).forEach((w) => {
    ;(w.comments || []).forEach((c) => bump('WC', c.id))
    ;(w.history || []).forEach((h) => bump('WH', h.id))
  })
}

function loadDb() {
  if (db) return db
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    if (raw) { db = JSON.parse(raw); return db }
  } catch { /* abaikan */ }
  db = JSON.parse(JSON.stringify(seedData))
  db.overrides = db.overrides || {}
  shiftDates(db, daysBetween(SEED_DATE, todayISO()))
  initSeq(db)
  return db
}

function persist() {
  try { sessionStorage.setItem(STORAGE_KEY, JSON.stringify(db)) } catch { /* kuota/non-browser */ }
}

export function resetMockData() {
  db = null
  try { sessionStorage.removeItem(STORAGE_KEY) } catch { /* abaikan */ }
  loadDb()
}

function nextID(prefix) {
  db.seq[prefix] = (db.seq[prefix] || 0) + 1
  return prefix + '-' + String(db.seq[prefix]).padStart(4, '0')
}

// ── Helper entitas ───────────────────────────────────────────────────────────
const userByID = (id) => db.users.find((u) => u.id === id) || null
const userByEmail = (e) => db.users.find((u) => u.email.toLowerCase() === String(e || '').toLowerCase()) || null
const claimByID = (id) => db.claims.find((c) => c.id === id) || null
const hospitalByID = (id) => db.hospitals.find((h) => h.id === id) || null
const insurerByID = (id) => db.insurers.find((i) => i.id === id) || null
const scopeOK = (u, c) => !u.hospitalId || c.hospitalId === u.hospitalId
const hasPermission = (u, p) => (ROLE_PERMS[u.role] || []).includes(p)
const clone = (x) => (x === undefined ? undefined : JSON.parse(JSON.stringify(x)))

function publicUser(u) {
  const { passHash, ...rest } = u
  return rest
}

function audit(actor, action, entity, entityId, before, after) {
  const e = {
    id: nextID('AL'), at: nowStamp(), actor: actor ? actor.name : 'sistem', actorRole: actor ? actor.role : 'SYSTEM',
    action, entity, entityId, ip: '127.0.0.1',
    correlationId: 'c-' + Math.random().toString(16).slice(2, 12).padEnd(10, '0'), scope: 'WEB',
  }
  if (before) e.before = before
  if (after) e.after = after
  db.auditLog.push(e)
}

// ── Rantai tarif ─────────────────────────────────────────────────────────────
const num = (v) => { const x = Number(v); return Number.isFinite(x) ? Math.trunc(x) : 0 }
const sumTariff = (t) => TARIFF_KEYS.reduce((s, k) => s + num(t && t[k]), 0)
const sumCmg = (t) => CMG_KEYS.reduce((s, k) => s + num(t && t[k]), 0)
const totalJKN = (base, cmg) => num(base) + sumCmg(cmg)
const claimGap = (c) => sumTariff(c.hospitalTariff) - totalJKN(c.inacbgBaseTariff, c.specialCmg)
const losOf = (adm, dis) => Math.max(0, daysBetween(adm, dis))

function paidAmount(claimId) {
  return db.payments.reduce((s, p) => (p.claimId === claimId ? s + p.amount : s), 0)
}
function outstandingOf(c) {
  return Math.max(0, claimGap(c) - paidAmount(c.id))
}
function recomputePaymentStatus(c) {
  if (!['approved', 'partially_approved', 'partially_paid'].includes(c.status)) return
  const total = claimGap(c)
  const paid = paidAmount(c.id)
  if (paid <= 0) return
  c.status = paid >= total ? 'paid' : 'partially_paid'
}
function lastPaymentDate(claimId) {
  let last = ''
  for (const p of db.payments) if (p.claimId === claimId && p.receivedDate > last) last = p.receivedDate
  return last
}
const orDefault = (a, b) => (a ? a : b)

function tariffSignature(c) {
  return [c.primaryDiagnosis, (c.secondaryDiagnoses || []).join(','), (c.procedures || []).join(','),
    String(c.los), c.careType, c.careClass].join('|')
}

function deadlineInfo(c, today) {
  if (!c.admissionDate) return { deadline: '', daysLeft: 0, zone: 'aman' }
  const t = parseISO(c.admissionDate)
  if (!t) return { deadline: '', daysLeft: 0, zone: 'aman' }
  t.setUTCMonth(t.getUTCMonth() + 6)
  const deadline = fmtUTC(t)
  const daysLeft = daysBetween(today, deadline)
  let zone = 'aman'
  if (daysLeft < 0) zone = 'lewat'
  else if (daysLeft <= 7) zone = 'kritis'
  else if (daysLeft <= 30) zone = 'waspada'
  return { deadline, daysLeft, zone }
}

// ── KAPJ ─────────────────────────────────────────────────────────────────────
function kapjSplit(totalBill, jknTotal, policyCeiling, careType, p) {
  let bpjsPays = Math.trunc((jknTotal * p.bpjsSharePct) / 100)
  if (bpjsPays > jknTotal) bpjsPays = jknTotal
  let remaining = totalBill - bpjsPays
  if (remaining < 0) remaining = 0
  let insurerPays = remaining
  let status = 'Lunas Penuh'
  if (policyCeiling != null && policyCeiling > 0 && insurerPays > policyCeiling) {
    insurerPays = policyCeiling
    status = 'Dibatasi Plafon'
  }
  let patientPays = 0
  if (p.coPayEnabled && p.patientCoPayPct > 0) {
    patientPays = Math.trunc((insurerPays * p.patientCoPayPct) / 100)
    const cap = careType === 'RAWAT_INAP' ? p.inpatientCoPayCap : p.outpatientCoPayCap
    if (patientPays > cap) {
      patientPays = cap
      if (status === 'Lunas Penuh') status = 'Co-pay Maks'
    }
  }
  const netInsurer = insurerPays - patientPays
  if (status === 'Lunas Penuh' && p.coPayEnabled && patientPays > 0) status = 'Sebagian (Plafon)'
  return { totalHospitalBill: totalBill, bpjsPays, insurerPays: netInsurer, patientPays, status }
}

// ── Scrubber (14 aturan) ─────────────────────────────────────────────────────
const ICD10 = /^[A-Z]\d{2}(\.\d{1,2})?$/
const ICD9 = /^\d{2}(\.\d{1,2})?$/

function evaluateClaim(c, today) {
  const findings = []
  const push = (r) => findings.push({
    ruleId: r.id, severity: r.severity, message: r.message, fixSuggestion: r.fixSuggestion,
    sourceRef: r.sourceRef, overridden: !!db.overrides[c.id + '|' + r.id],
  })
  const run = (id, cond) => {
    if (!cond) return
    const r = db.scrubRules.find((x) => x.id === id)
    if (r && r.enabled) push(r)
  }

  run('R01', !c.sepNo || !c.bpjsCardNo || !c.mrn || !c.patientName || !c.admissionDate || !c.dischargeDate || !c.dateOfBirth)

  const realLos = losOf(c.admissionDate, c.dischargeDate)
  run('R02', !!c.admissionDate && !!c.dischargeDate && realLos !== c.los)

  run('R03', c.participantType === 'PBI')

  run('R04', !!c.sepNo && db.claims.some((o) => o.id !== c.id && o.hospitalId === c.hospitalId && o.sepNo === c.sepNo && o.status !== 'written_off'))

  const sec = c.secondaryDiagnoses || []
  const procs = c.procedures || []
  const up = (c.primaryDiagnosis || '').toUpperCase()
  const badPrimary = !!c.primaryDiagnosis && !ICD10.test(c.primaryDiagnosis)
  const badSecondary = sec.some((d) => !ICD10.test(d))
  const secEqPrimary = sec.some((d) => up !== '' && d.toUpperCase() === up)
  const badProc = procs.some((p) => p !== '' && !ICD9.test(p))
  run('R05', badPrimary || badSecondary || secEqPrimary || badProc)

  const stale = !!c.resolutionSignature && c.resolutionSignature !== tariffSignature(c)
  run('R06', !c.inacbgCode || !c.tariffResolvedAt || stale)

  if (c.inacbgCode) {
    const isOutCode = c.inacbgCode.trim().endsWith('-0')
    run('R07', (c.careType === 'RAWAT_INAP' && isOutCode) || (c.careType === 'RAWAT_JALAN' && !isOutCode))
  }

  const hospitalTotal = sumTariff(c.hospitalTariff)
  const jknTotal = totalJKN(c.inacbgBaseTariff, c.specialCmg)
  run('R08', hospitalTotal !== 0 && hospitalTotal < jknTotal)

  const g = hospitalTotal - jknTotal
  run('R09', g > 0 && c.policyCeiling != null && c.policyCeiling > 0 && c.policyCeiling < g)

  const items = db.resumeItems.filter((r) => r.claimId === c.id)
  run('R10', items.length === 0 || items.some((r) => !r.reviewed))

  const di = deadlineInfo(c, today)
  run('R11', ['waspada', 'kritis', 'lewat'].includes(di.zone))

  run('R12', c.careClass === 'III' && (c.coverageType || '').toLowerCase().includes('selisih kelas'))

  run('R13', !c.declarationChecked || (c.status !== 'draft' && (!c.policyNo || !c.insuredName)))

  const ins = insurerByID(c.insurerId)
  if (ins && (c.status === 'draft' || c.status === 'ready_to_submit') && c.admissionDate) {
    run('R14', daysBetween(c.admissionDate, today) > ins.deadlineDays)
  }

  let blockers = 0, warnings = 0
  for (const f of findings) {
    if (f.overridden) continue
    if (f.severity === 'BLOCKER') blockers++
    else if (f.severity === 'WARNING') warnings++
  }
  const score = Math.max(0, 100 - blockers * 25 - warnings * 8)
  return { findings, score, hasOpenBlocker: blockers > 0 }
}

// ── Aging, DSO, matching, grouper ────────────────────────────────────────────
const BUCKETS = [
  { label: '0–30 hari', min: 0, max: 30 }, { label: '31–60 hari', min: 31, max: 60 },
  { label: '61–90 hari', min: 61, max: 90 }, { label: '> 90 hari', min: 91, max: Infinity },
]
function agingBuckets(outstanding, basisDate, today) {
  const buckets = BUCKETS.map((b) => ({ label: b.label, count: 0, amount: 0 }))
  let total = 0
  for (const id of Object.keys(outstanding)) {
    const age = daysBetween(basisDate[id], today)
    for (let i = 0; i < BUCKETS.length; i++) {
      if (age >= BUCKETS[i].min && age <= BUCKETS[i].max) {
        buckets[i].count++
        buckets[i].amount += outstanding[id]
        total += outstanding[id]
        break
      }
    }
  }
  return { buckets, total }
}
function dsoDays(pairs) {
  let sum = 0, n = 0
  for (const [a, b] of pairs) {
    if (b >= a) { sum += daysBetween(a, b); n++ }
  }
  return n === 0 ? null : Math.trunc(sum / n)
}

function matchScore(line, c, tolerance) {
  let score = 0
  const reasons = []
  const desc = (line.description || '').toUpperCase()
  if (line.description && c.sepNo && c.sepNo.length >= 6 && desc.includes(c.sepNo.slice(-6))) {
    score += 40; reasons.push('SEP pada deskripsi')
  }
  if (line.description && c.insuredName && desc.includes(c.insuredName.toUpperCase())) {
    score += 25; reasons.push('nama tertanggung')
  }
  if (line.description && c.policyNo && desc.includes(c.policyNo.toUpperCase())) {
    score += 20; reasons.push('no. polis')
  }
  const target = claimGap(c)
  if (target > 0 && Math.abs(line.amount - target) <= tolerance) {
    score += 15; reasons.push('nominal ± toleransi')
  }
  if (score > 100) score = 100
  return { score, reason: reasons.length ? reasons.join(', ') : 'tanpa indikator kuat' }
}

const GROUP_DESC = {
  I: 'GROUP JANTUN-ISKEMIK & PENYAKIT PENYERTA', J: 'GROUP RESPIRATORI INFEKSI/INFLAMASI',
  E: 'GROUP DIABETES & GANGGUAN METABOLIK', K: 'GROUP GIT & HEPATOBILIARI',
  N: 'GROUP GINJAL & SALURAN KEMIH', C: 'GROUP NEOPLASMA GANAS', S: 'GROUP TRAUMA TULANG & SENDI',
  A: 'GROUP INFEKSI & PARASIT', G: 'GROUP SARAF PUSAT', O: 'GROUP OBSTETRIK',
}
function mockGrouper(dx, careType, careClass, los, eklaimConfigured) {
  if (!eklaimConfigured) return { ok: false, error: 'EKLAIM_CONFIG_MISSING' }
  if (!/^[A-Z]\d{2}/i.test(dx || '')) return { ok: false, error: 'UNGROUPABLE' }
  const chapter = dx[0].toUpperCase()
  const inpatient = careType === 'RAWAT_INAP'
  const care = inpatient ? '1' : '3'
  const classDigit = careClass === 'I' ? '1' : careClass === 'II' ? '2' : '3'
  const severity = los > 6 ? '2' : '1'
  const code = inpatient
    ? `${chapter}-${care}-${severity}${classDigit}-${careClass}`
    : `${chapter}-${care}-1${severity}-0`
  const seed = (dx.charCodeAt(0) * 7 + dx.charCodeAt(1) * 13) % 900
  const mult = inpatient ? 4500000 : 750000
  return {
    ok: true, code, baseTariff: mult + seed * 5000 + los * 120000,
    description: GROUP_DESC[chapter] || 'GROUP INA-CBG TERPILIH (DEMO)',
  }
}

// ── Normalisasi body → struct (meniru decode JSON ke struct Go) ──────────────
const str = (v) => (v == null ? '' : String(v))
const arrStr = (v) => (Array.isArray(v) ? v.map(String) : [])
function toClaim(b) {
  const comp = (o, keys) => { const r = {}; keys.forEach((k) => { r[k] = num(o && o[k]) }); return r }
  return {
    id: str(b.id), hospitalId: str(b.hospitalId), sepNo: str(b.sepNo), bpjsCardNo: str(b.bpjsCardNo),
    mrn: str(b.mrn), patientName: str(b.patientName), dpjp: str(b.dpjp), participantType: str(b.participantType),
    careType: str(b.careType), careClass: str(b.careClass), sex: str(b.sex), dateOfBirth: str(b.dateOfBirth),
    admissionDate: str(b.admissionDate), dischargeDate: str(b.dischargeDate), los: num(b.los),
    primaryDiagnosis: str(b.primaryDiagnosis), secondaryDiagnoses: arrStr(b.secondaryDiagnoses), procedures: arrStr(b.procedures),
    inacbgCode: str(b.inacbgCode), inacbgDescription: str(b.inacbgDescription), inacbgBaseTariff: num(b.inacbgBaseTariff),
    specialCmg: comp(b.specialCmg, CMG_KEYS), hospitalTariff: comp(b.hospitalTariff, TARIFF_KEYS),
    tariffResolvedAt: str(b.tariffResolvedAt), declarationChecked: !!b.declarationChecked,
    insurerId: str(b.insurerId), policyNo: str(b.policyNo), insuredName: str(b.insuredName),
    insurerNameExtra: str(b.insurerNameExtra), insuranceType: str(b.insuranceType),
    insuranceMemberNo: str(b.insuranceMemberNo), coverageType: str(b.coverageType),
    policyCeiling: b.policyCeiling == null ? null : num(b.policyCeiling),
    status: str(b.status), createdAt: str(b.createdAt), updatedAt: str(b.updatedAt),
    submittedAt: b.submittedAt == null ? null : str(b.submittedAt),
    batchId: b.batchId == null ? null : str(b.batchId), createdBy: str(b.createdBy),
    scrubFindings: Array.isArray(b.scrubFindings) ? b.scrubFindings : [], readinessScore: num(b.readinessScore),
    rejectionReasonCode: str(b.rejectionReasonCode), rejectionReasonNote: str(b.rejectionReasonNote),
    resolutionSignature: str(b.resolutionSignature), statusHistory: Array.isArray(b.statusHistory) ? b.statusHistory : [],
  }
}
function toResume(b) {
  return {
    id: str(b.id), claimId: str(b.claimId), sep: str(b.sep), kategori: str(b.kategori), namaItem: str(b.namaItem),
    subKategori: str(b.subKategori), tanggal: str(b.tanggal), hasilCatatan: str(b.hasilCatatan),
    jumlah: num(b.jumlah), satuan: str(b.satuan), hargaSatuan: num(b.hargaSatuan), subtotal: num(b.subtotal),
    reviewed: !!b.reviewed,
  }
}

// ── Utilitas respons ─────────────────────────────────────────────────────────
const ok = (data, status = 200) => ({ status, data })
const fail = (status, message) => ({ status, data: { error: message } })
const rp = (n) => {
  let s = String(Math.abs(n))
  for (let i = s.length - 3; i > 0; i -= 3) s = s.slice(0, i) + '.' + s.slice(i)
  return (n < 0 ? 'Rp -' : 'Rp ') + s
}

// ── Handler: Auth ────────────────────────────────────────────────────────────
function handleLogin({ body }) {
  const u = userByEmail(body.email)
  if (!u || body.password !== DEMO_PASSWORD) {
    audit(null, 'LOGIN_FAILED', 'user', str(body.email), '', '')
    return fail(401, 'Email atau kata sandi salah')
  }
  if (u.mfaEnabled) return ok({ mfaRequired: true, email: u.email })
  audit(u, 'LOGIN', 'user', u.id, '', '')
  return ok({ token: 'mock.' + u.id, user: publicUser(u) })
}
function handleVerifyMFA({ body }) {
  const u = userByEmail(body.email)
  if (!u) return fail(401, 'Pengguna tidak ditemukan')
  if (body.code !== MFA_CODE) {
    audit(u, 'MFA_FAILED', 'user', u.id, '', '')
    return fail(401, 'Kode MFA salah (demo: 246810)')
  }
  audit(u, 'LOGIN_MFA', 'user', u.id, '', '')
  return ok({ token: 'mock.' + u.id, user: publicUser(u) })
}
const handleLogout = () => ok({ ok: true })
const handleMe = ({ user }) => ok({ user: publicUser(user), permissions: ROLE_PERMS[user.role] || [] })

function handleBootstrap({ user }) {
  const hospitals = user.hospitalId ? db.hospitals.filter((h) => h.id === user.hospitalId) : db.hospitals
  // Ringkasan klaim + work item untuk badge sidebar & pencarian global (Ctrl+K).
  const claims = db.claims.filter((c) => scopeOK(user, c)).map((c) => ({
    id: c.id, hospitalId: c.hospitalId, sepNo: c.sepNo, mrn: c.mrn,
    patientName: c.patientName, status: c.status, insurerId: c.insurerId,
  }))
  const workItems = db.workItems
    .filter((wi) => { const c = wi.claimId ? claimByID(wi.claimId) : null; return !(c && !scopeOK(user, c)) })
    .map((wi) => ({ id: wi.id, title: wi.title, status: wi.status, assignee: wi.assignee, role: wi.role, priority: wi.priority, claimId: wi.claimId }))
  return ok({
    hospitals, users: db.users.map(publicUser), insurers: db.insurers, rejectionCodes: db.rejectionCodes,
    scrubRules: db.scrubRules, regulations: db.regulations, notifications: db.notifications,
    partners: user.role === 'SUPER_ADMIN' ? db.partners : [], claimLabels: CLAIM_LABELS,
    componentLabels: COMPONENT_LABELS, specialCmgLabels: SPECIAL_CMG_LABELS, resumeCategories: RESUME_CATEGORIES,
    claims, workItems,
  })
}

// ── Handler: Dashboard, analitik, aging ──────────────────────────────────────
function insurerPerformance(u) {
  const rows = new Map()
  db.insurers.forEach((i) => rows.set(i.id, {
    insurerId: i.id, name: i.name, submitted: 0, approved: 0, rejected: 0,
    approvalRate: 0, rejectionRate: 0, avgDaysPay: null, receivedRp: 0,
  }))
  const payDays = {}
  for (const c of db.claims) {
    if (!scopeOK(u, c) || !c.insurerId) continue
    const row = rows.get(c.insurerId)
    if (!row) continue
    if (['submitted', 'approved', 'partially_approved', 'partially_paid', 'paid'].includes(c.status)) { row.submitted++; row.approved++ }
    else if (c.status === 'rejected') { row.submitted++; row.rejected++ }
    if (c.status === 'paid' && c.submittedAt) {
      const last = lastPaymentDate(c.id)
      if (last) (payDays[c.insurerId] = payDays[c.insurerId] || []).push(daysBetween(c.submittedAt, last))
    }
  }
  for (const p of db.payments) {
    const c = claimByID(p.claimId)
    if (c && scopeOK(u, c) && c.insurerId && rows.has(c.insurerId)) rows.get(c.insurerId).receivedRp += p.amount
  }
  const out = []
  for (const row of rows.values()) {
    if (row.submitted === 0 && row.receivedRp === 0) continue
    if (row.submitted > 0) {
      row.approvalRate = (row.approved / row.submitted) * 100
      row.rejectionRate = (row.rejected / row.submitted) * 100
    }
    const ds = payDays[row.insurerId]
    if (ds && ds.length) row.avgDaysPay = Math.trunc(ds.reduce((a, b) => a + b, 0) / ds.length)
    out.push(row)
  }
  return out
}

function handleDashboard({ user }) {
  const T = todayISO()
  const out = {
    eligibleClaims: 0, outsideBpjsRp: 0, receivedRp: 0, outstandingRp: 0, resolutionsToday: 0, nearDeadline: 0,
    blockersCount: 0, statusBreakdown: {}, nearDeadlineClaims: [], workQueueToday: [], agingBuckets: [],
    dso: null, insurerPerf: [], recentPayments: [],
  }
  const outstanding = {}, basis = {}, paidPairs = []
  for (const c of db.claims) {
    if (!scopeOK(user, c)) continue
    out.statusBreakdown[c.status] = (out.statusBreakdown[c.status] || 0) + 1
    const g = claimGap(c)
    if (c.participantType === 'NON_PBI' && g > 0 && c.status !== 'written_off') { out.eligibleClaims++; out.outsideBpjsRp += g }
    if (c.tariffResolvedAt && c.tariffResolvedAt.startsWith(T)) out.resolutionsToday++
    const di = deadlineInfo(c, T)
    if (['draft', 'ready_to_submit', 'submitted'].includes(c.status) && ['waspada', 'kritis', 'lewat'].includes(di.zone)) {
      out.nearDeadline++
      if (out.nearDeadlineClaims.length < 8) {
        out.nearDeadlineClaims.push({ id: c.id, patientName: c.patientName, status: c.status, deadline: di.deadline, daysLeft: di.daysLeft, zone: di.zone })
      }
    }
    const oc = evaluateClaim(c, T)
    if (oc.hasOpenBlocker && c.status !== 'paid' && c.status !== 'written_off') out.blockersCount++
    const o = outstandingOf(c)
    if (o > 0 && c.status !== 'written_off' && c.status !== 'rejected') {
      outstanding[c.id] = o
      basis[c.id] = orDefault(c.submittedAt, c.admissionDate)
    }
    if (c.submittedAt && c.status === 'paid') {
      const last = lastPaymentDate(c.id)
      if (last) paidPairs.push([c.submittedAt.slice(0, 10), last.slice(0, 10)])
    }
  }
  for (const p of db.payments) {
    const c = claimByID(p.claimId)
    if (c && scopeOK(user, c)) out.receivedRp += p.amount
  }
  out.outstandingRp = Object.values(outstanding).reduce((a, b) => a + b, 0)
  out.agingBuckets = agingBuckets(outstanding, basis, T).buckets
  out.dso = dsoDays(paidPairs)
  out.workQueueToday = db.workItems.filter((w) => ['New', 'Assigned', 'In Progress'].includes(w.status))
  out.insurerPerf = insurerPerformance(user)
  for (let k = db.payments.length - 1; k >= 0 && out.recentPayments.length < 6; k--) {
    const p = db.payments[k]
    const c = claimByID(p.claimId)
    if (c && scopeOK(user, c)) out.recentPayments.push(p)
  }
  return ok(out)
}

function handleAnalytics({ user }) {
  const T = todayISO()
  const now = new Date()
  const months = []
  for (let k = 11; k >= 0; k--) {
    const d = new Date(now.getFullYear(), now.getMonth() - k, 1)
    months.push(`${d.getFullYear()}-${pad(d.getMonth() + 1)}`)
  }
  const trend = months.map((m) => ({ month: m, claims: 0, gap: 0 }))
  const mIdx = Object.fromEntries(months.map((m, i) => [m, i]))
  const mk = () => ({ claims: 0, gap: 0, topUp: 0, uniqueGroups: new Set(), belowInacbg: 0 })
  const ri = mk(), rj = mk()
  const segments = {}
  const deadlineWindow = []
  for (const c of db.claims) {
    if (!scopeOK(user, c)) continue
    const idx = mIdx[(c.createdAt || '').slice(0, 7)]
    const g = claimGap(c)
    if (idx !== undefined) { trend[idx].claims++; trend[idx].gap += g }
    const hosp = sumTariff(c.hospitalTariff)
    const below = hosp > 0 && hosp < totalJKN(c.inacbgBaseTariff, c.specialCmg)
    const t = c.careType === 'RAWAT_INAP' ? ri : rj
    t.claims++; t.gap += g; t.topUp += sumCmg(c.specialCmg); t.uniqueGroups.add(c.inacbgCode)
    if (below) t.belowInacbg++
    const seg = c.participantType === 'NON_PBI' ? 'Kelas ' + c.careClass + ' Non-PBI' : 'PBI (tidak eligible)'
    segments[seg] = (segments[seg] || 0) + 1
    const di = deadlineInfo(c, T)
    if (['draft', 'ready_to_submit', 'submitted'].includes(c.status) && di.daysLeft <= 30 && di.daysLeft >= -365) {
      deadlineWindow.push({ id: c.id, patientName: c.patientName, deadline: di.deadline, daysLeft: di.daysLeft, critical: di.daysLeft <= 7 })
    }
  }
  const fin = (t) => ({ claims: t.claims, gap: t.gap, topUp: t.topUp, uniqueGroups: t.uniqueGroups.size, belowInacbg: t.belowInacbg })
  return ok({ trend, inpatient: fin(ri), outpatient: fin(rj), segments, deadlineWindow })
}

function handleAging({ user, query }) {
  const basisMode = query.get('basis') === 'service' ? 'service' : 'submission'
  const T = todayISO()
  const outstanding = {}, basisDate = {}, drill = [], paidPairs = []
  for (const c of db.claims) {
    if (!scopeOK(user, c)) continue
    const o = outstandingOf(c)
    if (o <= 0 || c.status === 'written_off' || c.status === 'rejected') continue
    const bd = basisMode === 'service' ? c.admissionDate : orDefault(c.submittedAt, c.admissionDate)
    outstanding[c.id] = o
    basisDate[c.id] = bd
    const ins = insurerByID(c.insurerId)
    drill.push({ id: c.id, patientName: c.patientName, sepNo: c.sepNo, insurer: ins ? ins.name : '', status: c.status, amount: o, ageDays: daysBetween(bd, T) })
  }
  const { buckets, total } = agingBuckets(outstanding, basisDate, T)
  drill.sort((a, b) => b.ageDays - a.ageDays)
  for (const c of db.claims) {
    if (scopeOK(user, c) && c.status === 'paid' && c.submittedAt) {
      const last = lastPaymentDate(c.id)
      if (last) paidPairs.push([c.submittedAt.slice(0, 10), last.slice(0, 10)])
    }
  }
  const over90 = (buckets.find((b) => b.label === '> 90 hari') || { amount: 0 }).amount
  return ok({
    buckets, total, dso: dsoDays(paidPairs), pctOver90: total > 0 ? (over90 / total) * 100 : 0,
    drilldown: drill, payerPerf: insurerPerformance(user),
  })
}

// ── Handler: Klaim ───────────────────────────────────────────────────────────
function withScrub(c, T) {
  const cp = { ...c }
  const oc = evaluateClaim(cp, T)
  cp.scrubFindings = oc.findings
  cp.readinessScore = oc.score
  return cp
}

function handleClaimList({ user, query }) {
  const T = todayISO()
  const status = query.get('status') || ''
  const careType = query.get('careType') || ''
  const search = (query.get('q') || '').toLowerCase()
  const batchId = query.get('batchId') || ''
  let page = parseInt(query.get('page'), 10)
  if (!(page >= 1)) page = 1
  let pageSize = parseInt(query.get('pageSize'), 10)
  if (!(pageSize >= 1 && pageSize <= 200)) pageSize = 15
  const rows = []
  for (const c of db.claims) {
    if (!scopeOK(user, c)) continue
    if (status && c.status !== status) continue
    if (careType && c.careType !== careType) continue
    if (batchId === 'none' && c.batchId) continue
    if (batchId && batchId !== 'none' && c.batchId !== batchId) continue
    if (search && !(c.sepNo + ' ' + c.patientName + ' ' + c.mrn).toLowerCase().includes(search)) continue
    rows.push(withScrub(c, T))
  }
  const total = rows.length
  const start = Math.min((page - 1) * pageSize, total)
  const end = Math.min(start + pageSize, total)
  return ok({ rows: rows.slice(start, end), total, page, pageSize })
}

function handleClaimGet({ user, params }) {
  const c = claimByID(params[0])
  if (!c || !scopeOK(user, c)) return fail(404, 'Klaim tidak ditemukan')
  return ok({
    claim: withScrub(c, todayISO()), resume: db.resumeItems.filter((r) => r.claimId === c.id),
    payments: db.payments.filter((p) => p.claimId === c.id), gap: claimGap(c), outstanding: outstandingOf(c),
    hospital: hospitalByID(c.hospitalId), insurer: insurerByID(c.insurerId),
  })
}

function handleClaimCreate({ user, body }) {
  const c = toClaim(body)
  if (!c.hospitalId) c.hospitalId = user.hospitalId
  if (user.hospitalId && c.hospitalId !== user.hospitalId) return fail(403, 'Anda hanya dapat membuat klaim untuk RS Anda')
  if (!hospitalByID(c.hospitalId)) return fail(400, 'Kode Faskes/RS tidak dikenal')
  c.id = nextID('CLM')
  c.status = 'draft'
  c.createdBy = user.name
  c.createdAt = nowStamp()
  c.updatedAt = c.createdAt
  c.los = losOf(c.admissionDate, c.dischargeDate)
  c.statusHistory = [{ at: nowStamp(), actor: user.name, from: '', to: 'draft', note: 'Klaim dibuat' }]
  db.claims.push(c)
  audit(user, 'CREATE_CLAIM', 'claim', c.id, '', c.patientName)
  return ok(c, 201)
}

function handleClaimUpdate({ user, params, body }) {
  const c = claimByID(params[0])
  if (!c || !scopeOK(user, c)) return fail(404, 'Klaim tidak ditemukan')
  if (!['draft', 'ready_to_submit', 'disputed'].includes(c.status)) {
    return fail(409, 'Klaim dengan status ' + CLAIM_LABELS[c.status] + ' tidak dapat diubah')
  }
  const before = JSON.stringify({ status: c.status, patient: c.patientName })
  const n = toClaim(body)
  n.id = c.id; n.status = c.status; n.statusHistory = c.statusHistory
  n.createdBy = c.createdBy; n.createdAt = c.createdAt
  n.scrubFindings = c.scrubFindings; n.readinessScore = c.readinessScore; n.batchId = c.batchId
  n.los = losOf(n.admissionDate, n.dischargeDate)
  n.updatedAt = nowStamp()
  Object.keys(c).forEach((k) => delete c[k])
  Object.assign(c, n)
  audit(user, 'UPDATE_CLAIM', 'claim', c.id, before, JSON.stringify({ status: c.status, patient: c.patientName }))
  return ok(c)
}

function handleClaimDelete({ user, params }) {
  const c = claimByID(params[0])
  if (!c || !scopeOK(user, c)) return fail(404, 'Klaim tidak ditemukan')
  const removed = db.resumeItems.filter((r) => r.claimId === c.id).length
  db.resumeItems = db.resumeItems.filter((r) => r.claimId !== c.id)
  db.claims = db.claims.filter((x) => x.id !== c.id)
  audit(user, 'DELETE_CLAIM', 'claim', c.id, c.patientName, 'menghapus ' + removed + ' item resume')
  return ok({ ok: true })
}

function handleResolveTariff({ user, params }) {
  const c = claimByID(params[0])
  if (!c || !scopeOK(user, c)) return fail(404, 'Klaim tidak ditemukan')
  const h = hospitalByID(c.hospitalId)
  const g = mockGrouper(c.primaryDiagnosis, c.careType, c.careClass, c.los, !!(h && h.eklaimConfigured))
  if (!g.ok) {
    audit(user, 'RESOLVE_TARIFF_FAILED', 'claim', c.id, '', g.error)
    return fail(422, 'Grouper gagal: ' + g.error)
  }
  c.inacbgCode = g.code
  c.inacbgDescription = g.description
  c.inacbgBaseTariff = g.baseTariff
  c.tariffResolvedAt = nowStamp()
  c.resolutionSignature = tariffSignature(c)
  c.updatedAt = nowStamp()
  audit(user, 'RESOLVE_TARIFF', 'claim', c.id, '', g.code + ' @ ' + g.baseTariff)
  return ok({ claim: c, grouper: g })
}

function handleTransition({ user, params, body }) {
  const c = claimByID(params[0])
  if (!c || !scopeOK(user, c)) return fail(404, 'Klaim tidak ditemukan')
  const to = body.to
  if (!canTransition(c.status, to)) return fail(409, 'Transisi tidak sah: ' + c.status + ' → ' + to)
  if (to === 'ready_to_submit') {
    const oc = evaluateClaim(c, todayISO())
    if (oc.hasOpenBlocker) {
      const ids = oc.findings.filter((f) => f.severity === 'BLOCKER' && !f.overridden).map((f) => f.ruleId)
      return fail(409, 'Klaim masih memiliki blocker terbuka (' + ids.join(', ') + ') — selesaikan dulu atau minta override Admin RS')
    }
    if (!c.declarationChecked) return fail(409, 'Pernyataan kebenaran data belum dicentang')
  } else if (to === 'submitted') {
    if (!hasPermission(user, 'APPROVE')) return fail(403, 'Pengajuan klaim memerlukan peran Reviewer/Admin (maker-checker)')
    if (evaluateClaim(c, todayISO()).hasOpenBlocker) return fail(409, 'Klaim masih memiliki blocker terbuka — tidak dapat diajukan')
    c.submittedAt = nowStamp()
  } else if (to === 'rejected') {
    if (!body.rejectionCode) return fail(400, 'Alasan penolakan terstruktur wajib diisi')
    c.rejectionReasonCode = body.rejectionCode
    c.rejectionReasonNote = str(body.note)
  }
  const before = c.status
  c.status = to
  c.updatedAt = nowStamp()
  c.statusHistory = c.statusHistory || []
  c.statusHistory.push({ at: nowStamp(), actor: user.name, from: before, to, note: str(body.note) })
  audit(user, 'STATUS_TRANSITION', 'claim', c.id, before, to + ' — ' + str(body.note))
  return ok(c)
}

// ── Handler: Scrubber ────────────────────────────────────────────────────────
function handleScrubClaim({ user, params }) {
  const c = claimByID(params[0])
  if (!c || !scopeOK(user, c)) return fail(404, 'Klaim tidak ditemukan')
  const oc = evaluateClaim(c, todayISO())
  c.scrubFindings = oc.findings
  c.readinessScore = oc.score
  return ok(oc)
}
function handleScrubRuleUpdate({ user, params, body }) {
  const r = db.scrubRules.find((x) => x.id === params[0])
  if (!r) return fail(404, 'Aturan tidak ditemukan')
  r.enabled = !!body.enabled
  audit(user, 'SCRUB_RULE_TOGGLE', 'scrub_rule', r.id, String(!r.enabled), String(r.enabled))
  return ok(r)
}
function handleScrubOverride({ user, body }) {
  const key = body.claimId + '|' + body.ruleId
  if (body.remove) delete db.overrides[key]
  else {
    if (!body.reason) return fail(400, 'Alasan override wajib diisi (tercatat di audit trail)')
    db.overrides[key] = true
  }
  audit(user, body.remove ? 'SCRUB_OVERRIDE_REMOVED' : 'SCRUB_OVERRIDE', 'claim|rule', key, '', str(body.reason))
  const c = claimByID(body.claimId)
  if (!c) return fail(404, 'Klaim tidak ditemukan')
  const oc = evaluateClaim(c, todayISO())
  c.scrubFindings = oc.findings
  c.readinessScore = oc.score
  return ok(oc)
}
function handleScrubTopCauses({ user }) {
  const T = todayISO()
  const counts = {}
  for (const c of db.claims) {
    if (!scopeOK(user, c)) continue
    for (const f of evaluateClaim(c, T).findings) if (!f.overridden) counts[f.ruleId] = (counts[f.ruleId] || 0) + 1
  }
  const out = Object.entries(counts).map(([id, count]) => {
    const rl = db.scrubRules.find((r) => r.id === id) || {}
    return { ruleId: id, count, name: rl.name || '', severity: rl.severity || '', category: rl.category || '' }
  })
  out.sort((a, b) => b.count - a.count || a.ruleId.localeCompare(b.ruleId))
  return ok(out)
}

// ── Handler: Resume medis ────────────────────────────────────────────────────
function handleResumeBulk({ user, body }) {
  const c = claimByID(body.claimId)
  if (!c || !scopeOK(user, c)) return fail(404, 'Klaim tidak ditemukan')
  let added = 0, skipped = 0
  for (const raw of body.items || []) {
    const it = toResume(raw)
    if (it.sep && it.sep !== c.sepNo) { skipped++; continue }
    if (it.subtotal === 0) it.subtotal = it.jumlah * it.hargaSatuan
    it.id = nextID('RS')
    it.claimId = c.id
    db.resumeItems.push(it)
    added++
  }
  audit(user, 'RESUME_UPLOAD', 'claim', c.id, '', `${added} item (skip ${skipped})`)
  return ok({ added, skipped })
}
function handleResumeUpdate({ user, params, body }) {
  const i = db.resumeItems.findIndex((r) => r.id === params[0])
  if (i < 0) return fail(404, 'Item tidak ditemukan')
  const old = db.resumeItems[i]
  const c = claimByID(old.claimId)
  if (!c || !scopeOK(user, c)) return fail(403, 'Di luar cakupan RS Anda')
  const n = toResume(body)
  if (n.subtotal === 0) n.subtotal = n.jumlah * n.hargaSatuan
  n.id = old.id
  n.claimId = old.claimId
  db.resumeItems[i] = n
  audit(user, 'RESUME_ITEM_EDIT', 'resume_item', n.id, '', n.namaItem)
  return ok(n)
}
function handleResumeDelete({ user, params }) {
  const it = db.resumeItems.find((r) => r.id === params[0])
  if (!it) return fail(404, 'Item tidak ditemukan')
  db.resumeItems = db.resumeItems.filter((r) => r.id !== it.id)
  audit(user, 'RESUME_ITEM_DELETE', 'resume_item', it.id, it.namaItem, '')
  return ok({ ok: true })
}

// ── Handler: Batch ───────────────────────────────────────────────────────────
const findBatch = (id) => db.batches.find((b) => b.id === id) || null

function handleBatchList({ user }) {
  return ok(db.batches
    .filter((b) => !user.hospitalId || b.hospitalId === user.hospitalId)
    .map((b) => {
      const cs = db.claims.filter((c) => c.batchId === b.id)
      return { batch: b, claimCount: cs.length, totalGap: cs.reduce((s, c) => s + claimGap(c), 0) }
    }))
}
function handleBatchCreate({ user, body }) {
  const b = {
    id: '', name: str(body.name), period: str(body.period), hospitalId: str(body.hospitalId),
    createdAt: '', createdBy: '', note: str(body.note),
  }
  if (!b.hospitalId) b.hospitalId = user.hospitalId
  if (user.hospitalId && b.hospitalId !== user.hospitalId) return fail(403, 'Di luar cakupan RS Anda')
  b.id = nextID('BT')
  b.createdBy = user.name
  b.createdAt = nowStamp()
  db.batches.push(b)
  audit(user, 'CREATE_BATCH', 'batch', b.id, '', b.name)
  return ok(b, 201)
}
function handleBatchGet({ user, params }) {
  const b = findBatch(params[0])
  if (!b || (user.hospitalId && b.hospitalId !== user.hospitalId)) return fail(404, 'Batch tidak ditemukan')
  const T = todayISO()
  const d = { batch: b, claims: [], unbatched: [], totalGap: 0, blockerHits: 0 }
  for (const c of db.claims) {
    if (c.hospitalId !== b.hospitalId) continue
    if (c.batchId === b.id) {
      if (evaluateClaim(c, T).hasOpenBlocker) d.blockerHits++
      d.totalGap += claimGap(c)
      d.claims.push(c)
    } else if (!c.batchId && (c.status === 'ready_to_submit' || c.status === 'draft')) d.unbatched.push(c)
  }
  return ok(d)
}
function handleBatchDelete({ user, params }) {
  const id = params[0]
  db.claims.forEach((c) => { if (c.batchId === id) c.batchId = null })
  db.batches = db.batches.filter((b) => b.id !== id)
  audit(user, 'DELETE_BATCH', 'batch', id, '', 'klaim dilepas (tidak dihapus)')
  return ok({ ok: true })
}
function handleBatchAddClaims({ user, params, body }) {
  const b = findBatch(params[0])
  if (!b || (user.hospitalId && b.hospitalId !== user.hospitalId)) return fail(404, 'Batch tidak ditemukan')
  const T = todayISO()
  let added = 0, blocked = 0
  for (const cid of body.claimIds || []) {
    const c = claimByID(cid)
    if (!c || c.hospitalId !== b.hospitalId || c.batchId) continue
    if (evaluateClaim(c, T).hasOpenBlocker) { blocked++; continue }
    c.batchId = b.id
    added++
  }
  audit(user, 'BATCH_ADD_CLAIMS', 'batch', b.id, '', `+${added} (ditolak blocker: ${blocked})`)
  return ok({ added, blocked })
}
function handleBatchRemoveClaim({ user, params }) {
  const [id, cid] = params
  const c = claimByID(cid)
  if (c && c.batchId === id) {
    c.batchId = null
    audit(user, 'BATCH_REMOVE_CLAIM', 'batch', id, cid, '')
  }
  return ok({ ok: true })
}

// ── Handler: Pembayaran & rekonsiliasi ───────────────────────────────────────
function handlePaymentList({ user }) {
  const out = []
  for (let k = db.payments.length - 1; k >= 0; k--) {
    const p = db.payments[k]
    const c = claimByID(p.claimId)
    if (!c || !scopeOK(user, c)) continue
    out.push({ payment: p, claim: { id: c.id, patientName: c.patientName, sepNo: c.sepNo, status: c.status, gap: claimGap(c), insurerId: c.insurerId } })
    if (out.length >= 100) break
  }
  return ok(out)
}
function handlePaymentCreate({ user, body }) {
  const c = claimByID(body.claimId)
  if (!c || !scopeOK(user, c)) return fail(404, 'Klaim tidak ditemukan')
  if (!['approved', 'partially_approved', 'partially_paid', 'paid'].includes(c.status)) {
    return fail(409, 'Pembayaran hanya untuk klaim yang sudah disetujui/dibayar sebagian')
  }
  const amount = num(body.amount)
  if (amount <= 0) return fail(400, 'Nominal pembayaran harus > 0')
  const p = {
    id: nextID('PAY'), claimId: c.id, receivedDate: str(body.receivedDate) || todayISO(), amount,
    payer: str(body.payer), transferRef: str(body.transferRef), notes: str(body.notes),
    createdBy: user.name, createdAt: nowStamp(), allocation: str(body.allocation) || 'GAP',
  }
  const g = claimGap(c)
  let warning = ''
  if (paidAmount(c.id) + amount > g && g > 0) warning = 'Lebih bayar: total pembayaran melebihi nilai gap'
  if (p.transferRef && db.payments.some((x) => x.transferRef === p.transferRef)) {
    warning = 'Potensi duplikasi: referensi transfer sudah pernah dicatat'
  }
  db.payments.push(p)
  recomputePaymentStatus(c)
  c.updatedAt = nowStamp()
  audit(user, 'PAYMENT_RECORD', 'claim', c.id, '', `${p.id} → ${p.amount} (${p.payer})`)
  db.notifications.push({
    id: nextID('NT'), at: nowStamp(), channel: 'IN_APP', template: 'PAYMENT_RECEIVED', claimId: c.id,
    recipient: 'finance@nusantara.id', message: `Pembayaran diterima untuk klaim ${c.patientName} (${p.id})`, status: 'TERKIRIM',
  })
  return ok({ payment: p, warning, claimStatus: c.status, totalPaid: paidAmount(c.id), gap: g }, 201)
}
function handlePaymentDelete({ user, params }) {
  const i = db.payments.findIndex((p) => p.id === params[0])
  if (i < 0) return fail(404, 'Pembayaran tidak ditemukan')
  const p = db.payments[i]
  const c = claimByID(p.claimId)
  if (!c || !scopeOK(user, c)) return fail(403, 'Di luar cakupan RS Anda')
  db.payments.splice(i, 1)
  if (c.status === 'paid' || c.status === 'partially_paid') c.status = paidAmount(c.id) > 0 ? 'partially_paid' : 'approved'
  c.updatedAt = nowStamp()
  audit(user, 'PAYMENT_UNDO', 'claim', c.id, `${p.id} ${p.amount}`, 'status → ' + c.status)
  return ok({ ok: true, claimStatus: c.status })
}

function autoMatchLine(bl) {
  let best = null, bestScore = 0, bestReason = ''
  for (const c of db.claims) {
    if (!['approved', 'partially_approved', 'partially_paid'].includes(c.status)) continue
    const ms = matchScore(bl, c, 50000)
    if (ms.score > bestScore) { best = c; bestScore = ms.score; bestReason = ms.reason }
  }
  if (best) {
    bl.matchScore = bestScore
    let note = `skor ${bestScore} (${bestReason})`
    if (bestScore >= 75) { bl.matchResult = 'matched'; bl.matchClaimId = best.id; note = 'auto-match: ' + bestReason }
    else if (bestScore >= 40) bl.matchResult = 'needs_review'
    bl.matchNote = note
  }
}
function handleBankImportCreate({ user, body }) {
  const lines = Array.isArray(body.lines) ? body.lines : []
  const imp = { id: nextID('BI'), bankName: str(body.bankName), periodLabel: str(body.periodLabel), importedAt: nowStamp(), lineCount: lines.length }
  db.bankImports.push(imp)
  let matched = 0, review = 0, unmatched = 0
  for (const ln of lines) {
    const bl = {
      id: nextID('BL'), importId: imp.id, valueDate: str(ln.valueDate), description: str(ln.description),
      amount: num(ln.amount), reference: str(ln.reference), matchResult: 'unmatched', matchClaimId: null,
      matchScore: 0, matchNote: null, confirmed: false,
    }
    autoMatchLine(bl)
    if (bl.matchResult === 'matched') matched++
    else if (bl.matchResult === 'needs_review') review++
    else unmatched++
    db.bankLines.push(bl)
  }
  audit(user, 'IMPORT_BANK_STATEMENT', 'bank_import', imp.id, '', `${imp.bankName} — ${imp.lineCount} baris (auto: ${matched} match, ${review} review, ${unmatched} unmatched)`)
  return ok({ import: imp, matched, needsReview: review, unmatched }, 201)
}
const handleBankImportsList = () => ok(db.bankImports)
function handleBankLines({ query }) {
  const importId = query.get('importId') || ''
  const out = []
  for (const bl of db.bankLines) {
    if (importId && bl.importId !== importId) continue
    const m = { line: bl }
    if (bl.matchClaimId) {
      const c = claimByID(bl.matchClaimId)
      if (c) m.claim = { id: c.id, patientName: c.patientName, sepNo: c.sepNo, status: c.status, gap: claimGap(c), paid: paidAmount(c.id) }
    }
    out.push(m)
  }
  return ok(out)
}
function handleBankLineCandidates({ params }) {
  const bl = db.bankLines.find((x) => x.id === params[0])
  if (!bl) return fail(404, 'Baris mutasi tidak ditemukan')
  const out = []
  for (const c of db.claims) {
    if (!['approved', 'partially_approved', 'partially_paid', 'paid'].includes(c.status)) continue
    const ms = matchScore(bl, c, 50000)
    if (ms.score >= 25) {
      out.push({ claimId: c.id, patientName: c.patientName, sepNo: c.sepNo, status: c.status, gap: claimGap(c), paid: paidAmount(c.id), score: ms.score, reason: ms.reason })
    }
  }
  out.sort((a, b) => b.score - a.score)
  return ok(out)
}
function handleBankLineConfirm({ user, params, body }) {
  const bl = db.bankLines.find((x) => x.id === params[0])
  if (!bl) return fail(404, 'Baris mutasi tidak ditemukan')
  const c = claimByID(body.claimId)
  if (!c) return fail(404, 'Klaim tidak ditemukan')
  const warnings = []
  if (paidAmount(c.id) > 0) warnings.push('Klaim sudah memiliki pembayaran tercatat — potensi pembayaran ganda')
  if (bl.amount !== claimGap(c)) {
    const d = claimGap(c) - bl.amount
    warnings.push(d > 0 ? 'Kurang bayar: nominal mutasi ' + rp(d) + ' di bawah nilai gap' : 'Lebih bayar: nominal mutasi ' + rp(-d) + ' di atas nilai gap')
  }
  if (warnings.length && !body.force) return { status: 409, data: { warnings } }
  const pay = {
    id: nextID('PAY'), claimId: c.id, receivedDate: bl.valueDate, amount: bl.amount, payer: 'Mutasi Bank',
    transferRef: bl.reference, notes: 'Konfirmasi rekonsiliasi ' + bl.id, createdBy: user.name, createdAt: nowStamp(), allocation: 'GAP',
  }
  db.payments.push(pay)
  bl.confirmed = true
  bl.matchResult = 'matched'
  bl.matchClaimId = c.id
  recomputePaymentStatus(c)
  audit(user, 'RECON_CONFIRM', 'bank_line', bl.id, '', `→ ${c.id} / ${pay.id}`)
  return ok({ ok: true, payment: pay, claimStatus: c.status })
}
function handleBankLineUnconfirm({ user, params }) {
  const bl = db.bankLines.find((x) => x.id === params[0])
  if (!bl || !bl.confirmed || !bl.matchClaimId) return fail(409, 'Baris ini tidak sedang terkonfirmasi')
  let removed = ''
  db.payments = db.payments.filter((p) => {
    if (p.claimId === bl.matchClaimId && (p.notes || '').includes(bl.id)) { removed = p.id; return false }
    return true
  })
  const c = claimByID(bl.matchClaimId)
  if (c && (c.status === 'paid' || c.status === 'partially_paid')) c.status = paidAmount(c.id) > 0 ? 'partially_paid' : 'approved'
  bl.confirmed = false
  bl.matchNote = 'undo oleh ' + user.name
  audit(user, 'RECON_UNDO', 'bank_line', bl.id, removed, 'dibatalkan')
  return ok({ ok: true })
}

// ── Handler: KAPJ ────────────────────────────────────────────────────────────
function handleKapjSimulate({ user, body }) {
  const raw = (body && body.params) || {}
  let p = {
    bpjsSharePct: num(raw.bpjsSharePct), coPayEnabled: !!raw.coPayEnabled, patientCoPayPct: num(raw.patientCoPayPct),
    outpatientCoPayCap: num(raw.outpatientCoPayCap), inpatientCoPayCap: num(raw.inpatientCoPayCap),
  }
  if (p.bpjsSharePct <= 0) p = { ...DEFAULT_KAPJ }
  const rows = []
  const totals = { totalHospitalBill: 0, bpjsPays: 0, insurerPays: 0, patientPays: 0 }
  for (const c of db.claims) {
    if (!scopeOK(user, c) || c.status === 'written_off' || c.status === 'rejected') continue
    const bill = sumTariff(c.hospitalTariff)
    const jkn = totalJKN(c.inacbgBaseTariff, c.specialCmg)
    if (bill <= 0 && jkn <= 0) continue
    const k = kapjSplit(bill, jkn, c.policyCeiling, c.careType, p)
    const ins = insurerByID(c.insurerId)
    rows.push({
      claimId: c.id, patientName: c.patientName, insurer: ins ? ins.name : '', totalHospitalBill: bill,
      bpjsPays: k.bpjsPays, insurerPays: k.insurerPays, patientPays: k.patientPays, status: k.status,
    })
    totals.totalHospitalBill += bill; totals.bpjsPays += k.bpjsPays; totals.insurerPays += k.insurerPays; totals.patientPays += k.patientPays
  }
  return ok({
    params: p, rows, totals,
    disclaimer: 'Hasil simulasi bersifat indikatif, bukan angka final. Co-pay bersifat opsional sesuai produk. Plafon kosong diperlakukan tak terbatas sehingga angka dapat lebih tinggi. Regulasi dalam masa transisi hingga 22 Desember 2026. [VERIFIKASI REGULASI]',
  })
}

// ── Handler: Antrean kerja ───────────────────────────────────────────────────
function handleWorkItemList({ user }) {
  return ok(db.workItems
    .filter((wi) => { const c = wi.claimId ? claimByID(wi.claimId) : null; return !(c && !scopeOK(user, c)) })
    .map((wi) => ({ workItem: wi })))
}
function handleWorkItemCreate({ user, body }) {
  const wi = {
    id: nextID('WI'), title: str(body.title), source: str(body.source), claimId: body.claimId == null ? null : str(body.claimId),
    assignee: body.assignee == null ? null : str(body.assignee), role: body.role == null ? null : str(body.role),
    dueDate: str(body.dueDate) || addDays(todayISO(), 3), priority: str(body.priority) || 'Sedang',
    status: str(body.status) || 'New', comments: [], history: [], createdAt: nowStamp(),
  }
  wi.history = [{ id: nextID('WH'), at: nowStamp(), actor: user.name, action: 'Dibuat' }]
  db.workItems.push(wi)
  audit(user, 'CREATE_WORK_ITEM', 'work_item', wi.id, '', wi.title)
  return ok(wi, 201)
}
function handleWorkItemUpdate({ user, params, body }) {
  const wi = db.workItems.find((w) => w.id === params[0])
  if (!wi) return fail(404, 'Work item tidak ditemukan')
  const changes = []
  if (body.status != null && body.status !== wi.status) { changes.push(`status ${wi.status} → ${body.status}`); wi.status = body.status }
  if (body.assignee != null && (wi.assignee == null || body.assignee !== wi.assignee)) {
    const us = userByID(body.assignee)
    changes.push('ditugaskan ke ' + (us ? us.name : body.assignee))
    wi.assignee = body.assignee
    if (wi.status === 'New') wi.status = 'Assigned'
  }
  if (body.priority != null && body.priority !== wi.priority) { changes.push(`prioritas ${wi.priority} → ${body.priority}`); wi.priority = body.priority }
  if (body.dueDate != null && body.dueDate !== wi.dueDate) { changes.push('tenggat → ' + body.dueDate); wi.dueDate = body.dueDate }
  if (changes.length) {
    wi.history = wi.history || []
    wi.history.push({ id: nextID('WH'), at: nowStamp(), actor: user.name, action: changes.join('; ') })
    audit(user, 'UPDATE_WORK_ITEM', 'work_item', wi.id, '', changes.join('; '))
  }
  return ok(wi)
}
function handleWorkItemComment({ user, params, body }) {
  const wi = db.workItems.find((w) => w.id === params[0])
  if (!wi) return fail(404, 'Work item tidak ditemukan')
  wi.comments = wi.comments || []
  wi.history = wi.history || []
  wi.comments.push({ id: nextID('WC'), author: user.name, text: str(body.text), at: nowStamp() })
  wi.history.push({ id: nextID('WH'), at: nowStamp(), actor: user.name, action: 'Komentar ditambahkan' })
  audit(user, 'WORK_ITEM_COMMENT', 'work_item', wi.id, '', str(body.text))
  return ok(wi)
}

// ── Handler: Audit trail ─────────────────────────────────────────────────────
function handleAuditList({ user, query }) {
  const scope = query.get('scope') || ''
  const search = (query.get('q') || '').toLowerCase()
  const failedOnly = query.get('failed') === '1'
  let page = parseInt(query.get('page'), 10)
  if (!(page >= 1)) page = 1
  const rows = []
  for (let k = db.auditLog.length - 1; k >= 0; k--) {
    const e = db.auditLog[k]
    if (scope && e.scope !== scope) continue
    if (failedOnly && !((e.statusCode || 0) >= 400)) continue
    if (search && !(`${e.actor} ${e.action} ${e.entityId} ${e.ip} ${e.partnerName || ''}`).toLowerCase().includes(search)) continue
    if (user.hospitalId && e.scope === 'ENGINE_API') {
      if (!db.apiIntakes.some((ai) => ai.hospitalId === user.hospitalId && e.entityId === ai.id)) continue
    }
    rows.push(e)
    if (rows.length >= page * 100) break
  }
  const start = (page - 1) * 100
  const end = Math.min(start + 100, rows.length)
  const ipSets = {}
  for (const e of db.auditLog) {
    if (e.scope !== 'ENGINE_API') continue
    ;(ipSets[e.partnerName] = ipSets[e.partnerName] || new Set()).add(e.ip)
  }
  const multiIp = Object.entries(ipSets).filter(([, s]) => s.size > 1).map(([partner, s]) => ({ partner, ips: [...s], count: s.size }))
  const ipAll = new Set(rows.map((e) => e.ip))
  return ok({
    rows: rows.slice(start, end),
    summary: { total: rows.length, rejected: rows.filter((e) => (e.statusCode || 0) >= 400).length, uniqueIps: ipAll.size },
    multiIp,
  })
}

// ── Handler: Pengaturan ──────────────────────────────────────────────────────
const initialsOf = (name) => (name.length >= 2 ? name[0] + name[name.length - 1] : name)

function handleHospitalCreate({ user, body }) {
  const kode = str(body.kodeFaskes)
  if (db.hospitals.some((h) => h.kodeFaskes === kode)) return fail(409, 'Kode Faskes sudah terdaftar')
  const h = {
    id: nextID('h'), kodeFaskes: kode, name: str(body.name), city: str(body.city), class: str(body.class),
    active: !!body.active, eklaimConfigured: !!body.eklaimConfigured, registeredAt: todayISO(),
  }
  if (!h.name) { h.name = `RS (Kode ${kode}) — NAMA PERLU DIVERIFIKASI`; h.eklaimConfigured = false }
  db.hospitals.push(h)
  audit(user, 'REGISTER_HOSPITAL', 'hospital', h.id, '', kode)
  return ok(h, 201)
}
function handleHospitalUpdate({ user, params, body }) {
  const i = db.hospitals.findIndex((h) => h.id === params[0])
  if (i < 0) return fail(404, 'RS tidak ditemukan')
  const old = db.hospitals[i]
  const h = {
    id: old.id, kodeFaskes: str(body.kodeFaskes), name: str(body.name), city: str(body.city), class: str(body.class),
    active: !!body.active, eklaimConfigured: !!body.eklaimConfigured, registeredAt: str(body.registeredAt),
  }
  db.hospitals[i] = h
  audit(user, 'UPDATE_HOSPITAL', 'hospital', h.id, '', h.name)
  return ok(h)
}
function handleUserCreate({ user, body }) {
  if (userByEmail(body.email)) return fail(409, 'Email sudah terdaftar')
  const u = {
    id: nextID('u'), name: str(body.name), email: str(body.email), role: str(body.role),
    hospitalId: user.hospitalId ? user.hospitalId : str(body.hospitalId), mfaEnabled: !!body.mfaEnabled, initials: '',
  }
  u.initials = initialsOf(u.name)
  db.users.push(u)
  audit(user, 'CREATE_USER', 'user', u.id, '', `${u.email} (${u.role})`)
  return ok(u, 201)
}
function handleUserUpdate({ user, params, body }) {
  const i = db.users.findIndex((u) => u.id === params[0])
  if (i < 0) return fail(404, 'Pengguna tidak ditemukan')
  const old = db.users[i]
  if (user.hospitalId && old.hospitalId !== user.hospitalId) return fail(403, 'Di luar cakupan RS Anda')
  const u = {
    id: old.id, name: str(body.name), email: str(body.email), role: str(body.role), hospitalId: str(body.hospitalId),
    mfaEnabled: !!body.mfaEnabled, initials: str(body.initials) || initialsOf(str(body.name)),
  }
  db.users[i] = u
  audit(user, 'UPDATE_USER', 'user', u.id, '', u.email)
  return ok(u)
}
function handleInsurerUpdate({ user, params, body }) {
  const i = db.insurers.findIndex((x) => x.id === params[0])
  if (i < 0) return fail(404, 'Penjamin tidak ditemukan')
  const n = {
    id: db.insurers[i].id, name: str(body.name), type: str(body.type), avgPaymentDays: num(body.avgPaymentDays),
    deadlineDays: num(body.deadlineDays), ceilingPolicy: str(body.ceilingPolicy), verified: !!body.verified,
  }
  db.insurers[i] = n
  audit(user, 'UPDATE_INSURER', 'insurer', n.id, '', `${n.name} deadline=${n.deadlineDays}d`)
  return ok(n)
}
function handlePartnerUpdate({ user, params, body }) {
  const i = db.partners.findIndex((x) => x.id === params[0])
  if (i < 0) return fail(404, 'Partner tidak ditemukan')
  const keep = db.partners[i]
  const n = {
    id: keep.id, company: str(body.company), contact: str(body.contact), rateLimitPerMin: num(body.rateLimitPerMin),
    whitelistIps: arrStr(body.whitelistIps), consumerId: keep.consumerId, consumerSecretMasked: keep.consumerSecretMasked,
    userKeyMasked: keep.userKeyMasked, active: !!body.active,
    hospitalIds: Array.isArray(body.hospitalIds) && body.hospitalIds.length ? arrStr(body.hospitalIds) : keep.hospitalIds,
    createdAt: str(body.createdAt) || keep.createdAt,
  }
  db.partners[i] = n
  audit(user, 'UPDATE_PARTNER', 'partner', n.id, '', n.company)
  return ok(n)
}

// ── Handler: Engine API monitoring ───────────────────────────────────────────
function handleEngineApiSummary({ user }) {
  const intakes = []
  let unresolved = 0, totalHosp = 0, totalJkn = 0
  for (const ai of db.apiIntakes) {
    if (user.hospitalId && ai.hospitalId !== user.hospitalId) continue
    if (ai.status === 'UNRESOLVED') unresolved++
    totalHosp += ai.hospitalTotal
    totalJkn += ai.jknTotal
    intakes.push(ai)
  }
  let calls = 0, rejected = 0
  const partners = new Set()
  for (const e of db.auditLog) {
    if (e.scope !== 'ENGINE_API') continue
    calls++
    if ((e.statusCode || 0) >= 400) rejected++
    partners.add(e.partnerName)
  }
  const latSum = db.apiIntakes.reduce((s, a) => s + a.latencyMs, 0)
  const avgLatencyMs = db.apiIntakes.length ? Math.trunc(latSum / db.apiIntakes.length) : 0
  const covered = db.hospitals.filter((h) => !user.hospitalId || h.id === user.hospitalId).length
  return ok({
    intakes, unresolved, totalHospitalBill: totalHosp, totalJkn, apiCalls: calls, rejected,
    successPct: calls === 0 ? 0 : ((calls - rejected) / calls) * 100, avgLatencyMs,
    partnerCount: partners.size, hospitalCoverage: covered,
  })
}

// ── Ekspor CSV ───────────────────────────────────────────────────────────────
function csvResponse(filename, rows) {
  const esc = (c) => (/[;"\n]/.test(c) ? '"' + c.replace(/"/g, '""') + '"' : c)
  const csv = '\uFEFF' + rows.map((r) => r.map((c) => esc(String(c))).join(';')).join('\n') + '\n'
  return { status: 200, data: null, csv, filename }
}
function handleExportClaims({ user }) {
  const T = todayISO()
  const rows = [['ID', 'SEP', 'Pasien', 'Status', 'Jenis', 'Kode INA-CBG', 'Tarif JKN', 'Tagihan RS', 'Gap', 'Penjamin', 'Readiness']]
  for (const c of db.claims) {
    if (!scopeOK(user, c)) continue
    const ins = insurerByID(c.insurerId)
    rows.push([c.id, c.sepNo, c.patientName, c.status, c.careType, c.inacbgCode, totalJKN(c.inacbgBaseTariff, c.specialCmg),
      sumTariff(c.hospitalTariff), claimGap(c), ins ? ins.name : '', evaluateClaim(c, T).score])
  }
  return csvResponse('klaim-medpay.csv', rows)
}
function handleExportAging({ user }) {
  const T = todayISO()
  const rows = [['ID', 'Pasien', 'Penjamin', 'Status', 'Nilai Tertunggak', 'Umur (hari)']]
  for (const c of db.claims) {
    if (!scopeOK(user, c)) continue
    const o = outstandingOf(c)
    if (o <= 0 || c.status === 'written_off' || c.status === 'rejected') continue
    const ins = insurerByID(c.insurerId)
    rows.push([c.id, c.patientName, ins ? ins.name : '', c.status, o, daysBetween(orDefault(c.submittedAt, c.admissionDate), T)])
  }
  return csvResponse('piutang-medpay.csv', rows)
}
function handleExportAudit() {
  const rows = [['Waktu', 'Aktor', 'Peran', 'Aksi', 'Entitas', 'ID', 'Scope', 'Status', 'IP']]
  for (const e of db.auditLog) rows.push([e.at, e.actor, e.actorRole, e.action, e.entity, e.entityId, e.scope, e.statusCode || 0, e.ip])
  return csvResponse('audit-medpay.csv', rows)
}

// ── Tabel rute ───────────────────────────────────────────────────────────────
// [metode, pola, izin (null = hanya login, 'public' = tanpa login), handler]
const ROUTES = [
  ['POST', /^auth\/login$/, 'public', handleLogin],
  ['POST', /^auth\/verify$/, 'public', handleVerifyMFA],
  ['POST', /^auth\/logout$/, 'public', handleLogout],
  ['GET', /^me$/, null, handleMe],
  ['GET', /^bootstrap$/, null, handleBootstrap],

  ['GET', /^dashboard$/, 'VIEW', handleDashboard],
  ['GET', /^analytics$/, 'VIEW', handleAnalytics],
  ['GET', /^aging$/, 'VIEW', handleAging],

  ['GET', /^claims$/, 'VIEW', handleClaimList],
  ['POST', /^claims$/, 'CREATE', handleClaimCreate],
  ['GET', /^claims\/([^/]+)$/, 'VIEW', handleClaimGet],
  ['PUT', /^claims\/([^/]+)$/, 'EDIT', handleClaimUpdate],
  ['DELETE', /^claims\/([^/]+)$/, 'DELETE', handleClaimDelete],
  ['POST', /^claims\/([^/]+)\/resolve$/, 'RESOLVE', handleResolveTariff],
  ['POST', /^claims\/([^/]+)\/transition$/, 'EDIT', handleTransition],
  ['POST', /^claims\/([^/]+)\/scrub$/, 'VIEW', handleScrubClaim],

  ['GET', /^scrubber\/top-causes$/, 'VIEW', handleScrubTopCauses],
  ['PUT', /^scrub-rules\/([^/]+)$/, 'CONFIGURE', handleScrubRuleUpdate],
  ['POST', /^scrub-rules\/override$/, 'OVERRIDE', handleScrubOverride],

  ['POST', /^resume\/bulk$/, 'CREATE', handleResumeBulk],
  ['PUT', /^resume\/([^/]+)$/, 'EDIT', handleResumeUpdate],
  ['DELETE', /^resume\/([^/]+)$/, 'DELETE', handleResumeDelete],

  ['GET', /^batches$/, 'VIEW', handleBatchList],
  ['POST', /^batches$/, 'CREATE', handleBatchCreate],
  ['GET', /^batches\/([^/]+)$/, 'VIEW', handleBatchGet],
  ['DELETE', /^batches\/([^/]+)$/, 'DELETE', handleBatchDelete],
  ['POST', /^batches\/([^/]+)\/claims$/, 'EDIT', handleBatchAddClaims],
  ['DELETE', /^batches\/([^/]+)\/claims\/([^/]+)$/, 'EDIT', handleBatchRemoveClaim],

  ['GET', /^payments$/, 'VIEW', handlePaymentList],
  ['POST', /^payments$/, 'CREATE', handlePaymentCreate],
  ['DELETE', /^payments\/([^/]+)$/, 'EDIT', handlePaymentDelete],
  ['POST', /^bank-imports$/, 'CREATE', handleBankImportCreate],
  ['GET', /^bank-imports$/, 'VIEW', handleBankImportsList],
  ['GET', /^bank-lines$/, 'VIEW', handleBankLines],
  ['GET', /^bank-lines\/([^/]+)\/candidates$/, 'VIEW', handleBankLineCandidates],
  ['POST', /^bank-lines\/([^/]+)\/confirm$/, 'APPROVE', handleBankLineConfirm],
  ['POST', /^bank-lines\/([^/]+)\/unconfirm$/, 'APPROVE', handleBankLineUnconfirm],

  ['POST', /^kapj\/simulate$/, 'VIEW', handleKapjSimulate],

  ['GET', /^work-items$/, 'VIEW', handleWorkItemList],
  ['POST', /^work-items$/, 'CREATE', handleWorkItemCreate],
  ['PUT', /^work-items\/([^/]+)$/, 'EDIT', handleWorkItemUpdate],
  ['POST', /^work-items\/([^/]+)\/comments$/, 'VIEW', handleWorkItemComment],

  ['GET', /^audit$/, 'VIEW', handleAuditList],

  ['POST', /^hospitals$/, 'CONFIGURE', handleHospitalCreate],
  ['PUT', /^hospitals\/([^/]+)$/, 'CONFIGURE', handleHospitalUpdate],
  ['POST', /^users$/, 'CONFIGURE', handleUserCreate],
  ['PUT', /^users\/([^/]+)$/, 'CONFIGURE', handleUserUpdate],
  ['PUT', /^insurers\/([^/]+)$/, 'CONFIGURE', handleInsurerUpdate],
  ['PUT', /^partners\/([^/]+)$/, 'CONFIGURE', handlePartnerUpdate],

  ['GET', /^engine-api\/summary$/, 'VIEW', handleEngineApiSummary],

  ['GET', /^export\/claims\.csv$/, 'EXPORT', handleExportClaims],
  ['GET', /^export\/aging\.csv$/, 'EXPORT', handleExportAging],
  ['GET', /^export\/audit\.csv$/, 'EXPORT', handleExportAudit],
]

function sessionUser(token) {
  if (!token || !token.startsWith('mock.')) return null
  return userByID(token.slice(5))
}

// ── Titik masuk ──────────────────────────────────────────────────────────────
// Mengembalikan { status, data, csv?, filename? } — data sudah di-clone (JSON)
// seperti respons jaringan, sehingga objek reaktif di UI tidak memutasi store.
export async function mockRequest(method, rawPath, body, token) {
  loadDb()
  if (LATENCY_MS > 0) await new Promise((r) => setTimeout(r, LATENCY_MS))

  const [path, qs = ''] = String(rawPath).split('?')
  const query = new URLSearchParams(qs)
  let route = null, params = []
  for (const r of ROUTES) {
    if (r[0] !== method) continue
    const m = r[1].exec(path)
    if (m) { route = r; params = m.slice(1); break }
  }
  if (!route) return { status: 404, data: { error: 'Endpoint tidak ditemukan' } }

  const perm = route[2]
  let user = null
  if (perm !== 'public') {
    user = sessionUser(token)
    if (!user) return { status: 401, data: { error: 'Sesi tidak valid — silakan login kembali' } }
    if (perm && !hasPermission(user, perm)) {
      return { status: 403, data: { error: `Peran Anda (${user.role}) tidak memiliki izin ${perm}` } }
    }
  }

  let res
  try {
    res = route[3]({ user, params, query, body: body || {} })
  } catch (e) {
    console.error('[mock-api]', method, rawPath, e)
    return { status: 500, data: { error: 'Kesalahan internal mock API: ' + (e && e.message) } }
  }
  if (method !== 'GET') persist()
  return { ...res, data: clone(res.data) }
}
