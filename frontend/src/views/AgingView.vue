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
      <div class="card-title">Drill-down Klaim Tertunggak</div>
      <div class="card-sub">Diurutkan dari yang paling tua. Setiap angka dapat ditelusuri ke klaim sumbernya.</div>
      <div class="tbl-wrap" style="max-height:380px;overflow-y:auto">
        <table class="tbl" style="min-width:640px">
          <thead><tr><th>Klaim</th><th>Pasien</th><th>Penjamin</th><th>Status</th><th class="num">Tertunggak</th><th class="num">Umur</th></tr></thead>
          <tbody>
            <tr v-for="r in a.drilldown || []" :key="r.id" @click="$router.push('/claims?q=' + r.id)">
              <td class="mono">{{ r.id }}</td>
              <td>{{ r.patientName }}</td>
              <td style="font-size:12px;max-width:220px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">{{ r.insurer }}</td>
              <td><span class="badge" :class="CLAIM_TONE[r.status]">{{ CLAIM_LABELS[r.status] }}</span></td>
              <td class="num" style="font-weight:700">{{ formatRp(r.amount) }}</td>
              <td class="num"><span class="badge" :class="r.ageDays > 90 ? 'danger' : r.ageDays > 60 ? 'warning' : 'neutral'">{{ r.ageDays }} hari</span></td>
            </tr>
            <tr v-if="!(a.drilldown || []).length"><td colspan="6"><Empty icon="✅" title="Tidak ada piutang" desc="Seluruh klaim lunas / tidak ada outstanding." /></td></tr>
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
import { formatRp, formatRpShort, CLAIM_LABELS, CLAIM_TONE, downloadBlobCsv } from '../format'

const a = ref({})
const basis = ref('submission')

async function load() {
  try { a.value = await api('aging?basis=' + basis.value) }
  catch (e) { toast(e.message, 'error') }
}

const bucketBars = computed(() => (a.value.buckets || []).map(b => ({ label: b.label, value: b.amount })))
const pctOver90 = computed(() => (a.value.pctOver90 ?? 0).toFixed(1).replace('.', ','))

function rateColor(r) { return r >= 80 ? 'var(--success)' : r >= 60 ? 'var(--warning)' : 'var(--danger)' }

function exportCsv() {
  const rows = [['Klaim', 'Pasien', 'Penjamin', 'Status', 'Tertunggak', 'Umur (hari)']]
  for (const r of a.value.drilldown || []) rows.push([r.id, r.patientName, r.insurer, CLAIM_LABELS[r.status], r.amount, r.ageDays])
  downloadBlobCsv('piutang-medpay.csv', rows)
}

onMounted(load)
</script>
