// ── SPKD MedPay — Kebijakan akses halaman & fitur per peran ───────────────────
// Satu sumber kebenaran untuk sidebar, pencarian global, tombol aksi, dan guard rute.
// Izin granular (VIEW/CREATE/…) tetap dari backend; aturan di sini melengkapi untuk
// pembatasan yang spesifik per peran (mis. Finance tidak membuat klaim).
import { state, hasPerm } from './store'

const role = () => state.user?.role

export const canSeeEngine = () => ['SUPER_ADMIN', 'IT_SIMRS', 'HOSPITAL_ADMIN'].includes(role())
// Menu UANG (Pembayaran & Piutang) disembunyikan untuk Staf Billing dan Coder.
export const canSeeMoney = () => !['BILLING_STAFF', 'CODER'].includes(role())
// Validasi (Scrubber) dan Batch & Pengajuan disembunyikan untuk Finance.
export const canSeeValidation = () => role() !== 'FINANCE'
export const canCreateClaim = () => hasPerm('CREATE') && role() !== 'FINANCE'
// Coder hanya boleh melihat batch (tanpa membuat/mengubah).
export const canWriteBatch = () => role() !== 'CODER'

// Dipakai guard router: kembalikan false bila peran tidak boleh membuka path tersebut.
export function routeAllowed(path) {
  if (path === '/settings') return hasPerm('CONFIGURE')
  if (path === '/engineapi') return canSeeEngine()
  if (path === '/claims/new') return canCreateClaim()
  if (/^\/claims\/[^/]+\/edit$/.test(path)) return hasPerm('EDIT')
  if (path === '/payments' || path === '/aging') return canSeeMoney()
  if (path === '/scrubber' || path === '/batches') return canSeeValidation()
  return true
}
