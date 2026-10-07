<template>
  <main class="page">
    <PageHeader icon="chart" title="Analitik" sub="Tren bulanan klaim & gap, perbandingan rawat inap vs rawat jalan, segmen eligibilitas, dan window deadline 6 bulan." />

    <div class="grid-2" style="margin-bottom:16px">
      <div class="card card-pad">
        <div class="card-title">Jumlah Klaim per Bulan (12 bulan)</div>
        <div class="card-sub">Berdasarkan tanggal pembuatan klaim. Sumber: modul Klaim.</div>
        <BarChart :data="claimBars" color="#0e5c3a" :formatter="formatNumber" aria-label="klaim per bulan" />
      </div>
      <div class="card card-pad">
        <div class="card-title">Nilai Gap per Bulan</div>
        <div class="card-sub">Σ selisih tarif klaim yang dibuat pada bulan tersebut.</div>
        <LineChart :data="gapLine" :formatter="formatRpShort" aria-label="gap per bulan" />
      </div>
    </div>

    <div class="grid-2" style="margin-bottom:16px">
      <div class="card card-pad">
        <div class="card-title">Rawat Inap vs Rawat Jalan</div>
        <div class="card-sub">Perbandingan volume, gap, total Top Up, dan tagihan di bawah tarif INA-CBG.</div>
        <table class="tbl" style="min-width:420px">
          <thead><tr><th>Metrik</th><th class="num">Rawat Inap</th><th class="num">Rawat Jalan</th></tr></thead>
          <tbody>
            <tr><td>Jumlah klaim</td><td class="num">{{ a.inpatient?.claims ?? 0 }}</td><td class="num">{{ a.outpatient?.claims ?? 0 }}</td></tr>
            <tr><td>Total gap</td><td class="num">{{ formatRp(a.inpatient?.gap) }}</td><td class="num">{{ formatRp(a.outpatient?.gap) }}</td></tr>
            <tr><td>Total Top Up (Special CMG)</td><td class="num">{{ formatRp(a.inpatient?.topUp) }}</td><td class="num">{{ formatRp(a.outpatient?.topUp) }}</td></tr>
            <tr><td>Grup INA-CBG unik</td><td class="num">{{ a.inpatient?.uniqueGroups ?? 0 }}</td><td class="num">{{ a.outpatient?.uniqueGroups ?? 0 }}</td></tr>
            <tr><td>Tagihan &lt; tarif INA-CBG</td><td class="num"><span :style="{ color: a.inpatient?.belowInacbg ? 'var(--warning)' : 'inherit', fontWeight: 700 }">{{ a.inpatient?.belowInacbg ?? 0 }}</span></td><td class="num"><span :style="{ color: a.outpatient?.belowInacbg ? 'var(--warning)' : 'inherit', fontWeight: 700 }">{{ a.outpatient?.belowInacbg ?? 0 }}</span></td></tr>
          </tbody>
        </table>
      </div>
      <div class="card card-pad">
        <div class="card-title">Segmen Eligibilitas</div>
        <div class="card-sub">Distribusi kelas Non-PBI (eligible) vs PBI (tidak eligible COB).</div>
        <Donut :data="segmentData" :formatter="formatNumber" aria-label="segmen eligibilitas" center="Non-PBI" centerSub="eligible COB" />
      </div>
    </div>

    <div class="card card-pad">
      <div class="card-title">Klaim dalam Window Deadline 6 Bulan <span class="verify-tag">[VERIFIKASI REGULASI]</span></div>
      <div class="card-sub">Aktif dengan sisa ≤ 30 hari. <span class="badge danger">merah</span> = sisa ≤ 7 hari (kritis).</div>
      <div class="tbl-wrap" v-if="(a.deadlineWindow || []).length">
        <table class="tbl" style="min-width:520px">
          <thead><tr><th>Klaim</th><th>Pasien</th><th>Deadline</th><th>Sisa</th><th></th></tr></thead>
          <tbody>
            <tr v-for="c in a.deadlineWindow" :key="c.id" @click="$router.push('/claims?q=' + c.id)">
              <td class="mono">{{ c.id }}</td>
              <td>{{ c.patientName }}</td>
              <td>{{ formatDate(c.deadline) }}</td>
              <td><span class="badge" :class="c.critical ? 'danger' : 'warning'">{{ c.daysLeft >= 0 ? c.daysLeft + ' hari' : 'lewat' }}</span></td>
              <td style="font-size:12px;color:var(--text-secondary)">{{ c.critical ? 'Segera ajukan atau minta override Admin RS.' : 'Masih dalam window aman-waspada.' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <Empty v-else icon="🗓" title="Tidak ada klaim di window deadline" desc="Tidak ada klaim aktif dengan sisa waktu ≤ 30 hari." />
    </div>
  </main>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api'
import { toast } from '../store'
import BarChart from '../components/BarChart.vue'
import PageHeader from '../components/PageHeader.vue'
import LineChart from '../components/LineChart.vue'
import Donut from '../components/Donut.vue'
import Empty from '../components/Empty.vue'
import { formatRp, formatRpShort, formatDate, formatNumber } from '../format'

const a = ref({})

const claimBars = computed(() => (a.value.trend || []).map(t => ({ label: t.month, value: t.claims })))
const gapLine = computed(() => (a.value.trend || []).map(t => ({ label: t.month, value: t.gap })))
const segmentData = computed(() => Object.entries(a.value.segments || {}).map(([k, v]) => ({ label: k, value: v })))

onMounted(async () => {
  try { a.value = await api('analytics') } catch (e) { toast(e.message, 'error') }
})
</script>
