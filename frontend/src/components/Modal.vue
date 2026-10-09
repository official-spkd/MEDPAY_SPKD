<template>
  <Teleport to="body">
    <div class="overlay" @click.self="$emit('close')">
      <div class="modal" :class="{ wide }" role="dialog" :aria-label="title">
        <div class="modal-head">
          <h3>{{ title }}</h3>
          <button class="x-btn" @click="$emit('close')" aria-label="Tutup">✕</button>
        </div>
        <div class="modal-body"><slot /></div>
        <div class="modal-foot" v-if="$slots.footer"><slot name="footer" /></div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
defineProps({ title: String, wide: Boolean })
defineEmits(['close'])
</script>
