import { defineStore } from 'pinia'
import { api, type AccountView, type CheckinSummary } from '../api/bindings'
import { useToast } from '../composables/toast'

let timer: number | undefined

export const useAccountsStore = defineStore('accounts', {
  state: () => ({
    accounts: [] as AccountView[],
    summary: { checkedIn: 0, total: 0, reloginRequired: 0 } as CheckinSummary,
    loading: false,
    checkedInId: '' as string,
  }),
  getters: {
    hasAccounts: (s) => s.accounts.length > 0,
  },
  actions: {
    async refresh() {
      this.accounts = await api.listAccounts()
      this.summary = await api.checkinSummary()
    },
    async checkinNow(id: string) {
      const toast = useToast()
      this.checkedInId = id
      try {
        const r = await api.checkinNow(id)
        toast.push(r.message, r.success ? 'success' : 'error')
      } catch (e) {
        toast.push(String(e), 'error')
      } finally {
        this.checkedInId = ''
        await this.refresh()
      }
    },
    async checkinAll() {
      const toast = useToast()
      try {
        const rs = await api.checkinAll()
        const ok = rs.filter((r) => r.success).length
        toast.push(`完成签到 ${ok}/${rs.length}`, ok === rs.length ? 'success' : 'error')
      } catch (e) {
        toast.push(String(e), 'error')
      } finally {
        await this.refresh()
      }
    },
    async refreshQuota(id: string) {
      const toast = useToast()
      try {
        await api.refreshQuota(id)
        toast.push('余额已刷新', 'success')
      } catch (e) {
        toast.push(String(e), 'error')
      } finally {
        await this.refresh()
      }
    },
    async remove(id: string) {
      await api.deleteAccount(id)
      await this.refresh()
    },
    async setActive(id: string) {
      const toast = useToast()
      try {
        await api.setActiveCredential(id)
        toast.push('已设为当前凭证', 'success')
      } catch (e) {
        toast.push(String(e), 'error')
      } finally {
        await this.refresh()
      }
    },
    startPolling() {
      this.stopPolling()
      this.refresh()
      timer = window.setInterval(() => this.refresh(), 30_000)
    },
    stopPolling() {
      if (timer) window.clearInterval(timer)
      timer = undefined
    },
  },
})
