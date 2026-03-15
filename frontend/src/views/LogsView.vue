<template>
  <div class="space-y-6 animate-fade-in">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white tracking-tight">{{ t('logs.title') }}</h1>
      <button
        :disabled="loading || refreshDone"
        :class="[
          'group inline-flex items-center gap-2 px-4 py-2 rounded-2xl text-[13px] font-medium backdrop-blur-md border transition-all duration-300 disabled:opacity-50',
          refreshDone
            ? 'bg-emerald-500/10 dark:bg-emerald-500/8 border-emerald-500/20 text-emerald-600 dark:text-emerald-400'
            : 'bg-white/30 dark:bg-white/6 border-white/25 dark:border-white/10 text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-white/50 dark:hover:bg-white/12 hover:border-white/40 dark:hover:border-white/20 hover:shadow-lg'
        ]"
        @click="refresh"
      >
        <svg v-if="refreshDone" class="w-3.5 h-3.5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
        </svg>
        <svg
          v-else
          :class="['w-3.5 h-3.5 transition-transform duration-500', loading ? 'animate-spin' : 'group-hover:rotate-180']"
          xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" />
        </svg>
        <span>{{ refreshDone ? t('dashboard.refresh.done') : t('logs.refresh') }}</span>
      </button>
    </div>

    <!-- Filter Bar -->
    <div class="glass-card px-4 py-3 flex items-center gap-3 relative z-10">
      <!-- ID search -->
      <div class="filter-pill h-[32px] flex-1 min-w-[80px]">
        <svg class="w-3.5 h-3.5 text-gray-400 shrink-0" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" /></svg>
        <input v-model="filters.runId" :placeholder="t('logs.filter.id.placeholder')" class="flex-1 bg-transparent outline-none text-[12px] font-medium min-w-0 text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500" @input="applyFilters()" />
      </div>

      <!-- Account dropdown -->
      <div ref="accountDropdownRef" class="relative flex-1 min-w-[120px]">
        <button
          class="filter-pill h-[32px] w-full"
          :class="filters.accountId ? 'text-gray-900 dark:text-white' : 'text-gray-400 dark:text-gray-500'"
          @click="accountDropdownOpen = !accountDropdownOpen"
        >
          <span class="flex-1 text-left truncate">{{ selectedAccountLabel }}</span>
          <svg class="w-3.5 h-3.5 text-gray-400 shrink-0 transition-transform duration-200" :class="accountDropdownOpen ? 'rotate-180' : ''" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" /></svg>
        </button>
        <Transition name="dropdown">
          <div v-if="accountDropdownOpen" class="popup-panel w-full min-w-[180px] max-h-[240px] overflow-y-auto scrollbar-hide">
            <button
              :class="['popup-item', !filters.accountId ? 'text-apple-blue bg-apple-blue/8' : '']"
              @click="filters.accountId = ''; accountDropdownOpen = false; applyFilters()"
            >
              {{ t('logs.filter.account.all') }}
            </button>
            <button
              v-for="acc in accountOptions"
              :key="acc.id"
              :class="['popup-item', filters.accountId === String(acc.id) ? 'text-apple-blue bg-apple-blue/8' : '']"
              @click="filters.accountId = String(acc.id); accountDropdownOpen = false; applyFilters()"
            >
              {{ acc.name }}
            </button>
          </div>
        </Transition>
      </div>

      <!-- Trigger type toggle -->
      <div class="flex gap-1 p-0.5 rounded-xl bg-gray-100/60 dark:bg-white/6 border border-black/[0.06] dark:border-white/8 h-[32px] items-center shrink-0">
        <button
          v-for="opt in triggerOptions" :key="opt.value"
          :class="['toggle-btn', filters.triggerType === opt.value ? triggerActiveClass(opt.value) : 'toggle-idle']"
          @click="filters.triggerType = opt.value; applyFilters()"
        >
          {{ t(opt.label) }}
        </button>
      </div>

      <!-- Status toggle -->
      <div class="flex gap-1 p-0.5 rounded-xl bg-gray-100/60 dark:bg-white/6 border border-black/[0.06] dark:border-white/8 h-[32px] items-center shrink-0">
        <button
          v-for="opt in statusOptions" :key="opt.value"
          :class="['toggle-btn', filters.status === opt.value ? statusActiveClass(opt.value) : 'toggle-idle']"
          @click="filters.status = opt.value; applyFilters()"
        >
          {{ t(opt.label) }}
        </button>
      </div>

      <!-- Date Range -->
      <div class="flex-1 min-w-[200px]">
        <DateRangePicker
          v-model="dateRange"
          :start-placeholder="t('logs.filter.dateRange.start')"
          :end-placeholder="t('logs.filter.dateRange.end')"
          @change="onDateRangeChange"
        />
      </div>
    </div>

    <!-- Table -->
    <div v-if="runs.length" class="glass-card overflow-hidden">
      <div class="overflow-x-auto">
        <!-- Header -->
        <div class="hidden md:grid items-center gap-3 px-5 py-3 bg-gray-100/50 dark:bg-white/4 border-b border-gray-200/60 dark:border-white/8 text-[11px] font-semibold tracking-wide uppercase text-gray-500 dark:text-gray-400 min-w-[900px] log-grid select-none">
          <span class="text-center">{{ t('logs.table.id') }}</span>
          <span class="text-center">{{ t('logs.table.account') }}</span>
          <span class="text-center">{{ t('logs.table.triggerType') }}</span>
          <span class="text-center">{{ t('logs.table.startedAt') }}</span>
          <span class="text-center">{{ t('logs.table.endpoints') }}</span>
          <span class="text-center">{{ t('logs.table.status') }}</span>
          <span class="text-center">{{ t('logs.table.duration') }}</span>
          <span></span>
        </div>

        <!-- Rows -->
        <div v-for="(run, idx) in runs" :key="run.id">
          <div
            :class="['grid items-center gap-3 px-5 py-3.5 text-[13px] transition-all duration-200 min-w-[900px] log-grid', rowClass(run, idx)]"
          >
            <span class="justify-self-center px-2 py-0.5 rounded-lg text-[11px] font-medium tabular-nums bg-apple-blue/8 text-apple-blue/60 dark:text-apple-blue/50">#{{ run.id }}</span>
            <span class="flex items-center justify-center gap-1.5 min-w-0">
              <span class="text-gray-700 dark:text-gray-300 font-semibold truncate text-[13px]">{{ run.account_name }}</span>
              <span :class="['shrink-0 px-1.5 py-px rounded text-[9px] font-semibold tracking-wide uppercase', run.account_auth_type === 'auth_code' ? 'bg-blue-100/70 dark:bg-blue-900/25 text-blue-600 dark:text-blue-400' : 'bg-amber-100/70 dark:bg-amber-900/25 text-amber-600 dark:text-amber-400']">
                {{ t(`accounts.type.${run.account_auth_type}`) }}
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
            <span :class="['justify-self-center inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-md text-[11px] font-semibold', statusBadgeClass(run)]">
              <span :class="['w-1.5 h-1.5 rounded-full', statusDotClass(run)]"></span>
              {{ statusLabel(run) }}
            </span>
            <span class="text-gray-400 dark:text-gray-500 tabular-nums text-center text-[12px] font-mono">{{ formatDuration(run.started_at, run.finished_at) }}</span>
            <span class="justify-self-center flex items-center gap-2">
              <button
                class="inline-flex items-center gap-1 px-2 py-0.5 rounded-lg text-[11px] font-medium text-apple-blue bg-apple-blue/8 hover:bg-apple-blue/15 border border-apple-blue/15 hover:border-apple-blue/30 transition-all duration-200 whitespace-nowrap"
                @click.stop="loadEndpointDetails(run.id)"
              >{{ t('logs.table.detail') }}</button>
            </span>
          </div>
        </div>
      </div>

      <!-- Pagination -->
      <div class="flex items-center justify-between px-5 py-3 border-t border-gray-200/60 dark:border-white/8 bg-gray-50/30 dark:bg-white/2">
        <span class="text-[12px] text-gray-400 dark:text-gray-500">{{ t('logs.pagination.total').replace('{total}', String(total)) }}</span>
        <div class="flex items-center gap-1">
          <button :disabled="page <= 1" class="page-btn" @click="goPage(page - 1)">
            <svg class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 19.5L8.25 12l7.5-7.5" /></svg>
          </button>
          <button v-for="p in visiblePages" :key="p" :class="['w-8 h-8 rounded-lg text-[12px] font-medium transition-all duration-200', p === page ? 'bg-apple-blue text-white shadow-md shadow-apple-blue/30' : 'text-gray-500 dark:text-gray-400 hover:bg-white/40 dark:hover:bg-white/8']" @click="goPage(p)">{{ p }}</button>
          <button :disabled="page >= totalPages" class="page-btn" @click="goPage(page + 1)">
            <svg class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" /></svg>
          </button>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else-if="!loading" class="flex flex-col items-center justify-center py-20 glass-card rounded-2xl">
      <div class="w-16 h-16 rounded-2xl bg-apple-blue/10 flex items-center justify-center mb-5">
        <svg class="w-8 h-8 text-apple-blue" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
        </svg>
      </div>
      <h3 class="text-[15px] font-semibold text-gray-900 dark:text-white mb-1.5">{{ t('logs.empty') }}</h3>
      <p class="text-[13px] text-gray-400 dark:text-gray-500">{{ t('logs.empty.hint') }}</p>
    </div>

    <!-- Endpoint Detail Drawer -->
    <Teleport to="body">
      <Transition name="drawer-overlay">
        <div
          v-if="showEndpointDrawer"
          class="fixed inset-0 z-50 flex justify-end"
          @keydown.esc="closeEndpointDrawer"
        >
          <!-- Backdrop -->
          <div class="absolute inset-0 bg-black/30 dark:bg-black/50 backdrop-blur-sm" @click="closeEndpointDrawer" />

          <!-- Drawer Panel -->
          <Transition name="drawer-panel" appear>
            <div
              v-if="showEndpointDrawer"
              class="relative w-full max-w-xl h-full flex flex-col backdrop-blur-[40px] bg-white/90 dark:bg-[rgb(38,38,38)]/90 border-l border-white/25 dark:border-white/10 shadow-2xl shadow-black/12 dark:shadow-black/40"
            >
              <!-- Header -->
              <div class="flex items-center justify-between px-6 pt-5 pb-4 border-b border-gray-100/60 dark:border-white/6 shrink-0">
                <div class="flex items-center gap-3">
                  <div class="w-8 h-8 rounded-xl bg-apple-blue/10 flex items-center justify-center">
                    <svg class="w-4 h-4 text-apple-blue" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M8.25 6.75h12M8.25 12h12m-12 5.25h12M3.75 6.75h.007v.008H3.75V6.75zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zM3.75 12h.007v.008H3.75V12zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm-.375 5.25h.007v.008H3.75v-.008zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z" />
                    </svg>
                  </div>
                  <div>
                    <h2 class="text-[15px] font-semibold text-gray-900 dark:text-white">{{ t('logs.drawer.title') }}</h2>
                    <p class="text-[12px] text-gray-400 dark:text-gray-500">{{ t('logs.drawer.runId').replace('{id}', String(selectedTaskLogId)) }}</p>
                  </div>
                </div>
                <button
                  class="w-8 h-8 flex items-center justify-center rounded-full text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 hover:bg-gray-100/60 dark:hover:bg-white/8 transition-all duration-200"
                  @click="closeEndpointDrawer"
                >
                  <svg class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>

              <!-- Body -->
              <div class="flex-1 overflow-y-auto scrollbar-hide px-6 py-4 space-y-2">
                <!-- Loading -->
                <div v-if="drawerLoading" class="space-y-2 py-2">
                  <div v-for="i in 4" :key="i" class="rounded-xl px-4 py-3 bg-gray-50/60 dark:bg-white/4 space-y-2">
                    <div class="h-4 rounded-md shimmer" style="width: 60%"></div>
                    <div class="h-3 rounded-md shimmer" style="width: 40%"></div>
                  </div>
                </div>

                <!-- Empty -->
                <div v-else-if="!endpointDetails.length" class="flex flex-col items-center justify-center py-16 gap-3">
                  <div class="w-12 h-12 rounded-xl bg-gray-100/60 dark:bg-white/6 flex items-center justify-center">
                    <svg class="w-6 h-6 text-gray-300 dark:text-gray-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25" />
                    </svg>
                  </div>
                  <p class="text-[13px] text-gray-400 dark:text-gray-500">{{ t('logs.drawer.empty') }}</p>
                </div>

                <!-- Endpoint list -->
                <template v-else>
                  <!-- Summary row -->
                  <div class="flex items-center gap-2 mb-1">
                    <span class="text-[12px] text-gray-400 dark:text-gray-500">
                      {{ t('logs.drawer.summary')
                        .replace('{success}', String(endpointDetails.filter(e => e.success).length))
                        .replace('{total}', String(endpointDetails.length)) }}
                    </span>
                  </div>

                  <div
                    v-for="ep in endpointDetails"
                    :key="ep.id"
                    :class="[
                      'rounded-xl border transition-colors duration-150',
                      ep.success
                        ? 'bg-white/60 dark:bg-white/4 border-gray-100/60 dark:border-white/6'
                        : 'bg-red-50/50 dark:bg-red-900/8 border-red-100/50 dark:border-red-500/10'
                    ]"
                  >
                    <!-- Main row -->
                    <div class="flex items-center gap-3 px-4 py-3">
                      <!-- Status dot + name -->
                      <span :class="['w-2 h-2 rounded-full shrink-0', ep.success ? 'bg-emerald-500' : 'bg-red-500']"></span>
                      <div class="flex-1 min-w-0">
                        <span
                          v-if="ep.scope"
                          class="text-[13px] font-medium text-gray-700 dark:text-gray-300 truncate cursor-pointer active:opacity-60 transition-opacity duration-100 inline-flex items-center gap-1.5"
                          :title="t('logs.error.copy')"
                          @click.stop="copyScope(ep.scope)"
                        >{{ ep.scope }}<svg v-if="copiedScope === ep.scope" class="w-3 h-3 text-emerald-500 shrink-0" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" /></svg></span>
                        <span v-else class="text-[13px] font-medium text-gray-700 dark:text-gray-300 truncate block">{{ ep.endpoint_name }}</span>
                        <span v-if="ep.scope" class="text-[11px] text-gray-400 dark:text-gray-500 font-mono truncate block">{{ ep.endpoint_name }}</span>
                      </div>

                      <!-- HTTP status -->
                      <span
                        :class="[
                          'text-[11px] font-semibold tabular-nums px-1.5 py-0.5 rounded-md',
                          !ep.http_status ? 'text-gray-300 dark:text-gray-600 bg-transparent'
                          : ep.http_status >= 400 ? 'text-red-600 dark:text-red-400 bg-red-100/60 dark:bg-red-900/20'
                            : 'text-emerald-600 dark:text-emerald-400 bg-emerald-100/60 dark:bg-emerald-900/20'
                        ]"
                      >{{ ep.http_status || '-' }}</span>

                      <!-- Success / fail badge -->
                      <span
                        :class="[
                          'inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-semibold',
                          ep.success
                            ? 'bg-emerald-100/60 dark:bg-emerald-900/20 text-emerald-600 dark:text-emerald-400'
                            : 'bg-red-100/60 dark:bg-red-900/20 text-red-600 dark:text-red-400'
                        ]"
                      >
                        <svg v-if="ep.success" class="w-3 h-3" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                          <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd" />
                        </svg>
                        <svg v-else class="w-3 h-3" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                          <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clip-rule="evenodd" />
                        </svg>
                        {{ ep.success ? t('logs.table.success') : t('logs.table.failed') }}
                      </span>

                      <!-- Executed at -->
                      <span class="text-[11px] text-gray-400 dark:text-gray-500 tabular-nums shrink-0">{{ formatTime(ep.executed_at) }}</span>
                    </div>

                    <!-- Failure details: single toggle to expand error + response body -->
                    <div v-if="!ep.success && (ep.error_message || ep.response_body)" class="px-4 pb-3 pt-1 ml-5">
                      <button
                        class="drawer-toggle-btn"
                        data-testid="toggle-error"
                        @click="toggleDrawerError(ep.id)"
                      >
                        <svg :class="['w-3 h-3 transition-transform duration-150', drawerExpandedErrors.has(ep.id) ? 'rotate-90' : '']" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" /></svg>
                        <span>{{ t('logs.drawer.errorMessage') }}</span>
                      </button>
                      <div v-if="drawerExpandedErrors.has(ep.id)" class="mt-1.5 space-y-2 border-t border-red-100/40 dark:border-red-500/8 pt-2">
                        <div v-if="ep.error_message" class="text-[12px] text-red-500/80 dark:text-red-400/80 leading-relaxed break-words">
                          {{ ep.error_message }}
                        </div>
                        <div v-if="ep.response_body" class="rounded-lg bg-red-50/60 dark:bg-red-900/10 border border-red-100/50 dark:border-red-500/8 px-3 py-2">
                          <pre class="text-[11px] text-red-600/80 dark:text-red-400/70 font-mono break-all whitespace-pre-wrap overflow-hidden leading-relaxed">{{ ep.response_body }}</pre>
                        </div>
                      </div>
                    </div>
                  </div>
                </template>
              </div>
            </div>
          </Transition>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import DateRangePicker from '../components/DateRangePicker.vue'
import { useI18n } from '../i18n'
import { apiClient } from '../api/client'

const { t } = useI18n()

// --- Types ---
interface EndpointDetail {
  id: number
  endpoint_name: string
  scope: string
  http_status: number
  success: boolean
  error_message: string
  response_body: string
  executed_at: string
}

interface TaskLogItem {
  id: number
  account_id: number
  account_name: string
  account_auth_type: string
  run_id: string
  trigger_type: 'scheduled' | 'manual'
  total_endpoints: number
  success_count: number
  fail_count: number
  started_at: string
  finished_at: string | null
  created_at: string
}

interface AccountOption {
  id: number
  name: string
  auth_type: string
}

// --- State ---
const loading = ref(false)
const runs = ref<TaskLogItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const accountOptions = ref<AccountOption[]>([])

const filters = reactive({ runId: '', accountId: '', triggerType: '', status: '', dateFrom: '', dateTo: '' })

const triggerOptions = [
  { value: '', label: 'logs.filter.triggerType.all' },
  { value: 'scheduled', label: 'logs.filter.triggerType.scheduled' },
  { value: 'manual', label: 'logs.filter.triggerType.manual' },
]
const statusOptions = [
  { value: '', label: 'logs.filter.status.all' },
  { value: 'success', label: 'logs.filter.status.success' },
  { value: 'partial', label: 'logs.filter.status.partial' },
  { value: 'failed', label: 'logs.filter.status.failed' },
]

// --- Account Dropdown ---
const accountDropdownOpen = ref(false)
const accountDropdownRef = ref<HTMLElement | null>(null)
const selectedAccountLabel = computed(() => {
  if (!filters.accountId) return t('logs.filter.account.all')
  return accountOptions.value.find(a => String(a.id) === filters.accountId)?.name || filters.accountId
})

// --- Date Range (Element Plus) ---
const dateRange = ref<[string, string] | ''>('')

function onDateRangeChange(val: [string, string] | null) {
  if (val && val[0] && val[1]) {
    filters.dateFrom = val[0]
    filters.dateTo = val[1]
  } else {
    filters.dateFrom = ''
    filters.dateTo = ''
  }
  applyFilters()
}

// --- Click Outside ---
function handleClickOutside(e: MouseEvent) {
  const target = e.target as Node
  if (accountDropdownRef.value && !accountDropdownRef.value.contains(target)) accountDropdownOpen.value = false
}

// --- Endpoint Detail Drawer ---
const endpointDetails = ref<EndpointDetail[]>([])
const showEndpointDrawer = ref(false)
const drawerLoading = ref(false)
const selectedTaskLogId = ref<number | null>(null)

async function loadEndpointDetails(taskLogId: number) {
  selectedTaskLogId.value = taskLogId
  drawerLoading.value = true
  endpointDetails.value = []
  drawerExpandedErrors.value = new Set()
  drawerExpandedBodies.value = new Set()
  try {
    const res = await apiClient.get(`/logs/${taskLogId}/endpoints`)
    endpointDetails.value = res.data || []
    showEndpointDrawer.value = true
  } finally {
    drawerLoading.value = false
  }
}

function closeEndpointDrawer() {
  showEndpointDrawer.value = false
}

// --- Drawer collapse toggles ---
const drawerExpandedErrors = ref<Set<number>>(new Set())
const drawerExpandedBodies = ref<Set<number>>(new Set())

const copiedScope = ref<string | null>(null)
let copiedTimer: ReturnType<typeof setTimeout> | null = null

function copyScope(scope: string) {
  navigator.clipboard.writeText(scope)
  copiedScope.value = scope
  if (copiedTimer) clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => { copiedScope.value = null }, 1500)
}

function toggleDrawerError(epId: number) {
  const s = new Set(drawerExpandedErrors.value)
  if (s.has(epId)) {
    s.delete(epId)
    // Also collapse response_body when collapsing error
    const b = new Set(drawerExpandedBodies.value)
    b.delete(epId)
    drawerExpandedBodies.value = b
  } else {
    s.add(epId)
  }
  drawerExpandedErrors.value = s
}

// --- Pagination ---
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const visiblePages = computed(() => {
  const pages: number[] = []
  const tp = totalPages.value
  let start = Math.max(1, page.value - 2)
  const end = Math.min(tp, start + 4)
  start = Math.max(1, end - 4)
  for (let i = start; i <= end; i++) pages.push(i)
  return pages
})
function goPage(p: number) { if (p >= 1 && p <= totalPages.value) { page.value = p; fetchRuns() } }
function applyFilters() { page.value = 1; fetchRuns() }
const refreshDone = ref(false)
let refreshDoneTimer: ReturnType<typeof setTimeout> | undefined
function refresh() {
  fetchRuns()
  refreshDone.value = true
  clearTimeout(refreshDoneTimer)
  refreshDoneTimer = setTimeout(() => { refreshDone.value = false }, 1500)
}

// --- Toggle color helpers ---
function triggerActiveClass(value: string) {
  if (value === 'scheduled') return 'toggle-active-violet'
  if (value === 'manual') return 'toggle-active-sky'
  return 'toggle-active'
}
function statusActiveClass(value: string) {
  if (value === 'success') return 'toggle-active-emerald'
  if (value === 'partial') return 'toggle-active-amber'
  if (value === 'failed') return 'toggle-active-red'
  return 'toggle-active'
}

// --- Status helpers ---
function runStatus(run: TaskLogItem): 'success' | 'partial' | 'failed' {
  if (run.fail_count === 0) return 'success'
  if (run.success_count > 0) return 'partial'
  return 'failed'
}
function statusLabel(run: TaskLogItem) { const s = runStatus(run); return t(s === 'success' ? 'logs.table.success' : s === 'partial' ? 'logs.table.partial' : 'logs.table.failed') }
function statusBadgeClass(run: TaskLogItem) { const s = runStatus(run); if (s === 'success') return 'bg-emerald-100/60 dark:bg-emerald-900/20 text-emerald-600 dark:text-emerald-400'; if (s === 'partial') return 'bg-amber-100/60 dark:bg-amber-900/20 text-amber-600 dark:text-amber-400'; return 'bg-red-100/60 dark:bg-red-900/20 text-red-600 dark:text-red-400' }
function statusDotClass(run: TaskLogItem) { const s = runStatus(run); return s === 'success' ? 'bg-emerald-500' : s === 'partial' ? 'bg-amber-500' : 'bg-red-500' }
function rowClass(run: TaskLogItem, idx: number) {
  const s = runStatus(run)
  if (s === 'partial') return 'bg-amber-50/30 dark:bg-amber-900/6 border-b border-gray-200/40 dark:border-white/6 hover:bg-amber-50/50 dark:hover:bg-amber-900/10'
  if (s === 'failed') return 'bg-red-50/40 dark:bg-red-900/8 border-b border-gray-200/40 dark:border-white/6 hover:bg-red-50/60 dark:hover:bg-red-900/12'
  return (idx % 2 === 1 ? 'bg-gray-50/40 dark:bg-white/2 ' : '') + 'border-b border-gray-200/40 dark:border-white/6 hover:bg-white/50 dark:hover:bg-white/5'
}

async function fetchRuns() {
  loading.value = true
  try {
    const params: Record<string, string | number> = { page: page.value, page_size: pageSize }
    if (filters.runId) params.id = filters.runId
    if (filters.accountId) params.account_id = filters.accountId
    if (filters.triggerType) params.trigger_type = filters.triggerType
    if (filters.status) params.status = filters.status
    if (filters.dateFrom) params.date_from = filters.dateFrom
    if (filters.dateTo) params.date_to = filters.dateTo
    const res = await apiClient.get('/logs', { params })
    runs.value = res.data.items || []; total.value = res.data.total || 0
  } catch { runs.value = []; total.value = 0 }
  finally { loading.value = false }
}

async function fetchAccounts() {
  try { const res = await apiClient.get('/accounts'); accountOptions.value = (res.data || []).map((a: any) => ({ id: a.id, name: a.name, auth_type: a.auth_type || '' })) }
  catch { accountOptions.value = [] }
}

function pad2(n: number) { return String(n).padStart(2, '0') }

function formatTime(iso: string) {
  if (!iso) return '-'
  const d = new Date(iso)
  return `${pad2(d.getMonth() + 1)}-${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}:${pad2(d.getSeconds())}`
}
function formatDuration(start: string, end: string | null) {
  if (!start || !end) return '-'
  const ms = new Date(end).getTime() - new Date(start).getTime()
  if (ms < 1000) return `${ms}ms`
  const secs = Math.floor(ms / 1000)
  if (secs < 60) return `${secs}.${String(ms % 1000).slice(0, 1)}s`
  return `${Math.floor(secs / 60)}m ${secs % 60}s`
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  // Read query params (e.g. from dashboard click)
  const route = useRoute()
  if (route.query.id) filters.runId = String(route.query.id)
  fetchAccounts(); fetchRuns()
})
onUnmounted(() => { document.removeEventListener('click', handleClickOutside) })
</script>

<style scoped>
/* Glass card */
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

/* Filter pill */
.filter-pill {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
  background: rgba(243, 244, 246, 0.6);
  border: 1px solid rgba(0, 0, 0, 0.06);
  transition: all 0.2s;
  cursor: pointer;
  .dark & { background: rgba(255, 255, 255, 0.06); border-color: rgba(255, 255, 255, 0.08); }
  .dark &:hover { background: rgba(255, 255, 255, 0.1); border-color: rgba(0, 113, 227, 0.25); }
}
.filter-pill:hover { border-color: rgba(0, 113, 227, 0.25); background: rgba(235, 237, 240, 0.8); }

/* Toggle buttons */
.toggle-btn {
  padding: 5px 10px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 500;
  transition: all 0.2s;
  white-space: nowrap;
}
.toggle-active {
  background: white;
  color: #111827;
  box-shadow: 0 1px 3px rgba(0,0,0,0.08);
  .dark & { background: rgba(255,255,255,0.15); color: white; }
}
/* Colored active states — trigger type */
.toggle-active-violet {
  background: rgba(139, 92, 246, 0.12); color: #7c3aed; box-shadow: 0 1px 3px rgba(139, 92, 246, 0.15);
  .dark & { background: rgba(139, 92, 246, 0.2); color: #a78bfa; }
}
.toggle-active-sky {
  background: rgba(14, 165, 233, 0.12); color: #0284c7; box-shadow: 0 1px 3px rgba(14, 165, 233, 0.15);
  .dark & { background: rgba(14, 165, 233, 0.2); color: #38bdf8; }
}
/* Colored active states — status */
.toggle-active-emerald {
  background: rgba(16, 185, 129, 0.12); color: #059669; box-shadow: 0 1px 3px rgba(16, 185, 129, 0.15);
  .dark & { background: rgba(16, 185, 129, 0.2); color: #34d399; }
}
.toggle-active-amber {
  background: rgba(245, 158, 11, 0.12); color: #d97706; box-shadow: 0 1px 3px rgba(245, 158, 11, 0.15);
  .dark & { background: rgba(245, 158, 11, 0.2); color: #fbbf24; }
}
.toggle-active-red {
  background: rgba(239, 68, 68, 0.12); color: #dc2626; box-shadow: 0 1px 3px rgba(239, 68, 68, 0.15);
  .dark & { background: rgba(239, 68, 68, 0.2); color: #f87171; }
}
.toggle-idle {
  color: #9ca3af;
  .dark & { color: #6b7280; }
  .dark &:hover { color: #d1d5db; }
}
.toggle-idle:hover { color: #4b5563; }

/* Popup panel */
.popup-panel {
  position: absolute;
  top: 100%;
  left: 0;
  margin-top: 4px;
  z-index: 50;
  padding: 4px 0;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid rgba(0, 0, 0, 0.08);
  box-shadow: 0 12px 40px -4px rgba(0, 0, 0, 0.12), 0 4px 12px -2px rgba(0, 0, 0, 0.06);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  .dark & {
    background: rgba(38, 38, 42, 0.95);
    border-color: rgba(255, 255, 255, 0.1);
    box-shadow: 0 12px 40px -4px rgba(0, 0, 0, 0.4), 0 4px 12px -2px rgba(0, 0, 0, 0.2);
  }
}

.popup-item {
  display: block;
  width: 100%;
  text-align: left;
  padding: 8px 12px;
  font-size: 12px;
  font-weight: 500;
  color: #4b5563;
  transition: background 0.15s, color 0.15s;
  .dark & { color: #d1d5db; }
  .dark &:hover { background: rgba(255, 255, 255, 0.06); }
}
.popup-item:hover { background: rgba(0, 0, 0, 0.04); }

/* Page button */
.page-btn {
  padding: 6px 10px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 500;
  color: #6b7280;
  transition: background 0.2s;
  .dark & { color: #9ca3af; }
  .dark &:hover { background: rgba(255, 255, 255, 0.06); }
}
.page-btn:hover { background: rgba(0, 0, 0, 0.04); }
.page-btn:disabled { opacity: 0.3; pointer-events: none; }

/* Grids */
.log-grid { grid-template-columns: 1fr 3fr 1.5fr 2.5fr 1.5fr 2fr 1.5fr 1.6fr; }

/* Drawer toggle button */
.drawer-toggle-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: rgba(239, 68, 68, 0.6);
  transition: background 0.15s, color 0.15s;
  cursor: pointer;
  .dark & { color: rgba(239, 68, 68, 0.5); }
  .dark &:hover { background: rgba(239, 68, 68, 0.12); color: rgba(248, 113, 113, 0.8); }
}
.drawer-toggle-btn:hover {
  background: rgba(239, 68, 68, 0.08);
  color: rgba(239, 68, 68, 0.8);
}

/* Shimmer */
.shimmer {
  background-color: rgba(0, 0, 0, 0.06);
  background-image: linear-gradient(90deg, transparent 25%, rgba(255,255,255,0.5) 50%, transparent 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  .dark & {
    background-color: rgba(255, 255, 255, 0.06);
    background-image: linear-gradient(90deg, transparent 25%, rgba(255,255,255,0.04) 50%, transparent 75%);
  }
}
@keyframes shimmer { from { background-position: 200% 0; } to { background-position: -200% 0; } }

/* Scrollbar hide */
:global(.scrollbar-hide) { scrollbar-width: none; -ms-overflow-style: none; }
:global(.scrollbar-hide::-webkit-scrollbar) { display: none; }

/* Dropdown transition */
.dropdown-enter-active { transition: all 0.15s ease-out; }
.dropdown-leave-active { transition: all 0.1s ease-in; }
.dropdown-enter-from { opacity: 0; transform: translateY(-4px) scale(0.97); }
.dropdown-leave-to { opacity: 0; transform: translateY(-2px) scale(0.98); }

/* Drawer transitions */
.drawer-overlay-enter-active,
.drawer-overlay-leave-active {
  transition: opacity 0.2s ease;
}
.drawer-overlay-enter-from,
.drawer-overlay-leave-to {
  opacity: 0;
}
.drawer-panel-enter-active {
  transition: transform 0.28s cubic-bezier(0.22, 1, 0.36, 1);
}
.drawer-panel-leave-active {
  transition: transform 0.2s ease-in;
}
.drawer-panel-enter-from,
.drawer-panel-leave-to {
  transform: translateX(100%);
}

</style>
