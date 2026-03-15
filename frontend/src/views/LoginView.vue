<template>
  <div class="min-h-screen flex items-center justify-center bg-gradient-animated font-apple transition-colors duration-300">
    <div class="w-full max-w-md mx-4 animate-fade-in">
      <div :class="['p-10 rounded-2xl shadow-lg backdrop-blur-[40px] bg-white/72 dark:bg-[rgba(40,40,40,0.72)] border border-white/20 dark:border-white/10 relative will-change-transform', shaking ? 'animate-shake' : '']">
        <!-- 语言切换 - 右上角小按钮 -->
        <button
          type="button"
          class="absolute top-4 right-4 flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-gray-100/60 dark:bg-white/8 hover:bg-gray-200/80 dark:hover:bg-white/15 transition-all duration-200 group stagger-item stagger-1"
          @click="toggleLocale"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5 text-apple-gray group-hover:text-gray-600 dark:group-hover:text-gray-300 transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 008.716-6.747M12 21a9.004 9.004 0 01-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 017.843 4.582M12 3a8.997 8.997 0 00-7.843 4.582m15.686 0A11.953 11.953 0 0112 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0121 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0112 16.5a17.92 17.92 0 01-8.716-2.247m0 0A9 9 0 013 12c0-1.605.42-3.113 1.157-4.418" />
          </svg>
          <span class="text-[11px] font-medium text-apple-gray group-hover:text-gray-600 dark:group-hover:text-gray-300 transition-colors">{{ localeLabel }}</span>
        </button>

        <!-- Logo -->
        <div class="text-center mb-8 stagger-item stagger-2">
          <div class="flex justify-center mb-4">
            <AppLogo :size="56" :animated="true" />
          </div>
          <h1 class="text-3xl font-semibold text-gray-900 dark:text-white tracking-tight">
            {{ t('app.title') }}
          </h1>
          <p class="mt-2 text-sm text-apple-gray">
            {{ t('login.subtitle') }}
          </p>
        </div>

        <transition name="error-banner">
          <div
            v-if="requestError"
            class="mb-5 flex items-start gap-2.5 rounded-xl border border-rose-200/80 bg-rose-50/80 px-3.5 py-3 text-sm text-rose-700 dark:border-rose-400/30 dark:bg-rose-900/20 dark:text-rose-200"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="mt-0.5 h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zM12 15.75h.007v.008H12v-.008z" />
            </svg>
            <p class="leading-5">{{ requestError }}</p>
          </div>
        </transition>

        <!-- 表单 -->
        <form class="space-y-5" novalidate @submit.prevent="handleLogin">
          <!-- 密钥 -->
          <div class="stagger-item stagger-3">
            <div class="relative">
              <input
                v-model="key"
                :type="showKey ? 'text' : 'password'"
                :placeholder="t('login.key')"
                autocomplete="current-password"
                :disabled="loading"
                :class="['w-full px-4 py-3.5 pr-11 rounded-xl bg-gray-100/80 dark:bg-white/10 border text-gray-900 dark:text-white placeholder-apple-gray text-[15px] outline-none transition-all duration-200 disabled:opacity-50', fieldErrors.key || requestError ? 'border-red-400 focus:border-red-500 focus:ring-2 focus:ring-red-400/30' : 'border-transparent focus:border-apple-blue focus:ring-2 focus:ring-apple-blue/30 focus:animate-focus-glow']"
                @input="handleInputEdit"
              />
              <button
                type="button"
                class="absolute right-3 top-1/2 -translate-y-1/2 text-apple-gray hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
                tabindex="-1"
                @click="showKey = !showKey"
              >
                <svg v-if="!showKey" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178z" />
                  <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M3.98 8.223A10.477 10.477 0 001.934 12C3.226 16.338 7.244 19.5 12 19.5c.993 0 1.953-.138 2.863-.395M6.228 6.228A10.45 10.45 0 0112 4.5c4.756 0 8.773 3.162 10.065 7.498a10.523 10.523 0 01-4.293 5.774M6.228 6.228L3 3m3.228 3.228l3.65 3.65m7.894 7.894L21 21m-3.228-3.228l-3.65-3.65m0 0a3 3 0 10-4.243-4.243m4.242 4.242L9.88 9.88" />
                </svg>
              </button>
            </div>
            <p v-if="fieldErrors.key" class="mt-1.5 text-xs text-red-500">{{ fieldErrors.key }}</p>
          </div>

          <!-- 记住我 -->
          <label class="flex items-center gap-3 cursor-pointer select-none group stagger-item stagger-5">
            <button
              type="button"
              role="switch"
              :aria-checked="rememberMe"
              :disabled="loading"
              :class="['relative inline-flex h-[22px] w-[40px] shrink-0 rounded-full transition-colors duration-200 ease-in-out focus:outline-none disabled:opacity-50', rememberMe ? 'bg-apple-blue' : 'bg-gray-300 dark:bg-gray-600']"
              @click="rememberMe = !rememberMe"
            >
              <span
                :class="['pointer-events-none inline-block h-[18px] w-[18px] rounded-full bg-white shadow-sm transform transition-transform duration-200 ease-in-out mt-[2px]', rememberMe ? 'translate-x-[20px] ml-0' : 'translate-x-[2px]']"
              />
            </button>
            <span class="text-sm text-apple-gray group-hover:text-gray-600 dark:group-hover:text-gray-400 transition-colors">{{ t('login.remember') }}</span>
          </label>

          <!-- 提交按钮 -->
          <button
            type="submit"
            :disabled="loading || !formReady"
            :class="['btn-shine w-full py-3.5 rounded-xl text-white text-[15px] font-medium transition-all duration-300 flex items-center justify-center gap-2 stagger-item stagger-6', formReady ? 'bg-apple-blue hover:bg-apple-blue-hover active:scale-[0.98] shadow-md shadow-apple-blue/25' : 'bg-gray-300 dark:bg-gray-600 cursor-not-allowed']"
          >
            <svg v-if="loading" class="animate-spin h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
            {{ loading ? t('login.submitting') : t('login.submit') }}
          </button>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { useRouter } from 'vue-router'
import { apiClient } from '../api/client'
import { useAuth } from '../stores/auth'
import { useI18n } from '../i18n'
import { pathPrefix as prefix } from '../config'
import AppLogo from '../components/AppLogo.vue'

const router = useRouter()
const { setAuth } = useAuth()
const { t, toggleLocale, localeLabel } = useI18n()


const key = ref('')
const rememberMe = ref(false)
const showKey = ref(false)
const loading = ref(false)
const requestError = ref('')
const shaking = ref(false)
const fieldErrors = reactive({ key: '' })

const formReady = computed(() => key.value.trim() !== '')

function handleInputEdit() {
  if (fieldErrors.key) {
    fieldErrors.key = ''
  }
  if (requestError.value) {
    requestError.value = ''
  }
}

function validate(): boolean {
  fieldErrors.key = ''
  const valid = key.value.trim() !== ''
  if (!valid) {
    fieldErrors.key = t('login.error.key')
  }
  return valid
}

async function handleLogin() {
  if (loading.value) return
  requestError.value = ''
  if (!validate()) return

  loading.value = true

  try {
    const res = await apiClient.post('/login', {
      key: key.value
    })
    setAuth(res.data.token, rememberMe.value)
    router.push(`${prefix}/dashboard`)
  } catch (e: any) {
    requestError.value = getLoginErrorMessage(e)
    triggerShake()
  } finally {
    loading.value = false
  }
}

function triggerShake() {
  shaking.value = true
  setTimeout(() => { shaking.value = false }, 240)
}

function getLoginErrorMessage(errorPayload: any): string {
  const status = errorPayload?.response?.status as number | undefined
  const backendMessage = errorPayload?.response?.data?.error as string | undefined

  if (typeof backendMessage === 'string' && backendMessage.trim()) {
    return backendMessage
  }

  if (!status) {
    return t('login.error.network')
  }

  if (status === 400) return t('login.error.badRequest')
  if (status === 401) return t('login.error.unauthorized')
  if (status === 403) return t('login.error.forbidden')
  if (status === 404) return t('login.error.notFound')
  if (status === 429) return t('login.error.tooManyRequests')
  if (status >= 500 && status < 600) {
    if (status === 503) return t('login.error.unavailable')
    return t('login.error.server')
  }

  return t('login.error.default')
}
</script>
