<template>
  <main class="page">
    <PageHeader icon="fingerprint" title="Audit Trail" sub="Log append-only: siapa, apa, kapan, di mana, sumber (WEB/ENGINE_API), sebelum/sesudah. Baris audit tidak dapat diubah atau dihapus dari aplikasi.">
      <template #actions>
        <button class="btn btn-outline btn-sm" @click="exportCsv">Unduh CSV</button>
      </template>
    </PageHeader>

    <div class="kpi-grid">
      <Kpi icon="fingerprint" label="Total Entri (halaman ini)" :value="String(sum.total || 0)" tone="navy" source="Log aksi pengguna (WEB) + akses Engine API partner." />
      <Kpi icon="x" label="Ditolak (≥ 400)" :value="String(sum.rejected || 0)" tone="danger" :val-class="sum.rejected ? 'danger' : ''"
        source="Panggilan API ditolak: 401 kredensial salah, 403 IP di luar whitelist, 405 method, 429 rate limit." />
      <Kpi icon="database" label="IP Unik" :value="String(sum.uniqueIps || 0)" tone="" source="Alamat sumber pada entri tampil." />
    </div>

    <div class="alert warning" v-if="multiIp.length">
      <span>🚩</span>
      <span>
        <b>Kredensial dipakai dari banyak alamat</b> (perlu ditinjau):
        <div v-for="m in multiIp" :key="m.partner" style="margin-top:4px">
          <b>{{ m.partner }}</b> — {{ m.count }} alamat: <code>{{ m.ips.join(', ') }}</code>
        </div>
        <div style="margin-top:4px;font-size:12px">Gunakan bahasa netral: "potensi ketidaksesuaian" — verifikasi manusia wajib sebelum tindakan.</div>
      </span>
    </div>

    <div class="filterbar">
      <input class="input search-input" v-model="q" placeholder="Cari aktor / aksi / ID / IP…" @input="load(1)" />
      <select class="input" v-model="scope" @change="load(1)">
        <option value="">Semua Scope</option>
        <option value="WEB">WEB (aksi pengguna)</option>
        <option value="ENGINE_API">ENGINE_API (partner)</option>
      </select>
      <label style="display:flex;gap:6px;align-items:center;font-size:12.5px">
        <input type="checkbox" v-model="failedOnly" @change="load(1)" /> Hanya gagal (≥400)
      </label>
    </div>

    <div class="tbl-wrap">
      <table class="tbl" style="min-width:820px">
        <thead><tr><th>Waktu</th><th>Aktor</th><th>Aksi</th><th>Entitas</th><th>Scope</th><th>Kode</th><th>IP</th><th>Detail</th></tr></thead>
        <tbody>
          <tr v-for="e in rows" :key="e.id">
            <td style="font-size:11.5px;white-space:nowrap">{{ formatDateTime(e.at) }}</td>
            <td><b>{{ e.actor }}</b><div style="font-size:10.5px;color:var(--text-muted)">{{ e.actorRole }}</div></td>
            <td><span class="badge" :class="e.statusCode >= 400 ? 'danger' : e.scope === 'ENGINE_API' ? 'navy' : 'teal'">{{ e.action }}</span></td>
            <td style="font-size:12px">{{ e.entity }}<div class="mono" style="font-size:10.5px;color:var(--text-muted)">{{ e.entityId }}</div></td>
            <td><span class="badge neutral">{{ e.scope }}</span></td>
            <td class="num"><span v-if="e.statusCode" class="badge" :class="e.statusCode >= 400 ? 'danger' : 'success'">{{ e.statusCode }}</span><span v-else>—</span></td>
            <td class="mono" style="font-size:11px">{{ e.ip }}</td>
            <td style="font-size:11px;max-width:220px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap" :title="(e.before || '') + ' → ' + (e.after || '')">
              {{ [e.before, e.after].filter(Boolean).join(' → ') || (e.partnerName || '—') }}
            </td>
          </tr>
          <tr v-if="!rows.length"><td colspan="8"><Empty icon="⌘" title="Tidak ada entri" desc="Coba ubah filter." /></td></tr>
        </tbody>
      </table>
    </div>
    <div style="display:flex;gap:10px;align-items:center;justify-content:flex-end;margin-top:12px">
      <button class="btn btn-outline btn-sm" :disabled="page <= 1" @click="load(page - 1)">‹ Sebelumnya</button>
      <span style="font-size:12.5px;color:var(--text-muted)">Hal. {{ page }}</span>
      <button class="btn btn-outline btn-sm" :disabled="rows.length < 100" @click="load(page + 1)">Berikutnya ›</button>
    </div>
  </main>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { api } from '../api'
import { toast } from '../store'
import Kpi from '../components/Kpi.vue'
import PageHeader from '../components/PageHeader.vue'
import Empty from '../components/Empty.vue'
import { formatDateTime, downloadBlobCsv, debounce } from '../format'

const rows = ref([])
const sum = ref({})
const multiIp = ref([])
const q = ref('')
const scope = ref('')
const failedOnly = ref(false)
const page = ref(1)

const onSearch = debounce(() => load(1), 300)
watch(q, onSearch)

async function load(p = 1) {
  page.value = p
  const params = new URLSearchParams()
  if (q.value) params.set('q', q.value)
  if (scope.value) params.set('scope', scope.value)
  if (failedOnly.value) params.set('failed', '1')
  params.set('page', String(p))
  try {
    const res = await api('audit?' + params.toString())
    rows.value = res.rows
    sum.value = res.summary
    multiIp.value = res.multiIp
  } catch (e) { toast(e.message, 'error') }
}

function exportCsv() {
  const r = [['Waktu', 'Aktor', 'Peran', 'Aksi', 'Entitas', 'ID', 'Scope', 'Kode', 'IP', 'Detail']]
  for (const e of rows.value) r.push([e.at, e.actor, e.actorRole, e.action, e.entity, e.entityId, e.scope, e.statusCode || '', e.ip, [e.before, e.after].filter(Boolean).join(' → ')])
  downloadBlobCsv('audit-medpay.csv', r)
}

onMounted(() => load(1))
</script>
