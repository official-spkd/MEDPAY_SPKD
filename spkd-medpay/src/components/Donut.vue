<template>
  <div style="display:flex;align-items:center;gap:18px;flex-wrap:wrap">
    <svg width="130" height="130" viewBox="0 0 130 130" role="img" :aria-label="'Donat: ' + ariaLabel">
      <circle cx="65" cy="65" r="52" fill="none" stroke="var(--border)" stroke-width="17" />
      <template v-for="(s, i) in segs" :key="i">
        <circle class="donut-seg" cx="65" cy="65" r="52" fill="none" :stroke="s.color" stroke-width="17"
          :stroke-dasharray="s.dash" :stroke-dashoffset="s.offset" transform="rotate(-90 65 65)">
          <title>{{ s.label }}: {{ s.display }}</title>
        </circle>
      </template>
      <text x="65" y="60" text-anchor="middle" font-family="Poppins" font-weight="700" font-size="19" fill="var(--text-primary)">{{ center }}</text>
      <text x="65" y="77" text-anchor="middle" font-size="9.5" fill="var(--text-muted)">{{ centerSub }}</text>
    </svg>
    <div style="flex:1;min-width:170px">
      <div v-for="(s, i) in segs" :key="'l' + i" style="display:flex;align-items:center;gap:8px;padding:3.5px 0;font-size:12.5px">
        <span class="sev-dot" :style="{ background: s.color }" />
        <span style="flex:1">{{ s.label }}</span>
        <b style="font-variant-numeric:tabular-nums">{{ s.display }}</b>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  data: { type: Array, default: () => [] }, // [{label, value, color?}]
  center: { type: String, default: '' },
  centerSub: { type: String, default: '' },
  formatter: { type: Function, default: (v) => String(v) },
  ariaLabel: { type: String, default: 'komposisi' },
})

// Palet keluarga hijau MedPay (deep green menggantikan navy lama)
const palette = ['#1f9a5c', '#0e5c3a', '#37b573', '#b97324', '#2b6cb0', '#d64541', '#8b93a8']
const GAP = 3 // celah tipis antar segmen

const segs = computed(() => {
  const ds = (props.data || []).filter(d => d.value > 0)
  const total = ds.reduce((a, d) => a + d.value, 0) || 1
  const C = 2 * Math.PI * 52
  let acc = 0
  return ds.map((d, i) => {
    const frac = d.value / total
    const len = Math.max(0.5, frac * C - GAP)
    const seg = {
      label: d.label, display: props.formatter(d.value),
      color: d.color || palette[i % palette.length],
      dash: len + ' ' + (C - len),
      offset: -(acc * C + GAP / 2),
    }
    acc += frac
    return seg
  })
})
</script>

<style scoped>
.donut-seg { transition: opacity 0.15s ease; }
.donut-seg:hover { opacity: 0.78; cursor: pointer; }
</style>
