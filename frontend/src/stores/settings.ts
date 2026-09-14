import { defineStore } from 'pinia'
import { api, type Settings } from '../api/bindings'

export const useSettingsStore = defineStore('settings', {
  state: () => ({
    settings: null as Settings | null,
  }),
  actions: {
    async load() {
      this.settings = await api.getSettings()
    },
    async save(s: Settings) {
      await api.saveSettings(s)
      this.settings = { ...s }
    },
  },
})
