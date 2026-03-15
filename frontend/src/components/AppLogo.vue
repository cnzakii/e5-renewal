<template>
  <div
    class="logo-container"
    :class="{ 'no-hover': !animated }"
    data-testid="app-logo"
  >
    <svg
      xmlns="http://www.w3.org/2000/svg"
      :width="size"
      :height="size"
      viewBox="0 0 32 32"
      class="logo-svg"
    >
      <defs>
        <linearGradient id="app-logo-bg" x1=".15" y1="0" x2=".85" y2="1">
          <stop offset="0%" stop-color="#3B9EFF" />
          <stop offset="100%" stop-color="#0071E3" />
        </linearGradient>
        <clipPath id="app-logo-clip"><rect width="32" height="32" rx="7.5" /></clipPath>
      </defs>
      <rect width="32" height="32" rx="7.5" fill="url(#app-logo-bg)" />
      <rect width="32" height="15" fill="white" opacity="0.1" clip-path="url(#app-logo-clip)" />
      <g fill="white">
        <path d="M 5.5 8.5 H 14 V 11.3 H 8.3 V 14.6 H 13 V 17.4 H 8.3 V 20.7 H 14 V 23.5 H 5.5 Z" />
        <path d="M 26.5 8.5 H 17 V 14.6 H 22.5 C 25.5 14.6 26.8 17 26.8 19.3 C 26.8 22 25 23.5 22.5 23.5 H 17 V 20.7 H 22 C 23.3 20.7 23.8 20 23.8 19 C 23.8 17.8 23 17.4 22 17.4 H 19.8 V 11.3 H 26.5 Z" />
      </g>
    </svg>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  size?: number
  animated?: boolean
}>(), {
  size: 40,
  animated: false,
})
</script>

<style scoped>
.logo-container {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: default;
}

.logo-svg {
  transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
  filter: drop-shadow(0 4px 12px rgba(0, 113, 227, 0.25));
  flex-shrink: 0;
}

/* Hover effects */
.logo-container:not(.no-hover):hover .logo-svg {
  transform: scale(1.08) translateY(-2px);
  filter: drop-shadow(0 8px 20px rgba(0, 113, 227, 0.35));
  cursor: pointer;
}

.logo-container:not(.no-hover):hover {
  cursor: pointer;
}
</style>
