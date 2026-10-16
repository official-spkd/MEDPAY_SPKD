<template>
  <div v-if="isLogin" style="min-height:100vh">
    <router-view />
  </div>
  <div v-else :class="['shell', { 'sb-open': state.sidebarMobileOpen }]">
    <Shell />
    <div v-if="state.sidebarMobileOpen" class="sidebar-backdrop" @click="state.sidebarMobileOpen = false"></div>
    <div class="main-wrap" :class="{ expanded: state.sidebarCollapsed && !isMobile }">
      <Topbar />
      <router-view :key="state.scopeVersion" />
      <footer class="footer">
        <b>SPKD MedPay</b>
        <span>Dari selisih tarif sampai uang masuk.</span>
        <span style="margin-left:auto">© 2026 PT Sistem Pelayanan Kesehatan dan Data — Data demo, seluruh data bersifat sintetis.</span>
      </footer>
    </div>
  </div>
  <!-- Toast global: dirender di luar kedua layout agar tampil di semua halaman -->
  <div id="toasts" aria-live="polite">
    <div v-for="t in state.toasts" :key="t.id" class="toast" :class="t.type" @click="state.toasts.splice(state.toasts.indexOf(t), 1)">
      <span class="toast-ico">{{ ICONS[t.type] || 'ℹ' }}</span>{{ t.msg }}
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import Shell from './components/Shell.vue'
import Topbar from './components/Topbar.vue'
import { state, loadMe, loadBoot, refreshBoot, restoreHospitalFilter } from './store'
import { getToken } from './api'

const ICONS = { success: '✓', error: '✕', warning: '⚠', info: 'ℹ' }
const route = useRoute()
const isLogin = computed(() => route.path === '/login')
const isMobile = computed(() => window.innerWidth <= 900)
let resizeHandler
let pollTimer
restoreHospitalFilter()
onMounted(() => {
  resizeHandler = () => { if (window.innerWidth > 900) state.sidebarMobileOpen = false }
  window.addEventListener('resize', resizeHandler)
  // Sinkronkan badge sidebar & notifikasi secara berkala tanpa perlu refresh halaman.
  pollTimer = setInterval(() => { if (getToken() && state.booted && !document.hidden) refreshBoot() }, 20000)
  // Pulihkan sesi user setelah refresh halaman (token masih ada).
  if (getToken() && !state.user) {
    loadMe().then(ok => { if (ok) loadBoot().catch(() => {}) })
  }
})
onUnmounted(() => { window.removeEventListener('resize', resizeHandler); clearInterval(pollTimer) })
</script>
