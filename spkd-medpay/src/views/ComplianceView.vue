<template>
  <main class="page">
    <PageHeader icon="scale" title="Kepatuhan Regulasi" sub="Matriks regulasi → persyaratan → proses → data → validasi → fitur → bukti → audit. Versi regulasi eksplisit; item belum terverifikasi ditandai jelas, tidak ada klaim kepatuhan mutlak." />

    <div class="alert info">
      <span>ℹ</span>
      <span>Perangkat ini dirancang untuk mendukung regulasi kesehatan Indonesia yang berlaku dan kebutuhan rumah sakit, <b>tergantung verifikasi regulasi terkini dan implementasi spesifik rumah sakit</b>. Item bertanda <span class="verify-tag">[VERIFIKASI REGULASI]</span> wajib dikonfirmasi ke sumber resmi sebelum digunakan sebagai dasar keputusan.</span>
    </div>

    <div class="kpi-grid">
      <Kpi label="Terverifikasi" :value="String(count('TERVERIFIKASI'))" tone="success" val-class="success" source="Entri matriks dengan reviewer & tanggal review tercatat." />
      <Kpi label="Perlu Verifikasi" :value="String(count('PERLU_VERIFIKASI'))" tone="warning" val-class="warning" source="Tandai [VERIFIKASI REGULASI] — konfirmasi ke sumber resmi." />
      <Kpi label="Digantikan (superseded)" :value="String(count('SUPERSEDED'))" tone="" source="Versi lama dipertahankan untuk jejak versi regulasi." />
    </div>

    <div class="tbl-wrap">
      <table class="tbl" style="min-width:860px">
        <thead><tr><th>Regulasi</th><th>Artikel</th><th>Persyaratan</th><th>Modul / Fitur</th><th>Berlaku</th><th>Status</th><th>Review</th></tr></thead>
        <tbody>
          <tr v-for="r in regs" :key="r.id">
            <td>
              <div style="font-weight:700">{{ r.number }}</div>
              <div style="font-size:11.5px;color:var(--text-muted)">{{ r.title }}</div>
            </td>
            <td style="font-size:12px">{{ r.article }}</td>
            <td style="font-size:12.5px;max-width:300px">{{ r.requirement }} <span v-if="r.note" class="verify-tag">{{ r.note }}</span></td>
            <td style="font-size:12px">{{ r.module }}<div style="font-size:11px;color:var(--text-muted)">{{ r.feature }}</div></td>
            <td style="font-size:12px;white-space:nowrap">{{ formatDate(r.effectiveDate) }}</td>
            <td>
              <span class="badge" :class="r.status === 'TERVERIFIKASI' ? 'success' : r.status === 'PERLU_VERIFIKASI' ? 'warning' : 'neutral'">{{ r.status }}</span>
            </td>
            <td style="font-size:11.5px">{{ formatDate(r.lastReviewed) }}<div style="color:var(--text-muted)">{{ r.reviewer }}</div></td>
          </tr>
        </tbody>
      </table>
    </div>
    <div class="note-strip">Dukungan versi regulasi: VERSI LAMA → DIGANTIKAN → VERSI BARU → ANALISIS DAMPAK → PEMBARUAN DIPERLUKAN. Tidak ada persyaratan regulasi atau aturan tarif yang di-hard-code tanpa informasi versi.</div>
  </main>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { state, loadBoot } from '../store'
import Kpi from '../components/Kpi.vue'
import PageHeader from '../components/PageHeader.vue'
import { formatDate } from '../format'

const regs = computed(() => state.boot?.regulations || [])
function count(s) { return regs.value.filter(r => r.status === s).length }

onMounted(() => loadBoot())
</script>
