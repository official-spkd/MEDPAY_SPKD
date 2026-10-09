import { createRouter, createWebHashHistory } from 'vue-router'
import { getToken } from './api'

const routes = [
  { path: '/login', component: () => import('./views/LoginView.vue'), meta: { public: true, title: 'Masuk' } },
  { path: '/', component: () => import('./views/DashboardView.vue'), meta: { title: 'Dashboard', crumb: 'Beranda' } },
  { path: '/workqueue', component: () => import('./views/WorkQueueView.vue'), meta: { title: 'Antrean Kerja', crumb: 'Utama' } },
  { path: '/claims', component: () => import('./views/ClaimsView.vue'), meta: { title: 'Daftar Klaim', crumb: 'Klaim' } },
  { path: '/claims/new', component: () => import('./views/ClaimFormView.vue'), meta: { title: 'Klaim Baru', crumb: 'Klaim' } },
  { path: '/claims/:id/edit', component: () => import('./views/ClaimFormView.vue'), meta: { title: 'Edit Klaim', crumb: 'Klaim' } },
  { path: '/scrubber', component: () => import('./views/ScrubberView.vue'), meta: { title: 'Validasi Klaim (Scrubber)', crumb: 'Klaim' } },
  { path: '/batches', component: () => import('./views/BatchesView.vue'), meta: { title: 'Batch & Pengajuan', crumb: 'Klaim' } },
  { path: '/payments', component: () => import('./views/PaymentsView.vue'), meta: { title: 'Pembayaran & Rekonsiliasi', crumb: 'Uang' } },
  { path: '/aging', component: () => import('./views/AgingView.vue'), meta: { title: 'Piutang (AR Aging)', crumb: 'Uang' } },
  { path: '/analytics', component: () => import('./views/AnalyticsView.vue'), meta: { title: 'Analitik', crumb: 'Wawasan' } },
  { path: '/kapj', component: () => import('./views/KapjView.vue'), meta: { title: 'Simulasi KAPJ', crumb: 'Wawasan' } },
  { path: '/bi', component: () => import('./views/BiView.vue'), meta: { title: 'BI & Laporan', crumb: 'Wawasan' } },
  { path: '/engineapi', component: () => import('./views/EngineApiView.vue'), meta: { title: 'Engine API', crumb: 'Sistem' } },
  { path: '/audit', component: () => import('./views/AuditView.vue'), meta: { title: 'Audit Trail', crumb: 'Sistem' } },
  { path: '/compliance', component: () => import('./views/ComplianceView.vue'), meta: { title: 'Kepatuhan Regulasi', crumb: 'Sistem' } },
  { path: '/settings', component: () => import('./views/SettingsView.vue'), meta: { title: 'Pengaturan', crumb: 'Sistem' } },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

export const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

router.beforeEach((to) => {
  if (!to.meta.public && !getToken()) return '/login'
  if (to.path === '/login' && getToken()) return '/'
  document.title = (to.meta.title ? to.meta.title + ' — ' : '') + 'SPKD MedPay'
  return true
})
