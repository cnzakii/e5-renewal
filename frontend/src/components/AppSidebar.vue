<template>
  <aside
    :class="[
      'fixed top-0 left-0 h-full z-30 flex flex-col overflow-hidden transition-[width,background-color,border-color] duration-300 ease-out backdrop-blur-[40px] border-r',
      props.collapsed ? 'w-16' : 'w-56',
      'bg-white/60 dark:bg-[rgba(30,30,30,0.65)] border-white/20 dark:border-white/10'
    ]"
  >
    <!-- Logo -->
    <div class="flex items-center gap-3 px-4 h-16 shrink-0">
      <AppLogo :size="32" />
      <span
        :class="[
          'text-base font-semibold text-gray-900 dark:text-white whitespace-nowrap overflow-hidden transition-[max-width,opacity] duration-200',
          props.collapsed ? 'max-w-0 opacity-0' : 'max-w-[120px] opacity-100'
        ]"
      >Renewal</span>
    </div>

    <!-- Nav Items -->
    <nav class="flex-1 px-2 py-4 space-y-1">
      <router-link
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        :class="[
          'group flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all duration-200 cursor-pointer select-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-apple-blue/40',
          isActive(item.path)
            ? 'bg-apple-blue/12 text-apple-blue dark:bg-apple-blue/20'
            : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100/60 dark:hover:bg-white/8'
        ]"
      >
        <component :is="item.icon" class="w-5 h-5 shrink-0 transition-transform duration-200 group-hover:scale-110 group-active:scale-95" />
        <span
          :class="[
            'whitespace-nowrap overflow-hidden transition-[max-width,opacity] duration-200',
            props.collapsed ? 'max-w-0 opacity-0' : 'max-w-[120px] opacity-100'
          ]"
        >{{ t(item.label) }}</span>
      </router-link>
    </nav>

    <!-- Bottom Actions -->
    <div class="px-2 py-4 space-y-1 border-t border-white/15 dark:border-white/8">
      <!-- Collapse toggle -->
      <button
        class="group flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100/60 dark:hover:bg-white/8 transition-all duration-200 w-full cursor-pointer select-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-apple-blue/40"
        @click="emit('update:collapsed', !props.collapsed)"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="w-5 h-5 shrink-0 transition-transform duration-300 group-hover:scale-110 group-active:scale-95"
          :class="props.collapsed ? 'rotate-180' : ''"
          fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
        </svg>
        <span
          :class="[
            'whitespace-nowrap overflow-hidden transition-[max-width,opacity] duration-200',
            props.collapsed ? 'max-w-0 opacity-0' : 'max-w-[120px] opacity-100'
          ]"
        ></span>
      </button>

      <!-- Dark mode -->
      <button
        class="group flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100/60 dark:hover:bg-white/8 transition-all duration-200 w-full cursor-pointer select-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-apple-blue/40"
        @click="toggleDark()"
      >
        <svg v-if="isDark" xmlns="http://www.w3.org/2000/svg" class="w-5 h-5 shrink-0 transition-transform duration-200 group-hover:scale-110 group-active:scale-95" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 3v2.25m6.364.386l-1.591 1.591M21 12h-2.25m-.386 6.364l-1.591-1.591M12 18.75V21m-4.773-4.227l-1.591 1.591M5.25 12H3m4.227-4.773L5.636 5.636M15.75 12a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0z" />
        </svg>
        <svg v-else xmlns="http://www.w3.org/2000/svg" class="w-5 h-5 shrink-0 transition-transform duration-200 group-hover:scale-110 group-active:scale-95" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M21.752 15.002A9.718 9.718 0 0118 15.75c-5.385 0-9.75-4.365-9.75-9.75 0-1.33.266-2.597.748-3.752A9.753 9.753 0 003 11.25C3 16.635 7.365 21 12.75 21a9.753 9.753 0 009.002-5.998z" />
        </svg>
        <span
          :class="[
            'whitespace-nowrap overflow-hidden transition-[max-width,opacity] duration-200',
            props.collapsed ? 'max-w-0 opacity-0' : 'max-w-[120px] opacity-100'
          ]"
        >{{ isDark ? t('sidebar.theme.light') : t('sidebar.theme.dark') }}</span>
      </button>

      <!-- Language -->
      <button
        class="group flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100/60 dark:hover:bg-white/8 transition-all duration-200 w-full cursor-pointer select-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-apple-blue/40"
        @click="toggleLocale"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5 shrink-0 transition-transform duration-200 group-hover:scale-110 group-active:scale-95" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 008.716-6.747M12 21a9.004 9.004 0 01-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 017.843 4.582M12 3a8.997 8.997 0 00-7.843 4.582m15.686 0A11.953 11.953 0 0112 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0121 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0112 16.5a17.92 17.92 0 01-8.716-2.247m0 0A9 9 0 013 12c0-1.605.42-3.113 1.157-4.418" />
        </svg>
        <span
          :class="[
            'whitespace-nowrap overflow-hidden transition-[max-width,opacity] duration-200',
            props.collapsed ? 'max-w-0 opacity-0' : 'max-w-[120px] opacity-100'
          ]"
        >{{ localeLabel }}</span>
      </button>

      <!-- Logout -->
      <button
        class="group flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm text-red-500 dark:text-red-400 hover:text-red-600 dark:hover:text-red-300 hover:bg-red-50/60 dark:hover:bg-red-900/15 transition-all duration-200 w-full cursor-pointer select-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400/40"
        @click="handleLogout"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5 shrink-0 transition-transform duration-200 group-hover:scale-110 group-active:scale-95" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15m3 0l3-3m0 0l-3-3m3 3H9" />
        </svg>
        <span
          :class="[
            'whitespace-nowrap overflow-hidden transition-[max-width,opacity] duration-200',
            props.collapsed ? 'max-w-0 opacity-0' : 'max-w-[120px] opacity-100'
          ]"
        >{{ t('nav.logout') }}</span>
      </button>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, h, type FunctionalComponent } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useDark, useToggle } from '@vueuse/core'
import { useAuth } from '../stores/auth'
import { useI18n } from '../i18n'
import AppLogo from './AppLogo.vue'

const route = useRoute()
const router = useRouter()
const { clearAuth } = useAuth()
const { t, toggleLocale, localeLabel } = useI18n()

const props = defineProps<{ collapsed: boolean }>()
const emit = defineEmits<{ 'update:collapsed': [value: boolean] }>()

const prefix = import.meta.env.VITE_PATH_PREFIX || ''
const isDark = useDark({ storageKey: 'e5-theme' })
const toggleDark = useToggle(isDark)

function isActive(path: string) {
  return route.path === path
}

function handleLogout() {
  clearAuth()
  router.push(`${prefix}/login`)
}

// Icon components
const IconDashboard: FunctionalComponent = () =>
  h('svg', { xmlns: 'http://www.w3.org/2000/svg', fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [
    h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z' })
  ])

const IconAccounts: FunctionalComponent = () =>
  h('svg', { xmlns: 'http://www.w3.org/2000/svg', fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [
    h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M15 19.128a9.38 9.38 0 002.625.372 9.337 9.337 0 004.121-.952 4.125 4.125 0 00-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 018.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0111.964-3.07M12 6.375a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zm8.25 2.25a2.625 2.625 0 11-5.25 0 2.625 2.625 0 015.25 0z' })
  ])

const IconLogs: FunctionalComponent = () =>
  h('svg', { xmlns: 'http://www.w3.org/2000/svg', fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [
    h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z' })
  ])

const IconSettings: FunctionalComponent = () =>
  h('svg', { xmlns: 'http://www.w3.org/2000/svg', fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [
    h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281z' }),
    h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M15 12a3 3 0 11-6 0 3 3 0 016 0z' })
  ])

const navItems = computed(() => [
  { path: `${prefix}/dashboard`, label: 'nav.dashboard', icon: IconDashboard },
  { path: `${prefix}/accounts`, label: 'nav.accounts', icon: IconAccounts },
  { path: `${prefix}/logs`, label: 'nav.logs', icon: IconLogs },
  { path: `${prefix}/settings`, label: 'nav.settings', icon: IconSettings },
])
</script>
