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
        <option v-for="s in statuses" :key="s">{{ s }}</option>
      </select>
      <select class="input" v-model="fSource" @change="load">
        <option value="">Semua Sumber</option>
        <option>SCRUBBER</option><option>DEADLINE</option><option>REJECTION</option><option>RECONCILIATION</option><option>AGING</option>
      </select>
    </div>

    <div class="grid-31">
      <div class="tbl-wrap">
        <table class="tbl">
          <thead><tr><th>Item</th><th>Sumber</th><th>Klaim</th><th>Ditugaskan</th><th>Prioritas</th><th>Status</th><th>Tenggat</th></tr></thead>
          <tbody>
            <tr v-for="w in items" :key="w.workItem.id" @click="sel = w.workItem" :style="sel && sel.id === w.workItem.id ? 'background:var(--teal-bg)' : ''">
              <td style="max-width:300px">{{ w.workItem.title }}</td>
              <td><span class="badge navy">{{ w.workItem.source }}</span></td>
              <td class="mono" style="font-size:11.5px">{{ w.workItem.claimId || '—' }}</td>
              <td style="font-size:12px">{{ assigneeName(w.workItem.assignee) || '— belum —' }}</td>
              <td><span class="badge" :class="w.workItem.priority === 'Kritis' ? 'danger' : w.workItem.priority === 'Tinggi' ? 'warning' : 'neutral'">{{ w.workItem.priority }}</span></td>
              <td><span class="badge" :class="w.workItem.status === 'Resolved' || w.workItem.status === 'Closed' ? 'success' : w.workItem.status === 'Reopened' ? 'danger' : 'teal'">{{ w.workItem.status }}</span></td>
              <td style="font-size:12px">{{ formatDate(w.workItem.dueDate) }}</td>
            </tr>
            <tr v-if="!items.length"><td colspan="7"><Empty icon="🎉" title="Antrean kosong" desc="Tidak ada tindakan dengan filter ini." /></td></tr>
          </tbody>
        </table>
      </div>

      <div class="card card-pad" v-if="sel">
        <div class="card-title">{{ sel.title }}</div>
        <div class="card-sub">{{ sel.source }} · dibuat {{ formatDateTime(sel.createdAt) }}</div>
        <div class="field">
          <label>Status</label>
          <select class="input" :value="sel.status" :disabled="!hasPerm('EDIT')" @change="update({ status: $event.target.value })">
            <option v-for="s in statuses" :key="s">{{ s }}</option>
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
            <option v-for="p in ['Rendah', 'Sedang', 'Tinggi', 'Kritis']" :key="p">{{ p }}</option>
          </select>
        </div>
        <div class="field">
          <label>Tenggat</label>
          <input class="input" type="date" :value="sel.dueDate" :disabled="!hasPerm('EDIT')" @change="update({ dueDate: $event.target.value })" />
        </div>
        <div class="card-title" style="font-size:13px;margin-top:8px">Komentar</div>
        <div style="max-height:150px;overflow-y:auto;margin-bottom:8px">
          <div v-for="c in sel.comments" :key="c.id" style="font-size:12.5px;padding:6px 0;border-bottom:1px dashed var(--border)">
            <b>{{ c.author }}</b> · <span style="color:var(--text-muted)">{{ formatDateTime(c.at) }}</span><br />{{ c.text }}
          </div>
          <div v-if="!sel.comments.length" style="color:var(--text-muted);font-size:12.5px">Belum ada komentar.</div>
        </div>
        <div style="display:flex;gap:8px">
          <input class="input" v-model="comment" placeholder="Tulis komentar…" @keyup.enter="addComment" />
          <button class="btn btn-outline btn-sm" :disabled="!comment.trim()" @click="addComment">Kirim</button>
        </div>
        <div class="card-title" style="font-size:13px;margin-top:14px">Riwayat</div>
        <div style="max-height:140px;overflow-y:auto">
          <div v-for="h in [...sel.history].reverse()" :key="h.id" style="font-size:12px;padding:5px 0;border-bottom:1px dashed var(--border)">
            <span style="color:var(--text-muted)">{{ formatDateTime(h.at) }}</span> — <b>{{ h.actor }}</b>: {{ h.action }}
          </div>
        </div>
        <router-link v-if="sel.claimId" :to="'/claims?q=' + sel.claimId" class="btn btn-outline btn-sm" style="margin-top:12px">Buka Klaim Terkait</router-link>
      </div>
      <div class="card card-pad" v-else><Empty icon="←" title="Pilih item" desc="Klik baris untuk melihat detail, menugaskan, dan berkomentar." /></div>
    </div>

    <Modal v-if="showNew" title="Tindakan Baru" @close="showNew = false">
      <div class="field"><label>Judul <span class="req">*</span></label><input class="input" v-model="newItem.title" /></div>
      <div class="form-grid">
        <div class="field">
          <label>Sumber</label>
          <select class="input" v-model="newItem.source"><option>SCRUBBER</option><option>DEADLINE</option><option>REJECTION</option><option>RECONCILIATION</option><option>AGING</option></select>
        </div>
        <div class="field">
          <label>Prioritas</label>
          <select class="input" v-model="newItem.priority"><option v-for="p in ['Rendah', 'Sedang', 'Tinggi', 'Kritis']" :key="p">{{ p }}</option></select>
        </div>
        <div class="field"><label>Klaim terkait (opsional)</label><input class="input" v-model="newItem.claimId" placeholder="CLM-0001" /></div>
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
import { ref, onMounted } from 'vue'
import { api } from '../api'
import { state, hasPerm, toast, loadBoot } from '../store'
import Modal from '../components/Modal.vue'
import Empty from '../components/Empty.vue'
import PageHeader from '../components/PageHeader.vue'
import Icon from '../components/Icon.vue'
import { formatDate, formatDateTime, addDays, ROLE_LABELS } from '../format'

const statuses = ['New', 'Assigned', 'In Progress', 'Waiting Verification', 'Resolved', 'Reopened', 'Closed']
const items = ref([])
const sel = ref(null)
const fStatus = ref('')
const fSource = ref('')
const comment = ref('')
const showNew = ref(false)
const newItem = ref({ title: '', source: 'SCRUBBER', priority: 'Sedang', claimId: '', dueDate: addDays(new Date().toISOString().slice(0, 10), 3) })

async function load() {
  const params = new URLSearchParams()
  if (fStatus.value) params.set('status', fStatus.value)
  if (fSource.value) params.set('source', fSource.value)
  items.value = await api('work-items?' + params.toString())
  if (sel.value) {
    const found = items.value.find(x => x.workItem.id === sel.value.id)
    sel.value = found ? found.workItem : null
  }
}

function assigneeName(id) {
  if (!id) return ''
  const u = state.boot?.users.find(x => x.id === id)
  return u ? u.name : id
}

async function update(patch) {
  try {
    sel.value = await api('work-items/' + sel.value.id, { method: 'PUT', body: patch })
    toast('Work item diperbarui.', 'success')
    await load()
  } catch (e) { toast(e.message, 'error') }
}

async function addComment() {
  try {
    sel.value = await api('work-items/' + sel.value.id + '/comments', { method: 'POST', body: { text: comment.value } })
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

onMounted(async () => { await loadBoot(); await load() })
</script>
