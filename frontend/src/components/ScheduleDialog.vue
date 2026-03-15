<template>
  <Teleport to="body">
    <Transition name="dialog-overlay">
      <div
        v-if="visible"
        class="fixed inset-0 z-50 flex items-center justify-center p-4"
        @keydown.esc="close"
      >
        <!-- Backdrop -->
        <div
          class="absolute inset-0 bg-black/30 dark:bg-black/50 backdrop-blur-sm"
          @click="close"
        />

        <!-- Dialog -->
        <Transition name="dialog-content" appear>
          <div
            v-if="visible"
            class="relative w-full max-w-sm flex flex-col rounded-2xl backdrop-blur-[40px] bg-white/85 dark:bg-[rgb(38,38,38)]/85 border border-white/25 dark:border-white/10 shadow-2xl shadow-black/12 dark:shadow-black/40"
          >
            <!-- Header -->
            <div class="flex items-center justify-between px-6 pt-5 pb-4 shrink-0">
              <div class="flex items-center gap-3">
                <div class="w-9 h-9 rounded-xl flex items-center justify-center bg-apple-blue/10 text-apple-blue">
                  <svg class="w-4.5 h-4.5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                </div>
                <h2 class="text-[16px] font-semibold text-gray-900 dark:text-white">
                  {{ t('accounts.schedule.title') }}
                </h2>
              </div>
              <button
                class="w-8 h-8 flex items-center justify-center rounded-full text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 hover:bg-gray-100/60 dark:hover:bg-white/8 transition-all duration-200"
                @click="close"
              >
                <svg class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <!-- Content -->
            <div class="px-6 pb-4 space-y-4">
              <!-- Current Status -->
              <div v-if="schedule" class="space-y-2">
                <div class="flex items-center gap-2">
                  <span
                    :class="[
                      'px-2.5 py-1 rounded-lg text-[11px] font-semibold',
                      schedule.paused
                        ? 'bg-amber-100/70 dark:bg-amber-900/25 text-amber-600 dark:text-amber-400'
                        : schedule.enabled
                          ? 'bg-emerald-100/70 dark:bg-emerald-900/25 text-emerald-600 dark:text-emerald-400'
                          : 'bg-gray-100/70 dark:bg-gray-700/40 text-gray-500 dark:text-gray-400'
                    ]"
                  >
                    {{ schedule.paused ? t('accounts.schedule.paused') : schedule.enabled ? t('accounts.schedule.enabled') : t('accounts.schedule.disabled') }}
                  </span>
                  <span v-if="schedule.paused && schedule.pause_reason" class="text-[11px] text-amber-600 dark:text-amber-400 truncate">
                    {{ schedule.pause_reason }}
                  </span>
                </div>

                <!-- Resume button if paused -->
                <button
                  v-if="schedule.paused"
                  :disabled="saving"
                  @click="resume"
                  class="w-full flex items-center justify-center gap-1.5 py-2 rounded-xl text-[12px] font-semibold text-amber-600 dark:text-amber-400 bg-amber-100/50 dark:bg-amber-900/20 hover:bg-amber-100 dark:hover:bg-amber-900/30 border border-amber-200/50 dark:border-amber-500/15 transition-all duration-200"
                >
                  <svg class="w-3.5 h-3.5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.348a1.125 1.125 0 010 1.971l-11.54 6.347a1.125 1.125 0 01-1.667-.985V5.653z" />
                  </svg>
                  {{ t('accounts.schedule.resume') }}
                </button>

                <!-- Info rows -->
                <div class="space-y-1">
                  <div v-if="schedule.next_run_at" class="info-row">
                    <span class="info-label">{{ t('accounts.schedule.nextRun') }}</span>
                    <span class="info-value">{{ formatDateTime(schedule.next_run_at) }}</span>
                  </div>
                  <div v-if="schedule.last_run_at" class="info-row">
                    <span class="info-label">{{ t('accounts.schedule.lastRun') }}</span>
                    <span class="info-value">{{ formatDateTime(schedule.last_run_at) }}</span>
                  </div>
                </div>
              </div>

              <!-- Divider -->
              <div class="flex items-center gap-3">
                <div class="flex-1 h-px bg-gray-200/50 dark:bg-white/6"></div>
              </div>

              <!-- Schedule Enabled Toggle -->
              <div class="flex items-center justify-between px-3 py-3 rounded-xl bg-gray-50/50 dark:bg-white/3 border border-gray-100/50 dark:border-white/5">
                <span class="text-[13px] font-medium text-gray-700 dark:text-gray-300">{{ t('accounts.schedule.enabled') }}</span>
                <button
                  type="button"
                  :class="[
                    'relative w-11 h-6 rounded-full transition-colors duration-200 focus:outline-none',
                    form.enabled ? 'bg-apple-blue' : 'bg-gray-300 dark:bg-gray-600'
                  ]"
                  @click="form.enabled = !form.enabled"
                >
                  <span :class="['absolute top-0.5 left-0.5 w-5 h-5 rounded-full bg-white shadow-sm transition-transform duration-200', form.enabled ? 'translate-x-5' : 'translate-x-0']" />
                </button>
              </div>

              <!-- Pause Threshold -->
              <div>
                <label class="form-label">{{ t('accounts.schedule.pauseThreshold') }}</label>
                <div class="flex items-center gap-2">
                  <input
                    v-model.number="form.pause_threshold"
                    type="number"
                    min="0"
                    max="100"
                    class="form-input w-24 text-center tabular-nums"
                  />
                  <span class="text-[13px] text-gray-500 dark:text-gray-400">%</span>
                </div>
                <p class="mt-1.5 text-[11px] text-gray-400 dark:text-gray-500">
                  {{ t('accounts.schedule.pauseThreshold.hint') }}
                </p>
              </div>
            </div>

            <!-- Footer -->
            <div class="flex items-center justify-end gap-2.5 px-6 py-4 shrink-0 border-t border-gray-100/50 dark:border-white/5">
              <button
                class="px-5 py-2.5 rounded-xl text-[13px] font-medium text-gray-600 dark:text-gray-300 bg-gray-100/60 dark:bg-white/8 hover:bg-gray-200/60 dark:hover:bg-white/12 border border-gray-200/60 dark:border-white/10 transition-all duration-200"
                @click="close"
              >
                {{ t('accounts.form.cancel') }}
              </button>
              <button
                :disabled="saving"
                @click="save"
                class="px-5 py-2.5 rounded-xl text-[13px] font-medium text-white bg-apple-blue hover:bg-apple-blue-hover border border-apple-blue/80 shadow-md shadow-apple-blue/20 transition-all duration-200 disabled:opacity-60 btn-shine"
              >
                {{ saving ? t('accounts.form.saving') : t('accounts.form.save') }}
              </button>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { reactive, watch, ref } from 'vue'
import { useI18n } from '../i18n'
import type { AccountSchedule } from './AccountFormDialog.vue'

const props = defineProps<{
  visible: boolean
  accountId: number | null
  schedule: AccountSchedule | null
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'save', accountId: number, data: { enabled: boolean; pause_threshold: number }): void
  (e: 'resume', accountId: number): void
}>()

const { t } = useI18n()
const saving = ref(false)

const form = reactive({
  enabled: true,
  pause_threshold: 30,
})

function formatDateTime(iso: string): string {
  if (!iso) return '-'
  const d = new Date(iso)
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  return `${mm}-${dd} ${hh}:${mi}`
}

function close() {
  emit('update:visible', false)
}

function save() {
  if (!props.accountId) return
  emit('save', props.accountId, { ...form })
}

function resume() {
  if (!props.accountId) return
  emit('resume', props.accountId)
}

watch(() => props.visible, (val) => {
  if (val && props.schedule) {
    form.enabled = props.schedule.enabled
    form.pause_threshold = props.schedule.pause_threshold
  }
})
</script>

<style scoped>
.info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  border-radius: 10px;
}
.info-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: #9ca3af;
  .dark & {
    color: #6b7280;
  }
}
.info-value {
  font-size: 12px;
  font-weight: 500;
  color: #374151;
  font-variant-numeric: tabular-nums;
  .dark & {
    color: #d1d5db;
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
  .dark &:focus {
    background: rgba(255, 255, 255, 0.08);
  }
}
.form-input:focus {
  border-color: #0071e3;
  box-shadow: 0 0 0 3px rgba(0, 113, 227, 0.12);
  background: rgba(255, 255, 255, 0.8);
}

/* Dialog transitions */
.dialog-overlay-enter-active,
.dialog-overlay-leave-active {
  transition: opacity 0.2s ease;
}
.dialog-overlay-enter-from,
.dialog-overlay-leave-to {
  opacity: 0;
}
.dialog-content-enter-active {
  transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.dialog-content-leave-active {
  transition: all 0.15s ease;
}
.dialog-content-enter-from {
  opacity: 0;
  transform: scale(0.95) translateY(8px);
}
.dialog-content-leave-to {
  opacity: 0;
  transform: scale(0.97);
}
</style>
