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
            <div class="notif-head">Notifikasi</div>
            <div v-if="!notifs.length" style="padding:14px;text-align:center;color:var(--text-muted)">Tidak ada notifikasi</div>
            <div v-for="n in notifs" :key="n.id" class="notif-item" @click="state.notifOpen = false">
              <div>{{ n.message }}</div>
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
  </header>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import Icon from './Icon.vue'
import SearchModal from './SearchModal.vue'
import { state, toggleSidebar, toggleTheme, myHospital, logout } from '../store'
import { ROLE_LABELS, formatDateTime as fmtDateTime } from '../format'

const route = useRoute()
const searchOpen = ref(false)
const roleLabel = computed(() => ROLE_LABELS[state.user?.role] || state.user?.role)
const myHosp = computed(() => myHospital())
const notifs = computed(() => (state.boot?.notifications || []).slice(0, 12))
const notifCount = computed(() => notifs.value.length)

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
