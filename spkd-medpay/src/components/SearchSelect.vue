<template>
  <div style="position:relative">
    <input class="input" :value="queryText" @focus="open = true" @input="onType" :placeholder="placeholder" role="combobox" :aria-expanded="open" />
    <div v-if="open && filtered.length" style="position:absolute;z-index:50;top:42px;left:0;right:0;background:var(--surface);border:1px solid var(--border);border-radius:10px;box-shadow:var(--shadow-lg);max-height:260px;overflow-y:auto">
      <div v-for="o in filtered" :key="o.id" @mousedown.prevent="pick(o)"
        style="padding:9px 13px;cursor:pointer;font-size:13px;border-bottom:1px solid var(--border)"
        @mouseover="$event.currentTarget.style.background='var(--primary-light)'"
        @mouseout="$event.currentTarget.style.background=''">
        {{ o[labelKey] }} <span v-if="o.sub" style="color:var(--text-muted);font-size:11.5px">· {{ o.sub }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  options: { type: Array, default: () => [] }, // [{id, label, sub?}]
  modelValue: { type: String, default: '' },
  placeholder: String,
  labelKey: { type: String, default: 'label' },
})
const emit = defineEmits(['update:modelValue'])
const open = ref(false)
const queryText = ref('')

const filtered = computed(() => {
  const q = queryText.value.toLowerCase()
  return props.options.filter(o => !q || o[props.labelKey].toLowerCase().includes(q)).slice(0, 60)
})

function onType(e) { queryText.value = e.target.value; open.value = true }
function pick(o) {
  queryText.value = o[props.labelKey]
  emit('update:modelValue', o.id)
  open.value = false
}
defineExpose({ setText: (t) => { queryText.value = t } })
</script>
