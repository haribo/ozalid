<script setup lang="ts">
/**
 * What a comment *is* — a defect or an improvement — which is not what it is
 * *waiting for*.
 *
 * It needs its own shapes: `StateIcon`'s plain circle already means "waiting
 * for the reviewer", and borrowing it for "improvement" put two meanings on one
 * shape. The kind is written on the comment, where it is exact, and never
 * colours the case's state (ADR 0012).
 */
import { computed } from 'vue'

const props = withDefaults(defineProps<{ kind: 'defect' | 'improvement'; size?: number }>(), {
  size: 13,
})

const title = computed(() => (props.kind === 'defect' ? 'défaut' : 'amélioration'))
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
    <path v-if="kind === 'defect'" d="M2.2 3.4h11.6v7.4H8.4L5 13.4v-2.6H2.2z" />
    <path v-else d="M8 1.8l1.9 3.9 4.3.6-3.1 3 .7 4.3L8 11.6l-3.8 2 .7-4.3-3.1-3 4.3-.6z" />
  </svg>
</template>
