<script setup lang="ts">
/**
 * A capture the run never produced, where its siblings have one.
 *
 * A status like the others, so it wears the disc (frontend ADR 0003). It is not
 * a verdict — nobody judged anything — but it answers the grid's question all
 * the same: this capture has not been accepted, and it cannot be.
 */
withDefaults(defineProps<{ size?: number; label?: string }>(), {
  size: 13,
  label: 'missing',
})
</script>

<template>
  <span
    class="grid shrink-0 place-items-center rounded-full border-[1.5px] border-current"
    :style="{ width: `${Math.round(size * 1.55)}px`, height: `${Math.round(size * 1.55)}px` }"
    role="img"
    :aria-label="label"
  >
    <svg
      :width="size"
      :height="size"
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      stroke-width="1.6"
      stroke-linecap="round"
      stroke-linejoin="round"
      aria-hidden="true"
    >
      <title>{{ label }}</title>
      <!-- A crossed-out frame says one thing: there is no image here (#209). -->
      <rect x="2.6" y="3.6" width="10.8" height="8.8" rx="1" />
      <path d="M3.4 11.6l9.2-7.2" />
    </svg>
  </span>
</template>
