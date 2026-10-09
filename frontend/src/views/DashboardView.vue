<template>
  <main class="page">
    <PageHeader icon="dashboard" title="Dashboard Eksekutif" sub="Kondisi revenue cycle dalam 30 detik — potensi piutang, dana masuk, risiko klaim, dan tindakan yang perlu dilakukan. Setiap angka dapat diklik ke modul sumbernya.">
      <template #badges>
        <span class="demo-pill"><Icon name="flask" :size="12" /> Data Demo</span>
      </template>
      <template #actions>
        <router-link to="/claims/new" class="btn btn-primary btn-sm" v-if="hasPerm('CREATE')"><Icon name="plus" :size="14" /> Klaim Baru</router-link>
        <button class="btn btn-outline btn-sm" @click="refresh" :disabled="busy"><span v-if="busy" class="spin" /><Icon name="refresh" :size="14" /> Muat Ulang</button>
      </template>
    </PageHeader>

    <div v-if="error" class="alert danger"><span>⚠</span><span>{{ error }} — <a href="#" @click.prevent="refresh" style="text-decoration:underline">coba lagi</a></span></div>

    <div v-if="loading && !d">
      <div class="card" style="height:220px;margin-bottom:18px"><div class="skeleton" style="height:100%;border-radius:18px" /></div>
      <div class="kpi-grid"><div class="kpi" v-for="i in 4" :key="i"><div class="skeleton" style="width:60%;height:12px;margin-bottom:10px" /><div class="skeleton" style="width:80%;height:24px" /></div></div>
      <div class="grid-2"><div class="card card-pad"><div class="skeleton" style="height:160px" /></div><div class="card card-pad"><div class="skeleton" style="height:160px" /></div></div>
    </div>

    <template v-if="d">
      <!-- Baris utama: hero skor + radar risiko + peringatan dini -->
      <div class="grid-hero">
        <div class="hero">
          <div class="gauge-block">
            <div>
              <div class="gauge-label">Recovery<br />Score</div>
              <svg class="gauge-svg" width="132" height="132" viewBox="0 0 132 132" role="img" aria-label="Skor pemulihan">
                <circle cx="66" cy="66" r="54" fill="none" stroke="rgba(255,255,255,0.25)" stroke-width="11" />
                <circle cx="66" cy="66" r="54" fill="none" stroke="#ffffff" stroke-width="11" stroke-linecap="round"
                  :stroke-dasharray="gaugeDash" transform="rotate(-90 66 66)" />
                <text x="66" y="72" text-anchor="middle" class="gauge-num">{{ recoveryScore }}</text>
                <text x="103" y="90" text-anchor="middle" class="gauge-den">/100</text>
              </svg>
            </div>
          </div>
          <div class="hero-info">
            <div class="hero-title">{{ heroTitle }}</div>
            <div class="hero-desc">Rasio dana tertagih terhadap total tagihan COB. Skor memperhitungkan dana diterima, outstanding, dan kesehatan umur piutang.</div>
            <div class="hero-meta">
              <span>Tertagih: <b>{{ formatRpShort(d.receivedRp) }}</b></span>
              <span>Outstanding: <b>{{ formatRpShort(d.outstandingRp) }}</b></span>
              <span v-if="d.dso !== null && d.dso !== undefined">DSO: <b>{{ d.dso }} hari</b></span>
            </div>
            <div class="hero-chips">
              <span class="hero-chip">Eligible COB: <b>{{ d.eligibleClaims }}</b></span>
              <span class="hero-chip">Blocker: <b>{{ d.blockersCount }}</b></span>
              <span class="hero-chip">Deadline ≤6 bln: <b>{{ d.nearDeadline }}</b></span>
              <span class="hero-chip">Resolusi hari ini: <b>{{ d.resolutionsToday }}</b></span>
            </div>
          </div>
          <div class="hero-track">
            <div class="ht-row"><span>Progres pemulihan piutang COB</span><b>{{ recoveryScore }}%</b></div>
            <div class="progress-track"><div class="progress-fill" :style="{ width: recoveryScore + '%' }" /></div>
          </div>
        </div>

        <div class="card card-pad radar-panel">
          <div class="radar-head">
            <span class="ct-ico" style="width:34px;height:34px;border-radius:10px;background:var(--primary-light);color:var(--primary);display:inline-flex;align-items:center;justify-content:center"><Icon name="target" :size="17" /></span>
            <div style="flex:1">
              <div class="rt">Radar Risiko</div>
              <div class="rs">{{ radarItems.length }} risiko terdeteksi · prioritas penanganan</div>
            </div>
            <router-link to="/bi" class="card-link">Semua</router-link>
          </div>
          <div class="radar-list">
            <div v-if="!radarItems.length" class="empty" style="padding:24px 12px">
              <div class="ico">🛡</div><div class="t">Tidak ada risiko material</div>
              <div class="d">Seluruh indikator dalam batas aman.</div>
            </div>
            <div v-for="(r, i) in radarItems" :key="i" class="radar-item" @click="$router.push(r.to)">
              <div class="ri-top">
                <div class="ri-title">{{ r.title }}</div>
                <span class="badge" :class="r.sev === 'Tinggi' ? 'danger' : 'warning'"><Icon name="x" :size="10" /> {{ r.sev }}</span>
              </div>
              <div class="ri-meta">{{ r.meta }}</div>
            </div>
          </div>
        </div>

        <div class="card card-pad radar-panel">
          <div class="radar-head">
            <span style="width:34px;height:34px;border-radius:10px;background:var(--warning-bg);color:var(--warning);display:inline-flex;align-items:center;justify-content:center"><Icon name="alert" :size="17" /></span>
            <div style="flex:1">
              <div class="rt">Peringatan Dini</div>
              <div class="rs">Deteksi dini ambang batas &amp; deadline</div>
            </div>
            <router-link to="/aging" class="card-link">Semua</router-link>
          </div>
          <div class="radar-list">
            <div v-if="!warnItems.length" class="empty" style="padding:24px 12px">
              <div class="ico">🗓</div><div class="t">Aman</div>
              <div class="d">Tidak ada klaim dalam window deadline.</div>
            </div>
            <div v-for="(w, i) in warnItems" :key="i" class="radar-item" @click="$router.push('/claims?q=' + w.id)">
              <div class="ri-top">
                <div class="ri-title">{{ w.title }}</div>
                <span class="badge" :class="w.sev === 'Tinggi' ? 'danger' : 'warning'"><Icon name="x" :size="10" /> {{ w.sev }}</span>
              </div>
              <div class="ri-meta"><span class="mono">{{ w.id }}</span> · {{ w.meta }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- 1. Prioritas: potensi piutang, diterima, tertunggak -->
      <div class="kpi-grid">
        <Kpi icon="banknote" label="Potensi Piutang COB" :value="formatRp(d.outsideBpjsRp)" tone="navy"
          source="Σ selisih tarif klaim Non-PBI aktif. Gap = Tagihan RS − Tarif JKN. Potensi, bukan pendapatan pasti." />
        <Kpi icon="wallet" label="Dana Diterima" :value="formatRp(d.receivedRp)" tone="success" val-class="success"
          source="Σ pembayaran tercatat + konfirmasi rekonsiliasi mutasi bank (modul Pembayaran)." />
        <Kpi icon="clock" label="Masih Tertunggak" :value="formatRp(d.outstandingRp)" tone="warning" val-class="warning"
          source="Σ (gap − pembayaran) klaim belum lunas. Detail per umur piutang di modul Piutang." />
        <Kpi icon="alert" label="Klaim Ber-Blocker" :value="String(d.blockersCount)" tone="danger" :val-class="d.blockersCount ? 'danger' : ''"
          source="Hasil scrubber: BLOCKER terbuka memblokir masuk batch & pengajuan." />
      </div>
      <div class="kpi-grid">
        <Kpi icon="shieldCheck" label="Klaim Eligible COB" :value="String(d.eligibleClaims)" tone="navy"
          source="Klaim Non-PBI dengan gap > 0, belum dihapus buku." />
        <Kpi icon="calendar" label="Dekati Deadline 6 Bulan" :value="String(d.nearDeadline)" tone="warning" :val-class="d.nearDeadline ? 'warning' : ''"
          source="Batas 6 bulan sejak tanggal pelayanan [VERIFIKASI REGULASI]. Waspada ≤ 30 hari, kritis ≤ 7 hari." />
        <Kpi icon="zap" label="Resolusi Tarif Hari Ini" :value="String(d.resolutionsToday)" tone="teal"
          source="Klaim yang tarif INA-CBG-nya diresolve hari ini via grouper e-Klaim (adapter demo)." />
        <Kpi icon="activity" label="DSO (hari)" :value="d.dso !== null && d.dso !== undefined ? String(d.dso) : '—'" tone="teal"
          source="Rata-rata hari dari pengajuan sampai pembayaran diterima (klaim lunas)." />
      </div>

      <!-- 2+3. Work queue + status breakdown -->
      <div class="grid-31" style="margin-bottom:16px">
        <div class="card card-pad">
          <div class="card-title">
            <span class="ct-ico"><Icon name="queue" :size="15" /></span> Antrean Kerja Hari Ini
            <router-link to="/workqueue" class="card-link">Kelola</router-link>
          </div>
          <div class="card-sub">Tindakan yang sedang berjalan (New / Assigned / In Progress). Kelola di modul Antrean Kerja.</div>
          <div v-if="!d.workQueueToday.length"><Empty icon="✅" title="Tidak ada item aktif" desc="Semua tindakan sudah selesai — bagus!" /></div>
          <div class="tbl-wrap" v-else>
            <table class="tbl" style="min-width:520px">
              <thead><tr><th>Item</th><th>Sumber</th><th>Prioritas</th><th>Status</th><th>Tenggat</th></tr></thead>
              <tbody>
                <tr v-for="w in d.workQueueToday.slice(0, 6)" :key="w.id" @click="$router.push('/workqueue')">
                  <td style="max-width:280px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">{{ w.title }}</td>
                  <td><span class="badge navy">{{ w.source }}</span></td>
                  <td><span class="badge" :class="w.priority === 'Kritis' ? 'danger' : w.priority === 'Tinggi' ? 'warning' : 'neutral'">{{ w.priority }}</span></td>
                  <td><span class="badge green">{{ w.status }}</span></td>
                  <td>{{ formatDate(w.dueDate) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
        <div class="card card-pad">
          <div class="card-title"><span class="ct-ico blue"><Icon name="pie" :size="15" /></span> Komposisi Status Klaim</div>
          <div class="card-sub">Seluruh klaim dalam cakupan RS Anda.</div>
          <Donut :data="statusDonut" :center="String(totalClaims)" centerSub="klaim" :formatter="formatNumber" aria-label="komposisi status klaim" />
        </div>
      </div>

      <!-- 4. Aging + 5. Insurer perf -->
      <div class="grid-2" style="margin-bottom:16px">
        <div class="card card-pad">
          <div class="card-title"><span class="ct-ico amber"><Icon name="clock" :size="15" /></span> Ringkasan Umur Piutang</div>
          <div class="card-sub">Basis tanggal pengajuan. Klik modul Piutang untuk drill-down.</div>
          <BarChart :data="agingBars" :formatter="formatRpShort" :short-formatter="formatRpShort" aria-label="umur piutang" />
          <router-link to="/aging" class="btn btn-outline btn-sm" style="margin-top:10px"><Icon name="arrowUpRight" :size="13" /> Buka Piutang (AR Aging)</router-link>
        </div>
        <div class="card card-pad">
          <div class="card-title"><span class="ct-ico"><Icon name="banknote" :size="15" /></span> Performa Penjamin Kedua</div>
          <div class="card-sub">Rata-rata hari bayar & tingkat persetujuan (klaim terkirim).</div>
          <div class="tbl-wrap">
            <table class="tbl" style="min-width:420px">
              <thead><tr><th>Penjamin</th><th class="num">Hari Bayar</th><th class="num">Setuju</th><th class="num">Tolak</th><th class="num">Diterima</th></tr></thead>
              <tbody>
                <tr v-for="p in d.insurerPerf.slice(0, 6)" :key="p.insurerId" @click="$router.push('/bi')">
                  <td>{{ p.name }}</td>
                  <td class="num">{{ p.avgDaysPay ?? '—' }}</td>
                  <td class="num"><span :style="{ color: rateColor(p.approvalRate), fontWeight: 700 }">{{ p.approvalRate.toFixed(0) }}%</span></td>
                  <td class="num">{{ p.rejectionRate.toFixed(0) }}%</td>
                  <td class="num">{{ formatRpShort(p.receivedRp) }}</td>
                </tr>
                <tr v-if="!d.insurerPerf.length"><td colspan="5" style="text-align:center;color:var(--text-muted)">Belum ada aktivitas pengajuan</td></tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <!-- Deadline + recent payments -->
      <div class="grid-2">
        <div class="card card-pad">
          <div class="card-title"><span class="ct-ico red"><Icon name="calendar" :size="15" /></span> Mendekati Batas 6 Bulan <span class="verify-tag">[VERIFIKASI REGULASI]</span></div>
          <div class="card-sub">Klaim aktif yang mendekati/melewati batas pengajuan sejak tanggal pelayanan.</div>
          <div v-if="!d.nearDeadlineClaims.length"><Empty icon="🗓" title="Aman" desc="Tidak ada klaim dalam window deadline." /></div>
          <div class="tbl-wrap" v-else>
            <table class="tbl" style="min-width:460px">
              <thead><tr><th>Klaim</th><th>Pasien</th><th>Deadline</th><th>Sisa</th></tr></thead>
              <tbody>
                <tr v-for="c in d.nearDeadlineClaims" :key="c.id" @click="$router.push('/claims?q=' + c.id)">
                  <td class="mono">{{ c.id }}</td>
                  <td>{{ c.patientName }}</td>
                  <td>{{ formatDate(c.deadline) }}</td>
                  <td><span class="badge" :class="c.zone === 'lewat' ? 'danger' : c.zone === 'kritis' ? 'danger' : 'warning'">
                    {{ c.zone === 'lewat' ? 'Lewat ' + Math.abs(c.daysLeft) + ' hr' : c.daysLeft + ' hr' }}</span></td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
        <div class="card card-pad">
          <div class="card-title"><span class="ct-ico"><Icon name="wallet" :size="15" /></span> Pembayaran Terbaru</div>
          <div class="card-sub">Enam pembayaran terakhir yang tercatat atau dikonfirmasi dari mutasi bank.</div>
          <div v-if="!d.recentPayments.length"><Empty icon="💰" title="Belum ada pembayaran" desc="Catat pembayaran atau konfirmasi rekonsiliasi mutasi bank." /></div>
          <div class="tbl-wrap" v-else>
            <table class="tbl" style="min-width:460px">
              <thead><tr><th>Tanggal</th><th>Klaim</th><th>Penjamin</th><th class="num">Nominal</th></tr></thead>
              <tbody>
                <tr v-for="p in d.recentPayments" :key="p.id" @click="$router.push('/payments')">
                  <td>{{ formatDate(p.receivedDate) }}</td>
                  <td class="mono">{{ p.claimId }}</td>
                  <td style="max-width:180px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">{{ p.payer }}</td>
                  <td class="num" style="font-weight:700">{{ formatRp(p.amount) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </template>
  </main>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api'
import { state, toast, hasPerm, loadBoot } from '../store'
import Kpi from '../components/Kpi.vue'
import PageHeader from '../components/PageHeader.vue'
import Empty from '../components/Empty.vue'
import BarChart from '../components/BarChart.vue'
import Donut from '../components/Donut.vue'
import Icon from '../components/Icon.vue'
import { formatRp, formatRpShort, formatDate, formatNumber, CLAIM_LABELS } from '../format'

const d = ref(null)
const loading = ref(true)
const busy = ref(false)
const error = ref('')

async function refresh() {
  busy.value = true
  error.value = ''
  try {
    await loadBoot()
    d.value = await api('dashboard')
  } catch (e) {
    error.value = e.message
  } finally { loading.value = false; busy.value = false }
}

// ── Hero: skor pemulihan ────────────────────────────────────────────────────
const recoveryScore = computed(() => {
  if (!d.value) return 0
  const total = (d.value.receivedRp || 0) + (d.value.outstandingRp || 0)
  if (total <= 0) return 0
  return Math.round((d.value.receivedRp / total) * 100)
})
const gaugeDash = computed(() => {
  const C = 2 * Math.PI * 54
  return (Math.max(0, Math.min(100, recoveryScore.value)) / 100) * C + ' ' + C
})
const heroTitle = computed(() => {
  const s = recoveryScore.value
  if (s >= 70) return 'Sehat dengan Area Perbaikan'
  if (s >= 40) return 'Perlu Perhatian Manajemen'
  return 'Butuh Tindakan Segera'
})

// ── Radar risiko (turunan data dashboard, tanpa API baru) ───────────────────
const radarItems = computed(() => {
  if (!d.value) return []
  const items = []
  if (d.value.blockersCount > 0) {
    items.push({
      title: `${d.value.blockersCount} klaim dengan blocker scrubber terbuka`,
      meta: 'Pemicu: scrub_findings severity BLOCKER → tidak bisa masuk batch. Selesaikan di modul Validasi.',
      sev: 'Tinggi', to: '/scrubber',
    })
  }
  const b90 = (d.value.agingBuckets || []).filter(b => b.amount > 0 && /90/.test(String(b.label)))
  const oldBucket = b90.length ? b90 : (d.value.agingBuckets || []).filter(b => b.amount > 0).slice(-1)
  if (oldBucket.length) {
    items.push({
      title: `Piutang ${oldBucket[0].label} hari: ${formatRpShort(oldBucket[0].amount)}`,
      meta: 'Pemicu: umur piutang melewati ambang — risiko macet meningkat. Prioritaskan tagihan & sanggahan.',
      sev: 'Tinggi', to: '/aging',
    })
  }
  for (const p of (d.value.insurerPerf || [])) {
    if (p.approvalRate < 60) {
      items.push({
        title: `Approval ${p.name} rendah — ${p.approvalRate.toFixed(0)}%`,
        meta: `Pemicu: rejection_rate >= 40% (${p.rejectionRate.toFixed(0)}% ditolak). Cek pola penolakan per kode.`,
        sev: 'Sedang', to: '/bi',
      })
      break
    }
  }
  return items.slice(0, 4)
})

// ── Peringatan dini (deadline 6 bulan) ──────────────────────────────────────
const warnItems = computed(() => {
  if (!d.value) return []
  return (d.value.nearDeadlineClaims || []).slice(0, 4).map(c => ({
    id: c.id,
    title: `${c.patientName} — ${c.zone === 'lewat' ? 'melewati batas pengajuan' : 'mendekati batas 6 bulan'}`,
    meta: `Pemicu: deadline_info ≤ 30 hari → sisa ${c.zone === 'lewat' ? 'lewat ' + Math.abs(c.daysLeft) : c.daysLeft} hari`,
    sev: (c.zone === 'lewat' || c.zone === 'kritis') ? 'Tinggi' : 'Sedang',
  }))
})

const totalClaims = computed(() => d.value ? Object.values(d.value.statusBreakdown).reduce((a, b) => a + b, 0) : 0)
const statusDonut = computed(() => {
  if (!d.value) return []
  return Object.entries(d.value.statusBreakdown).map(([k, v]) => ({ label: CLAIM_LABELS[k] || k, value: v }))
})
const agingBars = computed(() => d.value ? d.value.agingBuckets.map(b => ({ label: b.label, value: b.amount })) : [])

function rateColor(r) { return r >= 80 ? 'var(--success)' : r >= 60 ? 'var(--warning)' : 'var(--danger)' }

onMounted(async () => {
  if (!state.booted) {
    try { await loadBoot() } catch (e) { toast(e.message, 'error') }
  }
  refresh()
})
</script>
