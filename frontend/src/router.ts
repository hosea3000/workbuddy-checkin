import { createRouter, createWebHashHistory } from 'vue-router'
import AccountsView from './views/AccountsView.vue'
import ProxyView from './views/ProxyView.vue'
import SettingsView from './views/SettingsView.vue'
import WelcomeView from './views/WelcomeView.vue'

export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/accounts' },
    { path: '/accounts', name: 'accounts', component: AccountsView },
    { path: '/proxy', name: 'proxy', component: ProxyView },
    { path: '/settings', name: 'settings', component: SettingsView },
    { path: '/welcome', name: 'welcome', component: WelcomeView },
  ],
})
