<template>
  <main class="page">
    <PageHeader icon="queue" title="Antrean Kerja Hari Ini" sub="Closed-loop: masalah terdeteksi (scrubber/deadline/penolakan/rekonsiliasi/aging) → work item → ditugaskan → terselesaikan → metrik membaik.">
      <template #actions>
        <button class="btn btn-primary" v-if="hasPerm('CREATE')" @click="showNew = true"><Icon name="plus" :size="14" /> Tindakan Baru</button>
      </template>
    </PageHeader>

    <div class="filterbar">
      <select class="input" v-model="fStatus" @change="load">
        <option value="">Semua Status</option>
        <option v-for="s in statuses" :key="s.code" :value="s.code">{{ s.label }}</option>
      </select>
      <select class="input" v-model="fSource" @change="load">
        <option value="">Semua Sumber</option>
        <option v-for="s in sources" :key="s.code" :value="s.code">{{ s.label }}</option>
      </select>
    </div>

    <div class="grid-31">
      <div class="tbl-wrap">
        <table class="tbl">
          <thead><tr><th>Item</th><th>Sumber</th><th>Klaim</th><th>Ditugaskan</th><th>Prioritas</th><th>Status</th><th>Tenggat</th></tr></thead>
          <tbody>
            <tr v-for="w in items" :key="w.workItem.id" @click="select(w.workItem)" style="cursor:pointer" :style="sel && sel.id === w.workItem.id ? 'background:var(--teal-bg)' : ''">
              <td style="max-width:300px">{{ w.workItem.title }}</td>
              <td><span class="badge navy">{{ refLabel('work_item_source', w.workItem.source) }}</span></td>
              <td class="mono" style="font-size:11.5px">{{ w.workItem.claimNo || w.workItem.claimId || '—' }}</td>
              <td style="font-size:12px">{{ assigneeName(w.workItem.assignee) || '— belum —' }}</td>
              <td><span class="badge" :class="refTone('work_item_priority', w.workItem.priority)">{{ refLabel('work_item_priority', w.workItem.priority) }}</span></td>
              <td><span class="badge" :class="refTone('work_item_status', w.workItem.status, 'teal')">{{ refLabel('work_item_status', w.workItem.status) }}</span></td>
              <td style="font-size:12px">{{ formatDate(w.workItem.dueDate) }}</td>
            </tr>
            <tr v-if="!items.length"><td colspan="7"><Empty icon="🎉" title="Antrean kosong" desc="Tidak ada tindakan dengan filter ini." /></td></tr>
          </tbody>
        </table>
      </div>

      <div class="card card-pad" v-if="sel">
        <div style="display:flex;justify-content:space-between;align-items:flex-start;gap:8px">
          <div class="card-title">{{ sel.title }}</div>
          <button class="x-btn" @click="closeDetail" aria-label="Tutup panel detail">✕</button>
        </div>
        <div class="card-sub">{{ docNo(sel, 'itemNo') }} · {{ refLabel('work_item_source', sel.source) }} · dibuat {{ formatDateTime(sel.createdAt) }}</div>
        <div class="field">
          <label>Status</label>
          <select class="input" :value="sel.status" :disabled="!hasPerm('EDIT')" @change="update({ status: $event.target.value })">
            <option v-for="s in statuses" :key="s.code" :value="s.code">{{ s.label }}</option>
          </select>
        </div>
        <div class="field">
          <label>Ditugaskan ke</label>
          <select class="input" :value="sel.assignee || ''" :disabled="!hasPerm('ASSIGN')" @change="update({ assignee: $event.target.value || null })">
            <option value="">— pilih —</option>
            <option v-for="u in state.boot?.users || []" :key="u.id" :value="u.id">{{ u.name }} ({{ ROLE_LABELS[u.role] }})</option>
          </select>
        </div>
        <div class="field">
          <label>Prioritas</label>
          <select class="input" :value="sel.priority" :disabled="!hasPerm('EDIT')" @change="update({ priority: $event.target.value })">
            <option v-for="p in priorities" :key="p.code" :value="p.code">{{ p.label }}</option>
          </select>
        </div>
        <div class="field">
          <label>Tenggat</label>
          <input class="input" type="date" :value="sel.dueDate" :disabled="!hasPerm('EDIT')" @change="update({ dueDate: $event.target.value })" />
        </div>
        <div class="card-title" style="font-size:13px;margin-top:8px">Komentar</div>
        <div style="max-height:150px;overflow-y:auto;margin-bottom:8px">
          <div v-for="c in sel.comments || []" :key="c.id" style="font-size:12.5px;padding:6px 0;border-bottom:1px dashed var(--border)">
            <b>{{ c.author }}</b> · <span style="color:var(--text-muted)">{{ formatDateTime(c.at) }}</span><br />{{ c.text }}
          </div>
          <div v-if="!(sel.comments || []).length" style="color:var(--text-muted);font-size:12.5px">Belum ada komentar.</div>
        </div>
        <div style="display:flex;gap:8px">
          <input class="input" v-model="comment" placeholder="Tulis komentar…" @keyup.enter="addComment" />
          <button class="btn btn-outline btn-sm" :disabled="!comment.trim()" @click="addComment">Kirim</button>
        </div>
        <div class="card-title" style="font-size:13px;margin-top:14px">Riwayat</div>
        <div style="max-height:140px;overflow-y:auto">
          <div v-for="h in [...(sel.history || [])].reverse()" :key="h.id" style="font-size:12px;padding:5px 0;border-bottom:1px dashed var(--border)">
            <span style="color:var(--text-muted)">{{ formatDateTime(h.at) }}</span> — <b>{{ h.actor }}</b>: {{ h.action }}
          </div>
        </div>
        <router-link v-if="sel.claimId" :to="'/claims?open=' + sel.claimId + '&q=' + encodeURIComponent(sel.claimNo || '')" class="btn btn-outline btn-sm" style="margin-top:12px">Buka Klaim Terkait</router-link>
      </div>
      <div class="card card-pad" v-else><Empty icon="←" title="Pilih item" desc="Klik baris untuk melihat detail, menugaskan, dan berkomentar." /></div>
    </div>

    <Modal v-if="showNew" title="Tindakan Baru" @close="showNew = false">
      <div class="field"><label>Judul <span class="req">*</span></label><input class="input" v-model="newItem.title" /></div>
      <div class="form-grid">
        <div class="field">
          <label>Sumber</label>
          <select class="input" v-model="newItem.source"><option v-for="s in sources" :key="s.code" :value="s.code">{{ s.label }}</option></select>
        </div>
        <div class="field">
          <label>Prioritas</label>
          <select class="input" v-model="newItem.priority"><option v-for="p in priorities" :key="p.code" :value="p.code">{{ p.label }}</option></select>
        </div>
        <div class="field">
          <label>Klaim terkait (opsional)</label>
          <select class="input" v-model="newItem.claimId">
            <option value="">— tanpa klaim —</option>
            <option v-for="c in state.boot?.claims || []" :key="c.id" :value="c.id">{{ docNo(c) }} — {{ c.patientName }}</option>
          </select>
        </div>
        <div class="field"><label>Tenggat</label><input class="input" type="date" v-model="newItem.dueDate" /></div>
      </div>
      <template #footer>
        <button class="btn btn-outline" @click="showNew = false">Batal</button>
        <button class="btn btn-teal" :disabled="!newItem.title.trim()" @click="create">Buat</button>
      </template>
    </Modal>
  </main>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
import { state, hasPerm, toast, loadBoot, refreshBoot } from '../store'
import Modal from '../components/Modal.vue'
import Empty from '../components/Empty.vue'
import PageHeader from '../components/PageHeader.vue'
import Icon from '../components/Icon.vue'
import { formatDate, formatDateTime, addDays, ROLE_LABELS, docNo, refOptions, refLabel, refTone } from '../format'

const route = useRoute()
const router = useRouter()
const statuses = computed(() => refOptions('work_item_status'))
const priorities = computed(() => refOptions('work_item_priority'))
const sources = computed(() => refOptions('work_item_source'))
const items = ref([])
const sel = ref(null)
const fStatus = ref('')
const fSource = ref('')
const comment = ref('')
const showNew = ref(false)
const newItem = ref({ title: '', source: 'SCRUBBER', priority: 'Sedang', claimId: '', dueDate: addDays(new Date().toISOString().slice(0, 10), 3) })

// Data lama bisa berisi comments/history null; pastikan selalu array agar panel tidak crash.
function normalize(w) { return { ...w, comments: w.comments || [], history: w.history || [] } }

async function load() {
  const params = new URLSearchParams()
  if (fStatus.value) params.set('status', fStatus.value)
  if (fSource.value) params.set('source', fSource.value)
  const res = await api('work-items?' + params.toString())
  items.value = res.map(r => ({ ...r, workItem: normalize(r.workItem) }))
  if (sel.value) {
    const found = items.value.find(x => x.workItem.id === sel.value.id)
    sel.value = found ? found.workItem : null
  }
  refreshBoot() // badge Antrean Kerja di sidebar ikut ter-update
}

// Detail tampil sebagai panel di kanan (bukan halaman baru). ID item disimpan di query
// dengan replace agar tombol Back tidak menumpuk riwayat & tetap kembali dari menu Antrean Kerja.
function select(w) {
  sel.value = w
  router.replace({ path: '/workqueue', query: { item: w.id } })
}
function closeDetail() {
  sel.value = null
  router.replace({ path: '/workqueue' })
}

function assigneeName(id) {
  if (!id) return ''
  const u = state.boot?.users.find(x => x.id === id)
  return u ? u.name : id
}

async function update(patch) {
  try {
    sel.value = normalize(await api('work-items/' + sel.value.id, { method: 'PUT', body: patch }))
    toast('Work item diperbarui.', 'success')
    await load()
  } catch (e) { toast(e.message, 'error') }
}

async function addComment() {
  try {
    sel.value = normalize(await api('work-items/' + sel.value.id + '/comments', { method: 'POST', body: { text: comment.value } }))
    comment.value = ''
    await load()
  } catch (e) { toast(e.message, 'error') }
}

async function create() {
  try {
    await api('work-items', { method: 'POST', body: { ...newItem.value, claimId: newItem.value.claimId || null } })
    toast('Work item dibuat.', 'success')
    showNew.value = false
    newItem.value.title = ''
    await load()
  } catch (e) { toast(e.message, 'error') }
}

onMounted(async () => {
  await loadBoot()
  await load()
  const id = route.query.item
  if (id) sel.value = items.value.find(x => x.workItem.id === String(id))?.workItem || null
})
</script>
