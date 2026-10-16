<template>
  <main class="page">
    <PageHeader icon="clock" title="Piutang (AR Aging)" sub="Umur piutang per bucket, DSO, dan performa penjamin. Basis: tanggal pengajuan (dapat diganti tanggal pelayanan).">
      <template #actions>
        <button class="btn btn-outline btn-sm" @click="exportCsv">Ekspor CSV</button>
      </template>
    </PageHeader>

    <div class="filterbar">
      <span style="font-size:12.5px;color:var(--text-secondary);font-weight:700">Basis umur:</span>
      <button class="btn btn-sm" :class="basis === 'submission' ? 'btn-teal' : 'btn-outline'" @click="basis = 'submission'; load()">Tanggal Pengajuan</button>
      <button class="btn btn-sm" :class="basis === 'service' ? 'btn-teal' : 'btn-outline'" @click="basis = 'service'; load()">Tanggal Pelayanan</button>
    </div>

    <div class="kpi-grid">
      <Kpi icon="clock" label="Total Piutang Tertunggak" :value="formatRp(a.total)" tone="warning" val-class="warning"
        source="Σ (gap − pembayaran) klaim belum lunas & belum ditolak/dihapus buku." />
      <Kpi icon="activity" label="DSO (hari)" :value="a.dso !== null && a.dso !== undefined ? String(a.dso) : '—'" tone="navy"
        source="Rata-rata hari dari pengajuan sampai pembayaran terakhir diterima (klaim lunas)." />
      <Kpi icon="alert" label="% Piutang > 90 hari" :value="pctOver90 + '%'" tone="danger" :val-class="pctOver90 > 20 ? 'danger' : ''"
        source="Porsi piutang bucket > 90 hari dari total — indikator piutang macet." />
      <Kpi icon="claims" label="Klaim Tertunggak" :value="String(a.drilldown?.length || 0)" tone=""
        source="Jumlah klaim dengan outstanding > 0 (drill-down di bawah)." />
    </div>

    <div class="grid-2" style="margin-bottom:16px">
      <div class="card card-pad">
        <div class="card-title">Bucket Umur Piutang</div>
        <div class="card-sub">0–30 · 31–60 · 61–90 · >90 hari (basis: {{ basis === 'submission' ? 'pengajuan' : 'pelayanan' }}).</div>
        <BarChart :data="bucketBars" :formatter="formatRpShort" :short-formatter="formatRpShort" aria-label="bucket piutang" />
      </div>
      <div class="card card-pad">
        <div class="card-title">Performa Penjamin</div>
        <div class="card-sub">Rata-rata hari bayar, tingkat persetujuan & penolakan, dana diterima.</div>
        <div class="tbl-wrap" style="max-height:240px;overflow-y:auto">
          <table class="tbl" style="min-width:430px">
            <thead><tr><th>Penjamin</th><th class="num">Hari</th><th class="num">Setuju</th><th class="num">Tolak</th><th class="num">Diterima</th></tr></thead>
            <tbody>
              <tr v-for="p in a.payerPerf || []" :key="p.insurerId">
                <td style="max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">{{ p.name }}</td>
                <td class="num">{{ p.avgDaysPay ?? '—' }}</td>
                <td class="num"><b :style="{ color: rateColor(p.approvalRate) }">{{ p.approvalRate.toFixed(0) }}%</b></td>
                <td class="num">{{ p.rejectionRate.toFixed(0) }}%</td>
                <td class="num">{{ formatRpShort(p.receivedRp) }}</td>
              </tr>
              <tr v-if="!(a.payerPerf || []).length"><td colspan="5" style="text-align:center;color:var(--text-muted)">Belum ada data</td></tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <div class="card card-pad">
      <div class="card-title">Drill-down Klaim Tertunggak (per Batch)</div>
      <div class="card-sub">Dikelompokkan per batch pengajuan, diurutkan dari tenggat terlama. Klik batch untuk melihat klaim di dalamnya. Klaim yang belum masuk batch dikelompokkan di "Tanpa Batch".</div>
      <div class="tbl-wrap" style="max-height:380px;overflow-y:auto">
        <table class="tbl" style="min-width:640px">
          <thead><tr><th>Batch</th><th>Penjamin</th><th class="num">Klaim</th><th class="num">Tertunggak</th><th class="num">Tenggat</th></tr></thead>
          <tbody>
            <tr v-for="g in batchRows" :key="g.key" :style="g.batchId ? 'cursor:pointer' : ''" @click="g.batchId && $router.push('/batches?open=' + g.batchId)">
              <td>
                <div style="font-weight:600">{{ g.batchId ? g.name : 'Tanpa Batch' }}</div>
                <div class="mono" style="font-size:11px;color:var(--text-muted)">{{ g.batchId ? g.batchNo : 'belum masuk batch' }}</div>
              </td>
              <td style="font-size:12px;max-width:220px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">{{ g.insurers.join(', ') }}</td>
              <td class="num">{{ g.count }}</td>
              <td class="num" style="font-weight:700">{{ formatRp(g.amount) }}</td>
              <td class="num"><span class="badge" :class="g.maxAge > 90 ? 'danger' : g.maxAge > 60 ? 'warning' : 'neutral'">{{ g.maxAge }} hari</span></td>
            </tr>
            <tr v-if="!batchRows.length"><td colspan="5"><Empty icon="✅" title="Tidak ada piutang" desc="Seluruh klaim lunas / tidak ada outstanding." /></td></tr>
          </tbody>
        </table>
      </div>
    </div>
  </main>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api'
import { toast } from '../store'
import Kpi from '../components/Kpi.vue'
import PageHeader from '../components/PageHeader.vue'
import BarChart from '../components/BarChart.vue'
import Empty from '../components/Empty.vue'
import { formatRp, formatRpShort, CLAIM_LABELS, CLAIM_TONE, downloadBlobCsv, docNo } from '../format'

const a = ref({})
const basis = ref('submission')

async function load() {
  try { a.value = await api('aging?basis=' + basis.value) }
  catch (e) { toast(e.message, 'error') }
}

const bucketBars = computed(() => (a.value.buckets || []).map(b => ({ label: b.label, value: b.amount })))
// Drill-down per batch: jumlahkan tertunggak, hitung klaim, dan ambil umur terlama sebagai tenggat.
const batchRows = computed(() => {
  const map = new Map()
  for (const r of a.value.drilldown || []) {
    const key = r.batchId || '_none'
    let g = map.get(key)
    if (!g) { g = { key, batchId: r.batchId || '', batchNo: r.batchNo, name: r.batchName || r.batchNo, count: 0, amount: 0, maxAge: 0, insurers: [] }; map.set(key, g) }
    g.count++
    g.amount += r.amount
    g.maxAge = Math.max(g.maxAge, r.ageDays)
    if (r.insurer && !g.insurers.includes(r.insurer)) g.insurers.push(r.insurer)
  }
  return [...map.values()].sort((x, y) => y.maxAge - x.maxAge)
})
const pctOver90 = computed(() => (a.value.pctOver90 ?? 0).toFixed(1).replace('.', ','))

function rateColor(r) { return r >= 80 ? 'var(--success)' : r >= 60 ? 'var(--warning)' : 'var(--danger)' }

function exportCsv() {
  const rows = [['Batch', 'Nama Batch', 'Penjamin', 'Jumlah Klaim', 'Tertunggak', 'Tenggat (hari)']]
  for (const g of batchRows.value) rows.push([g.batchId ? g.batchNo : 'Tanpa Batch', g.batchId ? g.name : '', g.insurers.join(', '), g.count, g.amount, g.maxAge])
  downloadBlobCsv('piutang-medpay.csv', rows)
}

onMounted(load)
</script>
