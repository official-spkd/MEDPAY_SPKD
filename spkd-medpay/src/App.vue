<template>
  <div v-if="isLogin" style="min-height:100vh">
    <router-view />
    <div id="toasts">
      <div v-for="t in state.toasts" :key="t.id" class="toast" :class="t.type">{{ t.msg }}</div>
    </div>
  </div>
  <div v-else :class="['shell', { 'sb-open': state.sidebarMobileOpen }]">
    <Shell />
    <div class="main-wrap" :class="{ expanded: state.sidebarCollapsed && !isMobile }">
      <Topbar />
      <router-view />
      <footer class="footer">
        <b>SPKD MedPay</b>
        <span>Dari selisih tarif sampai uang masuk.</span>
        <span style="margin-left:auto">© 2026 PT Sistem Pelayanan Kesehatan dan Data — Data demo, seluruh data bersifat sintetis.</span>
      </footer>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import Shell from './components/Shell.vue'
import Topbar from './components/Topbar.vue'
import { state, loadMe, loadBoot } from './store'
import { getToken } from './api'

const route = useRoute()
const isLogin = computed(() => route.path === '/login')
const isMobile = computed(() => window.innerWidth <= 900)
let resizeHandler
onMounted(() => {
  resizeHandler = () => { if (window.innerWidth > 900) state.sidebarMobileOpen = false }
  window.addEventListener('resize', resizeHandler)
  // Pulihkan sesi user setelah refresh halaman (token masih ada).
  if (getToken() && !state.user) {
    loadMe().then(ok => { if (ok) loadBoot().catch(() => {}) })
  }
})
onUnmounted(() => window.removeEventListener('resize', resizeHandler))
</script>
