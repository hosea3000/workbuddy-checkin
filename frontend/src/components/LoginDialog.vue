<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { X, Copy, ExternalLink, Loader2 } from '@lucide/vue'
import { api, type LoginStatus } from '../api/bindings'
import { useToast } from '../composables/toast'

const props = defineProps<{ open: boolean; authUrl: string }>()
const emit = defineEmits<{ close: []; success: [] }>()

const status = ref<LoginStatus>({ stage: 'idle', done: false, error: '' } as LoginStatus)
const remaining = ref(0)
const toast = useToast()
let poll: number | undefined
let countdown: number | undefined

watch(
  () => props.open,
  (open) => {
    if (open) startPolling()
    else stopPolling()
  },
)

function startPolling() {
  stopPolling()
  status.value = { stage: 'awaiting_login', done: false, error: '' } as LoginStatus
  poll = window.setInterval(async () => {
    try {
      const s = await api.loginStatus()
      status.value = s
      if (s.done || s.stage === 'failed' || s.stage === 'canceled') stopPolling()
      if (s.done) emit('success')
      if (s.stage === 'failed') toast.push(s.error || '登录失败', 'error')
    } catch {
      /* 单次轮询失败忽略，下一轮重试 */
    }
  }, 5000)
}

function stopPolling() {
  if (poll) window.clearInterval(poll)
  if (countdown) window.clearInterval(countdown)
  poll = undefined
  countdown = undefined
}

function copy() {
  navigator.clipboard.writeText(props.authUrl)
  toast.push('链接已复制', 'success')
}

async function reopen() {
  await api.openAuthUrl(props.authUrl)
}

const mmss = () => {
  const m = Math.floor(remaining.value / 60)
  const s = remaining.value % 60
  return `${m}:${String(s).padStart(2, '0')}`
}

watch(
  () => props.authUrl,
  () => {
    remaining.value = 10 * 60
    if (countdown) window.clearInterval(countdown)
    countdown = window.setInterval(() => {
      if (remaining.value > 0) remaining.value--
    }, 1000)
  },
  { immediate: true },
)

function cancel() {
  api.cancelLogin().catch(() => {})
  stopPolling()
  emit('close')
}

onBeforeUnmount(stopPolling)
</script>

<template>
  <div v-if="open" class="fixed inset-0 bg-black/40 grid place-items-center p-6 z-20" @click.self="cancel">
    <div class="w-[440px] max-w-full bg-white dark:bg-slate-900 rounded-xl shadow-2xl p-5">
      <div class="flex items-center justify-between">
        <h2 class="font-semibold text-sm">添加账号</h2>
        <button class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 cursor-pointer" @click="cancel">
          <X class="w-4 h-4" />
        </button>
      </div>
      <p class="text-xs text-slate-500 mt-2 leading-relaxed">
        已自动用系统浏览器打开授权页面，请完成 CodeBuddy 登录。完成后本窗口会自动关闭。
      </p>
      <div class="mt-4 rounded-lg bg-slate-100 dark:bg-slate-800 px-3 py-2.5 text-[11px] font-mono break-all text-slate-500 leading-relaxed">
        {{ authUrl || '正在申请授权链接…' }}
      </div>
      <div class="flex gap-2 mt-3">
        <button class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 text-sm cursor-pointer" @click="copy">
          <Copy class="w-4 h-4" /> 复制链接
        </button>
        <button class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 text-sm cursor-pointer" @click="reopen">
          <ExternalLink class="w-4 h-4" /> 重新打开浏览器
        </button>
      </div>
      <div class="flex items-center gap-2 mt-4 text-xs text-slate-500">
        <Loader2 class="w-3.5 h-3.5 animate-spin text-indigo-500" />
        <span v-if="status.error">{{ status.error }}</span>
        <span v-else-if="status.stage === 'awaiting_account'">正在获取账号信息…</span>
        <span v-else>等待浏览器完成授权…</span>
        <span class="ml-auto tabular-nums">链接 {{ mmss() }} 后过期</span>
      </div>
      <button class="inline-flex w-full justify-center items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-800 text-sm cursor-pointer mt-4" @click="cancel">
        取消
      </button>
    </div>
  </div>
</template>
