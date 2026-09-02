<script setup lang="ts">
/**
 * A token, at the one moment it can be read.
 *
 * Only its hash is stored, so nothing can give it back once this panel goes —
 * a lost token is replaced, not recovered. It is held in a prop and nowhere
 * else, which is why a reload loses it.
 *
 * Shared by the two screens that mint one: the tokens screen of a program, and
 * the access screen where a program is created. A second copy of this panel
 * would drift from the first, and the half that drifted would be the one
 * telling somebody their token is gone forever.
 */
import { ref } from 'vue'
import AdminIcon from './AdminIcon.vue'

defineProps<{ token: string }>()
const emit = defineEmits<{ dismiss: [] }>()

const copied = ref(false)

async function copy(value: string) {
  await navigator.clipboard.writeText(value)
  copied.value = true
}
</script>

<template>
  <section class="rounded-md border-[1.5px] border-amber-500 bg-amber-50 p-4 dark:bg-amber-950">
    <p class="mb-2 flex items-center gap-2 font-semibold text-amber-800 dark:text-amber-300">
      <AdminIcon name="warn" :size="15" />
      Nouveau jeton à copier
    </p>
    <div
      class="mb-2 flex items-center gap-2 rounded border border-slate-300 bg-white px-2.5 py-2 dark:border-slate-600 dark:bg-slate-900"
    >
      <code class="flex-1 font-mono text-[12px] break-all">{{ token }}</code>
      <button
        type="button"
        class="inline-flex shrink-0 items-center gap-1.5 rounded border border-indigo-600 bg-indigo-600 px-3 py-1.5 text-[12px] font-medium text-white dark:border-indigo-500 dark:bg-indigo-500"
        @click="copy(token)"
      >
        <AdminIcon name="copy" :size="12" />{{ copied ? 'Copié' : 'Copier' }}
      </button>
    </div>
    <div class="flex items-center gap-3">
      <p class="text-[12px] text-amber-800 dark:text-amber-300">Il ne sera plus jamais affiché.</p>
      <button
        type="button"
        class="ml-auto rounded border border-indigo-600 px-3 py-1.5 text-[12px] font-medium text-indigo-700 dark:border-indigo-500 dark:text-indigo-300"
        @click="emit('dismiss')"
      >
        J'ai copié le jeton
      </button>
    </div>
  </section>
</template>
