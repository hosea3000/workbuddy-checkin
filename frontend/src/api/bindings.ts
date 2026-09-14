// Wails 运行时绑定封装：统一错误 → 由调用方决定 toast。
// 生成物：wailsjs/go/main/App.js + wailsjs/go/models.ts（wails generate module）。
import * as App from '../../wailsjs/go/main/App'
import type { model } from '../../wailsjs/go/models'

export type AccountView = model.AccountView
export type CheckinResult = model.CheckinResult
export type CheckinSummary = model.CheckinSummary
export type Settings = model.Settings
export type LoginStart = model.LoginStart
export type LoginStatus = model.LoginStatus
export type QuotaView = model.QuotaView
export type UpdateCheckResult = model.UpdateCheckResult
export type UpdateDownloadEvent = model.UpdateDownloadEvent
export type PendingUpdateInfo = model.PendingUpdateInfo

export const api = {
  listAccounts: () => App.ListAccounts(),
  checkinSummary: () => App.CheckinSummary(),
  startLogin: () => App.StartLogin(),
  loginStatus: () => App.LoginStatus(),
  cancelLogin: () => App.CancelLogin(),
  openAuthUrl: (url: string) => App.OpenAuthURL(url),
  checkinNow: (id: string) => App.CheckinNow(id),
  checkinAll: () => App.CheckinAll(),
  refreshQuota: (id: string) => App.RefreshQuota(id),
  deleteAccount: (id: string) => App.DeleteAccount(id),
  getSettings: () => App.GetSettings(),
  saveSettings: (s: Settings) => App.SaveSettings(s),
  openDataDir: () => App.OpenDataDir(),
  hasAccounts: () => App.HasAccounts(),
  getVersion: () => App.GetVersion(),
  checkUpdate: () => App.CheckUpdate(),
  downloadAndApplyUpdate: () => App.DownloadAndApplyUpdate(),
  applyUpdateAndRestart: () => App.ApplyUpdateAndRestart(),
  updateProgress: () => App.UpdateProgress(),
  pendingUpdateInfo: () => App.PendingUpdateInfo(),
}
