<template>
  <header style="position:sticky;top:0;z-index:30">
    <div class="topbar">
      <button class="burger" @click="toggleSidebar" aria-label="Buka/tutup menu"><Icon name="menu" :size="19" /></button>
      <div class="faskes-id" v-if="myHosp">
        <span class="h-ico"><Icon name="hospital" :size="17" /></span>
        <span class="h-name">{{ myHosp.name }}</span>
        <span class="chip green">Kelas {{ myHosp.class }}</span>
        <span class="chip teal faskes" v-if="myHosp.eklaimConfigured"><Icon name="check" :size="11" /> e-Klaim Aktif</span>
      </div>
      <div class="faskes-id" v-else>
        <span class="h-ico"><Icon name="hospital" :size="17" /></span>
        <span class="h-name">Semua Rumah Sakit</span>
        <span class="chip green">SUPER ADMIN</span>
      </div>
      <div class="topbar-right">
        <button class="search-box" @click="searchOpen = true" aria-label="Buka pencarian global (Ctrl+K)">
          <Icon name="search" :size="15" />
          <span class="search-ph">Cari klaim, SEP, pasien…</span>
          <kbd>Ctrl K</kbd>
        </button>
        <button class="icon-btn" @click="toggleTheme" :aria-label="'Ganti tema ' + state.theme" title="Mode terang/gelap">
          <Icon :name="state.theme === 'light' ? 'moon' : 'sun'" :size="16" />
        </button>
        <div style="position:relative">
          <button class="icon-btn" @click="state.notifOpen = !state.notifOpen; state.userOpen = false" aria-label="Notifikasi">
            <Icon name="bell" :size="16" /><span v-if="notifCount" class="dot-badge">{{ notifCount }}</span>
          </button>
          <div v-if="state.notifOpen" class="notif-pop">
            <div class="notif-head" style="display:flex;justify-content:space-between;align-items:center">
              <span>Notifikasi</span>
              <button v-if="notifCount" class="btn btn-outline btn-xs" @click.stop="readAll">Tandai semua dibaca</button>
            </div>
            <div v-if="!notifs.length" style="padding:14px;text-align:center;color:var(--text-muted)">Tidak ada notifikasi</div>
            <div v-for="n in notifs" :key="n.id" class="notif-item" :style="n.read ? 'opacity:.6' : ''" @click="openNotif(n)">
              <div><b v-if="!n.read && n.recipientUserId === state.user?.id">● </b>{{ n.message }}</div>
              <div class="tm">{{ fmtDateTime(n.at) }} · {{ n.channel }} · {{ n.template }}</div>
            </div>
          </div>
        </div>
        <button class="avatar" @click="state.userOpen = !state.userOpen; state.notifOpen = false"
          :aria-label="'Akun ' + state.user?.name">{{ state.user?.initials }}</button>
        <div class="user-id" @click="state.userOpen = !state.userOpen; state.notifOpen = false">
          <div class="un">{{ state.user?.name }}</div>
          <div class="ur">{{ roleLabel }}</div>
        </div>
        <div v-if="state.userOpen" class="user-pop">
          <div class="nm">{{ state.user?.name }}</div>
          <div class="rl">{{ state.user?.email }}</div>
          <div class="rl" style="margin-bottom:12px"><span class="badge green">{{ roleLabel }}</span></div>
          <button class="btn btn-outline btn-sm" style="width:100%;justify-content:center;margin-bottom:8px" @click="openPw(false)">
            <Icon name="settings" :size="14" /> Ganti Kata Sandi
          </button>
          <button class="btn btn-outline btn-sm" style="width:100%;justify-content:center" @click="doLogout">
            <Icon name="logout" :size="14" /> Keluar
          </button>
        </div>
      </div>
    </div>
    <div class="subbar">
      <span class="crumb">
        <a href="#/">MedPay</a>
        <span class="crumb-sep"><Icon name="chevRight" :size="11" /></span>
        <template v-if="route.meta.crumb && route.meta.crumb !== 'Beranda'">
          <span>{{ route.meta.crumb }}</span>
          <span class="crumb-sep"><Icon name="chevRight" :size="11" /></span>
        </template>
        <b>{{ route.meta.title }}</b>
      </span>
      <span class="sub-right">
        <Icon name="calendar" :size="13" />
        <span>{{ todayLabel }}</span>
      </span>
    </div>
    <SearchModal v-model:open="searchOpen" />
    <Modal v-if="pw.open" :title="pw.forced ? 'Ganti Kata Sandi (wajib)' : 'Ganti Kata Sandi'" @close="closePw">
      <div v-if="pw.forced" class="alert warning" style="margin-bottom:12px"><span>⚠</span><span>Kata sandi Anda dibuat/direset admin. Ganti kata sandi sebelum melanjutkan.</span></div>
      <div class="field"><label>Kata sandi saat ini</label><input class="input" type="password" v-model="pw.current" autocomplete="current-password" /></div>
      <div class="field"><label>Kata sandi baru</label><input class="input" type="password" v-model="pw.next" autocomplete="new-password" />
        <div class="hint">Minimal 10 karakter, mengandung huruf besar, huruf kecil, dan angka.</div></div>
      <div class="field"><label>Ulangi kata sandi baru</label><input class="input" type="password" v-model="pw.confirm" autocomplete="new-password" /></div>
      <div v-if="pw.error" class="alert danger"><span>✕</span><span>{{ pw.error }}</span></div>
      <template #footer>
        <button class="btn btn-outline" @click="pw.forced ? doLogout() : closePw()">{{ pw.forced ? 'Keluar' : 'Batal' }}</button>
        <button class="btn btn-teal" :disabled="pw.busy || !pw.current || !pw.next" @click="changePw">Simpan</button>
      </template>
    </Modal>
  </header>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Icon from './Icon.vue'
import SearchModal from './SearchModal.vue'
import Modal from './Modal.vue'
import { api } from '../api'
import { state, toggleSidebar, toggleTheme, myHospital, logout, toast } from '../store'
import { ROLE_LABELS, formatDateTime as fmtDateTime } from '../format'

const route = useRoute()
const router = useRouter()
const searchOpen = ref(false)
const roleLabel = computed(() => ROLE_LABELS[state.user?.role] || state.user?.role)
const myHosp = computed(() => myHospital())
const notifs = computed(() => (state.boot?.notifications || []).slice(0, 12))
// Notifikasi belum dibaca milik pengguna ini (notifikasi untuk penerima lain hanya tampil sebagai log).
const notifCount = computed(() => notifs.value.filter(n => !n.read && n.recipientUserId === state.user?.id).length)

async function openNotif(n) {
  state.notifOpen = false
  if (!n.read && n.recipientUserId === state.user?.id) {
    try { await api('notifications/' + n.id + '/read', { method: 'PUT' }); n.read = true } catch { /* abaikan */ }
  }
  if (n.claimId) router.push('/claims?open=' + n.claimId + '&q=' + encodeURIComponent(n.claimNo || ''))
}

async function readAll() {
  try {
    await api('notifications/read-all', { method: 'POST' })
    for (const n of state.boot?.notifications || []) if (n.recipientUserId === state.user?.id) n.read = true
  } catch (e) { toast(e.message, 'error') }
}

// Ganti kata sandi (SRS: wajib saat sandi dibuat/direset admin).
const pw = reactive({ open: false, forced: false, current: '', next: '', confirm: '', error: '', busy: false })
function openPw(forced) {
  Object.assign(pw, { open: true, forced, current: '', next: '', confirm: '', error: '', busy: false })
  state.userOpen = false
}
function closePw() { if (!pw.forced) pw.open = false }
async function changePw() {
  pw.error = ''
  if (pw.next !== pw.confirm) { pw.error = 'Konfirmasi kata sandi baru tidak sama.'; return }
  pw.busy = true
  try {
    await api('auth/change-password', { method: 'POST', body: { currentPassword: pw.current, newPassword: pw.next } })
    if (state.user) state.user.mustChangePassword = false
    pw.forced = false
    pw.open = false
    toast('Kata sandi diganti. Sesi lain telah dikeluarkan.', 'success')
  } catch (e) { pw.error = e.message } finally { pw.busy = false }
}
watch(() => state.user?.mustChangePassword, (v) => { if (v) openPw(true) }, { immediate: true })

const todayLabel = computed(() =>
  new Date().toLocaleDateString('id-ID', { weekday: 'long', day: 'numeric', month: 'short', year: 'numeric' }))

function doLogout() { logout() }

function onKey(e) {
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    searchOpen.value = true
  }
}
function onDocClick(e) {
  if (!e.target.closest('.notif-pop') && !e.target.closest('.icon-btn') && !e.target.closest('.user-pop') && !e.target.closest('.user-id') && !e.target.closest('.avatar')) {
    state.notifOpen = false
    state.userOpen = false
  }
}
onMounted(() => {
  window.addEventListener('keydown', onKey)
  document.addEventListener('click', onDocClick)
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKey)
  document.removeEventListener('click', onDocClick)
})
</script>
