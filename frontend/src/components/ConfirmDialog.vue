<template>
  <Teleport to="body">
    <Transition name="confirm-overlay">
      <div
        v-if="visible"
        class="fixed inset-0 z-50 flex items-center justify-center p-4"
        @keydown.esc="cancel"
      >
        <!-- Backdrop -->
        <div
          class="absolute inset-0 bg-black/30 dark:bg-black/50 backdrop-blur-sm"
          @click="cancel"
        />

        <!-- Dialog -->
        <Transition name="confirm-dialog" appear>
          <div
            v-if="visible"
            class="relative w-full max-w-sm rounded-2xl backdrop-blur-[40px] bg-white/80 dark:bg-[rgb(40,40,40)]/80 border border-white/20 dark:border-white/8 shadow-2xl shadow-black/10 dark:shadow-black/30 p-6"
          >
            <!-- Title -->
            <h3 class="text-[15px] font-semibold text-gray-900 dark:text-white">
              {{ title }}
            </h3>

            <!-- Message -->
            <p class="mt-2 text-[13px] text-gray-500 dark:text-gray-400 leading-relaxed">
              {{ message }}
            </p>

            <!-- Actions -->
            <div class="mt-5 flex items-center justify-end gap-2.5">
              <button
                class="px-4 py-2 rounded-xl text-[13px] font-medium text-gray-600 dark:text-gray-300 bg-gray-100/60 dark:bg-white/8 hover:bg-gray-200/60 dark:hover:bg-white/12 border border-gray-200/60 dark:border-white/10 transition-all duration-200"
                @click="cancel"
              >
                {{ cancelLabel }}
              </button>
              <button
                ref="confirmBtn"
                :class="[
                  'px-4 py-2 rounded-xl text-[13px] font-medium text-white transition-all duration-200 border',
                  danger
                    ? 'bg-red-500 hover:bg-red-600 border-red-500/80 hover:border-red-600/80 shadow-md shadow-red-500/20'
                    : 'bg-apple-blue hover:bg-apple-blue-hover border-apple-blue/80 hover:border-apple-blue-hover/80 shadow-md shadow-apple-blue/20'
                ]"
                @click="confirm"
              >
                {{ confirmLabel }}
              </button>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, computed } from 'vue'
import { useI18n } from '../i18n'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  visible: boolean
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
}>(), {
  danger: false,
})

const confirmLabel = computed(() => props.confirmText || t('confirm.ok'))
const cancelLabel = computed(() => props.cancelText || t('confirm.cancel'))

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'confirm'): void
  (e: 'cancel'): void
}>()

const confirmBtn = ref<HTMLButtonElement | null>(null)

watch(() => props.visible, (val) => {
  if (val) {
    nextTick(() => confirmBtn.value?.focus())
  }
})

function cancel() {
  emit('update:visible', false)
  emit('cancel')
}

function confirm() {
  emit('update:visible', false)
  emit('confirm')
}
</script>

<style scoped>
.confirm-overlay-enter-active,
.confirm-overlay-leave-active {
  transition: opacity 0.2s ease;
}
.confirm-overlay-enter-from,
.confirm-overlay-leave-to {
  opacity: 0;
}

.confirm-dialog-enter-active {
  transition: all 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.confirm-dialog-leave-active {
  transition: all 0.15s ease;
}
.confirm-dialog-enter-from {
  opacity: 0;
  transform: scale(0.95);
}
.confirm-dialog-leave-to {
  opacity: 0;
  transform: scale(0.97);
}
</style>
