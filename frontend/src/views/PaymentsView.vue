<template>
  <main class="page">
    <PageHeader icon="wallet" title="Pembayaran &amp; Rekonsiliasi" sub="Satu klaim bisa punya banyak pembayaran (cicilan/sebagian). Impor mutasi bank → auto-match → review two-panel → konfirmasi dengan deteksi kurang/lebih bayar & duplikat." />

    <div class="tabs">
      <button class="tab" :class="{ active: tab === 'catat' }" @click="tab = 'catat'">Catat Pembayaran</button>
      <button v-if="canRecord" class="tab" :class="{ active: tab === 'impor' }" @click="tab = 'impor'">Impor Mutasi Rekening</button>
      <button class="tab" :class="{ active: tab === 'rekonsiliasi' }" @click="tab = 'rekonsiliasi'">Rekonsiliasi</button>
    </div>

    <!-- Catat pembayaran -->
    <div v-if="tab === 'catat'">
      <div :class="canRecord ? 'grid-2' : ''">
        <div class="card card-pad" v-if="canRecord">
          <div class="card-title">Catat Pembayaran Manual</div>
          <div class="card-sub">Penjamin membayar per batch. Boleh dicicil, total tidak boleh melebihi sisa tagihan batch; nominal dibagi otomatis ke klaim di dalam batch.</div>
          <div class="field">
            <label>Batch <span class="req">*</span></label>
            <select class="input" v-model="pay.batchId">
              <option value="">— pilih batch —</option>
              <option v-for="b in payable" :key="b.batch.id" :value="b.batch.id">{{ docNo(b.batch, 'batchNo') }} — {{ b.batch.name }}</option>
            </select>
          </div>
          <div class="hint" v-if="selectedBatch" style="margin:-6px 0 10px">Sisa tagihan batch ini: <b>{{ formatRp(selectedBatch.remaining) }}</b> · {{ selectedBatch.payableCount }} klaim menunggu bayar<span v-if="selectedBatch.paid > 0"> (sudah dibayar {{ formatRp(selectedBatch.paid) }} dari {{ formatRp(selectedBatch.gap) }})</span></div>
          <div class="form-grid">
            <div class="field"><label>Tanggal Diterima <span class="req">*</span></label><input class="input" type="date" v-model="pay.receivedDate" /></div>
            <div class="field"><label>Nominal (Rp) <span class="req">*</span></label><MoneyInput v-model="pay.amount" /></div>
            <div class="field"><label>Payer</label><input class="input" v-model="pay.payer" placeholder="Nama penjamin / bank" /></div>
            <div class="field"><label>Referensi Transfer</label><input class="input" v-model="pay.transferRef" placeholder="TRF-…" /></div>
          </div>
          <div class="alert danger" v-if="overAmount > 0" style="margin-bottom:12px"><span><b>Nominal Melebihi Sisa Tagihan.</b> Nominal {{ formatRp(pay.amount) }} lebih besar dari sisa {{ formatRp(selectedBatch.remaining) }} (kelebihan {{ formatRp(overAmount) }}). Turunkan nominal agar bisa disimpan.</span></div>
          <div class="field"><label>Catatan</label><input class="input" v-model="pay.notes" /></div>
          <button class="btn btn-teal" :disabled="!pay.batchId || !pay.amount || overAmount > 0 || busy" @click="record"><span v-if="busy" class="spin" style="border-top-color:#fff" /> Simpan Pembayaran</button>
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
                  <td class="mono" style="font-size:11.5px">{{ p.payment.claimNo || p.payment.claimId }} <span style="color:var(--text-muted)">{{ docNo(p.payment, 'paymentNo') }}</span><br /><span class="badge" :class="CLAIM_TONE[p.claim.status]" style="margin-top:3px">{{ CLAIM_LABELS[p.claim.status] }}</span></td>
                  <td class="num" style="font-weight:700">{{ formatRp(p.payment.amount) }}</td>
                  <td style="font-size:12px;max-width:130px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">{{ p.payment.payer }}</td>
                  <td><button v-if="hasPerm('EDIT')" class="btn btn-outline btn-xs" @click="undoTarget = p.payment" title="Undo pembayaran">↩</button></td>
                </tr>
                <tr v-if="!payments.length"><td colspan="5" style="text-align:center;color:var(--text-muted)">Belum ada pembayaran</td></tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>

    <!-- Impor mutasi -->
    <div v-if="tab === 'impor' && canRecord">
      <div class="grid-2">
        <div class="card card-pad">
          <div class="card-title">Impor Mutasi Bank (CSV)</div>
          <div class="card-sub">Unggah file CSV mutasi dengan kolom <code>tanggal;deskripsi;nominal;referensi</code>. Auto-match memakai SEP (6 digit akhir), nama tertanggung, no. polis, dan nominal ± toleransi Rp50.000.</div>
          <div class="form-grid">
            <div class="field"><label>Nama Bank / Rekening <span class="req">*</span></label><input class="input" v-model="imp.bankName" placeholder="Bank BCA — Rek Giro RS" /></div>
            <div class="field">
              <label>Batch (Periode) <span class="req">*</span></label>
              <select class="input" v-model="imp.batchId">
                <option value="">— pilih batch —</option>
                <option v-for="b in payable" :key="b.batch.id" :value="b.batch.id">{{ docNo(b.batch, 'batchNo') }} — {{ b.batch.name }}</option>
              </select>
            </div>
          </div>
          <div class="field">
            <label>File Mutasi (CSV) <span class="req">*</span></label>
            <div class="chips-row">
              <label class="btn btn-outline btn-sm" style="cursor:pointer">
                {{ impFile ? 'Ganti File' : '+ Pilih File CSV' }}
                <input type="file" accept=".csv,text/csv" style="display:none" @change="onImpFile" />
              </label>
              <button class="btn btn-outline btn-sm" type="button" @click="downloadImpSample">⬇ Unduh Contoh CSV</button>
              <span v-if="impFile" class="hint">{{ impFile }} · {{ impRows.length }} baris mutasi</span>
            </div>
            <div class="alert danger" v-if="impErr" style="margin-top:10px"><span>⛔</span><span>{{ impErr }}</span></div>
            <div class="tbl-wrap" v-if="impRows.length" style="margin-top:10px;max-height:170px;overflow-y:auto">
              <table class="tbl" style="min-width:420px">
                <thead><tr><th>Tanggal</th><th>Deskripsi</th><th class="num">Nominal</th><th>Referensi</th></tr></thead>
                <tbody>
                  <tr v-for="(r, i) in impRows.slice(0, 5)" :key="i">
                    <td style="font-size:12px">{{ r.valueDate }}</td>
                    <td style="font-size:12px;max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap" :title="r.description">{{ r.description }}</td>
                    <td class="num">{{ formatRp(r.amount) }}</td>
                    <td class="mono" style="font-size:11px">{{ r.reference }}</td>
                  </tr>
                </tbody>
              </table>
              <div class="hint" v-if="impRows.length > 5" style="padding:6px 10px">… dan {{ impRows.length - 5 }} baris lainnya</div>
            </div>
          </div>
          <button class="btn btn-teal" :disabled="!imp.bankName || !imp.batchId || !impRows.length || busy" @click="doImport"><span v-if="busy" class="spin" style="border-top-color:#fff" /> Impor &amp; Auto-Match</button>
          <div class="alert success" v-if="impResult" style="margin-top:12px"><span>✓</span>
            <span>Impor <b>{{ docNo(impResult.import, 'importNo') }}</b> selesai: {{ impResult.matched }} matched, {{ impResult.needsReview }} perlu review, {{ impResult.unmatched }} unmatched. Lanjut ke tab Rekonsiliasi.</span>
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
                  <td class="mono">{{ docNo(b, 'importNo') }}</td>
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
      <div class="two-panel" :style="canApprove ? '' : 'grid-template-columns:1fr'">
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
                    <div v-if="l.claim" style="font-size:11px;color:var(--text-muted)">→ {{ docNo(l.claim) }} {{ l.claim.patientName }}</div>
                  </td>
                  <td class="num" style="font-weight:700">{{ formatRp(l.line.amount) }}</td>
                  <td>
                    <span class="badge" :class="refTone('match_result', l.line.matchResult)">
                      {{ refLabel('match_result', l.line.matchResult) }}
                    </span>
                    <div v-if="l.line.confirmed" style="font-size:10.5px;color:var(--success);margin-top:3px">✓ terkonfirmasi</div>
                  </td>
                </tr>
                <tr v-if="!lines.length"><td colspan="4" style="text-align:center;color:var(--text-muted)">Tidak ada baris — impor mutasi terlebih dahulu.</td></tr>
              </tbody>
            </table>
          </div>
        </div>
        <div v-if="canApprove">
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
                {{ c.claimNo || c.claimId }} · SEP …{{ c.sepNo.slice(-6) }} · gap {{ formatRp(c.gap) }} · sudah dibayar {{ formatRp(c.paid) }}
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


    <Modal v-if="undoTarget" title="Batalkan pembayaran?" @close="undoTarget = null">
      <p style="font-size:13.5px">Pembayaran <b>{{ docNo(undoTarget, 'paymentNo') }}</b> sebesar <b>{{ formatRp(undoTarget.amount) }}</b> untuk klaim <b>{{ undoTarget.claimNo || undoTarget.claimId }}</b> akan dibatalkan dan status klaim dihitung ulang. Tindakan ini tercatat di audit trail.</p>
      <template #footer>
        <button class="btn btn-outline" @click="undoTarget = null">Batal</button>
        <button class="btn btn-danger" @click="undo">Ya, Batalkan Pembayaran</button>
      </template>
    </Modal>
  </main>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { api, downloadBlobCsv } from '../api'
import { state, hasPerm, toast, loadBoot, refreshBoot } from '../store'
import MoneyInput from '../components/MoneyInput.vue'
import PageHeader from '../components/PageHeader.vue'
import Empty from '../components/Empty.vue'
import Modal from '../components/Modal.vue'
import { formatRp, formatRpShort, formatDate, formatDateTime, parseMoney, todayISO, sumTariff, jknTotal, CLAIM_LABELS, CLAIM_TONE, docNo, refLabel, refTone } from '../format'

// Section input & kandidat hanya untuk peran berizin; peran baca-saja melihat daftar saja.
const canRecord = computed(() => hasPerm('CREATE'))
const canApprove = computed(() => hasPerm('APPROVE'))
const undoTarget = ref(null)
const tab = ref('catat')
const busy = ref(false)
const payments = ref([])
const payable = ref([])
const pay = ref({ batchId: '', receivedDate: todayISO(), amount: 0, payer: '', transferRef: '', notes: '' })
const payWarn = ref('')
const payOk = ref('')
const imp = ref({ bankName: '', batchId: '' })
const impRows = ref([])
const impFile = ref('')
const impErr = ref('')
const impResult = ref(null)
const imports = ref([])
const lines = ref([])
const reconImportId = ref('')
const selLine = ref(null)
const candidates = ref([])
const confirmClaimId = ref('')
const confirmWarns = ref([])

function gapOf(c) { return sumTariff(c.hospitalTariff) - jknTotal(c.inacbgBaseTariff, c.specialCmg) }

const selectedBatch = computed(() => payable.value.find(b => b.batch.id === pay.value.batchId) || null)
const overAmount = computed(() => selectedBatch.value ? Math.max(0, pay.value.amount - selectedBatch.value.remaining) : 0)

// Klaim yang sudah lunas (status Lunas / sisa tagihan 0) tidak ditawarkan lagi agar tidak
// terjadi pembayaran ganda. Untuk yang dibayar sebagian, tampilkan sisa (gap − sudah dibayar).
async function loadPayments() {
  payments.value = await api('payments')
  // Batch yang masih punya tagihan menunggu pembayaran (sisa > 0).
  payable.value = (await api('payments/batches')).filter(b => b.remaining > 0)
  if (pay.value.batchId && !selectedBatch.value) pay.value.batchId = ""
  refreshBoot() // notifikasi pembayaran & badge langsung ter-update tanpa refresh halaman
}

async function record() {
  if (overAmount.value > 0) return
  busy.value = true
  payWarn.value = ''
  payOk.value = ''
  try {
    const res = await api('payments/batch', { method: 'POST', body: { ...pay.value } })
    payOk.value = 'Pembayaran batch ' + docNo(res.batch, 'batchNo') + ' ' + formatRp(res.totalPaid) + ' tercatat, dibagi ke ' + res.parts.length + ' klaim. Sisa tagihan batch ' + formatRp(res.remaining) + '.'
    if (res.warning) payWarn.value = res.warning + ' — pembayaran tetap tercatat.'
    toast('Pembayaran tercatat.', 'success')
    pay.value.amount = 0
    pay.value.transferRef = ''
    await loadPayments()
  } catch (e) { toast(e.message, 'error') } finally { busy.value = false }
}

async function undo() {
  const p = undoTarget.value
  if (!p) return
  undoTarget.value = null
  try {
    await api('payments/' + p.id, { method: 'DELETE' })
    toast('Pembayaran dibatalkan (undo).', 'success')
    await loadPayments()
  } catch (e) { toast(e.message, 'error') }
}

// Baca file CSV mutasi: tanggal;deskripsi;nominal;referensi (baris header boleh ada).
function onImpFile(ev) {
  const file = ev.target.files[0]
  ev.target.value = ''
  if (!file) return
  const reader = new FileReader()
  reader.onload = () => {
    impErr.value = ''
    const rows = String(reader.result || '').replace(/^\uFEFF/, '').split(/\r?\n/).filter(l => l.trim())
      .filter((l, i) => !(i === 0 && /^tanggal/i.test(l.trim())))
      .map(ln => {
        const [valueDate, description, amount, reference] = ln.split(';').map(s => (s || '').trim())
        return { valueDate: valueDate || todayISO(), description: description || '', amount: parseMoney(amount), reference: reference || '' }
      })
    if (!rows.length) { impErr.value = 'File kosong atau format tidak sesuai (tanggal;deskripsi;nominal;referensi).'; impRows.value = []; impFile.value = ''; return }
    if (rows.some(r => !r.amount)) { impErr.value = 'Ada baris dengan nominal kosong/0 — periksa kolom ke-3 (nominal).'; impRows.value = []; impFile.value = ''; return }
    impRows.value = rows
    impFile.value = file.name
  }
  reader.readAsText(file)
}

function downloadImpSample() {
  const d = todayISO()
  downloadBlobCsv('contoh-mutasi-bank.csv', [
    ['tanggal', 'deskripsi', 'nominal', 'referensi'],
    [d, 'TRANSFER MASUK AIA FINANCIAL KLAIM 000798 HENDRA KUSNADI POL-177091', 7617000, 'BCA/UAT/001'],
    [d, 'TRANSFER 000798 HENDRA KUSNADI', 5000000, 'BCA/UAT/002'],
    [d, 'BIAYA ADMIN BULANAN', 15000, 'BCA/UAT/003'],
  ])
}

async function doImport() {
  busy.value = true
  try {
    const bt = payable.value.find(b => b.batch.id === imp.value.batchId)?.batch
    const periodLabel = bt ? docNo(bt, 'batchNo') + ' — ' + bt.name : ''
    impResult.value = await api('bank-imports', { method: 'POST', body: { bankName: imp.value.bankName, periodLabel, hospitalId: bt?.hospitalId || state.user?.hospitalId || state.hospitalFilter || undefined, lines: impRows.value } })
    toast('Impor selesai — buka tab Rekonsiliasi.', 'success')
    impRows.value = []
    impFile.value = ''
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
    toast('Ter Rekonsiliasi → pembayaran ' + docNo(res.payment, 'paymentNo') + ' dibuat, status klaim → ' + CLAIM_LABELS[res.claimStatus] + '.', 'success')
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

watch(selLine, (l) => { if (l && canApprove) pickLine(l) })

onMounted(async () => {
  await loadBoot()
  await loadPayments()
  await refreshImports()
  await loadLines()
})
</script>
