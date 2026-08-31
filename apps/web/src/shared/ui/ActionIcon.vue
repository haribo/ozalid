<script setup lang="ts">
/**
 * A gesture, not a state: validate, comment, accept, refuse.
 *
 * Bare, deliberately. The disc is what marks a status (`StateIcon`), and putting
 * one on a button would say that pressing it *is* the state rather than the way
 * to reach it. The two are told apart by their shape before they are read.
 */
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{ name: 'check' | 'comment' | 'accept' | 'refuse'; size?: number; label?: string }>(),
  { size: 13, label: '' },
)

const NAMES: Record<string, string> = {
  check: 'valider',
  comment: 'commenter',
  accept: 'accepter',
  refuse: 'refuser',
}
const title = computed(() => props.label || NAMES[props.name])
</script>

<template>
  <svg
    class="shrink-0"
    :width="size"
    :height="size"
    viewBox="0 0 16 16"
    fill="none"
    stroke="currentColor"
    stroke-width="1.5"
    stroke-linecap="round"
    stroke-linejoin="round"
    role="img"
    :aria-label="title"
  >
    <title>{{ title }}</title>
    <path v-if="name === 'check'" d="M3.8 8.4l2.8 2.8 5.6-6.2" />
    <path v-else-if="name === 'comment'" d="M2.2 3.4h11.6v7.4H8.4L5 13.4v-2.6H2.2z" />
    <path
      v-else-if="name === 'accept'"
      d="M10.8 13.8V7.4M10.8 7.4H7.7l.7-3.1a1.5 1.5 0 0 0-2.6-1.2L3.2 7.4v6.4h7.6zM13.6 13.8h-2.8V7.4h2.8z"
    />
    <path
      v-else
      d="M5.2 2.2v6.4M5.2 8.6h3.1l-.7 3.1a1.5 1.5 0 0 0 2.6 1.2l2.6-4.3V2.2H5.2zM2.4 2.2h2.8v6.4H2.4z"
    />
  </svg>
</template>
