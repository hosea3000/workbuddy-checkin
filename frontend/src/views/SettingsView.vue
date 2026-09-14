<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { FolderOpen, RefreshCw, LogOut, ArrowLeft } from '@lucide/vue'
import { useSettingsStore } from '../stores/settings'
import { useToast } from '../composables/toast'
import { api, type Settings } from '../api/bindings'

const router = useRouter()
const store = useSettingsStore()
const toast = useToast()
const form = reactive<Settings>({
  checkinHour: 9,
  checkinMinute: 30,
  catchUpOnStart: true,
  retryOnFailure: true,
  notifySuccess: true,
  notifyFailure: true,
  autoStart: false,
  minimizeToTrayOnClose: true,
  checkUpdateOnStart: true,
})
const saving = ref(false)

onMounted(async () => {
  await store.load()
  if (store.settings) Object.assign(form, store.settings)
})

async function save() {
  saving.value = true
  try {
    await store.save({ ...form })
    toast.push('设置已保存', 'success')
  } catch (e) {
    toast.push(String(e), 'error')
  } finally {
    saving.value = false
  }
}

async function openDataDir() {
  try {
    await api.openDataDir()
  } catch (e) {
    toast.push(String(e), 'error')
  }
}
</script>

<template>
  <section class="p-5 max-w-2xl mx-auto">
    <div class="flex items-center gap-2 mb-1">
      <button class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800 cursor-pointer" @click="router.push({ name: 'accounts' })">
        <ArrowLeft class="w-4 h-4" />
      </button>
      <h1 class="font-semibold">设置</h1>
    </div>
    <p class="text-xs text-slate-500 mb-4 ml-9">所有数据仅保存在本机</p>

    <div class="bg-white dark:bg-slate-900 rounded-xl ring-1 ring-slate-200 dark:ring-slate-800 p-0 divide-y divide-slate-100 dark:divide-slate-800">
      <div class="flex items-center justify-between gap-4 px-4 py-3">
        <div><div class="text-sm font-medium">每日签到时间</div><div class="text-xs text-slate-500 mt-0.5">本地时区，到点自动签到</div></div>
        <input v-model.number="form.checkinHour" type="number" min="0" max="23" class="w-14 rounded-lg ring-1 ring-slate-200 dark:ring-slate-700 bg-white dark:bg-slate-800 px-2.5 py-1.5 text-sm" />
        <span class="text-slate-400">:</span>
        <input v-model.number="form.checkinMinute" type="number" min="0" max="59" class="w-14 rounded-lg ring-1 ring-slate-200 dark:ring-slate-700 bg-white dark:bg-slate-800 px-2.5 py-1.5 text-sm" />
      </div>
      <label class="flex items-center justify-between gap-4 px-4 py-3 cursor-pointer">
        <div><div class="text-sm font-medium">启动时补签</div><div class="text-xs text-slate-500 mt-0.5">错过了签到时间，开机后立即补签</div></div>
        <input v-model="form.catchUpOnStart" type="checkbox" class="w-9 h-5 accent-indigo-600" />
      </label>
      <label class="flex items-center justify-between gap-4 px-4 py-3 cursor-pointer">
        <div><div class="text-sm font-medium">失败自动重试</div><div class="text-xs text-slate-500 mt-0.5">间隔 30 分钟，每天最多 3 次</div></div>
        <input v-model="form.retryOnFailure" type="checkbox" class="w-9 h-5 accent-indigo-600" />
      </label>
      <label class="flex items-center justify-between gap-4 px-4 py-3 cursor-pointer">
        <div><div class="text-sm font-medium">成功通知</div><div class="text-xs text-slate-500 mt-0.5">签到成功时弹一条通知</div></div>
        <input v-model="form.notifySuccess" type="checkbox" class="w-9 h-5 accent-indigo-600" />
      </label>
      <label class="flex items-center justify-between gap-4 px-4 py-3 cursor-pointer">
        <div><div class="text-sm font-medium">失败通知</div><div class="text-xs text-slate-500 mt-0.5">签到失败时弹一条通知</div></div>
        <input v-model="form.notifyFailure" type="checkbox" class="w-9 h-5 accent-indigo-600" />
      </label>
      <label class="flex items-center justify-between gap-4 px-4 py-3 cursor-pointer">
        <div><div class="text-sm font-medium">开机自启</div><div class="text-xs text-slate-500 mt-0.5">写入 HKCU，无需管理员权限</div></div>
        <input v-model="form.autoStart" type="checkbox" class="w-9 h-5 accent-indigo-600" />
      </label>
      <label class="flex items-center justify-between gap-4 px-4 py-3 cursor-pointer">
        <div><div class="text-sm font-medium">关闭窗口时最小化到托盘</div><div class="text-xs text-slate-500 mt-0.5">关闭后仍在后台运行并按时签到</div></div>
        <input v-model="form.minimizeToTrayOnClose" type="checkbox" class="w-9 h-5 accent-indigo-600" />
      </label>
      <label class="flex items-center justify-between gap-4 px-4 py-3 cursor-pointer">
        <div><div class="text-sm font-medium">启动时检查更新</div><div class="text-xs text-slate-500 mt-0.5">从 GitHub Release 检查新版本</div></div>
        <input v-model="form.checkUpdateOnStart" type="checkbox" class="w-9 h-5 accent-indigo-600" />
      </label>
    </div>

    <div class="flex items-center gap-2 mt-4">
      <button class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium cursor-pointer disabled:opacity-50" :disabled="saving" @click="save">
        保存设置
      </button>
      <button class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 text-sm cursor-pointer" @click="openDataDir">
        <FolderOpen class="w-4 h-4" /> 打开数据目录
      </button>
      <button class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 text-sm cursor-pointer" @click="toast.push('更新检查待 P2 实现', 'info')">
        <RefreshCw class="w-4 h-4" /> 检查更新
      </button>
    </div>
  </section>
</template>
