<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Plus } from '@lucide/vue'
import AccountCard from '../components/AccountCard.vue'
import LoginDialog from '../components/LoginDialog.vue'
import { useAccountsStore } from '../stores/accounts'
import { useToast } from '../composables/toast'
import { api } from '../api/bindings'

const accounts = useAccountsStore()
const toast = useToast()
const dialogOpen = ref(false)
const authUrl = ref('')

onMounted(async () => {
  await accounts.refresh()
})

async function openLogin() {
  try {
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
}
</script>

<template>
  <section class="p-5">
    <div class="flex items-center justify-between mb-4">
      <div>
        <h1 class="font-semibold">账号</h1>
        <p class="text-xs text-slate-500 mt-0.5">
          自动签到，电脑开机后自动补签
        </p>
      </div>
      <button class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium cursor-pointer" @click="openLogin">
        <Plus class="w-4 h-4" /> 添加账号
      </button>
    </div>

    <div v-if="accounts.accounts.length" class="space-y-3">
      <AccountCard
        v-for="a in accounts.accounts"
        :key="a.id"
        :account="a"
        :checking="accounts.checkedInId === a.id"
        @checkin="accounts.checkinNow"
        @refresh-quota="accounts.refreshQuota"
        @remove="accounts.remove"
      />
    </div>
    <div v-else class="text-sm text-slate-400 text-center py-16">还没有账号，点击「添加账号」开始。</div>

    <LoginDialog :open="dialogOpen" :auth-url="authUrl" @close="dialogOpen = false" @success="onSuccess" />
  </section>
</template>
