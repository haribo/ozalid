<script setup lang="ts">
/**
 * The door. One field, one button, then one message.
 *
 * There is no password, so there is nothing to reset and no second form. What
 * would be a recovery procedure elsewhere is the ordinary way in here
 * (ADR 0019).
 */
import { ref } from 'vue'
import { useSession } from './useSession'

const { linkSent, busy, error, requestLink, askAgain } = useSession()
const email = ref('')
</script>

<template>
  <div class="mx-auto w-full max-w-sm py-10">
    <div
      v-if="linkSent"
      class="rounded border border-emerald-600 bg-emerald-50 p-3.5 text-[13px] leading-relaxed text-emerald-800 dark:border-emerald-500 dark:bg-emerald-950 dark:text-emerald-300"
    >
      <b class="block font-semibold">The link is on its way.</b>
      Nothing yet? Check the address, then your spam.
      <button type="button" class="mt-2 block underline underline-offset-2" @click="askAgain">
        Use another address
      </button>
    </div>

    <form v-else class="flex flex-col gap-3" @submit.prevent="requestLink(email)">
      <h1 class="text-lg font-semibold">Sign in</h1>

      <label class="flex flex-col gap-1.5">
        <span
          class="font-mono text-[10.5px] tracking-wider text-slate-500 uppercase dark:text-slate-400"
        >
          address
        </span>
        <input
          v-model="email"
          type="email"
          required
          autocomplete="email"
          placeholder="you@example.com"
          class="rounded border border-slate-300 bg-white px-2.5 py-2 text-[13.5px] text-slate-900 focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-indigo-500 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100"
        />
      </label>

      <p v-if="error" class="text-[13px] text-rose-700 dark:text-rose-400">{{ error }}</p>

      <button
        type="submit"
        :disabled="busy"
        class="rounded border border-indigo-600 bg-indigo-600 px-3.5 py-2 text-[13px] font-medium text-white focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-indigo-500 disabled:opacity-60 dark:border-indigo-500 dark:bg-indigo-500"
      >
        Send the link
      </button>
    </form>
  </div>
</template>
