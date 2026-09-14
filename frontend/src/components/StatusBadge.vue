<script setup lang="ts">
import { computed } from 'vue'
import { Loader2, AlertTriangle } from '@lucide/vue'

const props = defineProps<{ status: string; checkedIn: boolean; checking: boolean }>()

const kind = computed(() => {
  if (props.checking) return 'checking'
  if (props.status === 'relogin_required') return 'relogin'
  return props.checkedIn ? 'done' : 'pending'
})
</script>

<template>
  <span
    class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md text-[11px] font-medium"
    :class="{
      'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-400': kind === 'done',
      'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400': kind === 'pending',
      'bg-rose-50 text-rose-700 dark:bg-rose-500/15 dark:text-rose-400': kind === 'relogin',
      'bg-indigo-50 text-indigo-600 dark:bg-indigo-500/15 dark:text-indigo-300': kind === 'checking',
    }"
  >
    <Loader2 v-if="kind === 'checking'" class="w-3 h-3 animate-spin" />
    <AlertTriangle v-else-if="kind === 'relogin'" class="w-3 h-3" />
    {{ kind === 'done' ? '已签到' : kind === 'pending' ? '待签到' : kind === 'relogin' ? '需重新登录' : '签到中' }}
  </span>
</template>
