<script setup lang="ts">
/**
 * The colour code, taught once under a table (#238): a filled swatch of the
 * segment colour and the status name. It speaks for the gauge segments and
 * the state discs alike — one palette, indigo for the reviewer, emerald for
 * done, amber for the dev, slate for what was never wired.
 */
const ENTRIES = [
  { label: 'to review', swatch: 'bg-indigo-600 dark:bg-indigo-500' },
  { label: 'accepted', swatch: 'bg-emerald-600 dark:bg-emerald-500' },
  { label: 'refused', swatch: 'bg-amber-600 dark:bg-amber-500' },
  { label: 'not instrumented', swatch: 'bg-slate-400 dark:bg-slate-600' },
]
</script>

<template>
  <div
    class="mt-2.5 flex flex-wrap justify-center gap-x-4 gap-y-1.5 font-mono text-mono text-slate-500 dark:text-slate-400"
  >
    <span v-for="e in ENTRIES" :key="e.label" class="inline-flex items-center gap-1.5">
      <span class="h-3.5 w-3.5 rounded" :class="e.swatch" aria-hidden="true" />{{ e.label }}
    </span>
  </div>
</template>
