<template>
  <div class="logo-row" :style="{ gap: gap + 'px' }">
    <div class="logo-badge" :style="{ width: bx + 'px', height: bx + 'px', borderRadius: Math.round(bx * 0.3) + 'px' }">
      <svg :width="bx * 0.56" :height="bx * 0.56" viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path d="M9.5 3h5v6.5H21v5h-6.5V21h-5v-6.5H3v-5h6.5V3z" fill="#ffffff" />
      </svg>
    </div>
    <div class="logo-text brand-text" v-if="withText">
      <div class="logo-name" :style="{ fontSize: fs + 'px', color: light ? '#ffffff' : '#0b3d2a' }">
        Med<span :style="{ color: light ? '#3fd487' : '#1f9a5c' }">Pay</span>
      </div>
      <div v-if="sub" class="logo-sub" :style="{ color: light ? 'rgba(255,255,255,0.72)' : '#5e8a74' }">{{ sub }}</div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  light: { type: Boolean, default: false },
  withText: { type: Boolean, default: true },
  sub: { type: String, default: '' },
  fs: { type: Number, default: 20 },
  bx: { type: Number, default: 38 },
  gap: { type: Number, default: 11 },
})
</script>

<style scoped>
.logo-row { display: inline-flex; align-items: center; }
.logo-badge {
  background: linear-gradient(135deg, #1c8a53 0%, #1f9a5c 55%, #2fb56f 100%);
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
  box-shadow: 0 4px 12px rgba(31, 154, 92, 0.4);
}
.logo-name { font-family: 'Poppins', sans-serif; font-weight: 700; line-height: 1.05; white-space: nowrap; letter-spacing: 0.2px; }
.logo-sub { font-size: 8.5px; font-weight: 600; letter-spacing: 1.6px; text-transform: uppercase; color: #5e8a74; margin-top: 1px; white-space: nowrap; }
</style>
