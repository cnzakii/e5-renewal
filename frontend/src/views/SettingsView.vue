<template>
  <div class="space-y-6 animate-fade-in">
    <div class="flex items-start justify-between gap-3 flex-wrap">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white tracking-tight">
          {{ t('settings.title') }}
        </h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('settings.subtitle') }}
        </p>
      </div>
    </div>

    <!-- Notification Settings Card -->
    <div class="glass-card p-6 space-y-6">
      <h2 class="text-[15px] font-semibold text-gray-900 dark:text-white flex items-center gap-2">
        <svg class="w-4.5 h-4.5 text-apple-blue" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M14.857 17.082a23.848 23.848 0 005.454-1.31A8.967 8.967 0 0118 9.75v-.7V9A6 6 0 006 9v.75a8.967 8.967 0 01-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 01-5.714 0m5.714 0a3 3 0 11-5.714 0" />
        </svg>
        {{ t('settings.notification.title') }}
      </h2>

      <!-- Notification URL -->
      <div>
        <label class="form-label">{{ t('settings.notification.url') }}</label>
        <input
          v-model="form.url"
          type="text"
          :placeholder="t('settings.notification.url.placeholder')"
          class="form-input font-mono text-[13px]"
        />
        <p class="mt-1.5 text-[11px] text-gray-400 dark:text-gray-500">
          {{ t('settings.notification.url.hint') }}
        </p>
      </div>

      <!-- Notification Language -->
      <div>
        <label class="form-label">{{ t('settings.notification.language') }}</label>
        <div class="flex gap-2">
          <button
            v-for="lang in ['zh', 'en']"
            :key="lang"
            type="button"
            :class="[
              'px-4 py-2 rounded-xl text-[13px] font-medium border transition-all duration-200',
              form.language === lang
                ? 'bg-apple-blue text-white border-apple-blue/80 shadow-sm'
                : 'bg-gray-100/60 dark:bg-white/8 text-gray-600 dark:text-gray-300 border-gray-200/60 dark:border-white/10 hover:bg-gray-200/60 dark:hover:bg-white/12'
            ]"
            @click="form.language = lang"
          >
            {{ t(`settings.notification.language.${lang}`) }}
          </button>
        </div>
      </div>

      <!-- Notification Conditions -->
      <div class="space-y-3">
        <label class="form-label">{{ t('settings.notification.conditions') }}</label>

        <!-- Auth Expiry -->
        <div class="rounded-xl bg-gray-50/50 dark:bg-white/3 border border-gray-100/50 dark:border-white/5 overflow-hidden">
          <div class="flex items-center justify-between px-4 py-3">
            <div class="flex items-center gap-2.5">
              <svg class="w-4 h-4 text-amber-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span class="text-[13px] font-medium text-gray-700 dark:text-gray-300">{{ t('settings.notification.onAuthExpiry') }}</span>
            </div>
            <button
              type="button"
              :class="[
                'relative w-11 h-6 rounded-full transition-colors duration-200 focus:outline-none',
                form.on_auth_expiry ? 'bg-apple-blue' : 'bg-gray-300 dark:bg-gray-600'
              ]"
              @click="form.on_auth_expiry = !form.on_auth_expiry"
            >
              <span :class="['absolute top-0.5 left-0.5 w-5 h-5 rounded-full bg-white shadow-sm transition-transform duration-200', form.on_auth_expiry ? 'translate-x-5' : 'translate-x-0']" />
            </button>
          </div>
          <Transition name="field-slide">
            <div v-if="form.on_auth_expiry" class="px-4 pb-3 pt-0">
              <div class="flex items-center gap-2">
                <label class="text-[12px] text-gray-500 dark:text-gray-400 shrink-0">{{ t('settings.notification.expiryDays') }}</label>
                <input
                  v-model.number="form.expiry_days_before"
                  type="number"
                  min="1"
                  max="365"
                  class="form-input w-20 text-center tabular-nums text-[13px]"
                />
              </div>
            </div>
          </Transition>
        </div>

        <!-- All Tasks Failed -->
        <div class="flex items-center justify-between px-4 py-3 rounded-xl bg-gray-50/50 dark:bg-white/3 border border-gray-100/50 dark:border-white/5">
          <div class="flex items-center gap-2.5">
            <svg class="w-4 h-4 text-red-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
            </svg>
            <span class="text-[13px] font-medium text-gray-700 dark:text-gray-300">{{ t('settings.notification.onTaskAllFailed') }}</span>
          </div>
          <button
            type="button"
            :class="[
              'relative w-11 h-6 rounded-full transition-colors duration-200 focus:outline-none',
              form.on_task_all_failed ? 'bg-apple-blue' : 'bg-gray-300 dark:bg-gray-600'
            ]"
            @click="form.on_task_all_failed = !form.on_task_all_failed"
          >
            <span :class="['absolute top-0.5 left-0.5 w-5 h-5 rounded-full bg-white shadow-sm transition-transform duration-200', form.on_task_all_failed ? 'translate-x-5' : 'translate-x-0']" />
          </button>
        </div>

        <!-- Health Below Threshold -->
        <div class="rounded-xl bg-gray-50/50 dark:bg-white/3 border border-gray-100/50 dark:border-white/5 overflow-hidden">
          <div class="flex items-center justify-between px-4 py-3">
            <div class="flex items-center gap-2.5">
              <svg class="w-4 h-4 text-emerald-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M21 8.25c0-2.485-2.099-4.5-4.688-4.5-1.935 0-3.597 1.126-4.312 2.733-.715-1.607-2.377-2.733-4.313-2.733C5.1 3.75 3 5.765 3 8.25c0 7.22 9 12 9 12s9-4.78 9-12z" />
              </svg>
              <span class="text-[13px] font-medium text-gray-700 dark:text-gray-300">{{ t('settings.notification.onHealthLow') }}</span>
            </div>
            <button
              type="button"
              :class="[
                'relative w-11 h-6 rounded-full transition-colors duration-200 focus:outline-none',
                form.on_health_low ? 'bg-apple-blue' : 'bg-gray-300 dark:bg-gray-600'
              ]"
              @click="form.on_health_low = !form.on_health_low"
            >
              <span :class="['absolute top-0.5 left-0.5 w-5 h-5 rounded-full bg-white shadow-sm transition-transform duration-200', form.on_health_low ? 'translate-x-5' : 'translate-x-0']" />
            </button>
          </div>
          <Transition name="field-slide">
            <div v-if="form.on_health_low" class="px-4 pb-3 pt-0">
              <div class="flex items-center gap-2">
                <label class="text-[12px] text-gray-500 dark:text-gray-400 shrink-0">{{ t('settings.notification.healthThreshold') }}</label>
                <input
                  v-model.number="form.health_threshold"
                  type="number"
                  min="1"
                  max="100"
                  class="form-input w-20 text-center tabular-nums text-[13px]"
                />
                <span class="text-[12px] text-gray-400 dark:text-gray-500">%</span>
              </div>
            </div>
          </Transition>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="flex items-center justify-end gap-3 pt-2">
        <button
          :disabled="testing || !saved"
          :title="!saved ? t('settings.notification.test.saveFirst') : ''"
          class="inline-flex items-center gap-1.5 px-4 py-2.5 rounded-xl text-[13px] font-medium text-gray-600 dark:text-gray-300 bg-gray-100/60 dark:bg-white/8 hover:bg-gray-200/60 dark:hover:bg-white/12 border border-gray-200/60 dark:border-white/10 transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
          @click="testNotification"
        >
          <svg :class="['w-3.5 h-3.5', testing ? 'animate-spin' : '']" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 12L3.269 3.126A59.768 59.768 0 0121.485 12 59.77 59.77 0 013.27 20.876L5.999 12zm0 0h7.5" />
          </svg>
          {{ testing ? t('settings.notification.test.sending') : t('settings.notification.test') }}
        </button>
        <button
          :disabled="saving"
          class="inline-flex items-center gap-1.5 px-5 py-2.5 rounded-xl text-[13px] font-medium text-white bg-apple-blue hover:bg-apple-blue-hover border border-apple-blue/80 shadow-md shadow-apple-blue/20 transition-all duration-200 disabled:opacity-60 btn-shine"
          @click="saveSettings"
        >
          {{ saving ? t('settings.notification.saving') : t('settings.notification.save') }}
        </button>
      </div>
    </div>

    <!-- Toast -->
    <Teleport to="body">
      <Transition name="toast">
        <div
          v-if="toast.show"
          :class="[
            'fixed top-6 right-6 z-[100] flex items-center gap-2 px-4 py-3 rounded-xl text-[13px] font-medium backdrop-blur-xl shadow-lg border',
            toast.type === 'success'
              ? 'bg-emerald-50/90 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-300 border-emerald-200/60 dark:border-emerald-500/20'
              : 'bg-red-50/90 dark:bg-red-900/30 text-red-700 dark:text-red-300 border-red-200/60 dark:border-red-500/20'
          ]"
        >
          <svg v-if="toast.type === 'success'" class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd" />
          </svg>
          <svg v-else class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clip-rule="evenodd" />
          </svg>
          {{ toast.message }}
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from '../i18n'
import { apiClient } from '../api/client'

const { t } = useI18n()

interface NotificationConfig {
  url: string
  language: string
  on_auth_expiry: boolean
  expiry_days_before: number
  on_task_all_failed: boolean
  on_health_low: boolean
  health_threshold: number
}

const form = reactive<NotificationConfig>({
  url: '',
  language: 'zh',
  on_auth_expiry: false,
  expiry_days_before: 7,
  on_task_all_failed: false,
  on_health_low: false,
  health_threshold: 50,
})

const saving = ref(false)
const testing = ref(false)
const saved = ref(false)

const toast = reactive({ show: false, type: 'success' as 'success' | 'error', message: '' })
let toastTimer: ReturnType<typeof setTimeout> | undefined

function showToast(type: 'success' | 'error', message: string) {
  clearTimeout(toastTimer)
  toast.show = true
  toast.type = type
  toast.message = message
  toastTimer = setTimeout(() => { toast.show = false }, 2500)
}

async function fetchSettings() {
  try {
    const res = await apiClient.get('/settings/notification')
    Object.assign(form, res.data)
    saved.value = !!form.url
  } catch {
    // use defaults
  }
}

async function saveSettings() {
  if (saving.value) return
  saving.value = true
  try {
    await apiClient.put('/settings/notification', { ...form })
    saved.value = true
    showToast('success', t('settings.notification.save.success'))
  } catch {
    showToast('error', t('settings.notification.save.error'))
  } finally {
    saving.value = false
  }
}

async function testNotification() {
  if (testing.value) return
  if (!saved.value) {
    showToast('error', t('settings.notification.test.saveFirst'))
    return
  }
  testing.value = true
  try {
    await apiClient.post('/settings/notification/test')
    showToast('success', t('settings.notification.test.success'))
  } catch (err: any) {
    const detail = err?.response?.data?.error
    showToast('error', detail || t('settings.notification.test.error'))
  } finally {
    testing.value = false
  }
}

onMounted(() => fetchSettings())
</script>

<style scoped>
.glass-card {
  border-radius: 16px;
  backdrop-filter: blur(40px);
  -webkit-backdrop-filter: blur(40px);
  background: rgba(255, 255, 255, 0.55);
  border: 1px solid rgba(255, 255, 255, 0.2);
  transition: box-shadow 0.3s ease, border-color 0.3s ease, background 0.3s ease;
  .dark & {
    background: rgba(40, 40, 40, 0.55);
    border-color: rgba(255, 255, 255, 0.08);
  }
}

.form-label {
  display: block;
  font-size: 12px;
  font-weight: 600;
  color: #6b7280;
  margin-bottom: 6px;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  .dark & {
    color: #9ca3af;
  }
}

.form-input {
  width: 100%;
  padding: 10px 14px;
  border-radius: 12px;
  font-size: 14px;
  background: rgba(255, 255, 255, 0.6);
  border: 1.5px solid rgba(0, 0, 0, 0.06);
  color: #1f2937;
  transition: all 0.2s;
  outline: none;
  .dark & {
    background: rgba(255, 255, 255, 0.05);
    border-color: rgba(255, 255, 255, 0.08);
    color: #f3f4f6;
  }
}
.form-input::placeholder {
  color: #9ca3af;
}
.form-input {
  .dark &::placeholder {
    color: #6b7280;
  }
}
.form-input:focus {
  border-color: #0071e3;
  box-shadow: 0 0 0 3px rgba(0, 113, 227, 0.12);
  background: rgba(255, 255, 255, 0.8);
}
.form-input {
  .dark &:focus {
    background: rgba(255, 255, 255, 0.08);
  }
}

/* Field slide transition */
.field-slide-enter-active {
  transition: all 0.3s ease;
  overflow: hidden;
}
.field-slide-leave-active {
  transition: all 0.2s ease;
  overflow: hidden;
}
.field-slide-enter-from {
  opacity: 0;
  max-height: 0;
  transform: translateY(-8px);
}
.field-slide-enter-to {
  max-height: 100px;
}
.field-slide-leave-from {
  max-height: 100px;
}
.field-slide-leave-to {
  opacity: 0;
  max-height: 0;
  transform: translateY(-8px);
}

/* Toast */
.toast-enter-active {
  transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.toast-leave-active {
  transition: all 0.2s ease;
}
.toast-enter-from {
  opacity: 0;
  transform: translateY(-12px) scale(0.95);
}
.toast-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
