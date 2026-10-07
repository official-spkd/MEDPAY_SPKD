<template>
  <Teleport to="body">
    <div class="drawer-overlay" @click="$emit('close')" />
    <div class="drawer" role="dialog" aria-label="Detail klaim">
      <div class="drawer-head" v-if="d">
        <div>
          <div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap">
            <h3 style="font-size:17px">{{ d.claim.id }}</h3>
            <span class="badge" :class="CLAIM_TONE[d.claim.status]">{{ CLAIM_LABELS[d.claim.status] }}</span>
            <ScoreRing :score="d.claim.readinessScore" with-label />
          </div>
          <div style="font-size:12.5px;color:var(--text-muted);margin-top:4px">
            {{ d.claim.patientName }} · {{ d.claim.sepNo }} · {{ d.claim.inacbgCode || 'INA-CBG belum resolve' }}
          </div>
        </div>
        <button class="x-btn" @click="$emit('close')" aria-label="Tutup">✕</button>
      </div>
      <div class="drawer-body" v-if="d">
        <div class="tabs">
          <button class="tab" :class="{ active: tab === 'ringkasan' }" @click="tab = 'ringkasan'">Ringkasan</button>
          <button class="tab" :class="{ active: tab === 'tarif' }" @click="tab = 'tarif'">Rantai Tarif</button>
          <button class="tab" :class="{ active: tab === 'resume' }" @click="tab = 'resume'">Resume Medis</button>
          <button class="tab" :class="{ active: tab === 'pembayaran' }" @click="tab = 'pembayaran'">Pembayaran</button>
          <button class="tab" :class="{ active: tab === 'riwayat' }" @click="tab = 'riwayat'">Riwayat & Validasi</button>
        </div>

        <!-- Ringkasan -->
        <div v-if="tab === 'ringkasan'">
          <div class="kv">
            <span class="k">RS / Kode Faskes</span><span class="v">{{ d.hospital?.name }} ({{ d.hospital?.kodeFaskes }})</span>
            <span class="k">Jenis Peserta</span><span class="v">{{ d.claim.participantType === 'PBI' ? 'PBI — tidak eligible COB' : 'Non-PBI' }}</span>
            <span class="k">No. Kartu BPJS</span><span class="v mono">{{ mask(d.claim.bpjsCardNo) }}</span>
            <span class="k">MRN</span><span class="v">{{ d.claim.mrn }}</span>
            <span class="k">DPJP</span><span class="v">{{ d.claim.dpjp }}</span>
            <span class="k">Jenis Rawat</span><span class="v">{{ d.claim.careType === 'RAWAT_INAP' ? 'Rawat Inap' : 'Rawat Jalan' }} · Kelas {{ d.claim.careClass }}</span>
            <span class="k">Masuk → Pulang</span><span class="v">{{ formatDate(d.claim.admissionDate) }} → {{ formatDate(d.claim.dischargeDate) }} ({{ d.claim.los }} hari, otomatis)</span>
            <span class="k">Diagnosa Utama</span><span class="v mono">{{ d.claim.primaryDiagnosis }}</span>
            <span class="k">Diagnosa Sekunder</span><span class="v mono">{{ d.claim.secondaryDiagnoses.join(', ') || '—' }}</span>
            <span class="k">Prosedur (ICD-9-CM)</span><span class="v mono">{{ d.claim.procedures.join(', ') || '—' }}</span>
            <span class="k">Deadline Pengajuan</span><span class="v">{{ formatDate(deadline.deadline) }} <span class="badge" :class="zoneTone">{{ zoneLabel }}</span> <span class="verify-tag">[VR]</span></span>
            <span class="k">Penjamin Kedua</span><span class="v">{{ d.insurer?.name || '—' }}</span>
            <span class="k">No. Polis / Tertanggung</span><span class="v">{{ d.claim.policyNo || '—' }} / {{ d.claim.insuredName || '—' }}</span>
            <span class="k">Plafon Polis</span><span class="v">{{ d.claim.policyCeiling ? formatRp(d.claim.policyCeiling) : 'Tidak dibatasi' }}</span>
            <span class="k">Jenis Perlindungan</span><span class="v">{{ d.claim.coverageType || '—' }}</span>
          </div>
          <div v-if="d.claim.rejectionReasonCode" class="alert danger" style="margin-top:14px">
            <span>⛔</span>
            <span><b>Ditolak ({{ d.claim.rejectionReasonCode }}):</b> {{ d.claim.rejectionReasonNote || '—' }}</span>
          </div>
        </div>

        <!-- Rantai tarif -->
        <div v-if="tab === 'tarif'">
          <div class="card card-pad" style="margin-bottom:12px">
            <div class="card-title">Rantai Tarif — Explainable</div>
            <div class="card-sub">Tarif INA-CBG murni (grouper) + Special CMG/Top Up = Tarif JKN. Gap = Σ 18 komponen tarif RS − Tarif JKN.</div>
            <div style="display:grid;grid-template-columns:1fr 1fr;gap:16px" class="two-panel">
              <div>
                <div class="tarif-row"><span>Tarif INA-CBG Murni</span><b>{{ formatRp(d.claim.inacbgBaseTariff) }}</b></div>
                <div v-for="k in SPECIAL_CMG_KEYS" :key="k" class="tarif-row" v-show="d.claim.specialCmg[k]">
                  <span>{{ SPECIAL_CMG_LABELS[k] }}</span><span>{{ formatRp(d.claim.specialCmg[k]) }}</span>
                </div>
                <div class="tarif-row total"><span>Total Tarif JKN <small>(dibayar BPJS)</small></span><span>{{ formatRp(jkn) }}</span></div>
              </div>
              <div>
                <div v-for="k in TARIFF_COMPONENT_KEYS" :key="k" class="tarif-row" v-show="d.claim.hospitalTariff[k]">
                  <span>{{ COMPONENT_LABELS[k] }}</span><span>{{ formatRp(d.claim.hospitalTariff[k]) }}</span>
                </div>
                <div class="tarif-row total"><span>Total Tagihan RS (18 komponen)</span><span>{{ formatRp(hosp) }}</span></div>
              </div>
            </div>
            <div class="gap-box" :class="gap >= 0 ? 'pos' : 'neg'">
              <span>{{ gap >= 0 ? 'Selisih Tarif (Gap) — potensi piutang' : 'Negatif: tagihan di bawah tarif JKN' }}</span>
              <span>{{ formatRp(gap) }}</span>
            </div>
            <div class="note-strip">Angka gap adalah potensi piutang, bukan pendapatan pasti — BPJS membayar sampai Total Tarif JKN dan plafon penjamin kedua mungkin membatasi. Sumber: klaim {{ d.claim.id }} (modul Klaim).</div>
          </div>
          <div v-if="hasPerm('RESOLVE')" class="chips-row">
            <button class="btn btn-outline btn-sm" :disabled="resolving" @click="resolveTariff"><span v-if="resolving" class="spin" /> Resolve Ulang Tarif INA-CBG</button>
          </div>
        </div>

        <!-- Resume medis -->
        <div v-if="tab === 'resume'">
          <div class="filterbar">
            <label class="btn btn-outline btn-sm" v-if="hasPerm('CREATE')">
              + Unggah CSV
              <input type="file" accept=".csv,text/csv" style="display:none" @change="onCsv" />
            </label>
            <span class="hint" v-if="resumeCount">{{ resumeReviewed }}/{{ resumeCount }} item direview</span>
          </div>
          <div class="alert info" v-if="hasPerm('CREATE')"><span>📄</span><span>Format CSV: <code>sep;kategori;nama_item;sub_kategori;tanggal;hasil_catatan;jumlah;satuan;harga_satuan</code>. Hanya baris dengan SEP yang cocok dengan klaim ini yang diimpor.</span></div>
          <div v-if="!d.resume.length"><Empty icon="📁" title="Belum ada resume medis" desc="Unggah CSV resume medis (7 kategori) dan review setiap item sebelum generate paket dokumen." /></div>
          <div v-else>
            <div class="tbl-wrap" style="max-height:340px;overflow-y:auto">
              <table class="tbl" style="min-width:560px">
                <thead><tr><th>Kategori</th><th>Item</th><th>Tanggal</th><th class="num">Subtotal</th><th>Reviewed</th></tr></thead>
                <tbody>
                  <tr v-for="r in d.resume" :key="r.id">
                    <td><span class="badge neutral">{{ catLabel(r.kategori) }}</span></td>
                    <td>
                      <div style="font-weight:600">{{ r.namaItem }}</div>
                      <div style="font-size:11.5px;color:var(--text-muted)">{{ r.hasilCatatan }}</div>
                    </td>
                    <td style="font-size:12px">{{ formatDate(r.tanggal) }}</td>
                    <td class="num">{{ formatRp(r.subtotal) }}</td>
                    <td>
                      <input type="checkbox" :checked="r.reviewed" :disabled="!hasPerm('EDIT')" @change="toggleReview(r, $event)" :aria-label="'Review item ' + r.namaItem" />
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div class="alert" :class="allReviewed ? 'success' : 'warning'" style="margin-top:12px">
              <span>{{ allReviewed ? '✓' : '⏳' }}</span>
              <span>{{ allReviewed ? 'Seluruh item sudah direview — Generate PDF paket dokumen diaktifkan (label demo).' : 'Masih ada item yang belum direview — generate PDF terkunci sampai seluruh item direview.' }}</span>
            </div>
            <button class="btn btn-teal btn-sm" :disabled="!allReviewed" @click="toast('Generate PDF paket dokumen — stub demo (implementasi generator menyusul).', 'info')">Generate PDF Paket Dokumen</button>
          </div>
        </div>

        <!-- Pembayaran -->
        <div v-if="tab === 'pembayaran'">
          <div class="kpi-grid" style="grid-template-columns:repeat(3,1fr)">
            <Kpi label="Nilai Gap" :value="formatRp(d.gap)" tone="navy" />
            <Kpi label="Sudah Dibayar" :value="formatRp(paidSum)" tone="success" val-class="success" />
            <Kpi label="Tertunggak" :value="formatRp(Math.max(0, d.gap - paidSum))" tone="warning" val-class="warning" />
          </div>
          <div class="tbl-wrap" v-if="d.payments.length">
            <table class="tbl" style="min-width:520px">
              <thead><tr><th>ID</th><th>Tanggal</th><th class="num">Nominal</th><th>Payer</th><th>Ref</th><th>Oleh</th></tr></thead>
              <tbody>
                <tr v-for="p in d.payments" :key="p.id">
                  <td class="mono">{{ p.id }}</td>
                  <td>{{ formatDate(p.receivedDate) }}</td>
                  <td class="num" style="font-weight:700">{{ formatRp(p.amount) }}</td>
                  <td>{{ p.payer }}</td>
                  <td class="mono" style="font-size:11px">{{ p.transferRef || '—' }}</td>
                  <td style="font-size:12px">{{ p.createdBy }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <Empty v-else icon="💰" title="Belum ada pembayaran" desc="Catat pembayaran dari modul Pembayaran setelah klaim disetujui penjamin." />
          <router-link v-if="canPay" to="/payments" class="btn btn-teal btn-sm" style="margin-top:10px">Catat Pembayaran</router-link>
        </div>

        <!-- Riwayat & validasi -->
        <div v-if="tab === 'riwayat'">
          <div class="card card-pad" style="margin-bottom:12px">
            <div class="card-title">Hasil Validasi (Scrubber)</div>
            <div class="card-sub">Dijalankan otomatis saat buka klaim. BLOCKER terbuka memblokir masuk batch & pengajuan.</div>
            <div v-if="!d.claim.scrubFindings.length"><Empty icon="✅" title="Bersih" desc="Tidak ada temuan aturan." /></div>
            <div v-for="f in d.claim.scrubFindings" :key="f.ruleId" class="alert" :class="f.severity === 'BLOCKER' ? 'danger' : 'warning'" style="margin-bottom:8px">
              <span>{{ f.severity === 'BLOCKER' ? '⛔' : '⚠' }}</span>
              <span>
                <b>{{ f.ruleId }}</b> — {{ f.message }}
                <div style="font-size:12px;margin-top:3px">Saran: {{ f.fixSuggestion }}</div>
                <div style="font-size:11px;color:var(--text-muted);margin-top:2px">Sumber: {{ f.sourceRef }}</div>
                <div v-if="f.overridden" style="margin-top:4px"><span class="badge warning">Di-override Admin RS</span></div>
                <button v-else-if="hasPerm('OVERRIDE') && f.severity === 'BLOCKER'" class="btn btn-outline btn-xs" style="margin-top:6px" @click="askOverride(f)">Override dengan Alasan</button>
              </span>
            </div>
          </div>
          <div class="card card-pad">
            <div class="card-title">Riwayat Status (append-only)</div>
            <div v-for="(h, i) in [...d.claim.statusHistory].reverse()" :key="i" style="display:flex;gap:10px;padding:7px 0;border-bottom:1px dashed var(--border);font-size:12.5px">
              <span style="color:var(--text-muted);min-width:130px">{{ formatDateTime(h.at) }}</span>
              <span><b>{{ h.actor }}</b>: {{ h.from ? h.from + ' → ' : '' }}{{ h.to }} {{ h.note ? '— ' + h.note : '' }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="drawer-foot" v-if="d">
        <template v-for="t in availableTransitions" :key="t">
          <button class="btn" :class="transBtnClass(t)" @click="askTransition(t)">{{ transLabel(t) }}</button>
        </template>
        <router-link v-if="canEdit" :to="'/claims/' + d.claim.id + '/edit'" class="btn btn-outline">Edit Klaim</router-link>
        <button v-if="hasPerm('DELETE')" class="btn btn-danger" @click="askDelete">Hapus</button>
        <span style="flex:1" />
        <span style="font-size:11px;color:var(--text-muted);align-self:center">Aksi diaudit (append-only)</span>
      </div>
    </div>

    <!-- Modal konfirmasi transisi -->
    <Modal v-if="confirmState" :title="confirmState.title" @close="confirmState = null">
      <p style="font-size:13.5px">{{ confirmState.desc }}</p>
      <div v-if="confirmState.rejection" class="field">
        <label>Kode Alasan <span class="req">*</span></label>
        <select class="input" v-model="confirmState.rejCode">
          <option value="">— pilih —</option>
          <option v-for="r in state.boot?.rejectionCodes || []" :key="r.code" :value="r.code">{{ r.code }} — {{ r.label }}</option>
        </select>
      </div>
      <div class="field">
        <label>Catatan (tercatat di audit)</label>
        <input class="input" v-model="confirmState.note" placeholder="Opsional" />
      </div>
      <template #footer>
        <button class="btn btn-outline" @click="confirmState = null">Batal</button>
        <button class="btn btn-primary" :disabled="confirmState.rejection && !confirmState.rejCode" @click="confirmState.action">Ya, Lanjutkan</button>
      </template>
    </Modal>

    <!-- Modal override blocker -->
    <Modal v-if="overrideTarget" title="Override Blocker (Admin RS)" @close="overrideTarget = null">
      <div class="alert warning"><span>⚠</span><span><b>{{ overrideTarget.ruleId }}</b> — {{ overrideTarget.message }}<br />Override menonaktifkan temuan ini untuk klaim ini saja. Alasan wajib dan tercatat di audit trail.</span></div>
      <div class="field">
        <label>Alasan Override <span class="req">*</span></label>
        <textarea class="input" v-model="overrideReason" placeholder="Contoh: konfirmasi manual dengan penjamin, dokumen menyusul…" />
      </div>
      <template #footer>
        <button class="btn btn-outline" @click="overrideTarget = null">Batal</button>
        <button class="btn btn-primary" :disabled="!overrideReason.trim()" @click="doOverride">Override</button>
      </template>
    </Modal>
  </Teleport>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api'
import { state, hasPerm, toast } from '../store'
import Modal from './Modal.vue'
import Kpi from './Kpi.vue'
import Empty from './Empty.vue'
import ScoreRing from './ScoreRing.vue'
import { formatRp, formatDate, formatDateTime, CLAIM_LABELS, CLAIM_TONE, COMPONENT_LABELS, TARIFF_COMPONENT_KEYS, SPECIAL_CMG_LABELS, SPECIAL_CMG_KEYS, sumTariff, jknTotal, RESUME_CATEGORIES } from '../format'

const props = defineProps({ claimId: String })
const emit = defineEmits(['close', 'changed'])

const d = ref(null)
const tab = ref('ringkasan')
const resolving = ref(false)
const confirmState = ref(null) // {title, desc, danger, action}
const overrideTarget = ref(null)
const overrideReason = ref('')

const hosp = computed(() => d.value ? sumTariff(d.value.claim.hospitalTariff) : 0)
const jkn = computed(() => d.value ? jknTotal(d.value.claim.inacbgBaseTariff, d.value.claim.specialCmg) : 0)
const gap = computed(() => hosp.value - jkn.value)
const paidSum = computed(() => d.value ? d.value.payments.reduce((a, p) => a + p.amount, 0) : 0)
const resumeCount = computed(() => d.value?.resume.length || 0)
const resumeReviewed = computed(() => d.value ? d.value.resume.filter(r => r.reviewed).length : 0)
const allReviewed = computed(() => resumeCount.value > 0 && resumeReviewed.value === resumeCount.value)
const canEdit = computed(() => hasPerm('EDIT') && d.value && ['draft', 'ready_to_submit', 'disputed'].includes(d.value.claim.status))
const canPay = computed(() => hasPerm('CREATE') && d.value && ['approved', 'partially_approved', 'partially_paid'].includes(d.value.claim.status))

const TRANS_LABELS = {
  ready_to_submit: '✓ Siapkan Pengajuan', submitted: '📤 Ajukan', approved: '✅ Setujui',
  partially_approved: '☑ Setujui Sebagian', rejected: '⛔ Tandai Ditolak', disputed: '⚔ Disanggah',
  partially_paid: '💳 Bayar Sebagian', paid: '💰 Tandai Lunas', draft: '✎ Kembalikan ke Draft', written_off: '🗑 Hapus Buku',
}
const allowedTrans = {
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
const availableTrans = computed(() => d.value ? allowedTrans[d.value.claim.status] || [] : [])
function transLabel(t) { return TRANS_LABELS[t] || t }
function transBtnClass(t) {
  return t === 'written_off' || t === 'rejected' ? 'btn-danger' : t === 'submitted' ? 'btn-teal' : 'btn-outline'
}

const deadline = computed(() => {
  if (!d.value?.claim.admissionDate) return { deadline: '', daysLeft: 0, zone: 'aman' }
  const dl = new Date(d.value.claim.admissionDate + 'T00:00:00')
  dl.setMonth(dl.getMonth() + 6)
  const iso = dl.toISOString().slice(0, 10)
  const days = Math.round((dl.getTime() - Date.now()) / 86400000)
  const zone = days < 0 ? 'lewat' : days <= 7 ? 'kritis' : days <= 30 ? 'waspada' : 'aman'
  return { deadline: iso, daysLeft: days, zone }
})
const zoneTone = computed(() => ['lewat', 'kritis'].includes(deadline.value.zone) ? 'danger' : deadline.value.zone === 'waspada' ? 'warning' : 'success')
const zoneLabel = computed(() => ({ aman: 'Aman', waspada: 'Waspada', kritis: 'Kritis', lewat: 'Lewat' })[deadline.value.zone])

function mask(s) { return s ? s.slice(0, 4) + '••••' + s.slice(-3) : '—' }
function catLabel(k) { return RESUME_CATEGORIES.find(c => c.key === k)?.label || k }

async function load() {
  d.value = await api('claims/' + props.claimId)
}
onMounted(load)

async function resolveTariff() {
  resolving.value = true
  try {
    const res = await api('claims/' + props.claimId + '/resolve', { method: 'POST' })
    toast('Tarif diresolve: ' + res.grouper.code + ' = ' + formatRp(res.grouper.baseTariff), 'success')
    await load()
    emit('changed')
  } catch (e) { toast(e.message, 'error') } finally { resolving.value = false }
}

function askTransition(t) {
  const c = d.value.claim
  if (t === 'submitted') {
    confirmState.value = { title: 'Ajukan klaim?', desc: 'Klaim ' + c.id + ' akan berstatus Diajukan. Tindakan ini tercatat di audit trail. Memerlukan peran Reviewer/Admin (maker-checker).', action: () => doTransition(t) }
    return
  }
  if (t === 'rejected') {
    confirmState.value = { title: 'Tandai ditolak?', desc: 'Pilih kode alasan penolakan terstruktur untuk analitik.', rejection: true, action: () => doTransition(t) }
    return
  }
  if (t === 'written_off') {
    confirmState.value = { title: 'Hapus buku?', desc: 'Status berubah menjadi Dihapus Buku dan tidak dapat dikembalikan.', danger: true, action: () => doTransition(t) }
    return
  }
  confirmState.value = { title: 'Ubah status → ' + transLabel(t), desc: 'Transisi divalidasi state machine. Setiap perubahan tercatat di audit trail.', action: () => doTransition(t) }
}

async function doTransition(t) {
  const cs = confirmState.value
  try {
    await api('claims/' + props.claimId + '/transition', {
      method: 'POST',
      body: { to: t, note: cs?.note || '', rejectionCode: cs?.rejCode || '' },
    })
    toast('Status klaim diperbarui → ' + (CLAIM_LABELS[t] || t), 'success')
    confirmState.value = null
    await load()
    emit('changed')
  } catch (e) { toast(e.message, 'error') }
}

function askOverride(f) {
  overrideTarget.value = f
  overrideReason.value = ''
}

async function doOverride() {
  try {
    await api('scrub-rules/override', { method: 'POST', body: { claimId: props.claimId, ruleId: overrideTarget.value.ruleId, reason: overrideReason.value } })
    toast('Override ' + overrideTarget.value.ruleId + ' dicatat di audit trail.', 'success')
    overrideTarget.value = null
    await load()
  } catch (e) { toast(e.message, 'error') }
}

async function toggleReview(r, ev) {
  try {
    await api('resume/' + r.id, { method: 'PUT', body: { ...r, reviewed: ev.target.checked } })
    r.reviewed = ev.target.checked
  } catch (e) { toast(e.message, 'error'); ev.target.checked = !ev.target.checked }
}

function onCsv(ev) {
  const file = ev.target.files[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = async () => {
    const text = String(reader.result || '').trim()
    const lines = text.split(/\r?\n/).filter(Boolean)
    const items = []
    for (const ln of lines) {
      const cols = ln.split(';').map(s => s.trim())
      if (!cols.length) continue
      if (/^sep$/i.test(cols[0])) continue // header
      items.push({
        sep: cols[0] || '', kategori: cols[1] || 'REKAM_MEDIS', namaItem: cols[2] || '',
        subKategori: cols[3] || '', tanggal: cols[4] || '', hasilCatatan: cols[5] || '',
        jumlah: parseInt(cols[6] || '1', 10) || 1, satuan: cols[7] || 'kali',
        hargaSatuan: parseInt((cols[8] || '0').replace(/[^\d]/g, ''), 10) || 0, subtotal: 0, reviewed: false,
      })
    }
    try {
      const res = await api('resume/bulk', { method: 'POST', body: { claimId: props.claimId, items } })
      toast(res.added + ' item resume diimpor (' + res.skipped + ' dilewati — SEP tidak cocok).', 'success')
      await load()
    } catch (e) { toast(e.message, 'error') }
  }
  reader.readAsText(file)
  ev.target.value = ''
}

function askDelete() {
  confirmState.value = {
    title: 'Hapus klaim permanen?',
    desc: 'Klaim ' + d.value.claim.id + ' dan seluruh item resume medisnya akan dihapus permanen.',
    danger: true,
    action: async () => {
      try {
        await api('claims/' + props.claimId, { method: 'DELETE' })
        toast('Klaim dihapus.', 'success')
        confirmState.value = null
        emit('changed')
        emit('close')
      } catch (e) { toast(e.message, 'error') }
    },
  }
}

defineExpose({ load })
</script>
