<template>
  <main class="page">
    <PageHeader icon="shieldCheck" title="Validasi Klaim (Claim Scrubber)" sub="Rule engine dengan 14 aturan lokal (ICD-10, ICD-9-CM, INA-CBG, e-Klaim, BPJS, POJK). Klaim dengan BLOCKER terbuka tidak dapat masuk batch atau diajukan." />

    <div class="tabs">
      <button class="tab" :class="{ active: tab === 'hasil' }" @click="tab = 'hasil'">Hasil Validasi</button>
      <button class="tab" :class="{ active: tab === 'library' }" @click="tab = 'library'">Library Aturan</button>
      <button class="tab" :class="{ active: tab === 'causes' }" @click="tab = 'causes'">Penyebab Teratas</button>
    </div>

    <!-- Hasil per klaim -->
    <div v-if="tab === 'hasil'">
      <div class="filterbar">
        <select class="input" v-model="claimId" @change="scrub" style="min-width:340px">
          <option value="">— pilih klaim untuk divalidasi —</option>
          <option v-for="c in claims" :key="c.id" :value="c.id">{{ c.id }} — {{ c.patientName }} ({{ CLAIM_LABELS[c.status] }})</option>
        </select>
        <button class="btn btn-teal btn-sm" :disabled="!claimId || busy" @click="scrub"><span v-if="busy" class="spin" /> Jalankan Validasi</button>
      </div>
      <div v-if="result">
        <div class="kpi-grid">
          <Kpi icon="target" label="Claim Readiness Score" :value="result.score + ' / 100'" :tone="result.score >= 80 ? 'success' : result.score >= 50 ? 'warning' : 'danger'"
            source="Skor = 100 − 25×BLOCKER − 8×WARNING (temuan yang tidak di-override)." />
          <Kpi icon="x" label="Blocker Terbuka" :value="String(openBlockers)" tone="danger" :val-class="openBlockers ? 'danger' : ''"
            source="BLOCKER terbuka memblokir masuk batch & transisi ke Siap Diajukan." />
          <Kpi icon="alert" label="Peringatan" :value="String(openWarnings)" tone="warning"
            source="WARNING tidak memblokir, tetapi menandai risiko penolakan." />
        </div>
        <div v-if="!result.findings.length"><Empty icon="✅" title="Klaim bersih" desc="Tidak ada temuan — klaim siap diproses ke batch." /></div>
        <div v-for="f in result.findings" :key="f.ruleId" class="card card-pad" style="margin-bottom:10px">
          <div style="display:flex;gap:10px;align-items:flex-start;flex-wrap:wrap">
            <span class="badge" :class="f.severity === 'BLOCKER' ? 'danger' : 'warning'">{{ f.severity }}</span>
            <div style="flex:1;min-width:260px">
              <b>{{ f.ruleId }}</b> — {{ f.message }}
              <div style="font-size:12.5px;margin-top:5px;color:var(--text-secondary)"><b>Saran perbaikan:</b> {{ f.fixSuggestion }}</div>
              <div style="font-size:11px;color:var(--text-muted);margin-top:3px">Sumber/regulasi: {{ f.sourceRef }}</div>
            </div>
            <div>
              <span v-if="f.overridden" class="badge warning">Di-override Admin RS</span>
              <button v-else-if="hasPerm('OVERRIDE') && f.severity === 'BLOCKER'" class="btn btn-outline btn-xs" @click="overrideRule = f.ruleId">Override</button>
            </div>
          </div>
        </div>
      </div>
      <Empty v-else icon="🔍" title="Pilih klaim" desc="Pilih klaim dari dropdown lalu jalankan validasi untuk melihat temuan, saran perbaikan, dan skor kesiapan." />
    </div>

    <!-- Library aturan -->
    <div v-if="tab === 'library'">
      <div class="tbl-wrap">
        <table class="tbl">
          <thead><tr><th>ID</th><th>Aturan</th><th>Kategori</th><th>Severity</th><th>Aktif</th><th>Sumber</th></tr></thead>
          <tbody>
            <tr v-for="r in rules" :key="r.id">
              <td class="mono"><b>{{ r.id }}</b></td>
              <td>
                <div style="font-weight:600">{{ r.name }}</div>
                <div style="font-size:11.5px;color:var(--text-muted)">{{ r.message }}</div>
              </td>
              <td><span class="badge neutral">{{ r.category }}</span></td>
              <td><span class="badge" :class="r.severity === 'BLOCKER' ? 'danger' : r.severity === 'WARNING' ? 'warning' : 'info'">{{ r.severity }}</span></td>
              <td>
                <input type="checkbox" :checked="r.enabled" :disabled="!hasPerm('CONFIGURE')" @change="toggleRule(r, $event)" :aria-label="'Aktifkan ' + r.id" />
              </td>
              <td style="font-size:11.5px;max-width:280px">{{ r.sourceRef }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="note-strip">Aturan dapat diaktifkan/nonaktifkan per RS (izin CONFIGURE). Setiap perubahan tercatat di audit trail.</div>
    </div>

    <!-- Penyebab teratas -->
    <div v-if="tab === 'causes'">
      <div class="card card-pad">
        <div class="card-title">Penyebab Teratas Klaim Bermasalah</div>
        <div class="card-sub">Agregasi temuan scrubber aktif seluruh klaim dalam cakupan RS Anda.</div>
        <div v-for="c in causes" :key="c.ruleId" style="margin-bottom:12px">
          <div style="display:flex;gap:10px;align-items:center;font-size:13px">
            <span class="badge" :class="c.severity === 'BLOCKER' ? 'danger' : 'warning'">{{ c.ruleId }}</span>
            <span style="flex:1">{{ c.name }}</span>
            <b style="font-family:var(--font-head)">{{ c.count }}</b>
          </div>
          <div class="progress-track" style="margin-top:5px">
            <div class="progress-fill" :style="{ width: (c.count / maxCause * 100) + '%' }" />
          </div>
        </div>
        <Empty v-if="!causes.length" icon="✅" title="Tidak ada temuan" desc="Seluruh klaim lolos scrubber." />
      </div>
    </div>

    <Modal v-if="overrideRule" title="Override Blocker" @close="overrideRule = ''">
      <div class="alert warning"><span>⚠</span><span>Override <b>{{ overrideRule }}</b> untuk klaim terpilih. Alasan wajib, tercatat di audit trail.</span></div>
      <div class="field"><label>Alasan <span class="req">*</span></label><textarea class="input" v-model="overrideReason" /></div>
      <template #footer>
        <button class="btn btn-outline" @click="overrideRule = ''">Batal</button>
        <button class="btn btn-primary" :disabled="!overrideReason.trim() || !claimId" @click="doOverride">Override</button>
      </template>
    </Modal>
  </main>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api'
import { state, hasPerm, toast, loadBoot } from '../store'
import Kpi from '../components/Kpi.vue'
import PageHeader from '../components/PageHeader.vue'
import Empty from '../components/Empty.vue'
import Modal from '../components/Modal.vue'
import { CLAIM_LABELS } from '../format'

const tab = ref('hasil')
const claims = ref([])
const claimId = ref('')
const result = ref(null)
const busy = ref(false)
const causes = ref([])
const overrideRule = ref('')
const overrideReason = ref('')

const rules = computed(() => state.boot?.scrubRules || [])
const openBlockers = computed(() => result.value ? result.value.findings.filter(f => f.severity === 'BLOCKER' && !f.overridden).length : 0)
const openWarnings = computed(() => result.value ? result.value.findings.filter(f => f.severity === 'WARNING' && !f.overridden).length : 0)
const maxCause = computed(() => causes.value.length ? Math.max(...causes.value.map(c => c.count), 1) : 1)

async function loadClaims() {
  const res = await api('claims?pageSize=200')
  claims.value = res.rows
}

async function scrub() {
  if (!claimId.value) return
  busy.value = true
  try { result.value = await api('claims/' + claimId.value + '/scrub', { method: 'POST' }) }
  catch (e) { toast(e.message, 'error') }
  finally { busy.value = false }
}

async function toggleRule(r, ev) {
  try {
    await api('scrub-rules/' + r.id, { method: 'PUT', body: { enabled: ev.target.checked } })
    r.enabled = ev.target.checked
    toast('Aturan ' + r.id + (ev.target.checked ? ' diaktifkan.' : ' dinonaktifkan.'), 'success')
  } catch (e) { toast(e.message, 'error'); ev.target.checked = !ev.target.checked }
}

async function doOverride() {
  try {
    result.value = await api('scrub-rules/override', { method: 'POST', body: { claimId: claimId.value, ruleId: overrideRule.value, reason: overrideReason.value } })
    toast('Override dicatat di audit trail.', 'success')
    overrideRule.value = ''
    overrideReason.value = ''
  } catch (e) { toast(e.message, 'error') }
}

onMounted(async () => {
  await loadBoot()
  await loadClaims()
  causes.value = await api('scrubber/top-causes')
})
</script>
