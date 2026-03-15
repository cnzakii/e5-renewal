<template>
  <Teleport to="body">
    <Transition name="dialog-overlay">
      <div
        v-if="visible"
        class="fixed inset-0 z-50 flex items-center justify-center p-4"
        @keydown.esc="close"
      >
        <div class="absolute inset-0 bg-black/30 dark:bg-black/50 backdrop-blur-sm" @click="close" />

        <Transition name="dialog-content" appear>
          <div
            v-if="visible"
            class="relative w-full max-w-md flex flex-col rounded-2xl backdrop-blur-[40px] bg-white/85 dark:bg-[rgb(38,38,38)]/85 border border-white/25 dark:border-white/10 shadow-2xl shadow-black/12 dark:shadow-black/40"
          >
            <!-- Header -->
            <div class="flex items-center justify-between px-6 pt-5 pb-4 shrink-0">
              <div class="flex items-center gap-3">
                <!-- Loading state -->
                <div v-if="loading" class="w-9 h-9 rounded-xl flex items-center justify-center bg-apple-blue/10 text-apple-blue">
                  <svg class="w-4.5 h-4.5 animate-spin" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
                  </svg>
                </div>
                <!-- Result state -->
                <div v-else :class="['w-9 h-9 rounded-xl flex items-center justify-center', allSuccess ? 'bg-emerald-500/10 text-emerald-500' : 'bg-amber-500/10 text-amber-500']">
                  <svg v-if="allSuccess" class="w-4.5 h-4.5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd" />
                  </svg>
                  <svg v-else class="w-4.5 h-4.5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 6a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 6zm0 9a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" />
                  </svg>
                </div>
                <div>
                  <h2 class="text-[16px] font-semibold text-gray-900 dark:text-white">{{ t('accounts.trigger.title') }}</h2>
                  <p v-if="loading" class="text-[12px] text-apple-blue">
                    {{ triggerAccountName }}
                  </p>
                  <p v-else-if="result" class="text-[12px] text-gray-500 dark:text-gray-400">
                    {{ triggerTypeLabel }} &middot; {{ result.task_log.total_endpoints }} {{ t('accounts.trigger.endpoint') }}
                  </p>
                </div>
              </div>
              <span></span>
            </div>

            <!-- Content -->
            <div class="px-6 pb-4">
              <!-- Loading state -->
              <template v-if="loading">
                <div class="flex flex-col items-center py-8 gap-4">
                  <div class="running-dots flex items-center gap-1.5">
                    <span class="w-2 h-2 rounded-full bg-apple-blue/60 animate-bounce" style="animation-delay: 0s"></span>
                    <span class="w-2 h-2 rounded-full bg-apple-blue/60 animate-bounce" style="animation-delay: 0.15s"></span>
                    <span class="w-2 h-2 rounded-full bg-apple-blue/60 animate-bounce" style="animation-delay: 0.3s"></span>
                  </div>
                  <p class="text-[13px] text-gray-500 dark:text-gray-400">{{ t('accounts.trigger.running') }}</p>
                </div>
              </template>
              <!-- Result state -->
              <template v-else-if="result">
                <!-- Summary bar -->
                <div class="flex items-center justify-between mb-3">
                  <span class="text-[12px] font-medium text-gray-500 dark:text-gray-400">
                    {{ t('accounts.trigger.successCount').replace('{success}', String(result.task_log.success_count)).replace('{total}', String(result.task_log.total_endpoints)) }}
                  </span>
                  <span :class="['text-[12px] font-semibold', allSuccess ? 'text-emerald-500' : 'text-amber-500']">
                    {{ t('accounts.trigger.complete') }}
                  </span>
                </div>

                <!-- Endpoint list -->
                <div class="space-y-1.5 max-h-[320px] overflow-y-auto hide-scrollbar">
                  <div
                    v-for="ep in sortedEndpoints"
                    :key="ep.id"
                    :class="[
                      'px-3 py-2.5 rounded-xl text-[13px]',
                      ep.success
                        ? 'bg-emerald-50/50 dark:bg-emerald-900/10'
                        : 'bg-red-50/50 dark:bg-red-900/10'
                    ]"
                  >
                    <div class="flex items-center gap-3">
                      <span :class="['w-1.5 h-1.5 rounded-full shrink-0', ep.success ? 'bg-emerald-500' : 'bg-red-500']"></span>
                      <div class="flex-1 min-w-0">
                        <span
                          v-if="ep.scope"
                          class="font-medium text-gray-700 dark:text-gray-300 truncate cursor-pointer active:opacity-60 transition-opacity duration-100 inline-flex items-center gap-1.5"
                          :title="t('logs.error.copy')"
                          @click.stop="copyScope(ep.scope)"
                        >{{ ep.scope }}<svg v-if="copiedScope === ep.scope" class="w-3 h-3 text-emerald-500 shrink-0" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" /></svg></span>
                        <span v-else class="font-medium text-gray-700 dark:text-gray-300 truncate block">{{ ep.endpoint_name }}</span>
                        <span v-if="ep.scope" class="text-[10px] text-gray-400 dark:text-gray-500 font-mono truncate block">{{ ep.endpoint_name }}</span>
                      </div>
                      <span :class="['text-[11px] font-semibold tabular-nums px-1.5 py-0.5 rounded', ep.http_status >= 400 || !ep.http_status ? 'text-red-600 dark:text-red-400 bg-red-100/60 dark:bg-red-900/20' : 'text-emerald-600 dark:text-emerald-400 bg-emerald-100/60 dark:bg-emerald-900/20']">
                        {{ ep.http_status || '-' }}
                      </span>
                    </div>
                  </div>
                </div>
              </template>
            </div>

            <!-- Footer -->
            <div v-if="!loading" class="flex items-center justify-end gap-2.5 px-6 py-4 shrink-0 border-t border-gray-100/50 dark:border-white/5">
              <router-link
                v-if="result"
                :to="`${pathPrefix}/logs?id=${result.task_log.id}`"
                class="inline-flex items-center gap-1.5 px-4 py-2.5 rounded-xl text-[13px] font-medium text-apple-blue bg-apple-blue/8 hover:bg-apple-blue/15 border border-apple-blue/15 hover:border-apple-blue/30 transition-all duration-200 no-underline"
                @click="close"
              >
                {{ t('accounts.trigger.viewLogs') }}
              </router-link>
              <button
                class="px-5 py-2.5 rounded-xl text-[13px] font-medium text-gray-600 dark:text-gray-300 bg-gray-100/60 dark:bg-white/8 hover:bg-gray-200/60 dark:hover:bg-white/12 border border-gray-200/60 dark:border-white/10 transition-all duration-200"
                @click="close"
              >
                {{ t('accounts.trigger.close') }}
              </button>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from '../i18n'

export interface EndpointLog {
  id: number
  endpoint_name: string
  scope: string
  http_status: number
  success: boolean
  error_message: string
  response_body: string
  executed_at: string
}

export interface TaskLog {
  id: number
  trigger_type: string
  total_endpoints: number
  success_count: number
  fail_count: number
  started_at: string
  finished_at: string | null
}

export interface TriggerResult {
  task_log: TaskLog
  endpoints: EndpointLog[]
}

const props = defineProps<{
  visible: boolean
  result: TriggerResult | null
  loading?: boolean
  triggerAccountName?: string
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
}>()

const { t } = useI18n()
const pathPrefix = import.meta.env.VITE_PATH_PREFIX || ''

const copiedScope = ref<string | null>(null)
let copiedTimer: ReturnType<typeof setTimeout> | null = null

function copyScope(scope: string) {
  navigator.clipboard.writeText(scope)
  copiedScope.value = scope
  if (copiedTimer) clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => { copiedScope.value = null }, 1500)
}

const allSuccess = computed(() =>
  props.result != null &&
  props.result.task_log.total_endpoints > 0 &&
  props.result.task_log.fail_count === 0
)

const triggerTypeLabel = computed(() => {
  if (!props.result) return ''
  const type = props.result.task_log.trigger_type
  if (type === 'manual') return t('logs.table.manual')
  if (type === 'scheduled') return t('logs.table.scheduled')
  return type
})

// Sort: successes first, failures last
const sortedEndpoints = computed(() => {
  if (!props.result) return []
  return [...props.result.endpoints].sort((a, b) => {
    if (a.success === b.success) return 0
    return a.success ? -1 : 1
  })
})

function close() {
  emit('update:visible', false)
}
</script>

<style scoped>
.hide-scrollbar {
  scrollbar-width: none;
  -ms-overflow-style: none;
}
.hide-scrollbar::-webkit-scrollbar {
  display: none;
}
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
