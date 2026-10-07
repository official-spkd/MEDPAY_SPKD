<template>
  <main class="page">
    <PageHeader icon="claims" title="Daftar Klaim COB" sub="Klaim penjamin kedua untuk selisih tarif (Tagihan RS − Tarif JKN). Cari berdasarkan SEP / nama pasien / MRN.">
      <template #actions>
        <router-link to="/claims/new" class="btn btn-primary" v-if="hasPerm('CREATE')"><Icon name="plus" :size="14" /> Klaim Baru</router-link>
      </template>
    </PageHeader>

    <div class="filterbar">
      <input class="input search-input" v-model="q" placeholder="Cari SEP / nama pasien / MRN…" @input="onSearch" />
      <select class="input" v-model="status" @change="load(1)">
        <option value="">Semua Status</option>
        <option v-for="(lbl, k) in CLAIM_LABELS" :key="k" :value="k">{{ lbl }}</option>
      </select>
      <select class="input" v-model="careType" @change="load(1)">
        <option value="">Semua Jenis Rawat</option>
        <option value="RAWAT_INAP">Rawat Inap</option>
        <option value="RAWAT_JALAN">Rawat Jalan</option>
      </select>
      <span style="color:var(--text-muted);font-size:12.5px">{{ total }} klaim</span>
    </div>

    <div class="tbl-wrap">
      <table class="tbl">
        <thead>
          <tr>
            <th>Klaim / SEP</th><th>Pasien</th><th>Episode</th><th>INA-CBG</th>
            <th class="num">Gap</th><th>Penjamin</th><th>Kesiapan</th><th>Status</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in rows" :key="c.id" @click="openDetail(c.id)">
            <td>
              <div class="mono" style="font-weight:700">{{ c.id }}</div>
              <div class="mono" style="color:var(--text-muted);font-size:11px">{{ c.sepNo }}</div>
            </td>
            <td>
              <div style="font-weight:600">{{ c.patientName }}</div>
              <div style="color:var(--text-muted);font-size:11.5px">{{ c.mrn }} · {{ c.sex === 'L' ? 'Laki-laki' : 'Perempuan' }}</div>
            </td>
            <td style="font-size:12px">{{ careLabel(c.careType) }}<br /><span style="color:var(--text-muted)">Kelas {{ c.careClass }} · LOS {{ c.los }} hr</span></td>
            <td><span class="code-chip" v-if="c.inacbgCode">{{ c.inacbgCode }}</span><span v-else style="color:var(--warning);font-size:12px">⚠ belum resolve</span></td>
            <td class="num" :style="{ fontWeight: 700, color: gapTone(c) }">{{ formatRpShort(gapOf(c)) }}</td>
            <td style="font-size:12px">{{ insurerName(c.insurerId) }}</td>
            <td><ScoreRing :score="c.readinessScore" /></td>
            <td><span class="badge" :class="CLAIM_TONE[c.status]">{{ CLAIM_LABELS[c.status] }}</span></td>
          </tr>
          <tr v-if="!rows.length && !loading"><td colspan="8"><Empty icon="🔍" title="Tidak ada klaim" desc="Coba ubah filter/kata kunci, atau buat klaim baru." /></td></tr>
        </tbody>
      </table>
    </div>
    <div style="display:flex;gap:10px;align-items:center;justify-content:flex-end;margin-top:12px" v-if="totalPages > 1">
      <button class="btn btn-outline btn-sm" :disabled="page <= 1" @click="load(page - 1)">‹ Sebelumnya</button>
      <span style="font-size:12.5px;color:var(--text-muted)">Hal. {{ page }} / {{ totalPages }}</span>
      <button class="btn btn-outline btn-sm" :disabled="page >= totalPages" @click="load(page + 1)">Berikutnya ›</button>
    </div>

    <!-- Drawer detail -->
    <ClaimDrawer v-if="detailId" :claim-id="detailId" @close="detailId = null" @changed="load(page)" />
  </main>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api'
import { state, hasPerm, loadBoot } from '../store'
import Empty from '../components/Empty.vue'
import PageHeader from '../components/PageHeader.vue'
import Icon from '../components/Icon.vue'
import ScoreRing from '../components/ScoreRing.vue'
import ClaimDrawer from '../components/ClaimDrawer.vue'
import { formatRpShort, sumTariff, jknTotal, CLAIM_LABELS, CLAIM_TONE, debounce } from '../format'

const route = useRoute()
const rows = ref([])
const total = ref(0)
const page = ref(1)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / 15)))
const q = ref('')
const status = ref('')
const careType = ref('')
const loading = ref(false)
const detailId = ref(null)

const onSearch = debounce(() => load(1), 300)

async function load(p = 1) {
  loading.value = true
  page.value = p
  const params = new URLSearchParams()
  if (q.value) params.set('q', q.value)
  if (status.value) params.set('status', status.value)
  if (careType.value) params.set('careType', careType.value)
  params.set('page', String(p))
  params.set('pageSize', '15')
  try {
    const res = await api('claims?' + params.toString())
    rows.value = res.rows
    total.value = res.total
  } catch (e) { state.toasts.push({ id: Date.now(), msg: e.message, type: 'error' }) }
  loading.value = false
}

function openDetail(id) { detailId.value = id }

function careLabel(t) { return t === 'RAWAT_INAP' ? 'Rawat Inap' : 'Rawat Jalan' }
function insurerName(id) { return state.boot?.insurers.find(i => i.id === id)?.name || '—' }
function gapOf(c) { return sumTariff(c.hospitalTariff) - jknTotal(c.inacbgBaseTariff, c.specialCmg) }
function gapTone(c) { const g = gapOf(c); return g < 0 ? 'var(--danger)' : g === 0 ? 'var(--text-muted)' : 'var(--success)' }

onMounted(async () => {
  await loadBoot()
  if (route.query.q) { q.value = String(route.query.q); status.value = String(route.query.status || '') }
  load(1)
})
</script>
