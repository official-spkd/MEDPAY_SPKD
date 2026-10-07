<template>
  <main class="page">
    <PageHeader icon="file" :title="isEdit ? 'Edit Klaim ' + form.id : 'Klaim COB Baru'" sub="Struktur field mengikuti form e-Klaim INA-CBG. Klaim baru berstatus Draft. LOS dihitung otomatis; PBI tidak eligible COB.">
      <template #actions>
        <router-link to="/claims" class="btn btn-outline btn-sm">‹ Daftar Klaim</router-link>
      </template>
    </PageHeader>

    <div class="stepper">
      <div v-for="(s, i) in steps" :key="i" class="step" :class="{ active: step === i }" @click="step = i" role="button" :aria-label="'Langkah ' + (i + 1)">
        <span class="no">{{ i + 1 }}</span>
        <span><span class="t">{{ s.t }}</span><br /><span class="d">{{ s.d }}</span></span>
      </div>
    </div>

    <!-- Langkah 1 -->
    <div class="card card-pad" v-show="step === 0">
      <div class="card-title">Identitas &amp; Episode Pelayanan</div>
      <div class="card-sub">Kode Faskes terkunci sesuai akun (multi-tenant otomatis, bukan setting UI).</div>
      <div class="form-grid">
        <div class="field">
          <label>Kode Faskes <span class="req">*</span></label>
          <select class="input" v-model="form.hospitalId" :disabled="!isSuper">
            <option v-for="h in state.boot?.hospitals || []" :key="h.id" :value="h.id">{{ h.kodeFaskes }} — {{ h.name }}</option>
          </select>
        </div>
        <div class="field">
          <label>Nama RS (otomatis)</label>
          <input class="input" :value="hospName" disabled />
        </div>
        <div class="field">
          <label>No. SEP <span class="req">*</span></label>
          <input class="input" v-model="form.sepNo" placeholder="0301R001…" />
        </div>
        <div class="field">
          <label>No. Kartu BPJS <span class="req">*</span></label>
          <input class="input" v-model="form.bpjsCardNo" placeholder="0002…" />
        </div>
        <div class="field">
          <label>MRN <span class="req">*</span></label>
          <input class="input" v-model="form.mrn" placeholder="RM-…" />
        </div>
        <div class="field">
          <label>Nama Pasien <span class="req">*</span></label>
          <input class="input" v-model="form.patientName" />
        </div>
        <div class="field">
          <label>DPJP</label>
          <input class="input" v-model="form.dpjp" placeholder="dr. …, Sp.…" />
        </div>
        <div class="field">
          <label>Jenis Peserta <span class="req">*</span></label>
          <select class="input" v-model="form.participantType">
            <option value="NON_PBI">Non-PBI (eligible COB)</option>
            <option value="PBI">PBI — tidak eligible COB</option>
          </select>
          <div class="hint" v-if="form.participantType === 'PBI'" style="color:var(--danger)">PBI tidak dapat diajukan sebagai COB — scrubber R03 akan memblokir.</div>
        </div>
        <div class="field">
          <label>Jenis Rawat <span class="req">*</span></label>
          <select class="input" v-model="form.careType">
            <option value="RAWAT_INAP">Rawat Inap</option>
            <option value="RAWAT_JALAN">Rawat Jalan</option>
          </select>
        </div>
        <div class="field">
          <label>Kelas Rawat <span class="req">*</span></label>
          <select class="input" v-model="form.careClass">
            <option>I</option><option>II</option><option>III</option>
          </select>
        </div>
        <div class="field">
          <label>Jenis Kelamin <span class="req">*</span></label>
          <select class="input" v-model="form.sex"><option value="L">Laki-laki</option><option value="P">Perempuan</option></select>
        </div>
        <div class="field">
          <label>Tanggal Lahir <span class="req">*</span></label>
          <input class="input" type="date" v-model="form.dateOfBirth" />
        </div>
        <div class="field">
          <label>Tanggal Masuk <span class="req">*</span></label>
          <input class="input" type="date" v-model="form.admissionDate" @change="recalcLos" />
        </div>
        <div class="field">
          <label>Tanggal Pulang <span class="req">*</span></label>
          <input class="input" type="date" v-model="form.dischargeDate" @change="recalcLos" />
        </div>
        <div class="field">
          <label>LOS (otomatis — tidak dapat diketik)</label>
          <input class="input" :value="form.los + ' hari'" disabled />
        </div>
      </div>
    </div>

    <!-- Langkah 2 -->
    <div class="card card-pad" v-show="step === 1">
      <div class="card-title">Diagnosa &amp; Prosedur</div>
      <div class="card-sub">ICD-10 utama (satu, wajib), sekunder (bisa banyak, wajib), prosedur ICD-9-CM (opsional — kirim kosong, jangan placeholder).</div>
      <div class="form-grid">
        <div class="field">
          <label>Diagnosa Utama (ICD-10) <span class="req">*</span></label>
          <input class="input" v-model="form.primaryDiagnosis" placeholder="Contoh: I25.1" style="font-family:ui-monospace,monospace" />
        </div>
        <div class="field">
          <label>Tambah Diagnosa Sekunder</label>
          <input class="input" v-model="secDraft" placeholder="Contoh: E11.9 — tekan Enter" @keyup.enter="addCode('secondaryDiagnoses', secDraft); secDraft = ''" />
        </div>
        <div class="field full">
          <label>Diagnosa Sekunder <span class="req">*</span></label>
          <div class="line-item-chips">
            <span v-for="(c, i) in form.secondaryDiagnoses" :key="i" class="code-chip">{{ c }} <button @click="form.secondaryDiagnoses.splice(i, 1)" aria-label="Hapus">✕</button></span>
            <span v-if="!form.secondaryDiagnoses.length" style="color:var(--text-muted);font-size:12px">belum ada</span>
          </div>
        </div>
        <div class="field">
          <label>Tambah Prosedur (ICD-9-CM)</label>
          <input class="input" v-model="procDraft" placeholder="Contoh: 88.55 — tekan Enter" @keyup.enter="addCode('procedures', procDraft); procDraft = ''" />
        </div>
        <div class="field full">
          <label>Prosedur</label>
          <div class="line-item-chips">
            <span v-for="(c, i) in form.procedures" :key="i" class="code-chip">{{ c }} <button @click="form.procedures.splice(i, 1)" aria-label="Hapus">✕</button></span>
            <span v-if="!form.procedures.length" style="color:var(--text-muted);font-size:12px">kosong (dikirim kosong, bukan placeholder)</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Langkah 3 -->
    <div class="card card-pad" v-show="step === 2">
      <div class="card-title">Tarif — Resolusi INA-CBG &amp; Komponen RS</div>
      <div class="card-sub">“Resolve INA-CBG Tariff” memanggil grouper e-Klaim melalui adapter (demo deterministik — bukan integrasi nyata).</div>
      <div class="chips-row" style="margin-bottom:14px">
        <button class="btn btn-teal btn-sm" :disabled="resolving" @click="resolveTariff" v-if="hasPerm('RESOLVE') || isEdit === false"><span v-if="resolving" class="spin" /> Resolve Tarif INA-CBG</button>
        <span v-if="form.inacbgCode" class="code-chip">{{ form.inacbgCode }} — {{ form.inacbgDescription }} · {{ formatRp(form.inacbgBaseTariff) }}</span>
      </div>
      <div class="alert danger" v-if="resolveErr"><span>⛔</span><span>Grouper gagal: <b>{{ resolveErr }}</b> — {{ resolveErr === 'EKLAIM_CONFIG_MISSING' ? 'konfigurasi e-Klaim RS belum tersedia (atur di Pengaturan).' : 'kombinasi diagnosa tidak dapat digrouping.' }}</span></div>

      <div style="display:grid;grid-template-columns:1fr 1fr;gap:16px" class="two-panel">
        <div>
          <div class="card-title" style="font-size:13px">Special CMG / Top Up (6)</div>
          <div class="hint" style="margin-bottom:8px">Top Up Tariff = Σ enam komponen. Total Tarif JKN = Tarif Murni + Top Up (angka yang dibayar BPJS).</div>
          <div v-for="k in SPECIAL_CMG_KEYS" :key="k" class="field">
            <label>{{ SPECIAL_CMG_LABELS[k] }}</label>
            <MoneyInput v-model="form.specialCmg[k]" />
          </div>
        </div>
        <div>
          <div class="card-title" style="font-size:13px">18 Komponen Tarif RS</div>
          <div class="hint" style="margin-bottom:8px">Total Tarif RS = Σ 18 komponen (nilai tagihan rumah sakit).</div>
          <div v-for="k in TARIFF_COMPONENT_KEYS" :key="k" class="field">
            <label>{{ COMPONENT_LABELS[k] }}</label>
            <MoneyInput v-model="form.hospitalTariff[k]" />
          </div>
        </div>
      </div>
      <div class="gap-box" :class="gap >= 0 ? 'pos' : 'neg'">
        <span>{{ gap >= 0 ? 'Selisih Tarif (Gap) — potensi piutang COB' : 'Negatif — tidak ada selisih untuk diklaim (R08)' }}</span>
        <span>{{ formatRp(gap) }}</span>
      </div>
      <div class="alert warning" v-if="hospTotal > 0 && hospTotal < jknTot"><span>⚠</span><span>Total tagihan RS lebih rendah dari tarif INA-CBG — BPJS dapat mem-pending klaim. Periksa kembali komponen atau resolusi tarif.</span></div>
      <div class="field" style="margin-top:14px">
        <label style="display:flex;gap:9px;align-items:flex-start;cursor:pointer">
          <input type="checkbox" v-model="form.declarationChecked" style="margin-top:3px" />
          <span><b>Pernyataan kebenaran data</b> — saya menyatakan data klaim ini benar dan dapat dipertanggungjawabkan. Wajib dicentang sebelum klaim disiapkan pengajuan (R13).</span>
        </label>
      </div>
    </div>

    <!-- Langkah 4 -->
    <div class="card card-pad" v-show="step === 3">
      <div class="card-title">Penjamin Kedua (Asuransi / COB)</div>
      <div class="card-sub">Penjamin dipilih dari daftar tertutup nama resmi e-Klaim (55 penjamin; subset demo di Library Penjamin) — tanpa teks bebas.</div>
      <div class="form-grid">
        <div class="field full">
          <label>Penjamin (COB) <span class="req">*</span></label>
          <SearchSelect ref="insSel" :options="insurerOptions" v-model="form.insurerId" placeholder="Cari nama penjamin…" />
        </div>
        <div class="field">
          <label>No. Polis <span class="req">*</span></label>
          <input class="input" v-model="form.policyNo" />
        </div>
        <div class="field">
          <label>Nama Tertanggung <span class="req">*</span></label>
          <input class="input" v-model="form.insuredName" />
        </div>
        <div class="field">
          <label>Nama Penjamin Tambahan (opsional)</label>
          <input class="input" v-model="form.insurerNameExtra" />
        </div>
        <div class="field">
          <label>Jenis Asuransi</label>
          <input class="input" v-model="form.insuranceType" placeholder="Individu / Keluarga / Perusahaan" />
        </div>
        <div class="field">
          <label>No. Anggota</label>
          <input class="input" v-model="form.insuranceMemberNo" />
        </div>
        <div class="field">
          <label>Jenis Perlindungan</label>
          <select class="input" v-model="form.coverageType">
            <option>Naik Kelas</option><option>Selisih Kelas</option><option>Manfaat Rawat Inap</option><option>Komersial Umum</option>
          </select>
          <div class="hint" v-if="form.careClass === 'III' && form.coverageType.toLowerCase().includes('selisih kelas')" style="color:var(--warning)">Kelas III + manfaat selisih kelas → scrubber R12 memberi peringatan.</div>
        </div>
        <div class="field">
          <label>Plafon / Ceiling Polis (opsional)</label>
          <MoneyInput v-model="form.policyCeilingInput" />
          <div class="hint">Kosong = dianggap tak terbatas (angka KAPJ bisa lebih tinggi).</div>
        </div>
      </div>
      <div class="alert info" v-if="selectedInsurer"><span>ℹ</span><span>Aturan penjamin (Payer Rule Library): rata-rata bayar <b>{{ selectedInsurer.avgPaymentDays }} hari</b>, batas kontraktual pengajuan <b>{{ selectedInsurer.deadlineDays }} hari</b>, kebijakan plafon: {{ selectedInsurer.ceilingPolicy }}. Status: <span class="badge" :class="selectedInsurer.verified ? 'success' : 'warning'">{{ selectedInsurer.verified ? 'Terverifikasi' : 'Perlu verifikasi' }}</span></span></div>
    </div>

    <div style="display:flex;gap:10px;margin-top:18px;flex-wrap:wrap">
      <button class="btn btn-outline" :disabled="step === 0" @click="step--">‹ Sebelumnya</button>
      <button class="btn btn-outline" v-if="step < 3" @click="step++">Berikutnya ›</button>
      <span style="flex:1" />
      <button class="btn btn-teal" :disabled="busy" @click="save">
        <span v-if="busy" class="spin" style="border-top-color:#fff" /> {{ isEdit ? 'Simpan Perubahan' : 'Simpan Draft Klaim' }}
      </button>
    </div>
  </main>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
import { state, hasPerm, toast, loadBoot } from '../store'
import MoneyInput from '../components/MoneyInput.vue'
import PageHeader from '../components/PageHeader.vue'
import SearchSelect from '../components/SearchSelect.vue'
import { formatRp, sumTariff, sumCmg, parseMoney, addDays } from '../format'

const route = useRoute()
const router = useRouter()
const steps = [
  { t: 'Identitas & Episode', d: 'SEP, peserta, tanggal' },
  { t: 'Diagnosa & Prosedur', d: 'ICD-10 / ICD-9-CM' },
  { t: 'Tarif', d: 'Grouper + 18 komponen' },
  { t: 'Penjamin Kedua', d: 'Polis & plafon' },
]
const isEdit = computed(() => !!route.params.id)
const isSuper = computed(() => state.user?.role === 'SUPER_ADMIN')
const step = ref(0)
const busy = ref(false)
const resolving = ref(false)
const resolveErr = ref('')
const secDraft = ref('')
const procDraft = ref('')
const insSel = ref(null)

function blankForm() {
  const cmg = {}, comp = {}
  for (const k of SPECIAL_CMG_KEYS_LIST) cmg[k] = 0
  for (const k of TARIFF_KEYS_LIST) comp[k] = 0
  return {
    hospitalId: state.user?.hospitalId || '', sepNo: '', bpjsCardNo: '', mrn: '', patientName: '',
    dpjp: '', participantType: 'NON_PBI', careType: 'RAWAT_INAP', careClass: 'I', sex: 'L',
    dateOfBirth: '', admissionDate: '', dischargeDate: '', los: 0,
    primaryDiagnosis: '', secondaryDiagnoses: [], procedures: [],
    inacbgCode: '', inacbgDescription: '', inacbgBaseTariff: 0, specialCmg: cmg, hospitalTariff: comp,
    declarationChecked: false, insurerId: '', policyNo: '', insuredName: '', insurerNameExtra: '',
    insuranceType: 'Individu', insuranceMemberNo: '', coverageType: 'Naik Kelas', policyCeiling: null,
    policyCeilingInput: 0,
  }
}

import { SPECIAL_CMG_KEYS as SPECIAL_CMG_KEYS_LIST, TARIFF_COMPONENT_KEYS as TARIFF_KEYS_LIST, SPECIAL_CMG_LABELS, COMPONENT_LABELS } from '../format'

const form = ref(blankForm())
// Jika sesi user dipulihkan setelah form dirender (refresh halaman), isi Kode Faskes otomatis.
watch(() => state.user?.hospitalId, (hid) => {
  if (hid && !form.value.hospitalId && !isEdit.value) form.value.hospitalId = hid
}, { immediate: true })

const hospName = computed(() => state.boot?.hospitals.find(h => h.id === form.value.hospitalId)?.name || '—')
const insurerOptions = computed(() => (state.boot?.insurers || []).map(i => ({ id: i.id, label: i.name, sub: i.type })))
const selectedInsurer = computed(() => state.boot?.insurers.find(i => i.id === form.value.insurerId) || null)
const hospTotal = computed(() => sumTariff(form.value.hospitalTariff))
const jknTot = computed(() => form.value.inacbgBaseTariff + sumCmg(form.value.specialCmg))
const gap = computed(() => hospTotal.value - jknTot.value)

function recalcLos() {
  const a = form.value.admissionDate, b = form.value.dischargeDate
  if (!a || !b) { form.value.los = 0; return }
  const d = Math.round((new Date(b + 'T00:00:00') - new Date(a + 'T00:00:00')) / 86400000)
  form.value.los = Math.max(0, d)
}

function addCode(list, code) {
  const c = (code || '').trim().toUpperCase()
  if (c) form.value[list].push(c)
}

async function resolveTariff() {
  resolveErr.value = ''
  if (!form.value.primaryDiagnosis) { toast('Isi diagnosa utama terlebih dahulu (Langkah 2).', 'warning'); return }
  resolving.value = true
  // Grouper mock client-side deterministik — sama seperti adapter server (demo).
  const dx = form.value.primaryDiagnosis
  if (!/^[A-Za-z]\d{2}/.test(dx)) {
    resolveErr.value = 'UNGROUPABLE'
    resolving.value = false
    return
  }
  const chapter = dx[0].toUpperCase()
  const care = form.value.careType === 'RAWAT_INAP' ? '1' : '3'
  const classDigit = { I: '1', II: '2', III: '3' }[form.value.careClass] || '3'
  const severity = form.value.los > 6 ? '2' : '1'
  const code = form.value.careType === 'RAWAT_INAP'
    ? chapter + '-' + care + '-' + severity + classDigit + '-' + form.value.careClass
    : chapter + '-' + care + '-1' + severity + '-0'
  const seed = (dx.charCodeAt(0) * 7 + dx.charCodeAt(1) * 13) % 900
  const mult = form.value.careType === 'RAWAT_INAP' ? 4500000 : 750000
  form.value.inacbgCode = code
  form.value.inacbgBaseTariff = Math.round(mult + seed * 5000 + form.value.los * 120000)
  form.value.inacbgDescription = 'GROUP INA-CBG TERPILIH (DEMO)'
  toast('Tarif diresolve (adapter demo): ' + code + ' = ' + formatRp(form.value.inacbgBaseTariff), 'success')
  resolving.value = false
}

async function save() {
  busy.value = true
  try {
    const body = { ...form.value }
    body.policyCeiling = body.policyCeilingInput > 0 ? body.policyCeilingInput : null
    delete body.policyCeilingInput
    if (!body.hospitalId) { toast('Pilih Kode Faskes.', 'warning'); busy.value = false; return }
    if (isEdit.value) {
      await api('claims/' + route.params.id, { method: 'PUT', body })
      toast('Klaim diperbarui.', 'success')
    } else {
      const res = await api('claims', { method: 'POST', body })
      toast('Draft klaim ' + res.id + ' tersimpan.', 'success')
    }
    router.push('/claims')
  } catch (e) {
    toast(e.message, 'error')
  } finally { busy.value = false }
}

onMounted(async () => {
  await loadBoot()
  if (isEdit.value) {
    const res = await api('claims/' + route.params.id)
    const c = res.claim
    form.value = { ...blankForm(), ...c, policyCeilingInput: c.policyCeiling || 0 }
    if (insSel.value) insSel.value.setText(state.boot.insurers.find(i => i.id === c.insurerId)?.name || '')
  } else {
    form.value.admissionDate = addDays(new Date().toISOString().slice(0, 10), -5)
    form.value.dischargeDate = addDays(new Date().toISOString().slice(0, 10), -1)
    recalcLos()
  }
})
</script>
