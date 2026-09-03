<script setup lang="ts">
/**
 * Shown when a session died with a verdict in flight.
 *
 * The page underneath does not move: what was being looked at stays on screen,
 * and the bar says what was lost by name. "The last verdict was not recorded"
 * is a sentence somebody can act on; "something went wrong" is not.
 *
 * The address is already known — the person was signed in a moment ago — so
 * reconnecting is one click, not a form.
 */
import { useSession } from './useSession'

const { person, linkSent, busy, requestLink } = useSession()
</script>

<template>
  <div
    class="flex flex-wrap items-center gap-3 border-b border-amber-500 bg-amber-50 px-6 py-2.5 text-[13px] text-amber-800 dark:bg-amber-950 dark:text-amber-300"
    role="alert"
  >
    <template v-if="linkSent">
      <span class="flex-1 font-medium">
        A link went to {{ person?.email }}. Open it: this tab will carry on by itself.
      </span>
    </template>
    <template v-else>
      <span class="flex-1 font-medium"> Session expired. The last verdict was not recorded. </span>
      <button
        type="button"
        :disabled="busy || !person"
        class="rounded border border-amber-600 bg-amber-600 px-3 py-1.5 text-[13px] font-medium text-white focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-amber-500 disabled:opacity-60"
        @click="person && requestLink(person.email)"
      >
        Sign in again
      </button>
    </template>
  </div>
</template>
