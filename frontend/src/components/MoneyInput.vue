<template>
  <input class="input money-in" type="text" inputmode="numeric" :value="display" @input="onInput" :disabled="disabled" :placeholder="placeholder" />
</template>

<script setup>
import { computed } from 'vue'
import { parseMoney, formatNumber } from '../format'

const props = defineProps({ modelValue: { type: Number, default: 0 }, disabled: Boolean, placeholder: String })
const emit = defineEmits(['update:modelValue'])

const display = computed(() => props.modelValue ? formatNumber(props.modelValue) : '')

function onInput(e) {
  const v = parseMoney(e.target.value)
  emit('update:modelValue', v)
  e.target.value = v ? formatNumber(v) : ''
}
</script>
