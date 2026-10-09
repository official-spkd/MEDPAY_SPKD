<template>
  <main class="page">
    <PageHeader icon="pie" title="BI &amp; Laporan" sub="Laporan manajemen/keuangan: perbandingan penjamin dengan ambang persetujuan hijau ≥ 80%, kuning 60–79%, merah &lt; 60%. Setiap metrik menampilkan sumber datanya.">
      <template #actions>
        <button class="btn btn-outline btn-sm" @click="exportCsv">Ekspor Laporan CSV</button>
      </template>
    </PageHeader>

    <div class="kpi-grid">
      <Kpi icon="claims" label="Klaim Terkirim (submitted+)" :value="String(totalSubmitted)" tone="navy"
        source="Klaim berstatus Diajukan ke atas, termasuk disetujui/dibayar/ditolak." />
      <Kpi icon="wallet" label="Nilai Diterima" :value="formatRp(totalReceived)" tone="success" val-class="success"
        source="Σ pembayaran semua penjamin (modul Pembayaran)." />
      <Kpi icon="trendingUp" label="Tingkat Persetujuan Keseluruhan" :value="overallApproval + '%'" tone=""
        source="approved+partially_approved+paid+partially_paid ÷ submitted, semua penjamin." />
      <Kpi icon="trendingDown" label="Penjamin di Bawah 60%" :value="String(redCount)" tone="danger" :val-class="redCount ? 'danger' : ''"
        source="Penjamin dengan approval rate < 60% — perlu evaluasi kerja sama." />
    </div>

    <div class="card card-pad">
      <div class="card-title">Perbandingan Penjamin Kedua</div>
      <div class="card-sub">Klik baris untuk membuka daftar klaim penjamin tersebut. Formula, periode, dan sumber ditampilkan pada setiap kolom.</div>
      <div class="tbl-wrap">
        <table class="tbl">
          <thead>
            <tr>
              <th>Penjamin</th>
              <th class="num" title="Klaim berstatus Diajukan ke atas">Terkirim</th>
              <th class="num" title="approved + partially_approved + paid + partially_paid">Disetujui</th>
              <th class="num" title="status rejected">Ditolak</th>
              <th class="num" title="disetujui ÷ terkirim">Setuju %</th>
              <th class="num" title="ditolak ÷ terkirim">Tolak %</th>
              <th class="num" title="rata-rata hari dari pengajuan ke pembayaran terakhir">Hari Bayar</th>
              <th class="num" title="Σ pembayaran tercatat">Diterima</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in perf" :key="p.insurerId" @click="$router.push('/claims?q=' + (p.name))">
              <td>
                <div style="font-weight:600">{{ p.name }}</div>
                <div style="font-size:11px;color:var(--text-muted)">{{ insurerType(p.insurerId) }} · kebijakan: {{ insurerCeiling(p.insurerId) }}</div>
              </td>
              <td class="num">{{ p.submitted }}</td>
              <td class="num">{{ p.approved }}</td>
              <td class="num">{{ p.rejected }}</td>
              <td class="num"><span class="badge" :class="rateTone(p.approvalRate)">{{ p.approvalRate.toFixed(0) }}%</span></td>
              <td class="num">{{ p.rejectionRate.toFixed(0) }}%</td>
              <td class="num">{{ p.avgDaysPay ?? '—' }}</td>
              <td class="num" style="font-weight:700">{{ formatRpShort(p.receivedRp) }}</td>
            </tr>
            <tr v-if="!perf.length"><td colspan="8"><Empty icon="📊" title="Belum ada data" desc="Belum ada klaim terkirim dalam periode ini." /></td></tr>
          </tbody>
        </table>
      </div>
      <div class="note-strip">Sumber data: modul Klaim (status) & modul Pembayaran (nilai diterima). Periode: seluruh data dalam cakupan RS Anda. Angka potensi ≠ pendapatan pasti.</div>
    </div>
  </main>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api'
import { state, toast } from '../store'
import Kpi from '../components/Kpi.vue'
import PageHeader from '../components/PageHeader.vue'
import Empty from '../components/Empty.vue'
import { formatRp, formatRpShort, downloadBlobCsv } from '../format'

const perf = ref([])

const totalSubmitted = computed(() => perf.value.reduce((a, p) => a + p.submitted, 0))
const totalReceived = computed(() => perf.value.reduce((a, p) => a + p.receivedRp, 0))
const overallApproval = computed(() => {
  const ap = perf.value.reduce((a, p) => a + p.approved, 0)
  return totalSubmitted.value ? Math.round(ap / totalSubmitted.value * 100) : 0
})
const redCount = computed(() => perf.value.filter(p => p.submitted > 0 && p.approvalRate < 60).length)

function rateTone(r) { return r >= 80 ? 'success' : r >= 60 ? 'warning' : 'danger' }
function insurerType(id) { return state.boot?.insurers.find(i => i.id === id)?.type || '—' }
function insurerCeiling(id) { return state.boot?.insurers.find(i => i.id === id)?.ceilingPolicy || '—' }

function exportCsv() {
  const rows = [['Penjamin', 'Terkirim', 'Disetujui', 'Ditolak', 'Setuju %', 'Tolak %', 'Hari Bayar', 'Diterima (Rp)']]
  for (const p of perf.value) rows.push([p.name, p.submitted, p.approved, p.rejected, p.approvalRate.toFixed(1), p.rejectionRate.toFixed(1), p.avgDaysPay ?? '', p.receivedRp])
  downloadBlobCsv('bi-penjamin-medpay.csv', rows)
}

onMounted(async () => {
  try {
    const d = await api('dashboard')
    perf.value = d.insurerPerf || []
  } catch (e) { toast(e.message, 'error') }
})
</script>
