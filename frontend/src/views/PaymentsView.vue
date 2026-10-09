<template>
  <main class="page">
    <PageHeader icon="wallet" title="Pembayaran &amp; Rekonsiliasi" sub="Satu klaim bisa punya banyak pembayaran (cicilan/sebagian). Impor mutasi bank → auto-match → review two-panel → konfirmasi dengan deteksi kurang/lebih bayar & duplikat." />

    <div class="tabs">
      <button class="tab" :class="{ active: tab === 'catat' }" @click="tab = 'catat'">Catat Pembayaran</button>
      <button class="tab" :class="{ active: tab === 'impor' }" @click="tab = 'impor'">Impor Mutasi Rekening</button>
      <button class="tab" :class="{ active: tab === 'rekonsiliasi' }" @click="tab = 'rekonsiliasi'">Rekonsiliasi</button>
    </div>

    <!-- Catat pembayaran -->
    <div v-if="tab === 'catat'">
      <div class="grid-2">
        <div class="card card-pad">
          <div class="card-title">Catat Pembayaran Manual</div>
          <div class="card-sub">Untuk klaim berstatus Disetujui / Disetujui Sebagian / Sebagian Dibayar / Lunas (cicilan).</div>
          <div class="field">
            <label>Klaim <span class="req">*</span></label>
            <select class="input" v-model="pay.claimId">
              <option value="">— pilih klaim —</option>
              <option v-for="c in payable" :key="c.id" :value="c.id">{{ c.id }} — {{ c.patientName }} ({{ CLAIM_LABELS[c.status] }} · gap {{ formatRpShort(gapOf(c)) }})</option>
            </select>
          </div>
          <div class="form-grid">
            <div class="field"><label>Tanggal Diterima <span class="req">*</span></label><input class="input" type="date" v-model="pay.receivedDate" /></div>
            <div class="field"><label>Nominal (Rp) <span class="req">*</span></label><MoneyInput v-model="pay.amount" /></div>
            <div class="field"><label>Payer</label><input class="input" v-model="pay.payer" placeholder="Nama penjamin / bank" /></div>
            <div class="field"><label>Referensi Transfer</label><input class="input" v-model="pay.transferRef" placeholder="TRF-…" /></div>
          </div>
          <div class="field"><label>Catatan</label><input class="input" v-model="pay.notes" /></div>
          <button class="btn btn-teal" :disabled="!pay.claimId || !pay.amount || busy" @click="record"><span v-if="busy" class="spin" style="border-top-color:#fff" /> Simpan Pembayaran</button>
          <div class="alert warning" v-if="payWarn" style="margin-top:12px"><span>⚠</span><span>{{ payWarn }}</span></div>
          <div class="alert success" v-if="payOk" style="margin-top:12px"><span>✓</span><span>{{ payOk }}</span></div>
        </div>
        <div class="card card-pad">
          <div class="card-title">Pembayaran Terakhir</div>
          <div class="card-sub">Undo tersedia — setiap perubahan tercatat di audit trail (append-only).</div>
          <div class="tbl-wrap" style="max-height:420px;overflow-y:auto">
            <table class="tbl" style="min-width:420px">
              <thead><tr><th>Tanggal</th><th>Klaim</th><th class="num">Nominal</th><th>Payer</th><th></th></tr></thead>
              <tbody>
                <tr v-for="p in payments" :key="p.payment.id">
                  <td style="font-size:12px">{{ formatDate(p.payment.receivedDate) }}</td>
                  <td class="mono" style="font-size:11.5px">{{ p.payment.claimId }}<br /><span class="badge" :class="CLAIM_TONE[p.claim.status]" style="margin-top:3px">{{ CLAIM_LABELS[p.claim.status] }}</span></td>
                  <td class="num" style="font-weight:700">{{ formatRp(p.payment.amount) }}</td>
                  <td style="font-size:12px;max-width:130px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">{{ p.payment.payer }}</td>
                  <td><button v-if="hasPerm('EDIT')" class="btn btn-outline btn-xs" @click="undo(p.payment)" title="Undo pembayaran">↩</button></td>
                </tr>
                <tr v-if="!payments.length"><td colspan="5" style="text-align:center;color:var(--text-muted)">Belum ada pembayaran</td></tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>

    <!-- Impor mutasi -->
    <div v-if="tab === 'impor'">
      <div class="grid-2">
        <div class="card card-pad">
          <div class="card-title">Impor Mutasi Bank (CSV)</div>
          <div class="card-sub">Tempel baris CSV: <code>tanggal;deskripsi;nominal;referensi</code>. Auto-match memakai SEP (6 digit akhir), nama tertanggung, no. polis, dan nominal ± toleransi Rp50.000.</div>
          <div class="form-grid">
            <div class="field"><label>Nama Bank / Rekening <span class="req">*</span></label><input class="input" v-model="imp.bankName" placeholder="Bank BCA — Rek Giro RS" /></div>
            <div class="field"><label>Periode <span class="req">*</span></label><input class="input" v-model="imp.periodLabel" placeholder="April 2026" /></div>
          </div>
          <div class="field">
            <label>Baris Mutasi (CSV) <span class="req">*</span></label>
            <textarea class="input" v-model="imp.csv" style="min-height:150px;font-family:ui-monospace,monospace;font-size:12px" placeholder="2026-04-02;TRANSFER MASUK AIA FINANCIAL KLAIM 000188;12450000;BCA/FT/2604/001" />
          </div>
          <button class="btn btn-teal" :disabled="!imp.bankName || !imp.periodLabel || !imp.csv.trim() || busy" @click="doImport"><span v-if="busy" class="spin" style="border-top-color:#fff" /> Impor &amp; Auto-Match</button>
          <div class="alert success" v-if="impResult" style="margin-top:12px"><span>✓</span>
            <span>Impor <b>{{ impResult.import.id }}</b> selesai: {{ impResult.matched }} matched, {{ impResult.needsReview }} perlu review, {{ impResult.unmatched }} unmatched. Lanjut ke tab Rekonsiliasi.</span>
          </div>
        </div>
        <div class="card card-pad">
          <div class="card-title">Riwayat Impor</div>
          <div class="card-sub">Pilih impor untuk membuka rekonsiliasi.</div>
          <div class="tbl-wrap">
            <table class="tbl" style="min-width:380px">
              <thead><tr><th>Import</th><th>Bank</th><th>Periode</th><th class="num">Baris</th><th></th></tr></thead>
              <tbody>
                <tr v-for="b in imports" :key="b.id" @click="openRecon(b.id)" :style="reconImportId === b.id ? 'background:var(--teal-bg)' : ''">
                  <td class="mono">{{ b.id }}</td>
                  <td>{{ b.bankName }}</td>
                  <td>{{ b.periodLabel }}</td>
                  <td class="num">{{ b.lineCount }}</td>
                  <td style="font-size:11px;color:var(--text-muted)">{{ formatDateTime(b.importedAt) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>

    <!-- Rekonsiliasi two-panel -->
    <div v-if="tab === 'rekonsiliasi'">
      <div class="filterbar">
        <select class="input" v-model="reconImportId" @change="loadLines">
          <option value="">— semua impor —</option>
          <option v-for="b in imports" :key="b.id" :value="b.id">{{ b.id }} — {{ b.bankName }} ({{ b.periodLabel }})</option>
        </select>
      </div>
      <div class="two-panel">
        <div>
          <div class="card-title" style="margin-bottom:8px">Baris Mutasi Rekening</div>
          <div class="tbl-wrap" style="max-height:520px;overflow-y:auto">
            <table class="tbl" style="min-width:440px">
              <thead><tr><th>Tanggal</th><th>Deskripsi</th><th class="num">Nominal</th><th>Hasil</th></tr></thead>
              <tbody>
                <tr v-for="l in lines" :key="l.line.id" @click="selLine = l.line" :style="selLine && selLine.id === l.line.id ? 'background:var(--teal-bg)' : ''">
                  <td style="font-size:12px">{{ formatDate(l.line.valueDate) }}</td>
                  <td>
                    <div style="max-width:230px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap" :title="l.line.description">{{ l.line.description }}</div>
                    <div v-if="l.claim" style="font-size:11px;color:var(--text-muted)">→ {{ l.claim.id }} {{ l.claim.patientName }}</div>
                  </td>
                  <td class="num" style="font-weight:700">{{ formatRp(l.line.amount) }}</td>
                  <td>
                    <span class="badge" :class="l.line.matchResult === 'matched' ? 'success' : l.line.matchResult === 'needs_review' ? 'warning' : 'neutral'">
                      {{ l.line.matchResult === 'matched' ? 'Matched' : l.line.matchResult === 'needs_review' ? 'Perlu Review' : 'Unmatched' }}
                    </span>
                    <div v-if="l.line.confirmed" style="font-size:10.5px;color:var(--success);margin-top:3px">✓ terkonfirmasi</div>
                  </td>
                </tr>
                <tr v-if="!lines.length"><td colspan="4" style="text-align:center;color:var(--text-muted)">Tidak ada baris — impor mutasi terlebih dahulu.</td></tr>
              </tbody>
            </table>
          </div>
        </div>
        <div>
          <div class="card-title" style="margin-bottom:8px">Kandidat Klaim (skor kecocokan)</div>
          <div v-if="!selLine" class="card card-pad"><Empty icon="←" title="Pilih baris mutasi" desc="Klik baris di kiri untuk melihat kandidat klaim beserta skor dan alasan pencocokan." /></div>
          <template v-else>
            <div class="card card-pad" style="margin-bottom:10px">
              <div style="font-size:13px"><b>{{ selLine.description }}</b></div>
              <div style="display:flex;gap:14px;margin-top:6px;font-size:12.5px;color:var(--text-secondary)">
                <span>{{ formatDate(selLine.valueDate) }}</span>
                <span><b>{{ formatRp(selLine.amount) }}</b></span>
                <span class="mono">{{ selLine.reference }}</span>
              </div>
              <div v-if="selLine.matchNote" class="hint" style="margin-top:6px">{{ selLine.matchNote }}</div>
            </div>
            <div v-for="c in candidates" :key="c.claimId" class="cand" :class="{ sel: confirmClaimId === c.claimId }" @click="confirmClaimId = c.claimId">
              <div style="display:flex;gap:9px;align-items:center">
                <span class="score-pill" :class="c.score >= 75 ? '' : c.score >= 40 ? 'mid' : 'low'">{{ c.score }}</span>
                <b style="flex:1">{{ c.patientName }}</b>
                <span class="badge" :class="CLAIM_TONE[c.status]">{{ CLAIM_LABELS[c.status] }}</span>
              </div>
              <div style="font-size:12px;color:var(--text-secondary);margin-top:5px">
                {{ c.claimId }} · SEP …{{ c.sepNo.slice(-6) }} · gap {{ formatRp(c.gap) }} · sudah dibayar {{ formatRp(c.paid) }}
              </div>
              <div style="font-size:11.5px;color:var(--text-muted);margin-top:2px">Indikator: {{ c.reason }}</div>
            </div>
            <div v-if="!candidates.length" class="card card-pad"><Empty icon="🤷" title="Tidak ada kandidat" desc="Tidak ada klaim dengan skor ≥ 25 untuk baris ini." /></div>
            <div style="display:flex;gap:9px;margin-top:12px" v-if="candidates.length">
              <button class="btn btn-teal" :disabled="!confirmClaimId" @click="confirm(false)">Konfirmasi Pencocokan</button>
              <button v-if="selLine.confirmed" class="btn btn-outline" @click="unconfirm">Undo Konfirmasi</button>
            </div>
            <div class="alert warning" v-if="confirmWarns.length" style="margin-top:12px">
              <span>⚠</span>
              <span>
                <div v-for="(w, i) in confirmWarns" :key="i">{{ w }}</div>
                <button class="btn btn-danger btn-sm" style="margin-top:8px" @click="confirm(true)">Tetap Konfirmasi (paksa)</button>
              </span>
            </div>
          </template>
        </div>
      </div>
    </div>
  </main>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { api } from '../api'
import { state, hasPerm, toast, loadBoot } from '../store'
import MoneyInput from '../components/MoneyInput.vue'
import PageHeader from '../components/PageHeader.vue'
import Empty from '../components/Empty.vue'
import { formatRp, formatRpShort, formatDate, formatDateTime, parseMoney, todayISO, sumTariff, jknTotal, CLAIM_LABELS, CLAIM_TONE } from '../format'

const tab = ref('catat')
const busy = ref(false)
const payments = ref([])
const payable = ref([])
const pay = ref({ claimId: '', receivedDate: todayISO(), amount: 0, payer: '', transferRef: '', notes: '' })
const payWarn = ref('')
const payOk = ref('')
const imp = ref({ bankName: '', periodLabel: '', csv: '' })
const impResult = ref(null)
const imports = ref([])
const lines = ref([])
const reconImportId = ref('')
const selLine = ref(null)
const candidates = ref([])
const confirmClaimId = ref('')
const confirmWarns = ref([])

function gapOf(c) { return sumTariff(c.hospitalTariff) - jknTotal(c.inacbgBaseTariff, c.specialCmg) }

async function loadPayments() {
  payments.value = await api('payments')
  const res = await api('claims?pageSize=200')
  payable.value = res.rows.filter(c => ['approved', 'partially_approved', 'partially_paid', 'paid'].includes(c.status))
}

async function record() {
  busy.value = true
  payWarn.value = ''
  payOk.value = ''
  try {
    const res = await api('payments', { method: 'POST', body: { ...pay.value } })
    if (res.warning) payWarn.value = res.warning + ' — pembayaran tetap tercatat (revisi manual/rekonsiliasi menyusul).'
    else payOk.value = 'Pembayaran ' + res.payment.id + ' tercatat. Status klaim → ' + CLAIM_LABELS[res.claimStatus] + '. Dibayar ' + formatRp(res.totalPaid) + ' dari gap ' + formatRp(res.gap) + '.'
    toast('Pembayaran tercatat.', 'success')
    pay.value.amount = 0
    pay.value.transferRef = ''
    await loadPayments()
  } catch (e) { toast(e.message, 'error') } finally { busy.value = false }
}

async function undo(p) {
  if (!confirm('Undo pembayaran ' + p.id + '? Status klaim akan dikembalikan (tercatat di audit).')) return
  try {
    await api('payments/' + p.id, { method: 'DELETE' })
    toast('Pembayaran dibatalkan (undo).', 'success')
    await loadPayments()
  } catch (e) { toast(e.message, 'error') }
}

async function doImport() {
  busy.value = true
  try {
    const linesIn = imp.value.csv.trim().split(/\r?\n/).filter(Boolean).map(ln => {
      const [valueDate, description, amount, reference] = ln.split(';').map(s => (s || '').trim())
      return { valueDate: valueDate || todayISO(), description: description || '', amount: parseMoney(amount), reference: reference || '' }
    })
    impResult.value = await api('bank-imports', { method: 'POST', body: { bankName: imp.value.bankName, periodLabel: imp.value.periodLabel, lines: linesIn } })
    toast('Impor selesai — buka tab Rekonsiliasi.', 'success')
    imp.value.csv = ''
    await refreshImports()
  } catch (e) { toast(e.message, 'error') } finally { busy.value = false }
}

const knownImports = ref([])

async function refreshImports() {
  knownImports.value = await api('bank-imports')
  imports.value = knownImports.value
}

async function loadLines() {
  selLine.value = null
  candidates.value = []
  confirmWarns.value = []
  const params = reconImportId.value ? '?importId=' + reconImportId.value : ''
  lines.value = await api('bank-lines' + params)
}

async function openRecon(id) {
  reconImportId.value = id
  tab.value = 'rekonsiliasi'
  await loadLines()
}

async function pickLine(l) {
  selLine.value = l
  confirmWarns.value = []
  candidates.value = await api('bank-lines/' + l.id + '/candidates')
}

async function confirm(force) {
  try {
    const res = await api('bank-lines/' + selLine.value.id + '/confirm', { method: 'POST', body: { claimId: confirmClaimId.value, force } })
    toast('Ter Rekonsiliasi → pembayaran ' + res.payment.id + ' dibuat, status klaim → ' + CLAIM_LABELS[res.claimStatus] + '.', 'success')
    confirmWarns.value = []
    await loadLines()
    await loadPayments()
  } catch (e) {
    if (e.payload && e.payload.warnings) { confirmWarns.value = e.payload.warnings }
    else toast(e.message, 'error')
  }
}

async function unconfirm() {
  try {
    await api('bank-lines/' + selLine.value.id + '/unconfirm', { method: 'POST' })
    toast('Konfirmasi dibatalkan (undo, tercatat di audit).', 'success')
    await loadLines()
    await loadPayments()
  } catch (e) { toast(e.message, 'error') }
}

watch(selLine, (l) => { if (l) pickLine(l) })

onMounted(async () => {
  await loadBoot()
  await loadPayments()
  await refreshImports()
  await loadLines()
})
</script>
