import { defineStore } from 'pinia'
import {
  api,
  type UpdateCheckResult,
  type UpdateDownloadEvent,
  type PendingUpdateInfo,
} from '../api/bindings'

// useUpdateStore 管理更新检查与下载进度。
// 后端不通过 Wails 事件推送状态（见 AGENTS.md），前端轮询绑定方法获取进度。
export const useUpdateStore = defineStore('update', {
  state: () => ({
    version: '',
    result: null as UpdateCheckResult | null,
    progress: null as UpdateDownloadEvent | null,
    pending: null as PendingUpdateInfo | null,
    checking: false,
    downloading: false,
  }),
  getters: {
    hasUpdate: (s) => s.result?.status === 'update-available',
    latestVersion: (s) => s.result?.latestVersion ?? '',
    pendingVersion: (s) => (s.pending?.exists ? s.pending.version : ''),
    downloadPercent: (s) => s.progress?.percent ?? 0,
    canRestartUpdate: (s) => !!s.pending?.exists || s.progress?.phase === 'cancelled',
  },
  actions: {
    async loadVersion() {
      this.version = await api.getVersion()
    },
    async loadPending() {
      this.pending = await api.pendingUpdateInfo()
    },
    async check() {
      this.checking = true
      try {
        this.result = await api.checkUpdate()
        return this.result
      } finally {
        this.checking = false
      }
    },
    async download() {
      const err = await api.downloadAndApplyUpdate()
      if (err) throw new Error(err)
      this.downloading = true
      this.pollProgress()
    },
    pollProgress() {
      this.progress = null
      const timer = setInterval(async () => {
        const ev = await api.updateProgress()
        this.progress = ev
        if (ev.phase === 'completed' || ev.phase === 'error' || ev.phase === 'cancelled') {
          clearInterval(timer)
          this.downloading = false
          await this.loadPending()
        }
      }, 500)
    },
    async applyRestart(): Promise<string> {
      const msg = await api.applyUpdateAndRestart()
      if (!msg) await this.loadPending()
      return msg
    },
  },
})
