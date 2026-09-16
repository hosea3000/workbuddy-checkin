<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowRight, Pin } from '@lucide/vue'
import { useSettingsStore } from '../stores/settings'
import { useAccountsStore } from '../stores/accounts'
import { useToast } from '../composables/toast'
import { api, type ProxyStatus, type Settings } from '../api/bindings'

const router = useRouter()
const store = useSettingsStore()
const accounts = useAccountsStore()
const toast = useToast()
const port = ref(18080)
const busy = ref(false)

const proxyStatus = ref<ProxyStatus>({ running: false, port: 0, error: '' })
let statusTimer: number | undefined

const activeCredentialText = computed(() => {
  const id = store.settings?.activeCredentialId ?? ''
  if (id === '') return '未设置'
  const a = accounts.accounts.find((x) => x.isActive)
  if (!a) return `已失效（${id}）`
  return a.nickname || a.email || a.id
})

const proxyStatusText = computed(() => {
  const s = proxyStatus.value
  if (s.error) return `启动失败：${s.error}`
  if (s.running) return `运行中 · http://127.0.0.1:${s.port}`
  return '已停止'
})
const proxyStatusClass = computed(() => {
  if (proxyStatus.value.error) return 'text-rose-500'
  if (proxyStatus.value.running) return 'text-emerald-600 dark:text-emerald-400'
  return 'text-slate-400'
})

onMounted(async () => {
  await store.load()
  port.value = store.settings?.proxyPort ?? 18080
  accounts.refresh()
  await refreshProxyStatus()
  statusTimer = window.setInterval(refreshProxyStatus, 2_000)
})

onUnmounted(() => {
  if (statusTimer) window.clearInterval(statusTimer)
})

async function refreshProxyStatus() {
  try {
    proxyStatus.value = await api.proxyStatus()
  } catch {
    // 状态查询失败不打扰用户，保持上一次结果
  }
}

// 启停：以当前端口为准落库并重启服务，不依赖页面上的开关状态
async function toggle() {
  busy.value = true
  try {
    await store.load()
    await store.save({ ...(store.settings as Settings), proxyPort: port.value, proxyEnabled: !proxyStatus.value.running })
    await refreshProxyStatus()
  } catch (e) {
    toast.push(String(e), 'error')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <section class="p-5 max-w-2xl mx-auto">
    <div class="mb-4">
      <h1 class="font-semibold">模型代理</h1>
      <p class="text-xs text-slate-500 mt-0.5">在本机提供 OpenAI 兼容接口，供其他AI客户端使用</p>
    </div>

    <div class="bg-white dark:bg-slate-900 rounded-xl ring-1 ring-slate-200 dark:ring-slate-800 p-0 divide-y divide-slate-100 dark:divide-slate-800">
      <div class="flex items-center justify-between gap-4 px-4 py-3">
        <div>
          <div class="text-sm font-medium">服务状态</div>
          <div class="text-xs text-slate-500 mt-0.5">仅监听 127.0.0.1，不对局域网开放</div>
        </div>
        <button
          class="shrink-0 inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-white text-sm font-medium cursor-pointer disabled:opacity-50"
          :class="proxyStatus.running ? 'bg-rose-600 hover:bg-rose-500' : 'bg-indigo-600 hover:bg-indigo-500'"
          :disabled="busy"
          @click="toggle"
        >
          {{ busy ? '处理中…' : proxyStatus.running ? '停止' : '启动' }}
        </button>
      </div>
      <div class="flex items-center justify-between gap-4 px-4 py-3">
        <div>
          <div class="text-sm font-medium">监听端口</div>
          <div class="text-xs text-slate-500 mt-0.5">
            可填 1024–65535<span v-if="proxyStatus.running"> · 改动需先「停止」再「启动」生效</span>
          </div>
        </div>
        <input
          v-model.number="port"
          type="number"
          min="1024"
          max="65535"
          class="w-28 shrink-0 rounded-lg ring-1 ring-slate-200 dark:ring-slate-700 bg-white dark:bg-slate-800 px-2.5 py-1.5 text-sm"
        />
      </div>
      <div class="px-4 py-2.5">
        <div class="text-xs" :class="proxyStatusClass">状态：{{ proxyStatusText }}</div>
        <div class="text-xs text-slate-400 mt-1">
          客户端 Base URL 填 <code class="px-1 rounded bg-slate-100 dark:bg-slate-800">http://127.0.0.1:{{ proxyStatus.running ? proxyStatus.port : port || 18080 }}/v1</code>，API Key 随意填
        </div>
      </div>
    </div>

    <div class="bg-white dark:bg-slate-900 rounded-xl ring-1 ring-slate-200 dark:ring-slate-800 px-4 py-3 mt-4 flex items-center justify-between gap-4">
      <div class="min-w-0">
        <div class="text-sm font-medium flex items-center gap-1.5">
          <Pin class="w-3.5 h-3.5 text-slate-400" /> 当前凭证
        </div>
        <div class="text-xs text-slate-500 mt-0.5 truncate">
          代理转发使用的账号：{{ activeCredentialText }}
        </div>
      </div>
      <button
        class="shrink-0 inline-flex items-center gap-1 text-xs text-indigo-600 dark:text-indigo-400 cursor-pointer"
        @click="router.push({ name: 'accounts' })"
      >
        去账号页设置 <ArrowRight class="w-3.5 h-3.5" />
      </button>
    </div>
  </section>
</template>
