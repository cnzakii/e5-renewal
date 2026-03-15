<template>
  <div v-if="authType === 'auth_code'" class="space-y-3">
    <label class="form-label">Refresh Token</label>

    <!-- Mode selector — segmented controller -->
    <div class="grid grid-cols-3 gap-1.5 p-1 rounded-xl bg-gray-100/60 dark:bg-white/5">
      <button
        v-for="m in modes"
        :key="m.value"
        type="button"
        :class="[
          'px-3 py-1.5 rounded-lg text-[12px] font-medium transition-all duration-200',
          mode === m.value
            ? 'bg-white dark:bg-white/12 text-gray-900 dark:text-white shadow-sm'
            : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
        ]"
        @click="mode = m.value"
      >
        {{ m.label }}
      </button>
    </div>

    <!-- ═══ Auto mode ═══ -->
    <div v-if="mode === 'auto'" class="space-y-3">
      <div>
        <label class="sub-label">{{ t('accounts.form.refreshToken.auto.redirectUri') }}</label>
        <div class="form-input !bg-gray-50/80 dark:!bg-white/[0.03] !text-gray-500 dark:!text-gray-400 !cursor-default !border-dashed font-mono text-[12px] truncate select-all">
          {{ autoRedirectUri }}
        </div>
        <p class="form-hint">{{ t('accounts.form.refreshToken.auto.redirectUri.hint') }}</p>
      </div>

      <button
        type="button"
        :disabled="!canAuthorize"
        class="oauth-btn"
        data-testid="auto-authorize-btn"
        @click="startAutoOAuth"
      >
        <svg class="w-3.5 h-3.5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M13.5 6H5.25A2.25 2.25 0 003 8.25v10.5A2.25 2.25 0 005.25 21h10.5A2.25 2.25 0 0018 18.75V10.5m-10.5 6L21 3m0 0h-5.25M21 3v5.25" />
        </svg>
        {{ t('accounts.form.refreshToken.auto.authorize') }}
      </button>

      <p v-if="autoSuccess" class="flex items-center gap-1.5 text-[12px] text-emerald-600 dark:text-emerald-400 font-medium animate-fade-in">
        <svg class="w-3.5 h-3.5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd" />
        </svg>
        {{ t('accounts.form.refreshToken.auto.success') }}
      </p>
    </div>

    <!-- ═══ Manual mode ═══ -->
    <div v-if="mode === 'manual'" class="space-y-3">
      <!-- Redirect URI + Authorize -->
      <div>
        <label class="sub-label">{{ t('accounts.form.refreshToken.manual.redirectUri') }}</label>
        <input
          v-model="manualRedirectUri"
          type="text"
          :placeholder="t('accounts.form.refreshToken.manual.redirectUri.placeholder')"
          class="form-input font-mono"
          :class="{ 'form-input-error': redirectUriError }"
        />
        <p v-if="redirectUriError" class="form-error">{{ redirectUriError }}</p>
      </div>

      <button
        type="button"
        :disabled="!canAuthorize || !manualRedirectUri.trim()"
        class="oauth-btn"
        @click="openManualAuth"
      >
        <svg class="w-3.5 h-3.5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M13.5 6H5.25A2.25 2.25 0 003 8.25v10.5A2.25 2.25 0 005.25 21h10.5A2.25 2.25 0 0018 18.75V10.5m-10.5 6L21 3m0 0h-5.25M21 3v5.25" />
        </svg>
        {{ t('accounts.form.refreshToken.manual.authorize') }}
      </button>

      <!-- Thin divider -->
      <div class="h-px bg-gray-200/60 dark:bg-white/6"></div>

      <!-- Callback URL + Get Token -->
      <div>
        <label class="sub-label">{{ t('accounts.form.refreshToken.manual.callbackUrl') }}</label>
        <input
          v-model="callbackUrl"
          type="text"
          :placeholder="t('accounts.form.refreshToken.manual.callbackUrl.placeholder')"
          class="form-input font-mono"
          :class="{ 'form-input-error': callbackUrlError }"
        />
        <p v-if="callbackUrlError" class="form-error">{{ callbackUrlError }}</p>
      </div>

      <button
        type="button"
        :disabled="!callbackUrl.trim() || exchanging"
        class="oauth-btn oauth-btn-outline"
        data-testid="manual-exchange-btn"
        @click="exchangeToken"
      >
        <svg v-if="exchanging" class="w-3.5 h-3.5 animate-spin" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        <svg v-else class="w-3.5 h-3.5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M7.5 21L3 16.5m0 0L7.5 12M3 16.5h13.5m0-13.5L21 7.5m0 0L16.5 12M21 7.5H7.5" />
        </svg>
        {{ t('accounts.form.refreshToken.manual.exchange') }}
      </button>

      <p v-if="exchangeError" class="form-error">{{ exchangeError }}</p>

      <p v-if="manualSuccess" class="flex items-center gap-1.5 text-[12px] text-emerald-600 dark:text-emerald-400 font-medium animate-fade-in">
        <svg class="w-3.5 h-3.5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd" />
        </svg>
        {{ t('accounts.form.refreshToken.auto.success') }}
      </p>
    </div>

    <!-- ═══ Direct mode ═══ -->
    <div v-if="mode === 'direct'">
      <textarea
        :value="refreshToken"
        :placeholder="t('accounts.form.refreshToken.placeholder')"
        rows="4"
        class="form-input resize-none font-mono text-[12px] leading-relaxed"
        data-testid="direct-textarea"
        @input="$emit('update:refreshToken', ($event.target as HTMLTextAreaElement).value)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'
import { useI18n } from '../i18n'
import { apiClient } from '../api/client'
import { pathPrefix } from '../config'

const props = defineProps<{
  clientId: string
  clientSecret: string
  tenantId: string
  refreshToken: string
  authType: string
}>()

const emit = defineEmits<{
  (e: 'update:refreshToken', value: string): void
}>()

const { t } = useI18n()

type Mode = 'auto' | 'manual' | 'direct'
const mode = ref<Mode>('auto')

const modes = computed(() => [
  { value: 'auto' as const, label: t('accounts.form.refreshToken.mode.auto') },
  { value: 'manual' as const, label: t('accounts.form.refreshToken.mode.manual') },
  { value: 'direct' as const, label: t('accounts.form.refreshToken.mode.direct') },
])

const autoRedirectUri = computed(() => {
  return `${window.location.origin}${pathPrefix}/api/oauth/callback`
})

const canAuthorize = computed(() => {
  return props.clientId.trim() && props.tenantId.trim()
})

// --- Auto mode ---
let oauthPopup: Window | null = null
let oauthListener: ((e: MessageEvent) => void) | null = null
const autoSuccess = ref(false)

function startAutoOAuth() {
  if (!canAuthorize.value) return
  if (oauthListener) window.removeEventListener('message', oauthListener)
  autoSuccess.value = false

  apiClient.post('/oauth/authorize', {
    client_id: props.clientId.trim(),
    client_secret: props.clientSecret.trim(),
    tenant_id: props.tenantId.trim(),
  }).then(res => {
    const authorizeUrl = res.data.authorize_url
    oauthPopup = window.open(authorizeUrl, 'e5-oauth', 'width=600,height=700,scrollbars=yes')

    oauthListener = (e: MessageEvent) => {
      if (e.origin !== window.location.origin) return
      if (e.data?.type !== 'e5-oauth-result') return
      window.removeEventListener('message', oauthListener!)
      oauthListener = null

      const { status, payload } = e.data.data
      if (status === 'success') {
        try {
          const tokenData = JSON.parse(payload)
          emit('update:refreshToken', tokenData.refresh_token || '')
          autoSuccess.value = true
        } catch {
          // parse error
        }
      }
    }
    window.addEventListener('message', oauthListener)
  }).catch(() => {
    // error handled silently
  })
}

onUnmounted(() => {
  if (oauthListener) window.removeEventListener('message', oauthListener)
  oauthPopup?.close()
})

// --- Manual mode ---
const manualRedirectUri = ref('')
const callbackUrl = ref('')
const exchanging = ref(false)
const exchangeError = ref('')
const manualSuccess = ref(false)
const redirectUriError = ref('')
const callbackUrlError = ref('')

function validateRedirectUri(uri: string): boolean {
  redirectUriError.value = ''
  if (!uri.trim()) return false
  try {
    new URL(uri)
    return true
  } catch {
    redirectUriError.value = t('accounts.form.refreshToken.redirectUri.invalid')
    return false
  }
}

function openManualAuth() {
  if (!canAuthorize.value) return
  if (!validateRedirectUri(manualRedirectUri.value)) return

  apiClient.post('/oauth/authorize', {
    client_id: props.clientId.trim(),
    client_secret: props.clientSecret.trim(),
    tenant_id: props.tenantId.trim(),
    redirect_uri: manualRedirectUri.value.trim(),
  }).then(res => {
    const authorizeUrl = res.data.authorize_url
    window.open(authorizeUrl, '_blank')
    exchangeError.value = ''
  }).catch(() => {
    exchangeError.value = t('accounts.form.refreshToken.oauth.failed')
  })
}

function exchangeToken() {
  if (!callbackUrl.value.trim()) return
  callbackUrlError.value = ''

  try {
    new URL(callbackUrl.value.trim())
  } catch {
    callbackUrlError.value = t('accounts.form.refreshToken.callbackUrl.invalid')
    return
  }

  exchanging.value = true
  exchangeError.value = ''
  manualSuccess.value = false

  apiClient.post('/oauth/exchange', {
    callback_url: callbackUrl.value.trim(),
  }).then(res => {
    emit('update:refreshToken', res.data.refresh_token || '')
    exchangeError.value = ''
    manualSuccess.value = true
  }).catch(err => {
    exchangeError.value = err?.response?.data?.error || t('accounts.form.refreshToken.oauth.failed')
  }).finally(() => {
    exchanging.value = false
  })
}
</script>

<style scoped>
/* Form styles — duplicated from AccountFormDialog because Vue scoped styles don't penetrate child components */
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
.form-input-error {
  border-color: #ef4444;
}
.form-input-error:focus {
  border-color: #ef4444;
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.12);
}

.form-error {
  margin-top: 4px;
  font-size: 12px;
  color: #ef4444;
}

/* Sub-label — lighter weight than form-label, for fields within the panel */
.sub-label {
  display: block;
  font-size: 11px;
  font-weight: 500;
  color: #b0b0b5;
  margin-bottom: 4px;
  .dark & {
    color: #6b7280;
  }
}

/* Hint text — matches form-error but neutral color */
.form-hint {
  margin-top: 4px;
  font-size: 11px;
  color: #9ca3af;
  .dark & {
    color: #6b7280;
  }
}

/* OAuth action buttons — uses the same visual language as the "Add Account" button */
.oauth-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 100%;
  padding: 10px 16px;
  border-radius: 12px;
  font-size: 13px;
  font-weight: 500;
  color: white;
  background: #0071e3;
  border: none;
  cursor: pointer;
  transition: all 0.2s ease;
  .dark & {
    background: rgba(0, 113, 227, 0.8);
  }
  .dark &:hover:not(:disabled) {
    background: rgba(0, 113, 227, 0.9);
  }
}
.oauth-btn:hover:not(:disabled) {
  background: #0077ed;
  box-shadow: 0 4px 12px rgba(0, 113, 227, 0.2);
}
.oauth-btn:active:not(:disabled) {
  transform: scale(0.98);
}
.oauth-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

/* Outline variant for secondary actions */
.oauth-btn-outline {
  background: rgba(0, 113, 227, 0.06);
  color: #0071e3;
  border: 1.5px solid rgba(0, 113, 227, 0.15);
  .dark & {
    background: rgba(0, 113, 227, 0.08);
    border-color: rgba(0, 113, 227, 0.12);
    color: #60a5fa;
  }
  .dark &:hover:not(:disabled) {
    background: rgba(0, 113, 227, 0.14);
    border-color: rgba(0, 113, 227, 0.2);
    box-shadow: none;
  }
}
.oauth-btn-outline:hover:not(:disabled) {
  background: rgba(0, 113, 227, 0.1);
  border-color: rgba(0, 113, 227, 0.25);
  box-shadow: none;
}
</style>
