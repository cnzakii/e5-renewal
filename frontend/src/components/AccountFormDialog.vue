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
            class="relative w-full max-w-lg max-h-[90vh] flex flex-col rounded-2xl backdrop-blur-[40px] bg-white/85 dark:bg-[rgb(38,38,38)]/85 border border-white/25 dark:border-white/10 shadow-2xl shadow-black/12 dark:shadow-black/40"
          >
            <!-- Header -->
            <div class="flex items-center justify-between px-6 pt-5 pb-4 shrink-0">
              <div class="flex items-center gap-3">
                <div class="w-9 h-9 rounded-xl flex items-center justify-center bg-apple-blue/10 text-apple-blue">
                  <!-- Preview icon — info circle -->
                  <svg v-if="dialogMode === 'preview'" class="w-4.5 h-4.5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z" />
                  </svg>
                  <!-- Edit icon -->
                  <svg v-else-if="isEdit" class="w-4.5 h-4.5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10" />
                  </svg>
                  <!-- Add icon -->
                  <svg v-else class="w-4.5 h-4.5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
                  </svg>
                </div>
                <h2 class="text-[16px] font-semibold text-gray-900 dark:text-white">
                  {{ dialogMode === 'preview' ? t('accounts.detail.title') : isEdit ? t('accounts.form.edit') : t('accounts.form.add') }}
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

            <!-- ========== PREVIEW MODE ========== -->
            <div v-if="dialogMode === 'preview' && account" class="flex-1 overflow-y-auto custom-scrollbar px-6 pb-4">
              <div class="space-y-4">
                <!-- Name + Auth type -->
                <div>
                  <h3 class="flex items-baseline gap-1.5 text-[15px] font-semibold text-gray-900 dark:text-white">
                    <span class="text-[13px] font-medium text-gray-400 dark:text-gray-500 tabular-nums">#{{ account.id }}</span>
                    <span class="truncate">{{ account.name }}</span>
                  </h3>
                  <span class="inline-flex items-center gap-1.5 mt-1.5 px-2 py-0.5 rounded-md text-[10px] font-semibold tracking-wide uppercase bg-gray-100/70 dark:bg-white/6 text-gray-500 dark:text-gray-400">
                    <span :class="['w-1.5 h-1.5 rounded-full shrink-0', account.auth_type === 'auth_code' ? 'bg-blue-500' : 'bg-amber-500']"></span>
                    {{ t(`accounts.type.${account.auth_type}`) }}
                  </span>
                </div>

                <!-- Divider -->
                <div class="flex items-center gap-3">
                  <div class="flex-1 h-px bg-gray-200/50 dark:bg-white/6"></div>
                  <span class="text-[10px] font-semibold uppercase tracking-wider text-gray-300 dark:text-gray-600">Microsoft Entra</span>
                  <div class="flex-1 h-px bg-gray-200/50 dark:bg-white/6"></div>
                </div>

                <!-- Credential Fields -->
                <div class="space-y-1">
                  <div class="preview-field">
                    <span class="preview-field-label">Client ID</span>
                    <span class="preview-field-value font-mono truncate">{{ account.client_id }}</span>
                  </div>
                  <div class="preview-field">
                    <span class="preview-field-label">Client Secret</span>
                    <span class="preview-field-value font-mono">{{ maskSecret(account.client_secret) }}</span>
                  </div>
                  <div class="preview-field">
                    <span class="preview-field-label">Tenant ID</span>
                    <span class="preview-field-value font-mono truncate">{{ account.tenant_id }}</span>
                  </div>
                  <div v-if="account.auth_type === 'auth_code'" class="preview-field">
                    <span class="preview-field-label">Refresh Token</span>
                    <span class="preview-field-value font-mono">{{ maskSecret(account.refresh_token) }}</span>
                  </div>
                </div>

                <!-- Expiry + Notification -->
                <div class="flex items-center gap-2 px-3 py-2.5 rounded-xl bg-gray-50/50 dark:bg-white/3">
                  <svg class="w-3.5 h-3.5 shrink-0 text-gray-400 dark:text-gray-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <span v-if="account.auth_expires_at" class="text-[12px] font-medium text-gray-600 dark:text-gray-400">
                    {{ account.auth_expires_at }}
                    <span class="ml-1 opacity-60">{{ previewExpiryRemainingText }}</span>
                  </span>
                  <span v-else class="text-[12px] font-medium text-gray-300 dark:text-gray-600">
                    {{ t('accounts.expiry.none') }}
                  </span>
                  <span v-if="account.notify_enabled" class="ml-auto flex items-center gap-1 text-[10px] font-medium text-gray-400 dark:text-gray-500">
                    <svg class="w-3 h-3" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M14.857 17.082a23.848 23.848 0 005.454-1.31A8.967 8.967 0 0118 9.75v-.7V9A6 6 0 006 9v.75a8.967 8.967 0 01-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 01-5.714 0m5.714 0a3 3 0 11-5.714 0" />
                    </svg>
                    {{ t('accounts.notify.on') }}
                  </span>
                </div>
              </div>
              <!-- Load secrets error -->
              <Transition name="field-error">
                <p v-if="formError" class="form-error mt-2">{{ formError }}</p>
              </Transition>
            </div>

            <!-- ========== EDIT MODE ========== -->
            <form v-else class="flex-1 overflow-y-auto custom-scrollbar px-6 pb-2" @submit.prevent="submit">
              <div class="space-y-5">
                <!-- Account Name -->
                <div>
                  <label class="form-label">{{ t('accounts.form.name') }}</label>
                  <input
                    v-model="form.name"
                    type="text"
                    :placeholder="t('accounts.form.name.placeholder')"
                    maxlength="120"
                    class="form-input"
                    :class="{ 'form-input-error': errors.name }"
                    @blur="validateField('name')"
                  />
                  <Transition name="field-error">
                    <p v-if="errors.name" class="form-error">{{ errors.name }}</p>
                  </Transition>
                </div>

                <!-- Notification toggle -->
                <div class="flex items-center justify-between px-3 py-3 rounded-xl bg-gray-50/50 dark:bg-white/3 border border-gray-100/50 dark:border-white/5">
                  <div class="flex items-center gap-2.5">
                    <svg class="w-4 h-4 text-gray-400 dark:text-gray-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M14.857 17.082a23.848 23.848 0 005.454-1.31A8.967 8.967 0 0118 9.75v-.7V9A6 6 0 006 9v.75a8.967 8.967 0 01-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 01-5.714 0m5.714 0a3 3 0 11-5.714 0" />
                    </svg>
                    <span class="text-[13px] font-medium text-gray-700 dark:text-gray-300">{{ t('accounts.form.notifyExpiry') }}</span>
                  </div>
                  <button
                    type="button"
                    :class="[
                      'relative w-11 h-6 rounded-full transition-colors duration-200 focus:outline-none',
                      form.notify_enabled
                        ? 'bg-apple-blue'
                        : 'bg-gray-300 dark:bg-gray-600'
                    ]"
                    @click="form.notify_enabled = !form.notify_enabled"
                  >
                    <span
                      :class="[
                        'absolute top-0.5 left-0.5 w-5 h-5 rounded-full bg-white shadow-sm transition-transform duration-200',
                        form.notify_enabled ? 'translate-x-5' : 'translate-x-0'
                      ]"
                    />
                  </button>
                </div>

                <!-- Auth Type -->
                <div>
                  <label class="form-label">{{ t('accounts.form.authType') }}</label>
                  <div class="grid grid-cols-2 gap-2.5">
                    <button
                      v-for="at in authTypes"
                      :key="at.value"
                      type="button"
                      :class="[
                        'relative flex items-center justify-center gap-2 px-4 py-3 rounded-xl text-[13px] font-medium border-2 transition-all duration-200',
                        form.auth_type === at.value
                          ? at.activeClass
                          : 'bg-white/40 dark:bg-white/4 border-gray-200/50 dark:border-white/8 text-gray-500 dark:text-gray-400 hover:bg-white/60 dark:hover:bg-white/6 hover:border-gray-300/60 dark:hover:border-white/12'
                      ]"
                      @click="form.auth_type = at.value; validateField('auth_type')"
                    >
                      <span :class="['w-2 h-2 rounded-full', at.dotClass]"></span>
                      {{ at.label }}
                    </button>
                  </div>
                  <Transition name="field-error">
                    <p v-if="errors.auth_type" class="form-error">{{ errors.auth_type }}</p>
                  </Transition>
                </div>

                <!-- Divider -->
                <div class="flex items-center gap-3">
                  <div class="flex-1 h-px bg-gray-200/50 dark:bg-white/6"></div>
                  <span class="text-[10px] font-semibold uppercase tracking-wider text-gray-300 dark:text-gray-600">Microsoft Entra</span>
                  <div class="flex-1 h-px bg-gray-200/50 dark:bg-white/6"></div>
                </div>

                <!-- Tenant ID -->
                <div>
                  <label class="form-label">Tenant ID</label>
                  <input
                    v-model="form.tenant_id"
                    type="text"
                    :placeholder="t('accounts.form.tenantId.placeholder')"
                    class="form-input"
                    :class="{ 'form-input-error': errors.tenant_id }"
                    @blur="validateField('tenant_id')"
                  />
                  <Transition name="field-error">
                    <p v-if="errors.tenant_id" class="form-error">{{ errors.tenant_id }}</p>
                  </Transition>
                </div>

                <!-- Client ID -->
                <div>
                  <label class="form-label">Client ID</label>
                  <input
                    v-model="form.client_id"
                    type="text"
                    :placeholder="t('accounts.form.clientId.placeholder')"
                    class="form-input font-mono text-[13px]"
                    :class="{ 'form-input-error': errors.client_id }"
                    @blur="validateField('client_id')"
                  />
                  <Transition name="field-error">
                    <p v-if="errors.client_id" class="form-error">{{ errors.client_id }}</p>
                  </Transition>
                </div>

                <!-- Client Secret -->
                <div>
                  <label class="form-label">Client Secret</label>
                  <div class="relative">
                    <input
                      v-model="form.client_secret"
                      :type="showSecret ? 'text' : 'password'"
                      :placeholder="t('accounts.form.clientSecret.placeholder')"
                      class="form-input pr-10"
                      :class="{ 'form-input-error': errors.client_secret }"
                      @blur="validateField('client_secret')"
                    />
                    <button
                      type="button"
                      class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
                      @click="showSecret = !showSecret"
                    >
                      <svg v-if="showSecret" class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178z" />
                        <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                      </svg>
                      <svg v-else class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M3.98 8.223A10.477 10.477 0 001.934 12C3.226 16.338 7.244 19.5 12 19.5c.993 0 1.953-.138 2.863-.395M6.228 6.228A10.45 10.45 0 0112 4.5c4.756 0 8.773 3.162 10.065 7.498a10.523 10.523 0 01-4.293 5.774M6.228 6.228L3 3m3.228 3.228l3.65 3.65m7.894 7.894L21 21m-3.228-3.228l-3.65-3.65m0 0a3 3 0 10-4.243-4.243m4.242 4.242L9.88 9.88" />
                      </svg>
                    </button>
                  </div>
                  <Transition name="field-error">
                    <p v-if="errors.client_secret" class="form-error">{{ errors.client_secret }}</p>
                  </Transition>
                </div>

                <!-- Refresh Token (auth_code only) -->
                <Transition name="field-slide">
                  <RefreshTokenModePanel
                    v-if="form.auth_type === 'auth_code'"
                    :client-id="form.client_id"
                    :client-secret="form.client_secret"
                    :tenant-id="form.tenant_id"
                    :refresh-token="form.refresh_token"
                    :auth-type="form.auth_type"
                    @update:refresh-token="form.refresh_token = $event"
                  />
                </Transition>
                <Transition name="field-error">
                  <p v-if="errors.refresh_token && form.auth_type === 'auth_code'" class="form-error">{{ errors.refresh_token }}</p>
                </Transition>

                <!-- Client Secret expiry reminder -->
                <div>
                  <label class="form-label">{{ t('accounts.form.expiresAt') }}</label>
                  <!-- auth_code: days input only -->
                  <div v-if="form.auth_type === 'auth_code' || !form.auth_type" class="flex items-center gap-2">
                    <input
                      :value="expiryDays ?? ''"
                      type="number"
                      min="1"
                      :placeholder="t('accounts.form.expiresAt.days.placeholder')"
                      class="form-input w-28 text-center tabular-nums"
                      @input="onExpiryDaysInput(($event.target as HTMLInputElement).value)"
                    />
                    <span class="shrink-0 whitespace-nowrap text-[13px] text-gray-500 dark:text-gray-400">{{ t('accounts.form.expiresAt.days.suffix') }}</span>
                  </div>
                  <!-- client_credentials: days input too -->
                  <div v-else class="flex items-center gap-2">
                    <input
                      :value="expiryDays ?? ''"
                      type="number"
                      min="1"
                      :placeholder="t('accounts.form.expiresAt.days.placeholder')"
                      class="form-input w-28 text-center tabular-nums"
                      @input="onExpiryDaysInput(($event.target as HTMLInputElement).value)"
                    />
                    <span class="shrink-0 whitespace-nowrap text-[13px] text-gray-500 dark:text-gray-400">{{ t('accounts.form.expiresAt.days.suffix') }}</span>
                  </div>
                  <!-- Hints -->
                  <p v-if="form.auth_type === 'auth_code'" class="mt-1.5 text-[11px] text-gray-400 dark:text-gray-500">
                    {{ t('accounts.form.expiresAt.hint.authCode') }}
                  </p>
                  <p class="mt-1 text-[11px] text-gray-400 dark:text-gray-500">
                    {{ t('accounts.form.expiresAt.hint.optional') }}
                  </p>
                  <!-- Show computed date -->
                  <p v-if="form.auth_expires_at" class="mt-1 text-[11px] text-gray-400 dark:text-gray-500">
                    {{ t('accounts.expiry') }}: {{ form.auth_expires_at }}
                  </p>
                </div>

                <!-- Verify Credentials -->
                <div>
                  <button
                    type="button"
                    :disabled="verifying"
                    :class="[
                      'inline-flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-[12px] font-medium transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed',
                      verifyResult === 'valid'
                        ? 'text-emerald-600 dark:text-emerald-400 bg-emerald-500/8 border border-emerald-500/15'
                        : verifyResult === 'error'
                          ? 'text-red-600 dark:text-red-400 bg-red-500/8 border border-red-500/15'
                          : 'text-apple-blue bg-apple-blue/8 hover:bg-apple-blue/15 border border-apple-blue/15 hover:border-apple-blue/30'
                    ]"
                    @click="verify"
                  >
                    <svg v-if="verifying" class="w-3.5 h-3.5 animate-spin" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" />
                    </svg>
                    <svg v-else-if="verifyResult === 'valid'" class="w-3.5 h-3.5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                      <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd" />
                    </svg>
                    <svg v-else-if="verifyResult === 'error'" class="w-3.5 h-3.5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                      <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clip-rule="evenodd" />
                    </svg>
                    <svg v-else class="w-3.5 h-3.5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z" />
                    </svg>
                    {{ verifying ? t('accounts.verify.checking') : verifyResult === 'valid' ? t('accounts.verify.valid') : verifyResult === 'error' ? t('accounts.verify.invalid') : t('accounts.verify') }}
                  </button>
                  <Transition name="field-error">
                    <p v-if="verifyResult === 'error' && verifyError" class="form-error mt-1.5">{{ verifyError }}</p>
                  </Transition>
                </div>
              </div>

              <!-- Bottom spacer -->
              <div class="h-4"></div>
            </form>

            <!-- Footer -->
            <div class="flex items-center justify-end gap-2.5 px-6 py-4 shrink-0 border-t border-gray-100/50 dark:border-white/5">
              <!-- Preview mode footer -->
              <template v-if="dialogMode === 'preview'">
                <button
                  class="px-5 py-2.5 rounded-xl text-[13px] font-medium text-gray-600 dark:text-gray-300 bg-gray-100/60 dark:bg-white/8 hover:bg-gray-200/60 dark:hover:bg-white/12 border border-gray-200/60 dark:border-white/10 transition-all duration-200"
                  @click="close"
                >
                  {{ t('accounts.form.cancel') }}
                </button>
                <button
                  :disabled="loadingSecrets"
                  class="inline-flex items-center gap-1.5 px-5 py-2.5 rounded-xl text-[13px] font-medium text-white bg-apple-blue hover:bg-apple-blue-hover border border-apple-blue/80 shadow-md shadow-apple-blue/20 transition-all duration-200 btn-shine disabled:opacity-60"
                  @click="enterEditMode"
                >
                  <svg class="w-3.5 h-3.5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10" />
                  </svg>
                  {{ t('accounts.action.edit') }}
                </button>
              </template>
              <!-- Edit mode footer -->
              <template v-else>
                <button
                  class="px-5 py-2.5 rounded-xl text-[13px] font-medium text-gray-600 dark:text-gray-300 bg-gray-100/60 dark:bg-white/8 hover:bg-gray-200/60 dark:hover:bg-white/12 border border-gray-200/60 dark:border-white/10 transition-all duration-200"
                  @click="close"
                >
                  {{ t('accounts.form.cancel') }}
                </button>
                <button
                  :disabled="saving"
                  class="px-5 py-2.5 rounded-xl text-[13px] font-medium text-white bg-apple-blue hover:bg-apple-blue-hover border border-apple-blue/80 shadow-md shadow-apple-blue/20 transition-all duration-200 disabled:opacity-60 btn-shine"
                  @click="submit"
                >
                  {{ saving ? t('accounts.form.saving') : t('accounts.form.save') }}
                </button>
              </template>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onUnmounted } from 'vue'
import { useI18n } from '../i18n'
import { apiClient } from '../api/client'
import { pathPrefix } from '../config'
import RefreshTokenModePanel from './RefreshTokenModePanel.vue'

export interface AccountFormData {
  name: string
  auth_type: 'auth_code' | 'client_credentials' | ''
  client_id: string
  client_secret: string
  tenant_id: string
  refresh_token: string
  notify_enabled: boolean
  auth_expires_at: string
}

export interface AccountSchedule {
  enabled: boolean
  paused: boolean
  pause_reason: string
  pause_threshold: number
  next_run_at: string | null
  last_run_at: string | null
}

export interface Account extends AccountFormData {
  id: number
  auth_type: 'auth_code' | 'client_credentials'
  health?: number
  total_runs?: number
  success_runs?: number
  last_run?: string
  schedule?: AccountSchedule
}

const props = defineProps<{
  visible: boolean
  account?: Account | null
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'save', data: AccountFormData): void
}>()

const { t } = useI18n()

const saving = ref(false)
const showSecret = ref(false)
const showRefreshToken = ref(false)
const dialogMode = ref<'preview' | 'edit'>('preview')
const loadingSecrets = ref(false)
const formError = ref('')

const isEdit = computed(() => !!props.account)

const authTypes = computed(() => [
  {
    value: 'auth_code' as const,
    label: t('accounts.type.auth_code'),
    activeClass: 'bg-blue-50/80 dark:bg-blue-900/15 border-blue-400 dark:border-blue-500/40 text-blue-600 dark:text-blue-400 shadow-sm shadow-blue-500/10',
    dotClass: 'bg-blue-500',
  },
  {
    value: 'client_credentials' as const,
    label: t('accounts.type.client_credentials'),
    activeClass: 'bg-amber-50/80 dark:bg-amber-900/15 border-amber-400 dark:border-amber-500/40 text-amber-600 dark:text-amber-400 shadow-sm shadow-amber-500/10',
    dotClass: 'bg-amber-500',
  },
])

const expiryDays = ref<number | null>(null)

const emptyForm = (): AccountFormData => ({
  name: '',
  auth_type: 'auth_code',
  client_id: '',
  client_secret: '',
  tenant_id: '',
  refresh_token: '',
  notify_enabled: false,
  auth_expires_at: '',
})

const form = reactive<AccountFormData>(emptyForm())
const errors = reactive<Record<string, string>>({})

function validateField(field: string): boolean {
  delete errors[field]

  if (field === 'name') {
    if (!form.name.trim()) { errors.name = t('accounts.form.error.name'); return false }
    if (form.name.length > 120) { errors.name = t('accounts.form.error.name.max'); return false }
  }
  if (field === 'auth_type') {
    if (!form.auth_type) { errors.auth_type = t('accounts.form.error.authType'); return false }
  }
  if (field === 'client_id') {
    if (!form.client_id.trim()) { errors.client_id = t('accounts.form.error.clientId'); return false }
  }
  if (field === 'client_secret') {
    if (!form.client_secret.trim()) { errors.client_secret = t('accounts.form.error.clientSecret'); return false }
  }
  if (field === 'tenant_id') {
    if (!form.tenant_id.trim()) { errors.tenant_id = t('accounts.form.error.tenantId'); return false }
  }
  if (field === 'refresh_token' && form.auth_type === 'auth_code') {
    if (!form.refresh_token.trim()) { errors.refresh_token = t('accounts.form.error.refreshToken'); return false }
  }
  return true
}

function validateAll(): boolean {
  const fields = ['name', 'auth_type', 'client_id', 'client_secret', 'tenant_id']
  if (form.auth_type === 'auth_code') fields.push('refresh_token')
  return fields.map(f => validateField(f)).every(Boolean)
}

function onExpiryDaysInput(val: string) {
  const n = parseInt(val, 10)
  if (!isNaN(n) && n > 0) {
    expiryDays.value = n
    const d = new Date()
    d.setDate(d.getDate() + n)
    form.auth_expires_at = d.toISOString().slice(0, 10)
  } else {
    expiryDays.value = null
    form.auth_expires_at = ''
  }
}

function onExpiryDateInput(val: string) {
  form.auth_expires_at = val
  if (val) {
    const diff = Math.ceil((new Date(val).getTime() - Date.now()) / 86400000)
    expiryDays.value = diff > 0 ? diff : 0
  } else {
    expiryDays.value = null
  }
}

const canOAuth = computed(() => {
  return form.client_id.trim() && form.tenant_id.trim()
})

let oauthPopup: Window | null = null
let oauthListener: ((e: MessageEvent) => void) | null = null

function startOAuth() {
  if (!canOAuth.value) return

  // Clean up old listener
  if (oauthListener) window.removeEventListener('message', oauthListener)

  // Request authorize URL
  apiClient.post('/oauth/authorize', {
    client_id: form.client_id.trim(),
    client_secret: form.client_secret.trim(),
    tenant_id: form.tenant_id.trim(),
    redirect_uri: `${window.location.origin}${pathPrefix}/api/oauth/callback`,
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
          form.refresh_token = tokenData.refresh_token || ''
          verifyResult.value = 'valid'
          setTimeout(() => { verifyResult.value = null }, 3000)
        } catch {
          verifyError.value = t('accounts.form.refreshToken.oauth.parseError')
          verifyResult.value = 'error'
        }
      } else {
        verifyError.value = payload || t('accounts.form.refreshToken.oauth.failed')
        verifyResult.value = 'error'
      }
    }
    window.addEventListener('message', oauthListener)
  }).catch(err => {
    verifyError.value = err?.response?.data?.error || t('accounts.form.refreshToken.oauth.failed')
    verifyResult.value = 'error'
  })
}

onUnmounted(() => {
  if (oauthListener) window.removeEventListener('message', oauthListener)
  oauthPopup?.close()
})

function submit() {
  if (!validateAll()) return
  emit('save', { ...form })
}

function close() {
  emit('update:visible', false)
}

function maskSecret(value: string): string {
  if (!value) return '-'
  if (value.length <= 8) return '\u2022'.repeat(value.length)
  return value.slice(0, 4) + '\u2022'.repeat(Math.min(value.length - 8, 16)) + value.slice(-4)
}

// --- Verify credentials ---
const verifying = ref(false)
const verifyResult = ref<'valid' | 'error' | null>(null)
const verifyError = ref('')
let verifyTimer: ReturnType<typeof setTimeout> | undefined

async function verify() {
  if (verifying.value) return
  // Require key credential fields
  if (!form.client_id.trim() || !form.client_secret.trim() || !form.tenant_id.trim()) return
  if (form.auth_type === 'auth_code' && !form.refresh_token.trim()) return

  verifying.value = true
  verifyResult.value = null
  verifyError.value = ''
  try {
    await apiClient.post('/accounts/verify', {
      auth_type: form.auth_type || 'auth_code',
      client_id: form.client_id.trim(),
      client_secret: form.client_secret.trim(),
      tenant_id: form.tenant_id.trim(),
      refresh_token: form.refresh_token.trim(),
    })
    verifyResult.value = 'valid'
    clearTimeout(verifyTimer)
    verifyTimer = setTimeout(() => { verifyResult.value = null }, 3000)
  } catch (err: any) {
    verifyResult.value = 'error'
    verifyError.value = err?.response?.data?.error || err?.message || t('accounts.verify.invalid')
  } finally {
    verifying.value = false
  }
}

// --- Preview expiry helpers ---
function daysUntilExpiry(dateStr: string): number {
  return Math.ceil((new Date(dateStr).getTime() - Date.now()) / 86400000)
}

const previewExpiryRemainingText = computed(() => {
  if (!props.account?.auth_expires_at) return ''
  const days = daysUntilExpiry(props.account.auth_expires_at)
  if (days < 0) return t('accounts.expiry.expired')
  if (days === 0) return t('accounts.expiry.today')
  return t('accounts.expiry.remaining').replace('{days}', String(days))
})

async function enterEditMode() {
  if (!props.account?.id) {
    dialogMode.value = 'edit'
    return
  }
  loadingSecrets.value = true
  formError.value = ''
  try {
    const { data } = await apiClient.get(`/accounts/${props.account.id}`)
    form.client_secret = data.client_secret
    form.refresh_token = data.refresh_token
    dialogMode.value = 'edit'
  } catch (err: any) {
    formError.value = err?.response?.data?.error || t('accounts.form.loadSecretsError')
  } finally {
    loadingSecrets.value = false
  }
}

watch(() => props.visible, (val) => {
  if (val) {
    showSecret.value = false
    showRefreshToken.value = false
    verifying.value = false
    verifyResult.value = null
    verifyError.value = ''
    Object.keys(errors).forEach(k => delete errors[k])
    formError.value = ''
    loadingSecrets.value = false

    if (props.account) {
      // Open in preview mode for existing accounts
      dialogMode.value = 'preview'
      form.name = props.account.name
      form.auth_type = props.account.auth_type
      form.client_id = props.account.client_id
      form.client_secret = props.account.client_secret
      form.tenant_id = props.account.tenant_id
      form.refresh_token = props.account.refresh_token
      form.notify_enabled = props.account.notify_enabled
      form.auth_expires_at = props.account.auth_expires_at || ''
      if (form.auth_expires_at) {
        const diff = Math.ceil((new Date(form.auth_expires_at).getTime() - Date.now()) / 86400000)
        expiryDays.value = diff > 0 ? diff : 0
      } else {
        expiryDays.value = null
      }
    } else {
      // Open in edit mode for new accounts
      dialogMode.value = 'edit'
      Object.assign(form, emptyForm())
      expiryDays.value = null
    }
  }
})

defineExpose({ saving, loadingSecrets, formError })
</script>

<style scoped>
/* === Hidden scrollbar === */
.custom-scrollbar {
  scrollbar-width: none;
  -ms-overflow-style: none;
}
.custom-scrollbar::-webkit-scrollbar {
  display: none;
}

/* === Preview fields === */
.preview-field {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px 10px;
  border-radius: 10px;
  transition: background 0.15s;
  .dark &:hover {
    background: rgba(255, 255, 255, 0.04);
  }
}
.preview-field:hover {
  background: rgba(0, 0, 0, 0.03);
}
.preview-field-label {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #9ca3af;
  .dark & {
    color: #6b7280;
  }
}
.preview-field-value {
  font-size: 12px;
  color: #374151;
  .dark & {
    color: #d1d5db;
  }
}
/* === Form styles === */
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
  max-height: 300px;
}
.field-slide-leave-from {
  max-height: 300px;
}
.field-slide-leave-to {
  opacity: 0;
  max-height: 0;
  transform: translateY(-8px);
}

/* Field error transition */
.field-error-enter-active {
  transition: all 0.2s ease;
}
.field-error-leave-active {
  transition: all 0.15s ease;
}
.field-error-enter-from,
.field-error-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
