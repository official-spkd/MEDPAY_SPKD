// ── SPKD MedPay — Global state reaktif (tanpa library eksternal) ────────────
import { reactive } from 'vue'
import { api, getToken, setToken } from './api'

export const state = reactive({
  user: null,
  permissions: [],
  boot: null,        // bootstrap: hospitals, insurers, users, rules, dst.
  booted: false,
  theme: localStorage.getItem('spkd_theme') || 'light',
  sidebarCollapsed: localStorage.getItem('spkd_sb') === '1',
  sidebarMobileOpen: false,
  toasts: [],
  notifOpen: false,
  userOpen: false,
})

export function hasPerm(p) { return state.permissions.includes(p) }

export function toast(msg, type = 'info', ms = 4200) {
  const id = Date.now() + Math.random()
  state.toasts.push({ id, msg, type })
  setTimeout(() => { const i = state.toasts.findIndex(t => t.id === id); if (i >= 0) state.toasts.splice(i, 1) }, ms)
}

export async function loadMe() {
  if (!getToken()) return false
  try {
    const me = await api('me')
    state.user = me.user
    state.permissions = me.permissions || []
    return true
  } catch {
    return false
  }
}

export async function loadBoot(force = false) {
  if (state.booted && !force) return
  state.boot = await api('bootstrap')
  state.booted = true
}

export function logout() {
  api('auth/logout', { method: 'POST' }).catch(() => {})
  setToken('')
  state.user = null
  state.permissions = []
  state.booted = false
  state.boot = null
  window.location.hash = '#/login'
}

export function toggleTheme() {
  state.theme = state.theme === 'light' ? 'dark' : 'light'
  localStorage.setItem('spkd_theme', state.theme)
  document.documentElement.setAttribute('data-theme', state.theme)
}

export function toggleSidebar() {
  if (window.innerWidth <= 900) {
    state.sidebarMobileOpen = !state.sidebarMobileOpen
  } else {
    state.sidebarCollapsed = !state.sidebarCollapsed
    localStorage.setItem('spkd_sb', state.sidebarCollapsed ? '1' : '0')
  }
}

export function initTheme() {
  document.documentElement.setAttribute('data-theme', state.theme)
}

export function myHospital() {
  if (!state.boot || !state.user) return null
  if (!state.user.hospitalId) return null
  return state.boot.hospitals.find(h => h.id === state.user.hospitalId) || null
}
