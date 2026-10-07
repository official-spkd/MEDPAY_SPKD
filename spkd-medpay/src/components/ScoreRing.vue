<template>
  <div style="display:inline-flex;align-items:center;gap:9px">
    <svg width="44" height="44" viewBox="0 0 44 44" role="img" :aria-label="'Skor kesiapan ' + score">
      <circle cx="22" cy="22" r="18" fill="none" stroke="var(--border)" stroke-width="5" />
      <circle cx="22" cy="22" r="18" fill="none" :stroke="ringColor" stroke-width="5" stroke-linecap="round"
        :stroke-dasharray="dash" transform="rotate(-90 22 22)" />
      <text x="22" y="26.5" text-anchor="middle" font-family="Poppins" font-weight="700" font-size="13" fill="var(--text-primary)">{{ score }}</text>
    </svg>
    <span v-if="withLabel" class="score-pill" :class="pillClass">{{ label }}</span>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  score: { type: Number, default: 0 },
  withLabel: { type: Boolean, default: false },
})

const dash = computed(() => {
  const C = 2 * Math.PI * 18
  return (Math.max(0, Math.min(100, props.score)) / 100) * C + ' ' + C
})
const ringColor = computed(() => props.score >= 80 ? '#1f9a5c' : props.score >= 50 ? '#b97324' : '#d64541')
const pillClass = computed(() => props.score >= 80 ? '' : props.score >= 50 ? 'mid' : 'low')
const label = computed(() => props.score >= 80 ? 'Siap' : props.score >= 50 ? 'Perlu perhatian' : 'Risiko tinggi')
</script>
