<template>
  <main class="page">
    <PageHeader icon="layers" title="Batch &amp; Pengajuan" sub="Kelompokkan klaim siap diajukan per penjamin/periode. Menghapus batch tidak pernah menghapus klaim. Klaim dengan blocker terbuka tidak dapat masuk batch.">
      <template #actions>
        <button class="btn btn-primary" v-if="hasPerm('CREATE') && canWriteBatch()" @click="showNew = true"><Icon name="plus" :size="14" /> Batch Baru</button>
      </template>
    </PageHeader>

    <div class="grid-31">
      <div>
        <div class="tbl-wrap">
          <table class="tbl">
            <thead><tr><th>Batch</th><th>Periode</th><th class="num">Klaim</th><th class="num">Total Gap</th><th>Dibuat</th><th></th></tr></thead>
            <tbody>
              <tr v-for="b in batches" :key="b.batch.id" @click="select(b.batch.id)" :style="selected === b.batch.id ? 'background:var(--teal-bg)' : ''">
                <td><b>{{ b.batch.name }}</b> <span class="mono" style="font-size:11px;color:var(--text-muted)">{{ docNo(b.batch, 'batchNo') }}</span><div style="font-size:11.5px;color:var(--text-muted)">{{ b.batch.note || '—' }}</div></td>
                <td style="font-size:12px">{{ periodLabel(b.batch) }}</td>
                <td class="num">{{ b.claimCount }}</td>
                <td class="num" style="font-weight:700">{{ formatRpShort(b.totalGap) }}</td>
                <td style="font-size:12px">{{ formatDate(b.batch.createdAt) }}<br /><span style="color:var(--text-muted);font-size:11px">{{ b.batch.createdBy }}</span></td>
                <td><button class="btn btn-danger btn-xs" v-if="hasPerm('DELETE') && canWriteBatch()" @click.stop="askDelete(b.batch.id)">Hapus</button></td>
              </tr>
              <tr v-if="!batches.length"><td colspan="6"><Empty icon="▥" title="Belum ada batch" desc="Buat batch untuk mengelompokkan klaim sebelum diajukan ke penjamin." /></td></tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="card card-pad" v-if="detail">
        <div class="card-title">{{ detail.batch.name }}</div>
        <div class="card-sub">Periode {{ periodLabel(detail.batch) }} · {{ detail.claims.length }} klaim · Total gap {{ formatRp(detail.totalGap) }} <span v-if="detail.blockerHits" style="color:var(--danger)">· {{ detail.blockerHits }} klaim ber-blocker</span></div>
        <div class="chips-row" style="margin-bottom:12px">
          <button class="btn btn-outline btn-sm" @click="showAdd = true" v-if="hasPerm('EDIT') && canWriteBatch()">+ Tambah Klaim</button>
          <button class="btn btn-teal btn-sm" v-if="hasPerm('APPROVE') && canWriteBatch() && readyCount" :disabled="busy || detail.blockerHits > 0" :title="detail.blockerHits > 0 ? 'Selesaikan blocker terlebih dahulu' : ''" @click="submitAll">Ajukan Semua ({{ readyCount }})</button>
          <button class="btn btn-outline btn-sm" @click="exportCsv">Ekspor CSV</button>
          <button class="btn btn-outline btn-sm" @click="exportPdf">Ekspor PDF (landscape)</button>
        </div>
        <div class="alert danger" v-if="detail.blockerHits > 0" style="margin-bottom:12px"><span>⛔</span><span>Batch belum bisa diajukan: {{ detail.blockerHits }} klaim masih punya blocker terbuka. Selesaikan blocker di klaim tersebut, atau keluarkan klaimnya dari batch.</span></div>
        <div class="tbl-wrap" style="max-height:400px;overflow-y:auto">
          <table class="tbl" style="min-width:420px">
            <thead><tr><th>Klaim</th><th>Pasien</th><th>Status</th><th class="num">Gap</th><th></th></tr></thead>
            <tbody>
              <tr v-for="c in detail.claims" :key="c.id">
                <td class="mono">{{ docNo(c) }}</td>
                <td>{{ c.patientName }}</td>
                <td><span class="badge" :class="CLAIM_TONE[c.status]">{{ CLAIM_LABELS[c.status] }}</span></td>
                <td class="num">{{ formatRpShort(gapOf(c)) }}</td>
                <td><button class="btn btn-outline btn-xs" v-if="hasPerm('EDIT') && canWriteBatch()" @click="removeClaim(c.id)">✕</button></td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="note-strip">Ekspor CSV memuat seluruh 18 komponen tarif agar penerima dapat memverifikasi rantai tarif: INA-CBG → Top Up → Tarif JKN → Tagihan RS → Gap. PDF landscape dibuat lewat dialog cetak browser (pilih "Simpan sebagai PDF").</div>
      </div>
      <div class="card card-pad" v-else>
        <Empty icon="←" title="Pilih batch" desc="Klik baris batch untuk melihat detail, menambah/mengeluarkan klaim, dan mengekspor." />
      </div>
    </div>

    <Modal v-if="showNew" title="Batch Baru" @close="showNew = false">
      <div class="field">
        <label>Penjamin (Asuransi) <span class="req">*</span></label>
        <SearchSelect ref="insSel" :options="insurerOptions" v-model="newBatch.insurerId" placeholder="Cari nama penjamin…" />
      </div>
      <div class="form-grid">
        <div class="field"><label>Tanggal Mulai <span class="req">*</span></label><input class="input" type="date" v-model="newBatch.dateFrom" /></div>
        <div class="field"><label>Tanggal Akhir <span class="req">*</span></label><input class="input" type="date" v-model="newBatch.dateTo" /></div>
      </div>
      <div class="hint" style="margin:-4px 0 12px">Nama batch dibuat otomatis dari penjamin &amp; tanggal. Klaim penjamin ini yang <b>tanggal pulangnya</b> dalam rentang tersebut (status Draft/Siap Diajukan, belum ber-batch) otomatis masuk ke batch. Klaim yang masih punya blocker akan <b>otomatis menyusul</b> begitu blockernya selesai.</div>
      <div class="field">
        <label>Kode Faskes (tidak dapat diubah setelah dibuat) <span class="req">*</span></label>
        <select class="input" v-model="newBatch.hospitalId" :disabled="!isSuper">
          <option v-for="h in state.boot?.hospitals || []" :key="h.id" :value="h.id">{{ h.kodeFaskes }} — {{ h.name }}</option>
        </select>
      </div>
      <div class="field"><label>Catatan</label><input class="input" v-model="newBatch.note" /></div>
      <template #footer>
        <button class="btn btn-outline" @click="showNew = false">Batal</button>
        <button class="btn btn-teal" :disabled="!newBatch.insurerId || !newBatch.dateFrom || !newBatch.dateTo || newBatch.dateTo < newBatch.dateFrom || !newBatch.hospitalId || busy" @click="create">Buat Batch</button>
      </template>
    </Modal>

    <Modal v-if="showAdd" title="Tambah Klaim ke Batch" wide @close="showAdd = false">
      <p class="hint" style="margin-bottom:10px">Hanya klaim <b v-if="detail.batch.insurerId">penjamin {{ insurerName(detail.batch.insurerId) }}</b><span v-else>satu RS yang sama</span>, belum tergabung batch, dan tanpa blocker terbuka. Klaim ber-blocker ditolak otomatis. Klaim penjamin ini yang sudah bebas blocker dan tanggal pulangnya dalam periode batch juga masuk <b>otomatis</b>.</p>
      <div class="tbl-wrap" style="max-height:320px;overflow-y:auto">
        <table class="tbl" style="min-width:520px">
          <thead><tr><th></th><th>Klaim</th><th>Pasien</th><th>Penjamin</th><th class="num">Gap</th><th>Kesiapan</th></tr></thead>
          <tbody>
            <tr v-for="c in detail.unbatched" :key="c.id">
              <td><input type="checkbox" v-model="addSel" :value="c.id" :aria-label="'Pilih ' + docNo(c)" /></td>
              <td class="mono">{{ docNo(c) }}</td>
              <td>{{ c.patientName }}</td>
              <td style="font-size:12px">{{ insurerName(c.insurerId) }}</td>
              <td class="num">{{ formatRpShort(gapOf(c)) }}</td>
              <td><ScoreRing :score="c.readinessScore" /></td>
            </tr>
            <tr v-if="!detail.unbatched.length"><td colspan="6" style="text-align:center;color:var(--text-muted)">Tidak ada klaim belum ter-batch</td></tr>
          </tbody>
        </table>
      </div>
      <template #footer>
        <button class="btn btn-outline" @click="showAdd = false">Batal</button>
        <button class="btn btn-teal" :disabled="!addSel.length" @click="addClaims">Tambahkan {{ addSel.length ? '(' + addSel.length + ')' : '' }}</button>
      </template>
    </Modal>
  </main>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api'
import { useRoute } from 'vue-router'
import { state, hasPerm, toast, loadBoot } from '../store'
import { canWriteBatch } from '../access'
import Modal from '../components/Modal.vue'
import PageHeader from '../components/PageHeader.vue'
import Icon from '../components/Icon.vue'
import Empty from '../components/Empty.vue'
import SearchSelect from '../components/SearchSelect.vue'
import ScoreRing from '../components/ScoreRing.vue'
import { formatRp, formatRpShort, formatDate, sumTariff, jknTotal, CLAIM_LABELS, CLAIM_TONE, downloadBlobCsv, docNo } from '../format'

const batches = ref([])
const selected = ref('')
const detail = ref(null)
const showNew = ref(false)
const showAdd = ref(false)
const addSel = ref([])
const busy = ref(false)
const insSel = ref(null)
const monthStart = () => new Date().toISOString().slice(0, 8) + '01'
const blankBatch = () => ({ insurerId: '', dateFrom: monthStart(), dateTo: new Date().toISOString().slice(0, 10), hospitalId: state.user?.hospitalId || '', note: '' })
const newBatch = ref({ ...blankBatch(), hospitalId: '' })
const insurerOptions = computed(() => (state.boot?.insurers || []).map(i => ({ id: i.id, label: i.name, sub: i.type })))
function periodLabel(b) { return b.periodStart ? formatDate(b.periodStart) + ' – ' + formatDate(b.periodEnd) : b.period }
const isSuper = computed(() => state.user?.role === 'SUPER_ADMIN')

async function load() {
  batches.value = await api('batches')
  if (selected.value) detail.value = await api('batches/' + selected.value)
}
function select(id) { selected.value = id; load() }
function insurerName(id) { return state.boot?.insurers.find(i => i.id === id)?.name || '—' }
function gapOf(c) { return sumTariff(c.hospitalTariff) - jknTotal(c.inacbgBaseTariff, c.specialCmg) }

async function create() {
  busy.value = true
  try {
    const b = await api('batches', { method: 'POST', body: newBatch.value })
    toast('Batch ' + docNo(b, 'batchNo') + ' dibuat: ' + (b.autoAdded || 0) + ' klaim otomatis masuk' + (b.autoBlocked ? ', ' + b.autoBlocked + ' klaim ber-blocker tidak dimasukkan.' : '.'), b.autoBlocked ? 'warning' : 'success', 6000)
    showNew.value = false
    newBatch.value = { ...blankBatch(), hospitalId: newBatch.value.hospitalId }
    insSel.value?.setText('')
    selected.value = b.id
    await load()
  } catch (e) { toast(e.message, 'error') } finally { busy.value = false }
}

async function addClaims() {
  try {
    const res = await api('batches/' + selected.value + '/claims', { method: 'POST', body: { claimIds: addSel.value } })
    const extra = (res.blocked ? ', ' + res.blocked + ' ditolak karena blocker' : '') + (res.otherInsurer ? ', ' + res.otherInsurer + ' ditolak karena beda penjamin' : '')
    toast(res.added + ' klaim ditambahkan' + (extra ? extra + '.' : '.'), extra ? 'warning' : 'success')
    addSel.value = []
    showAdd.value = false
    await load()
  } catch (e) { toast(e.message, 'error') }
}

async function removeClaim(cid) {
  try {
    await api('batches/' + selected.value + '/claims/' + cid, { method: 'DELETE' })
    toast('Klaim dikeluarkan dari batch (klaim tidak dihapus).', 'success')
    await load()
  } catch (e) { toast(e.message, 'error') }
}

function askDelete(id) {
  if (!confirm('Hapus batch? Klaim di dalamnya TIDAK ikut terhapus (batchId dilepas).')) return
  api('batches/' + id, { method: 'DELETE' }).then(() => { toast('Batch dihapus.', 'success'); selected.value = ''; load() }).catch(e => toast(e.message, 'error'))
}

function exportCsv() {
  const rows = [['ID', 'SEP', 'Pasien', 'Penjamin', 'Kode INA-CBG', 'Tarif Murni', 'Top Up', 'Total JKN', 'Total RS', 'Gap']]
  for (const c of detail.value.claims) {
    rows.push([docNo(c), c.sepNo, c.patientName, insurerName(c.insurerId), c.inacbgCode || '',
      c.inacbgBaseTariff, Object.values(c.specialCmg).reduce((a, b) => a + b, 0),
      jknTotal(c.inacbgBaseTariff, c.specialCmg), sumTariff(c.hospitalTariff), gapOf(c)])
  }
  downloadBlobCsv('batch-' + docNo(detail.value.batch, 'batchNo') + '.csv', rows)
}

const readyCount = computed(() => (detail.value?.claims || []).filter(c => c.status === 'ready_to_submit').length)

// F-BAT-04: ajukan seluruh klaim Siap Diajukan di batch (maker-checker dicek backend).
async function submitAll() {
  if (!confirm('Ajukan ' + readyCount.value + ' klaim Siap Diajukan di batch ini ke penjamin? Tindakan tercatat di audit trail.')) return
  busy.value = true
  try {
    const res = await api('batches/' + selected.value + '/submit', { method: 'POST' })
    const skip = res.skipped || []
    toast(res.submitted.length + ' klaim diajukan' + (skip.length ? ', ' + skip.length + ' dilewati: ' + skip.map(s => (s.claimNo || s.id) + ' (' + s.reason + ')').join('; ') : '.'), skip.length ? 'warning' : 'success', 7000)
    await load()
  } catch (e) { toast(e.message, 'error') } finally { busy.value = false }
}

// Ekspor PDF: dokumen batch landscape A4 dibuka di jendela cetak browser (Simpan sebagai PDF).
function exportPdf() {
  const b = detail.value.batch
  const esc = (v) => String(v ?? '').replace(/[&<>"]/g, (ch) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[ch]))
  const rows = detail.value.claims.map(c => `<tr><td>${esc(docNo(c))}</td><td>${esc(c.sepNo)}</td><td>${esc(c.patientName)}</td>
    <td>${esc(insurerName(c.insurerId))}</td><td>${esc(c.inacbgCode || '-')}</td><td class="n">${esc(formatRp(jknTotal(c.inacbgBaseTariff, c.specialCmg)))}</td>
    <td class="n">${esc(formatRp(sumTariff(c.hospitalTariff)))}</td><td class="n">${esc(formatRp(gapOf(c)))}</td></tr>`).join('')
  const w = window.open('', '_blank')
  if (!w) { toast('Pop-up diblokir browser — izinkan pop-up untuk mengekspor PDF.', 'warning'); return }
  w.document.write(`<!doctype html><html><head><meta charset="utf-8"><title>Batch ${esc(docNo(b, 'batchNo'))}</title>
    <style>@page{size:A4 landscape;margin:12mm}body{font-family:Arial,sans-serif;font-size:11px;color:#111}
    h1{font-size:16px;margin:0 0 4px}table{width:100%;border-collapse:collapse;margin-top:10px}
    th,td{border:1px solid #999;padding:4px 6px;text-align:left}th{background:#eee}.n{text-align:right}</style></head>
    <body><h1>${esc(b.name)} (${esc(docNo(b, 'batchNo'))})</h1>
    <div>Periode ${esc(b.period)} · ${detail.value.claims.length} klaim · Total gap ${esc(formatRp(detail.value.totalGap))}</div>
    <table><thead><tr><th>Klaim</th><th>SEP</th><th>Pasien</th><th>Penjamin</th><th>INA-CBG</th><th class="n">Tarif JKN</th>
    <th class="n">Tagihan RS</th><th class="n">Gap</th></tr></thead><tbody>${rows}</tbody></table></body></html>`)
  w.document.close()
  w.focus()
  w.print()
}

onMounted(async () => {
  await loadBoot()
  newBatch.value.hospitalId = state.user?.hospitalId || state.hospitalFilter || state.boot?.hospitals[0]?.id || ''
  await load()
  // Dibuka dari drill-down Piutang: ?open=<id batch>
  const open = useRoute().query.open
  if (open) select(String(open))
})
</script>
