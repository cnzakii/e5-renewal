<template>
  <div class="space-y-6 animate-fade-in">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white tracking-tight">{{ t('accounts.title') }}</h1>
      <button
        class="group inline-flex items-center gap-2 px-4 py-2.5 rounded-2xl text-[13px] font-medium text-white bg-apple-blue hover:bg-apple-blue-hover shadow-md shadow-apple-blue/20 hover:shadow-lg hover:shadow-apple-blue/30 transition-all duration-300 btn-shine"
        @click="openAdd"
      >
        <svg class="w-4 h-4 transition-transform duration-200 group-hover:rotate-90" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
        </svg>
        {{ t('accounts.add') }}
      </button>
    </div>

    <!-- Skeleton Loading -->
    <div v-if="accountsLoading && !accounts.length" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      <div
        v-for="i in 3"
        :key="'skeleton-' + i"
        :class="['glass-card p-5 space-y-4 stagger-item', `stagger-${i}`]"
      >
        <!-- Name + badge skeleton -->
        <div class="flex items-start justify-between">
          <div class="space-y-2 flex-1">
            <div class="flex items-center gap-2">
              <div class="w-10 h-5 rounded-lg skeleton-shimmer"></div>
              <div class="w-28 h-5 rounded-lg skeleton-shimmer"></div>
            </div>
            <div class="w-20 h-4 rounded-md skeleton-shimmer"></div>
          </div>
          <div class="flex gap-1">
            <div class="w-7 h-7 rounded-lg skeleton-shimmer"></div>
            <div class="w-7 h-7 rounded-lg skeleton-shimmer"></div>
          </div>
        </div>
        <!-- Health bar skeleton -->
        <div class="space-y-1.5">
          <div class="flex justify-between">
            <div class="w-12 h-3 rounded skeleton-shimmer"></div>
            <div class="w-10 h-4 rounded skeleton-shimmer"></div>
          </div>
          <div class="w-full h-1.5 rounded-full skeleton-shimmer"></div>
        </div>
        <!-- Stats row skeleton -->
        <div class="w-full h-10 rounded-xl skeleton-shimmer"></div>
        <!-- Expiry skeleton -->
        <div class="w-full h-9 rounded-xl skeleton-shimmer"></div>
        <!-- Bottom tiles skeleton -->
        <div class="grid grid-cols-2 gap-2">
          <div class="h-[76px] rounded-xl skeleton-shimmer"></div>
          <div class="h-[76px] rounded-xl skeleton-shimmer"></div>
        </div>
      </div>
    </div>

    <!-- Card Grid -->
    <div v-else-if="accounts.length" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      <div
        v-for="(acc, i) in accounts"
        :key="acc.id"
        :ref="(el: any) => { if (el) cardRefs[acc.id] = el as HTMLElement }"
        :class="[
          'account-card',
          skipStagger ? '' : `stagger-item stagger-${(i % 6) + 1}`,
          highlightId === acc.id ? 'card-spotlight' : ''
        ]"
      >
        <div class="glass-card glass-card-hover flex flex-col group/card">
          <!-- ═══ Account body — click opens auth editor ═══ -->
          <div
            class="flex-1 flex flex-col gap-3 p-5 cursor-pointer"
            @click="openPreview(acc)"
          >
            <!-- Name + Badge + Delete (hover) -->
            <div class="flex items-start justify-between gap-3">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <span class="px-2 py-0.5 rounded-lg text-[11px] font-medium tabular-nums bg-gray-100/50 dark:bg-white/5 text-gray-400 dark:text-gray-500 shrink-0">#{{ acc.id }}</span>
                  <h3 class="text-[15px] font-semibold text-gray-900 dark:text-white truncate">{{ acc.name }}</h3>
                </div>
                <span class="inline-flex items-center gap-1.5 mt-1.5 px-2 py-0.5 rounded-md text-[10px] font-semibold tracking-wide uppercase bg-gray-100/70 dark:bg-white/6 text-gray-500 dark:text-gray-400">
                  <span :class="['w-1.5 h-1.5 rounded-full shrink-0', acc.auth_type === 'auth_code' ? 'bg-blue-500' : 'bg-amber-500']"></span>
                  {{ t(`accounts.type.${acc.auth_type}`) }}
                </span>
              </div>
              <div class="flex items-center gap-1">
                <button
                  :title="acc.notify_enabled ? t('accounts.notify.on') : t('accounts.notify.off')"
                  :class="[
                    'w-7 h-7 rounded-lg flex items-center justify-center shrink-0 transition-all duration-200',
                    acc.notify_enabled
                      ? 'text-apple-blue bg-apple-blue/10 hover:bg-apple-blue/20'
                      : 'text-gray-300 dark:text-gray-600 bg-gray-100/50 dark:bg-white/5 hover:bg-gray-200/50 dark:hover:bg-white/10'
                  ]"
                  @click.stop="toggleNotify(acc)"
                >
                  <svg class="w-3.5 h-3.5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M14.857 17.082a23.848 23.848 0 005.454-1.31A8.967 8.967 0 0118 9.75v-.7V9A6 6 0 006 9v.75a8.967 8.967 0 01-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 01-5.714 0m5.714 0a3 3 0 11-5.714 0" />
                  </svg>
                </button>
                <!-- Delete (appears on card hover) -->
                <button
                  class="w-7 h-7 rounded-lg flex items-center justify-center shrink-0 text-gray-300 dark:text-gray-600 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-500/8 transition-all duration-200"
                  :title="t('accounts.action.delete')"
                  @click.stop="confirmDelete(acc)"
                >
                  <svg class="w-3.5 h-3.5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
                  </svg>
                </button>
              </div>
            </div>

            <!-- Health (always rendered) -->
            <div class="space-y-1.5">
              <div class="flex items-center justify-between">
                <span class="text-[11px] text-gray-400 dark:text-gray-500">{{ t('accounts.health') }}</span>
                <span :class="['text-[13px] font-semibold tabular-nums', acc.health != null ? healthTextColor(acc.health) : 'text-gray-300 dark:text-gray-600']">
                  {{ acc.health != null ? acc.health.toFixed(1) + '%' : '--' }}
                </span>
              </div>
              <div class="w-full h-1.5 rounded-full bg-gray-200/60 dark:bg-gray-700/60 overflow-hidden">
                <div
                  v-if="acc.health != null"
                  class="h-full rounded-full transition-all duration-500"
                  :class="healthBarColor(acc.health)"
                  :style="{ width: `${acc.health}%` }"
                />
              </div>
            </div>

            <!-- Stats Row -->
            <div class="flex items-center justify-between px-2.5 py-2 rounded-xl bg-gray-50/50 dark:bg-white/3">
              <div class="flex items-center gap-3">
                <div class="flex items-baseline gap-1">
                  <span class="text-[14px] font-bold tabular-nums text-gray-900 dark:text-white">{{ acc.total_runs ?? 0 }}</span>
                  <span class="text-[10px] text-gray-400 dark:text-gray-500">{{ t('accounts.totalRuns') }}</span>
                </div>
                <div class="w-px h-3 bg-gray-200/80 dark:bg-gray-700/80"></div>
                <div class="flex items-baseline gap-1">
                  <span class="text-[14px] font-bold tabular-nums text-gray-900 dark:text-white">{{ acc.success_runs ?? 0 }}</span>
                  <span class="text-[10px] text-gray-400 dark:text-gray-500">{{ t('accounts.successRuns') }}</span>
                </div>
                <div class="w-px h-3 bg-gray-200/80 dark:bg-gray-700/80"></div>
                <div class="flex items-baseline gap-1">
                  <span class="text-[14px] font-bold tabular-nums text-gray-900 dark:text-white">{{ (acc.total_runs ?? 0) - (acc.success_runs ?? 0) }}</span>
                  <span class="text-[10px] text-gray-400 dark:text-gray-500">{{ t('accounts.failedRuns') }}</span>
                </div>
              </div>
              <span class="text-[10px] text-gray-400 dark:text-gray-500 tabular-nums">
                {{ acc.last_run ? formatScheduleTime(acc.last_run) : t('accounts.lastRun.never') }}
              </span>
            </div>

            <!-- Client Secret expiry (always rendered) -->
            <div class="flex items-center gap-2 px-2.5 py-2 rounded-xl bg-gray-50/50 dark:bg-white/3">
              <svg class="w-3.5 h-3.5 shrink-0 text-gray-400 dark:text-gray-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span v-if="acc.auth_expires_at" class="text-[12px] font-medium text-gray-600 dark:text-gray-400">
                {{ formatExpiryDate(acc.auth_expires_at) }}
              </span>
              <span v-if="acc.auth_expires_at" :class="['text-[11px] font-medium shrink-0 whitespace-nowrap', expiryUrgencyColor(acc.auth_expires_at)]">
                {{ expiryRemainingText(acc.auth_expires_at) }}
              </span>
              <span v-if="!acc.auth_expires_at" class="text-[12px] font-medium text-gray-300 dark:text-gray-600">
                {{ t('accounts.expiry.none') }}
              </span>
            </div>
          </div>

          <!-- ═══ Bottom tiles: Schedule + Trigger (fixed height) ═══ -->
          <div class="grid grid-cols-2 gap-2 px-4 pb-4">
            <!-- Schedule tile: icon+color = status, text = last run -->
            <div
              :class="['action-tile cursor-pointer h-[76px] overflow-hidden flex flex-col', scheduleTileColor(acc)]"
              @click.stop="openScheduleDialog(acc)"
            >
              <div class="flex items-center gap-2 mb-1.5">
                <!-- Paused: pause icon -->
                <svg v-if="acc.schedule?.paused" class="w-3.5 h-3.5 shrink-0" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25v13.5m-7.5-13.5v13.5" />
                </svg>
                <!-- Enabled: clock icon -->
                <svg v-else-if="acc.schedule?.enabled" class="w-3.5 h-3.5 shrink-0" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <!-- Disabled: stop icon -->
                <svg v-else class="w-3.5 h-3.5 shrink-0 opacity-40" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M5.25 7.5A2.25 2.25 0 017.5 5.25h9a2.25 2.25 0 012.25 2.25v9a2.25 2.25 0 01-2.25 2.25h-9a2.25 2.25 0 01-2.25-2.25v-9z" />
                </svg>
                <span class="text-[11px] font-semibold uppercase tracking-wider opacity-60">{{ t('accounts.schedule.title') }}</span>
              </div>
              <span class="text-[10px] opacity-50 tabular-nums">
                {{ scheduleDetailText(acc) }}
              </span>
            </div>

            <!-- Trigger tile -->
            <div
              class="action-tile action-tile-blue cursor-pointer h-[76px] overflow-hidden flex flex-col"
              @click.stop="triggerAccount(acc)"
            >
              <div class="flex items-center gap-2 mb-1.5">
                <svg class="w-3.5 h-3.5 shrink-0 opacity-60" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.348a1.125 1.125 0 010 1.971l-11.54 6.347a1.125 1.125 0 01-1.667-.985V5.653z" />
                </svg>
                <span class="text-[11px] font-semibold uppercase tracking-wider opacity-60">{{ t('accounts.action.trigger') }}</span>
              </div>
              <span class="text-[10px] opacity-50 tabular-nums truncate">
                {{ acc.last_run ? t('accounts.trigger.lastRun') + formatScheduleTime(acc.last_run) : t('accounts.trigger.neverTriggered') }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else class="flex flex-col items-center justify-center py-20 glass-card rounded-2xl">
      <div class="w-16 h-16 rounded-2xl bg-apple-blue/10 flex items-center justify-center mb-5">
        <svg class="w-8 h-8 text-apple-blue" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z" />
        </svg>
      </div>
      <h3 class="text-[15px] font-semibold text-gray-900 dark:text-white mb-1.5">{{ t('accounts.empty') }}</h3>
      <p class="text-[13px] text-gray-400 dark:text-gray-500 mb-6">{{ t('accounts.empty.hint') }}</p>
      <button
        class="inline-flex items-center gap-2 px-5 py-2.5 rounded-2xl text-[13px] font-medium text-white bg-apple-blue hover:bg-apple-blue-hover shadow-md shadow-apple-blue/20 transition-all duration-300 btn-shine"
        @click="openAdd"
      >
        <svg class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
        </svg>
        {{ t('accounts.add') }}
      </button>
    </div>

    <!-- Form Dialog -->
    <AccountFormDialog
      v-model:visible="showFormDialog"
      :account="editingAccount"
      @save="handleSave"
    />

    <!-- Schedule Dialog -->
    <ScheduleDialog
      v-model:visible="showScheduleDialog"
      :account-id="scheduleAccountId"
      :schedule="scheduleAccountData"
      @save="handleScheduleSave"
      @resume="handleScheduleResume"
    />

    <!-- Trigger Result Dialog -->
    <TriggerResultDialog
      v-model:visible="showTriggerDialog"
      :result="triggerResult"
      :loading="triggerLoading"
      :trigger-account-name="triggerAccountName"
    />

    <!-- Delete Confirmation -->
    <ConfirmDialog
      v-model:visible="showDeleteDialog"
      :title="t('accounts.delete.title')"
      :message="deleteMessage"
      :confirm-text="t('accounts.delete.confirm')"
      :cancel-text="t('accounts.delete.cancel')"
      danger
      @confirm="handleDelete"
    />

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
import { ref, reactive, onMounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from '../i18n'
import AccountFormDialog from '../components/AccountFormDialog.vue'
import ScheduleDialog from '../components/ScheduleDialog.vue'
import TriggerResultDialog from '../components/TriggerResultDialog.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import type { Account, AccountFormData, AccountSchedule } from '../components/AccountFormDialog.vue'
import type { TriggerResult } from '../components/TriggerResultDialog.vue'
import { apiClient } from '../api/client'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const accounts = ref<Account[]>([])
const accountsLoading = ref(true)
const showFormDialog = ref(false)
const editingAccount = ref<Account | null>(null)
const showDeleteDialog = ref(false)
const deletingAccount = ref<Account | null>(null)
const showTriggerDialog = ref(false)
const triggerResult = ref<TriggerResult | null>(null)
const triggerLoading = ref(false)
const triggerAccountName = ref('')
const showScheduleDialog = ref(false)
const scheduleAccountId = ref<number | null>(null)
const scheduleAccountData = ref<AccountSchedule | null>(null)

const highlightId = ref<number | null>(null)
const skipStagger = ref(false)
const cardRefs = ref<Record<number, HTMLElement | null>>({})

const toast = reactive({ show: false, type: 'success' as 'success' | 'error', message: '' })
let toastTimer: ReturnType<typeof setTimeout> | undefined

function showToast(type: 'success' | 'error', message: string) {
  clearTimeout(toastTimer)
  toast.show = true
  toast.type = type
  toast.message = message
  toastTimer = setTimeout(() => { toast.show = false }, 2500)
}

const deleteMessage = ref('')

function openAdd() {
  editingAccount.value = null
  showFormDialog.value = true
}

function openPreview(acc: Account) {
  editingAccount.value = acc
  showFormDialog.value = true
}

function confirmDelete(acc: Account) {
  deletingAccount.value = acc
  deleteMessage.value = t('accounts.delete.message').replace('{name}', acc.name)
  showDeleteDialog.value = true
}

async function handleSave(data: AccountFormData) {
  try {
    if (editingAccount.value) {
      await apiClient.put(`/accounts/${editingAccount.value.id}`, data)
    } else {
      await apiClient.post('/accounts', data)
    }
    showFormDialog.value = false
    showToast('success', t('accounts.form.save.success'))
    await fetchAccounts()
  } catch {
    showToast('error', t('accounts.form.save.error'))
  }
}

async function handleDelete() {
  if (!deletingAccount.value) return

  try {
    await apiClient.delete(`/accounts/${deletingAccount.value.id}`)
    showToast('success', t('accounts.delete.success'))
    deletingAccount.value = null
    await fetchAccounts()
  } catch {
    showToast('error', t('accounts.delete.error'))
  }
}

async function triggerAccount(acc: Account) {
  triggerResult.value = null
  triggerAccountName.value = acc.name
  triggerLoading.value = true
  showTriggerDialog.value = true
  try {
    const res = await apiClient.post(`/accounts/${acc.id}/trigger`)
    triggerResult.value = res.data
  } catch (err: any) {
    if (err?.response?.data?.task_log) {
      triggerResult.value = err.response.data
    } else {
      showTriggerDialog.value = false
      showToast('error', t('accounts.trigger.error'))
    }
  } finally {
    triggerLoading.value = false
  }
}

async function fetchAccounts() {
  accountsLoading.value = true
  try {
    const res = await apiClient.get('/accounts')
    accounts.value = res.data
  } catch {
    // keep current data on error
  } finally {
    accountsLoading.value = false
  }
}

async function toggleNotify(acc: Account) {
  const newVal = !acc.notify_enabled
  try {
    await apiClient.put(`/accounts/${acc.id}`, {
      name: acc.name,
      auth_type: acc.auth_type,
      client_id: acc.client_id,
      client_secret: acc.client_secret,
      tenant_id: acc.tenant_id,
      refresh_token: acc.refresh_token,
      notify_enabled: newVal,
      auth_expires_at: acc.auth_expires_at || '',
    })
    acc.notify_enabled = newVal
  } catch {
    showToast('error', t('accounts.form.save.error'))
  }
}

onMounted(async () => {
  await fetchAccounts()

  const hlParam = route.query.highlight as string | undefined
  if (!hlParam) return
  skipStagger.value = true
  router.replace({ query: { ...route.query, highlight: undefined } })
  const targetId = Number(hlParam)
  if (!targetId) return
  await nextTick()
  const acc = accounts.value.find(a => a.id === targetId)
  if (!acc) return
  highlightId.value = targetId
  await nextTick()
  const el = cardRefs.value[acc.id]
  if (el) {
    el.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }
  setTimeout(() => { highlightId.value = null }, 5000)
})


function healthBarColor(health: number): string {
  if (health >= 90) return 'bg-gradient-to-r from-emerald-400 to-emerald-500'
  if (health >= 70) return 'bg-gradient-to-r from-amber-400 to-amber-500'
  return 'bg-gradient-to-r from-red-400 to-red-500'
}

function healthTextColor(health: number): string {
  if (health >= 90) return 'text-emerald-600 dark:text-emerald-400'
  if (health >= 70) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
}

function scheduleDetailText(acc: Account): string {
  if (!acc.schedule) return t('accounts.schedule.notSet')
  if (acc.schedule.paused) return t('accounts.schedule.paused')
  if (!acc.schedule.enabled) return t('accounts.schedule.disabled')
  if (acc.schedule.next_run_at) return t('accounts.schedule.nextRun') + ' ' + formatScheduleTime(acc.schedule.next_run_at)
  return t('accounts.schedule.enabled')
}

function formatScheduleTime(iso: string): string {
  if (!iso) return '-'
  const d = new Date(iso)
  const yy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  return `${yy}-${mm}-${dd} ${hh}:${mi}`
}

function openScheduleDialog(acc: Account) {
  scheduleAccountId.value = acc.id
  scheduleAccountData.value = acc.schedule || null
  showScheduleDialog.value = true
}

async function handleScheduleSave(accountId: number, data: { enabled: boolean; pause_threshold: number }) {
  try {
    await apiClient.put(`/accounts/${accountId}/schedule`, data)
    showScheduleDialog.value = false
    showToast('success', t('accounts.form.save.success'))
    await fetchAccounts()
  } catch {
    showToast('error', t('accounts.schedule.error'))
  }
}

async function handleScheduleResume(accountId: number) {
  try {
    await apiClient.put(`/accounts/${accountId}/schedule`, { paused: false })
    showScheduleDialog.value = false
    showToast('success', t('accounts.schedule.resume.success'))
    await fetchAccounts()
  } catch {
    showToast('error', t('accounts.schedule.error'))
  }
}

function scheduleTileColor(acc: Account): string {
  if (acc.schedule?.paused) return 'action-tile-red'
  if (acc.schedule?.enabled) return 'action-tile-green'
  return 'action-tile-gray'
}



// --- Expiry helpers ---
function daysUntilExpiry(dateStr: string): number {
  return Math.ceil((new Date(dateStr).getTime() - Date.now()) / 86400000)
}

function formatExpiryDate(dateStr: string): string {
  return dateStr // already YYYY-MM-DD
}

function expiryRemainingText(dateStr: string): string {
  const days = daysUntilExpiry(dateStr)
  if (days < 0) return t('accounts.expiry.expired')
  if (days === 0) return t('accounts.expiry.today')
  return t('accounts.expiry.remaining').replace('{days}', String(days))
}

function expiryUrgencyColor(dateStr: string): string {
  const days = daysUntilExpiry(dateStr)
  if (days <= 7) return 'text-red-500 dark:text-red-400'
  if (days <= 30) return 'text-amber-500 dark:text-amber-400'
  return 'text-gray-400 dark:text-gray-500'
}
</script>

<style scoped>
/* === Glass card === */
.glass-card {
  border-radius: 16px;
  backdrop-filter: blur(40px);
  -webkit-backdrop-filter: blur(40px);
  background: rgba(255, 255, 255, 0.55);
  border: 1px solid rgba(255, 255, 255, 0.2);
  transition: box-shadow 0.3s ease,
              border-color 0.3s ease,
              background 0.3s ease;
  .dark & {
    background: rgba(40, 40, 40, 0.55);
    border-color: rgba(255, 255, 255, 0.08);
  }
}
.glass-card-hover:hover {
  box-shadow: 0 12px 40px -8px rgba(0, 0, 0, 0.1), 0 4px 16px -4px rgba(0, 0, 0, 0.06);
  border-color: rgba(255, 255, 255, 0.35);
  background: rgba(255, 255, 255, 0.65);
}
.glass-card-hover {
  .dark &:hover {
    box-shadow: 0 12px 40px -8px rgba(0, 0, 0, 0.4), 0 0 20px -4px rgba(0, 113, 227, 0.08);
    border-color: rgba(255, 255, 255, 0.14);
    background: rgba(45, 45, 45, 0.7);
  }
}

/* === Spotlight: soft ring that fades out === */
.card-spotlight > .glass-card {
  animation: spotlight-ring 3s ease-out forwards;
  border-color: rgba(0, 113, 227, 0.35);
}
.card-spotlight {
  .dark & > .glass-card {
    border-color: rgba(0, 113, 227, 0.3);
  }
}
@keyframes spotlight-ring {
  0%   { box-shadow: 0 0 0 0 rgba(0, 113, 227, 0.3); }
  15%  { box-shadow: 0 0 0 4px rgba(0, 113, 227, 0.15); }
  40%  { box-shadow: 0 0 0 4px rgba(0, 113, 227, 0.1); }
  100% { box-shadow: 0 0 0 0 rgba(0, 113, 227, 0); border-color: transparent; }
}
/* === Action tiles === */
.action-tile {
  padding: 10px 12px;
  border-radius: 12px;
  border: 1px solid transparent;
  transition: all 0.2s ease;
  height: 76px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.action-tile:hover {
  filter: brightness(0.96);
}
.action-tile {
  .dark &:hover {
    filter: brightness(1.15);
  }
}

.action-tile-red {
  background: rgba(239, 68, 68, 0.06);
  border-color: rgba(239, 68, 68, 0.15);
  color: #dc2626;
  .dark & {
    background: rgba(239, 68, 68, 0.06);
    border-color: rgba(239, 68, 68, 0.1);
    color: #f87171;
  }
}

.action-tile-green {
  background: rgba(16, 185, 129, 0.06);
  border-color: rgba(16, 185, 129, 0.12);
  color: #059669;
  .dark & {
    background: rgba(16, 185, 129, 0.06);
    border-color: rgba(16, 185, 129, 0.1);
    color: #6ee7b7;
  }
}

.action-tile-amber {
  background: rgba(245, 158, 11, 0.06);
  border-color: rgba(245, 158, 11, 0.15);
  color: #d97706;
  .dark & {
    background: rgba(245, 158, 11, 0.06);
    border-color: rgba(245, 158, 11, 0.1);
    color: #fbbf24;
  }
}

.action-tile-gray {
  background: rgba(107, 114, 128, 0.04);
  border-color: rgba(107, 114, 128, 0.1);
  color: #6b7280;
  .dark & {
    background: rgba(255, 255, 255, 0.02);
    border-color: rgba(255, 255, 255, 0.06);
    color: #9ca3af;
  }
}

.action-tile-blue {
  background: rgba(0, 113, 227, 0.04);
  border-color: rgba(0, 113, 227, 0.1);
  color: #0071e3;
  .dark & {
    background: rgba(0, 113, 227, 0.05);
    border-color: rgba(0, 113, 227, 0.1);
    color: #60a5fa;
  }
}

/* === Toast === */
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
