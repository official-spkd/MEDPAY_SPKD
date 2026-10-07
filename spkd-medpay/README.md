# SPKD MedPay — Frontend (Vue 3 + Vite)

> **v2.2 — UI Redesign "MedCredix-style · Green Edition" (2026-10)**
> Sistem UI diperbarui dengan bahasa desain hijau penuh: sidebar **hijau tua (#0B3D2A)** bergradasi dengan
> grup menu & badge, item menu aktif berupa pill gradasi hijau cerah (#1F9A5C) dengan glow, topbar modern
> (pencarian `Ctrl+K`, dark mode, notifikasi, profil user), breadcrumb sub-header, serta kartu hero gradasi
> hijau dengan *Recovery Score* (gauge melingkar), panel *Radar Risiko* dan *Peringatan Dini* di Dashboard.
> Seluruh aksen navy/biru diganti keluarga hijau tua; chart batang kini bergradasi, line chart memakai
> kurva halus (smooth bezier) dengan stroke gradasi, dan donut chart memiliki celah tipis antar segmen.
> Struktur modul, alur kerja, API mock, dan data demo tidak berubah.
>
> **Riwayat:**
> - **v2.2 (Green Edition):** sidebar & seluruh aksen navy → hijau tua; chart dipoles (gradient bar,
>   smooth line, donut gap); pill menu aktif gradasi + glow; meta `theme-color`; KPI radius 16 + aksen gradien.
> - **v2.1:** pencarian global `Ctrl+K` (SearchModal), PageHeader & KPI berikon, chip akun demo di login,
>   perbaikan badge sidebar dari data `bootstrap`.


## Deploy ke Vercel (frontend + data dummy, tanpa backend)

1. Import repo di Vercel.
2. **Root Directory** → `frontend` (Framework otomatis terdeteksi: Vite).
3. Deploy. Build: `npm run build`, output: `dist` (sudah diatur di `vercel.json`).

Tidak perlu environment variable. Semua panggilan API dilayani oleh mock in-browser
(`src/mock/server.js`) dengan data demo (`src/mock/data.json`).

**Akun demo** — kata sandi `medpay2026`, kode MFA `246810`:
`superadmin@spkd.co.id`, `admin@nusantara.id`, `billing@nusantara.id`, `coder@`, `reviewer@`,
`finance@`, `auditor@`, `it@`, `manajemen@`, `viewer@nusantara.id`.

Catatan perilaku mock:
- Perubahan data (klaim, pembayaran, dll.) tersimpan di `sessionStorage` — bertahan saat refresh,
  kembali ke data awal pada tab/sesi baru.
- Seluruh tanggal data demo digeser relatif ke "hari ini", sehingga deadline, aging, dan KPI tetap bermakna.

## Pakai backend Go

```bash
cp .env.example .env.local   # lalu set VITE_USE_MOCK=false
npm install && npm run dev   # proxy /api → http://localhost:8080
```
