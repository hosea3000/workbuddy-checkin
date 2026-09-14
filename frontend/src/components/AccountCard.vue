<script setup lang="ts">
import { computed } from 'vue'
import { RefreshCw, Trash2 } from '@lucide/vue'
import StatusBadge from './StatusBadge.vue'
import type { AccountView } from '../api/bindings'

const props = defineProps<{ account: AccountView; checking: boolean }>()
const emit = defineEmits<{
  checkin: [id: string]
  refreshQuota: [id: string]
  remove: [id: string]
}>()

const title = computed(() => props.account.nickname || props.account.id.slice(0, 12))
const avatar = computed(() => title.value.slice(0, 1).toUpperCase())

const subtitle = computed(() => {
  const parts = [props.account.email].filter(Boolean)
  if (props.account.tokenSuffix) parts.push(`令牌尾号 ${props.account.tokenSuffix}`)
  if (props.account.expiresAt) {
    const d = new Date(props.account.expiresAt * 1000)
    parts.push(`有效期至 ${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`)
  }
  return parts.join(' · ')
})

const todayText = computed(() => {
  const t = props.account.today
  if (props.account.status === 'relogin_required') return '凭证已过期，自动签到已跳过该账号'
  if (t.checkedIn) {
    const credit = t.credit != null ? `，+${t.credit} 积分` : ''
    return `今日 ${t.time} 已签到${credit}`
  }
  if (t.message) return t.message
  return '今天还未签到，到点自动执行'
})

const balance = computed(() => (props.account.creditBalance == null ? '—' : format(props.account.creditBalance)))
const balanceSub = computed(() =>
  props.account.creditBalanceAt ? `积分余额 · ${timeOf(props.account.creditBalanceAt)}` : '积分余额',
)

function format(n: number) {
  return n.toLocaleString('zh-CN')
}
function timeOf(ts: number) {
  if (!ts) return ''
  const d = new Date(ts * 1000)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

function del() {
  if (window.confirm(`确定删除账号「${title.value}」？仅删除本地凭证。`)) emit('remove', props.account.id)
}
</script>

<template>
  <div
    class="flex items-center gap-4 bg-white dark:bg-slate-900 rounded-xl ring-1 p-4"
    :class="account.status === 'relogin_required' ? 'ring-rose-200 dark:ring-rose-500/30' : 'ring-slate-200 dark:ring-slate-800'"
  >
    <div
      class="w-10 h-10 shrink-0 rounded-full grid place-items-center font-semibold"
      :class="account.status === 'relogin_required' ? 'bg-rose-100 dark:bg-rose-500/20 text-rose-600 dark:text-rose-300' : 'bg-indigo-100 dark:bg-indigo-500/20 text-indigo-600 dark:text-indigo-300'"
    >
      {{ avatar }}
    </div>
    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-2">
        <span class="font-medium text-sm truncate">{{ title }}</span>
        <StatusBadge :status="account.status" :checked-in="account.today.checkedIn" :checking="checking" />
      </div>
      <div class="text-xs text-slate-500 mt-0.5 truncate">{{ subtitle }}</div>
      <div
        class="text-xs mt-1"
        :class="account.today.checkedIn ? 'text-emerald-600 dark:text-emerald-400' : 'text-slate-400'"
      >
        {{ todayText }}
      </div>
    </div>
    <button class="shrink-0 group text-right mr-1 cursor-pointer" title="点击刷新余额" @click="emit('refreshQuota', account.id)">
      <div class="text-sm font-semibold tabular-nums flex items-center justify-end gap-1">
        {{ balance }}
        <RefreshCw class="w-3 h-3 text-slate-300 group-hover:text-indigo-500" />
      </div>
      <div class="text-[10px] text-slate-400">{{ balanceSub }}</div>
    </button>
    <button
      v-if="account.status !== 'relogin_required'"
      class="shrink-0 inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 text-sm cursor-pointer disabled:opacity-50"
      :disabled="checking"
      @click="emit('checkin', account.id)"
    >
      {{ checking ? '签到中…' : '立即签到' }}
    </button>
    <template v-else>
      <button
        class="shrink-0 inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-rose-50 dark:bg-rose-500/15 text-rose-600 dark:text-rose-400 hover:bg-rose-100 text-sm font-medium cursor-pointer"
        @click="emit('checkin', account.id)"
      >
        重新登录
      </button>
    </template>
    <button class="shrink-0 p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 cursor-pointer" title="删除" @click="del">
      <Trash2 class="w-4 h-4" />
    </button>
  </div>
</template>
