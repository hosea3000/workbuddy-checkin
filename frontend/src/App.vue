<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Settings as SettingsIcon, CalendarCheck } from '@lucide/vue'
import ToastHost from './components/ui/ToastHost.vue'
import { useAccountsStore } from './stores/accounts'
import { api } from './api/bindings'

const route = useRoute()
const router = useRouter()
const accounts = useAccountsStore()

const summaryText = computed(() => {
  if (accounts.summary.reloginRequired > 0) return '有账号需重新登录'
  return `今日已签到 ${accounts.summary.checkedIn}/${accounts.summary.total}`
})

onMounted(async () => {
  if (route.name === 'welcome') return
  const has = await api.hasAccounts()
  if (!has) {
    router.replace({ name: 'welcome' })
    return
  }
  accounts.startPolling()
})
</script>

<template>
  <div class="h-screen flex flex-col bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-slate-100">
    <header class="h-14 shrink-0 bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 flex items-center gap-3 px-4">
      <img src="/icon.png" alt="workbuddy-checkin" class="w-7 h-7 rounded-lg object-cover" />
      <div class="font-semibold text-sm">workbuddy-checkin</div>
      <div class="ml-auto flex items-center gap-3">
        <span
          v-if="route.name === 'accounts' && accounts.summary.total"
          class="text-xs flex items-center gap-1"
          :class="accounts.summary.reloginRequired > 0 ? 'text-rose-500' : 'text-slate-500 dark:text-slate-400'"
        >
          <CalendarCheck class="w-3.5 h-3.5" :class="accounts.summary.reloginRequired > 0 ? 'text-rose-500' : 'text-emerald-500'" />
          {{ summaryText }}
        </span>
        <button
          v-if="route.name !== 'settings'"
          class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 cursor-pointer"
          title="设置"
          @click="router.push({ name: 'settings' })"
        >
          <SettingsIcon class="w-4 h-4" />
        </button>
      </div>
    </header>
    <main class="flex-1 overflow-y-auto">
      <router-view />
    </main>
    <ToastHost />
  </div>
</template>
