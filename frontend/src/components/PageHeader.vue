<template>
  <div class="page-head">
    <div class="ph-left">
      <span v-if="icon" class="ph-ico" :class="tone"><Icon :name="icon" :size="19" /></span>
      <div class="ph-txt">
        <div class="page-title-row">
          <div class="page-title">{{ title }}</div>
          <slot name="badges" />
        </div>
        <div v-if="sub" class="page-sub">{{ sub }}</div>
      </div>
    </div>
    <div v-if="$slots.actions" class="chips-row"><slot name="actions" /></div>
  </div>
</template>

<script setup>
import Icon from './Icon.vue'

defineProps({
  icon: String,
  title: { type: String, required: true },
  sub: String,
  tone: { type: String, default: '' }, // ''|red|amber|blue|navy
})
</script>
