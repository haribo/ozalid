<script setup lang="ts">
/**
 * An empty list is an answer, not a failure (#109, #112): every screen says
 * so the same way, in one dashed box. Four hand-rolled copies converged here
 * (#155); a new `border-dashed` outside shared/ui is a lint error.
 *
 * The default reading is the one-line mono sentence. A screen with more to
 * say — a headline and the way forward — passes `prose` and writes body text.
 */
withDefaults(defineProps<{ prose?: boolean }>(), { prose: false })
</script>

<template>
  <p
    class="rounded-md border border-dashed border-slate-300 px-4 text-center dark:border-slate-600"
    :class="
      prose
        ? 'py-8 text-body text-slate-600 dark:text-slate-400'
        : 'py-6 font-mono text-mono text-slate-500 dark:text-slate-400'
    "
  >
    <slot />
  </p>
</template>
