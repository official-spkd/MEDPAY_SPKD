<template>
  <main class="page">
    <PageHeader icon="calc" title="Simulasi KAPJ (POJK 36/2025)" sub="Pembagian tagihan: BPJS / penjamin kedua / pasien (co-pay). Parameter versi konfigurasi — bukan hard-coded. Hitung ulang otomatis 300 ms setelah parameter berubah." />

    <div class="grid-31">
      <div>
        <div class="kpi-grid">
          <Kpi icon="banknote" label="Total Tagihan RS" :value="formatRp(t.totalHospitalBill)" tone="navy" source="Σ 18 komponen tarif klaim aktif dalam simulasi." />
          <Kpi icon="shieldCheck" label="Dibayar BPJS" :value="formatRp(t.bpjsPays)" tone="navy" :source="bpjsSource" />
          <Kpi icon="wallet" label="Ditanggung Penjamin" :value="formatRp(t.insurerPays)" tone="success" val-class="success" source="Sisa setelah BPJS, dipotong plafon polis &amp; ko-pay pasien." />
          <Kpi icon="users" label="Ko-pay Pasien" :value="formatRp(t.patientPays)" tone="warning" :source="copaySource" />
        </div>

        <div class="tbl-wrap">
          <table class="tbl">
            <thead><tr><th>Klaim</th><th>Pasien</th><th>Penjamin</th><th class="num">Tagihan</th><th class="num">BPJS</th><th class="num">Penjamin</th><th class="num">Pasien</th><th>Status</th></tr></thead>
            <tbody>
              <tr v-for="r in rows" :key="r.claimId">
                <td class="mono">{{ r.claimId }}</td>
                <td>{{ r.patientName }}</td>
                <td style="font-size:12px;max-width:170px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">{{ r.insurer || '—' }}</td>
                <td class="num">{{ formatRp(r.totalHospitalBill) }}</td>
                <td class="num">{{ formatRp(r.bpjsPays) }}</td>
                <td class="num" style="font-weight:700">{{ formatRp(r.insurerPays) }}</td>
                <td class="num">{{ formatRp(r.patientPays) }}</td>
                <td><span class="badge" :class="statusTone(r.status)">{{ r.status }}</span></td>
              </tr>
              <tr style="background:var(--background)">
                <td colspan="3" style="font-weight:700;font-family:var(--font-head)">TOTAL</td>
                <td class="num" style="font-weight:700">{{ formatRp(t.totalHospitalBill) }}</td>
                <td class="num" style="font-weight:700">{{ formatRp(t.bpjsPays) }}</td>
                <td class="num" style="font-weight:700">{{ formatRp(t.insurerPays) }}</td>
                <td class="num" style="font-weight:700">{{ formatRp(t.patientPays) }}</td>
                <td></td>
              </tr>
              <tr v-if="!rows.length"><td colspan="8"><Empty icon="∑" title="Tidak ada klaim" desc="Tidak ada klaim aktif dengan nilai tarif untuk disimulasikan." /></td></tr>
            </tbody>
          </table>
        </div>

        <div class="alert warning" style="margin-top:14px"><span>⚠</span><span><b>Disclaimer (permanen):</b> hasil simulasi bersifat indikatif, bukan angka final. Co-pay bersifat opsional sesuai produk. Plafon kosong diperlakukan tak terbatas sehingga angka dapat lebih tinggi dari realisasi. Regulasi dalam masa transisi hingga 22 Desember 2026. <b>[VERIFIKASI REGULASI]</b></span></div>
      </div>

      <div class="card card-pad" style="align-self:start;position:sticky;top:74px">
        <div class="card-title">Parameter</div>
        <div class="card-sub">Semua parameter &amp; aturan hidup di konfigurasi versi.</div>
        <div class="field">
          <label>Bagian BPJS (% dari Tarif JKN): <b>{{ p.bpjsSharePct }}%</b></label>
          <input type="range" min="0" max="100" v-model.number="p.bpjsSharePct" style="width:100%" />
        </div>
        <div class="field">
          <label style="display:flex;gap:8px;align-items:center"><input type="checkbox" v-model="p.coPayEnabled" /> Co-payment aktif</label>
        </div>
        <div class="field">
          <label>Ko-pay pasien (% bagian penjamin): <b>{{ p.patientCoPayPct }}%</b></label>
          <input type="range" min="0" max="20" v-model.number="p.patientCoPayPct" style="width:100%" :disabled="!p.coPayEnabled" />
        </div>
        <div class="field"><label>Cap ko-pay Rawat Jalan (Rp)</label><MoneyInput v-model="p.outpatientCoPayCap" :disabled="!p.coPayEnabled" /></div>
        <div class="field"><label>Cap ko-pay Rawat Inap (Rp)</label><MoneyInput v-model="p.inpatientCoPayCap" :disabled="!p.coPayEnabled" /></div>
        <button class="btn btn-outline btn-sm" style="width:100%;justify-content:center" @click="reset">Kembalikan ke default</button>
      </div>
    </div>
  </main>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { api } from '../api'
import { toast } from '../store'
import Kpi from '../components/Kpi.vue'
import PageHeader from '../components/PageHeader.vue'
import MoneyInput from '../components/MoneyInput.vue'
import Empty from '../components/Empty.vue'
import { formatRp, formatRpShort, debounce } from '../format'

const p = reactive({ bpjsSharePct: 75, coPayEnabled: true, patientCoPayPct: 5, outpatientCoPayCap: 300000, inpatientCoPayCap: 3000000 })
const rows = ref([])
const t = computed(() => rowsMeta.value.totals || { totalHospitalBill: 0, bpjsPays: 0, insurerPays: 0, patientPays: 0 })
const rowsMeta = ref({})
const bpjsSource = computed(() => 'min(Tarif JKN, ' + p.bpjsSharePct + '% × Tarif JKN)')
const copaySource = computed(() => {
  if (!p.coPayEnabled) return 'Co-pay non-aktif'
  return p.patientCoPayPct + '% × bagian penjamin, cap ' + formatRpShort(p.inpatientCoPayCap) + ' (rawat inap)'
})

async function run() {
  try {
    const res = await api('kapj/simulate', { method: 'POST', body: { params: { ...p } } })
    rows.value = res.rows
    rowsMeta.value = res
  } catch (e) { toast(e.message, 'error') }
}
const debounced = debounce(run, 300)
watch(p, () => debouncedRun(), { deep: true })
function debouncedRun() { debounced() }

function reset() {
  p.bpjsSharePct = 75; p.coPayEnabled = true; p.patientCoPayPct = 5
  p.outpatientCoPayCap = 300000; p.inpatientCoPayCap = 3000000
}

function statusTone(s) {
  return s === 'Lunas Penuh' ? 'success' : s === 'Dibatasi Plafon' ? 'warning' : s === 'Co-pay Maks' ? 'warning' : 'info'
}

onMounted(run)
</script>
