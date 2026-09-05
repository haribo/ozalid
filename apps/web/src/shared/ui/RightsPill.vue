<script setup lang="ts">
/**
 * What a membership carries, in a word.
 *
 * Two values and no more (`product.md` §8.1). The colour separates them at a
 * glance without spelling out the difference every time: a member acts, a
 * reader does not, and the quieter one is the one that changes nothing.
 */
const props = defineProps<{ rights: 'reader' | 'member' }>()

const CLASSES: Record<string, string> = {
  member:
    'border-indigo-400 bg-indigo-50 text-indigo-700 dark:border-indigo-500 dark:bg-indigo-950 dark:text-indigo-300',
  reader:
    'border-slate-300 bg-slate-100 text-slate-600 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-400',
}
</script>

<template>
  <span
    class="inline-block rounded border px-1.5 py-0.5 font-mono text-label tracking-wide uppercase"
    :class="CLASSES[props.rights]"
  >
    {{ rights }}
  </span>
</template>
