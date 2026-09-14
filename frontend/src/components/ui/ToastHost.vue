<script setup lang="ts">
import { toasts } from '../../composables/toast'
</script>

<template>
  <div class="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm">
    <div
      v-for="t in toasts"
      :key="t.id"
      class="px-3 py-2 rounded-lg shadow-lg text-sm text-white"
      :class="{
        'bg-emerald-600': t.kind === 'success',
        'bg-rose-600': t.kind === 'error',
        'bg-slate-700': t.kind === 'info',
      }"
    >
      {{ t.text }}
    </div>
  </div>
</template>
