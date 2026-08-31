<script setup lang="ts">
/**
 * A segmented bar with a counted legend. Segments carry no words, so the
 * legend does: an icon nobody can read is decoration.
 */
import { computed } from 'vue'
import { TONE_LABELS, type Tone } from '@/shared/lib'
import StateIcon from './StateIcon.vue'

const props = defineProps<{ parts: { tone: Tone; count: number; label?: string }[] }>()

/** Zero-count segments are dropped: an empty slice is noise, not information. */
const shown = computed(() => props.parts.filter((p) => p.count > 0))
const total = computed(() => shown.value.reduce((n, p) => n + p.count, 0))

const BAR: Record<string, string> = {
  idle: 'bg-slate-400 dark:bg-slate-600',
  reviewer: 'bg-indigo-600 dark:bg-indigo-400',
  dev: 'bg-amber-500 dark:bg-amber-400',
  done: 'bg-emerald-600 dark:bg-emerald-400',
}
const TEXT: Record<string, string> = {
  idle: 'text-slate-500 dark:text-slate-400',
  reviewer: 'text-indigo-700 dark:text-indigo-300',
  dev: 'text-amber-700 dark:text-amber-400',
  done: 'text-emerald-700 dark:text-emerald-400',
}
</script>

<template>
  <div v-if="total > 0" class="min-w-[130px]">
    <div class="flex h-1.5 overflow-hidden rounded bg-slate-200 dark:bg-slate-700">
      <span
        v-for="p in shown"
        :key="p.tone"
        class="basis-0"
        :class="BAR[p.tone]"
        :style="{ flexGrow: p.count }"
      />
    </div>
    <div class="mt-1.5 flex flex-wrap gap-x-2.5 gap-y-1 font-mono text-[10.5px]">
      <span
        v-for="p in shown"
        :key="p.tone"
        class="inline-flex items-center gap-1"
        :class="TEXT[p.tone]"
      >
        <StateIcon :tone="p.tone" :label="`${p.count} ${p.label ?? TONE_LABELS[p.tone]}`" />{{
          p.count
        }}
      </span>
    </div>
  </div>
  <span v-else class="font-mono text-[11px] text-slate-400 dark:text-slate-500">—</span>
</template>
