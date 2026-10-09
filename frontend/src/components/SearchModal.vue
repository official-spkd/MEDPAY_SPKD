<template>
  <Teleport to="body">
    <Transition name="sm-fade">
      <div v-if="open" class="sm-overlay" @click.self="close">
        <div class="sm-panel" role="dialog" aria-modal="true" aria-label="Pencarian global">
          <div class="sm-input-row">
            <span class="sm-ico"><Icon name="search" :size="17" /></span>
            <input ref="inputEl" v-model="q" class="sm-input" placeholder="Cari halaman, klaim, SEP, pasien, MRN…"
              @keydown.down.prevent="move(1)" @keydown.up.prevent="move(-1)"
              @keydown.enter.prevent="pickActive" @keydown.esc.prevent="close" />
            <button class="sm-esc" @click="close">esc</button>
          </div>

          <div class="sm-body">
            <!-- Navigasi -->
            <div v-if="navResults.length" class="sm-group">
              <div class="sm-group-label">Navigasi</div>
              <button v-for="(r, i) in navResults" :key="'n' + r.to" class="sm-item"
                :class="{ active: flatIndex('nav', i) === active }" @mouseenter="active = flatIndex('nav', i)" @click="go(r)">
                <span class="sm-item-ico"><Icon :name="r.ico" :size="15" /></span>
                <span class="sm-item-txt">
                  <span class="sm-item-title">{{ r.label }}</span>
                  <span class="sm-item-sub">{{ r.group }} · {{ r.desc }}</span>
                </span>
                <span class="sm-arrow"><Icon name="chevRight" :size="13" /></span>
              </button>
            </div>

            <!-- Klaim -->
            <div v-if="claimResults.length" class="sm-group">
              <div class="sm-group-label">Klaim COB</div>
              <button v-for="(c, i) in claimResults" :key="'c' + c.id" class="sm-item"
                :class="{ active: flatIndex('claim', i) === active }" @mouseenter="active = flatIndex('claim', i)" @click="goClaim(c)">
                <span class="sm-item-ico claim"><Icon name="claims" :size="15" /></span>
                <span class="sm-item-txt">
                  <span class="sm-item-title">{{ c.patientName }} <span class="mono sm-mono">{{ c.id }}</span></span>
                  <span class="sm-item-sub">SEP {{ c.sepNo }} · MRN {{ c.mrn }} · {{ insurerName(c.insurerId) }}</span>
                </span>
                <span class="badge sm-badge" :class="CLAIM_TONE[c.status]">{{ CLAIM_LABELS[c.status] }}</span>
              </button>
            </div>

            <!-- Kosong -->
            <div v-if="q && !navResults.length && !claimResults.length" class="sm-empty">
              <div class="sm-empty-ico"><Icon name="search" :size="22" /></div>
              <div class="sm-empty-t">Tidak ada hasil untuk “{{ q }}”</div>
              <div class="sm-empty-d">Coba kata kunci lain: nama pasien, nomor SEP, MRN, atau nama modul.</div>
            </div>

            <!-- Default: pintasan -->
            <div v-if="!q" class="sm-group">
              <div class="sm-group-label">Pintasan</div>
              <div class="sm-hints">
                <span class="sm-hint" v-for="h in quickHints" :key="h.to" @click="go(h)">
                  <Icon :name="h.ico" :size="13" /> {{ h.label }}
                </span>
              </div>
            </div>
          </div>

          <div class="sm-foot">
            <span><kbd>↑</kbd><kbd>↓</kbd> navigasi</span>
            <span><kbd>↵</kbd> buka</span>
            <span><kbd>esc</kbd> tutup</span>
            <span class="sm-foot-right" v-if="totalResults">{{ totalResults }} hasil</span>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import Icon from './Icon.vue'
import { state, hasPerm } from '../store'
import { CLAIM_LABELS, CLAIM_TONE } from '../format'

const props = defineProps({ open: Boolean })
const emit = defineEmits(['update:open'])

const router = useRouter()
const q = ref('')
const active = ref(0)
const inputEl = ref(null)

// ── Katalog navigasi (selaras sidebar) ─────────────────────────────────────
const NAV = [
  { to: '/', label: 'Dashboard', ico: 'dashboard', group: 'Utama', desc: 'Ringkasan eksekutif revenue cycle', kw: 'beranda home ringkasan' },
  { to: '/workqueue', label: 'Antrean Kerja', ico: 'queue', group: 'Utama', desc: 'Tindakan & penugasan', kw: 'tugas workitem todo' },
  { to: '/claims', label: 'Daftar Klaim', ico: 'claims', group: 'Klaim', desc: 'Klaim COB penjamin kedua', kw: 'klaim cob daftar list' },
  { to: '/claims/new', label: 'Klaim Baru', ico: 'plus', group: 'Klaim', desc: 'Buat klaim COB', kw: 'buat tambah tambah klaim' },
  { to: '/scrubber', label: 'Validasi (Scrubber)', ico: 'shieldCheck', group: 'Klaim', desc: 'Aturan validasi & blocker', kw: 'validasi rule blocker scrub' },
  { to: '/batches', label: 'Batch & Pengajuan', ico: 'layers', group: 'Klaim', desc: 'Kelompok klaim per penjamin', kw: 'batch kirim pengajuan' },
  { to: '/payments', label: 'Pembayaran', ico: 'wallet', group: 'Uang', desc: 'Pembayaran & rekonsiliasi bank', kw: 'bayar rekonsiliasi mutasi bank' },
  { to: '/aging', label: 'Piutang (AR Aging)', ico: 'clock', group: 'Uang', desc: 'Umur piutang & DSO', kw: 'piutang aging dso ar' },
  { to: '/analytics', label: 'Analitik', ico: 'chart', group: 'Wawasan', desc: 'Tren klaim & gap tarif', kw: 'tren grafik analitik chart' },
  { to: '/kapj', label: 'Simulasi KAPJ', ico: 'calc', group: 'Wawasan', desc: 'Pembagian tagihan BPJS/COB/pasien', kw: 'kapj simulasi pojk co-pay' },
  { to: '/bi', label: 'BI & Laporan', ico: 'pie', group: 'Wawasan', desc: 'Laporan manajemen', kw: 'laporan bi report manajemen' },
  { to: '/engineapi', label: 'Engine API', ico: 'code', group: 'Sistem', desc: 'Monitoring intake partner', kw: 'engine api partner intake', roles: ['SUPER_ADMIN', 'IT_SIMRS', 'MANAGEMENT', 'AUDITOR'] },
  { to: '/audit', label: 'Audit Trail', ico: 'fingerprint', group: 'Sistem', desc: 'Log aktivitas append-only', kw: 'audit log jejak' },
  { to: '/compliance', label: 'Kepatuhan Regulasi', ico: 'scale', group: 'Sistem', desc: 'Matriks regulasi & bukti', kw: 'kepatuhan regulasi compliance' },
  { to: '/settings', label: 'Pengaturan', ico: 'settings', group: 'Sistem', desc: 'RS, pengguna, penjamin', kw: 'pengaturan setting konfigurasi', perm: 'CONFIGURE' },
]

const navAvailable = computed(() => NAV.filter(n => {
  if (n.roles && !n.roles.includes(state.user?.role)) return false
  if (n.perm && !hasPerm(n.perm)) return false
  return true
}))

const quickHints = computed(() => navAvailable.value.filter(n =>
  ['/', '/claims', '/workqueue', '/payments', '/aging', '/analytics'].includes(n.to)))

// ── Hasil pencarian ────────────────────────────────────────────────────────
const navResults = computed(() => {
  const s = q.value.trim().toLowerCase()
  if (!s) return []
  return navAvailable.value.filter(n =>
    n.label.toLowerCase().includes(s) || n.group.toLowerCase().includes(s) || n.kw.includes(s)).slice(0, 5)
})

const claimResults = computed(() => {
  const s = q.value.trim().toLowerCase()
  if (!s || s.length < 2 || !state.boot?.claims) return []
  const hid = state.user?.hospitalId
  return state.boot.claims
    .filter(c => (!hid || c.hospitalId === hid) && (
      c.id.toLowerCase().includes(s) || c.sepNo?.toLowerCase().includes(s) ||
      c.patientName.toLowerCase().includes(s) || c.mrn?.toLowerCase().includes(s)))
    .slice(0, 6)
})

const totalResults = computed(() => navResults.value.length + claimResults.value.length)

// ── Navigasi keyboard ──────────────────────────────────────────────────────
function flatIndex(section, i) {
  return section === 'nav' ? i : navResults.value.length + i
}
const flatItems = computed(() => [
  ...navResults.value.map(r => () => go(r)),
  ...claimResults.value.map(c => () => goClaim(c)),
])
function move(dir) {
  if (!flatItems.value.length) return
  active.value = (active.value + dir + flatItems.value.length) % flatItems.value.length
  nextTick(() => document.querySelector('.sm-item.active')?.scrollIntoView({ block: 'nearest' }))
}
function pickActive() { flatItems.value[active.value]?.() }

function go(r) { close(); router.push(r.to) }
function goClaim(c) { close(); router.push('/claims?q=' + encodeURIComponent(c.id)) }
function insurerName(id) { return state.boot?.insurers?.find(i => i.id === id)?.shortName || state.boot?.insurers?.find(i => i.id === id)?.name || '—' }
function close() { emit('update:open', false) }

watch(() => props.open, async (v) => {
  if (v) {
    q.value = ''
    active.value = 0
    await nextTick()
    inputEl.value?.focus()
  }
})
watch(q, () => { active.value = 0 })
</script>
