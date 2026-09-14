<script setup lang="ts">
/**
 * An identifier the reader can take with them.
 *
 * The ids this product shows are short and public — twelve hex characters —
 * and their whole purpose is to be quoted somewhere else: a ticket, a thread,
 * a request against the API. Shown, they are still one selection away from
 * being mistyped; this makes them one click away from being right.
 *
 * No acknowledgement: clicking a thing that copies is its own answer, and the
 * bars this sits in have no room for a word that says what just happened.
 */
import AdminIcon from './AdminIcon.vue'

const props = defineProps<{
  value: string
  /** What a screen reader hears — "Copy the capture id", not "Copy". */
  label: string
}>()

async function copy() {
  await navigator.clipboard.writeText(props.value)
}
</script>

<template>
  <button
    type="button"
    :aria-label="label"
    class="inline-flex flex-none cursor-pointer items-center gap-1.5 rounded border border-slate-200 px-1.5 tabular-nums hover:border-indigo-600 hover:text-indigo-700 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-500 dark:border-slate-700 dark:hover:border-indigo-400 dark:hover:text-indigo-300"
    @click="copy"
  >
    {{ value }}<AdminIcon name="copy" :size="12" />
  </button>
</template>
