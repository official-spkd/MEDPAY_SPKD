<template>
  <main class="page">
    <PageHeader icon="settings" title="Pengaturan" sub="RS, pengguna, penjamin (Payer Rule Library), dan Partner Engine API. Setiap perubahan tercatat di audit trail." />

    <div class="tabs">
      <button class="tab" :class="{ active: tab === 'rs' }" @click="tab = 'rs'">Rumah Sakit</button>
      <button class="tab" :class="{ active: tab === 'user' }" @click="tab = 'user'">Pengguna</button>
      <button class="tab" :class="{ active: tab === 'insurer' }" @click="tab = 'insurer'">Penjamin (Payer Library)</button>
      <button class="tab" :class="{ active: tab === 'partner' }" @click="tab = 'partner'" v-if="isSuper">Partner Engine API</button>
    </div>

    <!-- RS -->
    <div v-if="tab === 'rs'">
      <div class="filterbar">
        <button class="btn btn-teal btn-sm" v-if="hasPerm('CONFIGURE')" @click="showNewHosp = true">+ Daftarkan RS</button>
        <span class="hint">Pendaftaran hanya memakai Kode Faskes; nama dicocokkan via lookup e-Klaim (demo deterministik — bila tidak ditemukan, nama wajib diverifikasi manual).</span>
      </div>
      <div class="tbl-wrap">
        <table class="tbl">
          <thead><tr><th>Kode Faskes</th><th>Nama RS</th><th>Kota</th><th>Kelas</th><th>e-Klaim</th><th>Aktif</th><th>Terdaftar</th></tr></thead>
          <tbody>
            <tr v-for="h in state.boot?.hospitals || []" :key="h.id">
              <td class="mono"><b>{{ h.kodeFaskes }}</b></td>
              <td>{{ h.name }} <span v-if="h.name.includes('DIVERIFIKASI')" class="verify-tag">[VERIFIKASI REGULASI]</span></td>
              <td>{{ h.city }}</td>
              <td><span class="badge neutral">Kelas {{ h.class }}</span></td>
              <td><span class="badge" :class="h.eklaimConfigured ? 'success' : 'warning'">{{ h.eklaimConfigured ? 'Terkonfigurasi' : 'Belum' }}</span></td>
              <td><input type="checkbox" :checked="h.active" :disabled="!hasPerm('CONFIGURE')" @change="toggleHosp(h, $event)" /></td>
              <td style="font-size:12px">{{ formatDate(h.registeredAt) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Pengguna -->
    <div v-if="tab === 'user'">
      <div class="filterbar">
        <button class="btn btn-teal btn-sm" v-if="hasPerm('CONFIGURE')" @click="showNewUser = true">+ Pengguna Baru</button>
        <span class="hint">MFA (TOTP) wajib untuk Super Admin, Admin RS, dan Finance. Kata sandi default pengguna baru (demo): <code>medpay2026</code>.</span>
      </div>
      <div class="tbl-wrap">
        <table class="tbl" style="min-width:640px">
          <thead><tr><th>Nama</th><th>Email</th><th>Peran</th><th>RS</th><th>MFA</th></tr></thead>
          <tbody>
            <tr v-for="u in users" :key="u.id">
              <td><b>{{ u.name }}</b> <span class="badge neutral">{{ u.initials }}</span></td>
              <td class="mono" style="font-size:12px">{{ u.email }}</td>
              <td>
                <select v-if="canEditUser(u)" class="input" style="min-height:30px;font-size:12px" :value="u.role" @change="updateUser(u, { role: $event.target.value })">
                  <option v-for="(lbl, k) in ROLE_LABELS" :key="k" :value="k">{{ lbl }}</option>
                </select>
                <span v-else class="badge navy">{{ ROLE_LABELS[u.role] }}</span>
              </td>
              <td style="font-size:12px">{{ u.hospitalId ? hospName(u.hospitalId) : 'Semua RS' }}</td>
              <td><span class="badge" :class="u.mfaEnabled ? 'success' : 'neutral'">{{ u.mfaEnabled ? 'Aktif' : 'Non-aktif' }}</span></td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Penjamin -->
    <div v-if="tab === 'insurer'">
      <div class="alert info"><span>ℹ</span><span>Library Payer Rule: plafon, perilaku co-pay, pengecualian, aturan selisih kelas, batas pengajuan, istilah pembayaran rata-rata. Entri membawa sumber, tanggal efektif, versi, dan verifikator. Entri belum terverifikasi ditandai jelas.</span></div>
      <div class="tbl-wrap">
        <table class="tbl" style="min-width:760px">
          <thead><tr><th>Penjamin (daftar resmi e-Klaim)</th><th>Jenis</th><th class="num">Rerata Bayar</th><th class="num">Batas Pengajuan</th><th>Kebijakan Plafon</th><th>Verifikasi</th></tr></thead>
          <tbody>
            <tr v-for="i in state.boot?.insurers || []" :key="i.id">
              <td><b>{{ i.name }}</b></td>
              <td><span class="badge neutral">{{ i.type }}</span></td>
              <td class="num">{{ i.avgPaymentDays }} hari</td>
              <td class="num">
                <input v-if="hasPerm('CONFIGURE')" class="input" style="min-height:30px;width:86px;text-align:right" type="number" :value="i.deadlineDays" @change="updateInsurer(i, { deadlineDays: parseInt($event.target.value || '0', 10) })" />
                <span v-else>{{ i.deadlineDays }} hari</span>
              </td>
              <td style="font-size:12px">{{ i.ceilingPolicy }}</td>
              <td><span class="badge" :class="i.verified ? 'success' : 'warning'">{{ i.verified ? 'Terverifikasi' : 'Perlu verifikasi' }}</span></td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Partner -->
    <div v-if="tab === 'partner' && isSuper">
      <div class="grid-2">
        <div class="card card-pad" v-for="p in state.boot?.partners || []" :key="p.id">
          <div style="display:flex;justify-content:space-between;align-items:flex-start">
            <div>
              <div class="card-title">{{ p.company }}</div>
              <div class="card-sub" style="margin-bottom:8px">{{ p.contact }} · sejak {{ formatDate(p.createdAt) }}</div>
            </div>
            <span class="badge" :class="p.active ? 'success' : 'neutral'">{{ p.active ? 'Aktif' : 'Non-aktif' }}</span>
          </div>
          <div class="kv">
            <span class="k">Consumer ID</span><span class="v mono">{{ p.consumerId }}</span>
            <span class="k">Consumer Secret</span><span class="v mono">{{ p.consumerSecretMasked }} <span class="hint">(terenkripsi, tampil tersamar)</span></span>
            <span class="k">User Key</span><span class="v mono">{{ p.userKeyMasked }}</span>
            <span class="k">Rate Limit</span><span class="v">{{ p.rateLimitPerMin }}/menit</span>
            <span class="k">IP Whitelist</span><span class="v mono" style="font-size:11.5px">{{ p.whitelistIps.join(', ') }}</span>
            <span class="k">RS Terikat</span><span class="v">{{ p.hospitalIds.map(hospName).join(', ') }}</span>
          </div>
          <div class="chips-row" style="margin-top:12px">
            <button class="btn btn-outline btn-xs" @click="togglePartner(p)">{{ p.active ? 'Non-aktifkan' : 'Aktifkan' }}</button>
            <button class="btn btn-outline btn-xs" @click="editPartner = { ...p, whitelistStr: p.whitelistIps.join(', ') }">Edit Batas & IP</button>
          </div>
          <div class="note-strip">Kontrol tiga lapis: whitelist IP · pencatatan akses penuh · user key terikat RS tertentu. Kredensial dienkripsi; rotasi didukung; kredensial tidak berubah saat edit profil.</div>
        </div>
        <Empty v-if="!(state.boot?.partners || []).length" icon="🔌" title="Belum ada partner" desc="Super Admin mendaftarkan partner Engine API di sini." />
      </div>

      <Modal v-if="editPartner" title="Edit Partner (kredensial tidak berubah)" @close="editPartner = null">
        <div class="field"><label>Rate limit (per menit)</label><input class="input" type="number" v-model.number="editPartner.rateLimitPerMin" /></div>
        <div class="field"><label>IP whitelist (pisahkan koma)</label><input class="input" v-model="editPartner.whitelistStr" style="font-family:ui-monospace,monospace" /></div>
        <template #footer>
          <button class="btn btn-outline" @click="editPartner = null">Batal</button>
          <button class="btn btn-teal" @click="savePartner">Simpan</button>
        </template>
      </Modal>
    </div>

    <Modal v-if="showNewHosp" title="Daftarkan RS (Kode Faskes)" @close="showNewHosp = false">
      <div class="field">
        <label>Kode Faskes <span class="req">*</span></label>
        <input class="input" v-model="newHosp.kodeFaskes" placeholder="0304R001" style="font-family:ui-monospace,monospace" />
        <div class="hint">Nama RS dicocokkan otomatis via lookup e-Klaim. Jika tidak ditemukan, entri ditandai untuk diverifikasi.</div>
      </div>
      <div class="form-grid">
        <div class="field"><label>Kota</label><input class="input" v-model="newHosp.city" /></div>
        <div class="field"><label>Kelas</label><select class="input" v-model="newHosp.class"><option>A</option><option>B</option><option>C</option><option>D</option></select></div>
      </div>
      <template #footer>
        <button class="btn btn-outline" @click="showNewHosp = false">Batal</button>
        <button class="btn btn-teal" :disabled="!newHosp.kodeFaskes.trim()" @click="createHosp">Daftarkan</button>
      </template>
    </Modal>

    <Modal v-if="showNewUser" title="Pengguna Baru" @close="showNewUser = false">
      <div class="form-grid">
        <div class="field"><label>Nama <span class="req">*</span></label><input class="input" v-model="newUser.name" /></div>
        <div class="field"><label>Email <span class="req">*</span></label><input class="input" v-model="newUser.email" type="email" /></div>
        <div class="field">
          <label>Peran <span class="req">*</span></label>
          <select class="input" v-model="newUser.role">
            <option v-for="(lbl, k) in ROLE_LABELS" :key="k" :value="k">{{ lbl }}</option>
          </select>
        </div>
        <div class="field" v-if="isSuper">
          <label>RS</label>
          <select class="input" v-model="newUser.hospitalId">
            <option value="">Semua RS (Super Admin saja)</option>
            <option v-for="h in state.boot?.hospitals || []" :key="h.id" :value="h.id">{{ h.name }}</option>
          </select>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-outline" @click="showNewUser = false">Batal</button>
        <button class="btn btn-teal" :disabled="!newUser.name || !newUser.email || !newUser.role" @click="createUser">Buat Pengguna</button>
      </template>
    </Modal>
  </main>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api'
import { state, hasPerm, toast, loadBoot } from '../store'
import Modal from '../components/Modal.vue'
import PageHeader from '../components/PageHeader.vue'
import { formatDate, ROLE_LABELS } from '../format'

const tab = ref('rs')
const showNewHosp = ref(false)
const showNewUser = ref(false)
const editPartner = ref(null)
const newHosp = ref({ kodeFaskes: '', city: '', class: 'B' })
const newUser = ref({ name: '', email: '', role: 'BILLING_STAFF', hospitalId: '' })
const isSuper = computed(() => state.user?.role === 'SUPER_ADMIN')
const users = computed(() => (state.boot?.users || []).filter(u => !state.user?.hospitalId || u.hospitalId === state.user?.hospitalId))

function hospName(id) { return state.boot?.hospitals.find(h => h.id === id)?.name || id }
function canEditUser(u) { return hasPerm('CONFIGURE') && u.role !== 'SUPER_ADMIN' }

async function createHosp() {
  try {
    const h = await api('hospitals', { method: 'POST', body: { kodeFaskes: newHosp.value.kodeFaskes, city: newHosp.value.city, class: newHosp.value.class } })
    toast('RS terdaftar: ' + h.name, 'success')
    showNewHosp.value = false
    await loadBoot(true)
  } catch (e) { toast(e.message, 'error') }
}

async function toggleHosp(h, ev) {
  try {
    await api('hospitals/' + h.id, { method: 'PUT', body: { ...h, active: ev.target.checked } })
    toast('Status RS diperbarui.', 'success')
    await loadBoot(true)
  } catch (e) { toast(e.message, 'error'); ev.target.checked = !ev.target.checked }
}

async function createUser() {
  try {
    await api('users', { method: 'POST', body: { ...newUser.value } })
    toast('Pengguna dibuat (sandi default demo: medpay2026).', 'success')
    showNewUser.value = false
    newUser.value = { name: '', email: '', role: 'BILLING_STAFF', hospitalId: '' }
    await loadBoot(true)
  } catch (e) { toast(e.message, 'error') }
}

async function updateUser(u, patch) {
  try {
    await api('users/' + u.id, { method: 'PUT', body: { ...u, ...patch } })
    toast('Pengguna diperbarui.', 'success')
    await loadBoot(true)
  } catch (e) { toast(e.message, 'error') }
}

async function updateInsurer(i, patch) {
  try {
    await api('insurers/' + i.id, { method: 'PUT', body: { ...i, ...patch } })
    toast('Aturan penjamin diperbarui (tercatat di audit).', 'success')
    await loadBoot(true)
  } catch (e) { toast(e.message, 'error') }
}

async function togglePartner(p) {
  try {
    await api('partners/' + p.id, { method: 'PUT', body: { ...p, active: !p.active } })
    toast('Partner ' + (p.active ? 'dinonaktifkan.' : 'diaktifkan.'), 'success')
    await loadBoot(true)
  } catch (e) { toast(e.message, 'error') }
}

async function savePartner() {
  try {
    await api('partners/' + editPartner.value.id, {
      method: 'PUT',
      body: { ...editPartner.value, whitelistIps: editPartner.value.whitelistStr.split(',').map(s => s.trim()).filter(Boolean) },
    })
    toast('Partner diperbarui (kredensial tidak berubah).', 'success')
    editPartner.value = null
    await loadBoot(true)
  } catch (e) { toast(e.message, 'error') }
}

onMounted(() => loadBoot())
</script>
