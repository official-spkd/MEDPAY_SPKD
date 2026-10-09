<template>
  <aside class="sidebar" :class="{ collapsed: state.sidebarCollapsed, open: state.sidebarMobileOpen }">
    <div class="brand">
      <Logo light :fs="19" :bx="38" sub="BY SPKD" />
      <button class="sb-close" @click="state.sidebarMobileOpen = false" aria-label="Tutup menu">
        <Icon name="x" :size="18" />
      </button>
    </div>
    <nav>
      <div class="nav-group" v-for="g in groups" :key="g.label">
        <div class="nav-group-label">{{ g.label }}</div>
        <router-link v-for="it in g.items" :key="it.to" v-show="!it.hidden" :to="it.to" class="nav-item"
          :class="{ active: isActive(it.to) }">
          <span class="nav-ico"><Icon :name="it.ico" :size="18" /></span>
          <span class="txt">{{ it.label }}</span>
          <span v-if="it.badge" class="nav-badge">{{ it.badge }}</span>
        </router-link>
      </div>
    </nav>
    <div class="sidebar-foot">
      <span class="mini-av">{{ state.user?.initials }}</span>
      <div class="who">
        <div class="nm">{{ state.user?.name }}</div>
        <div class="rl">{{ roleLabel }}</div>
      </div>
    </div>
  </aside>
</template>

<script setup>
import { computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import Logo from './Logo.vue'
import Icon from './Icon.vue'
import { state, hasPerm } from '../store'
import { ROLE_LABELS } from '../format'

const route = useRoute()
// Tutup sidebar mobile setiap kali pindah halaman
watch(() => route.path, () => { state.sidebarMobileOpen = false })
const isActive = (p) => route.path === p || (p !== '/' && route.path.startsWith(p))
const canSeeEngine = computed(() => ['SUPER_ADMIN', 'IT_SIMRS', 'MANAGEMENT', 'AUDITOR'].includes(state.user?.role))
const canConfigure = computed(() => hasPerm('CONFIGURE'))
const roleLabel = computed(() => ROLE_LABELS[state.user?.role] || state.user?.role)

// Badge hitung dari data boot (tanpa API tambahan)
const queueCount = computed(() => {
  const wi = state.boot?.workItems || []
  const uid = state.user?.id
  const mine = wi.filter(w => ['New', 'Assigned', 'In Progress'].includes(w.status) && (!uid || !w.assignee || w.assignee === uid || w.role === state.user?.role))
  return mine.length || wi.filter(w => ['New', 'Assigned', 'In Progress'].includes(w.status)).length
})
const claimCount = computed(() => {
  const cl = state.boot?.claims || []
  const hid = state.user?.hospitalId
  return cl.filter(c => (!hid || c.hospitalId === hid) && !['paid', 'written_off'].includes(c.status)).length
})

const groups = computed(() => [
  {
    label: 'Utama',
    items: [
      { to: '/', label: 'Dashboard', ico: 'dashboard' },
      { to: '/workqueue', label: 'Antrean Kerja', ico: 'queue', badge: queueCount.value || null },
    ],
  },
  {
    label: 'Klaim',
    items: [
      { to: '/claims', label: 'Daftar Klaim', ico: 'claims', badge: claimCount.value || null },
      { to: '/scrubber', label: 'Validasi (Scrubber)', ico: 'shieldCheck' },
      { to: '/batches', label: 'Batch & Pengajuan', ico: 'layers' },
    ],
  },
  {
    label: 'Uang',
    items: [
      { to: '/payments', label: 'Pembayaran', ico: 'wallet' },
      { to: '/aging', label: 'Piutang (AR Aging)', ico: 'clock' },
    ],
  },
  {
    label: 'Wawasan',
    items: [
      { to: '/analytics', label: 'Analitik', ico: 'chart' },
      { to: '/kapj', label: 'Simulasi KAPJ', ico: 'calc' },
      { to: '/bi', label: 'BI & Laporan', ico: 'pie' },
    ],
  },
  {
    label: 'Sistem',
    items: [
      { to: '/engineapi', label: 'Engine API', ico: 'code', hidden: !canSeeEngine.value },
      { to: '/audit', label: 'Audit Trail', ico: 'fingerprint' },
      { to: '/compliance', label: 'Kepatuhan Regulasi', ico: 'scale' },
      { to: '/settings', label: 'Pengaturan', ico: 'settings', hidden: !canConfigure.value },
    ],
  },
])
</script>
