<template>
  <main class="page">
    <PageHeader icon="code" title="Engine API Monitoring" sub="Pemantauan klaim masuk via Engine API partner (tiga lapis kontrol: IP whitelist, log penuh, user key terikat RS). Data demo.">
      <template #actions>
        <button class="btn btn-outline btn-sm" @click="load" :disabled="busy"><span v-if="busy" class="spin" /> Muat Ulang</button>
        <button class="btn btn-outline btn-sm" @click="exportCsv">Unduh CSV</button>
      </template>
    </PageHeader>

    <div v-if="loadError" class="alert danger"><span>⚠</span><span>Gagal memuat data pemantauan: {{ loadError }} — <a href="#" @click.prevent="load" style="text-decoration:underline">Coba Lagi</a></span></div>

    <template v-if="s && !loadError">
      <div class="kpi-grid">
        <Kpi icon="code" label="Klaim Masuk via API" :value="String(s.intakes.length)" tone="navy" :source="'dengan ' + s.unresolved + ' berstatus Unresolved (error: ' + topError + ')'" />
        <Kpi icon="banknote" label="Total Tagihan RS (API)" :value="formatRp(s.totalHospitalBill)" tone="navy" source="Σ komponen tarif klaim intake API." />
        <Kpi icon="calc" label="Total Tarif INA-CBG (API)" :value="formatRp(s.totalJkn)" tone="" source="Σ hasil grouper e-Klaim adapter." />
        <Kpi icon="zap" label="API Calls" :value="String(s.apiCalls)" tone="success" :val-class="s.successPct >= 90 ? 'success' : s.successPct >= 70 ? 'warning' : 'danger'"
          :source="'sukses ' + s.successPct.toFixed(1) + '% · rerata latensi ' + s.avgLatencyMs + ' ms · partner aktif: ' + s.partnerCount" />
      </div>

      <div class="filterbar">
        <select class="input" v-model="fPartner" @change="applyFilter">
          <option value="">Semua Partner</option>
          <option v-for="p in state.boot?.partners || []" :key="p.id" :value="p.id">{{ p.company }}</option>
        </select>
        <select class="input" v-model="fHosp" @change="applyFilter">
          <option value="">Semua RS</option>
          <option v-for="h in state.boot?.hospitals || []" :key="h.id" :value="h.id">{{ h.name }}</option>
        </select>
        <select class="input" v-model="fStatus">
          <option value="">Semua Status</option>
          <option value="RESOLVED">Resolved</option>
          <option value="UNRESOLVED">Unresolved</option>
        </select>
      </div>

      <div class="tbl-wrap">
        <table class="tbl">
          <thead><tr><th>Intake</th><th>Partner / RS</th><th>Pasien / SEP</th><th>INA-CBG</th><th class="num">Gap</th><th>Status</th><th class="num">Latensi</th><th>Waktu</th></tr></thead>
          <tbody>
            <tr v-for="i in filtered" :key="i.id" @click="sel = i">
              <td class="mono">{{ i.id }}</td>
              <td style="font-size:12px">{{ partnerName(i.partnerId) }}<br /><span style="color:var(--text-muted)">{{ hospName(i.hospitalId) }}</span></td>
              <td>{{ i.patientName }}<div class="mono" style="font-size:11px;color:var(--text-muted)">{{ i.sepNo }}</div></td>
              <td><span class="code-chip" v-if="i.inacbgCode">{{ i.inacbgCode }}</span><span v-else class="badge danger">{{ i.errorCode }}</span></td>
              <td class="num" style="font-weight:700" :style="{ color: i.gap > 0 ? 'var(--success)' : 'var(--text-muted)' }">{{ formatRpShort(i.gap) }}</td>
              <td><span class="badge" :class="i.status === 'RESOLVED' ? 'success' : 'danger'">{{ i.status }}</span></td>
              <td class="num">{{ i.latencyMs }} ms</td>
              <td style="font-size:11.5px;color:var(--text-muted)">{{ formatDateTime(i.at) }}</td>
            </tr>
            <tr v-if="!filtered.length"><td colspan="8"><Empty icon="📭" title="Tidak ada intake" desc="Tidak ada klaim API dengan filter ini." /></td></tr>
          </tbody>
        </table>
      </div>
    </template>

    <Modal v-if="sel" title="Detail Intake Klaim API" wide @close="sel = null">
      <div class="kv">
        <span class="k">Partner</span><span class="v">{{ partnerName(sel.partnerId) }}</span>
        <span class="k">RS</span><span class="v">{{ hospName(sel.hospitalId) }}</span>
        <span class="k">SEP / Pasien</span><span class="v mono">{{ sel.sepNo }} — {{ sel.patientName }}</span>
        <span class="k">Kode INA-CBG</span><span class="v mono">{{ sel.inacbgCode || sel.errorCode + ' (UNRESOLVED)' }}</span>
        <span class="k">Tagihan RS / JKN</span><span class="v">{{ formatRp(sel.hospitalTotal) }} / {{ formatRp(sel.jknTotal) }}</span>
        <span class="k">Gap</span><span class="v">{{ formatRp(sel.gap) }}</span>
        <span class="k">Latensi</span><span class="v">{{ sel.latencyMs }} ms · {{ formatDateTime(sel.at) }}</span>
      </div>
      <div class="card-title" style="margin-top:14px;font-size:13px">Split KAPJ (parameter default)</div>
      <div class="gap-box pos"><span>BPJS / Penjamin / Pasien</span><span>{{ formatRp(sel.bpjsShare) }} / {{ formatRp(sel.insurerShare) }} / {{ formatRp(sel.patientShare) }}</span></div>
      <div class="chips-row" style="margin-top:12px" v-if="sel.status === 'UNRESOLVED'">
        <span class="badge danger">Error: {{ sel.errorCode }}</span>
        <button class="btn btn-outline btn-sm" @click="toast('Buka di Claim Form — demo: buat klaim manual dari data intake.', 'info')">Buka di Claim Form</button>
      </div>
    </Modal>
  </main>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api'
import { state, toast, loadBoot } from '../store'
import Kpi from '../components/Kpi.vue'
import PageHeader from '../components/PageHeader.vue'
import Empty from '../components/Empty.vue'
import Modal from '../components/Modal.vue'
import { formatRp, formatRpShort, formatDateTime, downloadBlobCsv } from '../format'

const s = ref(null)
const busy = ref(false)
const loadError = ref('')
const fPartner = ref('')
const fHosp = ref('')
const fStatus = ref('')
const sel = ref(null)

const filtered = computed(() => (s.value?.intakes || []).filter(i =>
  (!fPartner.value || i.partnerId === fPartner.value) &&
  (!fHosp.value || i.hospitalId === fHosp.value) &&
  (!fStatus.value || i.status === fStatus.value)))
const topError = computed(() => {
  const errs = (s.value?.intakes || []).filter(i => i.status === 'UNRESOLVED').map(i => i.errorCode)
  return errs.length ? errs[0] : '—'
})

function partnerName(id) { return state.boot?.partners.find(p => p.id === id)?.company || id }
function hospName(id) { return state.boot?.hospitals.find(h => h.id === id)?.name || id }

async function load() {
  busy.value = true
  loadError.value = ''
  try { s.value = await api('engine-api/summary') }
  catch (e) { loadError.value = e.message }
  finally { busy.value = false }
}

function applyFilter() { /* computed filter */ }

function exportCsv() {
  const rows = [['ID', 'Partner', 'RS', 'SEP', 'Pasien', 'INA-CBG', 'Tagihan RS', 'JKN', 'Gap', 'Status', 'Error', 'Latensi']]
  for (const i of filtered.value) rows.push([i.id, partnerName(i.partnerId), hospName(i.hospitalId), i.sepNo, i.patientName, i.inacbgCode, i.hospitalTotal, i.jknTotal, i.gap, i.status, i.errorCode, i.latencyMs])
  downloadBlobCsv('engine-api-intake.csv', rows)
}

onMounted(async () => { await loadBoot(); await load() })
</script>
