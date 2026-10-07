// ── SPKD MedPay — API helper ────────────────────────────────────────────────
// Relative path + XTransformPort injection agar bekerja via gateway maupun langsung.

const PORT_QS = new URLSearchParams(window.location.search).get('XTransformPort')

// Mode mock (default): seluruh panggilan API dilayani in-browser oleh src/mock/server.js
// dengan data demo — cocok untuk deploy statis (Vercel) tanpa backend.
// Set VITE_USE_MOCK=false untuk memakai backend Go (proxy /api → :8080 saat dev).
const USE_MOCK = import.meta.env.VITE_USE_MOCK !== 'false'

let mockModule = null
async function mockRequest(method, path, body) {
  if (!mockModule) mockModule = await import('./mock/server.js')
  return mockModule.mockRequest(method, path, body, getToken())
}

function withPort(path) {
  if (!PORT_QS) return path
  return path + (path.includes('?') ? '&' : '?') + 'XTransformPort=' + PORT_QS
}

export function getToken() { return localStorage.getItem('spkd_token') || '' }
export function setToken(t) { t ? localStorage.setItem('spkd_token', t) : localStorage.removeItem('spkd_token') }

export class ApiError extends Error {
  constructor(message, status, payload) { super(message); this.status = status; this.payload = payload }
}

export async function api(path, { method = 'GET', body } = {}) {
  let ok, status, data
  if (USE_MOCK) {
    const r = await mockRequest(method, path, body)
    status = r.status
    ok = status >= 200 && status < 300
    data = r.data
  } else {
    const headers = { 'Content-Type': 'application/json' }
    const tok = getToken()
    if (tok) headers['Authorization'] = 'Bearer ' + tok
    let res
    try {
      res = await fetch(withPort('api/v1/' + path), {
        method, headers, body: body !== undefined ? JSON.stringify(body) : undefined,
      })
    } catch (e) {
      throw new ApiError('Tidak dapat menghubungi server MedPay. Pastikan layanan berjalan.', 0)
    }
    const text = await res.text()
    data = null
    try { data = text ? JSON.parse(text) : null } catch { data = { raw: text } }
    ok = res.ok
    status = res.status
  }
  if (!ok) {
    if (status === 401 && !path.startsWith('auth/')) {
      setToken('')
      window.location.hash = '#/login'
    }
    const msg = (data && (data.error || data.message)) || 'Permintaan gagal (' + status + ')'
    throw new ApiError(msg, status, data)
  }
  return data
}

export async function downloadCsv(path, filename) {
  let blob
  if (USE_MOCK) {
    const r = await mockRequest('GET', path)
    if (r.status >= 400) throw new ApiError('Ekspor gagal', r.status)
    blob = new Blob([r.csv], { type: 'text/csv;charset=utf-8' })
  } else {
    const headers = { }
    const tok = getToken()
    if (tok) headers['Authorization'] = 'Bearer ' + tok
    const res = await fetch(withPort('api/v1/' + path), { headers })
    if (!res.ok) throw new ApiError('Ekspor gagal', res.status)
    blob = await res.blob()
  }
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = filename; a.click()
  setTimeout(() => URL.revokeObjectURL(url), 2000)
}

export function toCsv(rows) {
  return rows.map(r => r.map(c => {
    const s = c === null || c === undefined ? '' : String(c)
    return /[",;\n]/.test(s) ? '"' + s.replace(/"/g, '""') + '"' : s
  }).join(';')).join('\n')
}

export function downloadBlobCsv(filename, rows) {
  const csv = '\uFEFF' + toCsv(rows)
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = filename; a.click()
  setTimeout(() => URL.revokeObjectURL(url), 2000)
}
