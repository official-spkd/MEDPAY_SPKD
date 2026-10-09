<template>
  <svg :viewBox="'0 0 ' + W + ' ' + H" width="100%" :height="height" role="img" :aria-label="ariaLabel">
    <defs>
      <linearGradient :id="uid + 'a'" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stop-color="#1f9a5c" stop-opacity="0.32" />
        <stop offset="100%" stop-color="#1f9a5c" stop-opacity="0.02" />
      </linearGradient>
      <linearGradient :id="uid + 's'" x1="0" y1="0" x2="1" y2="0">
        <stop offset="0%" stop-color="#17804b" />
        <stop offset="100%" stop-color="#2fb56f" />
      </linearGradient>
    </defs>
    <template v-if="pts.length">
      <path :d="areaD" :fill="'url(#' + uid + 'a)'" />
      <circle v-if="pts.length > 1" :cx="pts[pts.length - 1].x" :cy="pts[pts.length - 1].y" r="7.5" fill="#1f9a5c" opacity="0.16" />
      <path :d="lineD" fill="none" :stroke="'url(#' + uid + 's)'" stroke-width="2.6" stroke-linejoin="round" stroke-linecap="round" />
      <circle v-for="(p, i) in pts" :key="i" class="lc-dot" :cx="p.x" :cy="p.y" r="3.4" fill="#fff" stroke="#1f9a5c" stroke-width="2">
        <title>{{ p.label }}: {{ p.display }}</title>
      </circle>
      <text v-for="(p, i) in xLabels" :key="'x' + i" :x="p.x" :y="H - 4" text-anchor="middle" font-size="9" fill="var(--text-muted)">{{ p.t }}</text>
    </template>
  </svg>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  data: { type: Array, default: () => [] }, // [{label, value}]
  height: { type: Number, default: 190 },
  formatter: { type: Function, default: (v) => String(v) },
  ariaLabel: { type: String, default: 'tren' },
})

// ID unik per instance agar <linearGradient> tidak bentrok antar chart dalam satu halaman
const uid = 'lc' + Math.random().toString(36).slice(2, 9)

const W = 640
const H = computed(() => props.height)
const padL = 26
const padR = 14
const baseY = computed(() => H.value - 18)
const padT = 14

const pts = computed(() => {
  const ds = props.data || []
  if (ds.length < 2) return []
  const max = Math.max(...ds.map(d => d.value), 1)
  const min = Math.min(...ds.map(d => d.value), 0)
  const span = max - min || 1
  const usable = baseY.value - padT
  return ds.map((d, i) => ({
    x: padL + (i * (W - padL - padR)) / (ds.length - 1),
    y: baseY.value - ((d.value - min) / span) * usable,
    label: d.label, display: props.formatter(d.value),
  }))
})

// Kurva halus (cubic bezier melalui titik tengah antar data)
function smoothPath(p) {
  if (p.length < 2) return ''
  let d = `M ${p[0].x.toFixed(1)} ${p[0].y.toFixed(1)}`
  for (let i = 1; i < p.length; i++) {
    const a = p[i - 1], b = p[i]
    const mx = ((a.x + b.x) / 2).toFixed(1)
    d += ` C ${mx} ${a.y.toFixed(1)}, ${mx} ${b.y.toFixed(1)}, ${b.x.toFixed(1)} ${b.y.toFixed(1)}`
  }
  return d
}

const lineD = computed(() => smoothPath(pts.value))
const areaD = computed(() => {
  const p = pts.value
  if (!p.length) return ''
  return smoothPath(p) + ` L ${p[p.length - 1].x.toFixed(1)} ${baseY.value} L ${p[0].x.toFixed(1)} ${baseY.value} Z`
})
const xLabels = computed(() => pts.value.filter((_, i) => i % Math.ceil(pts.value.length / 6) === 0)
  .map(p => ({ x: p.x, t: String(p.label).slice(-2) })))
</script>

<style scoped>
.lc-dot { transition: r 0.12s ease; }
.lc-dot:hover { r: 4.6; cursor: pointer; }
</style>
