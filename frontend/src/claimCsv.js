// ── Impor klaim dari CSV (pemisah ";" atau ",") ──────────────────────────────
// Satu file = satu klaim (baris data pertama). Berisi identitas, diagnosa, 18 komponen
// tarif RS (wajib), dan data penjamin kedua. Tarif INA-CBG tidak ada di CSV — di-resolve
// oleh grouper setelah file dimuat.
import { TARIFF_COMPONENT_KEYS, parseMoney } from './format'

const snake = (k) => k.replace(/[A-Z]/g, (m) => '_' + m.toLowerCase())

export const COMPONENT_COLUMNS = TARIFF_COMPONENT_KEYS.map((k) => ({ key: k, col: snake(k) }))

const REQUIRED_BASE = [
  'sep', 'no_kartu_bpjs', 'mrn', 'nama_pasien', 'jenis_peserta', 'jenis_rawat', 'kelas', 'jenis_kelamin',
  'tgl_lahir', 'tgl_masuk', 'tgl_pulang', 'dx_utama', 'penjamin', 'no_polis', 'nama_tertanggung',
]

// Urutan kolom pada file contoh (kolom wajib + opsional).
export const SAMPLE_HEADER = [
  'kode_faskes', 'sep', 'no_kartu_bpjs', 'mrn', 'nama_pasien', 'dpjp', 'jenis_peserta', 'jenis_rawat', 'kelas',
  'jenis_kelamin', 'tgl_lahir', 'tgl_masuk', 'tgl_pulang', 'dx_utama', 'dx_sekunder', 'prosedur',
  ...COMPONENT_COLUMNS.map((c) => c.col),
  'penjamin', 'no_polis', 'nama_tertanggung', 'nama_penjamin_tambahan', 'jenis_asuransi', 'no_anggota',
  'jenis_perlindungan', 'plafon_polis',
]

export function sampleRows(admission, discharge) {
  const comp = { prosedurNonBedah: 2000000, laboratorium: 1500000, kamarAkomodasi: 4000000, obat: 2500000 }
  return [
    SAMPLE_HEADER,
    [
      '0304R001', '0301R001009001', '0002100099001', 'RM-90001', 'Tes UAT Satu', 'dr. Uji Coba, Sp.PD', 'NON_PBI',
      'RAWAT_INAP', 'I', 'L', '1980-05-12', admission, discharge, 'I25.1', 'E11.9|I10', '88.55',
      ...COMPONENT_COLUMNS.map((c) => comp[c.key] || 0),
      'AIA FINANCIAL', 'POL-UAT-001', 'Tes UAT Satu', '', 'Individu', 'MBR-001', 'Naik Kelas', 50000000,
    ],
  ]
}

function splitLine(line, d) {
  const out = []
  let cur = '', q = false
  for (let i = 0; i < line.length; i++) {
    const ch = line[i]
    if (q) {
      if (ch === '"' && line[i + 1] === '"') { cur += '"'; i++ } else if (ch === '"') q = false
      else cur += ch
    } else if (ch === '"') q = true
    else if (ch === d) { out.push(cur.trim()); cur = '' }
    else cur += ch
  }
  out.push(cur.trim())
  return out
}

function toISODate(s) {
  const v = (s || '').trim()
  if (/^\d{4}-\d{2}-\d{2}$/.test(v)) return v
  const m = /^(\d{2})[-/](\d{2})[-/](\d{4})$/.exec(v)
  return m ? `${m[3]}-${m[2]}-${m[1]}` : ''
}

const list = (s) => (s || '').split(/[|,]/).map((x) => x.trim().toUpperCase()).filter(Boolean)
const norm = (s) => (s || '').toUpperCase().replace(/[\s-]+/g, '_')

// ctx: { insurers, hospitals, lockedHospitalId }
export function parseClaimCsv(text, ctx) {
  const errors = [], warnings = []
  const lines = String(text || '').replace(/^﻿/, '').split(/\r?\n/).filter((l) => l.trim())
  if (lines.length < 2) return { errors: ['File kosong: butuh 1 baris header dan minimal 1 baris data.'], warnings }
  const d = (lines[0].match(/;/g) || []).length >= (lines[0].match(/,/g) || []).length ? ';' : ','
  const header = splitLine(lines[0], d).map((h) => h.toLowerCase())
  const need = [...REQUIRED_BASE, ...COMPONENT_COLUMNS.map((c) => c.col)]
  const missing = need.filter((c) => !header.includes(c))
  if (missing.length) return { errors: ['Kolom wajib tidak ditemukan: ' + missing.join(', ')], warnings }
  if (lines.length > 2) warnings.push('File berisi ' + (lines.length - 1) + ' baris data — hanya baris pertama yang dimuat (satu klaim per file).')
  const vals = splitLine(lines[1], d)
  const row = {}
  header.forEach((h, i) => { row[h] = (vals[i] || '').trim() })

  for (const c of REQUIRED_BASE) if (!row[c]) errors.push('Kolom "' + c + '" wajib diisi.')

  const peserta = norm(row.jenis_peserta)
  if (row.jenis_peserta && !['NON_PBI', 'PBI'].includes(peserta)) errors.push('jenis_peserta harus NON_PBI atau PBI.')
  const rawat = norm(row.jenis_rawat)
  if (row.jenis_rawat && !['RAWAT_INAP', 'RAWAT_JALAN'].includes(rawat)) errors.push('jenis_rawat harus RAWAT_INAP atau RAWAT_JALAN.')
  const kelas = (row.kelas || '').toUpperCase()
  if (row.kelas && !['I', 'II', 'III'].includes(kelas)) errors.push('kelas harus I, II, atau III.')
  const sex = (row.jenis_kelamin || '').toUpperCase().charAt(0)
  if (row.jenis_kelamin && !['L', 'P'].includes(sex)) errors.push('jenis_kelamin harus L atau P.')

  const dates = {}
  for (const [col, key] of [['tgl_lahir', 'dateOfBirth'], ['tgl_masuk', 'admissionDate'], ['tgl_pulang', 'dischargeDate']]) {
    if (!row[col]) continue
    dates[key] = toISODate(row[col])
    if (!dates[key]) errors.push(col + ' harus berformat YYYY-MM-DD atau DD-MM-YYYY.')
  }

  const hospitalTariff = {}
  for (const { key, col } of COMPONENT_COLUMNS) {
    const raw = row[col]
    if (raw === '' || raw === undefined) { errors.push('Komponen tarif "' + col + '" wajib diisi (isi 0 bila tidak ada).'); continue }
    if (!/^[\d.\sRp,]+$/i.test(raw)) { errors.push('Komponen "' + col + '" harus angka rupiah ≥ 0 (nilai: ' + raw + ').'); continue }
    hospitalTariff[key] = parseMoney(raw)
  }

  const insurer = (ctx.insurers || []).find((i) => i.name.toLowerCase() === (row.penjamin || '').toLowerCase())
  if (row.penjamin && !insurer) errors.push('Penjamin "' + row.penjamin + '" tidak ada di daftar penjamin resmi (cek penulisan nama).')

  let hospitalId = ctx.lockedHospitalId || ''
  if (row.kode_faskes) {
    const h = (ctx.hospitals || []).find((x) => x.kodeFaskes === row.kode_faskes)
    if (!h) errors.push('kode_faskes "' + row.kode_faskes + '" tidak dikenal / di luar RS Anda.')
    else if (ctx.lockedHospitalId && h.id !== ctx.lockedHospitalId) errors.push('kode_faskes tidak sesuai RS akun Anda.')
    else hospitalId = h.id
  }

  if (errors.length) return { errors, warnings }
  return {
    errors, warnings,
    claim: {
      hospitalId, sepNo: row.sep, bpjsCardNo: row.no_kartu_bpjs, mrn: row.mrn, patientName: row.nama_pasien,
      dpjp: row.dpjp || '', participantType: peserta, careType: rawat, careClass: kelas, sex, ...dates,
      primaryDiagnosis: row.dx_utama.toUpperCase(), secondaryDiagnoses: list(row.dx_sekunder), procedures: list(row.prosedur),
      hospitalTariff, insurerId: insurer.id, policyNo: row.no_polis, insuredName: row.nama_tertanggung,
      insurerNameExtra: row.nama_penjamin_tambahan || '', insuranceType: row.jenis_asuransi || 'Individu',
      insuranceMemberNo: row.no_anggota || '', coverageType: row.jenis_perlindungan || 'Naik Kelas',
      policyCeilingInput: row.plafon_polis ? parseMoney(row.plafon_polis) : 0, insurerName: insurer.name,
    },
  }
}
