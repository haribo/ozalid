<script setup lang="ts">
/**
 * A segmented bar carrying its own counts (#236): the number sits inside its
 * segment, white on the status hue, and no legend repeats it below.
 *
 * A non-empty segment keeps a minimum width so `214 accepted · 1 refused`
 * never hides the 1 — the bar is slightly less proportional, and that is the
 * accepted trade. The segment order is fixed here, whatever the caller sends:
 * position identifies the status as much as hue does (WCAG 1.4.1), and the
 * full name stays as the tooltip and the accessible name.
 */
import { computed } from 'vue'
import { TONE_LABELS, type Tone } from '@/shared/lib'

const props = defineProps<{ parts: { tone: Tone; count: number; label?: string }[] }>()

/** The reviewer's work first, then what settled, then the dev's, then what
 * was never wired. Zero-count segments are dropped: an empty slice is noise. */
const ORDER: Tone[] = ['reviewer', 'done', 'dev', 'idle']
const shown = computed(() =>
  [...props.parts]
    .filter((p) => p.count > 0)
    .toSorted((a, b) => ORDER.indexOf(a.tone) - ORDER.indexOf(b.tone)),
)
const total = computed(() => shown.value.reduce((n, p) => n + p.count, 0))

const BAR: Record<string, string> = {
  idle: 'bg-slate-400 dark:bg-slate-600',
  reviewer: 'bg-indigo-600 dark:bg-indigo-500',
  dev: 'bg-amber-600 dark:bg-amber-500',
  done: 'bg-emerald-600 dark:bg-emerald-500',
}
</script>

<template>
  <div
    v-if="total > 0"
    class="flex h-5 min-w-[130px] overflow-hidden rounded bg-slate-200 font-mono text-mono font-semibold text-white tabular-nums dark:bg-slate-700"
  >
    <span
      v-for="p in shown"
      :key="p.tone"
      class="inline-flex min-w-[1.7em] basis-0 items-center justify-center"
      :class="BAR[p.tone]"
      :style="{ flexGrow: p.count }"
      role="img"
      :aria-label="`${p.count} ${p.label ?? TONE_LABELS[p.tone]}`"
      :title="`${p.count} ${p.label ?? TONE_LABELS[p.tone]}`"
      >{{ p.count }}</span
    >
  </div>
  <span v-else class="font-mono text-mono text-slate-400 dark:text-slate-500">—</span>
</template>
