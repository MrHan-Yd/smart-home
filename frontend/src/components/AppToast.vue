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
  position: relative;
  border-radius: 0.6rem;
  border: 1px solid hsl(var(--glass-border-strong));
  background: hsl(var(--glass-strong));
  backdrop-filter: blur(16px) saturate(140%);
  -webkit-backdrop-filter: blur(16px) saturate(140%);
  padding: 0.65rem 1rem 0.65rem 1.1rem;
  font-size: 0.875rem;
  box-shadow: var(--shadow-lg), 0 0 0 1px hsl(var(--glass-border));
  text-align: center;
  overflow: hidden;
}
.toast::before {
  content: "";
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 2px;
  border-radius: 999px;
}
.toast-ok::before {
  background: hsl(var(--success));
  box-shadow: 0 0 8px hsl(var(--success) / 0.6);
}
.toast-ok {
  border-color: hsl(var(--success) / 0.35);
}
.toast-err::before {
  background: hsl(var(--destructive));
  box-shadow: 0 0 8px hsl(var(--destructive) / 0.6);
}
.toast-err {
  border-color: hsl(var(--destructive) / 0.45);
}
</style>
