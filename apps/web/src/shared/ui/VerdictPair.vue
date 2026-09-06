<script setup lang="ts">
/**
 * The verdict: one segmented control, two halves, three states (ADR 0020).
 *
 * `accept` and `refuse` are always both visible; the filled half is the
 * state, and there deliberately is a third state — nothing filled — that a
 * radio group could not express: a judged square can return to "not judged".
 * The control only reports which half was clicked; what the click means —
 * give, take back, switch — is the caller's reading of its own state.
 */
defineProps<{
  verdict: 'none' | 'accepted' | 'refused'
  disabled?: boolean
}>()

const emit = defineEmits<{ accept: []; refuse: [] }>()
</script>

<template>
  <div
    role="group"
    aria-label="verdict"
    class="inline-flex overflow-hidden rounded-md border-[1.5px] border-slate-300 dark:border-slate-600"
    :class="disabled ? 'opacity-50' : ''"
  >
    <!-- eslint-disable vue/no-restricted-html-elements -- the segmented halves ARE this primitive (#155) -->
    <button
      type="button"
      class="inline-flex min-w-[8.5rem] items-center justify-center gap-2 px-4 py-1.5 text-body font-medium focus-visible:z-10 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-indigo-500"
      :class="
        verdict === 'accepted'
          ? 'bg-emerald-700 text-white dark:bg-emerald-500 dark:text-slate-950'
          : 'cursor-pointer text-slate-600 hover:bg-slate-50 dark:text-slate-300 dark:hover:bg-slate-800'
      "
      :aria-pressed="verdict === 'accepted'"
      :disabled="disabled"
      :style="disabled ? '' : 'cursor: pointer'"
      @click="emit('accept')"
    >
      {{ verdict === 'accepted' ? '✓ accepted' : 'accept' }}
    </button>
    <button
      type="button"
      class="inline-flex min-w-[8.5rem] items-center justify-center gap-2 border-l-[1.5px] border-slate-300 px-4 py-1.5 text-body font-medium focus-visible:z-10 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-indigo-500 dark:border-slate-600"
      :class="
        verdict === 'refused'
          ? 'bg-amber-600 text-white dark:bg-amber-500 dark:text-slate-950'
          : 'cursor-pointer text-slate-600 hover:bg-slate-50 dark:text-slate-300 dark:hover:bg-slate-800'
      "
      :aria-pressed="verdict === 'refused'"
      :disabled="disabled"
      @click="emit('refuse')"
    >
      {{ verdict === 'refused' ? '✗ refused' : 'refuse' }}
    </button>
    <!-- eslint-enable vue/no-restricted-html-elements -->
  </div>
</template>
