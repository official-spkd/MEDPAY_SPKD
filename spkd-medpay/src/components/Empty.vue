<template>
  <div class="empty">
    <div class="ico">{{ icon }}</div>
    <div class="t">{{ title }}</div>
    <div class="d">{{ desc }}</div>
    <div style="margin-top:14px"><slot /></div>
  </div>
</template>

<script setup>
defineProps({ icon: { type: String, default: '🗂' }, title: String, desc: String })
</script>
