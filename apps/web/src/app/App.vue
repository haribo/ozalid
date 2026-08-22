<script setup lang="ts">
/**
 * The book's shell. Chrome stays quiet: this interface frames someone else's
 * screenshots, and nothing here may be mistaken for part of them.
 */
</script>

<template>
  <div class="min-h-dvh bg-white text-slate-900 dark:bg-slate-950 dark:text-slate-100">
    <header
      class="flex items-center gap-3 border-b border-slate-200 bg-slate-50 px-6 py-2.5 dark:border-slate-700 dark:bg-slate-900"
    >
      <RouterLink to="/" class="text-[15px] font-semibold"> ozalid </RouterLink>
    </header>
    <RouterView />
  </div>
</template>
