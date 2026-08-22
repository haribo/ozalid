<script setup lang="ts">
/**
 * One shape family for every state, at every level. A ticked circle on a
 * category gauge, on a case row and under a capture all mean "judged, nothing
 * to say" — learning the language once is the point.
 */
import { computed } from 'vue'
import { TONE_LABELS, type Tone } from '@/shared/lib'

const props = withDefaults(defineProps<{ tone: Tone; size?: number; label?: string }>(), {
  size: 13,
  label: '',
})

const title = computed(() => props.label || TONE_LABELS[props.tone])
</script>

<template>
  <svg
    class="shrink-0"
    :width="size"
    :height="size"
    viewBox="0 0 16 16"
    fill="none"
    stroke="currentColor"
    stroke-width="1.4"
    stroke-linecap="round"
    stroke-linejoin="round"
    role="img"
    :aria-label="title"
  >
    <title>{{ title }}</title>
    <template v-if="tone === 'done'">
      <circle cx="8" cy="8" r="6.3" />
      <path d="M5.2 8.3l2 2 3.6-4.2" />
    </template>
    <template v-else-if="tone === 'reviewer'">
      <circle cx="8" cy="8" r="6.3" />
    </template>
    <template v-else-if="tone === 'dev'">
      <path d="M2.2 3.4h11.6v7.4H8.4L5 13.4v-2.6H2.2z" />
    </template>
    <template v-else>
      <circle cx="8" cy="8" r="6.3" stroke-dasharray="2.4 2" />
    </template>
  </svg>
</template>
