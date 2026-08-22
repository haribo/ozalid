<script setup lang="ts">
/** A case's state, named and coloured by who holds the ball. */
import { computed } from 'vue'
import { toneOfCase, type CaseState } from '@/shared/lib'
import StateIcon from './StateIcon.vue'

const props = defineProps<{ state: CaseState }>()
const tone = computed(() => toneOfCase(props.state))

const CLASSES: Record<string, string> = {
  idle: 'border-slate-300 bg-slate-100 text-slate-500 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-400',
  reviewer:
    'border-indigo-400 bg-indigo-50 text-indigo-700 dark:border-indigo-500 dark:bg-indigo-950 dark:text-indigo-300',
  dev: 'border-amber-500 bg-amber-50 text-amber-800 dark:border-amber-500 dark:bg-amber-950 dark:text-amber-300',
  done: 'border-emerald-600 bg-emerald-50 text-emerald-800 dark:border-emerald-500 dark:bg-emerald-950 dark:text-emerald-300',
}
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 rounded border px-1.5 py-0.5 font-mono text-[10.5px] whitespace-nowrap"
    :class="CLASSES[tone]"
  >
    <StateIcon :tone="tone" :size="11" />{{ state }}
  </span>
</template>
