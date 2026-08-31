<script setup lang="ts">
/**
 * The gestures the administration screens need, and nothing else.
 *
 * Kept apart from `ActionIcon`, which carries a reviewer's verdicts: the two
 * sets are used on different screens by different people, and merging them
 * would make one component grow every time either screen does.
 */
import { computed } from 'vue'

type Name = 'people' | 'keys' | 'remove' | 'swap' | 'add' | 'copy' | 'warn' | 'person' | 'program'

const props = withDefaults(defineProps<{ name: Name; size?: number; label?: string }>(), {
  size: 13,
  label: '',
})

const NAMES: Record<Name, string> = {
  people: 'gérer les accès',
  keys: 'ses jetons',
  remove: 'retirer',
  swap: 'changer les droits',
  add: 'ajouter',
  copy: 'copier',
  warn: 'attention',
  person: 'personne',
  program: 'programme',
}
const title = computed(() => props.label || NAMES[props.name])

const PATHS: Record<Name, string> = {
  people:
    'M6 3.2a2.4 2.4 0 1 1 0 4.8 2.4 2.4 0 0 1 0-4.8zM1.8 13.4a4.4 4.4 0 0 1 8.4 0M11 3.4a2.4 2.4 0 0 1 0 4.4M12.4 13.4a4.4 4.4 0 0 0-1.6-3.4',
  keys: 'M5.4 8a2.6 2.6 0 1 1 0 5.2 2.6 2.6 0 0 1 0-5.2zM7.4 8.6 13 3M11 5l1.6 1.6M9.6 6.4 11.2 8',
  remove: 'M3 4.4h10M6.4 4.4V2.8h3.2v1.6M4.4 4.4l.7 8.4h5.8l.7-8.4',
  swap: 'M4.4 2.8v8.6M2.2 9.2l2.2 2.2 2.2-2.2M11.6 13.2V4.6M13.8 6.6l-2.2-2.2-2.2 2.2',
  add: 'M8 3.4v9.2M3.4 8h9.2',
  copy: 'M5.5 5.5h8v8h-8zM10.5 2.5H3.2A.7.7 0 0 0 2.5 3.2v7.3',
  warn: 'M8 2.2 14.6 13.4H1.4zM8 6.4v3.2M8 11.6v.1',
  person: 'M8 2.8a2.6 2.6 0 1 1 0 5.2 2.6 2.6 0 0 1 0-5.2zM2.8 13.4a5.2 5.2 0 0 1 10.4 0',
  program: 'M2.6 5.4h10.8v8H2.6zM8 5.4V2.8M5.6 9h.1M10.4 9h.1',
}
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
    <path :d="PATHS[name]" />
  </svg>
</template>
