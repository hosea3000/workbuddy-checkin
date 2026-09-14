<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { CheckCheck, Plus } from '@lucide/vue'
import LoginDialog from '../components/LoginDialog.vue'
import { useAccountsStore } from '../stores/accounts'
import { useToast } from '../composables/toast'
import { api } from '../api/bindings'

const router = useRouter()
const accounts = useAccountsStore()
const toast = useToast()
const dialogOpen = ref(false)
const authUrl = ref('')
const autoStart = ref(true)

onMounted(async () => {
  const s = await api.getSettings()
  autoStart.value = s.autoStart
})

async function openLogin() {
  try {
    if (autoStart.value !== (await api.getSettings()).autoStart) {
      const s = await api.getSettings()
      await api.saveSettings({ ...s, autoStart: autoStart.value })
    }
    const start = await api.startLogin()
    authUrl.value = start.authUrl
    dialogOpen.value = true
  } catch (e) {
    toast.push(`启动登录失败：${e}`, 'error')
  }
}

async function onSuccess() {
  dialogOpen.value = false
  await accounts.refresh()
  router.replace({ name: 'accounts' })
}
</script>

<template>
  <section class="h-full grid place-items-center p-5">
    <div class="text-center max-w-sm">
      <div class="w-14 h-14 mx-auto rounded-2xl bg-indigo-600 text-white grid place-items-center shadow-lg shadow-indigo-600/20">
        <CheckCheck class="w-7 h-7" />
      </div>
      <h1 class="text-lg font-semibold mt-5">每天自动帮你签到 CodeBuddy</h1>
      <p class="text-sm text-slate-500 mt-2 leading-relaxed">
        添加账号后，工具会常驻托盘，到点自动签到、开机自动补签，你什么都不用做。
      </p>
      <button class="inline-flex items-center gap-1.5 px-6 py-2.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium cursor-pointer mx-auto mt-6" @click="openLogin">
        <Plus class="w-4 h-4" /> 添加账号
      </button>
      <label class="mt-5 flex items-center justify-center gap-2 text-sm text-slate-500 cursor-pointer">
        <input v-model="autoStart" type="checkbox" class="accent-indigo-600" />
        开机时自动启动（推荐）
      </label>
    </div>

    <LoginDialog :open="dialogOpen" :auth-url="authUrl" @close="dialogOpen = false" @success="onSuccess" />
  </section>
</template>
