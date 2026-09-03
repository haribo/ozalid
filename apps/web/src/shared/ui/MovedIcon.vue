<script setup lang="ts">
/**
 * A capture that has moved since it was approved.
 *
 * A status like any other, so it wears the disc (frontend ADR 0003) — but its
 * own glyph. Freshness is an overlay, not a verdict (product.md §3.3): a capture
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
      <path d="M3.4 6h9.2M10.6 3.8l2 2.2-2 2.2" />
      <path d="M12.6 10h-9.2M5.4 7.8l-2 2.2 2 2.2" />
    </svg>
  </span>
</template>
