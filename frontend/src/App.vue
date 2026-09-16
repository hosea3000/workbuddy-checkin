<script setup lang="ts">
import { onMounted, computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Settings as SettingsIcon, CalendarCheck, Download, Plug, ChevronLeft, ChevronRight } from '@lucide/vue'
import ToastHost from './components/ui/ToastHost.vue'
import { useAccountsStore } from './stores/accounts'
import { useUpdateStore } from './stores/update'
import { api } from './api/bindings'

const route = useRoute()
const router = useRouter()
const accounts = useAccountsStore()
const update = useUpdateStore()

const navItems = [
  { name: 'accounts', label: '账号签到', icon: CalendarCheck },
  { name: 'proxy', label: '模型代理', icon: Plug },
  { name: 'settings', label: '设置', icon: SettingsIcon },
] as const

const SIDEBAR_KEY = 'sidebar-collapsed'
const collapsed = ref(readCollapsed())

function readCollapsed() {
  try {
    return localStorage.getItem(SIDEBAR_KEY) === '1'
  } catch {
    return false
  }
}

function toggleSidebar() {
  collapsed.value = !collapsed.value
  try {
    localStorage.setItem(SIDEBAR_KEY, collapsed.value ? '1' : '0')
  } catch {
    // 写入失败只是不记住，不影响本次切换
  }
}

const summaryText = computed(() => {
  if (accounts.summary.reloginRequired > 0) return '有账号需重新登录'
  return `今日已签到 ${accounts.summary.checkedIn}/${accounts.summary.total}`
})

onMounted(async () => {
  update.loadVersion()
  update.loadPending()
  update.check()
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
  <div class="h-screen bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-slate-100">
    <router-view v-if="route.name === 'welcome'" />

    <div v-else class="h-full flex flex-col">
      <header class="h-14 shrink-0 bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 flex items-center gap-3 px-4">
        <img src="/icon.png" alt="WorkBuddy 自动签到" class="w-7 h-7 rounded-lg object-cover" />
        <div class="font-semibold text-sm">WorkBuddy 自动签到</div>
        <div class="ml-auto flex items-center gap-3">
          <button
            v-if="update.hasUpdate || update.pendingVersion"
            class="text-xs flex items-center gap-1 text-indigo-600 dark:text-indigo-400 cursor-pointer"
            title="查看更新"
            @click="router.push({ name: 'settings' })"
          >
            <Download class="w-3.5 h-3.5" />
            {{ update.pendingVersion ? `新版本 v${update.pendingVersion} 待重启` : `发现新版本 v${update.latestVersion}` }}
          </button>
          <span
            v-if="route.name === 'accounts' && accounts.summary.total"
            class="text-xs flex items-center gap-1"
            :class="accounts.summary.reloginRequired > 0 ? 'text-rose-500' : 'text-slate-500 dark:text-slate-400'"
          >
            <CalendarCheck class="w-3.5 h-3.5" :class="accounts.summary.reloginRequired > 0 ? 'text-rose-500' : 'text-emerald-500'" />
            {{ summaryText }}
          </span>
        </div>
      </header>

      <div class="flex-1 flex min-h-0">
        <aside
          class="shrink-0 flex flex-col bg-white dark:bg-slate-900 border-r border-slate-200 dark:border-slate-800 p-2 transition-[width] duration-150"
          :class="collapsed ? 'w-14' : 'w-40'"
        >
          <nav class="space-y-1">
            <button
              v-for="item in navItems"
              :key="item.name"
              class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm cursor-pointer"
              :class="[
                collapsed ? 'justify-center' : '',
                route.name === item.name
                  ? 'bg-indigo-50 dark:bg-indigo-500/20 text-indigo-600 dark:text-indigo-400 font-medium'
                  : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800',
              ]"
              :title="item.label"
              @click="router.push({ name: item.name })"
            >
              <component :is="item.icon" class="w-4 h-4 shrink-0" />
              <span v-if="!collapsed">{{ item.label }}</span>
            </button>
          </nav>

          <button
            class="mt-auto w-full flex items-center justify-center px-3 py-2 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 cursor-pointer"
            :title="collapsed ? '展开侧栏' : '折叠侧栏'"
            @click="toggleSidebar"
          >
            <ChevronRight v-if="collapsed" class="w-4 h-4 shrink-0" />
            <ChevronLeft v-else class="w-4 h-4 shrink-0" />
          </button>
        </aside>

        <main class="flex-1 overflow-y-auto">
          <router-view />
        </main>
      </div>
    </div>

    <ToastHost />
  </div>
</template>
