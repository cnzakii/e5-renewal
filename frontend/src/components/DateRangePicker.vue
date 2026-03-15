<template>
  <div ref="triggerRef" class="date-range-trigger" @click="toggle">
    <svg class="w-3.5 h-3.5 text-gray-400 shrink-0" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
      <path stroke-linecap="round" stroke-linejoin="round" d="M6.75 3v2.25M17.25 3v2.25M3 18.75V7.5a2.25 2.25 0 012.25-2.25h13.5A2.25 2.25 0 0121 7.5v11.25m-18 0A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75m-18 0v-7.5A2.25 2.25 0 015.25 9h13.5A2.25 2.25 0 0121 11.25v7.5" />
    </svg>
    <span v-if="displayStart" class="date-text">{{ displayStart }}</span>
    <span v-else class="date-placeholder">{{ startPlaceholder }}</span>
    <span class="date-separator">~</span>
    <span v-if="displayEnd" class="date-text">{{ displayEnd }}</span>
    <span v-else class="date-placeholder">{{ endPlaceholder }}</span>
    <button
      v-if="clearable && hasValue"
      class="clear-btn"
      @click.stop="clear"
    >
      <svg class="w-3 h-3" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
        <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
      </svg>
    </button>
  </div>

  <Teleport to="body">
    <Transition name="calendar-pop">
      <div
        v-if="open"
        class="fixed inset-0 z-[9999] outline-none"
        @click.self="close"
        @keydown.esc="close"
        tabindex="-1"
        ref="overlayRef"
      >
        <div
          ref="panelRef"
          class="calendar-panel"
          :style="panelStyle"
          @click.stop
        >
          <DatePicker
            v-model.range="internalRange"
            mode="date"
            :locale="vcLocale"
            :is-dark="isDark"
            color="blue"
            :columns="1"
            :rows="1"
            borderless
            transparent
            :first-day-of-week="1"
            @dayclick="onDayClick"
          />
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { DatePicker } from 'v-calendar'
import 'v-calendar/style.css'
import { useI18n } from '../i18n'

const props = withDefaults(defineProps<{
  modelValue: [string, string] | ''
  startPlaceholder?: string
  endPlaceholder?: string
  clearable?: boolean
}>(), {
  startPlaceholder: '',
  endPlaceholder: '',
  clearable: true,
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: [string, string] | ''): void
  (e: 'change', value: [string, string] | null): void
}>()

const { locale } = useI18n()
const vcLocale = computed(() => locale.value === 'zh' ? 'zh-CN' : 'en-US')

// Dark mode detection — project uses .dark class on <html>
const isDark = ref(false)
let darkObserver: MutationObserver | null = null

function checkDark() {
  isDark.value = document.documentElement.classList.contains('dark')
}

onMounted(() => {
  checkDark()
  darkObserver = new MutationObserver(checkDark)
  darkObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
})

onUnmounted(() => {
  darkObserver?.disconnect()
})

// --- State ---
const open = ref(false)
const triggerRef = ref<HTMLElement | null>(null)
const panelRef = ref<HTMLElement | null>(null)
const overlayRef = ref<HTMLElement | null>(null)
const panelStyle = ref<Record<string, string>>({})
const clickCount = ref(0)

// Internal range for VCalendar: { start: Date, end: Date } | null
const internalRange = ref<{ start: Date; end: Date } | null>(null)

// --- Computed ---
const hasValue = computed(() => {
  return Array.isArray(props.modelValue) && props.modelValue[0] && props.modelValue[1]
})

const displayStart = computed(() => {
  if (Array.isArray(props.modelValue) && props.modelValue[0]) return props.modelValue[0]
  return ''
})

const displayEnd = computed(() => {
  if (Array.isArray(props.modelValue) && props.modelValue[1]) return props.modelValue[1]
  return ''
})

// --- Sync prop → internal ---
watch(() => props.modelValue, (val) => {
  if (Array.isArray(val) && val[0] && val[1]) {
    internalRange.value = {
      start: parseDate(val[0]),
      end: parseDate(val[1]),
    }
  } else {
    internalRange.value = null
  }
}, { immediate: true })

// --- Helpers ---
function parseDate(str: string): Date {
  const [y, m, d] = str.split('-').map(Number)
  return new Date(y, m - 1, d)
}

function formatDate(date: Date): string {
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

function positionPanel() {
  if (!triggerRef.value) return
  const rect = triggerRef.value.getBoundingClientRect()
  const panelWidth = 260
  const panelHeight = 320
  let top = rect.bottom + 8
  let left = rect.left

  // Adjust if overflowing right
  if (left + panelWidth > window.innerWidth - 16) {
    left = window.innerWidth - panelWidth - 16
  }
  // Adjust if overflowing bottom
  if (top + panelHeight > window.innerHeight - 16) {
    top = rect.top - panelHeight - 8
  }

  panelStyle.value = {
    position: 'fixed',
    top: `${top}px`,
    left: `${left}px`,
    zIndex: '10000',
  }
}

// --- Actions ---
function toggle() {
  if (open.value) {
    close()
  } else {
    openPanel()
  }
}

function openPanel() {
  clickCount.value = 0
  open.value = true
  nextTick(() => {
    positionPanel()
    overlayRef.value?.focus()
  })
}

function close() {
  open.value = false
}

function clear() {
  internalRange.value = null
  emit('update:modelValue', '')
  emit('change', null)
}

function onDayClick() {
  clickCount.value++
  // After two clicks (start + end selected), emit and close
  if (clickCount.value >= 2) {
    nextTick(() => {
      if (internalRange.value) {
        const start = formatDate(internalRange.value.start)
        const end = formatDate(internalRange.value.end)
        emit('update:modelValue', [start, end])
        emit('change', [start, end])
      }
      close()
    })
  }
}
</script>

<style scoped>
.date-range-trigger {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 32px;
  padding: 0 10px;
  border-radius: 12px;
  width: 100%;
  background: rgba(243, 244, 246, 0.6);
  border: 1px solid rgba(0, 0, 0, 0.06);
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    border-color: rgba(0, 113, 227, 0.25);
    background: rgba(235, 237, 240, 0.8);
  }

  .dark & {
    background: rgba(255, 255, 255, 0.06);
    border-color: rgba(255, 255, 255, 0.08);
  }
  .dark &:hover {
    background: rgba(255, 255, 255, 0.1);
    border-color: rgba(0, 113, 227, 0.25);
  }
}
.date-text {
  font-size: 12px;
  font-weight: 500;
  color: #111827;

  .dark & {
    color: #e5e7eb;
  }
}
.date-placeholder {
  font-size: 12px;
  font-weight: 500;
  color: #9ca3af;

  .dark & {
    color: #6b7280;
  }
}
.date-separator {
  font-size: 12px;
  color: #9ca3af;
  padding: 0 2px;
}
.clear-btn {
  margin-left: auto;
  color: #9ca3af;
  transition: color 0.15s;
  display: flex;
  align-items: center;

  &:hover {
    color: #4b5563;
  }
  .dark &:hover {
    color: #d1d5db;
  }
}

/* Calendar panel — matches popup-panel design token */
.calendar-panel {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 12px;
  padding: 8px;
  box-shadow: 0 12px 40px -4px rgba(0, 0, 0, 0.12), 0 4px 12px -2px rgba(0, 0, 0, 0.06);

  .dark & {
    background: rgba(38, 38, 42, 0.95);
    border-color: rgba(255, 255, 255, 0.1);
    box-shadow: 0 12px 40px -4px rgba(0, 0, 0, 0.4), 0 4px 12px -2px rgba(0, 0, 0, 0.2);
  }
}

/* Transition */
.calendar-pop-enter-active {
  transition: all 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.calendar-pop-leave-active {
  transition: all 0.12s ease;
}
.calendar-pop-enter-from {
  opacity: 0;
  transform: scale(0.95) translateY(-4px);
}
.calendar-pop-leave-to {
  opacity: 0;
  transform: scale(0.97);
}
</style>

<style>
/* VCalendar overrides — aligned with project popup-panel / filter-pill tokens */

/* Apple Blue accent */
.vc-blue {
  --vc-accent-50: rgba(0, 113, 227, 0.05);
  --vc-accent-100: rgba(0, 113, 227, 0.1);
  --vc-accent-200: rgba(0, 113, 227, 0.2);
  --vc-accent-300: rgba(0, 113, 227, 0.35);
  --vc-accent-400: rgba(0, 113, 227, 0.5);
  --vc-accent-500: #0071e3;
  --vc-accent-600: #0071e3;
  --vc-accent-700: #005bb5;
  --vc-accent-800: #004a94;
  --vc-accent-900: #003a75;
}

/* Container: transparent, no border, inherit font */
.calendar-panel .vc-container {
  font-family: inherit;
  background: transparent !important;
  border: none !important;
  color: inherit;
}

/* Reset all VC focus / outline noise */
.calendar-panel :focus { outline: none; box-shadow: none; }

/* ─── Calendar header (month title + arrows) ─── */
.calendar-panel .vc-header { padding: 2px 2px 6px; }

.calendar-panel .vc-title {
  font-size: 12px;
  font-weight: 600;
  color: #4b5563;
  border-radius: 8px;
  padding: 0 8px;
  height: 26px;
  transition: all 0.15s;
  .dark & { color: #d1d5db; }
}
.calendar-panel .vc-title:hover {
  background: rgba(0, 0, 0, 0.04);
  color: #111827;
  .dark & { background: rgba(255, 255, 255, 0.06); color: #fff; }
}

.calendar-panel .vc-arrow {
  border-radius: 8px;
  width: 26px;
  height: 26px;
  color: #9ca3af;
  transition: all 0.15s;
}
.calendar-panel .vc-arrow:hover {
  background: rgba(0, 0, 0, 0.04);
  color: #6b7280;
  .dark & { background: rgba(255, 255, 255, 0.06); color: #d1d5db; }
}

/* ─── Weekday headers ─── */
.calendar-panel .vc-weekday {
  font-size: 10px;
  font-weight: 500;
  color: #9ca3af;
  padding-bottom: 2px;
  .dark & { color: #6b7280; }
}

/* ─── Day cells ─── */
.calendar-panel .vc-weeks { padding: 0 2px; }

.calendar-panel .vc-day-content {
  font-size: 12px;
  font-weight: 500;
  width: 30px;
  height: 30px;
  line-height: 30px;
  border-radius: 8px;
  color: #374151;
  transition: all 0.15s;
  .dark & { color: #d1d5db; }
}
.calendar-panel .vc-day-content:hover {
  background: rgba(0, 0, 0, 0.04);
  .dark & { background: rgba(255, 255, 255, 0.06); }
}

/* Today */
.calendar-panel .vc-day.is-today .vc-day-content {
  color: #0071e3;
  font-weight: 600;
  .dark & { color: #5ba7f5; }
}

/* Off-month days */
.calendar-panel .is-not-in-month .vc-day-content {
  color: #d1d5db;
  .dark & { color: #4b5563; }
}

/* ─── Range highlights ─── */

/* All range bases: full-width */
.calendar-panel .vc-highlight.vc-highlight-base-start,
.calendar-panel .vc-highlight.vc-highlight-base-end,
.calendar-panel .vc-highlight.vc-highlight-base-middle {
  width: 100% !important;
}

/* Middle: flat continuous bar */
.calendar-panel .vc-highlight.vc-highlight-base-middle { border-radius: 0 !important; }
/* Start: rounded-left */
.calendar-panel .vc-highlight.vc-highlight-base-start { border-radius: 8px 0 0 8px !important; }
/* End: rounded-right */
.calendar-panel .vc-highlight.vc-highlight-base-end { border-radius: 0 8px 8px 0 !important; }

/* Row boundaries */
.calendar-panel .vc-weeks .vc-day:nth-child(7n+1) .vc-highlight.vc-highlight-base-middle { border-radius: 8px 0 0 8px !important; }
.calendar-panel .vc-weeks .vc-day:nth-child(7n) .vc-highlight.vc-highlight-base-middle { border-radius: 0 8px 8px 0 !important; }
/* Edge: start on last col / end on first col → full round */
.calendar-panel .vc-weeks .vc-day:nth-child(7n) .vc-highlight.vc-highlight-base-start { border-radius: 8px !important; }
.calendar-panel .vc-weeks .vc-day:nth-child(7n+1) .vc-highlight.vc-highlight-base-end { border-radius: 8px !important; }

/* Range bar color */
.calendar-panel .vc-highlight-bg-light {
  background: rgba(0, 113, 227, 0.08) !important;
  border-radius: inherit !important;
  .dark & { background: rgba(0, 113, 227, 0.15) !important; }
}
.calendar-panel .vc-highlight-content-light {
  color: #0071e3 !important;
  font-weight: 500;
  .dark & { color: #5ba7f5 !important; }
}

/* Selected start/end */
.calendar-panel .vc-highlight-bg-solid {
  border-radius: 8px !important;
  background: #0071e3 !important;
}
.calendar-panel .vc-highlight-content-solid {
  color: #fff !important;
  font-weight: 600;
}

/* Drag preview */
.calendar-panel .vc-highlight-bg-outline {
  border-radius: 8px !important;
  border-color: rgba(0, 113, 227, 0.25) !important;
  background: rgba(0, 113, 227, 0.04) !important;
}

/* ─── Month/Year navigation (click title) ─── */
.calendar-panel .vc-nav-header { padding: 0 2px 4px; }

.calendar-panel .vc-nav-title,
.calendar-panel .vc-nav-arrow {
  border-radius: 8px;
  height: 26px;
  transition: all 0.15s;
}
.calendar-panel .vc-nav-title {
  font-size: 12px;
  font-weight: 600;
  color: #4b5563;
  padding: 0 8px;
  .dark & { color: #d1d5db; }
}
.calendar-panel .vc-nav-title:hover {
  background: rgba(0, 0, 0, 0.04);
  color: #111827;
  .dark & { background: rgba(255, 255, 255, 0.06); color: #fff; }
}
.calendar-panel .vc-nav-arrow {
  width: 26px;
  color: #9ca3af;
}
.calendar-panel .vc-nav-arrow:hover {
  background: rgba(0, 0, 0, 0.04);
  color: #6b7280;
  .dark & { background: rgba(255, 255, 255, 0.06); color: #d1d5db; }
}

.calendar-panel .vc-nav-items {
  grid-row-gap: 2px;
  grid-column-gap: 2px;
  margin-top: 2px;
}
.calendar-panel .vc-nav-item {
  font-size: 12px;
  font-weight: 500;
  color: #4b5563;
  border-radius: 8px;
  padding: 5px 0;
  width: auto;
  transition: all 0.15s;
  .dark & { color: #d1d5db; }
}
.calendar-panel .vc-nav-item:hover {
  background: rgba(0, 0, 0, 0.04);
  .dark & { background: rgba(255, 255, 255, 0.06); }
}
.calendar-panel .vc-nav-item.is-active {
  background: #0071e3 !important;
  color: #fff !important;
  font-weight: 600;
  box-shadow: none !important;
}
.calendar-panel .vc-nav-item.is-current:not(.is-active) {
  color: #0071e3 !important;
  font-weight: 600;
  .dark & { color: #5ba7f5 !important; }
}
</style>
