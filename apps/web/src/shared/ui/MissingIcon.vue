<script setup lang="ts">
/**
 * A capture the run never produced, where its siblings have one.
 *
 * A status like the others, so it wears the disc (frontend ADR 0003). It is not
 * a verdict — nobody judged anything — but it answers the grid's question all
 * the same: this square has not been validated, and it cannot be.
 */
withDefaults(defineProps<{ size?: number; label?: string }>(), {
  size: 13,
  label: 'manquante',
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
      <path d="M8 3.4L13.6 12.6H2.4z" />
      <path d="M8 6.9v2.3M8 11v.1" />
    </svg>
  </span>
</template>
