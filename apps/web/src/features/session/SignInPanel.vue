<script setup lang="ts">
/**
 * The door. One field, one button, then one message.
 *
 * There is no password, so there is nothing to reset and no second form. What
 * would be a recovery procedure elsewhere is the ordinary way in here
 * (ADR 0019).
 *
 * The label starts inside the field, icon and all, and lands on the border once
 * something is typed — so naming the field never costs a line of height. Asked
 * for in the first real review of these screens (#129). The icon lives in the
 * label and follows it.
 */
import { computed, ref } from 'vue'
import { useSession } from './useSession'

const { linkSent, busy, error, requestLink, askAgain } = useSession()
const email = ref('')
const focused = ref(false)

const floating = computed(() => focused.value || email.value.length > 0)
</script>

<template>
  <div class="mx-auto w-full max-w-sm py-10">
    <!-- One green thing: the disc, the same language as the book's verdicts
         (frontend ADR 0004). The old state wore three at once (#130). -->
    <div v-if="linkSent" class="flex items-start gap-3">
      <span
        class="grid h-7 w-7 flex-none place-items-center rounded-full border-2 border-emerald-600 text-emerald-700 dark:border-emerald-500 dark:text-emerald-400"
        aria-hidden="true"
      >
        <svg
          width="13"
          height="13"
          viewBox="0 0 16 16"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="m2.8 8.6 3.4 3.4 7-7.4" />
        </svg>
      </span>
      <div class="text-body leading-relaxed">
        <b class="block font-semibold">The link is on its way.</b>
        <span class="text-slate-500 dark:text-slate-400">
          Nothing yet? Check the address, then your spam.
        </span>
        <button
          type="button"
          class="mt-2 block font-mono text-mono text-indigo-700 underline underline-offset-2 dark:text-indigo-300"
          @click="askAgain"
        >
          Use another address
        </button>
      </div>
    </div>

    <form v-else class="flex flex-col gap-3" @submit.prevent="requestLink(email)">
      <h1 class="text-title font-semibold">Sign in</h1>

      <div class="relative">
        <input
          id="sign-in-address"
          v-model="email"
          type="email"
          required
          autocomplete="email"
          class="h-[42px] w-full rounded border-2 bg-slate-100 px-3 text-body text-slate-900 outline-none dark:bg-slate-800 dark:text-slate-100"
          :class="
            focused
              ? 'border-indigo-500 dark:border-indigo-400'
              : 'border-slate-300 dark:border-slate-600'
          "
          @focus="focused = true"
          @blur="focused = false"
        />
        <label
          for="sign-in-address"
          class="pointer-events-none absolute flex items-center transition-all select-none"
          :class="[
            focused ? 'text-indigo-700 dark:text-indigo-300' : 'text-slate-500 dark:text-slate-400',
            floating
              ? '-top-2 left-2 gap-1.5 rounded bg-white px-1.5 font-mono text-label tracking-wider uppercase dark:bg-slate-950'
              : 'top-1/2 left-3 -translate-y-1/2 gap-2 text-label',
          ]"
        >
          <svg
            :width="floating ? 12 : 17"
            :height="floating ? 12 : 17"
            viewBox="0 0 16 16"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            aria-hidden="true"
          >
            <rect x="1.8" y="3.4" width="12.4" height="9.2" rx="1.4" />
            <path d="m2.4 4.4 5.6 4.4 5.6-4.4" />
          </svg>
          address
        </label>
      </div>

      <p v-if="error" class="text-body text-rose-700 dark:text-rose-400">{{ error }}</p>

      <button
        type="submit"
        :disabled="busy"
        class="rounded border border-indigo-600 bg-indigo-600 px-3.5 py-2 text-body font-medium text-white focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-indigo-500 disabled:opacity-60 dark:border-indigo-500 dark:bg-indigo-500"
      >
        Send the link
      </button>
    </form>
  </div>
</template>
