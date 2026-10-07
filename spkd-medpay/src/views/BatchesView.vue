<template>
  <main class="page">
    <PageHeader icon="layers" title="Batch &amp; Pengajuan" sub="Kelompokkan klaim siap diajukan per penjamin/periode. Menghapus batch tidak pernah menghapus klaim. Klaim dengan blocker terbuka tidak dapat masuk batch.">
      <template #actions>
        <button class="btn btn-primary" v-if="hasPerm('CREATE')" @click="showNew = true"><Icon name="plus" :size="14" /> Batch Baru</button>
      </template>
    </PageHeader>

    <div class="grid-31">
      <div>
        <div class="tbl-wrap">
          <table class="tbl">
            <thead><tr><th>Batch</th><th>Periode</th><th class="num">Klaim</th><th class="num">Total Gap</th><th>Dibuat</th><th></th></tr></thead>
            <tbody>
              <tr v-for="b in batches" :key="b.batch.id" @click="select(b.batch.id)" :style="selected === b.batch.id ? 'background:var(--teal-bg)' : ''">
                <td><b>{{ b.batch.name }}</b><div style="font-size:11.5px;color:var(--text-muted)">{{ b.batch.note || '—' }}</div></td>
                <td>{{ b.batch.period }}</td>
                <td class="num">{{ b.claimCount }}</td>
                <td class="num" style="font-weight:700">{{ formatRpShort(b.totalGap) }}</td>
                <td style="font-size:12px">{{ formatDate(b.batch.createdAt) }}<br /><span style="color:var(--text-muted);font-size:11px">{{ b.batch.createdBy }}</span></td>
                <td><button class="btn btn-danger btn-xs" v-if="hasPerm('DELETE')" @click.stop="askDelete(b.batch.id)">Hapus</button></td>
              </tr>
              <tr v-if="!batches.length"><td colspan="6"><Empty icon="▥" title="Belum ada batch" desc="Buat batch untuk mengelompokkan klaim sebelum diajukan ke penjamin." /></td></tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="card card-pad" v-if="detail">
        <div class="card-title">{{ detail.batch.name }}</div>
        <div class="card-sub">Periode {{ detail.batch.period }} · {{ detail.claims.length }} klaim · Total gap {{ formatRp(detail.totalGap) }} <span v-if="detail.blockerHits" style="color:var(--danger)">· {{ detail.blockerHits }} klaim ber-blocker</span></div>
        <div class="chips-row" style="margin-bottom:12px">
          <button class="btn btn-outline btn-sm" @click="showAdd = true" v-if="hasPerm('EDIT')">+ Tambah Klaim</button>
          <button class="btn btn-outline btn-sm" @click="exportCsv">Ekspor CSV</button>
          <button class="btn btn-outline btn-sm" @click="exportPdf">Ekspor PDF (landscape)</button>
        </div>
        <div class="tbl-wrap" style="max-height:400px;overflow-y:auto">
          <table class="tbl" style="min-width:420px">
            <thead><tr><th>Klaim</th><th>Pasien</th><th>Status</th><th class="num">Gap</th><th></th></tr></thead>
            <tbody>
              <tr v-for="c in detail.claims" :key="c.id">
                <td class="mono">{{ c.id }}</td>
                <td>{{ c.patientName }}</td>
                <td><span class="badge" :class="CLAIM_TONE[c.status]">{{ CLAIM_LABELS[c.status] }}</span></td>
                <td class="num">{{ formatRpShort(gapOf(c)) }}</td>
                <td><button class="btn btn-outline btn-xs" v-if="hasPerm('EDIT')" @click="removeClaim(c.id)">✕</button></td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="note-strip">Ekspor CSV memuat seluruh 18 komponen tarif agar penerima dapat memverifikasi rantai tarif: INA-CBG → Top Up → Tarif JKN → Tagihan RS → Gap. PDF landscape: pratinjau demo (generator server menyusul).</div>
      </div>
      <div class="card card-pad" v-else>
        <Empty icon="←" title="Pilih batch" desc="Klik baris batch untuk melihat detail, menambah/mengeluarkan klaim, dan mengekspor." />
      </div>
    </div>

    <Modal v-if="showNew" title="Batch Baru" @close="showNew = false">
      <div class="field"><label>Nama Batch <span class="req">*</span></label><input class="input" v-model="newBatch.name" placeholder="Batch Penjamin Q2 — Kiriman 1" /></div>
      <div class="field"><label>Periode <span class="req">*</span></label><input class="input" v-model="newBatch.period" placeholder="2026-06" /></div>
      <div class="field">
        <label>Kode Faskes (tidak dapat diubah setelah dibuat) <span class="req">*</span></label>
        <select class="input" v-model="newBatch.hospitalId" :disabled="!isSuper">
          <option v-for="h in state.boot?.hospitals || []" :key="h.id" :value="h.id">{{ h.kodeFaskes }} — {{ h.name }}</option>
        </select>
      </div>
      <div class="field"><label>Catatan</label><input class="input" v-model="newBatch.note" /></div>
      <template #footer>
        <button class="btn btn-outline" @click="showNew = false">Batal</button>
        <button class="btn btn-teal" :disabled="!newBatch.name || !newBatch.period || !newBatch.hospitalId || busy" @click="create">Buat Batch</button>
      </template>
    </Modal>

    <Modal v-if="showAdd" title="Tambah Klaim ke Batch" wide @close="showAdd = false">
      <p class="hint" style="margin-bottom:10px">Hanya klaim satu RS yang sama, belum tergabung batch, dan tanpa blocker terbuka. Klaim ber-blocker ditolak otomatis.</p>
      <div class="tbl-wrap" style="max-height:320px;overflow-y:auto">
        <table class="tbl" style="min-width:520px">
          <thead><tr><th></th><th>Klaim</th><th>Pasien</th><th>Penjamin</th><th class="num">Gap</th><th>Kesiapan</th></tr></thead>
          <tbody>
            <tr v-for="c in detail.unbatched" :key="c.id">
              <td><input type="checkbox" v-model="addSel" :value="c.id" :aria-label="'Pilih ' + c.id" /></td>
              <td class="mono">{{ c.id }}</td>
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
import { state, hasPerm, toast, loadBoot } from '../store'
import Modal from '../components/Modal.vue'
import PageHeader from '../components/PageHeader.vue'
import Icon from '../components/Icon.vue'
import Empty from '../components/Empty.vue'
import ScoreRing from '../components/ScoreRing.vue'
import { formatRp, formatRpShort, formatDate, sumTariff, jknTotal, CLAIM_LABELS, CLAIM_TONE, downloadBlobCsv } from '../format'

const batches = ref([])
const selected = ref('')
const detail = ref(null)
const showNew = ref(false)
const showAdd = ref(false)
const addSel = ref([])
const busy = ref(false)
const newBatch = ref({ name: '', period: '', hospitalId: '', note: '' })
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
    toast('Batch ' + b.id + ' dibuat.', 'success')
    showNew.value = false
    newBatch.value = { name: '', period: '', hospitalId: state.user?.hospitalId || '', note: '' }
    selected.value = b.id
    await load()
  } catch (e) { toast(e.message, 'error') } finally { busy.value = false }
}

async function addClaims() {
  try {
    const res = await api('batches/' + selected.value + '/claims', { method: 'POST', body: { claimIds: addSel.value } })
    toast(res.added + ' klaim ditambahkan' + (res.blocked ? ', ' + res.blocked + ' ditolak karena blocker.' : '.'), res.blocked ? 'warning' : 'success')
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
    rows.push([c.id, c.sepNo, c.patientName, insurerName(c.insurerId), c.inacbgCode || '',
      c.inacbgBaseTariff, Object.values(c.specialCmg).reduce((a, b) => a + b, 0),
      jknTotal(c.inacbgBaseTariff, c.specialCmg), sumTariff(c.hospitalTariff), gapOf(c)])
  }
  downloadBlobCsv('batch-' + detail.value.batch.id + '.csv', rows)
}

function exportPdf() {
  toast('Ekspor PDF landscape A4 — pratinjau demo. Implementasi generator PDF server tersedia pada fase berikutnya.', 'info')
}

onMounted(async () => {
  await loadBoot()
  newBatch.value.hospitalId = state.user?.hospitalId || state.boot?.hospitals[0]?.id || ''
  await load()
})
</script>
