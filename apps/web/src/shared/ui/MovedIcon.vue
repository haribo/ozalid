<script setup lang="ts">
/**
 * A capture that has moved since it was approved.
 *
 * A status like any other, so it wears the disc (frontend ADR 0003) — but its
 * own glyph. Movement keeps a shape of its own (product.md §3.3): a capture
 * can be *validated* and *moved* at once, and borrowing a verdict's check or
 * speech bubble would have one mark overwrite the other.
 */
withDefaults(defineProps<{ size?: number; label?: string }>(), {
  size: 13,
  label: 'moved since it was validated',
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
      <!-- Two frames that no longer coincide: the approved image and the
           incoming one (#209). -->
      <rect x="2.6" y="2.6" width="7.6" height="7.6" rx="1" />
      <rect x="5.8" y="5.8" width="7.6" height="7.6" rx="1" />
    </svg>
  </span>
</template>
