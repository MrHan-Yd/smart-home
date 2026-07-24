<script setup lang="ts">
import { useToast } from '@/composables/useToast'

const { items } = useToast()
</script>

<template>
  <div class="toast-host">
    <div
      v-for="t in items"
      :key="t.id"
      class="toast"
      :class="t.type === 'err' ? 'toast-err' : 'toast-ok'"
    >
      {{ t.message }}
    </div>
  </div>
</template>

<style scoped>
.toast-host {
  pointer-events: none;
  position: fixed;
  bottom: 5.5rem;
  left: 50%;
  transform: translateX(-50%);
  z-index: 100;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  width: min(22rem, calc(100vw - 2rem));
}
@media (min-width: 900px) {
  .toast-host {
    bottom: 1.5rem;
  }
}
.toast {
  border-radius: 0.5rem;
  border: 1px solid hsl(var(--border));
  background: hsl(var(--card));
  padding: 0.65rem 1rem;
  font-size: 0.875rem;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.25);
  text-align: center;
}
.toast-ok {
  border-color: hsl(var(--success) / 0.45);
}
.toast-err {
  border-color: hsl(var(--destructive) / 0.5);
}
</style>
