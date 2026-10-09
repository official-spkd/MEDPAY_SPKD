<template>
  <svg :viewBox="'0 0 ' + W + ' ' + H" :width="'100%'" :height="height" role="img" :aria-label="'Grafik batang: ' + ariaLabel">
    <defs>
      <linearGradient :id="uid" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" :stop-color="color" stop-opacity="1" />
        <stop offset="100%" :stop-color="color" stop-opacity="0.35" />
      </linearGradient>
    </defs>
    <template v-for="(b, i) in norm" :key="i">
      <rect class="bc-bar" :x="b.x" :y="b.y" :width="bw" :height="b.h" rx="4"
        :fill="b.neg ? colorNeg : 'url(#' + uid + ')'">
        <title>{{ b.label }}: {{ b.display }}</title>
      </rect>
      <text v-if="showLabels && bw > 26" :x="b.x + bw / 2" :y="H - 4" text-anchor="middle" font-size="9" fill="var(--text-muted)">{{ shortLabel(b.label) }}</text>
      <text v-if="showValues && bw > 30" :x="b.x + bw / 2" :y="b.y - 4" text-anchor="middle" font-size="9" font-weight="600" fill="var(--text-secondary)">{{ b.short }}</text>
    </template>
    <line v-if="norm.length" x1="0" :y1="baseY" :x2="W" :y2="baseY" stroke="var(--border)" stroke-width="1" />
  </svg>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  data: { type: Array, default: () => [] }, // [{label, value}]
  height: { type: Number, default: 190 },
  color: { type: String, default: '#1f9a5c' },
  colorNeg: { type: String, default: '#d64541' },
  showLabels: { type: Boolean, default: true },
  showValues: { type: Boolean, default: false },
  formatter: { type: Function, default: (v) => String(v) },
  shortFormatter: { type: Function, default: null },
  ariaLabel: { type: String, default: 'data' },
})

// ID unik per instance agar <linearGradient> tidak bentrok antar chart dalam satu halaman
const uid = 'bcg' + Math.random().toString(36).slice(2, 9)

const W = 640
const PAD_B = 18
const PAD_T = 16
const H = computed(() => props.height)
const baseY = computed(() => H.value - PAD_B)

const norm = computed(() => {
  const ds = props.data || []
  if (!ds.length) return []
  const max = Math.max(...ds.map(d => d.value), 1)
  const min = Math.min(...ds.map(d => d.value), 0)
  const span = max - min || 1
  const usable = baseY.value - PAD_T
  const bw = Math.min(46, (W - 8) / ds.length - 6)
  const gap = (W - bw * ds.length) / (ds.length + 1)
  return ds.map((d, i) => {
    const hVal = ((d.value - min) / span) * usable
    return {
      label: d.label,
      neg: d.value < 0,
      x: gap + i * (bw + gap),
      y: baseY.value - hVal,
      h: Math.max(2, hVal),
      display: props.formatter(d.value),
      short: props.shortFormatter ? props.shortFormatter(d.value) : props.formatter(d.value),
    }
  })
})

const bw = computed(() => norm.value.length ? Math.min(46, (W - 8) / norm.value.length - 6) : 0)

function shortLabel(l) { return String(l ?? '').length > 9 ? String(l).slice(0, 8) + '…' : l }
</script>

<style scoped>
.bc-bar { transition: opacity 0.14s ease; }
.bc-bar:hover { opacity: 0.78; cursor: pointer; }
</style>
