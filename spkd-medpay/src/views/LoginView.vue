<template>
  <div class="login-wrap">
    <div class="login-hero">
      <Logo light :fs="26" :bx="46" sub="BY SPKD" />
      <h1>Medical Payment Intelligence</h1>
      <p>Platform inteligensi pembayaran &amp; revenue cycle rumah sakit. Klaim bagian tagihan yang tidak ditanggung BPJS Kesehatan kepada penjamin kedua, tangkap kesalahan klaim sebelum ditolak, lacak setiap rupiah yang masuk, dan lihat piutang mana yang berisiko.</p>
      <div class="lp-badges">
        <span class="lp-badge">✓ Resolusi tarif INA-CBG</span>
        <span class="lp-badge">✓ Hitung selisih otomatis</span>
        <span class="lp-badge">✓ Pantau sampai dibayar</span>
      </div>
      <div class="tag">“Dari selisih tarif sampai uang masuk.”</div>
      <p style="font-size:12.5px;margin-top:10px;opacity:0.78">Klaim lebih bersih sebelum dikirim, uang lebih cepat terlacak setelah dikirim.</p>
    </div>
    <div class="login-form-side">
      <div class="login-card card">
        <div class="card-pad">
          <h2 style="font-size:19px;margin-bottom:4px">Masuk ke MedPay</h2>
          <p style="font-size:12.5px;color:var(--text-secondary);margin:0 0 18px">Gunakan akun korporat SPKD Anda.</p>
          <template v-if="!mfaStep">
            <div class="field">
              <label>Email</label>
              <input class="input" v-model="email" type="email" placeholder="nama@rumahsakit.id" @keyup.enter="doLogin" />
            </div>
            <div class="field">
              <label>Kata Sandi</label>
              <input class="input" v-model="password" type="password" placeholder="••••••••••••" @keyup.enter="doLogin" />
            </div>
            <button class="btn btn-grad" style="width:100%;justify-content:center" :disabled="busy" @click="doLogin">
              <span v-if="busy" class="spin" style="border-top-color:#fff" /> Masuk
            </button>
          </template>
          <template v-else>
            <div class="alert info"><span>🔒</span><span>MFA (TOTP) wajib untuk peran Anda. Masukkan 6 digit kode dari aplikasi autentikator. <b>Kode demo: 246810</b></span></div>
            <div class="field">
              <label>Kode MFA</label>
              <input class="input" v-model="mfaCode" inputmode="numeric" maxlength="6" style="letter-spacing:8px;font-size:20px;text-align:center" @keyup.enter="doVerify" />
            </div>
            <button class="btn btn-grad" style="width:100%;justify-content:center" :disabled="busy" @click="doVerify">Verifikasi &amp; Masuk</button>
          </template>
          <p v-if="error" class="err-text">{{ error }}</p>
          <div class="demo-accounts">
            <b>Akun demo</b> — klik untuk mengisi otomatis<br />
            <span style="color:var(--text-muted)">Kata sandi semua akun: <code>medpay2026</code> · kode MFA: <code>246810</code></span>
            <div class="demo-chips">
              <button v-for="a in demoAccounts" :key="a.email" type="button" class="demo-chip" @click="fillDemo(a.email)">
                <Icon name="users" :size="11" /> {{ a.label }} <span class="dc-role">{{ a.email.split('@')[0] }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { state, toast, loadMe, loadBoot } from '../store'
import Logo from '../components/Logo.vue'
import Icon from '../components/Icon.vue'

const router = useRouter()
const email = ref('')
const password = ref('')
const mfaCode = ref('')
const mfaStep = ref(false)
const busy = ref(false)
const error = ref('')

const demoAccounts = [
  { label: 'Super Admin', email: 'superadmin@spkd.co.id' },
  { label: 'Admin RS', email: 'admin@nusantara.id' },
  { label: 'Billing', email: 'billing@nusantara.id' },
  { label: 'Coder', email: 'coder@nusantara.id' },
  { label: 'Reviewer', email: 'reviewer@nusantara.id' },
  { label: 'Finance', email: 'finance@nusantara.id' },
  { label: 'Auditor', email: 'auditor@nusantara.id' },
  { label: 'IT SIMRS', email: 'it@nusantara.id' },
  { label: 'Manajemen', email: 'manajemen@nusantara.id' },
  { label: 'Viewer', email: 'viewer@nusantara.id' },
]
function fillDemo(mail) {
  email.value = mail
  password.value = 'medpay2026'
  mfaStep.value = false
  error.value = ''
}

async function doLogin() {
  error.value = ''
  if (!email.value || !password.value) { error.value = 'Email dan kata sandi wajib diisi.'; return }
  busy.value = true
  try {
    const res = await api('auth/login', { method: 'POST', body: { email: email.value, password: password.value } })
    if (res.mfaRequired) { mfaStep.value = true; return }
    await finishLogin(res.token)
  } catch (e) {
    error.value = e.message
  } finally { busy.value = false }
}

async function doVerify() {
  error.value = ''
  busy.value = true
  try {
    const res = await api('auth/verify', { method: 'POST', body: { email: email.value, code: mfaCode.value } })
    await finishLogin(res.token)
  } catch (e) {
    error.value = e.message
  } finally { busy.value = false }
}

async function finishLogin(tok) {
  localStorage.setItem('spkd_token', tok)
  const ok = await loadMe()
  if (!ok) { error.value = 'Sesi gagal dimuat.'; return }
  await loadBoot(true)
  toast('Selamat datang, ' + state.user.name + '!', 'success')
  router.push('/')
}
</script>
