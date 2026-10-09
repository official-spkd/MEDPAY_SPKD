<template>
  <div class="kpi" :class="tone">
    <div class="kpi-head">
      <div class="lbl">{{ label }}</div>
      <span v-if="icon" class="kpi-ico"><Icon :name="icon" :size="15" /></span>
    </div>
    <div class="val" :class="valClass">{{ value }}</div>
    <div class="src" v-if="source">{{ source }}</div>
  </div>
</template>

<script setup>
import Icon from './Icon.vue'

defineProps({
  label: String,
  value: [String, Number],
  source: String,
  icon: String,                                // nama ikon opsional
  tone: { type: String, default: '' },      // navy|success|warning|danger|''
  valClass: { type: String, default: '' },  // success|warning|danger
})
</script>
