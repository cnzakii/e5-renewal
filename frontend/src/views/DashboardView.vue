<template>
  <div class="space-y-6 animate-fade-in">
    <!-- Header: Title + Period Selector + Refresh -->
    <div class="flex items-center justify-between flex-wrap gap-4">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white tracking-tight">{{ t('dashboard.title') }}</h1>
      <div class="flex items-center gap-3">
        <div class="grid grid-cols-4 gap-1 p-1 rounded-2xl bg-white/30 dark:bg-white/6 backdrop-blur-md border border-white/25 dark:border-white/10 min-w-[420px]">
          <button
            v-for="p in periods"
            :key="p.value"
            :class="[
              'w-full px-2 py-1.5 rounded-xl text-[13px] font-medium transition-all duration-300 text-center whitespace-nowrap',
              period === p.value
                ? 'bg-apple-blue text-white shadow-md shadow-apple-blue/30'
                : 'text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-white/50 dark:hover:bg-white/8'
            ]"
            @click="switchPeriod(p.value)"
          >
            {{ t(p.label) }}
          </button>
        </div>
        <button
          :disabled="loading || refreshDone"
          :class="[
            'group flex items-center gap-2 px-4 py-2 rounded-2xl text-[13px] font-medium backdrop-blur-md border transition-all duration-300 disabled:opacity-50',
            refreshDone
              ? 'bg-emerald-500/10 dark:bg-emerald-500/8 border-emerald-500/20 text-emerald-600 dark:text-emerald-400'
              : 'bg-white/30 dark:bg-white/6 border-white/25 dark:border-white/10 text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-white/50 dark:hover:bg-white/12 hover:border-white/40 dark:hover:border-white/20 hover:shadow-lg'
          ]"
          @click="refresh"
        >
          <!-- 成功勾 -->
          <svg v-if="refreshDone" class="w-3.5 h-3.5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
          </svg>
          <!-- 加载中旋转 -->
          <svg
            v-else
            :class="['w-3.5 h-3.5 transition-transform duration-500', loading ? 'animate-spin' : 'group-hover:rotate-180']"
            xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" />
          </svg>
          <span>{{ refreshDone ? t('dashboard.refresh.done') : t('dashboard.refresh') }}</span>
        </button>
      </div>
    </div>

    <!-- Stat Cards -->
    <div :key="'stats-' + period" :class="['grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4 transition-opacity duration-300', loading ? 'opacity-50' : 'opacity-100']">
      <div
        v-for="(stat, i) in statCards"
        :key="stat.key"
        :class="['glass-card glass-card-hover p-5 flex items-center gap-4 stagger-item group cursor-default', `stagger-${i + 1}`]"
      >
        <div :class="['w-11 h-11 rounded-2xl flex items-center justify-center shrink-0 transition-transform duration-300 group-hover:scale-110', stat.iconBg]">
          <component :is="stat.icon" class="w-5 h-5" />
        </div>
        <div>
          <p class="text-[13px] text-apple-gray transition-colors duration-200 group-hover:text-gray-600 dark:group-hover:text-gray-300">{{ t(stat.label) }}</p>
          <p :class="['text-2xl font-semibold mt-0.5 tracking-tight transition-colors duration-200', stat.valueClass ? stat.valueClass() : 'text-gray-900 dark:text-white']">
            {{ stat.format(data) }}
          </p>
        </div>
      </div>
    </div>

    <!-- Charts Row -->
    <div class="grid grid-cols-1 lg:grid-cols-5 gap-4">
      <!-- Trend Chart (3/5) -->
      <div class="lg:col-span-3 glass-card glass-card-hover p-5">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('dashboard.chart.trend') }}</h3>
        </div>
        <v-chart ref="trendChartRef" :option="trendOption" autoresize style="width: 100%; height: 256px;" />
      </div>

      <!-- Account Overview (2/5) -->
      <div class="lg:col-span-2 glass-card glass-card-hover p-5 flex flex-col">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('dashboard.accounts.title') }}</h3>
          <div class="flex items-center gap-3">
            <span class="flex items-center gap-1.5 text-[11px] text-gray-400 dark:text-gray-500">
              <span class="w-2 h-2 rounded-full bg-blue-500"></span> {{ t('dashboard.auth.authCode') }}
            </span>
            <span class="flex items-center gap-1.5 text-[11px] text-gray-400 dark:text-gray-500">
              <span class="w-2 h-2 rounded-full bg-amber-500"></span> {{ t('dashboard.auth.credentials') }}
            </span>
          </div>
        </div>
        <div class="flex-1 space-y-2 overflow-y-auto max-h-[400px] pr-1 hide-scrollbar">
          <router-link
            v-for="acc in data.account_health"
            :key="acc.id"
            :to="{ path: `${pathPrefix}/accounts`, query: { highlight: String(acc.id) } }"
            class="flex items-center gap-3 px-3.5 py-3 rounded-xl bg-white/30 dark:bg-white/5 border border-white/15 dark:border-white/6 hover:bg-white/50 dark:hover:bg-white/10 hover:border-apple-blue/30 dark:hover:border-apple-blue/20 hover:shadow-md hover:shadow-apple-blue/5 transition-all duration-200 group/acc no-underline"
          >
            <span :class="['w-2.5 h-2.5 rounded-full shrink-0', acc.auth_type === 'auth_code' ? 'bg-blue-500' : 'bg-amber-500']"></span>
            <span class="text-[13px] font-medium text-gray-800 dark:text-gray-200 flex-1 truncate group-hover/acc:text-apple-blue transition-colors duration-200">{{ acc.name }}</span>
            <div class="w-20 flex items-center gap-2">
              <div class="flex-1 h-1.5 rounded-full bg-gray-200/60 dark:bg-gray-700/60 overflow-hidden">
                <div
                  class="h-full rounded-full transition-all duration-500"
                  :class="healthBarColor(acc.health)"
                  :style="{ width: `${acc.health}%` }"
                ></div>
              </div>
            </div>
            <span :class="['text-[13px] font-semibold tabular-nums w-12 text-right', healthTextColor(acc.health)]">
              {{ acc.health.toFixed(1) }}%
            </span>
            <span :class="['text-xs', acc.health >= 90 ? 'text-emerald-500' : acc.health >= 70 ? 'text-amber-500' : 'text-red-500']">
              <svg v-if="acc.health >= 90" xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd" /></svg>
              <svg v-else-if="acc.health >= 70" xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 6a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 6zm0 9a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" /></svg>
              <svg v-else xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clip-rule="evenodd" /></svg>
            </span>
            <svg class="w-3.5 h-3.5 text-gray-300 dark:text-gray-600 group-hover/acc:text-apple-blue transition-colors duration-200 shrink-0" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" /></svg>
          </router-link>
          <div v-if="!data.account_health.length" class="flex-1 flex items-center justify-center py-12 text-sm text-gray-400">
            {{ t('dashboard.noData') }}
          </div>
        </div>
      </div>
    </div>

    <!-- Recent Task Runs -->
    <div class="glass-card glass-card-hover p-5">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('dashboard.logs.title') }}</h3>
        <router-link
          :to="`${pathPrefix}/logs`"
          class="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-[13px] font-medium text-apple-blue bg-apple-blue/8 hover:bg-apple-blue/15 border border-apple-blue/15 hover:border-apple-blue/30 transition-all duration-200 no-underline group/link"
        >
          {{ t('dashboard.logs.viewAll') }}
          <svg class="w-3.5 h-3.5 transition-transform duration-200 group-hover/link:translate-x-0.5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" /></svg>
        </router-link>
      </div>
      <div class="overflow-x-auto">
        <!-- Header -->
        <div class="hidden md:grid items-center gap-3 px-4 py-3 bg-gray-100/50 dark:bg-white/4 border-b border-gray-200/60 dark:border-white/8 text-[11px] font-semibold tracking-wide uppercase text-gray-500 dark:text-gray-400 min-w-[800px] recent-run-grid select-none">
          <span class="text-center">ID</span>
          <span class="text-center">{{ t('dashboard.logs.account') }}</span>
          <span class="text-center">{{ t('logs.table.triggerType') }}</span>
          <span class="text-center">{{ t('dashboard.logs.time') }}</span>
          <span class="text-center">{{ t('dashboard.logs.endpoint') }}</span>
          <span class="text-center">{{ t('dashboard.logs.status') }}</span>
          <span class="text-center">{{ t('dashboard.runs.duration') }}</span>
        </div>
        <!-- Rows -->
        <div
          v-for="(run, idx) in data.recent_runs"
          :key="run.id"
          :class="['grid items-center gap-3 px-4 py-3 text-[13px] transition-all duration-200 min-w-[800px] cursor-pointer recent-run-grid', runRowClass(run, idx)]"
          @click="goToRun(run.id)"
        >
          <span class="justify-self-center px-2 py-0.5 rounded-lg text-[11px] font-medium tabular-nums bg-apple-blue/8 text-apple-blue/60 dark:text-apple-blue/50">#{{ run.id }}</span>
          <span class="flex items-center justify-center gap-1.5 min-w-0">
            <span class="text-gray-700 dark:text-gray-300 font-semibold truncate text-[13px]">{{ run.account_name }}</span>
            <span :class="['shrink-0 px-1.5 py-px rounded text-[9px] font-semibold tracking-wide uppercase', run.account_auth_type === 'auth_code' ? 'bg-blue-100/70 dark:bg-blue-900/25 text-blue-600 dark:text-blue-400' : 'bg-amber-100/70 dark:bg-amber-900/25 text-amber-600 dark:text-amber-400']">
              {{ run.account_auth_type === 'auth_code' ? t('dashboard.logs.authCode') : t('dashboard.logs.credentials') }}
            </span>
          </span>
          <span :class="['justify-self-center px-2.5 py-0.5 rounded-md text-[10px] font-semibold tracking-wide uppercase', run.trigger_type === 'scheduled' ? 'bg-violet-100/70 dark:bg-violet-900/25 text-violet-600 dark:text-violet-400' : 'bg-sky-100/70 dark:bg-sky-900/25 text-sky-600 dark:text-sky-400']">
            {{ run.trigger_type === 'scheduled' ? t('logs.table.scheduled') : t('logs.table.manual') }}
          </span>
          <span class="text-gray-500 dark:text-gray-400 tabular-nums text-center truncate text-[12px]">{{ formatTime(run.started_at) }}</span>
          <span class="text-center tabular-nums">
            <span class="text-emerald-600 dark:text-emerald-400 font-semibold">{{ run.success_count }}</span>
            <span class="text-gray-300 dark:text-gray-600 mx-0.5">/</span>
            <span class="text-gray-500 dark:text-gray-400">{{ run.total_endpoints }}</span>
          </span>
          <span :class="['justify-self-center inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-md text-[11px] font-semibold', runStatusBadgeClass(run)]">
            <span :class="['w-1.5 h-1.5 rounded-full', runStatusDotClass(run)]"></span>
            {{ runStatusLabel(run) }}
          </span>
          <span class="text-gray-400 dark:text-gray-500 tabular-nums text-center text-[12px] font-mono">{{ formatDuration(run.started_at, run.finished_at) }}</span>
        </div>
        <div v-if="!data.recent_runs.length" class="flex items-center justify-center py-12 text-sm text-gray-400">
          {{ t('dashboard.noData') }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h, type FunctionalComponent } from 'vue'
import { useRouter } from 'vue-router'
import { useDark } from '@vueuse/core'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent, GridComponent } from 'echarts/components'
import VChart from 'vue-echarts'

import { apiClient } from '../api/client'
import { useI18n } from '../i18n'

use([CanvasRenderer, LineChart, TooltipComponent, LegendComponent, GridComponent])

const { t } = useI18n()
const router = useRouter()
const pathPrefix = import.meta.env.VITE_PATH_PREFIX || ''

interface AccountHealth {
  id: number
  name: string
  auth_type: 'auth_code' | 'client_credentials'
  health: number
  total_runs: number
  success_runs: number
  last_run: string
}

interface TrendItem {
  date: string
  total_requests: number
}

interface RecentRun {
  id: number
  account_name: string
  account_auth_type: string
  trigger_type: 'scheduled' | 'manual'
  total_endpoints: number
  success_count: number
  fail_count: number
  started_at: string
  finished_at: string
}

interface AccountScheduleInfo {
  enabled: boolean
  paused: boolean
}

interface AccountWithSchedule {
  id: number
  schedule?: AccountScheduleInfo
}

interface DashboardData {
  total_accounts: number
  success_rate: number
  total_runs: number
  error_count: number
  auth_code_count: number
  credentials_count: number
  trend: TrendItem[]
  account_health: AccountHealth[]
  recent_runs: RecentRun[]
  active_schedules: number
  total_schedules: number
  avg_health: number
}

const period = ref('7d')
const loading = ref(false)
const data = ref<DashboardData>({
  total_accounts: 0,
  success_rate: 0,
  total_runs: 0,
  error_count: 0,
  auth_code_count: 0,
  credentials_count: 0,
  trend: [],
  account_health: [],
  recent_runs: [],
  active_schedules: 0,
  total_schedules: 0,
  avg_health: 0,
})
const trendChartRef = ref<InstanceType<typeof VChart> | null>(null)

const periods = [
  { value: '1d', label: 'dashboard.period.1d' },
  { value: '7d', label: 'dashboard.period.7d' },
  { value: '30d', label: 'dashboard.period.30d' },
  { value: 'all', label: 'dashboard.period.all' },
]

function switchPeriod(p: string) {
  period.value = p
  fetchData()
}

const refreshDone = ref(false)
let refreshDoneTimer: ReturnType<typeof setTimeout> | undefined

async function refresh() {
  await fetchData()
  refreshDone.value = true
  clearTimeout(refreshDoneTimer)
  refreshDoneTimer = setTimeout(() => { refreshDone.value = false }, 1500)
}

async function fetchData() {
  if (loading.value) return
  loading.value = true

  const params = { period: period.value }
  const timeout = 5000
  try {
    const [summaryRes, trendRes, healthRes, recentRes, accountsRes] = await Promise.all([
      apiClient.get('/dashboard/summary', { params, timeout }),
      apiClient.get('/dashboard/trend', { params, timeout }),
      apiClient.get('/dashboard/account-health', { timeout }),
      apiClient.get('/dashboard/recent-logs', { timeout }),
      apiClient.get('/accounts', { timeout }),
    ])
    const accountsList: AccountWithSchedule[] = accountsRes.data || []
    const healthList: AccountHealth[] = healthRes.data || []

    // Compute active schedules
    let activeSchedules = 0
    let totalSchedules = 0
    for (const acc of accountsList) {
      if (acc.schedule) {
        totalSchedules++
        if (acc.schedule.enabled && !acc.schedule.paused) activeSchedules++
      }
    }

    // Compute average health
    let avgHealth = 0
    if (healthList.length > 0) {
      avgHealth = +(healthList.reduce((sum, a) => sum + a.health, 0) / healthList.length).toFixed(1)
    }

    data.value = {
      ...summaryRes.data,
      trend: trendRes.data || [],
      account_health: healthList,
      recent_runs: recentRes.data || [],
      active_schedules: activeSchedules,
      total_schedules: totalSchedules,
      avg_health: avgHealth,
    }
  } catch {
    // keep current data on error
  } finally {
    loading.value = false
  }
}

onMounted(() => fetchData())

function formatTime(iso: string): string {
  if (!iso) return '-'
  const d = new Date(iso)
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  const ss = String(d.getSeconds()).padStart(2, '0')
  return `${mm}-${dd} ${hh}:${mi}:${ss}`
}

// --- Task run helpers ---
function runStatus(run: RecentRun): 'success' | 'partial' | 'failed' {
  if (run.fail_count === 0) return 'success'
  if (run.success_count > 0) return 'partial'
  return 'failed'
}
function runStatusLabel(run: RecentRun) {
  const s = runStatus(run)
  return t(s === 'success' ? 'dashboard.logs.success' : s === 'partial' ? 'logs.table.partial' : 'dashboard.logs.failed')
}
function runStatusBadgeClass(run: RecentRun) {
  const s = runStatus(run)
  if (s === 'success') return 'bg-emerald-100/60 dark:bg-emerald-900/20 text-emerald-600 dark:text-emerald-400'
  if (s === 'partial') return 'bg-amber-100/60 dark:bg-amber-900/20 text-amber-600 dark:text-amber-400'
  return 'bg-red-100/60 dark:bg-red-900/20 text-red-600 dark:text-red-400'
}
function runStatusDotClass(run: RecentRun) {
  const s = runStatus(run)
  return s === 'success' ? 'bg-emerald-500' : s === 'partial' ? 'bg-amber-500' : 'bg-red-500'
}
function runRowClass(run: RecentRun, idx: number) {
  const s = runStatus(run)
  if (s === 'partial') return 'bg-amber-50/30 dark:bg-amber-900/6 border-b border-gray-200/40 dark:border-white/6 hover:bg-amber-50/50 dark:hover:bg-amber-900/10'
  if (s === 'failed') return 'bg-red-50/40 dark:bg-red-900/8 border-b border-gray-200/40 dark:border-white/6 hover:bg-red-50/60 dark:hover:bg-red-900/12'
  return (idx % 2 === 1 ? 'bg-gray-50/40 dark:bg-white/2 ' : '') + 'border-b border-gray-200/40 dark:border-white/6 hover:bg-white/50 dark:hover:bg-white/5'
}
function formatDuration(start: string, end: string) {
  if (!start || !end) return '-'
  const ms = new Date(end).getTime() - new Date(start).getTime()
  if (ms < 1000) return `${ms}ms`
  const secs = Math.floor(ms / 1000)
  if (secs < 60) return `${secs}.${String(ms % 1000).slice(0, 1)}s`
  return `${Math.floor(secs / 60)}m ${secs % 60}s`
}
function goToRun(id: number) {
  router.push({ path: `${pathPrefix}/logs`, query: { id: String(id) } })
}

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

// Custom tooltip formatter
function tooltipFormatter(params: any): string {
  const items = Array.isArray(params) ? params : [params]
  const title = items[0]?.axisValue || ''
  const dark = isDark.value
  const titleColor = dark ? '#e5e7eb' : '#374151'
  const labelColor = dark ? '#9ca3af' : '#6b7280'
  const valueColor = dark ? '#f3f4f6' : '#111827'
  let html = `<div style="font-family:-apple-system,BlinkMacSystemFont,'SF Pro Display',sans-serif;padding:4px 2px">`
  html += `<div style="font-size:12px;font-weight:600;color:${titleColor};margin-bottom:8px">${title}</div>`
  for (const item of items) {
    html += `<div style="display:flex;align-items:center;gap:8px;margin-bottom:4px">`
    html += `<span style="width:8px;height:8px;border-radius:50%;background:${item.color};display:inline-block"></span>`
    html += `<span style="font-size:12px;color:${labelColor};flex:1">${item.seriesName}</span>`
    html += `<span style="font-size:13px;font-weight:600;color:${valueColor};font-variant-numeric:tabular-nums">${item.value?.toLocaleString()}</span>`
    html += `</div>`
  }
  html += `</div>`
  return html
}

// Stat card icons
const IconUsers: FunctionalComponent = () =>
  h('svg', { xmlns: 'http://www.w3.org/2000/svg', viewBox: '0 0 24 24', fill: 'currentColor' }, [
    h('path', { d: 'M7.5 6.5C7.5 8.433 8.567 10 10 10s2.5-1.567 2.5-3.5S11.433 3 10 3 7.5 4.567 7.5 6.5zM14.5 8a2 2 0 104 0 2 2 0 00-4 0zM3 18c0-3.5 3.5-6 7-6s7 2.5 7 6v1H3v-1zm14-1c0-.37-.037-.73-.107-1.08C18.2 14.7 19.5 14 21 14c2 0 3 1.5 3 3v1h-4.5a5.9 5.9 0 01-2.5-1z' }),
  ])
const IconCheck: FunctionalComponent = () =>
  h('svg', { xmlns: 'http://www.w3.org/2000/svg', viewBox: '0 0 24 24', fill: 'currentColor' }, [
    h('path', { 'fill-rule': 'evenodd', 'clip-rule': 'evenodd', d: 'M2.25 12c0-5.385 4.365-9.75 9.75-9.75s9.75 4.365 9.75 9.75-4.365 9.75-9.75 9.75S2.25 17.385 2.25 12zm13.36-1.814a.75.75 0 10-1.22-.872l-3.236 4.53L9.53 12.22a.75.75 0 00-1.06 1.06l2.25 2.25a.75.75 0 001.14-.094l3.75-5.25z' }),
  ])
const IconRuns: FunctionalComponent = () =>
  h('svg', { xmlns: 'http://www.w3.org/2000/svg', viewBox: '0 0 24 24', fill: 'currentColor' }, [
    h('path', { d: 'M18.375 2.25c-1.035 0-1.875.84-1.875 1.875v15.75c0 1.035.84 1.875 1.875 1.875s1.875-.84 1.875-1.875V4.125c0-1.036-.84-1.875-1.875-1.875zM12 7.5c-1.035 0-1.875.84-1.875 1.875v10.5c0 1.035.84 1.875 1.875 1.875s1.875-.84 1.875-1.875V9.375c0-1.036-.84-1.875-1.875-1.875zM5.625 12c-1.036 0-1.875.84-1.875 1.875v6c0 1.035.84 1.875 1.875 1.875s1.875-.84 1.875-1.875v-6c0-1.036-.84-1.875-1.875-1.875z' }),
  ])
const IconAlert: FunctionalComponent = () =>
  h('svg', { xmlns: 'http://www.w3.org/2000/svg', viewBox: '0 0 24 24', fill: 'currentColor' }, [
    h('path', { 'fill-rule': 'evenodd', 'clip-rule': 'evenodd', d: 'M9.401 3.003c1.155-2 4.043-2 5.197 0l7.355 12.748c1.154 2-.29 4.499-2.599 4.499H4.645c-2.309 0-3.752-2.5-2.598-4.5L9.4 3.004zM12 8.25a.75.75 0 01.75.75v3.75a.75.75 0 01-1.5 0V9a.75.75 0 01.75-.75zm0 8.25a.75.75 0 100-1.5.75.75 0 000 1.5z' }),
  ])
const IconSchedule: FunctionalComponent = () =>
  h('svg', { xmlns: 'http://www.w3.org/2000/svg', viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '1.5' }, [
    h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z' }),
  ])
const IconHeart: FunctionalComponent = () =>
  h('svg', { xmlns: 'http://www.w3.org/2000/svg', viewBox: '0 0 24 24', fill: 'currentColor' }, [
    h('path', { d: 'M21 8.25c0-2.485-2.099-4.5-4.688-4.5-1.935 0-3.597 1.126-4.312 2.733-.715-1.607-2.377-2.733-4.313-2.733C5.1 3.75 3 5.765 3 8.25c0 7.22 9 12 9 12s9-4.78 9-12z' }),
  ])

const statCards = computed(() => [
  { key: 'accounts', label: 'dashboard.stat.accounts', icon: IconUsers, iconBg: 'bg-blue-500/10 dark:bg-blue-500/15 text-blue-500 dark:text-blue-400', format: (d: DashboardData) => d.total_accounts.toLocaleString() },
  { key: 'success_rate', label: 'dashboard.stat.successRate', icon: IconCheck, iconBg: 'bg-emerald-500/10 dark:bg-emerald-500/15 text-emerald-500 dark:text-emerald-400', format: (d: DashboardData) => `${d.success_rate.toFixed(1)}%` },
  { key: 'total_runs', label: 'dashboard.stat.totalRuns', icon: IconRuns, iconBg: 'bg-violet-500/10 dark:bg-violet-500/15 text-violet-500 dark:text-violet-400', format: (d: DashboardData) => d.total_runs.toLocaleString() },
  { key: 'errors', label: 'dashboard.stat.errors', icon: IconAlert, iconBg: 'bg-rose-500/10 dark:bg-rose-500/15 text-rose-500 dark:text-rose-400', format: (d: DashboardData) => d.error_count.toLocaleString(), valueClass: () => data.value.error_count > 0 ? 'text-red-500' : 'text-gray-900 dark:text-white' },
  { key: 'active_schedules', label: 'dashboard.stat.activeSchedules', icon: IconSchedule, iconBg: 'bg-cyan-500/10 dark:bg-cyan-500/15 text-cyan-500 dark:text-cyan-400', format: (d: DashboardData) => `${d.active_schedules}/${d.total_schedules}` },
  { key: 'avg_health', label: 'dashboard.stat.systemHealth', icon: IconHeart, iconBg: 'bg-pink-500/10 dark:bg-pink-500/15 text-pink-500 dark:text-pink-400', format: (d: DashboardData) => `${d.avg_health.toFixed(1)}%`, valueClass: () => data.value.avg_health >= 90 ? 'text-emerald-600 dark:text-emerald-400' : data.value.avg_health >= 70 ? 'text-amber-600 dark:text-amber-400' : 'text-red-500' },
])

// Chart — reactive dark mode via useDark
const isDark = useDark({ storageKey: 'e5-theme' })

const textColor = computed(() => isDark.value ? '#a1a1aa' : '#71717a')

const tooltipStyle = computed(() => ({
  backgroundColor: isDark.value ? 'rgba(30,30,30,0.9)' : 'rgba(255,255,255,0.85)',
  borderColor: isDark.value ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.06)',
  borderWidth: 1,
  borderRadius: 12,
  padding: [10, 14],
  shadowColor: isDark.value ? 'rgba(0,0,0,0.3)' : 'rgba(0,0,0,0.08)',
  shadowBlur: 16,
  shadowOffsetY: 4,
  extraCssText: 'backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);',
}))

const trendOption = computed(() => ({
  tooltip: { trigger: 'axis' as const, formatter: tooltipFormatter, ...tooltipStyle.value },
  grid: { left: 40, right: 16, top: 18, bottom: 24 },
  xAxis: {
    type: 'category' as const,
    data: data.value.trend.map(d => d.date),
    axisLabel: { color: textColor.value, fontSize: 11 },
    axisLine: { lineStyle: { color: isDark.value ? '#333' : '#e5e7eb' } },
  },
  yAxis: {
    type: 'value' as const,
    axisLabel: { color: textColor.value, fontSize: 11 },
    splitLine: { lineStyle: { color: isDark.value ? '#222' : '#f0f0f0' } },
  },
  series: [
    { name: t('dashboard.chart.totalRequests'), type: 'line', data: data.value.trend.map(d => d.total_requests), smooth: true, symbolSize: 6, itemStyle: { color: '#3b82f6' }, areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(59,130,246,0.25)' }, { offset: 1, color: 'rgba(59,130,246,0.02)' }] } } },
  ],
}))
</script>

<style scoped>
.glass-card {
  border-radius: 16px;
  backdrop-filter: blur(40px);
  -webkit-backdrop-filter: blur(40px);
  background: rgba(255, 255, 255, 0.55);
  border: 1px solid rgba(255, 255, 255, 0.2);
  transition: transform 0.3s cubic-bezier(0.34, 1.56, 0.64, 1),
              box-shadow 0.3s ease,
              border-color 0.3s ease,
              background 0.3s ease;
  .dark & {
    background: rgba(40, 40, 40, 0.55);
    border-color: rgba(255, 255, 255, 0.08);
  }
}
.glass-card-hover:hover {
  transform: translateY(-3px);
  box-shadow: 0 12px 40px -8px rgba(0, 0, 0, 0.1), 0 4px 16px -4px rgba(0, 0, 0, 0.06);
  border-color: rgba(255, 255, 255, 0.35);
  background: rgba(255, 255, 255, 0.65);
  .dark & {
    box-shadow: 0 12px 40px -8px rgba(0, 0, 0, 0.4), 0 0 20px -4px rgba(0, 113, 227, 0.08);
    border-color: rgba(255, 255, 255, 0.14);
    background: rgba(45, 45, 45, 0.7);
  }
}
.recent-run-grid { grid-template-columns: 1fr 3fr 1.5fr 2.5fr 1.5fr 2fr 1.5fr; }

.hide-scrollbar {
  scrollbar-width: none;
  -ms-overflow-style: none;
}
.hide-scrollbar::-webkit-scrollbar {
  display: none;
}
</style>
