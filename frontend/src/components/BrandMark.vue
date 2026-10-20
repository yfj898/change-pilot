<template>
  <span class="brand-mark" :aria-label="product.name">
    <svg class="brand-mark__symbol" viewBox="0 0 32 32" fill="none" aria-hidden="true">
      <rect x="1" y="1" width="30" height="30" rx="9" fill="currentColor" />
      <path d="M21 10.5a8 8 0 1 0 1.3 9.2" stroke="var(--brand-symbol-ink, white)" stroke-width="2.3" stroke-linecap="round" />
      <path d="m12.3 16.1 3.3 3.2L24 11" stroke="var(--brand-symbol-ink, white)" stroke-width="2.3" stroke-linecap="round" stroke-linejoin="round" />
    </svg>
    <span class="brand-mark__name">{{ product.name }}</span>
  </span>
</template>

<script setup lang="ts">
import { product } from '@/config/branding'
</script>

<style scoped>
.brand-mark {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  white-space: nowrap;
  color: var(--td-brand-color, #07a35a);
}

.brand-mark__symbol {
  flex: 0 0 auto;
  width: 27px;
  height: 27px;
}

.brand-mark__name {
  color: var(--td-text-color-primary, #172d23);
  font-size: 18px;
  font-weight: 720;
  letter-spacing: -0.6px;
  line-height: 1.25;
}
</style>
