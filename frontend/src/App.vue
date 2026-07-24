<script setup lang="ts">
import { RouterView } from 'vue-router'
import { useTheme } from '@/composables/useTheme'

// apply theme early
useTheme()
</script>

<template>
  <RouterView />
</template>
