<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { FolderOpen, RefreshCw, LogOut, ArrowLeft, Download } from '@lucide/vue'
import { useSettingsStore } from '../stores/settings'
import { useUpdateStore } from '../stores/update'
import { useToast } from '../composables/toast'
import { api, type Settings } from '../api/bindings'

const router = useRouter()
const store = useSettingsStore()
const update = useUpdateStore()
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
  await update.loadVersion()
  await update.loadPending()
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

async function checkUpdate() {
  try {
    const result = await update.check()
    if (result && result.status === 'error') toast.push(result.message, 'error')
    else if (result) toast.push(result.message, 'info')
  } catch (e) {
    toast.push(String(e), 'error')
  }
}

async function downloadUpdate() {
  try {
    await update.download()
  } catch (e) {
    toast.push(String(e), 'error')
  }
}

async function restartUpdate() {
  const msg = await update.applyRestart()
  if (msg) toast.push(msg, 'error')
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

    <div class="bg-white dark:bg-slate-900 rounded-xl ring-1 ring-slate-200 dark:ring-slate-800 px-4 py-3 mt-4">
      <div class="flex items-center justify-between gap-4">
        <div>
          <div class="text-sm font-medium">软件更新</div>
          <div class="text-xs text-slate-500 mt-0.5">
            当前版本 v{{ update.version || 'dev' }}
            <span v-if="update.result && update.result.status !== 'update-available'"> · {{ update.result.message }}</span>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button
            class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 text-sm cursor-pointer disabled:opacity-50"
            :disabled="update.checking || update.downloading"
            @click="checkUpdate"
          >
            <RefreshCw class="w-4 h-4" /> {{ update.checking ? '检查中…' : '检查更新' }}
          </button>
          <button
            v-if="update.hasUpdate"
            class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium cursor-pointer disabled:opacity-50"
            :disabled="update.downloading"
            @click="downloadUpdate"
          >
            <Download class="w-4 h-4" /> 立即更新
          </button>
          <button
            v-else-if="update.canRestartUpdate"
            class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium cursor-pointer"
            @click="restartUpdate"
          >
            <Download class="w-4 h-4" /> 重启更新
          </button>
        </div>
      </div>
      <div v-if="update.downloading" class="mt-2">
        <div class="h-1.5 rounded-full bg-slate-100 dark:bg-slate-800 overflow-hidden">
          <div class="h-full bg-indigo-600 transition-all" :style="{ width: update.downloadPercent + '%' }"></div>
        </div>
        <div class="text-xs text-slate-500 mt-1">{{ update.progress?.message || `下载中 ${update.downloadPercent}%` }}</div>
      </div>
      <div v-else-if="update.progress?.phase === 'error'" class="text-xs text-rose-500 mt-2">{{ update.progress.message }}</div>
      <div v-else-if="update.pendingVersion" class="text-xs text-slate-500 mt-2">
        新版本 v{{ update.pendingVersion }} 已下载，点击「重启更新」生效
      </div>
    </div>

    <div class="flex items-center gap-2 mt-4">
      <button class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium cursor-pointer disabled:opacity-50" :disabled="saving" @click="save">
        保存设置
      </button>
      <button class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 text-sm cursor-pointer" @click="openDataDir">
        <FolderOpen class="w-4 h-4" /> 打开数据目录
      </button>
    </div>
  </section>
</template>
